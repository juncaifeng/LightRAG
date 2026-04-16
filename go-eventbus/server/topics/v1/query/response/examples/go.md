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
		SubscriberId:   "my-responder",
		Topic:          "rag.query.response",
		MaxConcurrency: 10,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Subscribe to event stream
	stream, err := client.Subscribe(context.Background(), &pb.RegisterRequest{
		SubscriberId: "my-responder",
		Topic:        "rag.query.response",
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
		context := string(envelope.Inputs["context"])
		log.Printf("Received response request: query=%q, context=%d bytes", query, len(context))

		// TODO: Your LLM response generation logic here
		response := "Generated response placeholder"

		// 4. Respond with results
		_, err = client.Respond(context.Background(), &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "my-responder",
			Outputs: map[string][]byte{
				"response": []byte(response),
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
