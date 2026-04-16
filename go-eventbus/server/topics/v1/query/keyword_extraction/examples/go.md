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
		SubscriberId:   "my-keyword-extractor",
		Topic:          "rag.query.keyword_extraction",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-keyword-extractor",
		Topic:        "rag.query.keyword_extraction",
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

		query := string(envelope.Inputs["query"])
		log.Printf("Received keyword extraction request: %q", query)

		// TODO: Your keyword extraction logic here
		hlKeywords := `["concept1", "concept2"]`
		llKeywords := `["entity1", "entity2"]`

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-keyword-extractor",
			Outputs: map[string][]byte{
				"hl_keywords": []byte(hlKeywords),
				"ll_keywords": []byte(llKeywords),
			},
			Strategy: pb.SubscriberReply_REPLACE,
			Weight:   10,
			Health:   pb.SubscriberReply_HEALTHY,
		})
		if err != nil {
			log.Printf("Failed to respond: %v", err)
		}
	}
}
```
