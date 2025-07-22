package workflow

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/core"
	"github.com/maxmcd/river/internal/logger"
	"github.com/maxmcd/river/internal/output"
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
	printer        *output.Printer
}

// NewEngine creates a new workflow engine
func NewEngine(executor ClaudeExecutor, linear LinearClient) *Engine {
	return &Engine{
		claudeExecutor: executor,
		linearClient:   linear,
		stateFile:      "claude_state.json",
		printer:        output.NewPrinter(),
	}
}

// Run executes the main workflow loop
func (e *Engine) Run(ctx context.Context, issueID string, noPlan bool) error {
	logger.WithField("issue_id", issueID).Debug("Starting workflow run")
	
	// Validate input
	if issueID == "" {
		return fmt.Errorf("issue ID cannot be empty")
	}

	// Fetch issue from Linear
	logger.Debug("Fetching issue from Linear")
	progress := e.printer.StartProgress("Fetching issue from Linear")
	issue, err := e.linearClient.FetchIssue(ctx, issueID)
	progress.Stop()
	
	if err != nil {
		logger.WithField("error", err).Error("Failed to fetch Linear issue")
		return fmt.Errorf("failed to fetch Linear issue: %w", err)
	}
	logger.WithFields(map[string]interface{}{
		"issue_id": issue.ID,
		"title": issue.Title,
	}).Debug("Successfully fetched Linear issue")

	// Initialize workflow
	logger.WithField("no_plan", noPlan).Debug("Initializing workflow")
	if err := e.initializeWorkflow(issue, noPlan); err != nil {
		logger.WithField("error", err).Error("Failed to initialize workflow")
		return fmt.Errorf("failed to initialize workflow: %w", err)
	}

	// Main execution loop
	iteration := 0
	for {
		iteration++
		logger.WithField("iteration", iteration).Debug("Starting workflow iteration")
		
		// Check context cancellation
		select {
		case <-ctx.Done():
			logger.Debug("Workflow interrupted by context cancellation")
			return fmt.Errorf("workflow interrupted: %w", ctx.Err())
		default:
		}

		// Load current state
		logger.Debug("Loading current state")
		state, err := core.LoadState(e.stateFile)
		if err != nil {
			logger.WithField("error", err).Error("Failed to load state")
			return fmt.Errorf("failed to load state: %w", err)
		}
		logger.WithFields(map[string]interface{}{
			"status": state.Status,
			"current_step": state.CurrentStepDescription,
		}).Debug("State loaded successfully")

		// Check if workflow is completed
		if state.Status == "completed" {
			logger.Info("Workflow completed successfully")
			e.printer.Success("Workflow completed successfully")
			return nil
		}

		// Execute Claude with the next prompt
		e.printer.Step("Executing Claude with prompt: %s", state.NextStepPrompt)
		logger.WithField("prompt", state.NextStepPrompt).Debug("Executing Claude")
		
		// Show progress indicator during Claude execution
		progress := e.printer.StartProgressWithIteration("Executing Claude", iteration)
		
		config := claude.ExecuteConfig{
			Prompt:    state.NextStepPrompt,
			StateFile: e.stateFile,
		}
		
		startTime := time.Now()
		claudeErr := func() error {
			if _, err := e.claudeExecutor.Execute(ctx, config); err != nil {
				return err
			}
			return nil
		}()
		
		progress.Stop()
		
		if claudeErr != nil {
			logger.WithFields(map[string]interface{}{
				"error": claudeErr,
				"duration": time.Since(startTime),
			}).Error("Claude execution failed")
			return fmt.Errorf("Claude execution failed: %w", claudeErr)
		}
		logger.WithField("duration", time.Since(startTime)).Debug("Claude execution completed")

		// Wait for state file to be updated
		logger.Debug("Waiting for state file update")
		if err := e.waitForStateUpdate(ctx, state); err != nil {
			logger.WithField("error", err).Error("Error waiting for state update")
			return fmt.Errorf("error waiting for state update: %w", err)
		}
		logger.Debug("State file updated, continuing to next iteration")
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

	e.printer.Info("Initializing workflow for Linear issue %s", issue.ID)
	
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

	// Show progress indicator while waiting
	progress := e.printer.StartProgress("Waiting for state file update")
	defer progress.Stop()

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

// SetPrinter allows overriding the printer (mainly for testing)
func (e *Engine) SetPrinter(printer *output.Printer) {
	e.printer = printer
}
