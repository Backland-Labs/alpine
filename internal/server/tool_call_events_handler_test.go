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

// TestToolCallEventsEndpointAuthentication tests that the endpoint requires authentication
// This is critical security functionality
func TestToolCallEventsEndpointAuthentication(t *testing.T) {
	server := NewServer(0)

	// Create request without authentication
	payload := map[string]interface{}{
		"type":       "tool_call_started",
		"runId":      "test-run",
		"timestamp":  time.Now().Format(time.RFC3339),
		"toolCallId": "tool-123",
		"toolName":   "bash",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/events/tool-calls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.toolCallEventsHandler(w, req)

	// Should return 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestToolCallEventsEndpointValidation tests that the endpoint validates incoming data
// This prevents invalid data from entering the system
func TestToolCallEventsEndpointValidation(t *testing.T) {
	server := NewServer(0)

	// Create request with invalid payload (missing required fields)
	payload := map[string]interface{}{
		"type": "invalid_type",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/events/tool-calls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token") // Mock auth

	w := httptest.NewRecorder()
	server.toolCallEventsHandler(w, req)

	// Should return 400 Bad Request for validation failure
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestToolCallEventsEndpointForwardsToEmitter tests that valid events are forwarded to the batching system
// This is the core happy path functionality
func TestToolCallEventsEndpointForwardsToEmitter(t *testing.T) {
	server := NewServer(0)

	// Mock batching emitter to capture forwarded events
	var capturedEvents []events.BaseEvent
	mockEmitter := &MockBatchingEmitter{
		EmittedEvents: &capturedEvents,
	}
	server.SetBatchingEmitter(mockEmitter)

	// Create valid tool call event
	payload := map[string]interface{}{
		"type":       "tool_call_started",
		"runId":      "test-run",
		"timestamp":  time.Now().Format(time.RFC3339),
		"toolCallId": "tool-123",
		"toolName":   "bash",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/events/tool-calls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token") // Mock auth

	w := httptest.NewRecorder()
	server.toolCallEventsHandler(w, req)

	// Should return 200 OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Should have forwarded event to emitter
	if len(capturedEvents) != 1 {
		t.Errorf("Expected 1 forwarded event, got %d", len(capturedEvents))
	}

	if len(capturedEvents) > 0 {
		if capturedEvents[0].GetType() != "tool_call_started" {
			t.Errorf("Expected event type tool_call_started, got %s", capturedEvents[0].GetType())
		}
	}
}

// MockBatchingEmitter for testing
type MockBatchingEmitter struct {
	EmittedEvents *[]events.BaseEvent
}

func (m *MockBatchingEmitter) EmitToolCallEvent(event events.BaseEvent) {
	*m.EmittedEvents = append(*m.EmittedEvents, event)
}

func (m *MockBatchingEmitter) RunStarted(runID string, task string)             {}
func (m *MockBatchingEmitter) RunFinished(runID string, task string)            {}
func (m *MockBatchingEmitter) RunError(runID string, task string, err error)    {}
func (m *MockBatchingEmitter) StateSnapshot(runID string, snapshot interface{}) {}
