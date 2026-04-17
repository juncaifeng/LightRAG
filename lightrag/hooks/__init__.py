from .base import HookDispatcher, EventEnvelope, SubscriberReply, MergeStrategy, LocalSubscriberAdapter
from .local import LocalMemoryDispatcher
from .adapters import (
    NativeChunkingSubscriber,
    NativeKeywordExtractionSubscriber,
    NativeQueryExpansionSubscriber,
    NativeKGSearchSubscriber,
    NativeVectorSearchSubscriber,
    NativeRerankSubscriber,
    NativeResponseSubscriber,
    NativeEmbeddingSubscriber,
)

__all__ = [
    "HookDispatcher",
    "EventEnvelope",
    "SubscriberReply",
    "MergeStrategy",
    "LocalSubscriberAdapter",
    "LocalMemoryDispatcher",
    "GrpcEventBusDispatcher",
    "NativeChunkingSubscriber",
    "NativeKeywordExtractionSubscriber",
    "NativeQueryExpansionSubscriber",
    "NativeKGSearchSubscriber",
    "NativeVectorSearchSubscriber",
    "NativeRerankSubscriber",
    "NativeResponseSubscriber",
    "NativeEmbeddingSubscriber",
]


def __getattr__(name):
    if name == "GrpcEventBusDispatcher":
        from .grpc_bus import GrpcEventBusDispatcher
        return GrpcEventBusDispatcher
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")