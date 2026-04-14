package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	pb "github.com/HKUDS/LightRAG/go-eventbus/proto/eventbus/v1"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewEventBusClient(conn)
	
	// Register subscriber
	req := &pb.RegisterRequest{
		SubscriberId: "go-dummy-synonym-service",
		Topic:        "rag.query.query_expansion",
		Capabilities: []string{"default"},
		MaxConcurrency: 10,
	}

	_, err = c.RegisterSubscriber(context.Background(), req)
	if err != nil {
		log.Fatalf("Register failed: %v", err)
	}
	
	log.Println("Registered dummy subscriber")

	// Subscribe to event stream
	stream, err := c.Subscribe(context.Background(), req)
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	for {
		envelope, err := stream.Recv()
		if err != nil {
			log.Fatalf("Error receiving stream: %v", err)
		}

		log.Printf("Received event: %s, topic: %s", envelope.CorrelationId, envelope.Topic)

		// Simulate processing delay
		time.Sleep(50 * time.Millisecond)

		// Create a dummy expanded query result (e.g. synonym)
		outputs := make(map[string][]byte)
		outputs["expanded_queries"] = []byte(`["artificial intelligence"]`)

		reply := &pb.SubscriberReply{
			CorrelationId: envelope.CorrelationId,
			SubscriberId:  "go-dummy-synonym-service",
			Outputs:       outputs,
			Strategy:      pb.SubscriberReply_APPEND,
			Weight:        10,
			LatencyMs:     50,
			Health:        pb.SubscriberReply_HEALTHY,
		}

		// Respond asynchronously via the Respond RPC
		_, err = c.Respond(context.Background(), reply)
		if err != nil {
			log.Printf("Failed to respond to correlation %s: %v", envelope.CorrelationId, err)
		} else {
			log.Printf("Successfully responded to correlation %s", envelope.CorrelationId)
		}
	}
}
