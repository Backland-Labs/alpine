package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	pc.cmd = &cobra.Command{
		Use:   "plan <task-description>",
		Short: "Generate an implementation plan using Gemini CLI",
		Long: `Generate a detailed implementation plan for a given task using Gemini CLI.
This command reads the project specifications and creates a structured plan
that can be used with River's implementation workflow.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := args[0]
			return generatePlan(task)
		},
	}

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

	// Find all spec files
	specsDir := "specs"
	specFiles, err := filepath.Glob(filepath.Join(specsDir, "*.md"))
	if err != nil {
		return fmt.Errorf("failed to find spec files: %w", err)
	}

	if len(specFiles) == 0 {
		return fmt.Errorf("no spec files found in %s", specsDir)
	}

	// Build the prompt with spec file references
	var specRefs []string
	for _, specFile := range specFiles {
		// Use @filename syntax for Gemini to include file contents
		specRefs = append(specRefs, "@"+specFile)
	}

	// Replace placeholders in the prompt template
	prompt := string(promptTemplate)
	prompt = strings.ReplaceAll(prompt, "{{TASK}}", task)

	// Use the prompt as-is without appending specs
	fullPrompt := prompt

	// Execute Gemini CLI
	cmd := exec.Command("gemini", "-p", fullPrompt)

	// Filter environment to remove CI variables that might trigger interactive mode
	env := filterEnvironment(os.Environ())
	cmd.Env = env

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("gemini command failed: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return fmt.Errorf("failed to execute gemini command: %w", err)
	}

	// Write output to plan.md
	err = os.WriteFile("plan.md", output, 0644)
	if err != nil {
		return fmt.Errorf("failed to write plan.md: %w", err)
	}

	fmt.Println("Plan generated successfully and saved to plan.md")
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
