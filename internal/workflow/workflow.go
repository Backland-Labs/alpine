package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/github"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/Backland-Labs/alpine/internal/output"
	"github.com/Backland-Labs/alpine/internal/prompts"
)

// ClaudeExecutor interface for executing Claude commands
type ClaudeExecutor interface {
	Execute(ctx context.Context, config claude.ExecuteConfig) (string, error)
}

// StreamingExecutor is an optional interface that executors can implement to support streaming
type StreamingExecutor interface {
	SetStreamer(streamer events.Streamer)
	SetRunID(runID string)
}

// Engine orchestrates the workflow execution
type Engine struct {
	claudeExecutor ClaudeExecutor
	wtMgr          gitx.WorktreeManager
	cfg            *config.Config
	stateFile      string
	printer        *output.Printer
	wt             *gitx.Worktree      // Current worktree if created
	originalDir    string              // Original directory to restore if needed
	eventEmitter   events.EventEmitter // Optional event emitter for lifecycle events
	runID          string              // Unique identifier for this run
	taskDesc       string              // Task description for event tracking
	streamer       events.Streamer     // Optional streamer for real-time output
}

// NewEngine creates a new workflow engine
func NewEngine(executor ClaudeExecutor, wtMgr gitx.WorktreeManager, cfg *config.Config, streamer events.Streamer) *Engine {
	return &Engine{
		claudeExecutor: executor,
		wtMgr:          wtMgr,
		cfg:            cfg,
		stateFile:      cfg.StateFile,
		printer:        output.NewPrinter(),
		streamer:       streamer,
	}
}

// Run executes the main workflow loop with a task description
func (e *Engine) Run(ctx context.Context, taskDescription string, generatePlan bool) (runErr error) {
	logger.WithFields(map[string]interface{}{
		"task_description": taskDescription,
		"generate_plan":    generatePlan,
		"state_file":       e.stateFile,
	}).Debug("Starting workflow run")

	// Generate a unique run ID for event tracking
	e.runID = uuid.New().String()
	e.taskDesc = taskDescription

	logger.WithFields(map[string]interface{}{
		"run_id":           e.runID,
		"task_description": taskDescription,
	}).Info("Workflow run initialized")

	// Emit RunStarted event if emitter is configured
	if e.eventEmitter != nil {
		logger.WithField("run_id", e.runID).Debug("Emitting RunStarted event")
		e.eventEmitter.RunStarted(e.runID, taskDescription)
	}

	// Ensure we emit RunError on any error return using named return value
	defer func() {
		if runErr != nil && e.eventEmitter != nil {
			e.eventEmitter.RunError(e.runID, e.taskDesc, runErr)
		}
	}()

	// Ensure agent_state directory exists
	stateDir := filepath.Dir(e.stateFile)
	logger.WithFields(map[string]interface{}{
		"directory":  stateDir,
		"state_file": e.stateFile,
	}).Debug("Ensuring agent_state directory exists")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":     err.Error(),
			"directory": stateDir,
		}).Error("Failed to create agent_state directory")
		return fmt.Errorf("failed to create agent_state directory: %w", err)
	}
	logger.WithField("directory", stateDir).Debug("Agent_state directory verified")

	// Check if this is bare mode
	isBareMode := taskDescription == "" && !generatePlan && !e.cfg.Git.WorktreeEnabled
	logger.WithFields(map[string]interface{}{
		"is_bare_mode":     isBareMode,
		"task_empty":       taskDescription == "",
		"generate_plan":    generatePlan,
		"worktree_enabled": e.cfg.Git.WorktreeEnabled,
	}).Debug("Determined execution mode")

	// Validate input (skip for bare mode)
	if !isBareMode && strings.TrimSpace(taskDescription) == "" {
		logger.Error("Task description validation failed: empty task description")
		return fmt.Errorf("task description cannot be empty")
	}

	// Create worktree if enabled and not in bare mode
	if !isBareMode {
		logger.WithField("task_description", taskDescription).Debug("Attempting to create worktree")
		if err := e.createWorktree(ctx, taskDescription); err != nil {
			logger.WithFields(map[string]interface{}{
				"error":            err.Error(),
				"task_description": taskDescription,
			}).Error("Failed to create worktree")
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		logger.Debug("Skipping worktree creation in bare mode")
	}

	// Setup cleanup
	defer func() {
		if e.wt != nil && e.cfg.Git.AutoCleanupWT {
			logger.WithFields(map[string]interface{}{
				"worktree_path": e.wt.Path,
				"branch":        e.wt.Branch,
			}).Debug("Cleaning up worktree")
			if err := e.wtMgr.Cleanup(ctx, e.wt); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":         err.Error(),
					"worktree_path": e.wt.Path,
				}).Error("Failed to cleanup worktree")
				e.printer.Warning("Failed to cleanup worktree: %v", err)
			} else {
				logger.WithField("worktree_path", e.wt.Path).Info("Successfully cleaned up worktree")
				e.printer.Info("Cleaned up worktree: %s", e.wt.Path)
			}
		}
		// Restore original directory if we changed it
		if e.originalDir != "" && e.wt != nil {
			logger.WithFields(map[string]interface{}{
				"original_dir": e.originalDir,
				"current_dir":  e.wt.Path,
			}).Debug("Restoring original directory")
			if err := os.Chdir(e.originalDir); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":        err.Error(),
					"original_dir": e.originalDir,
				}).Error("Failed to restore original directory")
			} else {
				logger.WithField("original_dir", e.originalDir).Debug("Successfully restored original directory")
			}
		}
	}()

	// Handle bare mode initialization
	if isBareMode {
		logger.WithField("state_file", e.stateFile).Debug("Checking for existing state file in bare mode")
		if _, err := os.Stat(e.stateFile); err == nil {
			// Continue from existing state
			logger.WithField("state_file", e.stateFile).Info("Continuing from existing state file")
			e.printer.Info("Continuing from existing state file")
			return e.runWorkflowLoop(ctx)
		} else if os.IsNotExist(err) {
			// Initialize with /start
			logger.WithField("state_file", e.stateFile).Info("Starting bare execution with /start")
			e.printer.Info("Starting bare execution with /start")
			if err := e.initializeWorkflow(ctx, "/start", false); err != nil {
				logger.WithFields(map[string]interface{}{
					"error":   err.Error(),
					"command": "/start",
				}).Error("Failed to initialize bare workflow")
				return fmt.Errorf("failed to initialize bare workflow: %w", err)
			}
		} else {
			logger.WithFields(map[string]interface{}{
				"error":      err.Error(),
				"state_file": e.stateFile,
			}).Error("Failed to check state file")
			return fmt.Errorf("failed to check state file: %w", err)
		}
	} else {
		// Normal initialization
		logger.WithFields(map[string]interface{}{
			"generate_plan":    generatePlan,
			"task_description": taskDescription,
			"state_file":       e.stateFile,
		}).Debug("Initializing workflow with task")
		if err := e.initializeWorkflow(ctx, taskDescription, generatePlan); err != nil {
			logger.WithFields(map[string]interface{}{
				"error":            err.Error(),
				"task_description": taskDescription,
				"generate_plan":    generatePlan,
			}).Error("Failed to initialize workflow")
			return fmt.Errorf("failed to initialize workflow: %w", err)
		}
	}

	logger.WithField("run_id", e.runID).Info("Starting workflow execution loop")
	return e.runWorkflowLoop(ctx)
}

// runWorkflowLoop runs the main execution loop
func (e *Engine) runWorkflowLoop(ctx context.Context) error {
	// Main execution loop
	iteration := 0
	for {
		iteration++
		logger.WithFields(map[string]interface{}{
			"iteration": iteration,
			"run_id":    e.runID,
		}).Debug("Starting workflow iteration")

		// Check context cancellation
		select {
		case <-ctx.Done():
			logger.WithFields(map[string]interface{}{
				"run_id":    e.runID,
				"iteration": iteration,
				"error":     ctx.Err().Error(),
			}).Warn("Workflow interrupted by context cancellation")
			return fmt.Errorf("workflow interrupted: %w", ctx.Err())
		default:
		}

		// Load current state
		logger.WithFields(map[string]interface{}{
			"state_file": e.stateFile,
			"iteration":  iteration,
		}).Debug("Loading current state")
		state, err := core.LoadState(e.stateFile)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"error":      err.Error(),
				"state_file": e.stateFile,
				"iteration":  iteration,
			}).Error("Failed to load state")
			return fmt.Errorf("failed to load state: %w", err)
		}
		logger.WithFields(map[string]interface{}{
			"status":       state.Status,
			"current_step": state.CurrentStepDescription,
		}).Debug("State loaded successfully")

		// Check if workflow is completed
		if state.Status == "completed" {
			logger.WithFields(map[string]interface{}{
				"run_id":     e.runID,
				"iterations": iteration,
				"final_step": state.CurrentStepDescription,
			}).Info("Workflow completed successfully")
			e.printer.Success("Workflow completed successfully")

			// Emit RunFinished event
			if e.eventEmitter != nil {
				logger.WithField("run_id", e.runID).Debug("Emitting RunFinished event")
				e.eventEmitter.RunFinished(e.runID, e.taskDesc)
			}

			// Clean up state file on successful completion
			logger.WithField("state_file", e.stateFile).Debug("Cleaning up state file")
			if err := os.Remove(e.stateFile); err != nil && !os.IsNotExist(err) {
				logger.WithFields(map[string]interface{}{
					"error":      err.Error(),
					"state_file": e.stateFile,
				}).Info("Failed to clean up state file")
				// Don't fail the workflow for cleanup issues
			} else {
				logger.WithField("state_file", e.stateFile).Debug("Successfully cleaned up state file")
			}

			return nil
		}

		// Execute Claude with the next prompt
		e.printer.Step("Executing Claude with prompt: %s", state.NextStepPrompt)
		logger.WithFields(map[string]interface{}{
			"prompt":       state.NextStepPrompt,
			"iteration":    iteration,
			"run_id":       e.runID,
			"current_step": state.CurrentStepDescription,
		}).Info("Executing Claude command")

		// Show progress indicator during Claude execution
		progress := e.printer.StartProgressWithIteration("Executing Claude", iteration)

		// Pass streamer and runID to executor if available
		if e.streamer != nil && e.claudeExecutor != nil {
			// Check if executor supports streaming (interface assertion)
			if exec, ok := e.claudeExecutor.(StreamingExecutor); ok {
				logger.WithField("run_id", e.runID).Debug("Setting up streaming for Claude executor")
				exec.SetStreamer(e.streamer)
				exec.SetRunID(e.runID)
			}
		}

		config := claude.ExecuteConfig{
			Prompt:    state.NextStepPrompt,
			StateFile: e.stateFile,
			WorkDir:   e.cfg.WorkDir,
		}

		logger.WithFields(map[string]interface{}{
			"prompt":     config.Prompt,
			"state_file": config.StateFile,
			"work_dir":   config.WorkDir,
			"run_id":     e.runID,
			"iteration":  iteration,
			"operation":  "workflow_claude_config",
		}).Info("Passing WorkDir to Claude executor")

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
				"error":       claudeErr.Error(),
				"duration":    time.Since(startTime).String(),
				"duration_ms": time.Since(startTime).Milliseconds(),
				"iteration":   iteration,
				"run_id":      e.runID,
			}).Error("Claude execution failed")
			return fmt.Errorf("claude execution failed: %w", claudeErr)
		}
		logger.WithFields(map[string]interface{}{
			"duration":    time.Since(startTime).String(),
			"duration_ms": time.Since(startTime).Milliseconds(),
			"iteration":   iteration,
			"run_id":      e.runID,
		}).Info("Claude execution completed successfully")

		// Wait for state file to be updated
		logger.WithFields(map[string]interface{}{
			"state_file": e.stateFile,
			"iteration":  iteration,
		}).Debug("Waiting for state file update")
		if err := e.waitForStateUpdate(ctx, state); err != nil {
			logger.WithFields(map[string]interface{}{
				"error":      err.Error(),
				"state_file": e.stateFile,
				"iteration":  iteration,
			}).Error("Error waiting for state update")
			return fmt.Errorf("error waiting for state update: %w", err)
		}
		logger.WithField("iteration", iteration).Debug("State file updated, continuing to next iteration")
	}
}

// initializeWorkflow creates the initial state file
func (e *Engine) initializeWorkflow(ctx context.Context, taskDescription string, generatePlan bool) error {
	var prompt string

	if generatePlan {
		// Check if taskDescription is a GitHub issue URL and fetch the description
		var taskText string
		if github.IsGitHubIssueURL(taskDescription) {
			logger.WithField("github_url", taskDescription).Info("Detected GitHub issue URL, fetching description")

			description, err := github.FetchIssueDescription(taskDescription)
			if err != nil {
				logger.WithFields(map[string]interface{}{
					"github_url": taskDescription,
					"error":      err.Error(),
				}).Warn("Failed to fetch GitHub issue description, falling back to URL")
				taskText = taskDescription
			} else {
				logger.WithField("github_url", taskDescription).Info("Successfully fetched GitHub issue description")
				taskText = description
			}
		} else {
			taskText = taskDescription
		}

		// Use the embedded prompt template and replace {{TASK}} with the task description
		prompt = strings.ReplaceAll(prompts.PromptPlan, "{{TASK}}", taskText)
	} else {
		prompt = "/start " + taskDescription
	}

	logger.WithFields(map[string]interface{}{
		"task_description": taskDescription,
		"generate_plan":    generatePlan,
		"prompt":           prompt,
		"run_id":           e.runID,
	}).Debug("Preparing initial workflow state")

	// For bare mode, taskDescription is just "/start"
	if taskDescription == "/start" {
		prompt = taskDescription
		logger.WithField("command", prompt).Info("Initializing bare workflow execution")
		e.printer.Info("Initializing bare workflow execution")
	} else {
		logger.WithFields(map[string]interface{}{
			"task":    taskDescription,
			"command": prompt,
		}).Info("Initializing workflow for task")
		e.printer.Info("Initializing workflow for task: %s", taskDescription)
	}

	// Ensure the directory exists before saving the state file
	stateDir := filepath.Dir(e.stateFile)
	logger.WithField("directory", stateDir).Debug("Ensuring state directory exists")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":     err.Error(),
			"directory": stateDir,
		}).Error("Failed to create state directory")
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	state := &core.State{
		CurrentStepDescription: "Initializing workflow for task",
		NextStepPrompt:         prompt,
		Status:                 "running",
	}

	logger.WithFields(map[string]interface{}{
		"state_file": e.stateFile,
		"status":     state.Status,
		"prompt":     state.NextStepPrompt,
	}).Debug("Saving initial state")

	if err := state.Save(e.stateFile); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"state_file": e.stateFile,
		}).Error("Failed to save initial state")
		return err
	}

	logger.WithField("state_file", e.stateFile).Info("Initial workflow state created successfully")
	return nil
}

// waitForStateUpdate waits for the state file to be updated
func (e *Engine) waitForStateUpdate(ctx context.Context, previousState *core.State) error {
	// Check immediately if state has already been updated (for synchronous updates in tests)
	logger.WithField("state_file", e.stateFile).Debug("Checking for immediate state update")
	newState, err := core.LoadState(e.stateFile)
	if err == nil && (newState.CurrentStepDescription != previousState.CurrentStepDescription ||
		newState.NextStepPrompt != previousState.NextStepPrompt ||
		newState.Status != previousState.Status) {
		logger.WithFields(map[string]interface{}{
			"old_status": previousState.Status,
			"new_status": newState.Status,
			"old_step":   previousState.CurrentStepDescription,
			"new_step":   newState.CurrentStepDescription,
		}).Debug("State already updated")
		return nil // State already updated
	}

	// Get initial modification time
	initialStat, err := os.Stat(e.stateFile)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error":      err.Error(),
			"state_file": e.stateFile,
		}).Error("Failed to stat state file")
		return fmt.Errorf("failed to stat state file: %w", err)
	}
	initialModTime := initialStat.ModTime()
	logger.WithFields(map[string]interface{}{
		"state_file": e.stateFile,
		"mod_time":   initialModTime.Format(time.RFC3339),
	}).Debug("Got initial state file modification time")

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
					logger.WithField("state_file", e.stateFile).Debug("State file deleted, waiting for recreation")
					continue
				}
				logger.WithFields(map[string]interface{}{
					"error":      err.Error(),
					"state_file": e.stateFile,
				}).Error("Failed to stat state file during wait")
				return fmt.Errorf("failed to stat state file: %w", err)
			}

			if stat.ModTime().After(initialModTime) {
				logger.WithFields(map[string]interface{}{
					"old_mod_time": initialModTime.Format(time.RFC3339),
					"new_mod_time": stat.ModTime().Format(time.RFC3339),
				}).Debug("State file modification detected")

				// File has been modified, load and check if content changed
				newState, err := core.LoadState(e.stateFile)
				if err != nil {
					// File might be in the middle of being written, try again
					logger.WithField("error", err.Error()).Debug("State file read error, likely mid-write")
					continue
				}

				// Check if state actually changed
				if newState.CurrentStepDescription != previousState.CurrentStepDescription ||
					newState.NextStepPrompt != previousState.NextStepPrompt ||
					newState.Status != previousState.Status {
					logger.WithFields(map[string]interface{}{
						"old_status":    previousState.Status,
						"new_status":    newState.Status,
						"state_changed": true,
					}).Info("State file content changed")
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

// SetEventEmitter allows setting an event emitter (mainly for HTTP server mode)
func (e *Engine) SetEventEmitter(emitter events.EventEmitter) {
	e.eventEmitter = emitter
}

// createWorktree creates a worktree for the task if enabled
func (e *Engine) createWorktree(ctx context.Context, taskDescription string) error {
	if !e.cfg.Git.WorktreeEnabled || e.wtMgr == nil {
		logger.WithFields(map[string]interface{}{
			"worktree_enabled": e.cfg.Git.WorktreeEnabled,
			"wtmgr_nil":        e.wtMgr == nil,
		}).Debug("Worktree creation skipped")
		return nil
	}

	// Save original directory to restore later if needed
	var err error
	e.originalDir, err = os.Getwd()
	if err != nil {
		logger.WithField("error", err.Error()).Error("Failed to get current directory")
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	logger.WithField("original_dir", e.originalDir).Debug("Saved original directory")

	// Create worktree
	logger.WithFields(map[string]interface{}{
		"task":   taskDescription,
		"run_id": e.runID,
	}).Info("Creating worktree for task")
	e.wt, err = e.wtMgr.Create(ctx, taskDescription)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
			"task":  taskDescription,
		}).Error("Failed to create worktree")
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	logger.WithFields(map[string]interface{}{
		"path":   e.wt.Path,
		"branch": e.wt.Branch,
		"run_id": e.runID,
	}).Info("Worktree created successfully")

	// Change to worktree directory
	logger.WithFields(map[string]interface{}{
		"from": e.originalDir,
		"to":   e.wt.Path,
	}).Debug("Changing to worktree directory")
	if err := os.Chdir(e.wt.Path); err != nil {
		logger.WithFields(map[string]interface{}{
			"error":         err.Error(),
			"worktree_path": e.wt.Path,
		}).Error("Failed to change to worktree directory")
		return fmt.Errorf("failed to change to worktree directory: %w", err)
	}

	// State file path remains constant
	// The working directory has changed to the worktree, but the state file
	// path remains relative to that directory
	logger.WithFields(map[string]interface{}{
		"state_file":    e.stateFile,
		"worktree_path": e.wt.Path,
		"working_dir":   e.wt.Path,
	}).Debug("State file path configured in worktree")

	e.printer.Info("Created worktree: %s (branch: %s)", e.wt.Path, e.wt.Branch)
	return nil
}
