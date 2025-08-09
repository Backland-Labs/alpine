package claude

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestResilientHookExecution_ContinuesOnFailure(t *testing.T) {
	executor := &Executor{}

	// Mock hook that always fails
	failingHook := func() error {
		return errors.New("hook failed")
	}

	// Execute hook with resilience
	err := executor.ExecuteHookWithResilience(context.Background(), "test-hook", failingHook)

	// Should not return error - workflow should continue
	if err != nil {
		t.Errorf("Expected no error for resilient hook execution, got: %v", err)
	}
}

func TestResilientHookExecution_CircuitBreakerTrips(t *testing.T) {
	executor := &Executor{
		hookCircuitBreaker: NewCircuitBreaker(2, 100*time.Millisecond),
	}

	failingHook := func() error {
		return errors.New("hook failed")
	}

	// Execute failing hook multiple times
	_ = executor.ExecuteHookWithResilience(context.Background(), "test-hook", failingHook)
	_ = executor.ExecuteHookWithResilience(context.Background(), "test-hook", failingHook)

	// Circuit should now be open
	if executor.hookCircuitBreaker.CanCall() {
		t.Error("Circuit breaker should be open after consecutive failures")
	}

	// Further executions should be fast-failed
	start := time.Now()
	_ = executor.ExecuteHookWithResilience(context.Background(), "test-hook", failingHook)
	duration := time.Since(start)

	// Should be very fast (< 10ms) due to circuit breaker
	if duration > 10*time.Millisecond {
		t.Errorf("Expected fast failure due to circuit breaker, took %v", duration)
	}
}

func TestResilientHookExecution_SuccessfulRecovery(t *testing.T) {
	executor := &Executor{
		hookCircuitBreaker: NewCircuitBreaker(2, 50*time.Millisecond),
	}

	// Trip the circuit
	failingHook := func() error {
		return errors.New("hook failed")
	}
	executor.ExecuteHookWithResilience(context.Background(), "test-hook", failingHook)
	executor.ExecuteHookWithResilience(context.Background(), "test-hook", failingHook)

	// Wait for recovery
	time.Sleep(60 * time.Millisecond)

	// Execute successful hook
	successfulHook := func() error {
		return nil
	}
	executor.ExecuteHookWithResilience(context.Background(), "test-hook", successfulHook)

	// Circuit should be closed again
	if !executor.hookCircuitBreaker.CanCall() {
		t.Error("Circuit breaker should be closed after successful recovery")
	}
}
