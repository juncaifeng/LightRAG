import asyncio
import time
import logging
from typing import Dict, List, Any

from .base import HookDispatcher, EventEnvelope, SubscriberReply, MergeStrategy, LocalSubscriberAdapter

logger = logging.getLogger(__name__)

class LocalMemoryDispatcher(HookDispatcher):
    """
    Fallback Dispatcher that runs purely in memory within the LightRAG Python process.
    Provides zero-latency short-circuiting for local subscribers.
    Does not connect to external Go Event Bus.
    """
    def __init__(self):
        self.subscribers: Dict[str, List[LocalSubscriberAdapter]] = {}

    def register_local_subscriber(self, topic: str, subscriber: LocalSubscriberAdapter):
        if topic not in self.subscribers:
            self.subscribers[topic] = []
        self.subscribers[topic].append(subscriber)
        logger.debug(f"Registered local subscriber {subscriber.subscriber_id} to topic {topic}")

    async def publish_and_wait(self, envelope: EventEnvelope) -> SubscriberReply:
        """
        Executes local functions directly and aggregates their results using the defined mechanical merge strategy.
        """
        topic = envelope.topic
        subs = self.subscribers.get(topic, [])

        if not subs:
            logger.debug(f"No subscribers found for topic {topic}, returning empty reply.")
            return SubscriberReply(
                outputs={},
                strategy=MergeStrategy.IGNORE,
                correlation_id=envelope.correlation_id
            )

        logger.debug(f"Dispatching event {envelope.correlation_id} on {topic} to {len(subs)} local subscribers.")

        # Execute all subscribers concurrently
        start_time = time.time()
        tasks = [sub.process(envelope) for sub in subs]
        replies: List[SubscriberReply] = await asyncio.gather(*tasks, return_exceptions=True)

        # Merge results
        merged_reply = SubscriberReply(
            outputs={},
            strategy=MergeStrategy.APPEND,
            correlation_id=envelope.correlation_id,
            weight=0
        )

        for i, reply in enumerate(replies):
            if isinstance(reply, Exception):
                logger.error(f"Subscriber {subs[i].subscriber_id} failed: {reply}")
                continue

            self._merge_responses(merged_reply, reply)

        latency = int((time.time() - start_time) * 1000)
        merged_reply.latency_ms = latency
        
        return merged_reply

    def _merge_responses(self, base: SubscriberReply, incoming: SubscriberReply):
        """
        Mechanical merge logic replicating the Event Bus behavior in memory.
        """
        if incoming.strategy == MergeStrategy.IGNORE:
            return

        if incoming.strategy == MergeStrategy.REPLACE:
            if incoming.weight > base.weight:
                base.outputs = incoming.outputs
                base.weight = incoming.weight
                base.strategy = MergeStrategy.REPLACE
            return

        if incoming.strategy == MergeStrategy.APPEND:
            if base.strategy == MergeStrategy.REPLACE and base.weight > 0:
                return
            
            for k, v in incoming.outputs.items():
                if k in base.outputs:
                    # In memory, we handle lists appending naturally instead of byte concatenation
                    if isinstance(base.outputs[k], list) and isinstance(v, list):
                        base.outputs[k].extend(v)
                    elif isinstance(base.outputs[k], dict) and isinstance(v, dict):
                        base.outputs[k].update(v)
                    else:
                        # Fallback for generic types (overwrite or append depending on type)
                        # For simple integration, we just overwrite if it's not a collection
                        base.outputs[k] = v
                else:
                    base.outputs[k] = v