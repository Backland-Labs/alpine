package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

var listAllOwners bool

type listOutput struct {
	SessionID string         `json:"session_id"`
	State     LifecycleState `json:"state"`
	RepoURL   string         `json:"repo_url"`
	Branch    string         `json:"branch"`
	UpdatedAt string         `json:"updated_at"`
	Owner     string         `json:"owner_principal_id"`
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List active and recent sessions",
	Aliases: []string{"ls"},
	RunE:    runList,
}

func init() {
	listCmd.Flags().BoolVar(&listAllOwners, "all-owners", false, "show sessions for all owners (requires alpine.admin)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	store, err := newSessionStore()
	if err != nil {
		return sysErr(fmt.Sprintf("failed to initialize session store: %v", err))
	}

	principalID := currentPrincipalID()
	if principalID == "" {
		return newCommandError(1, ErrCallerIdentityRequired, "caller identity is required", "set ALPINE_PRINCIPAL_ID or CF_ACCESS_SUB", false, "")
	}
	if listAllOwners && !hasAdminOverride() {
		return newCommandError(1, ErrSessionForbidden, "all-owner listing requires alpine.admin", "set ALPINE_ADMIN=1 or include alpine.admin in ALPINE_ROLES", false, "")
	}

	rows := make([]*sessionRecord, 0)
	err = store.withLedger(func(ledger *sessionLedger) error {
		now := store.now()
		for _, session := range ledger.Sessions {
			if !listAllOwners && session.OwnerPrincipalID != principalID {
				continue
			}

			updatedAt, parseErr := time.Parse(time.RFC3339, session.UpdatedAt)
			if parseErr != nil {
				continue
			}

			if isTerminalState(session.State) && now.Sub(updatedAt) > recentWindow {
				continue
			}

			rows = append(rows, session)
		}
		sortSessionsByUpdatedDesc(rows)
		return nil
	})
	if err != nil {
		return sysErr(fmt.Sprintf("failed to read session ledger: %v", err))
	}

	out := make([]listOutput, 0, len(rows))
	for _, session := range rows {
		out = append(out, listOutput{
			SessionID: session.SessionID,
			State:     session.State,
			RepoURL:   session.RepoURL,
			Branch:    session.Branch,
			UpdatedAt: session.UpdatedAt,
			Owner:     session.OwnerPrincipalID,
		})
	}

	if jsonOutput {
		return outputJSON(out)
	}

	if len(out) == 0 {
		fmt.Println("No active or recent sessions.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SESSION_ID\tSTATE\tBRANCH\tUPDATED\tOWNER") //nolint:errcheck
	for _, row := range out {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.SessionID, row.State, row.Branch, row.UpdatedAt, row.Owner) //nolint:errcheck
	}
	return w.Flush()
}
