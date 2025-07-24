package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/maxmcd/river/internal/claude"
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
	var ccFlag bool

	pc.cmd = &cobra.Command{
		Use:   "plan <task-description>",
		Short: "Generate an implementation plan using Gemini CLI or Claude Code",
		Long: `Generate a detailed implementation plan for a given task using Gemini CLI (default) or Claude Code.
This command reads the project specifications and creates a structured plan
that can be used with River's implementation workflow.

By default, the plan is generated using Gemini. Use the --cc flag to generate
the plan using Claude Code instead.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := args[0]

			// Route based on --cc flag
			if ccFlag {
				// Log that we're using Claude Code
				fmt.Println("Generating plan using Claude Code...")
				return generatePlanWithClaude(task)
			} else {
				// Default to Gemini (existing behavior)
				return generatePlan(task)
			}
		},
	}

	// Add the --cc flag
	pc.cmd.Flags().BoolVar(&ccFlag, "cc", false, "Use Claude Code instead of Gemini for plan generation")

	return pc
}

// Command returns the cobra command
func (pc *planCmd) Command() *cobra.Command {
	return pc.cmd
}

// generatePlan generates an implementation plan using Gemini CLI
func generatePlan(task string) error {
	// Notify user that plan generation is starting
	fmt.Println("Generating plan...")

	// Check if GEMINI_API_KEY is set
	if os.Getenv("GEMINI_API_KEY") == "" {
		return fmt.Errorf("GEMINI_API_KEY not set")
	}

	// Read the prompt template
	promptPath := filepath.Join("prompts", "prompt-plan.md")
	promptTemplate, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt template: %w", err)
	}

	// Replace placeholders in the prompt template
	prompt := string(promptTemplate)
	prompt = strings.ReplaceAll(prompt, "{{TASK}}", task)

	// Execute Gemini CLI in non-interactive mode
	cmd := exec.Command("gemini", "--all-files", "-y", "-p", prompt)

	// Filter environment to remove CI variables that might trigger interactive mode
	env := filterEnvironment(os.Environ())
	cmd.Env = env

	// Let Gemini output directly to stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute the command
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("gemini command failed with exit code %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute gemini command: %w", err)
	}

	fmt.Println("\nPlan generation completed.")
	return nil
}

// generatePlanWithClaude generates an implementation plan using Claude Code
func generatePlanWithClaude(task string) error {
	// Read the prompt template
	promptPath := filepath.Join("prompts", "prompt-plan.md")
	promptTemplate, err := os.ReadFile(promptPath)
	if err != nil {
		return fmt.Errorf("failed to read prompt template: %w", err)
	}

	// Replace placeholders in the prompt template
	prompt := string(promptTemplate)
	prompt = strings.ReplaceAll(prompt, "{{TASK}}", task)

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

	// Configure ExecuteConfig for planning
	config := claude.ExecuteConfig{
		Prompt:    prompt,
		StateFile: stateFile.Name(),
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
			"Follow TDD principles and River's planning conventions.",
		// 5-minute timeout for plan generation
		Timeout: 5 * time.Minute,
		// Add current directory for codebase context
		AdditionalArgs: []string{"--add-dir", "."},
	}

	// Create context with timeout
	ctx := context.Background()

	// Execute Claude
	fmt.Println("Generating plan using Claude Code...")
	_, err = executor.Execute(ctx, config)
	if err != nil {
		// Check for specific error types
		if execErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("claude Code execution failed with exit code %d", execErr.ExitCode())
		}
		if strings.Contains(err.Error(), "executable file not found") {
			return fmt.Errorf("claude Code CLI not found. Please install from https://claude.ai/download")
		}
		return fmt.Errorf("failed to execute Claude Code: %w", err)
	}

	fmt.Println("\nPlan generation completed.")
	return nil
}

// filterEnvironment removes CI-related environment variables
func filterEnvironment(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "CI_") && !strings.HasPrefix(e, "CI=") {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
