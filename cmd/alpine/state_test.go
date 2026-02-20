package main

import "testing"

func TestTransitionLifecycle(t *testing.T) {
	if err := transitionLifecycle(stateNew, stateProvisioning); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
	if err := transitionLifecycle(stateRunning, stateDestroyed); err == nil {
		t.Fatal("expected invalid transition")
	}
	if err := transitionLifecycle(stateRunning, stateRunning); err != nil {
		t.Fatalf("same-state transition should pass: %v", err)
	}
}

func TestCanExportFrom(t *testing.T) {
	if !canExportFrom(stateRunning) || !canExportFrom(stateCompleted) || !canExportFrom(stateDestroyed) {
		t.Fatal("expected running/completed/destroyed to be exportable")
	}
	if canExportFrom(stateSaving) || canExportFrom(stateTearingDown) {
		t.Fatal("saving and tearing_down should not be exportable")
	}
}
