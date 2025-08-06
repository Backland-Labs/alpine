package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Backland-Labs/alpine/internal/logger"
)

// runSpecificEventHub manages run-specific event subscriptions
// It allows filtering global events to specific run IDs
type runSpecificEventHub struct {
	mu               sync.RWMutex
	subscribers      map[string][]chan WorkflowEvent // runID -> list of subscriber channels
	bufferSize       int                             // Size of event buffer per client
	maxClientsPerRun int                             // Maximum clients allowed per run
}

// newRunSpecificEventHub creates a new event hub for run-specific subscriptions
func newRunSpecificEventHub() *runSpecificEventHub {
	return &runSpecificEventHub{
		subscribers:      make(map[string][]chan WorkflowEvent),
		bufferSize:       defaultEventBufferSize,
		maxClientsPerRun: 100, // Default limit
	}
}

// newRunSpecificEventHubWithConfig creates a new event hub with custom configuration
func newRunSpecificEventHubWithConfig(bufferSize int, maxClientsPerRun int) *runSpecificEventHub {
	// Validate configuration
	if bufferSize <= 0 {
		bufferSize = defaultEventBufferSize
	}
	if maxClientsPerRun <= 0 {
		maxClientsPerRun = 100
	}

	return &runSpecificEventHub{
		subscribers:      make(map[string][]chan WorkflowEvent),
		bufferSize:       bufferSize,
		maxClientsPerRun: maxClientsPerRun,
	}
}

// subscribe adds a new subscriber for a specific run ID
func (hub *runSpecificEventHub) subscribe(runID string) (chan WorkflowEvent, error) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	// Check client limit
	if len(hub.subscribers[runID]) >= hub.maxClientsPerRun {
		return nil, fmt.Errorf("maximum %d clients reached for run %s", hub.maxClientsPerRun, runID)
	}

	ch := make(chan WorkflowEvent, hub.bufferSize)
	hub.subscribers[runID] = append(hub.subscribers[runID], ch)
	return ch, nil
}

// unsubscribe removes a subscriber channel for a run ID
func (hub *runSpecificEventHub) unsubscribe(runID string, ch chan WorkflowEvent) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	subs := hub.subscribers[runID]
	for i, sub := range subs {
		if sub == ch {
			// Remove this subscriber
			hub.subscribers[runID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty run entries
	if len(hub.subscribers[runID]) == 0 {
		delete(hub.subscribers, runID)
	}
}

// broadcast sends an event to all subscribers of a specific run
func (hub *runSpecificEventHub) broadcast(event WorkflowEvent) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	// Get subscribers for this run ID
	subs := hub.subscribers[event.RunID]
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// enhancedRunEventsHandler provides SSE endpoint for run-specific events with global event filtering
func (s *Server) enhancedRunEventsHandler(w http.ResponseWriter, r *http.Request, hub *runSpecificEventHub) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runID := r.PathValue("id")

	s.mu.Lock()
	_, exists := s.runs[runID]
	s.mu.Unlock()

	if !exists {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe to run-specific events from hub
	eventChan, err := hub.subscribe(runID)
	if err != nil {
		// Client limit reached
		http.Error(w, "Too many clients connected to this run", http.StatusServiceUnavailable)
		return
	}
	defer hub.unsubscribe(runID, eventChan)

	// Send initial connection event after successful subscription
	if _, err := fmt.Fprintf(w, "data: {\"type\":\"connected\",\"runId\":\"%s\"}\n\n", runID); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"run_id": runID,
		}).Error("Failed to write initial SSE event")
		return
	}
	flusher.Flush()

	// Also subscribe to workflow engine events if available
	var workflowEvents <-chan WorkflowEvent
	if s.workflowEngine != nil {
		events, err := s.workflowEngine.SubscribeToEvents(r.Context(), runID)
		if err == nil {
			workflowEvents = events
		}
	}

	// Create keepalive ticker for connection health
	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	// Forward events to SSE client
	for {
		select {
		case event := <-eventChan:
			// Global event filtered by run ID
			data, _ := json.Marshal(event)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data)); err != nil {
				// Write failed, client disconnected
				return
			}
			flusher.Flush()

		case event, ok := <-workflowEvents:
			if !ok {
				workflowEvents = nil // Channel closed
				continue
			}
			// Workflow engine event
			data, _ := json.Marshal(event)
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data)); err != nil {
				// Write failed, client disconnected
				return
			}
			flusher.Flush()

		case <-keepaliveTicker.C:
			// Send keepalive comment to maintain connection
			if _, err := fmt.Fprintf(w, ":keepalive\n\n"); err != nil {
				// Keepalive failed, client disconnected
				return
			}
			flusher.Flush()

		case <-r.Context().Done():
			return // Client disconnected
		}
	}
}
