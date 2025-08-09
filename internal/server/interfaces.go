package server

import (
	"context"
	"time"

	"github.com/Backland-Labs/alpine/internal/core"
)

// WorkflowEngine interface represents the integration point with Alpine's workflow engine
type WorkflowEngine interface {
	// StartWorkflow initiates a new workflow run with the given GitHub issue URL and plan generation setting.
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - issueURL: GitHub issue URL to process
	//   - runID: Unique identifier for the workflow run
	//   - plan: Whether to generate a plan.md file before implementation (true) or skip directly to implementation (false)
	StartWorkflow(ctx context.Context, issueURL string, runID string, plan bool) (string, error)

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
	Type      string    `json:"type"`
	RunID     string    `json:"runId"`               // Changed to camelCase per AG-UI spec
	MessageID string    `json:"messageId,omitempty"` // For text message correlation
	Timestamp time.Time `json:"timestamp"`

	// AG-UI streaming fields
	Content  string `json:"content,omitempty"`  // Text chunks
	Delta    bool   `json:"delta,omitempty"`    // Incremental content flag
	Source   string `json:"source,omitempty"`   // Agent attribution (e.g., "claude")
	Complete bool   `json:"complete,omitempty"` // Stream completion marker

	// Flexible event data (backward compatibility)
	Data map[string]interface{} `json:"data,omitempty"`
}
