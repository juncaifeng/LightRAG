# Go subscriber example for `rag.insert.chunking`

```go
package main

import (
	"context"
	"encoding/json"
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
		SubscriberId:   "my-chunker",
		Topic:          "rag.insert.chunking",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Registered subscriber: my-chunker")

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-chunker",
		Topic:        "rag.insert.chunking",
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

		content := string(envelope.Inputs["content"])
		log.Printf("Received chunking request: %d bytes, correlation=%s",
			len(content), envelope.CorrelationId)

		// TODO: Your custom chunking logic here
		chunks := []map[string]string{
			{"content": content},
		}

		result, _ := json.Marshal(chunks)
		outputs := map[string][]byte{"chunks": result}

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-chunker",
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
