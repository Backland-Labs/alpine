package main

import "testing"

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from LifecycleState
		to   LifecycleState
		ok   bool
	}{
		{StateRequested, StateProvisioning, true},
		{StateProvisioning, StateRepoSyncing, true},
		{StateRepoSyncing, StateInstalling, true},
		{StateInstalling, StateReady, true},
		{StateReady, StateStopping, true},
		{StatePersisting, StateStopped, true},
		{StateStoppedUnpersisted, StatePersisting, true},
		{StateReady, StatePersisting, false},
		{StateRequested, StateReady, false},
		{StateFailed, StateReady, false},
	}

	for _, tt := range tests {
		if got := canTransition(tt.from, tt.to); got != tt.ok {
			t.Fatalf("canTransition(%s, %s) = %t, want %t", tt.from, tt.to, got, tt.ok)
		}
	}
}

func TestValidateTransition(t *testing.T) {
	if err := validateTransition(StateStopping, StatePersisting); err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
	if err := validateTransition(StateStopping, StateReady); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
