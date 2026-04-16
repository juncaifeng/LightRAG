# Go Example — rag.insert.ocr

```go
package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	pb "github.com/juncaifeng/LightRAG/go-eventbus/proto/eventbus/v1"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewEventBusClient(conn)

	// 1. Register subscriber
	_, err = client.RegisterSubscriber(context.Background(), &pb.RegisterRequest{
		SubscriberId:   "my-ocr-service",
		Topic:          "rag.insert.ocr",
		MaxConcurrency: 5,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Registered subscriber: my-ocr-service")

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-ocr-service",
		Topic:        "rag.insert.ocr",
	})
	if err != nil {
		log.Fatal(err)
	}

	// 3. Process events
	for {
		envelope, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}

		imageData := envelope.Inputs["image"]
		log.Printf("Received OCR request: %d bytes, correlation=%s",
			len(imageData), envelope.CorrelationId)

		// TODO: Your OCR logic here
		text := "OCR result placeholder"

		outputs := map[string][]byte{"text": []byte(text)}

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-ocr-service",
			Outputs:       outputs,
			Strategy:      pb.SubscriberReply_REPLACE,
			Weight:        10,
			Health:        pb.SubscriberReply_HEALTHY,
		})
		if err != nil {
			log.Printf("Failed to respond: %v", err)
		}
	}
}
```
