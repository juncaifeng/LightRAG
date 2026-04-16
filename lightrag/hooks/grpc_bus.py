import asyncio
import time
import uuid
import grpc
import json
import logging
from typing import Dict, List, Any

from .base import HookDispatcher, EventEnvelope, SubscriberReply, MergeStrategy, LocalSubscriberAdapter
import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc

logger = logging.getLogger(__name__)

class GrpcEventBusDispatcher(HookDispatcher):
    """
    Connects to the external Go Event Bus.
    Also supports local subscribers as built-in fallback — if the external bus
    has no subscribers for a topic, local subscribers handle it transparently.
    """
    def __init__(self, target: str):
        self.target = target
        self._stub = None
        self._local_subs: Dict[str, List[LocalSubscriberAdapter]] = {}

    def _get_stub(self):
        """Lazily create channel+stub on first use, ensuring they belong to the running event loop."""
        if self._stub is None:
            self.channel = grpc.aio.insecure_channel(self.target)
            self._stub = pb2_grpc.EventBusStub(self.channel)
        return self._stub

    def register_local_subscriber(self, topic: str, subscriber: LocalSubscriberAdapter):
        """Register a local subscriber as built-in fallback for this topic."""
        self._local_subs.setdefault(topic, []).append(subscriber)
        logger.debug(f"Registered local subscriber {subscriber.subscriber_id} to topic {topic}")

    async def _run_local_subscribers(self, envelope: EventEnvelope) -> SubscriberReply | None:
        """Run local subscribers for a topic and merge their results."""
        subs = self._local_subs.get(envelope.topic, [])
        if not subs:
            return None

        start = time.time()
        tasks = [sub.process(envelope) for sub in subs]
        replies: list = await asyncio.gather(*tasks, return_exceptions=True)

        merged = SubscriberReply(
            outputs={},
            strategy=MergeStrategy.APPEND,
            correlation_id=envelope.correlation_id,
            weight=0,
            subscriber_id="local",
        )

        for i, reply in enumerate(replies):
            if isinstance(reply, Exception):
                logger.error(f"Local subscriber {subs[i].subscriber_id} failed: {reply}")
                continue
            self._merge_reply(merged, reply)

        merged.latency_ms = int((time.time() - start) * 1000)
        return merged

    @staticmethod
    def _merge_reply(base: SubscriberReply, incoming: SubscriberReply):
        if incoming.strategy == MergeStrategy.IGNORE:
            return
        if incoming.strategy == MergeStrategy.REPLACE:
            if incoming.weight > base.weight:
                base.outputs = incoming.outputs
                base.weight = incoming.weight
                base.strategy = MergeStrategy.REPLACE
            return
        # APPEND
        if base.strategy == MergeStrategy.REPLACE and base.weight > 0:
            return
        for k, v in incoming.outputs.items():
            if k in base.outputs:
                if isinstance(base.outputs[k], list) and isinstance(v, list):
                    base.outputs[k].extend(v)
                elif isinstance(base.outputs[k], dict) and isinstance(v, dict):
                    base.outputs[k].update(v)
                else:
                    base.outputs[k] = v
            else:
                base.outputs[k] = v

    async def publish_and_wait(self, envelope: EventEnvelope) -> SubscriberReply:
        """
        Publishes the event to the external gRPC bus and also runs local subscribers.
        Merges results from both sources. If gRPC fails, local subscribers still provide results.
        """
        # Run gRPC and local subscribers concurrently
        grpc_task = asyncio.create_task(self._grpc_publish(envelope))
        local_task = asyncio.create_task(self._run_local_subscribers(envelope))

        grpc_reply = None
        local_reply = None

        # Wait for both, handle gRPC failure gracefully
        done, _ = await asyncio.wait(
            [grpc_task, local_task],
            return_when=asyncio.ALL_COMPLETED,
        )
        for t in done:
            if t is grpc_task:
                try:
                    grpc_reply = t.result()
                except Exception as e:
                    logger.warning(f"gRPC publish failed: {e}")
            else:
                try:
                    local_reply = t.result()
                except Exception as e:
                    logger.warning(f"Local subscriber failed: {e}")

        # If gRPC returned a real response (not empty IGNORE), merge with local
        grpc_has_data = (
            grpc_reply is not None
            and grpc_reply.outputs
            and grpc_reply.strategy != MergeStrategy.IGNORE
        )
        local_has_data = (
            local_reply is not None
            and local_reply.outputs
        )

        if grpc_has_data and local_has_data:
            # Both produced results — merge
            if grpc_reply.strategy == MergeStrategy.REPLACE or local_reply.strategy == MergeStrategy.REPLACE:
                # Use the one with higher weight
                winner = grpc_reply if grpc_reply.weight >= local_reply.weight else local_reply
                return SubscriberReply(
                    outputs=winner.outputs,
                    strategy=MergeStrategy.REPLACE,
                    weight=winner.weight,
                    correlation_id=envelope.correlation_id,
                    subscriber_id=winner.subscriber_id,
                    latency_ms=max(grpc_reply.latency_ms, local_reply.latency_ms),
                )
            else:
                # APPEND — combine outputs
                merged_outputs = {}
                for src in [grpc_reply, local_reply]:
                    for k, v in src.outputs.items():
                        if k in merged_outputs:
                            if isinstance(merged_outputs[k], list) and isinstance(v, list):
                                merged_outputs[k].extend(v)
                            else:
                                merged_outputs[k] = v
                        else:
                            merged_outputs[k] = v
                return SubscriberReply(
                    outputs=merged_outputs,
                    strategy=MergeStrategy.APPEND,
                    weight=max(grpc_reply.weight, local_reply.weight),
                    correlation_id=envelope.correlation_id,
                    subscriber_id="merged",
                    latency_ms=max(grpc_reply.latency_ms, local_reply.latency_ms),
                )

        if grpc_has_data:
            return grpc_reply
        if local_has_data:
            return local_reply

        # Both failed — return empty
        return SubscriberReply(
            outputs={},
            strategy=MergeStrategy.IGNORE,
            correlation_id=envelope.correlation_id,
            error_message="No subscribers responded (neither external nor local)",
        )

    async def _grpc_publish(self, envelope: EventEnvelope) -> SubscriberReply:
        """Publish to the external gRPC bus."""
        proto_inputs = {}
        for k, v in envelope.inputs.items():
            if isinstance(v, str):
                proto_inputs[k] = v.encode('utf-8')
            elif isinstance(v, bytes):
                proto_inputs[k] = v
            elif isinstance(v, (int, float, bool)):
                proto_inputs[k] = str(v).encode('utf-8')
            else:
                try:
                    proto_inputs[k] = json.dumps(v).encode('utf-8')
                except (TypeError, ValueError):
                    proto_inputs[k] = str(v).encode('utf-8')

        req = pb2.EventEnvelope(
            topic=envelope.topic,
            correlation_id=envelope.correlation_id or str(uuid.uuid4()),
            trace_id=envelope.trace_id,
            deadline_timestamp=envelope.deadline_timestamp,
            priority=envelope.priority,
            source_service=envelope.source_service,
            inputs=proto_inputs,
            metadata=envelope.metadata
        )

        resp: pb2.SubscriberReply = await self._get_stub().PublishAndWait(req)

        python_outputs = {}
        for k, v in resp.outputs.items():
            try:
                python_outputs[k] = json.loads(v.decode('utf-8'))
            except json.JSONDecodeError:
                python_outputs[k] = v.decode('utf-8')

        return SubscriberReply(
            outputs=python_outputs,
            strategy=resp.strategy,
            weight=resp.weight,
            correlation_id=resp.correlation_id,
            subscriber_id=resp.subscriber_id,
            latency_ms=resp.latency_ms,
            error_code=resp.error_code,
        )
