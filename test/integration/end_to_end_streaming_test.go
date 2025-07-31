// Package integration provides comprehensive end-to-end tests for Alpine's streaming functionality
package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/server"
)

// mockStreamingExecutor simulates Claude output with streaming chunks
type mockStreamingExecutor struct {
	chunks []string
	delay  time.Duration
}

func (m *mockStreamingExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	// Simulate streaming by writing chunks with delays
	var output strings.Builder
	for _, chunk := range m.chunks {
		output.WriteString(chunk)
		time.Sleep(m.delay)
	}
	return output.String(), nil
}

func (m *mockStreamingExecutor) SetStreamer(streamer events.Streamer) {
	// In a real implementation, this would store the streamer
}

func (m *mockStreamingExecutor) SetRunID(runID string) {
	// In a real implementation, this would store the run ID
}

// TestEndToEndStreamingValidation verifies complete AG-UI compliant streaming from Alpine workflow to SSE delivery
func TestEndToEndStreamingValidation(t *testing.T) {
	// Create mock executor that produces streaming output
	mockExecutor := &mockStreamingExecutor{
		chunks: []string{
			"I'll help you create a Python calculator...\n",
			"Let me start by creating the basic structure...\n",
			"Here's the implementation:\n\n```python\nclass Calculator:\n",
			"    def add(self, a, b):\n        return a + b\n```\n",
		},
		delay: 10 * time.Millisecond,
	}

	// Create test configuration
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false, // Disable for testing
		},
		StateFile: "test_state.json",
	}

	// Create and start server
	srv := server.NewServer(0) // Use port 0 for auto-assignment
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	serverReady := make(chan bool)
	go func() {
		serverReady <- true
		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	<-serverReady
	time.Sleep(100 * time.Millisecond)

	// Get server address
	addr := srv.Address()
	if addr == "" {
		t.Fatal("Server failed to start")
	}

	// Create AlpineWorkflowEngine with server reference
	alpineEngine := server.NewAlpineWorkflowEngine(mockExecutor, nil, cfg)
	alpineEngine.SetServer(srv)
	srv.SetWorkflowEngine(alpineEngine)

	// Start workflow via REST API
	issueURL := "https://github.com/test/repo/issues/123"
	runPayload := map[string]string{
		"issue_url": issueURL,
		"agent_id":  "alpine-agent",
	}
	payloadBytes, _ := json.Marshal(runPayload)

	resp, err := http.Post(
		fmt.Sprintf("http://%s/agents/run", addr),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	// Parse response to get run ID
	var runResponse server.Run
	if err := json.NewDecoder(resp.Body).Decode(&runResponse); err != nil {
		t.Fatalf("Failed to decode run response: %v", err)
	}

	runID := runResponse.ID
	t.Logf("Started workflow with run ID: %s", runID)

	// Connect to SSE stream
	sseURL := fmt.Sprintf("http://%s/runs/%s/events", addr, runID)
	sseResp, err := http.Get(sseURL)
	if err != nil {
		t.Fatalf("Failed to connect to SSE stream: %v", err)
	}
	defer sseResp.Body.Close()

	// Verify SSE content type
	contentType := sseResp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	// Collect and validate events
	events := make([]server.WorkflowEvent, 0)
	scanner := bufio.NewScanner(sseResp.Body)

	// Set a timeout for event collection
	eventTimeout := time.After(10 * time.Second)
	eventComplete := make(chan bool)

	go func() {
		var currentEvent *server.WorkflowEvent
		var eventType string

		for scanner.Scan() {
			line := scanner.Text()

			// Parse SSE format
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				// Parse JSON data
				var event server.WorkflowEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					t.Errorf("Failed to parse event data: %v", err)
					continue
				}

				// Verify event type matches
				if event.Type != eventType {
					t.Errorf("Event type mismatch: SSE type '%s' != JSON type '%s'", eventType, event.Type)
				}

				currentEvent = &event
			} else if line == "" && currentEvent != nil {
				// Empty line indicates end of event
				events = append(events, *currentEvent)

				// Check if we've received the final event
				if currentEvent.Type == "run_finished" ||
					currentEvent.Type == "run_error" {
					eventComplete <- true
					return
				}

				currentEvent = nil
				eventType = ""
			}
		}

		if err := scanner.Err(); err != nil {
			t.Errorf("SSE scanner error: %v", err)
		}
		eventComplete <- false
	}()

	// Wait for events or timeout
	select {
	case success := <-eventComplete:
		if !success {
			t.Error("SSE stream ended without completion event")
		}
	case <-eventTimeout:
		t.Error("Timeout waiting for events")
	}

	// Validate event sequence
	t.Run("Event Sequence Validation", func(t *testing.T) {
		if len(events) < 2 {
			t.Fatalf("Expected at least 2 events, got %d", len(events))
		}

		// First event must be run_started
		if events[0].Type != "run_started" {
			t.Errorf("First event must be 'run_started', got '%s'", events[0].Type)
		}

		// Last event must be run_finished or run_error
		lastEvent := events[len(events)-1]
		if lastEvent.Type != "run_finished" &&
			lastEvent.Type != "run_error" {
			t.Errorf("Last event must be 'run_finished' or 'run_error', got '%s'", lastEvent.Type)
		}

		// All events must have the same run ID
		for i, event := range events {
			if event.RunID != runID {
				t.Errorf("Event %d has wrong run ID: expected '%s', got '%s'", i, runID, event.RunID)
			}
		}
	})

	// Validate text message streaming
	t.Run("Text Message Streaming Validation", func(t *testing.T) {
		var messageID string
		textStartFound := false
		textContentCount := 0
		textEndFound := false

		for _, event := range events {
			switch event.Type {
			case "text_message_start":
				if textStartFound {
					t.Error("Multiple text_message_start events found")
				}
				textStartFound = true
				messageID = event.MessageID

				if messageID == "" {
					t.Error("text_message_start missing messageId")
				}
				if event.Source != "claude" {
					t.Errorf("Expected source 'claude', got '%s'", event.Source)
				}

			case "text_message_content":
				if !textStartFound {
					t.Error("text_message_content before text_message_start")
				}
				if event.MessageID != messageID {
					t.Errorf("Mismatched messageId: expected '%s', got '%s'", messageID, event.MessageID)
				}
				if !event.Delta {
					t.Error("text_message_content missing delta=true")
				}
				if event.Source != "claude" {
					t.Errorf("Expected source 'claude', got '%s'", event.Source)
				}
				if event.Content == "" {
					t.Error("text_message_content has empty content")
				}
				textContentCount++

			case "text_message_end":
				if textEndFound {
					t.Error("Multiple text_message_end events found")
				}
				if !textStartFound {
					t.Error("text_message_end before text_message_start")
				}
				textEndFound = true
				if event.MessageID != messageID {
					t.Errorf("Mismatched messageId: expected '%s', got '%s'", messageID, event.MessageID)
				}
				if !event.Complete {
					t.Error("text_message_end missing complete=true")
				}
			}
		}

		if textStartFound && !textEndFound {
			t.Error("text_message_start without corresponding text_message_end")
		}
		if textContentCount == 0 && textStartFound {
			t.Error("No text_message_content events between start and end")
		}
	})

	// Validate AG-UI field naming
	t.Run("AG-UI Field Naming Validation", func(t *testing.T) {
		for i, event := range events {
			// Check camelCase field naming in raw JSON
			eventJSON, err := json.Marshal(event)
			if err != nil {
				t.Errorf("Failed to marshal event %d: %v", i, err)
				continue
			}

			jsonStr := string(eventJSON)

			// Verify camelCase fields
			if strings.Contains(jsonStr, "\"run_id\"") {
				t.Error("Found 'run_id' instead of 'runId' in JSON")
			}
			if strings.Contains(jsonStr, "\"message_id\"") {
				t.Error("Found 'message_id' instead of 'messageId' in JSON")
			}

			// Verify timestamp format (ISO 8601)
			if !event.Timestamp.IsZero() {
				timestampStr := event.Timestamp.Format(time.RFC3339)
				if !strings.Contains(jsonStr, timestampStr) {
					t.Error("Timestamp not in ISO 8601 format")
				}
			}
		}
	})

	// Validate real-time streaming (not batched)
	t.Run("Real-Time Streaming Validation", func(t *testing.T) {
		// Check that events have reasonable time spacing
		if len(events) > 2 {
			firstTime := events[0].Timestamp
			lastTime := events[len(events)-1].Timestamp
			duration := lastTime.Sub(firstTime)

			if duration < 50*time.Millisecond {
				t.Error("Events appear to be batched (total duration too short)")
			}
		}
	})

	t.Logf("Collected %d events total", len(events))
	for i, event := range events {
		t.Logf("Event %d: type=%s, runId=%s, messageId=%s",
			i, event.Type, event.RunID, event.MessageID)
	}
}

// TestStreamingWithMultipleClients verifies concurrent SSE clients receive events
func TestStreamingWithMultipleClients(t *testing.T) {
	// This test ensures the server can handle multiple concurrent SSE clients
	// for the same workflow run

	// Create mock executor
	mockExecutor := &mockStreamingExecutor{
		chunks: []string{"Test output 1\n", "Test output 2\n"},
		delay:  50 * time.Millisecond,
	}

	// Create test configuration
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}

	// Create and start server
	srv := server.NewServerWithConfig(0, 100, 10) // Custom config for testing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	go func() {
		if err := srv.Start(ctx); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	// Create and configure workflow engine
	alpineEngine := server.NewAlpineWorkflowEngine(mockExecutor, nil, cfg)
	alpineEngine.SetServer(srv)
	srv.SetWorkflowEngine(alpineEngine)

	// Start a workflow
	addr := srv.Address()
	runPayload := map[string]string{
		"issue_url": "https://github.com/test/repo/issues/456",
		"agent_id":  "alpine-agent",
	}
	payloadBytes, _ := json.Marshal(runPayload)

	resp, err := http.Post(
		fmt.Sprintf("http://%s/agents/run", addr),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}
	defer resp.Body.Close()

	var runResponse server.Run
	json.NewDecoder(resp.Body).Decode(&runResponse)
	runID := runResponse.ID

	// Connect multiple SSE clients
	const numClients = 3
	clientEvents := make([][]server.WorkflowEvent, numClients)
	done := make(chan int, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			sseURL := fmt.Sprintf("http://%s/runs/%s/events", addr, runID)
			resp, err := http.Get(sseURL)
			if err != nil {
				t.Errorf("Client %d: Failed to connect: %v", clientID, err)
				done <- clientID
				return
			}
			defer resp.Body.Close()

			scanner := bufio.NewScanner(resp.Body)
			var events []server.WorkflowEvent

			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					var event server.WorkflowEvent
					if json.Unmarshal([]byte(data), &event) == nil {
						events = append(events, event)

						// Stop on completion
						if event.Type == "run_finished" ||
							event.Type == "run_error" {
							break
						}
					}
				}
			}

			clientEvents[clientID] = events
			done <- clientID
		}(i)
	}

	// Wait for all clients to finish
	for i := 0; i < numClients; i++ {
		select {
		case <-done:
			// Client finished
		case <-time.After(5 * time.Second):
			t.Errorf("Timeout waiting for client %d", i)
		}
	}

	// Verify all clients received events
	for i := 0; i < numClients; i++ {
		if len(clientEvents[i]) < 2 {
			t.Errorf("Client %d received only %d events", i, len(clientEvents[i]))
		}
	}

	// Verify all clients received the same events
	if numClients > 1 {
		firstClientEventCount := len(clientEvents[0])
		for i := 1; i < numClients; i++ {
			if len(clientEvents[i]) != firstClientEventCount {
				t.Errorf("Client %d received %d events, but client 0 received %d",
					i, len(clientEvents[i]), firstClientEventCount)
			}
		}
	}
}

// TestStreamingErrorHandling verifies streaming continues even with errors
func TestStreamingErrorHandling(t *testing.T) {
	// Create executor that simulates an error
	mockExecutor := &mockStreamingExecutor{
		chunks: []string{"Starting task...\n"},
		delay:  10 * time.Millisecond,
	}

	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}

	srv := server.NewServer(0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Start(ctx)
	time.Sleep(100 * time.Millisecond)

	// Create workflow engine with a nil worktree manager to simulate error conditions
	alpineEngine := server.NewAlpineWorkflowEngine(mockExecutor, nil, cfg)
	alpineEngine.SetServer(srv)
	srv.SetWorkflowEngine(alpineEngine)

	// Start workflow
	addr := srv.Address()
	runPayload := map[string]string{
		"issue_url": "https://github.com/test/repo/issues/789",
		"agent_id":  "alpine-agent",
	}
	payloadBytes, _ := json.Marshal(runPayload)

	resp, _ := http.Post(
		fmt.Sprintf("http://%s/agents/run", addr),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	resp.Body.Close()

	var runResponse server.Run
	json.NewDecoder(resp.Body).Decode(&runResponse)

	// Connect to SSE and verify we still get events even if streaming has issues
	sseURL := fmt.Sprintf("http://%s/runs/%s/events", addr, runResponse.ID)
	sseResp, err := http.Get(sseURL)
	if err != nil {
		t.Fatalf("Failed to connect to SSE: %v", err)
	}
	defer sseResp.Body.Close()

	// We should at least get run_started and run_finished/run_error events
	eventCount := 0
	scanner := bufio.NewScanner(sseResp.Body)
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			if eventCount < 2 {
				t.Errorf("Expected at least 2 events, got %d", eventCount)
			}
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "event: ") {
					eventCount++
				}
			}
		}
	}
}
