package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Backland-Labs/alpine/internal/output"
	"github.com/spf13/cobra"
)

// Context key types to avoid collisions
type contextKey string

const (
	noPlanKey     contextKey = "noPlan"
	noWorktreeKey contextKey = "noWorktree"
	continueKey   contextKey = "continue"
	serveKey      contextKey = "serve"
	portKey       contextKey = "port"
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
	var continueFlag bool
	var serve bool
	var port int

	cmd := &cobra.Command{
		Use:   "alpine <task-description>",
		Short: "Alpine - CLI orchestrator for Claude Code",
		Long: `Alpine - CLI orchestrator for Claude Code

Alpine automates iterative AI-assisted development workflows by running
Claude Code in a loop based on a state-driven workflow with your task description.

Examples:
  alpine "Implement user authentication"
  alpine "Fix bug in payment processing" --no-plan
  alpine --no-plan --no-worktree              # Bare execution mode (continue from existing state)
  alpine --serve                               # Run HTTP server with SSE support`,
		Args: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return nil
			}
			// If --serve is provided, no task description is needed
			if serve {
				if len(args) > 0 {
					return fmt.Errorf("cannot use --serve with a task description")
				}
				return nil
			}
			// If --continue is provided, check for conflicts
			if continueFlag {
				if len(args) > 0 {
					return fmt.Errorf("cannot use --continue with a task description")
				}
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
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "alpine version "+version)
				return err
			}
			// Delegate to run workflow
			return runWorkflow(cmd, args)
		},
	}

	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	cmd.Flags().BoolVar(&noPlan, "no-plan", false, "Skip plan generation and execute directly")
	cmd.Flags().BoolVar(&noWorktree, "no-worktree", false, "Disable git worktree creation")
	cmd.Flags().BoolVar(&continueFlag, "continue", false, "Continue from existing state (equivalent to --no-plan --no-worktree)")
	cmd.Flags().BoolVar(&serve, "serve", false, "Start HTTP server with Server-Sent Events support")
	cmd.Flags().IntVar(&port, "port", 3001, "HTTP server port (default: 3001)")

	// Store flags in command context for runWorkflow
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// If --continue is set, override noPlan and noWorktree
		if continueFlag {
			noPlan = true
			noWorktree = true
		}

		ctx := context.WithValue(cmd.Context(), noPlanKey, noPlan)
		ctx = context.WithValue(ctx, noWorktreeKey, noWorktree)
		ctx = context.WithValue(ctx, continueKey, continueFlag)
		ctx = context.WithValue(ctx, serveKey, serve)
		ctx = context.WithValue(ctx, portKey, port)
		cmd.SetContext(ctx)
		return nil
	}

	// Add subcommands
	cmd.AddCommand(newMultiCmd().Command())
	cmd.AddCommand(newPlanCmd().Command())
	cmd.AddCommand(newReviewCmd().Command())

	return cmd
}

// runWorkflow executes the main workflow
func runWorkflow(cmd *cobra.Command, args []string) error {
	// Get flags from context
	noPlan, _ := cmd.Context().Value(noPlanKey).(bool)
	noWorktree, _ := cmd.Context().Value(noWorktreeKey).(bool)
	continueFlag, _ := cmd.Context().Value(continueKey).(bool)

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
	return runWorkflowWithDependencies(ctx, args, noPlan, noWorktree, continueFlag, deps)
}
