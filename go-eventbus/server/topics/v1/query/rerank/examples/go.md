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
		SubscriberId:   "my-reranker",
		Topic:          "rag.query.rerank",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-reranker",
		Topic:        "rag.query.rerank",
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
		log.Printf("Received rerank request for query: %q", query)

		var chunks []map[string]interface{}
		json.Unmarshal(envelope.Inputs["chunks"], &chunks)

		// TODO: Your reranking logic here (e.g., BAAI/bge-reranker-v2-m3)
		rankedChunks := chunks

		result, _ := json.Marshal(rankedChunks)

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-reranker",
			Outputs: map[string][]byte{
				"ranked_chunks": result,
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
