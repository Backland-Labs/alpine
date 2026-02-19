package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunUpSuccess(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	upBranch = "main"
	upClientRequestID = "req-1"
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
	t.Setenv("GITHUB_TOKEN", "token")

	mockRun(t, []cmdResult{{stdout: "abc123 refs/heads/main"}})
	mockOpenBrowser(t, true)

	out := captureStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runUp(cmd, []string{"https://github.com/org/repo.git"}); err != nil {
			t.Fatalf("runUp returned error: %v", err)
		}
	})

	var result upOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if result.State != StateReady {
		t.Fatalf("state = %q, want %q", result.State, StateReady)
	}
	if result.SessionID == "" || result.OperationID == "" {
		t.Fatalf("expected session_id and operation_id, got %#v", result)
	}
	if !result.BrowserOpened {
		t.Fatal("expected browser_opened=true")
	}

	store, err := newSessionStore()
	if err != nil {
		t.Fatalf("newSessionStore: %v", err)
	}
	ledger, err := store.loadLedger()
	if err != nil {
		t.Fatalf("loadLedger: %v", err)
	}
	session := ledger.Sessions[result.SessionID]
	if session == nil {
		t.Fatalf("missing session %s", result.SessionID)
	}
	if session.TargetCommitSHA != "abc123" {
		t.Fatalf("target_commit_sha = %q, want %q", session.TargetCommitSHA, "abc123")
	}
}

func TestRunUpRequiresPrincipal(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	upBranch = "main"
	t.Setenv("ALPINE_PRINCIPAL_ID", "")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runUp(cmd, []string{"https://github.com/org/repo.git"})
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
}

func TestRunUpIdempotentByClientRequestID(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	upBranch = "main"
	upClientRequestID = "same-request"
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
	t.Setenv("GITHUB_TOKEN", "token")

	mockRun(t, []cmdResult{{stdout: "abc123 refs/heads/main"}})
	mockOpenBrowser(t, false)

	runOnce := func() upOutput {
		out := captureStdout(t, func() {
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			if err := runUp(cmd, []string{"https://github.com/org/repo.git"}); err != nil {
				t.Fatalf("runUp returned error: %v", err)
			}
		})
		var result upOutput
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse output JSON: %v", err)
		}
		return result
	}

	first := runOnce()
	second := runOnce()

	if first.SessionID != second.SessionID {
		t.Fatalf("session ids differ: %q vs %q", first.SessionID, second.SessionID)
	}
	if first.OperationID != second.OperationID {
		t.Fatalf("operation ids differ: %q vs %q", first.OperationID, second.OperationID)
	}
}

func TestResolveTargetCommitSHABranchNotFound(t *testing.T) {
	resetFlags(t)
	mockRun(t, []cmdResult{{stdout: ""}})
	_, err := resolveTargetCommitSHA(context.Background(), "https://github.com/org/repo.git", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "resolve branch tip failed: branch \"missing\" not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunUpInputValidation(t *testing.T) {
	tests := []struct {
		name      string
		branch    string
		repoURL   string
		public    bool
		setAuth   bool
		wantCode  string
		principal string
	}{
		{name: "invalid branch", branch: "bad branch", repoURL: "https://github.com/org/repo.git", principal: "u1", setAuth: true, wantCode: ErrInvalidBranch},
		{name: "invalid repo", branch: "main", repoURL: "http://insecure/repo", principal: "u1", setAuth: true, wantCode: ErrInvalidRepoURL},
		{name: "missing auth private", branch: "main", repoURL: "https://github.com/org/repo.git", principal: "u1", setAuth: false, wantCode: ErrRepoAuthMissing},
		{name: "missing principal", branch: "main", repoURL: "https://github.com/org/repo.git", principal: "", setAuth: true, wantCode: ErrCallerIdentityRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			setupLedgerPath(t)
			upBranch = tt.branch
			upPublicRepo = tt.public
			t.Setenv("ALPINE_PRINCIPAL_ID", tt.principal)
			if tt.setAuth {
				t.Setenv("GITHUB_TOKEN", "token")
			} else {
				t.Setenv("GITHUB_TOKEN", "")
				t.Setenv("GH_TOKEN", "")
				t.Setenv("SSH_AUTH_SOCK", "")
			}

			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			err := runUp(cmd, []string{tt.repoURL})
			if err == nil {
				t.Fatal("expected error")
			}
			ce, ok := err.(*commandError)
			if !ok {
				t.Fatalf("expected *commandError, got %T", err)
			}
			if ce.errorCode != tt.wantCode {
				t.Fatalf("error_code = %q, want %q", ce.errorCode, tt.wantCode)
			}
		})
	}
}

func TestRunUpPublicRepoWithoutAuth(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	jsonOutput = true
	upBranch = "main"
	upPublicRepo = true
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("SSH_AUTH_SOCK", "")

	mockRun(t, []cmdResult{{stdout: "abc123 refs/heads/main"}})
	mockOpenBrowser(t, false)

	out := captureStdout(t, func() {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		if err := runUp(cmd, []string{"https://github.com/org/repo.git"}); err != nil {
			t.Fatalf("runUp returned error: %v", err)
		}
	})

	var result upOutput
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v", err)
	}
	if result.SessionID == "" {
		t.Fatal("expected session_id")
	}
}

func TestRunUpIdempotencyFingerprintMismatch(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	upBranch = "main"
	upClientRequestID = "same-key"
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
	t.Setenv("GITHUB_TOKEN", "token")

	mockRun(t, []cmdResult{{stdout: "abc123 refs/heads/main"}})
	mockOpenBrowser(t, false)

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	if err := runUp(cmd, []string{"https://github.com/org/repo.git"}); err != nil {
		t.Fatalf("initial runUp failed: %v", err)
	}

	upBranch = "develop"
	err := runUp(cmd, []string{"https://github.com/org/repo.git"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected *commandError, got %T", err)
	}
	if ce.errorCode != ErrOperationConflict {
		t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrOperationConflict)
	}
}

func TestRunUpProvisionFailurePersistsFailedState(t *testing.T) {
	resetFlags(t)
	setupLedgerPath(t)
	upBranch = "main"
	upClientRequestID = "req-fail"
	t.Setenv("ALPINE_PRINCIPAL_ID", "user-1")
	t.Setenv("GITHUB_TOKEN", "token")

	mockRun(t, []cmdResult{errResult("auth failed")})
	mockOpenBrowser(t, false)

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runUp(cmd, []string{"https://github.com/org/repo.git"})
	if err == nil {
		t.Fatal("expected failure")
	}
	ce, ok := err.(*commandError)
	if !ok {
		t.Fatalf("expected *commandError, got %T", err)
	}
	if ce.errorCode != ErrSandboxProvisionFailed {
		t.Fatalf("error_code = %q, want %q", ce.errorCode, ErrSandboxProvisionFailed)
	}

	store, storeErr := newSessionStore()
	if storeErr != nil {
		t.Fatalf("newSessionStore: %v", storeErr)
	}
	ledger, loadErr := store.loadLedger()
	if loadErr != nil {
		t.Fatalf("loadLedger: %v", loadErr)
	}
	if len(ledger.Sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(ledger.Sessions))
	}
	for _, session := range ledger.Sessions {
		if session.State != StateFailed {
			t.Fatalf("session state = %q, want %q", session.State, StateFailed)
		}
	}
}

func TestHasGitAuth(t *testing.T) {
	resetFlags(t)
	t.Setenv("SSH_AUTH_SOCK", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	if hasGitAuth() {
		t.Fatal("expected false")
	}
	t.Setenv("GH_TOKEN", "x")
	if !hasGitAuth() {
		t.Fatal("expected true")
	}
}

func TestRepoAndBranchValidators(t *testing.T) {
	if !isValidRepoURL("https://github.com/org/repo.git") {
		t.Fatal("expected valid https URL")
	}
	if !isValidRepoURL("git@github.com:org/repo.git") {
		t.Fatal("expected valid ssh URL")
	}
	if isValidRepoURL("http://github.com/org/repo.git") {
		t.Fatal("expected invalid http URL")
	}

	if !isValidBranch("feature/add-auth") {
		t.Fatal("expected valid branch")
	}
	if isValidBranch("feature bad") {
		t.Fatal("expected invalid branch with space")
	}
	if isValidBranch("-bad") {
		t.Fatal("expected invalid branch with leading dash")
	}
}

func TestResolveTargetCommitSHAFailures(t *testing.T) {
	resetFlags(t)
	t.Run("command failure is sanitized", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("fatal: auth failed")})
		_, err := resolveTargetCommitSHA(context.Background(), "https://github.com/org/repo.git", "main")
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "resolve branch tip failed" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unexpected output", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "\n"}})
		_, err := resolveTargetCommitSHA(context.Background(), "https://github.com/org/repo.git", "main")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
