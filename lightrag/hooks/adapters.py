from .base import LocalSubscriberAdapter, EventEnvelope, SubscriberReply, MergeStrategy
from typing import Any
import logging

logger = logging.getLogger(__name__)

class NativeChunkingSubscriber(LocalSubscriberAdapter):
    """
    Wraps the native `chunking_by_token_size` function from `operate.py` into a standard EventBus subscriber.
    """
    def __init__(self, chunking_func: Any, topic: str = "rag.insert.chunking"):
        super().__init__(
            topic=topic,
            subscriber_id="native-token-chunker",
            strategy=MergeStrategy.APPEND,
            weight=10
        )
        # Store a reference to the actual function so we don't create circular imports
        self.chunking_func = chunking_func

    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        inputs = envelope.inputs
        
        # Extract parameters needed for chunking
        content = inputs.get("content")
        tokenizer = inputs.get("tokenizer")
        split_by_character = inputs.get("split_by_character")
        split_by_character_only = inputs.get("split_by_character_only", False)
        chunk_overlap_token_size = inputs.get("chunk_overlap_token_size", 100)
        chunk_token_size = inputs.get("chunk_token_size", 1200)

        if not content or not tokenizer:
            return SubscriberReply(
                outputs={},
                strategy=MergeStrategy.IGNORE,
                error_message="Missing content or tokenizer in chunking inputs"
            )

        try:
            # Call the native function synchronously (since it's a CPU-bound sync function in LightRAG)
            chunks = self.chunking_func(
                tokenizer=tokenizer,
                content=content,
                split_by_character=split_by_character,
                split_by_character_only=split_by_character_only,
                chunk_overlap_token_size=chunk_overlap_token_size,
                chunk_token_size=chunk_token_size
            )
            
            return SubscriberReply(
                outputs={"chunks": chunks},
                strategy=self.strategy,
                weight=self.weight,
                correlation_id=envelope.correlation_id
            )
        except Exception as e:
            logger.error(f"NativeChunkingSubscriber failed: {e}")
            return SubscriberReply(
                outputs={},
                strategy=MergeStrategy.IGNORE,
                error_code="CHUNKING_FAILED",
                error_message=str(e),
                correlation_id=envelope.correlation_id
            )