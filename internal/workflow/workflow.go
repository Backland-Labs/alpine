package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/core"
	"github.com/maxmcd/river/internal/gitx"
	"github.com/maxmcd/river/internal/logger"
	"github.com/maxmcd/river/internal/output"
)

// ClaudeExecutor interface for executing Claude commands
type ClaudeExecutor interface {
	Execute(ctx context.Context, config claude.ExecuteConfig) (string, error)
}

// Engine orchestrates the workflow execution
type Engine struct {
	claudeExecutor ClaudeExecutor
	wtMgr          gitx.WorktreeManager
	cfg            *config.Config
	stateFile      string
	printer        *output.Printer
	wt             *gitx.Worktree // Current worktree if created
	originalDir    string         // Original directory to restore if needed
}

// NewEngine creates a new workflow engine
func NewEngine(executor ClaudeExecutor, wtMgr gitx.WorktreeManager, cfg *config.Config) *Engine {
	return &Engine{
		claudeExecutor: executor,
		wtMgr:          wtMgr,
		cfg:            cfg,
		stateFile:      "claude_state.json",
		printer:        output.NewPrinter(),
	}
}

// Run executes the main workflow loop with a task description
func (e *Engine) Run(ctx context.Context, taskDescription string, generatePlan bool) error {
	logger.WithField("task_description", taskDescription).Debug("Starting workflow run")

	// Check if this is bare mode
	isBareMode := taskDescription == "" && !generatePlan && !e.cfg.Git.WorktreeEnabled

	// Validate input (skip for bare mode)
	if !isBareMode && strings.TrimSpace(taskDescription) == "" {
		return fmt.Errorf("task description cannot be empty")
	}

	// Create worktree if enabled and not in bare mode
	if !isBareMode {
		if err := e.createWorktree(ctx, taskDescription); err != nil {
			logger.WithField("error", err).Error("Failed to create worktree")
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Setup cleanup
	defer func() {
		if e.wt != nil && e.cfg.Git.AutoCleanupWT {
			logger.Debug("Cleaning up worktree")
			if err := e.wtMgr.Cleanup(ctx, e.wt); err != nil {
				logger.WithField("error", err).Error("Failed to cleanup worktree")
				e.printer.Warning("Failed to cleanup worktree: %v", err)
			} else {
				e.printer.Info("Cleaned up worktree: %s", e.wt.Path)
			}
		}
		// Restore original directory if we changed it
		if e.originalDir != "" && e.wt != nil {
			if err := os.Chdir(e.originalDir); err != nil {
				logger.WithField("error", err).Error("Failed to restore original directory")
			}
		}
	}()

	// Handle bare mode initialization
	if isBareMode {
		if _, err := os.Stat(e.stateFile); err == nil {
			// Continue from existing state
			logger.Info("Continuing from existing state file")
			e.printer.Info("Continuing from existing state file")
			return e.runWorkflowLoop(ctx)
		} else if os.IsNotExist(err) {
			// Initialize with /run_implementation_loop
			logger.Info("Starting bare execution with /run_implementation_loop")
			e.printer.Info("Starting bare execution with /run_implementation_loop")
			if err := e.initializeWorkflow(ctx, "/run_implementation_loop", false); err != nil {
				return fmt.Errorf("failed to initialize bare workflow: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check state file: %w", err)
		}
	} else {
		// Normal initialization
		logger.WithField("generate_plan", generatePlan).Debug("Initializing workflow")
		if err := e.initializeWorkflow(ctx, taskDescription, generatePlan); err != nil {
			logger.WithField("error", err).Error("Failed to initialize workflow")
			return fmt.Errorf("failed to initialize workflow: %w", err)
		}
	}

	return e.runWorkflowLoop(ctx)
}

// runWorkflowLoop runs the main execution loop
func (e *Engine) runWorkflowLoop(ctx context.Context) error {
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
			"status":       state.Status,
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
				"error":    claudeErr,
				"duration": time.Since(startTime),
			}).Error("Claude execution failed")
			return fmt.Errorf("claude execution failed: %w", claudeErr)
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
func (e *Engine) initializeWorkflow(ctx context.Context, taskDescription string, generatePlan bool) error {
	var prompt string

	if generatePlan {
		prompt = "/make_plan " + taskDescription
	} else {
		prompt = "/run_implementation_loop " + taskDescription
	}

	// For bare mode, taskDescription is just "/run_implementation_loop"
	if taskDescription == "/run_implementation_loop" {
		prompt = taskDescription
		e.printer.Info("Initializing bare workflow execution")
	} else {
		e.printer.Info("Initializing workflow for task: %s", taskDescription)
	}

	// Ensure the directory exists before saving the state file
	stateDir := filepath.Dir(e.stateFile)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	state := &core.State{
		CurrentStepDescription: "Initializing workflow for task",
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
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

// createWorktree creates a worktree for the task if enabled
func (e *Engine) createWorktree(ctx context.Context, taskDescription string) error {
	if !e.cfg.Git.WorktreeEnabled || e.wtMgr == nil {
		return nil
	}

	// Save original directory to restore later if needed
	var err error
	e.originalDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create worktree
	logger.WithField("task", taskDescription).Debug("Creating worktree")
	e.wt, err = e.wtMgr.Create(ctx, taskDescription)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	logger.WithFields(map[string]interface{}{
		"path":   e.wt.Path,
		"branch": e.wt.Branch,
	}).Debug("Worktree created")

	// Change to worktree directory
	if err := os.Chdir(e.wt.Path); err != nil {
		return fmt.Errorf("failed to change to worktree directory: %w", err)
	}

	// Update state file path to be in worktree
	e.stateFile = "claude_state.json" // Now relative to worktree directory
	logger.WithField("state_file", e.stateFile).Debug("Updated state file path")

	e.printer.Info("Created worktree: %s (branch: %s)", e.wt.Path, e.wt.Branch)
	return nil
}
