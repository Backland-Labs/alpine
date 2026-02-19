package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRunStatusJSON(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")

	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = store.withLedger(func(ledger *sessionLedger) error {
		ledger.Sessions["ses-1"] = &sessionRecord{
			SessionID:        "ses-1",
			OwnerPrincipalID: "owner-1",
			RepoURL:          "https://github.com/org/repo.git",
			Branch:           "main",
			State:            StateStoppedUnpersisted,
			UpdatedAt:        now,
			LastError: &sessionError{
				ErrorCode: ErrPersistPayloadTooLarge,
				Cause:     "payload too large",
			},
			OperationHistory: []operationRecord{{OperationID: "op-1"}},
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	out := captureStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runStatus(cmd, []string{"ses-1"}); err != nil {
			t.Fatalf("runStatus returned error: %v", err)
		}
	})

	var result statusOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if result.SessionID != "ses-1" {
		t.Fatalf("session_id = %q, want %q", result.SessionID, "ses-1")
	}
	if result.NextStep == "" {
		t.Fatal("expected next_step")
	}
	if result.OperationID != "op-1" {
		t.Fatalf("operation_id = %q, want %q", result.OperationID, "op-1")
	}
}

func TestRunStatusSessionNotFound(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runStatus(cmd, []string{"missing"})
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
}

func TestRunStatusAuthAndOutput(t *testing.T) {
	t.Run("requires principal", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runStatus(cmd, []string{"missing"})
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

	t.Run("forbidden without admin", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "owner-2")

		store, err := newSessionStore()
		if err != nil {
			t.Fatalf("newSessionStore: %v", err)
		}
		err = store.withLedger(func(ledger *sessionLedger) error {
			ledger.Sessions["ses-1"] = &sessionRecord{SessionID: "ses-1", OwnerPrincipalID: "owner-1", State: StateReady, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
			return nil
		})
		if err != nil {
			t.Fatalf("seed ledger: %v", err)
		}

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err = runStatus(cmd, []string{"ses-1"})
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
	})

	t.Run("human output includes next step", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")

		store, err := newSessionStore()
		if err != nil {
			t.Fatalf("newSessionStore: %v", err)
		}
		err = store.withLedger(func(ledger *sessionLedger) error {
			ledger.Sessions["ses-1"] = &sessionRecord{
				SessionID:        "ses-1",
				OwnerPrincipalID: "owner-1",
				RepoURL:          "https://github.com/org/repo.git",
				Branch:           "main",
				State:            StateReady,
				UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
			}
			return nil
		})
		if err != nil {
			t.Fatalf("seed ledger: %v", err)
		}

		out := captureStdout(t, func() {
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			if err := runStatus(cmd, []string{"ses-1"}); err != nil {
				t.Fatalf("runStatus returned error: %v", err)
			}
		})
		if !strings.Contains(out, "Next step") {
			t.Fatalf("expected next step in output: %s", out)
		}
	})
}

func TestRecommendedNextStep(t *testing.T) {
	tests := []struct {
		state LifecycleState
		want  string
	}{
		{StateReady, "run alpine down ses-1 when work is complete"},
		{StateStoppedUnpersisted, "run alpine down ses-1 --retry-persist"},
		{StatePersistFailed, "run alpine down ses-1 --retry-persist"},
		{StateFailed, "review status last_error and run alpine up again with a new client-request-id"},
		{StateStopped, "session is fully stopped and persisted"},
		{StateProvisioning, "wait for the active operation to finish, then run alpine status again"},
	}
	for _, tt := range tests {
		session := &sessionRecord{SessionID: "ses-1", State: tt.state}
		if got := recommendedNextStep(session); got != tt.want {
			t.Fatalf("recommendedNextStep(%s) = %q, want %q", tt.state, got, tt.want)
		}
	}
}
