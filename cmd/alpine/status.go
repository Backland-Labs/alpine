package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

type statusOutput struct {
	Name           string `json:"name"`
	Container      string `json:"container"`
	State          string `json:"state"` // "running", "stopped", "not_found"
	Branch         string `json:"branch"`
	Created        string `json:"created"`
	ClaudeRunning  bool   `json:"claude_running"`
	ClaudeExitCode *int   `json:"claude_exit_code,omitempty"` // nil if still running
}

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show environment status and Claude process state",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	if err := dockerHealthCheck(ctx, runtime.GOOS); err != nil {
		return err
	}

	// Discover the container
	container, err := discoverContainer(ctx, name)
	if err != nil {
		status := statusOutput{
			Name:  name,
			State: "not_found",
		}
		if jsonOutput {
			return outputJSON(status)
		}
		return fmt.Errorf("environment %q not found. Run 'alpine list' to see active environments", name)
	}

	// Get container state
	state, err := inspectContainer(ctx, container, "{{.State.Status}}")
	if err != nil {
		return fmt.Errorf("failed to inspect container %s: %w", container, err)
	}
	state = strings.TrimSpace(state)

	// Get branch label
	branch, _ := inspectContainer(ctx, container, `{{index .Config.Labels "alpine.branch"}}`)
	branch = strings.TrimSpace(branch)

	// Get created label
	created, _ := inspectContainer(ctx, container, `{{index .Config.Labels "alpine.created"}}`)
	created = strings.TrimSpace(created)

	// Check if Claude is running inside the container
	claudeRunning, claudeExitCode := checkClaudeProcess(ctx, container, state)

	status := statusOutput{
		Name:           name,
		Container:      container,
		State:          state,
		Branch:         branch,
		Created:        created,
		ClaudeRunning:  claudeRunning,
		ClaudeExitCode: claudeExitCode,
	}

	if jsonOutput {
		return outputJSON(status)
	}

	// Human-readable output
	fmt.Printf("Environment: %s\n", status.Name)
	fmt.Printf("Container:   %s\n", status.Container)
	fmt.Printf("State:       %s\n", status.State)
	if status.Branch != "" {
		fmt.Printf("Branch:      %s\n", status.Branch)
	}
	if status.Created != "" {
		fmt.Printf("Created:     %s\n", status.Created)
	}
	if status.ClaudeRunning {
		fmt.Printf("Claude:      running\n")
	} else if status.ClaudeExitCode != nil {
		fmt.Printf("Claude:      exited (code %d)\n", *status.ClaudeExitCode)
	} else {
		fmt.Printf("Claude:      not running\n")
	}

	return nil
}
