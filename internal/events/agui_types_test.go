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

	// Test JSON format follows AG-UI protocol
	var eventMap map[string]interface{}
	err = json.Unmarshal(jsonData, &eventMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal ToolCallStartEvent to map: %v", err)
	}

	// Verify AG-UI protocol compliance
	if eventMap["type"] != ToolCallStart {
		t.Errorf("Expected AG-UI type '%s', got '%s'", ToolCallStart, eventMap["type"])
	}
	if eventMap["toolCallName"] != "bash" {
		t.Errorf("Expected toolCallName 'bash', got '%s'", eventMap["toolCallName"])
	}
	if eventMap["runId"] != "test-run-123" {
		t.Errorf("Expected runId 'test-run-123', got '%s'", eventMap["runId"])
	}
	if eventMap["toolCallId"] != "tool-call-456" {
		t.Errorf("Expected toolCallId 'tool-call-456', got '%s'", eventMap["toolCallId"])
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
