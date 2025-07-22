package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/logger"
	"github.com/maxmcd/river/internal/output"
	"github.com/maxmcd/river/internal/workflow"
	"github.com/spf13/cobra"
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
				fmt.Fprintln(cmd.OutOrStdout(), "river version "+version)
				return nil
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
		ctx := context.WithValue(cmd.Context(), "noPlan", noPlan)
		ctx = context.WithValue(ctx, "fromFile", fromFile)
		cmd.SetContext(ctx)
		return nil
	}

	// Add subcommands
	// Note: validate command removed as it was Linear-specific

	return cmd
}

// runWorkflow executes the main workflow without Linear dependency
func runWorkflow(cmd *cobra.Command, args []string) error {
	var taskDescription string

	// Get task description from file or command line
	fromFile, _ := cmd.Context().Value("fromFile").(string)
	if fromFile != "" {
		content, err := os.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to read task file: %w", err)
		}
		taskDescription = string(content)
	} else {
		if len(args) == 0 {
			return fmt.Errorf("task description is required")
		}
		taskDescription = args[0]
	}

	// Validate task description
	if taskDescription == "" {
		return fmt.Errorf("task description cannot be empty")
	}

	// Load configuration (without Linear API key requirement)
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger based on configuration
	logger.InitializeFromConfig(cfg)
	logger.Debugf("Starting River workflow for task: %s", taskDescription)

	// Create Claude executor
	executor := claude.NewExecutor()

	// Create workflow engine without Linear client
	engine := workflow.NewEngine(executor)

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
	return engine.Run(ctx, taskDescription, !noPlan)
}