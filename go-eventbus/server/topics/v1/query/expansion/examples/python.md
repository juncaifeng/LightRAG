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
        subscriber_id="my-thesaurus-expander",
        topic="rag.query.query_expansion",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-thesaurus-expander")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-thesaurus-expander",
        topic="rag.query.query_expansion",
    ))

    # 3. Process events
    for envelope in stream:
        hl_keywords = json.loads(envelope.inputs["hl_keywords"].decode("utf-8"))
        ll_keywords = json.loads(envelope.inputs["ll_keywords"].decode("utf-8"))
        config = json.loads(envelope.inputs.get("expansion_config", b"{}").decode("utf-8"))
        print(f"Received expansion request: hl={hl_keywords}, ll={ll_keywords}")

        # TODO: Query your thesaurus service here
        expanded_hl = hl_keywords + ["expanded_concept1"]
        expanded_ll = ll_keywords + ["expanded_entity1"]

        outputs = {
            "expanded_hl_keywords": json.dumps(expanded_hl).encode("utf-8"),
            "expanded_ll_keywords": json.dumps(expanded_ll).encode("utf-8"),
        }

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-thesaurus-expander",
            outputs=outputs,
            strategy=pb2.SubscriberReply.APPEND,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
```
