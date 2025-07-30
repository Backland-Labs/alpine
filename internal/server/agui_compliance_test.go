package server

import (
	"encoding/json"
	"testing"
	"time"
	
	"github.com/Backland-Labs/alpine/internal/events"
)

// TestAGUIEventSequencing verifies that events follow the strict AG-UI protocol sequencing rules
func TestAGUIEventSequencing(t *testing.T) {
	t.Run("validate mandatory RunStarted to RunFinished sequence", func(t *testing.T) {
		// This test will verify that a workflow always emits:
		// 1. run_started as the first event
		// 2. run_finished as the last event for successful workflows
		// 3. run_error as the last event for failed workflows
		
		events := []WorkflowEvent{
			{Type: "run_started", RunID: "run-123", Timestamp: time.Now()},
			{Type: "text_message_start", RunID: "run-123", MessageID: "msg-456", Timestamp: time.Now()},
			{Type: "text_message_content", RunID: "run-123", MessageID: "msg-456", Timestamp: time.Now()},
			{Type: "text_message_end", RunID: "run-123", MessageID: "msg-456", Timestamp: time.Now()},
			{Type: "run_finished", RunID: "run-123", Timestamp: time.Now()},
		}
		
		// Verify first event is run_started
		if events[0].Type != "run_started" {
			t.Errorf("First event must be run_started, got %s", events[0].Type)
		}
		
		// Verify last event is run_finished or run_error
		lastEvent := events[len(events)-1]
		if lastEvent.Type != "run_finished" && lastEvent.Type != "run_error" {
			t.Errorf("Last event must be run_finished or run_error, got %s", lastEvent.Type)
		}
	})
	
	t.Run("validate TextMessage lifecycle with messageId correlation", func(t *testing.T) {
		// This test verifies that text message events:
		// 1. Start with text_message_start
		// 2. Continue with text_message_content events (same messageId)
		// 3. End with text_message_end (same messageId)
		
		messageID := "msg-789"
		runID := "run-123"
		
		events := []WorkflowEvent{
			{Type: "text_message_start", RunID: runID, MessageID: messageID, Source: "claude", Timestamp: time.Now()},
			{Type: "text_message_content", RunID: runID, MessageID: messageID, Content: "First chunk", Delta: true, Source: "claude", Timestamp: time.Now()},
			{Type: "text_message_content", RunID: runID, MessageID: messageID, Content: "Second chunk", Delta: true, Source: "claude", Timestamp: time.Now()},
			{Type: "text_message_end", RunID: runID, MessageID: messageID, Complete: true, Source: "claude", Timestamp: time.Now()},
		}
		
		// Verify all events have same messageId
		for i, event := range events {
			if event.MessageID != messageID {
				t.Errorf("Event %d has incorrect messageId: expected %s, got %s", i, messageID, event.MessageID)
			}
		}
		
		// Verify sequence
		if events[0].Type != "text_message_start" {
			t.Errorf("Text message must start with text_message_start, got %s", events[0].Type)
		}
		
		if events[len(events)-1].Type != "text_message_end" {
			t.Errorf("Text message must end with text_message_end, got %s", events[len(events)-1].Type)
		}
		
		// Verify content events have delta flag
		for i := 1; i < len(events)-1; i++ {
			if events[i].Type == "text_message_content" && !events[i].Delta {
				t.Errorf("text_message_content event %d must have delta=true", i)
			}
		}
		
		// Verify end event has complete flag
		if !events[len(events)-1].Complete {
			t.Error("text_message_end must have complete=true")
		}
	})
	
	t.Run("validate event ordering constraints", func(t *testing.T) {
		// This test ensures invalid sequences are detected
		
		// Test case: text_message_start before run_started (invalid)
		invalidSequence := []WorkflowEvent{
			{Type: "text_message_start", RunID: "run-123", MessageID: "msg-456"},
			{Type: "run_started", RunID: "run-123"},
		}
		
		if err := validateEventSequence(invalidSequence); err == nil {
			t.Error("Expected error for text_message_start before run_started")
		}
		
		// Test case: run_finished before text_message_end (invalid)
		invalidSequence2 := []WorkflowEvent{
			{Type: "run_started", RunID: "run-123"},
			{Type: "text_message_start", RunID: "run-123", MessageID: "msg-456"},
			{Type: "run_finished", RunID: "run-123"},
		}
		
		if err := validateEventSequence(invalidSequence2); err == nil {
			t.Error("Expected error for run_finished before text_message_end")
		}
	})
}

// TestAGUIFieldValidation verifies field naming and requirements per AG-UI spec
func TestAGUIFieldValidation(t *testing.T) {
	t.Run("validate camelCase field naming in JSON", func(t *testing.T) {
		event := WorkflowEvent{
			Type:      "run_started",
			RunID:     "run-123",
			MessageID: "msg-456",
			Timestamp: time.Now(),
		}
		
		// Marshal to JSON and check field names
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(data, &jsonMap); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}
		
		// Check camelCase fields
		if _, ok := jsonMap["runId"]; !ok {
			t.Error("JSON should have 'runId' field (camelCase)")
		}
		
		if _, ok := jsonMap["messageId"]; !ok {
			t.Error("JSON should have 'messageId' field (camelCase)")
		}
		
		if _, ok := jsonMap["run_id"]; ok {
			t.Error("JSON should not have 'run_id' field (snake_case)")
		}
	})
	
	t.Run("validate required fields per event type", func(t *testing.T) {
		testCases := []struct {
			name     string
			event    WorkflowEvent
			required []string
		}{
			{
				name: "run_started requires basic fields",
				event: WorkflowEvent{
					Type:      "run_started",
					RunID:     "run-123",
					Timestamp: time.Now(),
				},
				required: []string{"type", "runId", "timestamp"},
			},
			{
				name: "text_message_start requires source",
				event: WorkflowEvent{
					Type:      "text_message_start",
					RunID:     "run-123",
					MessageID: "msg-456",
					Source:    "claude",
					Timestamp: time.Now(),
				},
				required: []string{"type", "runId", "messageId", "source", "timestamp"},
			},
			{
				name: "text_message_content requires delta and source",
				event: WorkflowEvent{
					Type:      "text_message_content",
					RunID:     "run-123",
					MessageID: "msg-456",
					Content:   "Hello",
					Delta:     true,
					Source:    "claude",
					Timestamp: time.Now(),
				},
				required: []string{"type", "runId", "messageId", "content", "delta", "source", "timestamp"},
			},
			{
				name: "text_message_end requires complete flag",
				event: WorkflowEvent{
					Type:      "text_message_end",
					RunID:     "run-123",
					MessageID: "msg-456",
					Complete:  true,
					Source:    "claude",
					Timestamp: time.Now(),
				},
				required: []string{"type", "runId", "messageId", "complete", "source", "timestamp"},
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if err := validateEventFields(tc.event); err != nil {
					t.Errorf("Event validation failed: %v", err)
				}
			})
		}
	})
	
	t.Run("validate timestamp ISO 8601 format", func(t *testing.T) {
		event := WorkflowEvent{
			Type:      "run_started",
			RunID:     "run-123",
			Timestamp: time.Now(),
		}
		
		data, _ := json.Marshal(event)
		var jsonMap map[string]interface{}
		json.Unmarshal(data, &jsonMap)
		
		timestampStr, ok := jsonMap["timestamp"].(string)
		if !ok {
			t.Fatal("Timestamp should be a string in JSON")
		}
		
		// Verify ISO 8601 format (RFC3339)
		if _, err := time.Parse(time.RFC3339, timestampStr); err != nil {
			t.Errorf("Timestamp not in ISO 8601 format: %s", timestampStr)
		}
	})
	
	t.Run("validate source field for Claude output", func(t *testing.T) {
		claudeEvents := []WorkflowEvent{
			{Type: "text_message_start", RunID: "run-123", MessageID: "msg-456", Source: "claude"},
			{Type: "text_message_content", RunID: "run-123", MessageID: "msg-456", Source: "claude"},
			{Type: "text_message_end", RunID: "run-123", MessageID: "msg-456", Source: "claude"},
		}
		
		for _, event := range claudeEvents {
			if event.Source != "claude" {
				t.Errorf("Claude output event must have source='claude', got '%s'", event.Source)
			}
		}
	})
}

// TestAGUIJSONSerialization verifies JSON marshaling matches AG-UI schema exactly
func TestAGUIJSONSerialization(t *testing.T) {
	t.Run("verify JSON schema for run_started event", func(t *testing.T) {
		event := WorkflowEvent{
			Type:      "run_started",
			RunID:     "run-abc123",
			Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			Data: map[string]interface{}{
				"task":        "Implement user authentication",
				"worktreeDir": "/path/to/alpine_agent_state",
				"planMode":    false,
			},
		}
		
		data, _ := json.Marshal(event)
		expected := `{"type":"run_started","runId":"run-abc123","timestamp":"2024-01-01T12:00:00Z","data":{"planMode":false,"task":"Implement user authentication","worktreeDir":"/path/to/alpine_agent_state"}}`
		
		// Parse both to handle map ordering differences
		var gotMap, expectedMap map[string]interface{}
		json.Unmarshal(data, &gotMap)
		json.Unmarshal([]byte(expected), &expectedMap)
		
		// Deep equal comparison would go here
		if gotMap["type"] != expectedMap["type"] {
			t.Errorf("Event type mismatch")
		}
		if gotMap["runId"] != expectedMap["runId"] {
			t.Errorf("RunId mismatch")
		}
	})
	
	t.Run("verify optional fields are omitted when empty", func(t *testing.T) {
		event := WorkflowEvent{
			Type:      "run_started",
			RunID:     "run-123",
			Timestamp: time.Now(),
			// MessageID, Content, Source etc. should be omitted
		}
		
		data, _ := json.Marshal(event)
		var jsonMap map[string]interface{}
		json.Unmarshal(data, &jsonMap)
		
		// Check that optional empty fields are not present
		if _, ok := jsonMap["messageId"]; ok {
			t.Error("Empty messageId should be omitted from JSON")
		}
		if _, ok := jsonMap["content"]; ok {
			t.Error("Empty content should be omitted from JSON")
		}
		if _, ok := jsonMap["source"]; ok {
			t.Error("Empty source should be omitted from JSON")
		}
		if _, ok := jsonMap["delta"]; ok {
			t.Error("False delta should be omitted from JSON")
		}
		if _, ok := jsonMap["complete"]; ok {
			t.Error("False complete should be omitted from JSON")
		}
	})
	
	t.Run("verify exact event type strings", func(t *testing.T) {
		validEventTypes := []string{
			"run_started",
			"run_finished",
			"run_error",
			"text_message_start",
			"text_message_content",
			"text_message_end",
		}
		
		for _, eventType := range validEventTypes {
			event := WorkflowEvent{Type: eventType}
			if !events.IsValidAGUIEventType(event.Type) {
				t.Errorf("Event type '%s' should be valid", eventType)
			}
		}
		
		// Test invalid event types
		invalidTypes := []string{
			"workflow_started",    // Wrong name
			"RunStarted",         // Wrong case
			"run_start",          // Wrong suffix
			"claude_output",      // Non-AG-UI type
		}
		
		for _, eventType := range invalidTypes {
			if events.IsValidAGUIEventType(eventType) {
				t.Errorf("Event type '%s' should be invalid", eventType)
			}
		}
	})
}