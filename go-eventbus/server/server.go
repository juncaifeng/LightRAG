package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "github.com/HKUDS/LightRAG/go-eventbus/proto/eventbus/v1"
)

// Subscriber represents an active subscriber stream
type Subscriber struct {
	ID       string
	Topic    string
	Stream   pb.EventBus_SubscribeServer
	IsActive bool
}

// GatherTask represents an active PublishAndWait request
type GatherTask struct {
	CorrelationID string
	Responses     chan *pb.SubscriberReply
	ExpectedCount int
}

type EventBusServer struct {
	pb.UnimplementedEventBusServer
	
	// mu protects subscribers
	mu sync.RWMutex
	// topic -> list of subscribers
	subscribers map[string][]*Subscriber

	// taskMu protects gatherTasks
	taskMu sync.RWMutex
	// correlation_id -> GatherTask
	gatherTasks map[string]*GatherTask
}

func NewEventBusServer() *EventBusServer {
	return &EventBusServer{
		subscribers: make(map[string][]*Subscriber),
		gatherTasks: make(map[string]*GatherTask),
	}
}

// RegisterSubscriber handles subscriber registration
func (s *EventBusServer) RegisterSubscriber(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Registering subscriber: %s for topic: %s", req.SubscriberId, req.Topic)
	return &pb.RegisterResponse{
		Success: true,
		Message: "Subscriber registered successfully",
	}, nil
}

// Subscribe handles the one-way server streaming to subscribers
func (s *EventBusServer) Subscribe(req *pb.RegisterRequest, stream pb.EventBus_SubscribeServer) error {
	log.Printf("Subscriber connected to stream: %s for topic: %s", req.SubscriberId, req.Topic)
	
	sub := &Subscriber{
		ID:       req.SubscriberId,
		Topic:    req.Topic,
		Stream:   stream,
		IsActive: true,
	}

	s.mu.Lock()
	s.subscribers[req.Topic] = append(s.subscribers[req.Topic], sub)
	s.mu.Unlock()

	// Keep connection open
	<-stream.Context().Done()
	
	log.Printf("Subscriber disconnected: %s", req.SubscriberId)
	
	// Remove subscriber
	s.mu.Lock()
	subs := s.subscribers[req.Topic]
	for i, sub := range subs {
		if sub.ID == req.SubscriberId {
			s.subscribers[req.Topic] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	
	return nil
}

// Respond receives results from subscribers
func (s *EventBusServer) Respond(ctx context.Context, reply *pb.SubscriberReply) (*pb.RegisterResponse, error) {
	s.taskMu.RLock()
	task, exists := s.gatherTasks[reply.CorrelationId]
	s.taskMu.RUnlock()

	if !exists {
		log.Printf("Received reply for unknown/expired correlation_id: %s", reply.CorrelationId)
		return &pb.RegisterResponse{Success: false, Message: "Task not found or expired"}, nil
	}

	// Non-blocking send if possible
	select {
	case task.Responses <- reply:
		// Sent successfully
	default:
		log.Printf("Task channel full for correlation_id: %s", reply.CorrelationId)
	}

	return &pb.RegisterResponse{Success: true, Message: "Response received"}, nil
}
