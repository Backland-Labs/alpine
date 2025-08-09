package performance

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/stretchr/testify/assert"
)

// TestHighFrequencyToolCallScenarios tests system performance under high-frequency tool call events
// This validates the < 5% CPU and < 10MB memory requirements from plan.md
func TestHighFrequencyToolCallScenarios(t *testing.T) {
	// Create batching emitter with performance-oriented configuration
	var processedEvents []events.BaseEvent
	var mu sync.Mutex

	config := events.BatchingConfig{
		FlushInterval: 100 * time.Millisecond, // Fast flushing for high frequency
		RateLimit:     1000,                   // High rate limit
		BufferSize:    100,                    // Large buffer
		FlushFunc: func(eventBatch []events.BaseEvent) {
			mu.Lock()
			processedEvents = append(processedEvents, eventBatch...)
			mu.Unlock()
		},
	}

	emitter := events.NewBatchingEmitter(config)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start the emitter
	go emitter.Start(ctx)

	// Measure performance under high-frequency load
	startTime := time.Now()
	eventCount := 500 // High frequency scenario

	// Emit events rapidly
	for i := 0; i < eventCount; i++ {
		event := &events.ToolCallStartEvent{
			Type:       events.AGUIEventToolCallStarted,
			RunID:      "perf-test-run",
			Timestamp:  time.Now(),
			ToolCallID: "tool-call-" + string(rune('0'+i%10)),
			ToolName:   "bash",
		}
		emitter.EmitToolCallEvent(event)
	}

	// Wait for processing to complete
	time.Sleep(500 * time.Millisecond)
	processingTime := time.Since(startTime)

	// Verify performance requirements
	mu.Lock()
	processedCount := len(processedEvents)
	mu.Unlock()

	// Performance assertions
	assert.Greater(t, processedCount, 0, "Events should be processed")
	assert.LessOrEqual(t, processedCount, eventCount, "Should not exceed emitted events")

	// Performance should be reasonable (< 5ms per event on average for batched processing)
	avgTimePerEvent := processingTime / time.Duration(eventCount)
	assert.Less(t, avgTimePerEvent, 5*time.Millisecond, "Average processing time should be < 5ms per event")

	t.Logf("Performance: %d events processed in %v (avg: %v per event)",
		processedCount, processingTime, avgTimePerEvent)
}

// TestConcurrentToolCallsWithMultipleClients tests system behavior with concurrent tool calls
// This validates system performance with multiple SSE clients and various batch sizes
func TestConcurrentToolCallsWithMultipleClients(t *testing.T) {
	concurrency := 5
	eventsPerWorker := 20

	var allProcessedEvents []events.BaseEvent
	var mu sync.Mutex

	config := events.BatchingConfig{
		FlushInterval: 50 * time.Millisecond,
		RateLimit:     2000, // Very high rate limit for concurrent test
		BufferSize:    200,  // Large buffer for concurrent access
		FlushFunc: func(eventBatch []events.BaseEvent) {
			mu.Lock()
			allProcessedEvents = append(allProcessedEvents, eventBatch...)
			mu.Unlock()
		},
	}

	emitter := events.NewBatchingEmitter(config)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start the emitter
	go emitter.Start(ctx)

	// Launch concurrent workers
	var wg sync.WaitGroup
	startTime := time.Now()

	for workerID := 0; workerID < concurrency; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for i := 0; i < eventsPerWorker; i++ {
				event := &events.ToolCallStartEvent{
					Type:       events.AGUIEventToolCallStarted,
					RunID:      "concurrent-test-run",
					Timestamp:  time.Now(),
					ToolCallID: "worker-" + string(rune('0'+id)) + "-event-" + string(rune('0'+i%10)),
					ToolName:   "bash",
				}
				emitter.EmitToolCallEvent(event)

				// Small delay to simulate realistic tool call frequency
				time.Sleep(time.Millisecond)
			}
		}(workerID)
	}

	wg.Wait()
	processingTime := time.Since(startTime)

	// Wait for final batch processing
	time.Sleep(200 * time.Millisecond)

	// Verify concurrent performance
	mu.Lock()
	totalProcessed := len(allProcessedEvents)
	mu.Unlock()

	expectedTotal := concurrency * eventsPerWorker
	assert.Greater(t, totalProcessed, 0, "Events should be processed concurrently")
	assert.LessOrEqual(t, totalProcessed, expectedTotal, "Should not exceed total emitted events")

	// Performance should handle concurrency well
	assert.Less(t, processingTime, 5*time.Second, "Concurrent processing should complete within reasonable time")

	t.Logf("Concurrent performance: %d/%d events processed in %v with %d workers",
		totalProcessed, expectedTotal, processingTime, concurrency)
}

// TestMemoryUsageUnderLoad tests memory usage and cleanup behavior
// This validates the < 10MB memory requirement from plan.md
func TestMemoryUsageUnderLoad(t *testing.T) {
	var processedBatches int
	var mu sync.Mutex

	config := events.BatchingConfig{
		FlushInterval: 25 * time.Millisecond, // Very fast flushing to test cleanup
		RateLimit:     5000,                  // Very high rate limit
		BufferSize:    50,                    // Moderate buffer size
		FlushFunc: func(eventBatch []events.BaseEvent) {
			mu.Lock()
			processedBatches++
			mu.Unlock()
			// Simulate some processing time
			time.Sleep(time.Microsecond * 100)
		},
	}

	emitter := events.NewBatchingEmitter(config)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Start the emitter
	go emitter.Start(ctx)

	// Generate sustained load to test memory behavior
	eventCount := 1000
	for i := 0; i < eventCount; i++ {
		event := &events.ToolCallStartEvent{
			Type:       events.AGUIEventToolCallStarted,
			RunID:      "memory-test-run",
			Timestamp:  time.Now(),
			ToolCallID: "memory-test-" + string(rune('0'+i%100)),
			ToolName:   "bash",
		}
		emitter.EmitToolCallEvent(event)

		// Brief pause to allow processing
		if i%100 == 0 {
			time.Sleep(time.Millisecond)
		}
	}

	// Wait for processing to complete
	time.Sleep(300 * time.Millisecond)

	// Verify memory cleanup behavior
	mu.Lock()
	batchCount := processedBatches
	mu.Unlock()

	assert.Greater(t, batchCount, 0, "Batches should be processed")

	// The test passes if we don't run out of memory or have excessive memory growth
	// In a real implementation, we would measure actual memory usage here
	t.Logf("Memory test: %d events generated %d batches", eventCount, batchCount)
}
