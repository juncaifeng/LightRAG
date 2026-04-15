from abc import ABC, abstractmethod
from typing import Dict, Any, Callable, Awaitable
from dataclasses import dataclass
import uuid

# --- Data Models (Python representations of Protobuf messages) ---

class MergeStrategy:
    APPEND = 0
    REPLACE = 1
    IGNORE = 2

@dataclass
class EventEnvelope:
    topic: str
    inputs: Dict[str, Any]
    correlation_id: str = ""
    trace_id: str = ""
    deadline_timestamp: int = 0
    priority: int = 0
    source_service: str = "lightrag-core"
    metadata: Dict[str, str] = None

    def __post_init__(self):
        if not self.correlation_id:
            self.correlation_id = str(uuid.uuid4())
        if self.metadata is None:
            self.metadata = {}

@dataclass
class SubscriberReply:
    outputs: Dict[str, Any]
    strategy: int = MergeStrategy.APPEND
    weight: int = 0
    correlation_id: str = ""
    subscriber_id: str = "local"
    partial_result: bool = False
    latency_ms: int = 0
    error_code: str = ""
    error_message: str = ""

# --- Core Interfaces ---

class HookDispatcher(ABC):
    """
    The main interface for the Event Bus Dispatcher.
    LightRAG core engine will use this to publish events and wait for gathered results.
    """
    
    @abstractmethod
    async def publish_and_wait(self, envelope: EventEnvelope) -> SubscriberReply:
        """
        Publishes an event to the bus and waits for the aggregated reply.
        """
        pass

    @abstractmethod
    def register_local_subscriber(self, topic: str, subscriber: 'LocalSubscriberAdapter'):
        """
        Registers a local Python function as a subscriber.
        """
        pass


class LocalSubscriberAdapter(ABC):
    """
    Adapter to wrap LightRAG's native Python functions into standard Subscribers.
    """
    def __init__(self, topic: str, subscriber_id: str, strategy: int = MergeStrategy.APPEND, weight: int = 0):
        self.topic = topic
        self.subscriber_id = subscriber_id
        self.strategy = strategy
        self.weight = weight
        self.is_local = True # Indicates this can be short-circuited in memory

    @abstractmethod
    async def process(self, envelope: EventEnvelope) -> SubscriberReply:
        """
        The actual logic to process the inputs and return the outputs.
        Must be implemented by specific adapters (e.g. NativeChunkingSubscriber).
        """
        pass