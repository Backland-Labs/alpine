// Package events provides event emission interfaces and implementations for Alpine workflow lifecycle events.
// It supports emitting events at key points during workflow execution: RunStarted, RunFinished, and RunError.
// The package includes a mock implementation for testing and a no-op implementation for CLI mode.
package events

import "time"

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

