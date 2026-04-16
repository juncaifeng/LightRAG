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
        subscriber_id="my-keyword-extractor",
        topic="rag.query.keyword_extraction",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-keyword-extractor")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-keyword-extractor",
        topic="rag.query.keyword_extraction",
    ))

    # 3. Process events
    for envelope in stream:
        query = envelope.inputs["query"].decode("utf-8")
        print(f"Received keyword extraction request: {query!r}")

        # TODO: Your keyword extraction logic here
        hl_keywords = ["concept1", "concept2"]
        ll_keywords = ["entity1", "entity2"]

        outputs = {
            "hl_keywords": json.dumps(hl_keywords).encode("utf-8"),
            "ll_keywords": json.dumps(ll_keywords).encode("utf-8"),
        }

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-keyword-extractor",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
```
