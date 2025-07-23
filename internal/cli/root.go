package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxmcd/river/internal/output"
	"github.com/spf13/cobra"
)

// Context key types to avoid collisions
type contextKey string

const (
	noPlanKey     contextKey = "noPlan"
	fromFileKey   contextKey = "fromFile"
	noWorktreeKey contextKey = "noWorktree"
)

const version = "0.2.0" // Bumped version for new implementation

// Execute runs the CLI
func Execute() error {
	return NewRootCommand().Execute()
}

// NewRootCommand creates the root command
func NewRootCommand() *cobra.Command {
	var showVersion bool
	var noPlan bool
	var noWorktree bool
	var fromFile string

	cmd := &cobra.Command{
		Use:   "river <task-description>",
		Short: "River - CLI orchestrator for Claude Code",
		Long: `River - CLI orchestrator for Claude Code

River automates iterative AI-assisted development workflows by running
Claude Code in a loop based on a state-driven workflow with your task description.

Examples:
  river "Implement user authentication"
  river "Fix bug in payment processing" --no-plan
  river --file task.md
  river --no-plan --no-worktree              # Bare execution mode`,
		Args: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return nil
			}
			// If --file is provided, we don't need args
			if fromFile != "" {
				return nil
			}
			// Check for bare execution mode (both --no-plan and --no-worktree)
			if len(args) < 1 {
				if noPlan && noWorktree {
					// Bare execution mode allows no arguments
					return nil
				}
				return fmt.Errorf("requires a task description (use quotes for multi-word descriptions)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "river version "+version)
				return err
			}
			// Delegate to run workflow
			return runWorkflow(cmd, args)
		},
	}

	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	cmd.Flags().BoolVar(&noPlan, "no-plan", false, "Skip plan generation and execute directly")
	cmd.Flags().BoolVar(&noWorktree, "no-worktree", false, "Disable git worktree creation")
	cmd.Flags().StringVar(&fromFile, "file", "", "Read task description from a file")

	// Store flags in command context for runWorkflow
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.WithValue(cmd.Context(), noPlanKey, noPlan)
		ctx = context.WithValue(ctx, fromFileKey, fromFile)
		ctx = context.WithValue(ctx, noWorktreeKey, noWorktree)
		cmd.SetContext(ctx)
		return nil
	}

	// Add subcommands (currently none)

	return cmd
}

// runWorkflow executes the main workflow
func runWorkflow(cmd *cobra.Command, args []string) error {
	// Get flags from context
	fromFile, _ := cmd.Context().Value(fromFileKey).(string)
	noPlan, _ := cmd.Context().Value(noPlanKey).(bool)
	noWorktree, _ := cmd.Context().Value(noWorktreeKey).(bool)

	// Create real dependencies for production use
	deps := NewRealDependencies()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		printer := output.NewPrinter()
		printer.Warning("\nInterrupt received, shutting down gracefully...")
		cancel()
	}()

	// Use the testable workflow function (includes logger initialization)
	return runWorkflowWithDependencies(ctx, args, noPlan, noWorktree, fromFile, deps)
}
