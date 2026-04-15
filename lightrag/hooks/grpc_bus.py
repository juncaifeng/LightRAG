import time
import uuid
import grpc
import json
from typing import Dict, Any

from .base import HookDispatcher, EventEnvelope, SubscriberReply, MergeStrategy, LocalSubscriberAdapter
import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc

class GrpcEventBusDispatcher(HookDispatcher):
    """
    Connects to the external Go Event Bus.
    Converts EventEnvelopes to Protobuf and waits for gathered responses via gRPC.
    """
    def __init__(self, target: str):
        self.target = target
        self.channel = grpc.aio.insecure_channel(target)
        self.stub = pb2_grpc.EventBusStub(self.channel)

    def register_local_subscriber(self, topic: str, subscriber: LocalSubscriberAdapter):
        # Phase 3 scope: We only use this dispatcher to PUBLISH to the Go Bus.
        # In a fully distributed system, this method would call `RegisterSubscriber`
        # and start a streaming loop to listen for events to process locally.
        # For now, if we are in Grpc mode, we rely on the Go bus and external subscribers.
        pass

    async def publish_and_wait(self, envelope: EventEnvelope) -> SubscriberReply:
        """
        Publishes the event to the external gRPC bus and waits for the mechanically merged response.
        """
        # Convert Python dictionary inputs to bytes for the protobuf map
        proto_inputs = {}
        for k, v in envelope.inputs.items():
            if isinstance(v, str):
                proto_inputs[k] = v.encode('utf-8')
            elif isinstance(v, bytes):
                proto_inputs[k] = v
            else:
                proto_inputs[k] = json.dumps(v).encode('utf-8')

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

        try:
            resp: pb2.SubscriberReply = await self.stub.PublishAndWait(req)
            
            # Convert bytes back to strings/dicts
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
                error_code=resp.error_code
            )
        except grpc.RpcError as e:
            # Graceful degradation: return empty IGNORE response
            return SubscriberReply(
                outputs={},
                strategy=MergeStrategy.IGNORE,
                correlation_id=req.correlation_id,
                error_message=f"gRPC Error: {e.details()}"
            )
