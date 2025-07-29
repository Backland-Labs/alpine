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
	
	// Track active workflows with thread-safe access
	mu        sync.RWMutex
	workflows map[string]*workflowInstance
}

// workflowInstance tracks a single workflow execution with its associated
// resources and event stream.
type workflowInstance struct {
	engine      *workflow.Engine    // The workflow engine instance
	ctx         context.Context      // Workflow-specific context for cancellation
	cancel      context.CancelFunc   // Function to cancel the workflow
	events      chan WorkflowEvent   // Channel for broadcasting workflow events
	worktreeDir string               // Directory containing workflow files
	stateFile   string               // Path to the workflow state file
	createdAt   time.Time           // Timestamp when the workflow was created
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
	
	// Create workflow context
	workflowCtx, cancel := context.WithCancel(ctx)
	
	// Create custom config for this workflow
	workflowCfg := *e.cfg // Copy config
	
	// Create isolated directory for the workflow
	worktreeDir, err := e.createWorkflowDirectory(workflowCtx, runID, cancel)
	if err != nil {
		return "", err
	}
	
	// Update state file path to be in workflow directory
	workflowCfg.StateFile = filepath.Join(worktreeDir, stateFileRelativePath)
	
	// Create workflow engine
	engine := workflow.NewEngine(e.claudeExecutor, e.wtMgr, &workflowCfg)
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

// Cleanup removes completed workflows from memory.
// It cancels the workflow context and removes it from the active workflows map.
func (e *AlpineWorkflowEngine) Cleanup(runID string) {
	logger.Debugf("Cleaning up workflow: %s", runID)
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	if instance, exists := e.workflows[runID]; exists {
		instance.cancel()
		delete(e.workflows, runID)
		logger.Infof("Workflow %s cleaned up after %.2f minutes", runID, time.Since(instance.createdAt).Minutes())
	}
}

// Helper methods

// createWorkflowDirectory creates an isolated directory for workflow execution.
// It uses a worktree if available and enabled, otherwise creates a temporary directory.
func (e *AlpineWorkflowEngine) createWorkflowDirectory(ctx context.Context, runID string, cancel context.CancelFunc) (string, error) {
	if e.wtMgr != nil && e.cfg.Git.WorktreeEnabled {
		// Create worktree for the workflow
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
	
	// Use temporary directory for state
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
	
	// Send start event
	instance.events <- WorkflowEvent{
		Type:      "workflow_started",
		RunID:     runID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"issue": issueURL,
			"worktree_dir": instance.worktreeDir,
		},
	}
	
	// Run the workflow
	logger.Infof("Executing workflow %s", runID)
	err := instance.engine.Run(instance.ctx, issueURL, true) // Generate plan by default
	
	// Send completion event
	if err != nil {
		logger.Errorf("Workflow %s failed: %v", runID, err)
		instance.events <- WorkflowEvent{
			Type:      "workflow_failed",
			RunID:     runID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		}
	} else {
		logger.Infof("Workflow %s completed successfully", runID)
		instance.events <- WorkflowEvent{
			Type:      "workflow_completed",
			RunID:     runID,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{},
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