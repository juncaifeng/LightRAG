import grpc

import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc


def main():
    channel = grpc.insecure_channel("localhost:50051")
    stub = pb2_grpc.EventBusStub(channel)

    # 1. Register subscriber
    stub.RegisterSubscriber(pb2.RegisterRequest(
        subscriber_id="my-responder",
        topic="rag.query.response",
        max_concurrency=10,
    ))
    print("Registered subscriber: my-responder")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-responder",
        topic="rag.query.response",
    ))

    # 3. Process events
    for envelope in stream:
        query = envelope.inputs["query"].decode("utf-8")
        context = envelope.inputs["context"].decode("utf-8")
        print(f"Received response request: query={query!r}, context={len(context)} bytes")

        # TODO: Your LLM response generation logic here
        response = "Generated response placeholder"

        outputs = {"response": response.encode("utf-8")}

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-responder",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
