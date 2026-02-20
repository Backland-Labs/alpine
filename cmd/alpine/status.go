package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type statusIdentity struct {
	Repo         string `json:"repo"`
	ImageProfile string `json:"image_profile"`
}

type statusDurability struct {
	CheckpointID string `json:"checkpoint_id,omitempty"`
	Verified     bool   `json:"verified"`
	ContentHash  string `json:"content_hash,omitempty"`
}

type statusRuntime struct {
	WebURL     string `json:"web_url"`
	ProbeState string `json:"probe_state"`
	ProbeAt    string `json:"probe_at"`
	ProbeStale bool   `json:"probe_stale"`
}

type statusOperation struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type statusTeardown struct {
	AutoTeardown bool     `json:"auto_teardown"`
	Blockers     []string `json:"blockers,omitempty"`
}

type statusOutput struct {
	Name         string           `json:"name"`
	State        lifecycleState   `json:"state"`
	Identity     statusIdentity   `json:"identity"`
	Durability   statusDurability `json:"durability"`
	Runtime      statusRuntime    `json:"runtime"`
	Operation    statusOperation  `json:"operation"`
	Teardown     statusTeardown   `json:"teardown"`
	LastActivity string           `json:"last_activity"`
	ErrorReason  string           `json:"error_reason,omitempty"`
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show sandbox lifecycle and durability status",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(_ *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	orch := newOrchestrator(cfg)
	rec, ok, stale, err := orch.status(args[0])
	if err != nil {
		return sysErr(fmt.Sprintf("failed to read sandbox status: %v", err))
	}
	if !ok {
		return userErrReason(fmt.Sprintf("sandbox %q not found", args[0]), "sandbox_not_found")
	}

	out := statusOutput{
		Name:  rec.Identity.Name,
		State: rec.State,
		Identity: statusIdentity{
			Repo:         rec.Identity.Repo,
			ImageProfile: rec.Identity.ImageProfile,
		},
		Runtime: statusRuntime{
			WebURL:     rec.WebURL,
			ProbeState: rec.RuntimeProbe.Status,
			ProbeAt:    rec.RuntimeProbe.CheckedAt,
			ProbeStale: stale,
		},
		Operation: statusOperation{
			ID:        rec.OperationLock.ID,
			Type:      rec.OperationLock.Type,
			StartedAt: rec.OperationLock.StartedAt,
			ExpiresAt: rec.OperationLock.ExpiresAt,
		},
		Teardown: statusTeardown{
			AutoTeardown: cfg.Sandbox.AutoTeardown,
			Blockers:     rec.TeardownBlockers,
		},
		LastActivity: rec.LastActivityAt,
		ErrorReason:  rec.ErrorReason,
	}

	if rec.Checkpoint != nil {
		out.Durability.CheckpointID = rec.Checkpoint.Manifest.CheckpointID
		out.Durability.Verified = rec.Checkpoint.Verified
		out.Durability.ContentHash = rec.Checkpoint.Manifest.ContentHash
	}

	if jsonOutput {
		return outputJSON(out)
	}

	fmt.Printf("Sandbox:       %s\n", out.Name)
	fmt.Printf("State:         %s\n", out.State)
	fmt.Printf("Repo:          %s\n", out.Identity.Repo)
	fmt.Printf("Image profile: %s\n", out.Identity.ImageProfile)
	fmt.Printf("Runtime probe: %s (checked %s)\n", out.Runtime.ProbeState, out.Runtime.ProbeAt)
	if out.Runtime.ProbeStale {
		fmt.Printf("Freshness:     stale\n")
	}
	if out.Durability.CheckpointID != "" {
		fmt.Printf("Checkpoint:    %s (verified=%t)\n", out.Durability.CheckpointID, out.Durability.Verified)
	}
	if len(out.Teardown.Blockers) > 0 {
		fmt.Printf("Blockers:      %s\n", strings.Join(out.Teardown.Blockers, ", "))
	}
	if out.ErrorReason != "" {
		fmt.Printf("Error reason:  %s\n", out.ErrorReason)
	}

	return nil
}
