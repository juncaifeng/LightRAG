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
        subscriber_id="my-kg-searcher",
        topic="rag.query.kg_search",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-kg-searcher")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-kg-searcher",
        topic="rag.query.kg_search",
    ))

    # 3. Process events
    for envelope in stream:
        hl_keywords = json.loads(envelope.inputs["hl_keywords"].decode("utf-8"))
        ll_keywords = json.loads(envelope.inputs["ll_keywords"].decode("utf-8"))
        print(f"Received KG search request: hl={hl_keywords}, ll={ll_keywords}")

        # TODO: Query your knowledge graph here
        entities = [{"entity_name": "example", "entity_type": "PERSON"}]
        relations = [{"src_id": "A", "tgt_id": "B", "description": "related"}]

        outputs = {
            "entities": json.dumps(entities).encode("utf-8"),
            "relations": json.dumps(relations).encode("utf-8"),
        }

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-kg-searcher",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
```
