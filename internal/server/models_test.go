package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestAgentValidation tests the Agent struct validation and JSON marshaling.
// This ensures that agents have proper IDs, names, and descriptions,
// and can be correctly serialized for REST API responses.
func TestAgentValidation(t *testing.T) {
	tests := []struct {
		name    string
		agent   Agent
		wantErr bool
	}{
		{
			name: "valid agent",
			agent: Agent{
				ID:          "alpine-agent",
				Name:        "Alpine Agent",
				Description: "AI-powered development automation agent",
			},
			wantErr: false,
		},
		{
			name: "agent with empty ID",
			agent: Agent{
				ID:          "",
				Name:        "Alpine Agent",
				Description: "AI-powered development automation agent",
			},
			wantErr: true,
		},
		{
			name: "agent with empty name",
			agent: Agent{
				ID:          "alpine-agent",
				Name:        "",
				Description: "AI-powered development automation agent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Test JSON marshaling
			data, err := json.Marshal(tt.agent)
			if err != nil {
				t.Fatalf("Failed to marshal agent: %v", err)
			}

			var unmarshaled Agent
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal agent: %v", err)
			}

			if unmarshaled.ID != tt.agent.ID {
				t.Errorf("Agent ID mismatch: got %v, want %v", unmarshaled.ID, tt.agent.ID)
			}
		})
	}
}

// TestRunLifecycle tests the Run struct status transitions and validation.
// This ensures that runs follow the expected lifecycle: running -> completed/cancelled/failed,
// and that all required fields are properly validated.
func TestRunLifecycle(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		run         Run
		wantErr     bool
		validStatus bool
	}{
		{
			name: "new run in running state",
			run: Run{
				ID:      "run-123",
				AgentID: "alpine-agent",
				Status:  StatusRunning,
				Issue:   "https://github.com/owner/repo/issues/42",
				Created: now,
				Updated: now,
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "completed run",
			run: Run{
				ID:      "run-123",
				AgentID: "alpine-agent",
				Status:  StatusCompleted,
				Issue:   "https://github.com/owner/repo/issues/42",
				Created: now,
				Updated: now.Add(5 * time.Minute),
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "cancelled run",
			run: Run{
				ID:      "run-123",
				AgentID: "alpine-agent",
				Status:  StatusCancelled,
				Issue:   "https://github.com/owner/repo/issues/42",
				Created: now,
				Updated: now.Add(2 * time.Minute),
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "failed run",
			run: Run{
				ID:      "run-123",
				AgentID: "alpine-agent",
				Status:  StatusFailed,
				Issue:   "https://github.com/owner/repo/issues/42",
				Created: now,
				Updated: now.Add(1 * time.Minute),
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "run with invalid status",
			run: Run{
				ID:      "run-123",
				AgentID: "alpine-agent",
				Status:  "invalid-status",
				Issue:   "https://github.com/owner/repo/issues/42",
				Created: now,
				Updated: now,
			},
			wantErr:     true,
			validStatus: false,
		},
		{
			name: "run with empty ID",
			run: Run{
				ID:      "",
				AgentID: "alpine-agent",
				Status:  StatusRunning,
				Issue:   "https://github.com/owner/repo/issues/42",
				Created: now,
				Updated: now,
			},
			wantErr: true,
		},
		{
			name: "run with worktree directory",
			run: Run{
				ID:          "run-123",
				AgentID:     "alpine-agent",
				Status:      StatusRunning,
				Issue:       "https://github.com/owner/repo/issues/42",
				Created:     now,
				Updated:     now,
				WorktreeDir: "/tmp/alpine-worktree-123",
			},
			wantErr:     false,
			validStatus: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Run.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.validStatus {
				if !tt.run.IsValidStatus() {
					t.Errorf("Run.IsValidStatus() = false, want true for status %v", tt.run.Status)
				}
			}

			// Test JSON marshaling
			data, err := json.Marshal(tt.run)
			if err != nil {
				t.Fatalf("Failed to marshal run: %v", err)
			}

			var unmarshaled Run
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal run: %v", err)
			}

			if unmarshaled.ID != tt.run.ID {
				t.Errorf("Run ID mismatch: got %v, want %v", unmarshaled.ID, tt.run.ID)
			}
		})
	}
}

// TestPlanStatus tests the Plan struct approval workflow.
// This ensures that plans follow the expected lifecycle: pending -> approved/rejected,
// and that plan content and metadata are properly managed.
func TestPlanStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		plan        Plan
		wantErr     bool
		validStatus bool
	}{
		{
			name: "pending plan",
			plan: Plan{
				RunID:   "run-123",
				Content: "# Implementation Plan\n\n## Overview\nImplement user authentication...",
				Status:  PlanStatusPending,
				Created: now,
				Updated: now,
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "approved plan",
			plan: Plan{
				RunID:   "run-123",
				Content: "# Implementation Plan\n\n## Overview\nImplement user authentication...",
				Status:  PlanStatusApproved,
				Created: now,
				Updated: now.Add(1 * time.Minute),
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "rejected plan",
			plan: Plan{
				RunID:   "run-123",
				Content: "# Implementation Plan\n\n## Overview\nImplement user authentication...",
				Status:  PlanStatusRejected,
				Created: now,
				Updated: now.Add(30 * time.Second),
			},
			wantErr:     false,
			validStatus: true,
		},
		{
			name: "plan with invalid status",
			plan: Plan{
				RunID:   "run-123",
				Content: "# Implementation Plan",
				Status:  "invalid-status",
				Created: now,
				Updated: now,
			},
			wantErr:     true,
			validStatus: false,
		},
		{
			name: "plan with empty run ID",
			plan: Plan{
				RunID:   "",
				Content: "# Implementation Plan",
				Status:  PlanStatusPending,
				Created: now,
				Updated: now,
			},
			wantErr: true,
		},
		{
			name: "plan with empty content",
			plan: Plan{
				RunID:   "run-123",
				Content: "",
				Status:  PlanStatusPending,
				Created: now,
				Updated: now,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Plan.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.validStatus {
				if !tt.plan.IsValidStatus() {
					t.Errorf("Plan.IsValidStatus() = false, want true for status %v", tt.plan.Status)
				}
			}

			// Test JSON marshaling
			data, err := json.Marshal(tt.plan)
			if err != nil {
				t.Fatalf("Failed to marshal plan: %v", err)
			}

			var unmarshaled Plan
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal plan: %v", err)
			}

			if unmarshaled.RunID != tt.plan.RunID {
				t.Errorf("Plan RunID mismatch: got %v, want %v", unmarshaled.RunID, tt.plan.RunID)
			}
		})
	}
}

// TestModelSerialization tests JSON serialization/deserialization for all models.
// This ensures that all models can be properly converted to/from JSON for REST API
// request and response handling.
func TestModelSerialization(t *testing.T) {
	now := time.Now()

	t.Run("Agent serialization", func(t *testing.T) {
		agent := Agent{
			ID:          "test-agent",
			Name:        "Test Agent",
			Description: "Test agent for unit tests",
		}

		// Marshal to JSON
		data, err := json.Marshal(agent)
		if err != nil {
			t.Fatalf("Failed to marshal agent: %v", err)
		}

		// Verify JSON structure
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(data, &jsonMap); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		if jsonMap["id"] != agent.ID {
			t.Errorf("JSON id mismatch: got %v, want %v", jsonMap["id"], agent.ID)
		}
		if jsonMap["name"] != agent.Name {
			t.Errorf("JSON name mismatch: got %v, want %v", jsonMap["name"], agent.Name)
		}
		if jsonMap["description"] != agent.Description {
			t.Errorf("JSON description mismatch: got %v, want %v", jsonMap["description"], agent.Description)
		}
	})

	t.Run("Run serialization with omitempty", func(t *testing.T) {
		run := Run{
			ID:      "run-456",
			AgentID: "test-agent",
			Status:  StatusRunning,
			Issue:   "https://github.com/test/repo/issues/1",
			Created: now,
			Updated: now,
			// WorktreeDir is intentionally empty to test omitempty
		}

		data, err := json.Marshal(run)
		if err != nil {
			t.Fatalf("Failed to marshal run: %v", err)
		}

		// Verify that WorktreeDir is omitted when empty
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(data, &jsonMap); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		if _, exists := jsonMap["worktree_dir"]; exists {
			t.Error("worktree_dir should be omitted when empty")
		}

		// Test with WorktreeDir set
		run.WorktreeDir = "/tmp/test-worktree"
		data, err = json.Marshal(run)
		if err != nil {
			t.Fatalf("Failed to marshal run with worktree: %v", err)
		}

		if err := json.Unmarshal(data, &jsonMap); err != nil {
			t.Fatalf("Failed to unmarshal to map: %v", err)
		}

		if jsonMap["worktree_dir"] != run.WorktreeDir {
			t.Errorf("JSON worktree_dir mismatch: got %v, want %v", jsonMap["worktree_dir"], run.WorktreeDir)
		}
	})

	t.Run("Plan serialization", func(t *testing.T) {
		plan := Plan{
			RunID:   "run-789",
			Content: "# Test Plan\n\nThis is a test plan.",
			Status:  PlanStatusPending,
			Created: now,
			Updated: now,
		}

		data, err := json.Marshal(plan)
		if err != nil {
			t.Fatalf("Failed to marshal plan: %v", err)
		}

		var unmarshaled Plan
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal plan: %v", err)
		}

		if unmarshaled.Content != plan.Content {
			t.Errorf("Plan content mismatch: got %v, want %v", unmarshaled.Content, plan.Content)
		}
		if unmarshaled.Status != plan.Status {
			t.Errorf("Plan status mismatch: got %v, want %v", unmarshaled.Status, plan.Status)
		}
	})
}

// TestRunStatusTransitions tests that run status transitions follow expected patterns.
// This ensures the state machine for runs is properly implemented.
func TestRunStatusTransitions(t *testing.T) {
	tests := []struct {
		name            string
		fromStatus      string
		toStatus        string
		validTransition bool
	}{
		{"running to completed", StatusRunning, StatusCompleted, true},
		{"running to cancelled", StatusRunning, StatusCancelled, true},
		{"running to failed", StatusRunning, StatusFailed, true},
		{"completed to running", StatusCompleted, StatusRunning, false},
		{"cancelled to completed", StatusCancelled, StatusCompleted, false},
		{"failed to running", StatusFailed, StatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &Run{
				ID:      "test-run",
				AgentID: "test-agent",
				Status:  tt.fromStatus,
				Issue:   "https://github.com/test/repo/issues/1",
				Created: time.Now(),
				Updated: time.Now(),
			}

			canTransition := run.CanTransitionTo(tt.toStatus)
			if canTransition != tt.validTransition {
				t.Errorf("CanTransitionTo(%v) = %v, want %v", tt.toStatus, canTransition, tt.validTransition)
			}
		})
	}
}

// TestPlanStatusTransitions tests that plan status transitions follow expected patterns.
// This ensures the state machine for plans is properly implemented.
func TestPlanStatusTransitions(t *testing.T) {
	tests := []struct {
		name            string
		fromStatus      string
		toStatus        string
		validTransition bool
	}{
		{"pending to approved", PlanStatusPending, PlanStatusApproved, true},
		{"pending to rejected", PlanStatusPending, PlanStatusRejected, true},
		{"approved to pending", PlanStatusApproved, PlanStatusPending, false},
		{"approved to rejected", PlanStatusApproved, PlanStatusRejected, false},
		{"rejected to approved", PlanStatusRejected, PlanStatusApproved, false},
		{"rejected to pending", PlanStatusRejected, PlanStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := &Plan{
				RunID:   "test-run",
				Content: "Test plan content",
				Status:  tt.fromStatus,
				Created: time.Now(),
				Updated: time.Now(),
			}

			canTransition := plan.CanTransitionTo(tt.toStatus)
			if canTransition != tt.validTransition {
				t.Errorf("CanTransitionTo(%v) = %v, want %v", tt.toStatus, canTransition, tt.validTransition)
			}
		})
	}
}

// TestGenerateID tests the ID generation function.
// This ensures that generated IDs are unique and properly formatted.
func TestGenerateID(t *testing.T) {
	// Test ID generation with different prefixes
	prefixes := []string{"run", "plan", "session"}

	for _, prefix := range prefixes {
		t.Run(fmt.Sprintf("generate ID with prefix %s", prefix), func(t *testing.T) {
			id1 := GenerateID(prefix)
			id2 := GenerateID(prefix)

			// Check that IDs have the correct prefix
			if !strings.HasPrefix(id1, prefix+"-") {
				t.Errorf("ID %s should have prefix %s-", id1, prefix)
			}
			if !strings.HasPrefix(id2, prefix+"-") {
				t.Errorf("ID %s should have prefix %s-", id2, prefix)
			}

			// Check that IDs are unique
			if id1 == id2 {
				t.Errorf("Generated IDs should be unique, got %s twice", id1)
			}

			// Check that IDs have reasonable length
			if len(id1) < len(prefix)+5 {
				t.Errorf("ID %s seems too short", id1)
			}
		})
	}

	// Generate multiple IDs to ensure uniqueness
	t.Run("multiple IDs are unique", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := GenerateID("test")
			if ids[id] {
				t.Errorf("Duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})
}
