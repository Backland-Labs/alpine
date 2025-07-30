package server

import (
	"context"
	"time"

	"github.com/Backland-Labs/alpine/internal/core"
)

// WorkflowEngine interface represents the integration point with Alpine's workflow engine
type WorkflowEngine interface {
	// StartWorkflow initiates a new workflow run with the given GitHub issue URL
	StartWorkflow(ctx context.Context, issueURL string, runID string) (string, error)
	
	// CancelWorkflow cancels an active workflow run
	CancelWorkflow(ctx context.Context, runID string) error
	
	// GetWorkflowState returns the current state of a workflow run
	GetWorkflowState(ctx context.Context, runID string) (*core.State, error)
	
	// ApprovePlan approves a workflow plan and continues execution
	ApprovePlan(ctx context.Context, runID string) error
	
	// SubscribeToEvents subscribes to workflow events for a specific run
	SubscribeToEvents(ctx context.Context, runID string) (<-chan WorkflowEvent, error)
}

// WorkflowEvent represents an event emitted during workflow execution
type WorkflowEvent struct {
	Type      string                 `json:"type"`
	RunID     string                 `json:"run_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}