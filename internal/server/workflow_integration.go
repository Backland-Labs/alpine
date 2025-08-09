// Package server provides HTTP server functionality for Alpine, including
// workflow integration and REST API endpoints.
package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
func (e *AlpineWorkflowEngine) StartWorkflow(ctx context.Context, issueURL string, runID string, plan bool) (string, error) {
	logger.Infof("Starting workflow %s for issue: %s", runID, issueURL)

	// Check if workflow already exists (with limited mutex scope)
	e.mu.Lock()
	if _, exists := e.workflows[runID]; exists {
		e.mu.Unlock()
		logger.Infof("Attempted to start duplicate workflow: %s", runID)
		return "", fmt.Errorf("workflow %s already exists", runID)
	}
	e.mu.Unlock()

	// Create workflow context with issue URL
	// Use context.Background() for long-running workflows to avoid premature cancellation
	// when the HTTP request context is cancelled after the handler returns
	workflowCtx, cancel := context.WithCancel(context.Background())
	workflowCtx = context.WithValue(workflowCtx, "issue_url", issueURL)

	// Create custom config for this workflow
	workflowCfg := *e.cfg // Copy config

	// Create workflow instance early so it exists for directory tracking
	instance := &workflowInstance{
		engine:      nil, // Will be set after engine creation
		ctx:         workflowCtx,
		cancel:      cancel,
		events:      make(chan WorkflowEvent, defaultEventChannelSize),
		worktreeDir: "", // Will be set after directory creation
		stateFile:   "", // Will be set after directory creation
		createdAt:   time.Now(),
		clonedDirs:  make([]string, 0),
	}

	// Register instance before directory creation so cleanup tracking works (with mutex)
	e.mu.Lock()
	e.workflows[runID] = instance
	e.mu.Unlock()

	// Create isolated directory for the workflow
	worktreeDir, err := e.createWorkflowDirectory(workflowCtx, runID, cancel)
	if err != nil {
		// Clean up the instance if directory creation fails
		e.mu.Lock()
		delete(e.workflows, runID)
		e.mu.Unlock()
		cancel() // Cancel the context as well
		return "", err
	}

	// Update instance with directory information
	instance.worktreeDir = worktreeDir

	// Update state file path to be in workflow directory
	workflowCfg.StateFile = filepath.Join(worktreeDir, stateFileRelativePath)
	workflowCfg.WorkDir = worktreeDir

	// Disable worktree creation in workflow.Engine since we already created one
	workflowCfg.Git.WorktreeEnabled = false

	logger.WithFields(map[string]interface{}{
		"run_id":       runID,
		"worktree_dir": worktreeDir,
		"state_file":   workflowCfg.StateFile,
		"work_dir":     workflowCfg.WorkDir,
		"operation":    "workflow_config_setup",
	}).Info("Configured workflow with WorkDir for server execution")

	// Create workflow engine with server as streamer if available
	var streamer events.Streamer
	if e.server != nil {
		streamer = NewServerStreamer(e.server)
		logger.Debugf("Created server streamer for workflow %s", runID)
	}

	engine := workflow.NewEngine(e.claudeExecutor, nil, &workflowCfg, streamer)
	engine.SetStateFile(workflowCfg.StateFile)

	// Set up ServerEventEmitter for workflow lifecycle events
	if e.server != nil {
		broadcastFunc := func(eventType, runID string, data map[string]interface{}) {
			event := WorkflowEvent{
				Type:      eventType,
				RunID:     runID,
				Timestamp: time.Now(),
				Source:    "alpine",
				Data:      data,
			}
			logger.WithFields(map[string]interface{}{
				"event_type": eventType,
				"run_id":     runID,
				"source":     "server_event_emitter",
			}).Debug("ServerEventEmitter broadcasting event")
			e.server.BroadcastEvent(event)
		}

		serverEventEmitter := events.NewServerEventEmitter(runID, broadcastFunc)
		engine.SetEventEmitter(serverEventEmitter)
		logger.WithField("run_id", runID).Debug("ServerEventEmitter configured for workflow engine")
	}

	// Update the instance with the engine and state file
	instance.engine = engine
	instance.stateFile = workflowCfg.StateFile

	// CRITICAL FIX: Forward instance events to server's broadcast system
	if e.server != nil {
		go func() {
			logger.WithField("run_id", runID).Debug("Starting event forwarding goroutine")
			for {
				select {
				case event, ok := <-instance.events:
					if !ok {
						logger.WithField("run_id", runID).Debug("Instance events channel closed, stopping event forwarding")
						return
					}
					logger.WithFields(map[string]interface{}{
						"run_id":     runID,
						"event_type": event.Type,
					}).Debug("Forwarding instance event to server broadcast")

					// Forward event to server's broadcast system
					e.server.BroadcastEvent(event)
				case <-workflowCtx.Done():
					logger.WithField("run_id", runID).Debug("Context cancelled, stopping event forwarding")
					return
				}
			}
		}()
	}

	// Start workflow execution in background
	go e.runWorkflowAsync(instance, issueURL, runID, plan)

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

	// Skip worktree creation in Docker environments - use branches instead
	// Worktrees add unnecessary complexity when we already have repository isolation
	logger.WithFields(map[string]interface{}{
		"run_id":    runID,
		"clone_dir": clonedDir,
		"operation": "skip_worktree_use_branch",
	}).Debug("Skipping worktree creation in cloned repository, will use branch instead")

	// Create and publish a new branch for this workflow run
	branchName := fmt.Sprintf("alpine-run-%s", runID)
	logger.WithFields(map[string]interface{}{
		"run_id":      runID,
		"branch_name": branchName,
		"clone_dir":   clonedDir,
		"operation":   "branch_creation_attempt",
	}).Info("Attempting to create and publish branch for workflow")

	if err := e.createAndPublishBranch(ctx, clonedDir, branchName, runID); err != nil {
		logger.WithFields(map[string]interface{}{
			"run_id":      runID,
			"branch_name": branchName,
			"error":       err.Error(),
			"operation":   "branch_creation_failed",
		}).Error("Failed to create/publish branch for workflow - aborting")
		// Cannot continue without a published branch - no way to track changes
		return "", false
	}

	logger.WithFields(map[string]interface{}{
		"run_id":      runID,
		"branch_name": branchName,
		"clone_dir":   clonedDir,
		"operation":   "branch_creation_success",
	}).Info("Successfully created and published branch, using cloned repository for workflow")

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

// createAndPublishBranch creates a new branch in the cloned repository and publishes it to the remote.
// This ensures each workflow run has its own isolated branch for tracking changes.
// Returns an error if the branch cannot be created or published, as there would be no way to track changes.
func (e *AlpineWorkflowEngine) createAndPublishBranch(ctx context.Context, clonedDir, branchName, runID string) error {
	branchLog := logger.WithFields(map[string]interface{}{
		"run_id":      runID,
		"branch_name": branchName,
		"clone_dir":   clonedDir,
		"operation":   "create_and_publish_branch",
	})

	branchLog.Debug("Starting branch creation process for workflow run")

	// Step 1: Create and checkout the new branch
	branchLog.Info("Creating new branch from current HEAD")
	createCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName)
	createCmd.Dir = clonedDir

	createOutput, err := createCmd.CombinedOutput()
	if err != nil {
		branchLog.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"output": string(createOutput),
			"step":   "branch_creation",
		}).Error("Failed to create and checkout new branch")
		return fmt.Errorf("failed to create branch %s: %w (output: %s)", branchName, err, string(createOutput))
	}

	branchLog.WithField("output", string(createOutput)).Info("Successfully created and checked out new branch")

	// Step 2: Configure git user for commits (required for push and commits)
	branchLog.Debug("Configuring git user for server commits")

	configNameCmd := exec.CommandContext(ctx, "git", "config", "user.name", "Alpine Server")
	configNameCmd.Dir = clonedDir
	if output, err := configNameCmd.CombinedOutput(); err != nil {
		branchLog.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
			"step":   "git_config_name",
		}).Warn("Failed to set git user.name, continuing anyway")
	} else {
		branchLog.Debug("Set git user.name to 'Alpine Server'")
	}

	configEmailCmd := exec.CommandContext(ctx, "git", "config", "user.email", "alpine@localhost")
	configEmailCmd.Dir = clonedDir
	if output, err := configEmailCmd.CombinedOutput(); err != nil {
		branchLog.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
			"step":   "git_config_email",
		}).Warn("Failed to set git user.email, continuing anyway")
	} else {
		branchLog.Debug("Set git user.email to 'alpine@localhost'")
	}

	// Step 3: Configure Git to use GitHub token for authentication if available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		branchLog.Error("GITHUB_TOKEN not set - cannot push to remote")
		return fmt.Errorf("GITHUB_TOKEN environment variable not set: cannot push branch to remote")
	}

	// Extract owner/repo from the remote URL
	remoteCmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	remoteCmd.Dir = clonedDir
	remoteOutput, err := remoteCmd.Output()
	if err != nil {
		branchLog.WithField("error", err.Error()).Error("Failed to get remote URL")
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	// Configure Git to use token authentication
	// Set the remote URL to include the token for HTTPS authentication
	remoteURL := strings.TrimSpace(string(remoteOutput))
	if strings.HasPrefix(remoteURL, "https://github.com/") {
		// Replace https://github.com/ with https://TOKEN@github.com/
		authenticatedURL := strings.Replace(remoteURL, "https://github.com/", fmt.Sprintf("https://%s@github.com/", githubToken), 1)

		setRemoteCmd := exec.CommandContext(ctx, "git", "remote", "set-url", "origin", authenticatedURL)
		setRemoteCmd.Dir = clonedDir
		if output, err := setRemoteCmd.CombinedOutput(); err != nil {
			branchLog.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"output": string(output),
			}).Error("Failed to set authenticated remote URL")
			return fmt.Errorf("failed to configure Git authentication: %w", err)
		}
		branchLog.Debug("Configured Git remote with authentication token")
	}

	// Step 4: Publish the branch to remote (push the new branch upstream)
	branchLog.Info("Publishing branch to remote repository")
	pushCmd := exec.CommandContext(ctx, "git", "push", "-u", "origin", branchName)
	pushCmd.Dir = clonedDir

	pushOutput, err := pushCmd.CombinedOutput()
	outputStr := string(pushOutput)

	if err != nil {
		// Check if it's an authentication error
		if strings.Contains(outputStr, "Authentication") ||
			strings.Contains(outputStr, "403") ||
			strings.Contains(outputStr, "401") ||
			strings.Contains(outputStr, "could not read Username") ||
			strings.Contains(outputStr, "terminal prompts disabled") {
			branchLog.WithFields(map[string]interface{}{
				"error":  err.Error(),
				"output": outputStr,
				"step":   "branch_push_auth_failure",
			}).Error("Branch push failed due to authentication - cannot track changes without remote branch")
			return fmt.Errorf("authentication failed when pushing branch %s: cannot proceed without ability to publish changes", branchName)
		}

		// Any other push error is also fatal
		branchLog.WithFields(map[string]interface{}{
			"error":  err.Error(),
			"output": outputStr,
			"step":   "branch_push_failure",
		}).Error("Failed to push branch to remote - cannot track changes")
		return fmt.Errorf("failed to push branch %s to remote: %w (output: %s)", branchName, err, outputStr)
	}

	branchLog.WithFields(map[string]interface{}{
		"branch_name": branchName,
		"output":      outputStr,
		"step":        "branch_push_success",
	}).Info("Successfully published branch to remote repository")

	// Step 4: Verify the branch was created and we're on it
	verifyCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	verifyCmd.Dir = clonedDir

	if verifyOutput, err := verifyCmd.Output(); err == nil {
		currentBranch := strings.TrimSpace(string(verifyOutput))
		if currentBranch != branchName {
			branchLog.WithFields(map[string]interface{}{
				"expected_branch": branchName,
				"current_branch":  currentBranch,
				"step":            "branch_verification",
			}).Error("Branch verification failed - not on expected branch")
			return fmt.Errorf("branch verification failed: expected to be on %s but on %s", branchName, currentBranch)
		}
		branchLog.WithField("current_branch", currentBranch).Debug("Verified current branch is correct")
	}

	branchLog.WithField("branch_name", branchName).Info("Branch creation and publishing completed successfully")
	return nil
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
func (e *AlpineWorkflowEngine) runWorkflowAsync(instance *workflowInstance, issueURL string, runID string, plan bool) {
	defer close(instance.events)

	// Send start event (AG-UI compliant)
	startEvent := WorkflowEvent{
		Type:      events.AGUIEventRunStarted,
		RunID:     runID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task":        fmt.Sprintf("Process GitHub issue: %s", issueURL),
			"worktreeDir": instance.worktreeDir,
			"planMode":    plan,
		},
	}

	logger.WithFields(map[string]interface{}{
		"run_id":     runID,
		"event_type": startEvent.Type,
	}).Debug("Sending start event to instance channel")

	e.sendEventNonBlocking(instance, startEvent)

	// Run the workflow
	logger.Infof("Executing workflow %s", runID)
	err := instance.engine.Run(instance.ctx, issueURL, plan) // Use provided plan parameter

	// Send completion event (AG-UI compliant)
	if err != nil {
		logger.Errorf("Workflow %s failed: %v", runID, err)
		errorEvent := WorkflowEvent{
			Type:      events.AGUIEventRunError,
			RunID:     runID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		}

		logger.WithFields(map[string]interface{}{
			"run_id":     runID,
			"event_type": errorEvent.Type,
		}).Debug("Sending error event to instance channel")

		e.sendEventNonBlocking(instance, errorEvent)
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

		completionEvent := WorkflowEvent{
			Type:      events.AGUIEventRunFinished,
			RunID:     runID,
			Timestamp: time.Now(),
			Data:      result,
		}

		logger.WithFields(map[string]interface{}{
			"run_id":     runID,
			"event_type": completionEvent.Type,
		}).Debug("Sending completion event to instance channel")

		e.sendEventNonBlocking(instance, completionEvent)
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
