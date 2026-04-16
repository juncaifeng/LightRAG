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
		SubscriberId:   "my-kg-searcher",
		Topic:          "rag.query.kg_search",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-kg-searcher",
		Topic:        "rag.query.kg_search",
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
		log.Printf("Received KG search request: hl=%v, ll=%v", hlKeywords, llKeywords)

		// TODO: Query your knowledge graph here
		entities := []map[string]interface{}{{"entity_name": "example", "entity_type": "PERSON"}}
		relations := []map[string]interface{}{{"src_id": "A", "tgt_id": "B", "description": "related"}}

		entityResult, _ := json.Marshal(entities)
		relationResult, _ := json.Marshal(relations)

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-kg-searcher",
			Outputs: map[string][]byte{
				"entities":  entityResult,
				"relations": relationResult,
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
