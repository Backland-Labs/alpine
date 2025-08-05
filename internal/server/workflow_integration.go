// Package server provides HTTP server functionality for Alpine, including
// workflow integration and REST API endpoints.
package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/Backland-Labs/alpine/internal/workflow"
)

const (
	// defaultEventChannelSize is the buffer size for workflow event channels
	defaultEventChannelSize = 100

	// worktreeNamePrefix is the prefix for worktree directory names
	worktreeNamePrefix = "run-"

	// tempDirPrefix is the prefix for temporary workflow directories
	tempDirPrefix = "alpine-run-"

	// stateFileRelativePath is the relative path for state files within workflow directories
	stateFileRelativePath = "agent_state/agent_state.json"
)

// AlpineWorkflowEngine is the concrete implementation that wraps Alpine's workflow.Engine
// to provide REST API integration with workflow execution. It manages multiple concurrent
// workflow instances and provides event streaming capabilities.
type AlpineWorkflowEngine struct {
	claudeExecutor workflow.ClaudeExecutor
	wtMgr          gitx.WorktreeManager
	cfg            *config.Config
	server         *Server // Reference to server for streaming support

	// Track active workflows with thread-safe access
	mu        sync.RWMutex
	workflows map[string]*workflowInstance
}

// workflowInstance tracks a single workflow execution with its associated
// resources and event stream.
type workflowInstance struct {
	engine      *workflow.Engine   // The workflow engine instance
	ctx         context.Context    // Workflow-specific context for cancellation
	cancel      context.CancelFunc // Function to cancel the workflow
	events      chan WorkflowEvent // Channel for broadcasting workflow events
	worktreeDir string             // Directory containing workflow files
	stateFile   string             // Path to the workflow state file
	createdAt   time.Time          // Timestamp when the workflow was created
	clonedDirs  []string           // Directories of cloned repositories for cleanup
}

// NewAlpineWorkflowEngine creates a new workflow engine integration.
// It initializes the engine with the provided Claude executor, worktree manager,
// and configuration.
func NewAlpineWorkflowEngine(executor workflow.ClaudeExecutor, wtMgr gitx.WorktreeManager, cfg *config.Config) *AlpineWorkflowEngine {
	logger.Debugf("Creating new Alpine workflow engine")
	return &AlpineWorkflowEngine{
		claudeExecutor: executor,
		wtMgr:          wtMgr,
		cfg:            cfg,
		workflows:      make(map[string]*workflowInstance),
	}
}

// SetServer sets the server reference for streaming support
func (e *AlpineWorkflowEngine) SetServer(server *Server) {
	e.server = server
}

// StartWorkflow initiates a new workflow run with the given GitHub issue URL.
// It creates an isolated environment (worktree or temporary directory) for the workflow
// and starts execution in the background. Returns the workflow directory path.
func (e *AlpineWorkflowEngine) StartWorkflow(ctx context.Context, issueURL string, runID string) (string, error) {
	logger.Infof("Starting workflow %s for issue: %s", runID, issueURL)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if workflow already exists
	if _, exists := e.workflows[runID]; exists {
		logger.Infof("Attempted to start duplicate workflow: %s", runID)
		return "", fmt.Errorf("workflow %s already exists", runID)
	}

	// Create workflow context with issue URL
	workflowCtx, cancel := context.WithCancel(ctx)
	workflowCtx = context.WithValue(workflowCtx, "issue_url", issueURL)

	// Create custom config for this workflow
	workflowCfg := *e.cfg // Copy config

	// Create isolated directory for the workflow
	worktreeDir, err := e.createWorkflowDirectory(workflowCtx, runID, cancel)
	if err != nil {
		return "", err
	}

	// Update state file path to be in workflow directory
	workflowCfg.StateFile = filepath.Join(worktreeDir, stateFileRelativePath)
	workflowCfg.WorkDir = worktreeDir

	// Disable worktree creation in workflow.Engine since we already created one
	workflowCfg.Git.WorktreeEnabled = false

	// Create workflow engine with server as streamer if available
	var streamer events.Streamer
	if e.server != nil {
		streamer = NewServerStreamer(e.server)
		logger.Debugf("Created server streamer for workflow %s", runID)
	}

	engine := workflow.NewEngine(e.claudeExecutor, nil, &workflowCfg, streamer)
	engine.SetStateFile(workflowCfg.StateFile)

	// Create workflow instance
	instance := &workflowInstance{
		engine:      engine,
		ctx:         workflowCtx,
		cancel:      cancel,
		events:      make(chan WorkflowEvent, defaultEventChannelSize),
		worktreeDir: worktreeDir,
		stateFile:   workflowCfg.StateFile,
		createdAt:   time.Now(),
		clonedDirs:  make([]string, 0),
	}

	e.workflows[runID] = instance

	// Start workflow execution in background
	go e.runWorkflowAsync(instance, issueURL, runID)

	logger.Infof("Workflow %s started successfully in directory: %s", runID, worktreeDir)
	return worktreeDir, nil
}

// CancelWorkflow cancels an active workflow run.
// It triggers cancellation through the workflow's context and sends a cancellation event.
func (e *AlpineWorkflowEngine) CancelWorkflow(ctx context.Context, runID string) error {
	logger.Infof("Cancelling workflow: %s", runID)

	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.workflows[runID]
	if !exists {
		logger.Infof("Attempted to cancel non-existent workflow: %s", runID)
		return fmt.Errorf("workflow %s not found", runID)
	}

	// Cancel the workflow context
	instance.cancel()

	// Send cancellation event (non-blocking)
	e.sendEventNonBlocking(instance, WorkflowEvent{
		Type:      "workflow_cancelled",
		RunID:     runID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
	})

	logger.Infof("Workflow %s cancelled successfully", runID)
	return nil
}

// GetWorkflowState returns the current state of a workflow run.
// It reads the state from the workflow's state file.
func (e *AlpineWorkflowEngine) GetWorkflowState(ctx context.Context, runID string) (*core.State, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.workflows[runID]
	if !exists {
		logger.Debugf("Workflow state requested for non-existent workflow: %s", runID)
		return nil, fmt.Errorf("workflow %s not found", runID)
	}

	// Load state from file
	state, err := core.LoadState(instance.stateFile)
	if err != nil {
		logger.Errorf("Failed to load state for workflow %s: %v", runID, err)
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return state, nil
}

// ApprovePlan approves a workflow plan and continues execution.
// It updates the workflow state to trigger the implementation phase.
func (e *AlpineWorkflowEngine) ApprovePlan(ctx context.Context, runID string) error {
	logger.Infof("Approving plan for workflow: %s", runID)

	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.workflows[runID]
	if !exists {
		logger.Infof("Attempted to approve plan for non-existent workflow: %s", runID)
		return fmt.Errorf("workflow %s not found", runID)
	}

	// Update state to continue with implementation
	state, err := core.LoadState(instance.stateFile)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Update state to trigger implementation
	state.CurrentStepDescription = "Plan approved, continuing implementation"
	state.NextStepPrompt = "/run_implementation_loop"
	state.Status = core.StatusRunning

	if err := state.Save(instance.stateFile); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Send plan approved event (non-blocking)
	e.sendEventNonBlocking(instance, WorkflowEvent{
		Type:      "plan_approved",
		RunID:     runID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
	})

	logger.Infof("Plan approved for workflow %s", runID)
	return nil
}

// SubscribeToEvents subscribes to workflow events for a specific run.
// It returns a channel that receives all events from the workflow, including
// the current state as an initial event.
func (e *AlpineWorkflowEngine) SubscribeToEvents(ctx context.Context, runID string) (<-chan WorkflowEvent, error) {
	logger.Debugf("New event subscription for workflow: %s", runID)

	e.mu.RLock()
	defer e.mu.RUnlock()

	instance, exists := e.workflows[runID]
	if !exists {
		logger.Infof("Event subscription requested for non-existent workflow: %s", runID)
		return nil, fmt.Errorf("workflow %s not found", runID)
	}

	// Create a new channel for this subscriber
	subscriber := make(chan WorkflowEvent, defaultEventChannelSize)

	// Forward events from the workflow to the subscriber
	go func() {
		defer close(subscriber)

		for {
			select {
			case event, ok := <-instance.events:
				if !ok {
					return
				}
				select {
				case subscriber <- event:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send current state as an event
	if state, err := e.GetWorkflowState(ctx, runID); err == nil {
		select {
		case subscriber <- WorkflowEvent{
			Type:      "state_changed",
			RunID:     runID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"status":       state.Status,
				"current_step": state.CurrentStepDescription,
			},
		}:
		default:
		}
	}

	return subscriber, nil
}

// Cleanup removes completed workflows from memory and cleans up associated resources.
// It performs comprehensive cleanup including workflow context cancellation, cloned repository
// removal (if enabled), and memory cleanup from the active workflows map.
//
// This method implements the server-side resource cleanup requirements from Task 6,
// ensuring that cloned repositories are properly removed after workflow completion
// to prevent disk space accumulation in long-running server deployments.
//
// Cleanup behavior:
// - Respects ALPINE_GIT_AUTO_CLEANUP configuration setting
// - Handles cleanup failures gracefully without affecting workflow status
// - Provides comprehensive logging for monitoring and debugging
// - Thread-safe through mutex protection
//
// Parameters:
//   - runID: The unique identifier for the workflow to clean up
func (e *AlpineWorkflowEngine) Cleanup(runID string) {
	cleanupStartTime := time.Now()

	logger.WithFields(map[string]interface{}{
		"run_id":    runID,
		"operation": "workflow_cleanup",
	}).Debug("Starting workflow cleanup")

	e.mu.Lock()
	defer e.mu.Unlock()

	instance, exists := e.workflows[runID]
	if !exists {
		logger.WithFields(map[string]interface{}{
			"run_id": runID,
		}).Debug("Workflow cleanup requested for non-existent workflow")
		return
	}

	workflowDuration := time.Since(instance.createdAt)
	clonedDirsCount := len(instance.clonedDirs)

	// Clean up cloned repositories if auto cleanup is enabled and directories exist
	if e.cfg.Git.AutoCleanupWT && clonedDirsCount > 0 {
		e.cleanupClonedRepositories(runID, instance.clonedDirs)
	} else if clonedDirsCount > 0 {
		logger.WithFields(map[string]interface{}{
			"run_id":               runID,
			"cloned_dirs_count":    clonedDirsCount,
			"auto_cleanup_enabled": e.cfg.Git.AutoCleanupWT,
		}).Debug("Skipping cloned repository cleanup (disabled by configuration)")
	}

	// Cancel workflow context and remove from active workflows
	instance.cancel()
	delete(e.workflows, runID)

	cleanupDuration := time.Since(cleanupStartTime)
	logger.WithFields(map[string]interface{}{
		"run_id":            runID,
		"workflow_duration": workflowDuration,
		"cleanup_duration":  cleanupDuration,
		"cloned_dirs_count": clonedDirsCount,
		"auto_cleanup":      e.cfg.Git.AutoCleanupWT,
	}).Info("Workflow cleanup completed")
}

// cleanupClonedRepositories removes cloned repository directories for a workflow.
// It handles cleanup failures gracefully without failing the overall cleanup operation.
//
// This method implements defensive cleanup patterns:
// - Logs comprehensive context for debugging and monitoring
// - Continues cleanup even if individual directories fail to remove
// - Uses structured logging for consistent log analysis
// - Provides detailed progress tracking for multiple directory cleanup
//
// Parameters:
//   - runID: The workflow run identifier for logging context
//   - clonedDirs: Slice of directory paths to be removed
//
// The method respects the principle that cleanup failures should not prevent
// workflow completion, ensuring system stability over strict cleanup guarantees.
func (e *AlpineWorkflowEngine) cleanupClonedRepositories(runID string, clonedDirs []string) {
	cleanupLog := logger.WithFields(map[string]interface{}{
		"run_id":            runID,
		"cloned_dirs_count": len(clonedDirs),
		"operation":         "clone_cleanup",
	})

	cleanupLog.Info("Starting cleanup of cloned repositories")

	successCount := 0
	failureCount := 0

	for i, clonedDir := range clonedDirs {
		dirLog := cleanupLog.WithFields(map[string]interface{}{
			"clone_directory": clonedDir,
			"directory_index": i + 1,
			"total_dirs":      len(clonedDirs),
		})

		dirLog.Debug("Removing cloned repository directory")

		if err := os.RemoveAll(clonedDir); err != nil {
			failureCount++
			// Log error but don't fail the cleanup operation
			dirLog.WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Warn("Failed to remove cloned repository directory, continuing cleanup")
		} else {
			successCount++
			dirLog.Debug("Successfully removed cloned repository directory")
		}
	}

	// Log final cleanup summary
	cleanupLog.WithFields(map[string]interface{}{
		"success_count": successCount,
		"failure_count": failureCount,
		"total_dirs":    len(clonedDirs),
	}).Info("Completed cleanup of cloned repositories")
}

// Helper methods

// createWorkflowDirectory creates an isolated directory for workflow execution.
// It uses a worktree if available and enabled, otherwise creates a temporary directory.
// For GitHub issue URLs, it attempts to clone the repository first and create a worktree within it.
func (e *AlpineWorkflowEngine) createWorkflowDirectory(ctx context.Context, runID string, cancel context.CancelFunc) (string, error) {
	// Try to create worktree in cloned repository for GitHub issues
	if worktreeDir, created := e.tryCreateClonedWorktree(ctx, runID); created {
		return worktreeDir, nil
	}

	// Fallback to regular worktree creation
	return e.createFallbackWorktree(ctx, runID, cancel)
}

// tryCreateClonedWorktree attempts to clone a GitHub repository and create a worktree within it.
// Returns the worktree directory path and a boolean indicating if the operation succeeded.
func (e *AlpineWorkflowEngine) tryCreateClonedWorktree(ctx context.Context, runID string) (string, bool) {
	// Check if context contains a GitHub issue URL for cloning
	issueURL, ok := ctx.Value("issue_url").(string)
	if !ok || issueURL == "" || !isGitHubIssueURL(issueURL) || !e.cfg.Git.Clone.Enabled {
		return "", false
	}

	logger.Infof("Detected GitHub issue URL for workflow %s: %s", runID, issueURL)

	// Parse GitHub URL to extract repository information
	owner, repo, _, err := parseGitHubIssueURL(issueURL)
	if err != nil {
		logger.Warnf("Failed to parse GitHub issue URL %s: %v, falling back to regular worktree", issueURL, err)
		return "", false
	}

	// Clone the repository
	repoURL := buildGitCloneURL(owner, repo)
	clonedDir, err := e.cloneRepositoryWithLogging(ctx, repoURL, runID)
	if err != nil {
		logger.Warnf("Failed to clone repository %s: %v, falling back to regular worktree", repoURL, err)
		return "", false
	}

	// Create worktree in cloned repository if possible
	if worktreeDir, err := e.createWorktreeInClonedRepo(ctx, runID, clonedDir); err == nil {
		return worktreeDir, true
	}

	// Return cloned directory as fallback
	logger.Infof("Using cloned repository directory directly for workflow %s: %s", runID, clonedDir)
	return clonedDir, true
}

// cloneRepositoryWithLogging clones a repository with comprehensive logging and directory tracking.
// This method extends the basic cloneRepository functionality with server-specific requirements:
// - Automatic tracking of cloned directories for cleanup
// - Thread-safe directory tracking through mutex protection
// - Structured logging for monitoring and debugging
//
// The method registers cloned directories with the workflow instance to enable
// automatic cleanup when the workflow completes, preventing disk space accumulation
// in long-running server deployments.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - repoURL: The repository URL to clone
//   - runID: The workflow run identifier for tracking and logging
//
// Returns:
//   - string: Path to the cloned repository directory
//   - error: Any error that occurred during cloning or tracking
func (e *AlpineWorkflowEngine) cloneRepositoryWithLogging(ctx context.Context, repoURL, runID string) (string, error) {
	cloneLog := logger.WithFields(map[string]interface{}{
		"run_id":         runID,
		"repository_url": sanitizeURLForLogging(repoURL),
		"operation":      "server_clone_with_tracking",
	})

	cloneLog.Info("Attempting to clone repository for server workflow")

	clonedDir, err := cloneRepository(ctx, repoURL, &e.cfg.Git.Clone)
	if err != nil {
		cloneLog.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Repository clone failed for server workflow")
		return "", err
	}

	// Track the cloned directory for cleanup (thread-safe)
	e.mu.Lock()
	if instance, exists := e.workflows[runID]; exists {
		instance.clonedDirs = append(instance.clonedDirs, clonedDir)
		cloneLog.WithFields(map[string]interface{}{
			"clone_directory":    clonedDir,
			"tracked_dirs_count": len(instance.clonedDirs),
		}).Debug("Registered cloned directory for cleanup tracking")
	} else {
		cloneLog.Warn("Cannot track cloned directory: workflow instance not found")
	}
	e.mu.Unlock()

	cloneLog.WithFields(map[string]interface{}{
		"clone_directory": clonedDir,
	}).Info("Successfully cloned repository for server workflow")

	return clonedDir, nil
}

// createWorktreeInClonedRepo creates a worktree within a cloned repository.
func (e *AlpineWorkflowEngine) createWorktreeInClonedRepo(ctx context.Context, runID, clonedDir string) (string, error) {
	if e.wtMgr == nil || !e.cfg.Git.WorktreeEnabled {
		return "", fmt.Errorf("worktree manager not available")
	}

	// Create worktree name to indicate clone context
	worktreeName := fmt.Sprintf("cloned-%s%s", worktreeNamePrefix, runID)
	wt, err := e.wtMgr.Create(ctx, worktreeName)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree in cloned repository: %w", err)
	}

	logger.Infof("Created worktree in cloned repository for workflow %s at: %s", runID, wt.Path)
	return wt.Path, nil
}

// createFallbackWorktree creates a regular worktree or temporary directory as fallback.
func (e *AlpineWorkflowEngine) createFallbackWorktree(ctx context.Context, runID string, cancel context.CancelFunc) (string, error) {
	// Try to create regular worktree
	if e.wtMgr != nil && e.cfg.Git.WorktreeEnabled {
		worktreeName := fmt.Sprintf("%s%s", worktreeNamePrefix, runID)
		wt, err := e.wtMgr.Create(ctx, worktreeName)
		if err != nil {
			cancel()
			logger.Errorf("Failed to create worktree for workflow %s: %v", runID, err)
			return "", fmt.Errorf("failed to create worktree: %w", err)
		}
		logger.Infof("Created worktree for workflow %s at: %s", runID, wt.Path)
		return wt.Path, nil
	}

	// Use temporary directory as final fallback
	tempDirName := fmt.Sprintf("%s%s-", tempDirPrefix, runID)
	tempDir, err := os.MkdirTemp("", tempDirName)
	if err != nil {
		cancel()
		logger.Errorf("Failed to create temp directory for workflow %s: %v", runID, err)
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	logger.Infof("Created temporary directory for workflow %s at: %s", runID, tempDir)
	return tempDir, nil
}

// runWorkflowAsync executes the workflow in a goroutine and manages event broadcasting.
func (e *AlpineWorkflowEngine) runWorkflowAsync(instance *workflowInstance, issueURL string, runID string) {
	defer close(instance.events)

	// Send start event (AG-UI compliant)
	instance.events <- WorkflowEvent{
		Type:      events.AGUIEventRunStarted,
		RunID:     runID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task":        fmt.Sprintf("Process GitHub issue: %s", issueURL),
			"worktreeDir": instance.worktreeDir,
			"planMode":    true,
		},
	}

	// Run the workflow
	logger.Infof("Executing workflow %s", runID)
	err := instance.engine.Run(instance.ctx, issueURL, true) // Generate plan by default

	// Send completion event (AG-UI compliant)
	if err != nil {
		logger.Errorf("Workflow %s failed: %v", runID, err)
		instance.events <- WorkflowEvent{
			Type:      events.AGUIEventRunError,
			RunID:     runID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		}
	} else {
		logger.Infof("Workflow %s completed successfully", runID)

		// Load final state to get completion info
		finalState, stateErr := core.LoadState(instance.stateFile)
		result := map[string]interface{}{
			"status": "completed",
		}

		if stateErr == nil && finalState != nil {
			// Add more result data if available
			if finalState.Status == core.StatusCompleted {
				result["status"] = "completed"
			}
		}

		instance.events <- WorkflowEvent{
			Type:      events.AGUIEventRunFinished,
			RunID:     runID,
			Timestamp: time.Now(),
			Data:      result,
		}
	}
}

// sendEventNonBlocking attempts to send an event to the workflow's event channel.
// If the channel is full or closed, the event is dropped and a warning is logged.
func (e *AlpineWorkflowEngine) sendEventNonBlocking(instance *workflowInstance, event WorkflowEvent) {
	select {
	case instance.events <- event:
		logger.Debugf("Event sent for workflow %s: %s", event.RunID, event.Type)
	default:
		// Event channel might be full or closed
		logger.Infof("Failed to send event for workflow %s: channel full or closed", event.RunID)
	}
}
