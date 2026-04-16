# Python Example — rag.insert.ocr

```python
import grpc

import lightrag_eventbus_pb2 as pb2
import lightrag_eventbus_pb2_grpc as pb2_grpc


def main():
    channel = grpc.insecure_channel("localhost:50051")
    stub = pb2_grpc.EventBusStub(channel)

    # 1. Register subscriber
    stub.RegisterSubscriber(pb2.RegisterRequest(
        subscriber_id="my-ocr-service",
        topic="rag.insert.ocr",
        max_concurrency=5,
    ))
    print("Registered subscriber: my-ocr-service")

    # 2. Subscribe to event stream
    stream = stub.Subscribe(pb2.RegisterRequest(
        subscriber_id="my-ocr-service",
        topic="rag.insert.ocr",
    ))

    # 3. Process events
    for envelope in stream:
        image_data = envelope.inputs["image"]
        print(f"Received OCR request: {len(image_data)} bytes, "
              f"correlation={envelope.correlation_id}")

        # TODO: Your OCR logic here
        text = "OCR result placeholder"

        outputs = {"text": text.encode("utf-8")}

        # 4. Respond with results
        stub.Respond(pb2.SubscriberReply(
            correlation_id=envelope.correlation_id,
            subscriber_id="my-ocr-service",
            outputs=outputs,
            strategy=pb2.SubscriberReply.REPLACE,
            weight=10,
            health=pb2.SubscriberReply.HEALTHY,
        ))


if __name__ == "__main__":
    main()
```
