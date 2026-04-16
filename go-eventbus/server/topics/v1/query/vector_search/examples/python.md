import grpc
import json

import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc


def main():
    channel = grpc.insecure_channel("localhost:50051")
    stub = pb2_grpc.EventBusStub(channel)

    # 1. Register subscriber
    stub.RegisterSubscriber(pb2.RegisterRequest(
        subscriber_id="my-vector-searcher",
        topic="rag.query.vector_search",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-vector-searcher")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-vector-searcher",
        topic="rag.query.vector_search",
    ))

    # 3. Process events
    for envelope in stream:
        query = envelope.inputs["query"].decode("utf-8")
        print(f"Received vector search request: {query!r}")

        # TODO: Query your vector database here
        chunks = [
            {"chunk_id": "chunk-001", "content": "Example chunk content", "file_path": "/docs/example.txt"},
        ]

        outputs = {"chunks": json.dumps(chunks).encode("utf-8")}

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-vector-searcher",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
