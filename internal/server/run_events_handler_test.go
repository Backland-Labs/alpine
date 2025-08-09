package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/events"
)

// TestRunEventsHandlerAcceptsToolCallEvents tests that the /runs/{id}/events endpoint
// accepts POST requests for tool call events
func TestRunEventsHandlerAcceptsToolCallEvents(t *testing.T) {
	server := NewServer(0)

	// Create a test run
	runID := "test-run-123"
	server.runs[runID] = &Run{
		ID:      runID,
		Status:  "running",
		Created: time.Now(),
	}

	// Mock batching emitter to capture forwarded events
	var capturedEvents []events.BaseEvent
	mockEmitter := &MockBatchingEmitter{
		EmittedEvents: &capturedEvents,
	}
	server.SetBatchingEmitter(mockEmitter)

	// Create valid tool call event
	payload := map[string]interface{}{
		"type":       "tool_call_started",
		"runId":      runID,
		"timestamp":  time.Now().Format(time.RFC3339),
		"toolCallId": "tool-456",
		"toolName":   "bash",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/runs/"+runID+"/events", bytes.NewReader(body))
	req.SetPathValue("id", runID) // Set the path parameter
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()

	// Create a mock hub for the handler
	hub := newRunSpecificEventHub()
	server.runEventHub = hub

	// Call the enhanced handler directly
	server.enhancedRunEventsHandler(w, req, hub)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should have forwarded event to emitter
	if len(capturedEvents) != 1 {
		t.Errorf("Expected 1 event to be forwarded, got %d", len(capturedEvents))
	}

	// Verify the event was processed correctly
	if len(capturedEvents) > 0 {
		event := capturedEvents[0]
		if event.GetRunID() != runID {
			t.Errorf("Expected run ID %s, got %s", runID, event.GetRunID())
		}
	}
}

// TestRunEventsHandlerValidatesRunID tests that the endpoint validates
// that the event's run ID matches the URL parameter
func TestRunEventsHandlerValidatesRunID(t *testing.T) {
	server := NewServer(0)

	// Create a test run
	runID := "test-run-123"
	server.runs[runID] = &Run{
		ID:      runID,
		Status:  "running",
		Created: time.Now(),
	}

	// Create event with mismatched run ID
	payload := map[string]interface{}{
		"type":       "tool_call_started",
		"runId":      "different-run-456", // Mismatched run ID
		"timestamp":  time.Now().Format(time.RFC3339),
		"toolCallId": "tool-789",
		"toolName":   "bash",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/runs/"+runID+"/events", bytes.NewReader(body))
	req.SetPathValue("id", runID) // Set the path parameter
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()

	// Create a mock hub for the handler
	hub := newRunSpecificEventHub()
	server.runEventHub = hub

	// Call the enhanced handler directly
	server.enhancedRunEventsHandler(w, req, hub)

	// Should return 400 Bad Request for run ID mismatch
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for run ID mismatch, got %d", w.Code)
	}
}

// TestRunEventsHandlerRequiresAuthentication tests that the endpoint requires authentication
func TestRunEventsHandlerRequiresAuthentication(t *testing.T) {
	server := NewServer(0)

	// Create a test run
	runID := "test-run-123"
	server.runs[runID] = &Run{
		ID:      runID,
		Status:  "running",
		Created: time.Now(),
	}

	// Create valid event but without authentication
	payload := map[string]interface{}{
		"type":       "tool_call_started",
		"runId":      runID,
		"timestamp":  time.Now().Format(time.RFC3339),
		"toolCallId": "tool-999",
		"toolName":   "bash",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/runs/"+runID+"/events", bytes.NewReader(body))
	req.SetPathValue("id", runID) // Set the path parameter
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	w := httptest.NewRecorder()

	// Create a mock hub for the handler
	hub := newRunSpecificEventHub()
	server.runEventHub = hub

	// Call the enhanced handler directly
	server.enhancedRunEventsHandler(w, req, hub)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing authentication, got %d", w.Code)
	}
}
