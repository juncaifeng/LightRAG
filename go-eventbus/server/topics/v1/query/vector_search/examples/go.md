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
		SubscriberId:   "my-vector-searcher",
		Topic:          "rag.query.vector_search",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-vector-searcher",
		Topic:        "rag.query.vector_search",
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
		log.Printf("Received vector search request: %q", query)

		// TODO: Query your vector database here
		chunks := []map[string]interface{}{
			{"chunk_id": "chunk-001", "content": "Example chunk content", "file_path": "/docs/example.txt"},
		}

		result, _ := json.Marshal(chunks)

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-vector-searcher",
			Outputs: map[string][]byte{
				"chunks": result,
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
