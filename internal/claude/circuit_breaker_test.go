package claude

import (
	"testing"
	"time"
)

func TestCircuitBreaker_HappyPath(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)

	// Should allow calls when circuit is closed
	if !cb.CanCall() {
		t.Error("Circuit breaker should allow calls initially")
	}

	// Record successful call
	cb.RecordSuccess()

	// Should still allow calls after success
	if !cb.CanCall() {
		t.Error("Circuit breaker should allow calls after success")
	}
}

func TestCircuitBreaker_FailureThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)

	// Record failures up to threshold
	cb.RecordFailure()
	cb.RecordFailure()

	// Should still allow calls before threshold
	if !cb.CanCall() {
		t.Error("Circuit breaker should allow calls before threshold")
	}

	// Record failure that hits threshold
	cb.RecordFailure()

	// Should now block calls
	if cb.CanCall() {
		t.Error("Circuit breaker should block calls after threshold")
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	cb := NewCircuitBreaker(2, 100*time.Millisecond)

	// Trip the circuit
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.CanCall() {
		t.Error("Circuit should be open after failures")
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Should allow one call to test recovery
	if !cb.CanCall() {
		t.Error("Circuit should allow test call after timeout")
	}

	// Record success to close circuit
	cb.RecordSuccess()

	// Should now allow calls normally
	if !cb.CanCall() {
		t.Error("Circuit should be closed after successful recovery")
	}
}
