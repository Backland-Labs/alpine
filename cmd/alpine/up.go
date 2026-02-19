package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	upBranch          string
	upClientRequestID string
	upPublicRepo      bool
	upNoBrowser       bool
)

type upOutput struct {
	SessionID       string         `json:"session_id"`
	OperationID     string         `json:"operation_id"`
	State           LifecycleState `json:"state"`
	OpencodeURL     string         `json:"opencode_url"`
	ReadyAt         string         `json:"ready_at"`
	BrowserOpened   bool           `json:"browser_opened"`
	TargetCommitSHA string         `json:"target_commit_sha"`
}

var upCmd = &cobra.Command{
	Use:   "up <git-repo>",
	Short: "Provision a cloud sandbox development session",
	Args:  cobra.ExactArgs(1),
	RunE:  runUp,
}

func init() {
	upCmd.Flags().StringVar(&upBranch, "branch", "", "repository branch to provision")
	upCmd.Flags().StringVar(&upClientRequestID, "client-request-id", "", "idempotency key for retries")
	upCmd.Flags().BoolVar(&upPublicRepo, "public", false, "treat repository as public (skip credential preflight)")
	upCmd.Flags().BoolVar(&upNoBrowser, "no-browser", false, "do not auto-open Opencode URL")
	upCmd.MarkFlagRequired("branch") //nolint:errcheck
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if upBranch == "" {
		return newCommandError(1, ErrBranchRequired, "branch is required", "use --branch <branch>", false, "")
	}
	if !isValidBranch(upBranch) {
		return newCommandError(1, ErrInvalidBranch, "invalid branch name", "use a valid git branch name", false, "")
	}

	repoURL := strings.TrimSpace(args[0])
	if !isValidRepoURL(repoURL) {
		return newCommandError(1, ErrInvalidRepoURL, "invalid git repository URL", "use an HTTPS or SSH git URL", false, "")
	}

	principalID := currentPrincipalID()
	if principalID == "" {
		return newCommandError(1, ErrCallerIdentityRequired, "caller identity is required", "set ALPINE_PRINCIPAL_ID or CF_ACCESS_SUB", false, "")
	}

	if !upPublicRepo && !hasGitAuth() {
		return newCommandError(1, ErrRepoAuthMissing, "repository credentials are missing", "set SSH_AUTH_SOCK, GITHUB_TOKEN, or GH_TOKEN, or pass --public", false, "")
	}

	store, err := newSessionStore()
	if err != nil {
		return sysErr(fmt.Sprintf("failed to initialize session store: %v", err))
	}

	requestID := strings.TrimSpace(upClientRequestID)
	if requestID == "" {
		generated, idErr := newID("req")
		if idErr != nil {
			return sysErr(fmt.Sprintf("failed to generate client request id: %v", idErr))
		}
		requestID = generated
	}

	idempotencyKey := fmt.Sprintf("up:%s:%s", principalID, requestID)
	requestFingerprint := fmt.Sprintf("%s|%s|%t", normalizeRepoURL(repoURL), upBranch, upPublicRepo)

	var result upOutput
	var opErr error
	err = store.withLedger(func(ledger *sessionLedger) error {
		if existing, ok := ledger.Idempotency[idempotencyKey]; ok && existing.Action == "up" {
			if existing.RequestFingerprint != "" && existing.RequestFingerprint != requestFingerprint {
				opErr = newCommandError(1, ErrOperationConflict, "client-request-id was reused with different request parameters", "use a new --client-request-id for different repo/branch inputs", false, existing.OperationID)
				return nil
			}
			session, ok := ledger.Sessions[existing.SessionID]
			if !ok {
				opErr = newCommandError(2, ErrSandboxProvisionFailed, "idempotent operation exists but session record is missing", "retry with a new client-request-id", true, existing.OperationID)
				return nil
			}
			result = upOutput{
				SessionID:       session.SessionID,
				OperationID:     existing.OperationID,
				State:           session.State,
				OpencodeURL:     session.OpencodeURL,
				ReadyAt:         session.ReadyAt,
				BrowserOpened:   session.BrowserOpened,
				TargetCommitSHA: session.TargetCommitSHA,
			}
			return nil
		}

		now := store.now()
		sessionID, idErr := newID("ses")
		if idErr != nil {
			opErr = sysErr(fmt.Sprintf("failed to generate session id: %v", idErr))
			return nil
		}
		operationID, idErr := newID("op")
		if idErr != nil {
			opErr = sysErr(fmt.Sprintf("failed to generate operation id: %v", idErr))
			return nil
		}

		session := &sessionRecord{
			SchemaVersion:    persistSchemaVersion,
			SessionID:        sessionID,
			PrincipalID:      principalID,
			OwnerPrincipalID: principalID,
			RepoURL:          normalizeRepoURL(repoURL),
			Branch:           upBranch,
			StartedAt:        now.Format(time.RFC3339),
			UpdatedAt:        now.Format(time.RFC3339),
		}

		if err := appendTransition(session, StateRequested, principalID, "operation requested", operationID, now); err != nil {
			opErr = sysErr(err.Error())
			return nil
		}
		appendOperation(session, operationID, idempotencyKey, "up", now)

		if err := acquireSessionLock(session, operationID, now); err != nil {
			opErr = err
			return nil
		}

		if err := appendTransition(session, StateProvisioning, principalID, "sandbox provisioning started", operationID, now); err != nil {
			opErr = sysErr(err.Error())
			return nil
		}

		if err := appendTransition(session, StateRepoSyncing, principalID, "repository sync started", operationID, now); err != nil {
			opErr = sysErr(err.Error())
			return nil
		}

		targetCommitSHA, shaErr := resolveTargetCommitSHA(ctx, repoURL, upBranch)
		if shaErr != nil {
			_ = appendTransition(session, StateFailing, principalID, "setup failed", operationID, store.now())
			_ = appendTransition(session, StateFailed, principalID, "operation failed", operationID, store.now())
			session.LastError = &sessionError{
				ErrorCode:   ErrSandboxProvisionFailed,
				Cause:       shaErr.Error(),
				NextStep:    "verify repository access and retry with the same --client-request-id",
				Retryable:   true,
				OperationID: operationID,
			}
			completeOperation(session, operationID, session.State, store.now())
			releaseSessionLock(session, operationID)
			ledger.Sessions[sessionID] = session
			ledger.Idempotency[idempotencyKey] = idempotencyRecord{
				OperationID:        operationID,
				SessionID:          sessionID,
				Action:             "up",
				RequestFingerprint: requestFingerprint,
			}
			opErr = newCommandError(2, ErrSandboxProvisionFailed, shaErr.Error(), "verify repository access and retry", true, operationID)
			return nil
		}

		session.TargetCommitSHA = targetCommitSHA

		if err := appendTransition(session, StateInstalling, principalID, "dependency install started", operationID, store.now()); err != nil {
			opErr = sysErr(err.Error())
			return nil
		}

		if err := appendTransition(session, StateReady, principalID, "session ready", operationID, store.now()); err != nil {
			opErr = sysErr(err.Error())
			return nil
		}

		session.ReadyAt = store.now().Format(time.RFC3339)
		session.OpencodeURL = opencodeURLForSession(session.SessionID)
		session.BrowserOpened = false
		if !upNoBrowser {
			session.BrowserOpened = openBrowser(ctx, session.OpencodeURL, runtime.GOOS)
		}

		completeOperation(session, operationID, StateReady, store.now())
		releaseSessionLock(session, operationID)

		ledger.Sessions[sessionID] = session
		ledger.Idempotency[idempotencyKey] = idempotencyRecord{
			OperationID:        operationID,
			SessionID:          sessionID,
			Action:             "up",
			RequestFingerprint: requestFingerprint,
		}

		result = upOutput{
			SessionID:       sessionID,
			OperationID:     operationID,
			State:           StateReady,
			OpencodeURL:     session.OpencodeURL,
			ReadyAt:         session.ReadyAt,
			BrowserOpened:   session.BrowserOpened,
			TargetCommitSHA: session.TargetCommitSHA,
		}
		return nil
	})
	if err != nil {
		return sysErr(fmt.Sprintf("failed to persist session state: %v", err))
	}
	if opErr != nil {
		return opErr
	}

	if jsonOutput {
		return outputJSON(result)
	}

	fmt.Printf("Session:        %s\n", result.SessionID)
	fmt.Printf("Operation:      %s\n", result.OperationID)
	fmt.Printf("State:          %s\n", result.State)
	fmt.Printf("Target commit:  %s\n", result.TargetCommitSHA)
	fmt.Printf("Opencode URL:   %s\n", result.OpencodeURL)
	fmt.Printf("Ready at:       %s\n", result.ReadyAt)
	if result.BrowserOpened {
		fmt.Printf("Browser:        opened\n")
	} else {
		fmt.Printf("Browser:        not opened (URL provided above)\n")
	}
	return nil
}

func hasGitAuth() bool {
	if strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) != "" {
		return true
	}
	if strings.TrimSpace(os.Getenv("GH_TOKEN")) != "" {
		return true
	}
	return false
}

func isValidRepoURL(repoURL string) bool {
	if strings.HasPrefix(repoURL, "git@") {
		return strings.Contains(repoURL, ":")
	}
	if strings.HasPrefix(repoURL, "https://") {
		return strings.Contains(repoURL, "/")
	}
	return false
}

func isValidBranch(branch string) bool {
	if branch == "" || strings.HasPrefix(branch, "-") || strings.Contains(branch, " ") {
		return false
	}
	if strings.Contains(branch, "..") || strings.Contains(branch, "~") || strings.Contains(branch, "^") {
		return false
	}
	if strings.Contains(branch, ":") || strings.Contains(branch, "?") || strings.Contains(branch, "*") || strings.Contains(branch, "[") || strings.Contains(branch, "\\") {
		return false
	}
	if strings.HasSuffix(branch, "/") || strings.HasSuffix(branch, ".") {
		return false
	}
	return true
}

func resolveTargetCommitSHA(ctx context.Context, repoURL, branch string) (string, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	stdout, stderr, err := run(lookupCtx, "git", "ls-remote", repoURL, "refs/heads/"+branch)
	if err != nil {
		_ = stderr
		_ = err
		return "", fmt.Errorf("resolve branch tip failed")
	}

	line := strings.TrimSpace(stdout)
	if line == "" {
		return "", fmt.Errorf("resolve branch tip failed: branch %q not found", branch)
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", fmt.Errorf("resolve branch tip failed: unexpected output")
	}
	return fields[0], nil
}

func opencodeURLForSession(sessionID string) string {
	return fmt.Sprintf("https://opencode.local/sessions/%s", sessionID)
}
