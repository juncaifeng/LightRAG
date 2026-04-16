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
		SubscriberId:   "my-thesaurus-expander",
		Topic:          "rag.query.query_expansion",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-thesaurus-expander",
		Topic:        "rag.query.query_expansion",
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

		var hlKeywords, llKeywords []string
		json.Unmarshal(envelope.Inputs["hl_keywords"], &hlKeywords)
		json.Unmarshal(envelope.Inputs["ll_keywords"], &llKeywords)
		log.Printf("Received expansion request: hl=%v, ll=%v", hlKeywords, llKeywords)

		// TODO: Query your thesaurus service here
		expandedHl := append(hlKeywords, "expanded_concept1")
		expandedLl := append(llKeywords, "expanded_entity1")

		hlResult, _ := json.Marshal(expandedHl)
		llResult, _ := json.Marshal(expandedLl)

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-thesaurus-expander",
			Outputs: map[string][]byte{
				"expanded_hl_keywords": hlResult,
				"expanded_ll_keywords": llResult,
			},
			Strategy: pb.SubscriberReply_APPEND,
			Weight:   10,
			Health:   pb.SubscriberReply_HEALTHY,
		})
		if err != nil {
			log.Printf("Failed to respond: %v", err)
		}
	}
}
```
