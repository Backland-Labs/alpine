package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/maxkrieger/river/internal/claude"
)

// Runner orchestrates the execution of Linear issue processing using Claude
type Runner struct {
	claude claude.Claude
}

// NewRunner creates a new Runner instance with the provided Claude executor
func NewRunner(claudeExecutor claude.Claude) *Runner {
	return &Runner{
		claude: claudeExecutor,
	}
}

// Run executes the Claude workflow for processing a Linear issue
func (r *Runner) Run(ctx context.Context, issueID, workingDir string) error {
	// Validate inputs
	if issueID == "" {
		return errors.New("issue ID cannot be empty")
	}
	if workingDir == "" {
		return errors.New("working directory cannot be empty")
	}

	// Create the Claude command for processing the Linear issue
	cmd := claude.Command{
		Type:         claude.CommandTypePlan,
		Content:      fmt.Sprintf("Process Linear issue %s following TDD methodology", issueID),
		OutputFormat: "json",
		AllowedTools: []string{"linear-server", "code-editing"},
		SystemPrompt: "You are an AI assistant helping to implement features from Linear issues using Test-Driven Development. Follow the red-green-refactor cycle strictly.",
	}

	// Set up command options
	opts := claude.CommandOptions{
		Stream:     false,
		WorkingDir: workingDir,
	}

	// Execute the command
	_, err := r.claude.Execute(ctx, cmd, opts)
	if err != nil {
		return err
	}

	return nil
}