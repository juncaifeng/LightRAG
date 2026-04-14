package server

import (
	"context"
	"log"
	"time"
	"fmt"

	pb "github.com/HKUDS/LightRAG/go-eventbus/proto/eventbus/v1"
)

// PublishAndWait scatter-gather engine
func (s *EventBusServer) PublishAndWait(ctx context.Context, env *pb.EventEnvelope) (*pb.SubscriberReply, error) {
	s.mu.RLock()
	subs, exists := s.subscribers[env.Topic]
	s.mu.RUnlock()

	if !exists || len(subs) == 0 {
		log.Printf("No subscribers found for topic: %s", env.Topic)
		// Return empty reply indicating no processing
		return &pb.SubscriberReply{
			CorrelationId: env.CorrelationId,
			Outputs:       make(map[string][]byte),
			Strategy:      pb.SubscriberReply_IGNORE,
		}, nil
	}

	// Create a gather task
	task := &GatherTask{
		CorrelationID: env.CorrelationId,
		Responses:     make(chan *pb.SubscriberReply, len(subs)),
		ExpectedCount: len(subs),
	}

	s.taskMu.Lock()
	s.gatherTasks[env.CorrelationId] = task
	s.taskMu.Unlock()

	defer func() {
		// Cleanup task after function exits
		s.taskMu.Lock()
		delete(s.gatherTasks, env.CorrelationId)
		s.taskMu.Unlock()
	}()

	// Fan-out to all active subscribers
	log.Printf("Scattering event %s to %d subscribers", env.CorrelationId, len(subs))
	for _, sub := range subs {
		go func(subscriber *Subscriber) {
			err := subscriber.Stream.Send(env)
			if err != nil {
				log.Printf("Error sending to subscriber %s: %v", subscriber.ID, err)
			}
		}(sub)
	}

	// Wait and Gather (with deadline)
	var deadline time.Time
	if env.DeadlineTimestamp > 0 {
		deadline = time.UnixMilli(env.DeadlineTimestamp)
	} else {
		// Default 5 seconds if not provided
		deadline = time.Now().Add(5 * time.Second)
	}

	ctxDeadline, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	mergedReply := &pb.SubscriberReply{
		CorrelationId: env.CorrelationId,
		Outputs:       make(map[string][]byte),
		Strategy:      pb.SubscriberReply_APPEND, // default
		Weight:        0,
	}

	responsesReceived := 0

	for responsesReceived < task.ExpectedCount {
		select {
		case <-ctxDeadline.Done():
			log.Printf("Deadline exceeded for event %s, gathering partial results", env.CorrelationId)
			mergedReply.ErrorCode = "TIMEOUT"
			return mergedReply, nil

		case reply := <-task.Responses:
			responsesReceived++
			s.mergeResponses(mergedReply, reply)
		}
	}

	log.Printf("Successfully gathered %d responses for event %s", responsesReceived, env.CorrelationId)
	return mergedReply, nil
}

// mergeResponses executes the mechanical merge logic based on Subscriber metadata
func (s *EventBusServer) mergeResponses(base *pb.SubscriberReply, incoming *pb.SubscriberReply) {
	if incoming.Strategy == pb.SubscriberReply_IGNORE {
		return
	}

	if incoming.Strategy == pb.SubscriberReply_REPLACE {
		if incoming.Weight > base.Weight {
			// Replace outputs entirely
			base.Outputs = incoming.Outputs
			base.Weight = incoming.Weight
			base.Strategy = pb.SubscriberReply_REPLACE
		}
		return
	}

	if incoming.Strategy == pb.SubscriberReply_APPEND {
		// If base is currently REPLACE and has higher weight, ignore APPEND
		if base.Strategy == pb.SubscriberReply_REPLACE && base.Weight > 0 {
			return
		}

		// Simple byte append for outputs (in a real system, might parse JSON/Proto arrays)
		for k, v := range incoming.Outputs {
			existing, exists := base.Outputs[k]
			if exists {
				// Very basic concatenation for MVP
				// In reality, this would append JSON arrays or Protobuf repeated fields properly
				combined := append(existing, []byte(",")...)
				combined = append(combined, v...)
				base.Outputs[k] = combined
			} else {
				base.Outputs[k] = v
			}
		}
	}
}
