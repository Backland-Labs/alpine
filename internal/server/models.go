// Package server implements REST API data models for Alpine's HTTP server.
// These models represent agents, workflow runs, and plans used in the API.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Status constants for Run
const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
	StatusFailed    = "failed"
)

// Status constants for Plan
const (
	PlanStatusPending  = "pending"
	PlanStatusApproved = "approved"
	PlanStatusRejected = "rejected"
)

// Validation errors
var (
	ErrEmptyID           = errors.New("ID cannot be empty")
	ErrEmptyName         = errors.New("name cannot be empty")
	ErrEmptyRunID        = errors.New("run ID cannot be empty")
	ErrEmptyContent      = errors.New("content cannot be empty")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// Agent represents an AI agent available for workflow execution.
// Agents are the primary abstraction for different types of automation tasks.
type Agent struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate checks if the Agent has all required fields properly set.
func (a *Agent) Validate() error {
	if a.ID == "" {
		return ErrEmptyID
	}
	if a.Name == "" {
		return ErrEmptyName
	}
	return nil
}

// Run represents a workflow execution instance.
// Each run is associated with an agent and tracks the lifecycle of a workflow.
type Run struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"` // running, completed, cancelled, failed
	Issue       string    `json:"issue"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	WorktreeDir string    `json:"worktree_dir,omitempty"`
}

// Validate checks if the Run has all required fields properly set.
func (r *Run) Validate() error {
	if r.ID == "" {
		return ErrEmptyID
	}
	if r.AgentID == "" {
		return errors.New("agent ID cannot be empty")
	}
	if !r.IsValidStatus() {
		return ErrInvalidStatus
	}
	return nil
}

// IsValidStatus checks if the current status is a valid run status.
func (r *Run) IsValidStatus() bool {
	switch r.Status {
	case StatusRunning, StatusCompleted, StatusCancelled, StatusFailed:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the run can transition from its current status to the target status.
func (r *Run) CanTransitionTo(targetStatus string) bool {
	// Only running status can transition to other states
	if r.Status != StatusRunning {
		return false
	}

	// Running can transition to completed, cancelled, or failed
	switch targetStatus {
	case StatusCompleted, StatusCancelled, StatusFailed:
		return true
	default:
		return false
	}
}

// Plan represents a workflow execution plan that can be approved or rejected.
// Plans are generated for workflows and require user approval before execution.
type Plan struct {
	RunID   string    `json:"run_id"`
	Content string    `json:"content"`
	Status  string    `json:"status"` // pending, approved, rejected
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// Validate checks if the Plan has all required fields properly set.
func (p *Plan) Validate() error {
	if p.RunID == "" {
		return ErrEmptyRunID
	}
	if p.Content == "" {
		return ErrEmptyContent
	}
	if !p.IsValidStatus() {
		return ErrInvalidStatus
	}
	return nil
}

// IsValidStatus checks if the current status is a valid plan status.
func (p *Plan) IsValidStatus() bool {
	switch p.Status {
	case PlanStatusPending, PlanStatusApproved, PlanStatusRejected:
		return true
	default:
		return false
	}
}

// CanTransitionTo checks if the plan can transition from its current status to the target status.
func (p *Plan) CanTransitionTo(targetStatus string) bool {
	// Only pending status can transition to other states
	if p.Status != PlanStatusPending {
		return false
	}

	// Pending can transition to approved or rejected
	switch targetStatus {
	case PlanStatusApproved, PlanStatusRejected:
		return true
	default:
		return false
	}
}

// GenerateID creates a unique identifier for use in runs and other resources.
// It generates a cryptographically secure random hex string prefixed with the resource type.
func GenerateID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(bytes))
}
