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
]


def __getattr__(name):
    if name == "GrpcEventBusDispatcher":
        import sys
        # Ensure generated protobuf stubs are importable by their bare module names.
        # The auto-generated _grpc file does `import lightrag_eventbus_pb2` (bare),
        # but the stub lives inside the lightrag package. Pre-register in sys.modules.
        import lightrag.lightrag_eventbus_pb2 as pb2_mod
        sys.modules["lightrag_eventbus_pb2"] = pb2_mod
        import lightrag.lightrag_eventbus_pb2_grpc as grpc_mod
        sys.modules["lightrag_eventbus_pb2_grpc"] = grpc_mod
        from .grpc_bus import GrpcEventBusDispatcher
        return GrpcEventBusDispatcher
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")