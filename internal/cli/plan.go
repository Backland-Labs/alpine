package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/github"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/output"
	"github.com/Backland-Labs/alpine/internal/prompts"
	"github.com/spf13/cobra"
)

// planCmd represents the plan command structure
type planCmd struct {
	cmd *cobra.Command
}

// NewPlanCommand creates a new plan command (exported for tests)
func NewPlanCommand() *cobra.Command {
	return newPlanCmd().Command()
}

// newPlanCmd creates a new plan command
func newPlanCmd() *planCmd {
	pc := &planCmd{}
	var worktreeFlag bool
	var cleanupFlag bool

	pc.cmd = &cobra.Command{
		Use:   "plan <task-description>",
		Short: "Generate an implementation plan using Claude Code",
		Long: `Generate a detailed implementation plan for a given task using Claude Code.
This command reads the project specifications and creates a structured plan
that can be used with Alpine's implementation workflow.`,
		Example: `  # Generate a plan
  alpine plan "Implement user authentication"
  
  # Generate a plan from a GitHub issue
  alpine plan gh-issue https://github.com/owner/repo/issues/123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := args[0]

			// Check if worktree flag is set
			if worktreeFlag {
				return runPlanInWorktree(task, cleanupFlag)
			}

			// Always use Claude Code for plan generation
			return generatePlan(task)
		},
	}

	// Add the --worktree and --cleanup flags
	pc.cmd.Flags().BoolVar(&worktreeFlag, "worktree", false, "Generate the plan in an isolated git worktree")
	pc.cmd.Flags().BoolVar(&cleanupFlag, "cleanup", true, "Automatically clean up (remove) the worktree after plan generation")

	// Add gh-issue subcommand
	pc.cmd.AddCommand(newGhIssueCmd())

	return pc
}

// Command returns the cobra command
func (pc *planCmd) Command() *cobra.Command {
	return pc.cmd
}

// generatePlan generates an implementation plan using Claude Code
func generatePlan(task string) error {
	// Create printer for progress indicator
	printer := output.NewPrinter()

	// Replace placeholders in the prompt template
	prompt := strings.ReplaceAll(prompts.PromptPlan, "{{TASK}}", task)

	// Create a temporary state file (required by executor)
	stateFile, err := os.CreateTemp("", "claude_state_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary state file: %w", err)
	}
	defer func() {
		_ = os.Remove(stateFile.Name()) // Clean up after execution
	}()

	// Write initial state content
	initialState := `{"current_step_description": "Generating plan", "next_step_prompt": "", "status": "running"}`
	if err := os.WriteFile(stateFile.Name(), []byte(initialState), 0644); err != nil {
		return fmt.Errorf("failed to write initial state: %w", err)
	}

	// Create Claude executor
	executor := claude.NewExecutor()

	// Get current working directory for Claude execution
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Configure ExecuteConfig for planning
	config := claude.ExecuteConfig{
		Prompt:    prompt,
		StateFile: stateFile.Name(),
		WorkDir:   workDir,
		// No MCP servers for planning
		MCPServers: []string{},
		// Planning-specific allowed tools (read-only)
		AllowedTools: []string{
			"Read", "Grep", "Glob", "LS",
			"WebSearch", "WebFetch", "mcp__context7__*",
		},
		// Planning-specific system prompt
		SystemPrompt: "You are a senior Technical Product Manager creating implementation plans. " +
			"Focus on understanding the codebase and creating detailed plan.md files. " +
			"Follow TDD principles and Alpine's planning conventions.",
		// 5-minute timeout for plan generation
		Timeout: 5 * time.Minute,
		// Add current directory for codebase context
		AdditionalArgs: []string{"--add-dir", "."},
	}

	// Create context with timeout
	ctx := context.Background()

	// Start progress indicator
	printer.Info("Generating plan using Claude Code...")
	progress := printer.StartProgress("Analyzing codebase and creating plan")
	defer progress.Stop()

	// Execute Claude
	_, err = executor.Execute(ctx, config)

	// Stop progress before printing any messages
	progress.Stop()

	if err != nil {
		// Check for specific error types
		if execErr, ok := err.(*exec.ExitError); ok {
			printer.Error("Claude Code execution failed with exit code %d", execErr.ExitCode())
			return fmt.Errorf("claude Code execution failed with exit code %d", execErr.ExitCode())
		}
		if strings.Contains(err.Error(), "executable file not found") {
			printer.Error("Claude Code CLI not found. Please install from https://claude.ai/download")
			return fmt.Errorf("claude Code CLI not found. Please install from https://claude.ai/download")
		}
		printer.Error("Failed to execute Claude Code: %v", err)
		return fmt.Errorf("failed to execute Claude Code: %w", err)
	}

	printer.Success("Plan generation completed")
	return nil
}



// runPlanInWorktree executes plan generation in an isolated git worktree
func runPlanInWorktree(task string, cleanup bool) error {
	// Create printer for consistent output
	printer := output.NewPrinter()

	// Get current working directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create worktree manager
	ctx := context.Background()
	wtMgr := gitx.NewCLIWorktreeManager(".", "main")

	// Create worktree with sanitized task name
	printer.Info("Creating isolated worktree for plan generation...")
	wt, err := wtMgr.Create(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Ensure we return to original directory and handle cleanup
	defer func() {
		// Always return to original directory
		if err := os.Chdir(originalDir); err != nil {
			printer.Warning("Failed to return to original directory: %v", err)
		}

		// Handle cleanup if requested
		if cleanup {
			printer.Info("Cleaning up worktree...")
			if err := wtMgr.Cleanup(ctx, wt); err != nil {
				printer.Warning("Failed to cleanup worktree: %v", err)
			} else {
				printer.Success("Worktree cleaned up: %s", wt.Path)
			}
		} else {
			printer.Info("Worktree preserved at: %s", wt.Path)
		}
	}()

	// Change to the worktree directory
	if err := os.Chdir(wt.Path); err != nil {
		return fmt.Errorf("failed to change to worktree directory: %w", err)
	}

	printer.Info("Generating plan in worktree: %s", wt.Path)

	// Call the plan generation function
	return generatePlan(task)
}

// validatePlanFile checks if plan.md exists and has content
func validatePlanFile() error {
	// Check if plan.md exists
	info, err := os.Stat("plan.md")
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan.md does not exist")
		}
		// For permission errors or other issues, return the not exist error
		// to maintain consistent behavior
		return fmt.Errorf("plan.md does not exist")
	}

	// Check if plan.md is empty
	if info.Size() == 0 {
		return fmt.Errorf("plan.md is empty")
	}

	return nil
}

// newGhIssueCmd creates a new gh-issue subcommand
func newGhIssueCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gh-issue <url>",
		Short: "Generate a plan from a GitHub issue",
		Long: `Generate an implementation plan by fetching a GitHub issue using the gh CLI.
This command uses the GitHub CLI (gh) to fetch the issue title and body,
then generates a plan based on the combined information.

Example:
  alpine plan gh-issue https://github.com/owner/repo/issues/123

Requirements:
  - The gh CLI must be installed and authenticated
  - You must have access to the specified GitHub issue`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]

			// Create printer for consistent output
			printer := output.NewPrinter()

			// Fetch issue data
			printer.Info("Fetching GitHub issue...")
			task, err := github.FetchIssueDescription(url)
			if err != nil {
				printer.Error("Failed to fetch issue: %v", err)
				return fmt.Errorf("failed to fetch issue: %w", err)
			}

			// Access parent command's flags
			worktreeFlag, _ := cmd.Parent().Flags().GetBool("worktree")
			cleanupFlag, _ := cmd.Parent().Flags().GetBool("cleanup")

			// Check if worktree flag is set
			if worktreeFlag {
				return runPlanInWorktree(task, cleanupFlag)
			}

			// Always use Claude Code for plan generation
			return generatePlan(task)
		},
	}
}
