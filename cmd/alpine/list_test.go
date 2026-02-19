package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRunListOwnerScopedRecentWindow(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")

	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}
	now := time.Now().UTC()
	err = store.withLedger(func(ledger *sessionLedger) error {
		ledger.Sessions["ses-active"] = &sessionRecord{
			SessionID:        "ses-active",
			OwnerPrincipalID: "owner-1",
			Branch:           "main",
			RepoURL:          "https://github.com/org/repo.git",
			State:            StateReady,
			UpdatedAt:        now.Add(-48 * time.Hour).Format(time.RFC3339),
		}
		ledger.Sessions["ses-recent"] = &sessionRecord{
			SessionID:        "ses-recent",
			OwnerPrincipalID: "owner-1",
			Branch:           "dev",
			RepoURL:          "https://github.com/org/repo.git",
			State:            StateStopped,
			UpdatedAt:        now.Add(-1 * time.Hour).Format(time.RFC3339),
		}
		ledger.Sessions["ses-stale-terminal"] = &sessionRecord{
			SessionID:        "ses-stale-terminal",
			OwnerPrincipalID: "owner-1",
			Branch:           "old",
			RepoURL:          "https://github.com/org/repo.git",
			State:            StateStopped,
			UpdatedAt:        now.Add(-72 * time.Hour).Format(time.RFC3339),
		}
		ledger.Sessions["ses-other-owner"] = &sessionRecord{
			SessionID:        "ses-other-owner",
			OwnerPrincipalID: "owner-2",
			Branch:           "main",
			RepoURL:          "https://github.com/org/repo.git",
			State:            StateReady,
			UpdatedAt:        now.Format(time.RFC3339),
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	out := captureStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runList(cmd, nil); err != nil {
			t.Fatalf("runList returned error: %v", err)
		}
	})

	var rows []listOutput
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].SessionID != "ses-recent" {
		t.Fatalf("rows[0].session_id = %q, want %q", rows[0].SessionID, "ses-recent")
	}
	if rows[1].SessionID != "ses-active" {
		t.Fatalf("rows[1].session_id = %q, want %q", rows[1].SessionID, "ses-active")
	}
}

func TestRunListAllOwnersRequiresAdmin(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	listAllOwners = true
	t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")
	t.Setenv("ALPINE_ADMIN", "")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runList(cmd, nil)
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

func TestRunListAdditionalBranches(t *testing.T) {
	t.Run("requires principal", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "")
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		err := runList(cmd, nil)
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

	t.Run("all owners with admin", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		jsonOutput = true
		listAllOwners = true
		t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")
		t.Setenv("ALPINE_ADMIN", "1")

		store, err := newSessionStore()
		if err != nil {
			t.Fatalf("newSessionStore: %v", err)
		}
		err = store.withLedger(func(ledger *sessionLedger) error {
			ledger.Sessions["ses-1"] = &sessionRecord{SessionID: "ses-1", OwnerPrincipalID: "owner-1", State: StateReady, Branch: "main", UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
			ledger.Sessions["ses-2"] = &sessionRecord{SessionID: "ses-2", OwnerPrincipalID: "owner-2", State: StateReady, Branch: "dev", UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
			return nil
		})
		if err != nil {
			t.Fatalf("seed ledger: %v", err)
		}

		out := captureStdout(t, func() {
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			if err := runList(cmd, nil); err != nil {
				t.Fatalf("runList returned error: %v", err)
			}
		})

		var rows []listOutput
		if err := json.Unmarshal([]byte(out), &rows); err != nil {
			t.Fatalf("failed to parse output JSON: %v", err)
		}
		if len(rows) != 2 {
			t.Fatalf("len(rows) = %d, want 2", len(rows))
		}
	})

	t.Run("human output no sessions", func(t *testing.T) {
		resetFlags(t)
		setupLedgerPath(t)
		t.Setenv("ALPINE_PRINCIPAL_ID", "owner-1")

		out := captureStdout(t, func() {
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			if err := runList(cmd, nil); err != nil {
				t.Fatalf("runList returned error: %v", err)
			}
		})
		if !strings.Contains(out, "No active or recent sessions") {
			t.Fatalf("unexpected output: %s", out)
		}
	})
}
