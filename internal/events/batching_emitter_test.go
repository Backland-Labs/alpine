package events

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestBatchingEmitterInterface verifies that the batching emitter implements EventEmitter
// and provides additional tool call event methods
func TestBatchingEmitterInterface(t *testing.T) {
	// This test will compile only if BatchingEmitter implements EventEmitter
	var emitter EventEmitter = NewBatchingEmitter(BatchingConfig{
		FlushInterval: time.Second,
		RateLimit:     100,
		BufferSize:    1000,
	})
	_ = emitter // Prevent unused variable error
}

// TestBatchingEmitterFlushesEventsAfterInterval tests the core batching behavior
// This is critical business logic that must work correctly
func TestBatchingEmitterFlushesEventsAfterInterval(t *testing.T) {
	var flushedEvents []BaseEvent
	var mu sync.Mutex

	flushFunc := func(events []BaseEvent) {
		mu.Lock()
		defer mu.Unlock()
		flushedEvents = append(flushedEvents, events...)
	}

	emitter := NewBatchingEmitter(BatchingConfig{
		FlushInterval: 50 * time.Millisecond, // Short interval for testing
		RateLimit:     100,
		BufferSize:    1000,
		FlushFunc:     flushFunc,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Start the emitter
	go emitter.Start(ctx)

	// Emit a tool call event
	event := &ToolCallStartEvent{
		Type:       AGUIEventToolCallStarted,
		RunID:      "test-run",
		Timestamp:  time.Now(),
		ToolCallID: "tool-123",
		ToolName:   "bash",
	}

	emitter.EmitToolCallEvent(event)

	// Wait for flush interval + buffer
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(flushedEvents) != 1 {
		t.Errorf("Expected 1 flushed event, got %d", len(flushedEvents))
	}

	if len(flushedEvents) > 0 {
		if flushedEvents[0].GetType() != AGUIEventToolCallStarted {
			t.Errorf("Expected event type %s, got %s", AGUIEventToolCallStarted, flushedEvents[0].GetType())
		}
	}
}

// TestBatchingEmitterRateLimiting tests that events are throttled when rate limit is exceeded
// This prevents system overload under high-frequency tool call scenarios
func TestBatchingEmitterRateLimiting(t *testing.T) {
	var flushedEvents []BaseEvent
	var mu sync.Mutex

	flushFunc := func(events []BaseEvent) {
		mu.Lock()
		defer mu.Unlock()
		flushedEvents = append(flushedEvents, events...)
	}

	emitter := NewBatchingEmitter(BatchingConfig{
		FlushInterval: time.Second, // Long interval to test rate limiting
		RateLimit:     2,           // Very low rate limit for testing
		BufferSize:    1000,
		FlushFunc:     flushFunc,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the emitter
	go emitter.Start(ctx)

	// Emit 5 events rapidly - should be rate limited to 2
	for i := 0; i < 5; i++ {
		event := &ToolCallStartEvent{
			Type:       AGUIEventToolCallStarted,
			RunID:      "test-run",
			Timestamp:  time.Now(),
			ToolCallID: "tool-" + string(rune('1'+i)),
			ToolName:   "bash",
		}
		emitter.EmitToolCallEvent(event)
	}

	// Wait for processing
	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have rate limited to 2 events per second
	if len(flushedEvents) > 3 { // Allow some tolerance
		t.Errorf("Expected rate limiting to keep events <= 3, got %d", len(flushedEvents))
	}
}

// TestBatchingEmitterBackpressureHandling tests that the emitter handles buffer overflow gracefully
// This prevents memory overflow under extreme load
func TestBatchingEmitterBackpressureHandling(t *testing.T) {
	var flushedEvents []BaseEvent
	var mu sync.Mutex

	flushFunc := func(events []BaseEvent) {
		mu.Lock()
		defer mu.Unlock()
		flushedEvents = append(flushedEvents, events...)
		// Simulate slow processing
		time.Sleep(10 * time.Millisecond)
	}

	emitter := NewBatchingEmitter(BatchingConfig{
		FlushInterval: 100 * time.Millisecond,
		RateLimit:     1000, // High rate limit
		BufferSize:    5,    // Very small buffer for testing
		FlushFunc:     flushFunc,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Start the emitter
	go emitter.Start(ctx)

	// Emit more events than buffer can hold
	for i := 0; i < 10; i++ {
		event := &ToolCallStartEvent{
			Type:       AGUIEventToolCallStarted,
			RunID:      "test-run",
			Timestamp:  time.Now(),
			ToolCallID: "tool-" + string(rune('1'+i)),
			ToolName:   "bash",
		}
		// This should not block or panic due to backpressure handling
		emitter.EmitToolCallEvent(event)
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Test passes if no panic occurred and some events were processed
	mu.Lock()
	defer mu.Unlock()

	if len(flushedEvents) == 0 {
		t.Error("Expected some events to be processed despite backpressure")
	}
}
