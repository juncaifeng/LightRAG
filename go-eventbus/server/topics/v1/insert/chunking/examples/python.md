# Python subscriber example for `rag.insert.chunking`

```python
import grpc
import json

import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc


def main():
    channel = grpc.insecure_channel("localhost:50051")
    stub = pb2_grpc.EventBusStub(channel)

    # 1. Register subscriber
    stub.RegisterSubscriber(pb2.RegisterRequest(
        subscriber_id="my-chunker",
        topic="rag.insert.chunking",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-chunker")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-chunker",
        topic="rag.insert.chunking",
    ))

    # 3. Process events
    for envelope in stream:
        content = envelope.inputs["content"].decode("utf-8")
        print(f"Received chunking request: {len(content)} bytes, "
              f"correlation={envelope.correlation_id}")

        # TODO: Your custom chunking logic here
        chunks = [{"content": content}]

        outputs = {"chunks": json.dumps(chunks).encode("utf-8")}

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-chunker",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
```
