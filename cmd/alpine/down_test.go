package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func seedSession(t *testing.T, state LifecycleState, owner string) string {
	t.Helper()
	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}
	sessionID := "ses_test"
	now := time.Now().UTC().Format(time.RFC3339)
	err = store.withLedger(func(ledger *sessionLedger) error {
		ledger.Sessions[sessionID] = &sessionRecord{
			SchemaVersion:    persistSchemaVersion,
			SessionID:        sessionID,
			PrincipalID:      owner,
			OwnerPrincipalID: owner,
			RepoURL:          "https://github.com/org/repo.git",
			Branch:           "main",
			State:            state,
			StartedAt:        now,
			ReadyAt:          now,
			UpdatedAt:        now,
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed ledger: %v", err)
	}
	return sessionID
}

func TestRunDownSuccess(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")

	sessionID := seedSession(t, StateReady, "user-1")

	out := captureStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runDown(cmd, []string{sessionID}); err != nil {
			t.Fatalf("runDown returned error: %v", err)
		}
	})

	var result downOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if result.State != StateStopped {
		t.Fatalf("state = %q, want %q", result.State, StateStopped)
	}
	if !result.PersistenceVerified {
		t.Fatal("expected persistence_verified=true")
	}
	if result.DurableObjectID == "" {
		t.Fatal("expected durable_object_id")
	}
}

func TestRunDownForbidden(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-2")

	sessionID := seedSession(t, StateReady, "user-1")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runDown(cmd, []string{sessionID})
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected *commandError, got %T", err)
	}
	if ce.errorCode != ErrSessionForbidden {
		t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrSessionForbidden)
	}
}

func TestRunDownRetryPersistRequired(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")

	sessionID := seedSession(t, StateStoppedUnpersisted, "user-1")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runDown(cmd, []string{sessionID})
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected *commandError, got %T", err)
	}
	if ce.errorCode != ErrPersistenceVerification {
		t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrPersistenceVerification)
	}
}

func TestRunDownValidationAndIdempotency(t *testing.T) {
	t.Run("requires principal", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runDown(cmd, []string{"ses-missing"})
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*commandError)
		if !ok {
			t.Fatalf("expected *commandError, got %T", err)
		}
		if ce.errorCode != ErrCallerIdentityRequired {
			t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrCallerIdentityRequired)
		}
	})

	t.Run("session not found", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runDown(cmd, []string{"missing"})
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*commandError)
		if !ok {
			t.Fatalf("expected *commandError, got %T", err)
		}
		if ce.errorCode != ErrSessionNotFound {
			t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrSessionNotFound)
		}
	})

	t.Run("invalid lifecycle state", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
		sessionID := seedSession(t, StateProvisioning, "user-1")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runDown(cmd, []string{sessionID})
		if err == nil {
			t.Fatal("expected error")
		}
		ce, ok := err.(*commandError)
		if !ok {
			t.Fatalf("expected *commandError, got %T", err)
		}
		if ce.errorCode != ErrOperationConflict {
			t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrOperationConflict)
		}
	})

	t.Run("idempotent duplicate returns same operation", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		jsonOutput = true
		t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
		sessionID := seedSession(t, StateReady, "user-1")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		runOnce := func() downOutput {
			out := captureStdout(t, func() {
				if err := runDown(cmd, []string{sessionID}); err != nil {
					t.Fatalf("runDown returned error: %v", err)
				}
			})
			var result downOutput
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("failed to parse output JSON: %v", err)
			}
			return result
		}

		first := runOnce()
		second := runOnce()
		if first.OperationID != second.OperationID {
			t.Fatalf("operation ids differ: %q vs %q", first.OperationID, second.OperationID)
		}
	})
}

func TestRunDownRetryPersistSuccess(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	downRetryPersist = true
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")

	sessionID := seedSession(t, StateStoppedUnpersisted, "user-1")
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	out := captureStdout(t, func() {
		if err := runDown(cmd, []string{sessionID}); err != nil {
			t.Fatalf("runDown returned error: %v", err)
		}
	})

	var result downOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if result.State != StateStopped {
		t.Fatalf("state = %q, want %q", result.State, StateStopped)
	}
}

func TestPersistedPayloadWithCompaction(t *testing.T) {
	session := &sessionRecord{
		SessionID:   "ses-big",
		PrincipalID: "owner",
		RepoURL:     "https://github.com/org/repo.git",
		Branch:      "main",
		State:       StateStopped,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		StoppedAt:   time.Now().UTC().Format(time.RFC3339),
		StateTransitions: []stateTransition{
			{State: StateRequested, Timestamp: time.Now().UTC().Format(time.RFC3339), Actor: "a", Reason: strings.Repeat("x", 64), OperationID: "op"},
		},
		OperationHistory: []operationRecord{
			{OperationID: "op", Action: "up", IdempotencyKey: "k", StartedAt: time.Now().UTC().Format(time.RFC3339), CompletedAt: time.Now().UTC().Format(time.RFC3339), TerminalState: string(StateStopped)},
		},
	}

	for i := 0; i < 2000; i++ {
		session.StateTransitions = append(session.StateTransitions, stateTransition{
			State:       StateReady,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			Actor:       "actor",
			Reason:      strings.Repeat("reason", 20),
			OperationID: "op",
		})
		session.OperationHistory = append(session.OperationHistory, operationRecord{
			OperationID:    "op",
			IdempotencyKey: "k",
			Action:         "up",
			StartedAt:      time.Now().UTC().Format(time.RFC3339),
			CompletedAt:    time.Now().UTC().Format(time.RFC3339),
			TerminalState:  string(StateStopped),
		})
	}

	payload, truncated, err := persistedPayloadWithCompaction(session)
	if err != nil {
		t.Fatalf("persistedPayloadWithCompaction returned error: %v", err)
	}
	if !truncated {
		t.Fatal("expected truncated=true")
	}
	if payload["history_truncated"] != true {
		t.Fatalf("history_truncated = %v, want true", payload["history_truncated"])
	}
}

func TestPersistedPayloadTooLargeAfterCompaction(t *testing.T) {
	veryLarge := strings.Repeat("x", maxPersistBytes+1024)
	session := &sessionRecord{
		SessionID:   "ses-oversize",
		PrincipalID: "owner",
		RepoURL:     "https://github.com/org/repo.git",
		Branch:      "main",
		State:       StateStopped,
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		StoppedAt:   time.Now().UTC().Format(time.RFC3339),
		LastError: &sessionError{
			ErrorCode: ErrPersistPayloadTooLarge,
			Cause:     veryLarge,
		},
		StateTransitions: []stateTransition{{State: StateStopped}},
		OperationHistory: []operationRecord{{OperationID: "op"}},
	}

	_, _, err := persistedPayloadWithCompaction(session)
	if err == nil {
		t.Fatal("expected oversize error")
	}
}
