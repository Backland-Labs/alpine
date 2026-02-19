package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStoreLoadSaveAndWithLedger(t *testing.T) {
	resetFlags(t)
	ledgerPath := setupLedgerPath(t)

	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}

	err = store.withLedger(func(ledger *sessionLedger) error {
		ledger.Sessions["ses-1"] = &sessionRecord{SessionID: "ses-1", OwnerPrincipalID: "owner", UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
		return nil
	})
	if err != nil {
		t.Fatalf("withLedger: %v", err)
	}

	if _, statErr := os.Stat(ledgerPath); statErr != nil {
		t.Fatalf("expected ledger file at %s: %v", ledgerPath, statErr)
	}

	loaded, err := store.loadLedger()
	if err != nil {
		t.Fatalf("loadLedger: %v", err)
	}
	if loaded.Sessions["ses-1"] == nil {
		t.Fatal("expected persisted session")
	}
}

func TestSessionStoreLoadLedgerInvalidJSON(t *testing.T) {
	resetFlags(t)
	path := filepath.Join(t.TempDir(), "bad.json")
	t.Setenv("ALPINE_LEDGER_PATH", path)
	if err := os.WriteFile(path, []byte("{bad json"), 0600); err != nil {
		t.Fatalf("write bad file: %v", err)
	}

	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}
	if _, err := store.loadLedger(); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSessionStoreSaveLedgerWriteError(t *testing.T) {
	resetFlags(t)
	badPath := filepath.Join("/", "nonexistent", "forbidden", "sessions.json")
	t.Setenv("ALPINE_LEDGER_PATH", badPath)

	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}
	err = store.saveLedger(&sessionLedger{SchemaVersion: persistSchemaVersion})
	if err == nil {
		t.Fatal("expected save error")
	}
}

func TestSessionHelperFunctions(t *testing.T) {
	t.Run("newID", func(t *testing.T) {
		id, err := newID("op")
		if err != nil {
			t.Fatalf("newID: %v", err)
		}
		if len(id) == 0 || id[:3] != "op_" {
			t.Fatalf("unexpected id %q", id)
		}
	})

	t.Run("currentPrincipalID precedence", func(t *testing.T) {
		t.Setenv("ALPINE_PRINCIPAL_ID", "primary")
		t.Setenv("CF_ACCESS_SUB", "fallback")
		if got := currentPrincipalID(); got != "primary" {
			t.Fatalf("currentPrincipalID = %q, want %q", got, "primary")
		}
		t.Setenv("ALPINE_PRINCIPAL_ID", "")
		if got := currentPrincipalID(); got != "fallback" {
			t.Fatalf("currentPrincipalID = %q, want %q", got, "fallback")
		}
	})

	t.Run("hasAdminOverride", func(t *testing.T) {
		t.Setenv("ALPINE_ADMIN", "1")
		if !hasAdminOverride() {
			t.Fatal("expected ALPINE_ADMIN override")
		}
		t.Setenv("ALPINE_ADMIN", "")
		t.Setenv("ALPINE_ROLES", "viewer,alpine.admin")
		if !hasAdminOverride() {
			t.Fatal("expected role-based override")
		}
	})

	t.Run("normalizeRepoURL", func(t *testing.T) {
		if got := normalizeRepoURL("git@github.com:org/repo.git"); got != "git@github.com:org/repo.git" {
			t.Fatalf("unexpected git@ normalization: %q", got)
		}
		got := normalizeRepoURL("https://token@github.com/org/repo.git")
		if got != "https://github.com/org/repo.git" {
			t.Fatalf("unexpected https normalization: %q", got)
		}
	})
}

func TestSessionTransitionAndLockHelpers(t *testing.T) {
	now := time.Now().UTC()
	session := &sessionRecord{SessionID: "ses-1", State: StateReady}

	if err := appendTransition(session, StateStopping, "actor", "reason", "op-1", now); err != nil {
		t.Fatalf("appendTransition: %v", err)
	}
	if session.State != StateStopping {
		t.Fatalf("state = %q, want %q", session.State, StateStopping)
	}

	if err := appendTransition(session, StateReady, "actor", "reason", "op-1", now); err == nil {
		t.Fatal("expected invalid transition error")
	}

	if err := acquireSessionLock(session, "op-1", now); err != nil {
		t.Fatalf("acquireSessionLock first: %v", err)
	}
	if err := acquireSessionLock(session, "op-2", now); err == nil {
		t.Fatal("expected lock conflict")
	}
	releaseSessionLock(session, "op-1")
	if session.Lock != nil {
		t.Fatal("expected released lock")
	}
}

func TestOperationHistoryHelpers(t *testing.T) {
	now := time.Now().UTC()
	session := &sessionRecord{}
	appendOperation(session, "op-1", "key", "up", now)
	if len(session.OperationHistory) != 1 {
		t.Fatalf("len(operation_history) = %d, want 1", len(session.OperationHistory))
	}
	completeOperation(session, "op-1", StateReady, now)
	if session.OperationHistory[0].CompletedAt == "" {
		t.Fatal("expected completed_at")
	}
	if session.OperationHistory[0].TerminalState != string(StateReady) {
		t.Fatalf("terminal_state = %q, want %q", session.OperationHistory[0].TerminalState, StateReady)
	}
}
