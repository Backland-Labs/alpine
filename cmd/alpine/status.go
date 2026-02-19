package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type statusOutput struct {
	SessionID   string         `json:"session_id"`
	State       LifecycleState `json:"state"`
	RepoURL     string         `json:"repo_url"`
	Branch      string         `json:"branch"`
	ReadyAt     string         `json:"ready_at,omitempty"`
	StoppedAt   string         `json:"stopped_at,omitempty"`
	UpdatedAt   string         `json:"updated_at"`
	OperationID string         `json:"operation_id,omitempty"`
	LastError   *sessionError  `json:"last_error,omitempty"`
	NextStep    string         `json:"next_step"`
}

var statusCmd = &cobra.Command{
	Use:   "status <session-id>",
	Short: "Show lifecycle and recovery status for a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	sessionID := strings.TrimSpace(args[0])
	principalID := currentPrincipalID()
	if principalID == "" {
		return newCommandError(1, ErrCallerIdentityRequired, "caller identity is required", "set ALPINE_PRINCIPAL_ID or CF_ACCESS_SUB", false, "")
	}

	store, err := newSessionStore()
	if err != nil {
		return sysErr(fmt.Sprintf("failed to initialize session store: %v", err))
	}

	var out statusOutput
	var opErr error
	err = store.withLedger(func(ledger *sessionLedger) error {
		session, ok := ledger.Sessions[sessionID]
		if !ok {
			opErr = newCommandError(1, ErrSessionNotFound, "session not found", "run alpine list to discover active/recent sessions", false, "")
			return nil
		}

		if session.OwnerPrincipalID != principalID && !hasAdminOverride() {
			opErr = newCommandError(1, ErrSessionForbidden, "session is owned by another principal", "use the owning principal or alpine.admin role", false, "")
			return nil
		}

		operationID := ""
		if len(session.OperationHistory) > 0 {
			operationID = session.OperationHistory[len(session.OperationHistory)-1].OperationID
		}

		out = statusOutput{
			SessionID:   session.SessionID,
			State:       session.State,
			RepoURL:     session.RepoURL,
			Branch:      session.Branch,
			ReadyAt:     session.ReadyAt,
			StoppedAt:   session.StoppedAt,
			UpdatedAt:   session.UpdatedAt,
			OperationID: operationID,
			LastError:   session.LastError,
			NextStep:    recommendedNextStep(session),
		}
		return nil
	})
	if err != nil {
		return sysErr(fmt.Sprintf("failed to read session ledger: %v", err))
	}
	if opErr != nil {
		return opErr
	}

	if jsonOutput {
		return outputJSON(out)
	}

	fmt.Printf("Session:     %s\n", out.SessionID)
	fmt.Printf("State:       %s\n", out.State)
	fmt.Printf("Repository:  %s\n", out.RepoURL)
	fmt.Printf("Branch:      %s\n", out.Branch)
	if out.ReadyAt != "" {
		fmt.Printf("Ready at:    %s\n", out.ReadyAt)
	}
	if out.StoppedAt != "" {
		fmt.Printf("Stopped at:  %s\n", out.StoppedAt)
	}
	fmt.Printf("Updated at:  %s\n", out.UpdatedAt)
	if out.OperationID != "" {
		fmt.Printf("Operation:   %s\n", out.OperationID)
	}
	if out.LastError != nil {
		fmt.Printf("Last error:  %s (%s)\n", out.LastError.Cause, out.LastError.ErrorCode)
	}
	fmt.Printf("Next step:   %s\n", out.NextStep)

	return nil
}

func recommendedNextStep(session *sessionRecord) string {
	switch session.State {
	case StateReady:
		return fmt.Sprintf("run alpine down %s when work is complete", session.SessionID)
	case StateStoppedUnpersisted, StatePersistFailed:
		return fmt.Sprintf("run alpine down %s --retry-persist", session.SessionID)
	case StateFailed, StateCleanupFailed, StateCloseFailed:
		return "review status last_error and run alpine up again with a new client-request-id"
	case StateStopped:
		return "session is fully stopped and persisted"
	default:
		return "wait for the active operation to finish, then run alpine status again"
	}
}
