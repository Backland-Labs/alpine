package events

import (
	"encoding/json"
	"testing"
	"time"
)

// TestToolCallEventTypesAreValid tests that new tool call event types are recognized as valid AG-UI events
func TestToolCallEventTypesAreValid(t *testing.T) {
	// Test that tool call event types are valid
	if !IsValidAGUIEventType(AGUIEventToolCallStarted) {
		t.Errorf("%s should be a valid AG-UI event type", AGUIEventToolCallStarted)
	}
	if !IsValidAGUIEventType(AGUIEventToolCallFinished) {
		t.Errorf("%s should be a valid AG-UI event type", AGUIEventToolCallFinished)
	}
	if !IsValidAGUIEventType(AGUIEventToolCallError) {
		t.Errorf("%s should be a valid AG-UI event type", AGUIEventToolCallError)
	}
}

// TestBaseEventInterface tests that tool call events implement BaseEvent interface correctly
func TestBaseEventInterface(t *testing.T) {
	now := time.Now()

	// Test ToolCallStartEvent implements BaseEvent
	startEvent := &ToolCallStartEvent{
		Type:       AGUIEventToolCallStarted,
		RunID:      "test-run-123",
		Timestamp:  now,
		ToolCallID: "tool-call-456",
		ToolName:   "bash",
	}

	// Test BaseEvent interface methods
	if startEvent.GetType() != AGUIEventToolCallStarted {
		t.Errorf("Expected type '%s', got '%s'", AGUIEventToolCallStarted, startEvent.GetType())
	}
	if startEvent.GetRunID() != "test-run-123" {
		t.Errorf("Expected runID 'test-run-123', got '%s'", startEvent.GetRunID())
	}
	if !startEvent.GetTimestamp().Equal(now) {
		t.Errorf("Expected timestamp %v, got %v", now, startEvent.GetTimestamp())
	}
}

// TestToolCallEventSerialization tests that tool call events serialize to proper JSON
func TestToolCallEventSerialization(t *testing.T) {
	now := time.Now()

	startEvent := &ToolCallStartEvent{
		Type:       AGUIEventToolCallStarted,
		RunID:      "test-run-123",
		Timestamp:  now,
		ToolCallID: "tool-call-456",
		ToolName:   "bash",
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(startEvent)
	if err != nil {
		t.Fatalf("Failed to marshal ToolCallStartEvent: %v", err)
	}

	// Test JSON deserialization
	var deserializedEvent ToolCallStartEvent
	err = json.Unmarshal(jsonData, &deserializedEvent)
	if err != nil {
		t.Fatalf("Failed to unmarshal ToolCallStartEvent: %v", err)
	}

	// Verify critical fields
	if deserializedEvent.Type != AGUIEventToolCallStarted {
		t.Errorf("Expected type '%s', got '%s'", AGUIEventToolCallStarted, deserializedEvent.Type)
	}
	if deserializedEvent.ToolName != "bash" {
		t.Errorf("Expected tool name 'bash', got '%s'", deserializedEvent.ToolName)
	}
}

// TestAllToolCallEventTypesImplementBaseEvent tests that all tool call events implement BaseEvent
func TestAllToolCallEventTypesImplementBaseEvent(t *testing.T) {
	now := time.Now()

	events := []BaseEvent{
		&ToolCallStartEvent{
			Type:       AGUIEventToolCallStarted,
			RunID:      "test-run",
			Timestamp:  now,
			ToolCallID: "tool-call-1",
			ToolName:   "bash",
		},
		&ToolCallEndEvent{
			Type:       AGUIEventToolCallFinished,
			RunID:      "test-run",
			Timestamp:  now,
			ToolCallID: "tool-call-1",
			ToolName:   "bash",
			Duration:   "1.5s",
		},
		&ToolCallErrorEvent{
			Type:       AGUIEventToolCallError,
			RunID:      "test-run",
			Timestamp:  now,
			ToolCallID: "tool-call-1",
			ToolName:   "bash",
			Error:      "command failed",
		},
	}

	for _, event := range events {
		if event.GetRunID() != "test-run" {
			t.Errorf("Expected runID 'test-run', got '%s'", event.GetRunID())
		}
		if !event.GetTimestamp().Equal(now) {
			t.Errorf("Expected timestamp %v, got %v", now, event.GetTimestamp())
		}
		if err := event.Validate(); err != nil {
			t.Errorf("Event validation failed: %v", err)
		}
	}
}
