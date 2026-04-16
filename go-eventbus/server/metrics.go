package server

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// EventRecord is stored in the ring buffer and broadcast via WebSocket.
type EventRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"` // "publish", "scatter", "response", "timeout", "gather_complete"
	Topic        string    `json:"topic"`
	CorrelationID string   `json:"correlation_id"`
	SubscriberID string   `json:"subscriber_id,omitempty"`
	LatencyMs    int32     `json:"latency_ms,omitempty"`
	Strategy     string    `json:"strategy,omitempty"`
	Status       string    `json:"status"` // "success", "timeout", "error"
	SubscriberCount int    `json:"subscriber_count,omitempty"`
	ResponseCount   int    `json:"response_count,omitempty"`
}

// MetricsSnapshot is the JSON-serializable snapshot for HTTP API.
type MetricsSnapshot struct {
	StartTime           time.Time `json:"start_time"`
	Uptime              string    `json:"uptime"`
	UptimeSeconds       float64   `json:"uptime_seconds"`
	TotalEventsPublished int64    `json:"total_events_published"`
	TotalResponses      int64     `json:"total_responses"`
	TotalTimeouts       int64     `json:"total_timeouts"`
	EventsPerSecond     float64   `json:"events_per_second"`
	TimeoutRate         float64   `json:"timeout_rate"`
}

const ringBufferSize = 200

// wsConn wraps a websocket connection with a write mutex.
type wsConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// EventMetrics tracks observability data for the event bus.
type EventMetrics struct {
	startTime time.Time

	// Atomic counters (hot path, no lock needed)
	totalEventsPublished atomic.Int64
	totalResponses       atomic.Int64
	totalTimeouts        atomic.Int64

	// Ring buffer for recent events
	mu          sync.RWMutex
	events      [ringBufferSize]EventRecord
	head        int // next write position
	count       int // total events written (capped at ringBufferSize)

	// WebSocket clients
	wsMu      sync.RWMutex
	wsClients map[*wsConn]bool
}

// NewEventMetrics creates a new metrics accumulator.
func NewEventMetrics() *EventMetrics {
	return &EventMetrics{
		startTime:  time.Now(),
		wsClients:  make(map[*wsConn]bool),
	}
}

// --- Recording methods (called from scatter-gather hot path) ---

func (m *EventMetrics) RecordPublish(topic, correlationID string) {
	m.totalEventsPublished.Add(1)
	rec := EventRecord{
		Timestamp:     time.Now(),
		Type:          "publish",
		Topic:         topic,
		CorrelationID: correlationID,
		Status:        "success",
	}
	m.pushEvent(rec)
}

func (m *EventMetrics) RecordScatter(topic, correlationID string, subscriberCount int) {
	rec := EventRecord{
		Timestamp:        time.Now(),
		Type:             "scatter",
		Topic:            topic,
		CorrelationID:    correlationID,
		Status:           "success",
		SubscriberCount:  subscriberCount,
	}
	m.pushEvent(rec)
}

func (m *EventMetrics) RecordResponse(correlationID, subscriberID string, latencyMs int32, strategy string) {
	m.totalResponses.Add(1)
	rec := EventRecord{
		Timestamp:     time.Now(),
		Type:          "response",
		CorrelationID: correlationID,
		SubscriberID:  subscriberID,
		LatencyMs:     latencyMs,
		Strategy:      strategy,
		Status:        "success",
	}
	m.pushEvent(rec)
}

func (m *EventMetrics) RecordTimeout(correlationID, topic string) {
	m.totalTimeouts.Add(1)
	rec := EventRecord{
		Timestamp:     time.Now(),
		Type:          "timeout",
		Topic:         topic,
		CorrelationID: correlationID,
		Status:        "timeout",
	}
	m.pushEvent(rec)
}

func (m *EventMetrics) RecordGatherComplete(correlationID, topic string, responseCount int) {
	rec := EventRecord{
		Timestamp:      time.Now(),
		Type:           "gather_complete",
		Topic:          topic,
		CorrelationID:  correlationID,
		Status:         "success",
		ResponseCount:  responseCount,
	}
	m.pushEvent(rec)
}

// --- Ring buffer ---

func (m *EventMetrics) pushEvent(rec EventRecord) {
	m.mu.Lock()
	m.events[m.head] = rec
	m.head = (m.head + 1) % ringBufferSize
	if m.count < ringBufferSize {
		m.count++
	}
	m.mu.Unlock()

	// Broadcast to WebSocket clients (non-blocking)
	m.broadcastWS(rec)
}

// GetRecentEvents returns up to n most recent events (newest last).
func (m *EventMetrics) GetRecentEvents(n int) []EventRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n > m.count {
		n = m.count
	}
	result := make([]EventRecord, n)
	for i := 0; i < n; i++ {
		idx := (m.head - n + i + ringBufferSize) % ringBufferSize
		result[i] = m.events[idx]
	}
	return result
}

// GetSnapshot returns a JSON-serializable metrics snapshot.
func (m *EventMetrics) GetSnapshot() MetricsSnapshot {
	uptime := time.Since(m.startTime)
	uptimeSeconds := uptime.Seconds()

	total := m.totalEventsPublished.Load()
	timeouts := m.totalTimeouts.Load()

	var eventsPerSecond float64
	var timeoutRate float64
	if uptimeSeconds > 0 {
		eventsPerSecond = float64(total) / uptimeSeconds
	}
	if total > 0 {
		timeoutRate = float64(timeouts) / float64(total) * 100
	}

	return MetricsSnapshot{
		StartTime:            m.startTime,
		Uptime:               fmt.Sprintf("%.0fs", uptimeSeconds),
		UptimeSeconds:        uptimeSeconds,
		TotalEventsPublished: total,
		TotalResponses:       m.totalResponses.Load(),
		TotalTimeouts:        timeouts,
		EventsPerSecond:      eventsPerSecond,
		TimeoutRate:          timeoutRate,
	}
}

// --- WebSocket management ---

func (m *EventMetrics) AddWSClient(conn *websocket.Conn) {
	wc := &wsConn{conn: conn}
	m.wsMu.Lock()
	m.wsClients[wc] = true
	m.wsMu.Unlock()
}

func (m *EventMetrics) RemoveWSClient(conn *websocket.Conn) {
	m.wsMu.Lock()
	for wc := range m.wsClients {
		if wc.conn == conn {
			delete(m.wsClients, wc)
			break
		}
	}
	m.wsMu.Unlock()
}

func (m *EventMetrics) broadcastWS(rec EventRecord) {
	m.wsMu.RLock()
	clients := make([]*wsConn, 0, len(m.wsClients))
	for c := range m.wsClients {
		clients = append(clients, c)
	}
	m.wsMu.RUnlock()

	data, err := json.Marshal(rec)
	if err != nil {
		return
	}
	for _, c := range clients {
		// Non-blocking: if client is slow, skip it
		go func(wc *wsConn) {
			wc.mu.Lock()
			_ = wc.conn.WriteMessage(websocket.TextMessage, data)
			wc.mu.Unlock()
		}(c)
	}
}

// GetWSClientCount returns the number of connected WebSocket clients.
func (m *EventMetrics) GetWSClientCount() int {
	m.wsMu.RLock()
	defer m.wsMu.RUnlock()
	return len(m.wsClients)
}
