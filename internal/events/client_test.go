package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestClient_PostEvent(t *testing.T) {
	t.Run("posts event to configured endpoint", func(t *testing.T) {
		// Track received events
		var mu sync.Mutex
		var receivedEvents []map[string]interface{}

		// Create mock UI server
		mockUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST request, got %s", r.Method)
			}

			var event map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode event: %v", err)
			}

			mu.Lock()
			receivedEvents = append(receivedEvents, event)
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer mockUI.Close()

		// Create client
		client := NewClient(mockUI.URL, "test-run-123")

		// Post event
		err := client.PostEvent("RunStarted", map[string]interface{}{
			"task": "test task",
		})
		if err != nil {
			t.Fatalf("failed to post event: %v", err)
		}

		// Verify event was received
		time.Sleep(50 * time.Millisecond) // Small delay for async posting

		mu.Lock()
		defer mu.Unlock()

		if len(receivedEvents) != 1 {
			t.Fatalf("expected 1 event, got %d", len(receivedEvents))
		}

		event := receivedEvents[0]
		if event["type"] != "RunStarted" {
			t.Errorf("expected event type RunStarted, got %v", event["type"])
		}

		data, ok := event["data"].(map[string]interface{})
		if !ok {
			t.Fatal("event data is not a map")
		}

		if data["runId"] != "test-run-123" {
			t.Errorf("expected runId test-run-123, got %v", data["runId"])
		}

		if data["task"] != "test task" {
			t.Errorf("expected task 'test task', got %v", data["task"])
		}
	})

	t.Run("follows ag-ui protocol format", func(t *testing.T) {
		receivedEvent := make(chan map[string]interface{}, 1)

		mockUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode event: %v", err)
			}
			receivedEvent <- event
			w.WriteHeader(http.StatusOK)
		}))
		defer mockUI.Close()

		client := NewClient(mockUI.URL, "run-456")

		// Post a ToolCallStart event
		err := client.PostEvent("ToolCallStart", map[string]interface{}{
			"toolCallId":   "tool-789",
			"toolCallName": "Write",
		})
		if err != nil {
			t.Fatalf("failed to post event: %v", err)
		}

		// Verify format
		select {
		case event := <-receivedEvent:
			// Check required fields
			if _, ok := event["type"]; !ok {
				t.Error("event missing 'type' field")
			}
			if _, ok := event["data"]; !ok {
				t.Error("event missing 'data' field")
			}
			if _, ok := event["timestamp"]; !ok {
				t.Error("event missing 'timestamp' field")
			}

			// Verify data contains runId
			data := event["data"].(map[string]interface{})
			if data["runId"] != "run-456" {
				t.Error("event data missing or incorrect runId")
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("handles connection failures gracefully", func(t *testing.T) {
		// Use invalid endpoint
		client := NewClient("http://localhost:12345/nonexistent", "run-123")

		// Should not panic, should return error
		err := client.PostEvent("RunStarted", nil)
		if err == nil {
			t.Fatal("expected error for connection failure")
		}
	})

	t.Run("retries on temporary failures", func(t *testing.T) {
		attempts := 0
		mockUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				// Fail first 2 attempts
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Succeed on 3rd attempt
			w.WriteHeader(http.StatusOK)
		}))
		defer mockUI.Close()

		client := NewClient(mockUI.URL, "run-123")

		err := client.PostEvent("RunStarted", nil)
		if err != nil {
			t.Fatalf("failed after retries: %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("includes timestamp in events", func(t *testing.T) {
		receivedEvent := make(chan map[string]interface{}, 1)

		mockUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var event map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
				t.Errorf("failed to decode event: %v", err)
			}
			receivedEvent <- event
			w.WriteHeader(http.StatusOK)
		}))
		defer mockUI.Close()

		client := NewClient(mockUI.URL, "run-123")

		beforePost := time.Now()
		if err := client.PostEvent("TestEvent", nil); err != nil {
			t.Fatalf("failed to post event: %v", err)
		}
		afterPost := time.Now().Add(100 * time.Millisecond) // Add buffer for processing

		select {
		case event := <-receivedEvent:
			timestamp, ok := event["timestamp"].(string)
			if !ok {
				t.Fatal("timestamp not a string")
			}

			// Parse timestamp
			eventTime, err := time.Parse(time.RFC3339, timestamp)
			if err != nil {
				t.Fatalf("failed to parse timestamp: %v", err)
			}

			// Verify timestamp is reasonable (with some buffer for clock differences)
			if eventTime.Before(beforePost.Add(-1*time.Second)) || eventTime.After(afterPost) {
				t.Errorf("timestamp out of expected range: %v not between %v and %v",
					eventTime, beforePost, afterPost)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for event")
		}
	})
}

func TestClient_PostEventAsync(t *testing.T) {
	t.Run("posts events asynchronously", func(t *testing.T) {
		eventCount := 0
		var mu sync.Mutex

		mockUI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow processing
			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			eventCount++
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
		}))
		defer mockUI.Close()

		client := NewClient(mockUI.URL, "run-123")

		// Post multiple events quickly
		start := time.Now()
		for i := 0; i < 3; i++ {
			err := client.PostEventAsync("TestEvent", map[string]interface{}{
				"index": i,
			})
			if err != nil {
				t.Fatalf("failed to post event %d: %v", i, err)
			}
		}
		elapsed := time.Since(start)

		// Should return immediately (not wait for slow processing)
		if elapsed > 20*time.Millisecond {
			t.Errorf("async posting took too long: %v", elapsed)
		}

		// Wait for events to be processed
		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if eventCount != 3 {
			t.Errorf("expected 3 events, got %d", eventCount)
		}
	})
}

