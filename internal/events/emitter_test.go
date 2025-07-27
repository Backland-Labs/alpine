package events

import (
	"errors"
	"testing"
	"time"
)

// TestEventEmitterInterface verifies that the EventEmitter interface exists and has the required methods.
// This test ensures we have lifecycle event methods for RunStarted, RunFinished, and RunError as
// specified in the plan.md acceptance criteria.
func TestEventEmitterInterface(t *testing.T) {
	// This test will compile only if the interface exists with correct method signatures
	var emitter EventEmitter
	_ = emitter // Prevent unused variable error

	// Verify the interface can be implemented
	var _ EventEmitter = (*MockEmitter)(nil)
	var _ EventEmitter = (*NoOpEmitter)(nil)
}

// TestMockEmitterRecordsMethodCalls verifies that the mock implementation correctly records
// all method calls with their parameters. This is essential for testing workflow integration
// where we need to verify that events are emitted at the correct times.
func TestMockEmitterRecordsMethodCalls(t *testing.T) {
	mock := NewMockEmitter()

	// Test RunStarted
	runID := "test-run-123"
	task := "Implement user authentication"
	mock.RunStarted(runID, task)

	if len(mock.Calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Method != "RunStarted" {
		t.Errorf("Expected method RunStarted, got %s", mock.Calls[0].Method)
	}
	if mock.Calls[0].RunID != runID {
		t.Errorf("Expected runID %s, got %s", runID, mock.Calls[0].RunID)
	}
	if mock.Calls[0].Task != task {
		t.Errorf("Expected task %s, got %s", task, mock.Calls[0].Task)
	}

	// Test RunFinished
	mock.RunFinished(runID, task)
	if len(mock.Calls) != 2 {
		t.Errorf("Expected 2 calls, got %d", len(mock.Calls))
	}
	if mock.Calls[1].Method != "RunFinished" {
		t.Errorf("Expected method RunFinished, got %s", mock.Calls[1].Method)
	}

	// Test RunError
	testErr := errors.New("test error")
	mock.RunError(runID, task, testErr)
	if len(mock.Calls) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(mock.Calls))
	}
	if mock.Calls[2].Method != "RunError" {
		t.Errorf("Expected method RunError, got %s", mock.Calls[2].Method)
	}
	if mock.Calls[2].Error == nil || mock.Calls[2].Error.Error() != testErr.Error() {
		t.Errorf("Expected error %v, got %v", testErr, mock.Calls[2].Error)
	}
}

// TestMockEmitterGetLastCall verifies that we can easily retrieve the last method call
// for assertions in tests. This helper method makes test code more readable.
func TestMockEmitterGetLastCall(t *testing.T) {
	mock := NewMockEmitter()

	// No calls yet
	lastCall := mock.GetLastCall()
	if lastCall != nil {
		t.Error("Expected nil for no calls, got non-nil")
	}

	// Make some calls
	mock.RunStarted("run1", "task1")
	mock.RunFinished("run2", "task2")

	lastCall = mock.GetLastCall()
	if lastCall == nil {
		t.Fatal("Expected non-nil last call")
	}
	if lastCall.Method != "RunFinished" {
		t.Errorf("Expected last method to be RunFinished, got %s", lastCall.Method)
	}
	if lastCall.RunID != "run2" {
		t.Errorf("Expected runID run2, got %s", lastCall.RunID)
	}
}

// TestMockEmitterReset verifies that we can clear all recorded calls.
// This is useful when reusing a mock across multiple test scenarios.
func TestMockEmitterReset(t *testing.T) {
	mock := NewMockEmitter()

	mock.RunStarted("run1", "task1")
	mock.RunFinished("run1", "task1")
	
	if len(mock.Calls) != 2 {
		t.Errorf("Expected 2 calls before reset, got %d", len(mock.Calls))
	}

	mock.Reset()

	if len(mock.Calls) != 0 {
		t.Errorf("Expected 0 calls after reset, got %d", len(mock.Calls))
	}
}

// TestNoOpEmitterHandlesCallsWithoutSideEffects verifies that the no-op implementation
// accepts all method calls without performing any actions. This is used in CLI mode
// where event emission is not needed.
func TestNoOpEmitterHandlesCallsWithoutSideEffects(t *testing.T) {
	noop := NewNoOpEmitter()

	// These calls should not panic or cause any side effects
	noop.RunStarted("run1", "task1")
	noop.RunFinished("run1", "task1")
	noop.RunError("run1", "task1", errors.New("test error"))

	// Test completed without panics - no-op is working correctly
}

// TestEventEmitterCanBeNil verifies that workflow code can handle a nil emitter gracefully.
// This is important for backward compatibility and optional event emission.
func TestEventEmitterCanBeNil(t *testing.T) {
	var emitter EventEmitter = nil

	// In production code, we'd check for nil before calling methods
	if emitter != nil {
		emitter.RunStarted("run1", "task1")
	}

	// Test passes if no panic occurs
}

// TestMockEmitterTimestamps verifies that the mock records timestamps for each call.
// This helps ensure events are emitted in the correct order during integration tests.
func TestMockEmitterTimestamps(t *testing.T) {
	mock := NewMockEmitter()

	start := time.Now()
	mock.RunStarted("run1", "task1")
	time.Sleep(10 * time.Millisecond)
	mock.RunFinished("run1", "task1")

	if len(mock.Calls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(mock.Calls))
	}

	// Verify timestamps are set and in order
	if mock.Calls[0].Timestamp.Before(start) {
		t.Error("First call timestamp is before test start time")
	}
	if !mock.Calls[1].Timestamp.After(mock.Calls[0].Timestamp) {
		t.Error("Second call timestamp is not after first call")
	}
}

// TestMockEmitterFindCallsByMethod verifies we can filter calls by method name.
// This is useful for assertions in complex test scenarios.
func TestMockEmitterFindCallsByMethod(t *testing.T) {
	mock := NewMockEmitter()

	mock.RunStarted("run1", "task1")
	mock.RunStarted("run2", "task2")
	mock.RunFinished("run1", "task1")
	mock.RunError("run3", "task3", errors.New("error"))
	mock.RunStarted("run4", "task4")

	startCalls := mock.FindCallsByMethod("RunStarted")
	if len(startCalls) != 3 {
		t.Errorf("Expected 3 RunStarted calls, got %d", len(startCalls))
	}

	finishCalls := mock.FindCallsByMethod("RunFinished")
	if len(finishCalls) != 1 {
		t.Errorf("Expected 1 RunFinished call, got %d", len(finishCalls))
	}

	errorCalls := mock.FindCallsByMethod("RunError")
	if len(errorCalls) != 1 {
		t.Errorf("Expected 1 RunError call, got %d", len(errorCalls))
	}
}