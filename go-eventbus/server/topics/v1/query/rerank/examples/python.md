import grpc
import json

import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc


def main():
    channel = grpc.insecure_channel("localhost:50051")
    stub = pb2_grpc.EventBusStub(channel)

    # 1. Register subscriber
    stub.RegisterSubscriber(pb2.RegisterRequest(
        subscriber_id="my-reranker",
        topic="rag.query.rerank",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-reranker")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-reranker",
        topic="rag.query.rerank",
    ))

    # 3. Process events
    for envelope in stream:
        query = envelope.inputs["query"].decode("utf-8")
        chunks = json.loads(envelope.inputs["chunks"].decode("utf-8"))
        print(f"Received rerank request for query: {query!r}, {len(chunks)} chunks")

        # TODO: Your reranking logic here (e.g., BAAI/bge-reranker-v2-m3)
        ranked_chunks = chunks

        outputs = {"ranked_chunks": json.dumps(ranked_chunks).encode("utf-8")}

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-reranker",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
