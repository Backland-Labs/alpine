// Package events provides event emission interfaces and implementations for Alpine workflow lifecycle events.
// It supports emitting events at key points during workflow execution: RunStarted, RunFinished, and RunError.
// The package includes a mock implementation for testing and a no-op implementation for CLI mode.
package events

import (
	"context"
	"sync"
	"time"
)

// EventEmitter defines the interface for emitting lifecycle events during Alpine workflow execution.
// Implementations can emit events to various destinations (HTTP endpoints, logs, etc.)
type EventEmitter interface {
	// RunStarted is called when a workflow run begins
	RunStarted(runID string, task string)

	// RunFinished is called when a workflow run completes successfully
	RunFinished(runID string, task string)

	// RunError is called when a workflow run encounters an error
	RunError(runID string, task string, err error)

	// StateSnapshot is called when the agent state changes
	StateSnapshot(runID string, snapshot interface{})
}

// ToolCallEventEmitter extends EventEmitter with tool call event capabilities
type ToolCallEventEmitter interface {
	EventEmitter

	// EmitToolCallEvent emits a tool call event (start, end, or error)
	EmitToolCallEvent(event BaseEvent)
}

// MockCall represents a single method call to the MockEmitter
type MockCall struct {
	Method    string
	RunID     string
	Task      string
	Error     error
	Snapshot  interface{}
	Timestamp time.Time
}

// MockEmitter is a test implementation that records all method calls for verification
type MockEmitter struct {
	Calls []MockCall
}

// NewMockEmitter creates a new mock emitter for testing
func NewMockEmitter() *MockEmitter {
	return &MockEmitter{
		Calls: make([]MockCall, 0),
	}
}

// RunStarted records a RunStarted call
func (m *MockEmitter) RunStarted(runID string, task string) {
	m.Calls = append(m.Calls, MockCall{
		Method:    "RunStarted",
		RunID:     runID,
		Task:      task,
		Timestamp: time.Now(),
	})
}

// RunFinished records a RunFinished call
func (m *MockEmitter) RunFinished(runID string, task string) {
	m.Calls = append(m.Calls, MockCall{
		Method:    "RunFinished",
		RunID:     runID,
		Task:      task,
		Timestamp: time.Now(),
	})
}

// RunError records a RunError call
func (m *MockEmitter) RunError(runID string, task string, err error) {
	m.Calls = append(m.Calls, MockCall{
		Method:    "RunError",
		RunID:     runID,
		Task:      task,
		Error:     err,
		Timestamp: time.Now(),
	})
}

// StateSnapshot records a StateSnapshot call
func (m *MockEmitter) StateSnapshot(runID string, snapshot interface{}) {
	m.Calls = append(m.Calls, MockCall{
		Method:    "StateSnapshot",
		RunID:     runID,
		Snapshot:  snapshot,
		Timestamp: time.Now(),
	})
}

// GetLastCall returns the last recorded call or nil if no calls were made
func (m *MockEmitter) GetLastCall() *MockCall {
	if len(m.Calls) == 0 {
		return nil
	}
	return &m.Calls[len(m.Calls)-1]
}

// Reset clears all recorded calls
func (m *MockEmitter) Reset() {
	m.Calls = make([]MockCall, 0)
}

// FindCallsByMethod returns all calls matching the given method name
func (m *MockEmitter) FindCallsByMethod(method string) []MockCall {
	var filtered []MockCall
	for _, call := range m.Calls {
		if call.Method == method {
			filtered = append(filtered, call)
		}
	}
	return filtered
}

// GetStateSnapshots returns all StateSnapshot calls with their snapshot data
func (m *MockEmitter) GetStateSnapshots() []MockCall {
	return m.FindCallsByMethod("StateSnapshot")
}

// GetRawEvents returns all events as raw map data for testing event format
func (m *MockEmitter) GetRawEvents() []map[string]interface{} {
	var events []map[string]interface{}
	for _, call := range m.Calls {
		event := map[string]interface{}{
			"type": call.Method,
			"data": map[string]interface{}{
				"runId": call.RunID,
			},
		}

		// Add method-specific data
		switch call.Method {
		case "RunStarted", "RunFinished":
			event["data"].(map[string]interface{})["task"] = call.Task
		case "RunError":
			event["data"].(map[string]interface{})["task"] = call.Task
			if call.Error != nil {
				event["data"].(map[string]interface{})["error"] = call.Error.Error()
			}
		case "StateSnapshot":
			event["data"].(map[string]interface{})["snapshot"] = call.Snapshot
		}

		events = append(events, event)
	}
	return events
}

// NoOpEmitter is a no-operation implementation used in CLI mode where event emission is disabled
type NoOpEmitter struct{}

// NewNoOpEmitter creates a new no-op emitter
func NewNoOpEmitter() *NoOpEmitter {
	return &NoOpEmitter{}
}

// RunStarted does nothing
func (n *NoOpEmitter) RunStarted(runID string, task string) {
	// No operation
}

// RunFinished does nothing
func (n *NoOpEmitter) RunFinished(runID string, task string) {
	// No operation
}

// RunError does nothing
func (n *NoOpEmitter) RunError(runID string, task string, err error) {
	// No operation
}

// StateSnapshot does nothing
func (n *NoOpEmitter) StateSnapshot(runID string, snapshot interface{}) {
	// No operation
}

// ServerEventEmitter broadcasts events via server's event system
type ServerEventEmitter struct {
	runID string
	// Instead of trying to avoid import cycles, we'll use a function adapter
	broadcastFunc func(eventType, runID string, data map[string]interface{})
}

// NewServerEventEmitter creates a new server-based event emitter
// The broadcastFunc should handle converting to the server's event format
func NewServerEventEmitter(runID string, broadcastFunc func(eventType, runID string, data map[string]interface{})) *ServerEventEmitter {
	return &ServerEventEmitter{
		runID:         runID,
		broadcastFunc: broadcastFunc,
	}
}

// RunStarted broadcasts a RunStarted event
func (s *ServerEventEmitter) RunStarted(runID string, task string) {
	if s.broadcastFunc != nil {
		s.broadcastFunc("RunStarted", runID, map[string]interface{}{
			"task": task,
		})
	}
}

// RunFinished broadcasts a RunFinished event
func (s *ServerEventEmitter) RunFinished(runID string, task string) {
	if s.broadcastFunc != nil {
		s.broadcastFunc("RunFinished", runID, map[string]interface{}{
			"task": task,
		})
	}
}

// RunError broadcasts a RunError event
func (s *ServerEventEmitter) RunError(runID string, task string, err error) {
	if s.broadcastFunc == nil {
		return
	}

	data := map[string]interface{}{
		"task": task,
	}
	if err != nil {
		data["error"] = err.Error()
	}

	s.broadcastFunc("RunError", runID, data)
}

// StateSnapshot broadcasts a StateSnapshot event
func (s *ServerEventEmitter) StateSnapshot(runID string, snapshot interface{}) {
	if s.broadcastFunc != nil {
		s.broadcastFunc("StateSnapshot", runID, map[string]interface{}{
			"snapshot": snapshot,
		})
	}
}

// BatchingConfig holds configuration for the batching emitter
type BatchingConfig struct {
	// FlushInterval is how often to flush batched events (default: 1 second)
	FlushInterval time.Duration

	// RateLimit is the maximum events per second (default: 100)
	RateLimit int

	// BufferSize is the maximum number of events to buffer (default: 1000)
	BufferSize int

	// FlushFunc is called to process batched events
	FlushFunc func(events []BaseEvent)
}

// BatchingEmitter implements event batching and throttling for tool call events
type BatchingEmitter struct {
	config      BatchingConfig
	eventBuffer []BaseEvent
	rateLimiter chan struct{}
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewBatchingEmitter creates a new batching emitter with the given configuration
func NewBatchingEmitter(config BatchingConfig) *BatchingEmitter {
	// Set defaults
	if config.FlushInterval == 0 {
		config.FlushInterval = time.Second
	}
	if config.RateLimit == 0 {
		config.RateLimit = 100
	}
	if config.BufferSize == 0 {
		config.BufferSize = 1000
	}

	ctx, cancel := context.WithCancel(context.Background())

	emitter := &BatchingEmitter{
		config:      config,
		eventBuffer: make([]BaseEvent, 0, config.BufferSize),
		rateLimiter: make(chan struct{}, config.RateLimit),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Fill rate limiter initially
	for i := 0; i < config.RateLimit; i++ {
		emitter.rateLimiter <- struct{}{}
	}

	return emitter
}

// Start begins the batching emitter's background processing
func (b *BatchingEmitter) Start(ctx context.Context) {
	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	// Rate limiter refill goroutine
	go func() {
		refillTicker := time.NewTicker(time.Second)
		defer refillTicker.Stop()

		for {
			select {
			case <-refillTicker.C:
				// Refill rate limiter
			refillLoop:
				for len(b.rateLimiter) < b.config.RateLimit {
					select {
					case b.rateLimiter <- struct{}{}:
					default:
						// Channel is full, stop refilling
						break refillLoop
					}
				}
			case <-ctx.Done():
				return
			case <-b.ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-ctx.Done():
			b.flush() // Final flush
			return
		case <-b.ctx.Done():
			b.flush() // Final flush
			return
		}
	}
}

// EmitToolCallEvent adds a tool call event to the batch
func (b *BatchingEmitter) EmitToolCallEvent(event BaseEvent) {
	// Rate limiting - non-blocking
	select {
	case <-b.rateLimiter:
		// Rate limit allows this event
	default:
		// Rate limit exceeded, drop event
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Backpressure handling - drop oldest events if buffer is full
	if len(b.eventBuffer) >= b.config.BufferSize {
		// Drop oldest event
		b.eventBuffer = b.eventBuffer[1:]
	}

	b.eventBuffer = append(b.eventBuffer, event)
}

// flush processes all buffered events
func (b *BatchingEmitter) flush() {
	b.mu.Lock()
	if len(b.eventBuffer) == 0 {
		b.mu.Unlock()
		return
	}

	// Copy events to process
	events := make([]BaseEvent, len(b.eventBuffer))
	copy(events, b.eventBuffer)

	// Clear buffer
	b.eventBuffer = b.eventBuffer[:0]
	b.mu.Unlock()

	// Process events outside of lock
	if b.config.FlushFunc != nil {
		b.config.FlushFunc(events)
	}
}

// RunStarted implements EventEmitter interface
func (b *BatchingEmitter) RunStarted(runID string, task string) {
	// For now, just pass through to existing behavior
	// Could be extended to emit as BaseEvent in the future
}

// RunFinished implements EventEmitter interface
func (b *BatchingEmitter) RunFinished(runID string, task string) {
	// For now, just pass through to existing behavior
	// Could be extended to emit as BaseEvent in the future
}

// RunError implements EventEmitter interface
func (b *BatchingEmitter) RunError(runID string, task string, err error) {
	// For now, just pass through to existing behavior
	// Could be extended to emit as BaseEvent in the future
}

// StateSnapshot implements EventEmitter interface
func (b *BatchingEmitter) StateSnapshot(runID string, snapshot interface{}) {
	// For now, just pass through to existing behavior
	// Could be extended to emit as BaseEvent in the future
}
