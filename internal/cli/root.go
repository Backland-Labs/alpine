package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/linear"
	"github.com/maxmcd/river/internal/logger"
	"github.com/maxmcd/river/internal/output"
	"github.com/maxmcd/river/internal/workflow"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

// Execute runs the CLI
func Execute() error {
	return NewRootCommand().Execute()
}

// NewRootCommand creates the root command
func NewRootCommand() *cobra.Command {
	var showVersion bool
	var noPlan bool

	cmd := &cobra.Command{
		Use:   "river <issue-id>",
		Short: "River - CLI orchestrator for Claude Code",
		Long: `River - CLI orchestrator for Claude Code

River automates iterative AI-assisted development workflows by fetching
Linear issues and running Claude Code in a loop based on a state-driven workflow.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("requires a Linear issue ID")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), "river version "+version)
				return nil
			}
			// Delegate to run command when we have arguments
			if len(args) > 0 {
				return runWorkflow(cmd, args)
			}
			return fmt.Errorf("requires a Linear issue ID")
		},
	}

	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
	cmd.Flags().BoolVar(&noPlan, "no-plan", false, "Skip plan generation and execute directly")

	// Store noPlan flag in command context for runWorkflow
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.WithValue(cmd.Context(), "noPlan", noPlan)
		cmd.SetContext(ctx)
		return nil
	}

	// Add subcommands
	cmd.AddCommand(NewValidateCommand())

	return cmd
}

// runWorkflow executes the main workflow
func runWorkflow(cmd *cobra.Command, args []string) error {
	issueID := args[0]

	// Validate Linear issue ID format
	if !isValidLinearID(issueID) {
		return fmt.Errorf("invalid Linear issue ID format: %s", issueID)
	}

	// Load configuration
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger based on configuration
	logger.InitializeFromConfig(cfg)
	logger.Debugf("Starting River workflow for issue %s", issueID)

	// Create Claude executor
	executor := claude.NewExecutor()

	// Create Linear client
	linearClient, err := linear.NewWorkflowAdapter(cfg.LinearAPIKey)
	if err != nil {
		return fmt.Errorf("failed to create Linear client: %w", err)
	}

	// Create workflow engine
	engine := workflow.NewEngine(executor, linearClient)

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

	// Get noPlan flag from context
	noPlan, _ := ctx.Value("noPlan").(bool)

	// Run the workflow
	return engine.Run(ctx, issueID, noPlan)
}

