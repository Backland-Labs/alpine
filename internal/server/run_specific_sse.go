package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// runSpecificEventHub manages run-specific event subscriptions
// It allows filtering global events to specific run IDs
type runSpecificEventHub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan WorkflowEvent // runID -> list of subscriber channels
}

// newRunSpecificEventHub creates a new event hub for run-specific subscriptions
func newRunSpecificEventHub() *runSpecificEventHub {
	return &runSpecificEventHub{
		subscribers: make(map[string][]chan WorkflowEvent),
	}
}

// subscribe adds a new subscriber for a specific run ID
func (hub *runSpecificEventHub) subscribe(runID string) chan WorkflowEvent {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	
	ch := make(chan WorkflowEvent, defaultEventBufferSize)
	hub.subscribers[runID] = append(hub.subscribers[runID], ch)
	return ch
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
	
	// Send initial connection event
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"runId\":\"%s\"}\n\n", runID)
	flusher.Flush()
	
	// Subscribe to run-specific events from hub
	eventChan := hub.subscribe(runID)
	defer hub.unsubscribe(runID, eventChan)
	
	// Also subscribe to workflow engine events if available
	var workflowEvents <-chan WorkflowEvent
	if s.workflowEngine != nil {
		events, err := s.workflowEngine.SubscribeToEvents(r.Context(), runID)
		if err == nil {
			workflowEvents = events
		}
	}
	
	// Forward events to SSE client
	for {
		select {
		case event := <-eventChan:
			// Global event filtered by run ID
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
			flusher.Flush()
			
		case event, ok := <-workflowEvents:
			if !ok {
				workflowEvents = nil // Channel closed
				continue
			}
			// Workflow engine event
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
			flusher.Flush()
			
		case <-r.Context().Done():
			return // Client disconnected
		}
	}
}