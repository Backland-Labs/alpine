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
	noPlanKey  contextKey = "noPlan"
	fromFileKey contextKey = "fromFile"
)

const version = "0.2.0" // Bumped version for new implementation

// Execute runs the CLI without Linear dependency
func Execute() error {
	return NewRootCommand().Execute()
}

// NewRootCommand creates the root command without Linear dependency
func NewRootCommand() *cobra.Command {
	var showVersion bool
	var noPlan bool
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
  river --file task.md`,
		Args: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return nil
			}
			// If --file is provided, we don't need args
			if fromFile != "" {
				return nil
			}
			if len(args) < 1 {
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
	cmd.Flags().StringVar(&fromFile, "file", "", "Read task description from a file")

	// Store flags in command context for runWorkflow
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.WithValue(cmd.Context(), noPlanKey, noPlan)
		ctx = context.WithValue(ctx, fromFileKey, fromFile)
		cmd.SetContext(ctx)
		return nil
	}

	// Add subcommands
	// Note: validate command removed as it was Linear-specific

	return cmd
}

// runWorkflow executes the main workflow without Linear dependency
func runWorkflow(cmd *cobra.Command, args []string) error {
	// Get flags from context
	fromFile, _ := cmd.Context().Value(fromFileKey).(string)
	noPlan, _ := cmd.Context().Value(noPlanKey).(bool)

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
	return runWorkflowWithDependencies(ctx, args, noPlan, fromFile, deps)
}