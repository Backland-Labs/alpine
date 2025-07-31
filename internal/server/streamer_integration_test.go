package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestServerStreamerImplementation verifies that the server correctly implements
// the Streamer interface using the existing BroadcastEvent infrastructure.
// This ensures proper integration with the SSE event system.
func TestServerStreamerImplementation(t *testing.T) {
	t.Run("Server broadcasts streaming events via SSE", func(t *testing.T) {
		// Given a server
		server := NewServer(0)
		
		// Create a buffer to capture broadcast events
		capturedEvents := make([]string, 0)
		
		// Override the eventsChan to capture events
		originalChan := server.eventsChan
		testChan := make(chan string, 100)
		server.eventsChan = testChan
		defer func() {
			server.eventsChan = originalChan
		}()
		
		// Capture events in background
		done := make(chan bool)
		go func() {
			for {
				select {
				case event := <-testChan:
					capturedEvents = append(capturedEvents, event)
				case <-done:
					return
				}
			}
		}()
		
		// When we use the server as a Streamer
		streamer := NewServerStreamer(server)
		runID := "run-test-123"
		messageID := "msg-test-456"
		
		// Start streaming
		err := streamer.StreamStart(runID, messageID)
		if err != nil {
			t.Fatalf("StreamStart failed: %v", err)
		}
		
		// Stream content
		err = streamer.StreamContent(runID, messageID, "Hello from Claude!")
		if err != nil {
			t.Fatalf("StreamContent failed: %v", err)
		}
		
		err = streamer.StreamContent(runID, messageID, "Second chunk of output")
		if err != nil {
			t.Fatalf("StreamContent failed: %v", err)
		}
		
		// End streaming
		err = streamer.StreamEnd(runID, messageID)
		if err != nil {
			t.Fatalf("StreamEnd failed: %v", err)
		}
		
		// Allow time for events to propagate
		time.Sleep(50 * time.Millisecond)
		close(done)
		
		// Then we should have received the correct events
		if len(capturedEvents) != 4 {
			t.Fatalf("Expected 4 events, got %d", len(capturedEvents))
		}
		
		// Parse and verify each event
		for i, eventStr := range capturedEvents {
			// Extract JSON data from SSE format
			lines := strings.Split(eventStr, "\n")
			var eventType string
			var jsonData string
			
			for _, line := range lines {
				if strings.HasPrefix(line, "event: ") {
					eventType = strings.TrimPrefix(line, "event: ")
				} else if strings.HasPrefix(line, "data: ") {
					jsonData = strings.TrimPrefix(line, "data: ")
				}
			}
			
			// Parse JSON
			var event WorkflowEvent
			if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
				t.Fatalf("Failed to parse event %d: %v", i, err)
			}
			
			// Verify event type matches
			if event.Type != eventType {
				t.Errorf("Event %d: SSE event type %q doesn't match JSON type %q", i, eventType, event.Type)
			}
			
			// Verify common fields
			if event.RunID != runID {
				t.Errorf("Event %d: expected runID %q, got %q", i, runID, event.RunID)
			}
			
			if event.MessageID != messageID {
				t.Errorf("Event %d: expected messageID %q, got %q", i, messageID, event.MessageID)
			}
			
			if event.Source != "claude" {
				t.Errorf("Event %d: expected source 'claude', got %q", i, event.Source)
			}
		}
		
		// Verify specific event content
		var contentEvents []WorkflowEvent
		for _, eventStr := range capturedEvents {
			lines := strings.Split(eventStr, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					jsonData := strings.TrimPrefix(line, "data: ")
					var event WorkflowEvent
					if json.Unmarshal([]byte(jsonData), &event) == nil && event.Type == "text_message_content" {
						contentEvents = append(contentEvents, event)
					}
				}
			}
		}
		
		if len(contentEvents) != 2 {
			t.Fatalf("Expected 2 content events, got %d", len(contentEvents))
		}
		
		if contentEvents[0].Content != "Hello from Claude!" {
			t.Errorf("First content: expected %q, got %q", "Hello from Claude!", contentEvents[0].Content)
		}
		
		if contentEvents[1].Content != "Second chunk of output" {
			t.Errorf("Second content: expected %q, got %q", "Second chunk of output", contentEvents[1].Content)
		}
		
		// Verify AG-UI specific fields
		if !contentEvents[0].Delta || !contentEvents[1].Delta {
			t.Error("Content events should have Delta=true")
		}
	})
	
	t.Run("WorkflowEvent format matches streaming requirements", func(t *testing.T) {
		// Given a WorkflowEvent for streaming
		event := WorkflowEvent{
			Type:      "text_message_content",
			RunID:     "run-123",
			MessageID: "msg-456",
			Timestamp: time.Now(),
			Content:   "Test content",
			Delta:     true,
			Source:    "claude",
		}
		
		// When we marshal it to JSON
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatalf("Failed to marshal event: %v", err)
		}
		
		// Then it should use camelCase field names
		jsonStr := string(data)
		if !strings.Contains(jsonStr, `"runId"`) {
			t.Error("Expected 'runId' field in JSON")
		}
		if !strings.Contains(jsonStr, `"messageId"`) {
			t.Error("Expected 'messageId' field in JSON")
		}
		if strings.Contains(jsonStr, `"run_id"`) {
			t.Error("Should not have 'run_id' field (should be camelCase)")
		}
	})
	
	t.Run("Concurrent client streaming", func(t *testing.T) {
		// Given a server with multiple connected clients
		server := NewServer(0)
		streamer := &ServerStreamer{server: server}
		
		// Simulate multiple concurrent streaming sessions
		var wg sync.WaitGroup
		errors := make([]error, 0)
		errorsMu := sync.Mutex{}
		
		// Run 10 concurrent streaming sessions
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				
				runID := fmt.Sprintf("run-%d", id)
				messageID := fmt.Sprintf("msg-%d", id)
				
				// Perform streaming
				if err := streamer.StreamStart(runID, messageID); err != nil {
					errorsMu.Lock()
					errors = append(errors, err)
					errorsMu.Unlock()
					return
				}
				
				for j := 0; j < 5; j++ {
					content := fmt.Sprintf("Content %d-%d", id, j)
					if err := streamer.StreamContent(runID, messageID, content); err != nil {
						errorsMu.Lock()
						errors = append(errors, err)
						errorsMu.Unlock()
						return
					}
				}
				
				if err := streamer.StreamEnd(runID, messageID); err != nil {
					errorsMu.Lock()
					errors = append(errors, err)
					errorsMu.Unlock()
				}
			}(i)
		}
		
		// Wait for all streaming to complete
		wg.Wait()
		
		// Then no errors should have occurred
		if len(errors) > 0 {
			t.Errorf("Concurrent streaming produced %d errors: %v", len(errors), errors)
		}
	})
}

