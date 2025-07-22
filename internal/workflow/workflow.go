package workflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/core"
)

// LinearIssue represents a Linear issue
type LinearIssue struct {
	ID          string
	Title       string
	Description string
}

// LinearClient interface for fetching Linear issues
type LinearClient interface {
	FetchIssue(ctx context.Context, issueID string) (*LinearIssue, error)
}

// ClaudeExecutor interface for executing Claude commands
type ClaudeExecutor interface {
	Execute(ctx context.Context, config claude.ExecuteConfig) (string, error)
}

// Engine orchestrates the workflow execution
type Engine struct {
	claudeExecutor ClaudeExecutor
	linearClient   LinearClient
	stateFile      string
}

// NewEngine creates a new workflow engine
func NewEngine(executor ClaudeExecutor, linear LinearClient) *Engine {
	return &Engine{
		claudeExecutor: executor,
		linearClient:   linear,
		stateFile:      "claude_state.json",
	}
}

// Run executes the main workflow loop
func (e *Engine) Run(ctx context.Context, issueID string, noPlan bool) error {
	// Validate input
	if issueID == "" {
		return fmt.Errorf("issue ID cannot be empty")
	}

	// Fetch issue from Linear
	issue, err := e.linearClient.FetchIssue(ctx, issueID)
	if err != nil {
		return fmt.Errorf("failed to fetch Linear issue: %w", err)
	}

	// Initialize workflow
	if err := e.initializeWorkflow(issue, noPlan); err != nil {
		return fmt.Errorf("failed to initialize workflow: %w", err)
	}

	// Main execution loop
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("workflow interrupted: %w", ctx.Err())
		default:
		}

		// Load current state
		state, err := core.LoadState(e.stateFile)
		if err != nil {
			return fmt.Errorf("failed to load state: %w", err)
		}

		// Check if workflow is completed
		if state.Status == "completed" {
			fmt.Println("Workflow completed successfully")
			return nil
		}

		// Execute Claude with the next prompt
		fmt.Printf("Executing Claude with prompt: %s\n", state.NextStepPrompt)
		config := claude.ExecuteConfig{
			Prompt:    state.NextStepPrompt,
			StateFile: e.stateFile,
		}
		if _, err := e.claudeExecutor.Execute(ctx, config); err != nil {
			return fmt.Errorf("Claude execution failed: %w", err)
		}

		// Wait for state file to be updated
		if err := e.waitForStateUpdate(ctx, state); err != nil {
			return fmt.Errorf("error waiting for state update: %w", err)
		}
	}
}

// initializeWorkflow creates the initial state file
func (e *Engine) initializeWorkflow(issue *LinearIssue, noPlan bool) error {
	prompt := fmt.Sprintf("%s\n\n%s", issue.Title, issue.Description)

	if noPlan {
		prompt = "/ralph " + prompt
	} else {
		prompt = "/make_plan " + prompt
	}

	state := &core.State{
		CurrentStepDescription: fmt.Sprintf("Initializing workflow for Linear issue %s", issue.ID),
		NextStepPrompt:         prompt,
		Status:                 "running",
	}

	return state.Save(e.stateFile)
}

// waitForStateUpdate waits for the state file to be updated
func (e *Engine) waitForStateUpdate(ctx context.Context, previousState *core.State) error {
	// Check immediately if state has already been updated (for synchronous updates in tests)
	newState, err := core.LoadState(e.stateFile)
	if err == nil && (newState.CurrentStepDescription != previousState.CurrentStepDescription ||
		newState.NextStepPrompt != previousState.NextStepPrompt ||
		newState.Status != previousState.Status) {
		return nil // State already updated
	}

	// Get initial modification time
	initialStat, err := os.Stat(e.stateFile)
	if err != nil {
		return fmt.Errorf("failed to stat state file: %w", err)
	}
	initialModTime := initialStat.ModTime()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute) // 5 minute timeout for Claude execution

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for state update")
		case <-ticker.C:
			// Check if file has been modified
			stat, err := os.Stat(e.stateFile)
			if err != nil {
				if os.IsNotExist(err) {
					// File was deleted, wait for it to be recreated
					continue
				}
				return fmt.Errorf("failed to stat state file: %w", err)
			}

			if stat.ModTime().After(initialModTime) {
				// File has been modified, load and check if content changed
				newState, err := core.LoadState(e.stateFile)
				if err != nil {
					// File might be in the middle of being written, try again
					continue
				}

				// Check if state actually changed
				if newState.CurrentStepDescription != previousState.CurrentStepDescription ||
					newState.NextStepPrompt != previousState.NextStepPrompt ||
					newState.Status != previousState.Status {
					return nil // State has been updated
				}
			}
		}
	}
}

// SetStateFile allows overriding the state file path (mainly for testing)
func (e *Engine) SetStateFile(path string) {
	e.stateFile = path
}
