package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// --- JSON response types ---

type StatusResponse struct {
	Status               string  `json:"status"`
	Uptime               string  `json:"uptime"`
	UptimeSeconds        float64 `json:"uptime_seconds"`
	TotalSubscribers     int     `json:"total_subscribers"`
	ActiveTopics         int     `json:"active_topics"`
	InFlightTasks        int     `json:"in_flight_tasks"`
	TotalEventsPublished int64   `json:"total_events_published"`
	TotalResponses       int64   `json:"total_responses"`
	TotalTimeouts        int64   `json:"total_timeouts"`
	EventsPerSecond      float64 `json:"events_per_second"`
	TimeoutRate          float64 `json:"timeout_rate"`
	WsClients            int     `json:"ws_clients"`
}

type SubscriberInfo struct {
	ID          string `json:"id"`
	Topic       string `json:"topic"`
	IsActive    bool   `json:"is_active"`
	ConnectedAt string `json:"connected_at"`
}

type TopicInfo struct {
	Name            string `json:"name"`
	SubscriberCount int    `json:"subscriber_count"`
}

type RecentEventsResponse struct {
	Events []EventRecord `json:"events"`
}

// --- CORS middleware ---

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- WebSocket upgrader ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for local dev
	},
}

// StartHTTPServer starts the HTTP API + WebSocket server on the given address.
func StartHTTPServer(busServer *EventBusServer, addr string) error {
	mux := http.NewServeMux()

	// REST endpoints
	mux.HandleFunc("/api/status", busServer.handleStatus)
	mux.HandleFunc("/api/subscribers", busServer.handleSubscribers)
	mux.HandleFunc("/api/topics", busServer.handleTopics)
	mux.HandleFunc("/api/events/recent", busServer.handleRecentEvents)

	// Topic schema registry
	mux.HandleFunc("/api/topics/schemas", handleTopicSchemas)
	mux.HandleFunc("/api/topics/schemas/", handleTopicSchema)

	// WebSocket endpoint
	mux.HandleFunc("/ws/events", busServer.handleWSEvents)

	server := &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}

	log.Printf("Starting HTTP dashboard server on %s", addr)
	return server.ListenAndServe()
}

// --- Handlers ---

func (s *EventBusServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	snap := s.metrics.GetSnapshot()

	resp := StatusResponse{
		Status:               "running",
		Uptime:               snap.Uptime,
		UptimeSeconds:        snap.UptimeSeconds,
		TotalSubscribers:     s.GetSubscriberCount(),
		ActiveTopics:         s.GetTopicCount(),
		InFlightTasks:        s.GetInFlightTaskCount(),
		TotalEventsPublished: snap.TotalEventsPublished,
		TotalResponses:       snap.TotalResponses,
		TotalTimeouts:        snap.TotalTimeouts,
		EventsPerSecond:      snap.EventsPerSecond,
		TimeoutRate:          snap.TimeoutRate,
		WsClients:            s.metrics.GetWSClientCount(),
	}
	writeJSON(w, resp)
}

func (s *EventBusServer) handleSubscribers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.GetSubscribersSnapshot())
}

func (s *EventBusServer) handleTopics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.GetTopicsSnapshot())
}

func (s *EventBusServer) handleRecentEvents(w http.ResponseWriter, r *http.Request) {
	n := 100
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 && parsed <= ringBufferSize {
			n = parsed
		}
	}
	events := s.metrics.GetRecentEvents(n)
	writeJSON(w, RecentEventsResponse{Events: events})
}

func (s *EventBusServer) handleWSEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Send recent events as initial payload
	recent := s.metrics.GetRecentEvents(50)
	if len(recent) > 0 {
		data, _ := json.Marshal(RecentEventsResponse{Events: recent})
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}

	// Register client
	s.metrics.AddWSClient(conn)
	log.Printf("WebSocket client connected (total: %d)", s.metrics.GetWSClientCount())

	// Block until disconnect
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	// Unregister on disconnect
	s.metrics.RemoveWSClient(conn)
	log.Printf("WebSocket client disconnected (total: %d)", s.metrics.GetWSClientCount())
}

// --- Topic Schema handlers ---

func handleTopicSchemas(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, GetTopicSchemas())
}

func handleTopicSchema(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/topics/schemas/")
	if name == "" {
		http.Error(w, "topic name required", http.StatusBadRequest)
		return
	}
	schema := GetTopicSchema(name)
	if schema == nil {
		http.Error(w, "topic not found", http.StatusNotFound)
		return
	}
	writeJSON(w, schema)
}
