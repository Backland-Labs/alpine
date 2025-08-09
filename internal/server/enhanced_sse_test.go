package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEventReplayBuffer tests the core replay buffer functionality
func TestEventReplayBuffer(t *testing.T) {
	server := NewServerWithConfig(0, 100, 10)

	// Create test events
	events := []WorkflowEvent{
		{
			Type:      "workflow_started",
			RunID:     "test-run-1",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"step": 1},
		},
		{
			Type:      events.AGUIEventToolCallStarted,
			RunID:     "test-run-1",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"toolName": "read", "toolCallId": "call-1"},
		},
		{
			Type:      events.AGUIEventToolCallFinished,
			RunID:     "test-run-1",
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"toolName": "read", "toolCallId": "call-1"},
		},
	}

	// Broadcast events to populate replay buffer
	for _, event := range events {
		server.BroadcastEvent(event)
	}

	// Test that replay buffer contains events
	replayEvents := server.GetReplayBuffer("test-run-1")
	assert.Len(t, replayEvents, 3, "Replay buffer should contain all broadcasted events")

	// Test buffer size limit
	bufferLimit := server.GetReplayBufferLimit()
	assert.Equal(t, 1000, bufferLimit, "Default replay buffer limit should be 1000")
}

// TestToolCallEventBroadcasting tests broadcasting of tool call events
func TestToolCallEventBroadcasting(t *testing.T) {
	server := NewServerWithConfig(0, 100, 10)

	// Create batching emitter
	batchingEmitter := events.NewBatchingEmitter(events.BatchingConfig{
		FlushInterval: 100 * time.Millisecond,
		RateLimit:     100,
		BufferSize:    1000,
		FlushFunc: func(batchedEvents []events.BaseEvent) {
			// Convert batched tool call events to workflow events and broadcast
			for _, event := range batchedEvents {
				workflowEvent := WorkflowEvent{
					Type:      event.GetType(),
					RunID:     event.GetRunID(),
					Timestamp: event.GetTimestamp(),
					Data: map[string]interface{}{
						"toolCallId": "test-call-id",
						"toolName":   "test-tool",
					},
				}
				server.BroadcastEvent(workflowEvent)
			}
		},
	})

	server.SetBatchingEmitter(batchingEmitter)

	// Start batching emitter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go batchingEmitter.Start(ctx)

	// Create tool call event
	toolCallEvent := &events.ToolCallStartEvent{
		Type:       events.AGUIEventToolCallStarted,
		RunID:      "test-run-1",
		Timestamp:  time.Now(),
		ToolCallID: "call-123",
		ToolName:   "read",
	}

	// Emit tool call event
	batchingEmitter.EmitToolCallEvent(toolCallEvent)

	// Wait for batching
	time.Sleep(200 * time.Millisecond)

	// Verify event was broadcasted
	replayEvents := server.GetReplayBuffer("test-run-1")
	assert.Len(t, replayEvents, 1, "Tool call event should be in replay buffer")
	assert.Equal(t, events.AGUIEventToolCallStarted, replayEvents[0].Type)
}

// TestEventSequencing tests proper event ordering and correlation
func TestEventSequencing(t *testing.T) {
	server := NewServerWithConfig(0, 100, 10)

	runID := "test-run-sequencing"
	baseTime := time.Now()

	// Create events with specific timestamps to test ordering
	events := []WorkflowEvent{
		{
			Type:        "workflow_started",
			RunID:       runID,
			Timestamp:   baseTime,
			SequenceNum: 1,
			Data:        map[string]interface{}{"step": "start"},
		},
		{
			Type:        events.AGUIEventToolCallStarted,
			RunID:       runID,
			Timestamp:   baseTime.Add(1 * time.Second),
			SequenceNum: 2,
			Data:        map[string]interface{}{"toolCallId": "call-1", "toolName": "read"},
		},
		{
			Type:        events.AGUIEventToolCallFinished,
			RunID:       runID,
			Timestamp:   baseTime.Add(2 * time.Second),
			SequenceNum: 3,
			Data:        map[string]interface{}{"toolCallId": "call-1", "toolName": "read"},
		},
	}

	// Broadcast events out of order to test sequencing
	server.BroadcastEvent(events[2]) // Finish first
	server.BroadcastEvent(events[0]) // Start second
	server.BroadcastEvent(events[1]) // Tool call third

	// Get replay buffer and verify proper sequencing
	replayEvents := server.GetReplayBuffer(runID)
	require.Len(t, replayEvents, 3)

	// Events should be ordered by sequence number
	assert.Equal(t, int64(1), replayEvents[0].SequenceNum)
	assert.Equal(t, int64(2), replayEvents[1].SequenceNum)
	assert.Equal(t, int64(3), replayEvents[2].SequenceNum)
}

// TestRunSpecificSSEWithToolCallEvents tests SSE endpoint with tool call events
func TestRunSpecificSSEWithToolCallEvents(t *testing.T) {
	server := NewServerWithConfig(0, 100, 10)

	// Create test run
	runID := "test-sse-run"
	server.runs = map[string]*Run{
		runID: {
			ID:      runID,
			Status:  "running",
			Created: time.Now(),
			Updated: time.Now(),
		},
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = server.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create SSE request
	url := fmt.Sprintf("http://%s/runs/%s/events", server.Address(), runID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")

	// Make request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify SSE headers
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))

	// Broadcast tool call event after connection
	go func() {
		time.Sleep(50 * time.Millisecond)
		server.BroadcastEvent(WorkflowEvent{
			Type:      events.AGUIEventToolCallStarted,
			RunID:     runID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"toolCallId": "test-call",
				"toolName":   "read",
			},
		})
	}()

	// Read SSE response in chunks to get both connection and tool call events
	var response strings.Builder
	buffer := make([]byte, 512)

	// Read initial connection event
	n, err := resp.Body.Read(buffer)
	require.NoError(t, err)
	response.Write(buffer[:n])

	// Wait a bit and read the tool call event
	time.Sleep(100 * time.Millisecond)
	n, err = resp.Body.Read(buffer)
	if err == nil {
		response.Write(buffer[:n])
	}

	responseStr := response.String()

	// Should contain connection event and tool call event
	assert.Contains(t, responseStr, "connected")
	assert.Contains(t, responseStr, "tool_call_started")
	assert.Contains(t, responseStr, "test-call")
}

// TestEventReplayOnSSEConnection tests that late-joining clients receive replay buffer events
func TestEventReplayOnSSEConnection(t *testing.T) {
	server := NewServerWithConfig(0, 100, 10)

	runID := "test-replay-run"
	server.runs = map[string]*Run{
		runID: {
			ID:      runID,
			Status:  "running",
			Created: time.Now(),
			Updated: time.Now(),
		},
	}

	// Broadcast some events before any client connects
	events := []WorkflowEvent{
		{
			Type:      "workflow_started",
			RunID:     runID,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"step": "start"},
		},
		{
			Type:      events.AGUIEventToolCallStarted,
			RunID:     runID,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"toolCallId": "call-1", "toolName": "read"},
		},
	}

	for _, event := range events {
		server.BroadcastEvent(event)
	}

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = server.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Create SSE request (late-joining client)
	url := fmt.Sprintf("http://%s/runs/%s/events", server.Address(), runID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")

	// Make request
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Read SSE response
	var response strings.Builder
	buffer := make([]byte, 2048)

	// Read all available data
	for i := 0; i < 3; i++ { // Try to read multiple chunks
		n, err := resp.Body.Read(buffer)
		if err != nil {
			break
		}
		response.Write(buffer[:n])
		time.Sleep(50 * time.Millisecond) // Small delay between reads
	}

	responseStr := response.String()

	// Should contain connection event and replayed events
	assert.Contains(t, responseStr, "connected")
	assert.Contains(t, responseStr, "workflow_started")
	assert.Contains(t, responseStr, "tool_call_started")
	assert.Contains(t, responseStr, "call-1")
}
