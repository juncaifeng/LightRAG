package server

import (
	"context"
	"log"
	"sync"
	"time"

	pb "github.com/juncaifeng/LightRAG/go-eventbus/sdk/v1/go"
)

// Subscriber represents an active subscriber stream
type Subscriber struct {
	ID          string
	Topic       string
	Stream      pb.EventBus_SubscribeServer
	IsActive    bool
	ConnectedAt time.Time
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

	// metrics for observability
	metrics *EventMetrics

	// serviceCache for service instance registration and discovery
	serviceCache *ServiceCache
}

func NewEventBusServer(serviceCache *ServiceCache) *EventBusServer {
	return &EventBusServer{
		subscribers:  make(map[string][]*Subscriber),
		gatherTasks:  make(map[string]*GatherTask),
		metrics:      NewEventMetrics(),
		serviceCache: serviceCache,
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
		ID:          req.SubscriberId,
		Topic:       req.Topic,
		Stream:      stream,
		IsActive:    true,
		ConnectedAt: time.Now(),
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

// --- Service Instance Registration RPCs ---

const defaultServiceTTLSeconds int32 = 30

// RegisterService registers a service instance.
func (s *EventBusServer) RegisterService(ctx context.Context, req *pb.RegisterServiceRequest) (*pb.RegisterServiceResponse, error) {
	inst := req.Instance
	if inst == nil || inst.ServiceName == "" || inst.InstanceId == "" || inst.Address == "" {
		return &pb.RegisterServiceResponse{
			Success: false,
			Message: "service_name, instance_id, and address are required",
		}, nil
	}

	ttl := req.TtlSeconds
	if ttl <= 0 {
		ttl = defaultServiceTTLSeconds
	}

	now := time.Now()
	info := &ServiceInstanceInfo{
		ServiceName:   inst.ServiceName,
		InstanceID:    inst.InstanceId,
		Address:       inst.Address,
		Version:       inst.Version,
		Metadata:      inst.Metadata,
		Status:        "healthy",
		RegisteredAt:  now,
		LastHeartbeat: now,
		ExpiresAt:     now.Add(time.Duration(ttl) * time.Second),
	}

	if err := s.serviceCache.Register(info); err != nil {
		log.Printf("RegisterService error: %v", err)
		return &pb.RegisterServiceResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Service registered: %s/%s at %s (ttl=%ds)", inst.ServiceName, inst.InstanceId, inst.Address, ttl)
	return &pb.RegisterServiceResponse{
		Success:    true,
		Message:    "Service instance registered",
		TtlSeconds: ttl,
	}, nil
}

// Heartbeat refreshes a service instance's TTL.
func (s *EventBusServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	if req.ServiceName == "" || req.InstanceId == "" {
		return &pb.HeartbeatResponse{Success: false}, nil
	}

	ttl := defaultServiceTTLSeconds
	if err := s.serviceCache.Heartbeat(req.ServiceName, req.InstanceId, ttl); err != nil {
		return &pb.HeartbeatResponse{Success: false}, nil
	}

	expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
	return &pb.HeartbeatResponse{
		Success:   true,
		ExpiresAt: expiresAt.UnixMilli(),
	}, nil
}

// UnregisterService removes a service instance.
func (s *EventBusServer) UnregisterService(ctx context.Context, req *pb.UnregisterServiceRequest) (*pb.RegisterResponse, error) {
	if req.ServiceName == "" || req.InstanceId == "" {
		return &pb.RegisterResponse{Success: false, Message: "service_name and instance_id are required"}, nil
	}

	if err := s.serviceCache.Unregister(req.ServiceName, req.InstanceId); err != nil {
		return &pb.RegisterResponse{Success: false, Message: err.Error()}, nil
	}

	log.Printf("Service unregistered: %s/%s", req.ServiceName, req.InstanceId)
	return &pb.RegisterResponse{Success: true, Message: "Service instance unregistered"}, nil
}

// ListServiceInstances returns registered service instances.
func (s *EventBusServer) ListServiceInstances(ctx context.Context, req *pb.ListServiceInstancesRequest) (*pb.ListServiceInstancesResponse, error) {
	instances := s.serviceCache.List(req.ServiceName)

	result := make([]*pb.ServiceInstance, 0, len(instances))
	for _, inst := range instances {
		result = append(result, &pb.ServiceInstance{
			ServiceName: inst.ServiceName,
			InstanceId:  inst.InstanceID,
			Address:     inst.Address,
			Version:     inst.Version,
			Metadata:    inst.Metadata,
		})
	}

	return &pb.ListServiceInstancesResponse{Instances: result}, nil
}

// --- Helper methods for HTTP API ---

func (s *EventBusServer) GetSubscriberCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, subs := range s.subscribers {
		count += len(subs)
	}
	return count
}

func (s *EventBusServer) GetTopicCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers)
}

func (s *EventBusServer) GetInFlightTaskCount() int {
	s.taskMu.RLock()
	defer s.taskMu.RUnlock()
	return len(s.gatherTasks)
}

func (s *EventBusServer) GetSubscribersSnapshot() []SubscriberInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []SubscriberInfo
	for _, subs := range s.subscribers {
		for _, sub := range subs {
			result = append(result, SubscriberInfo{
				ID:          sub.ID,
				Topic:       sub.Topic,
				IsActive:    sub.IsActive,
				ConnectedAt: sub.ConnectedAt.Format(time.RFC3339),
			})
		}
	}
	return result
}

func (s *EventBusServer) GetTopicsSnapshot() []TopicInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []TopicInfo
	for topic, subs := range s.subscribers {
		result = append(result, TopicInfo{
			Name:            topic,
			SubscriberCount: len(subs),
		})
	}
	return result
}
