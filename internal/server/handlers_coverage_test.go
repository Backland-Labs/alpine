package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/gitx"
)

// TestPlanFeedbackHandlerCoverage tests edge cases for planFeedbackHandler to increase coverage
func TestPlanFeedbackHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		name         string
		runID        string
		setupFunc    func()
		payload      interface{}
		expectedCode int
	}{
		{
			name:  "plan not found",
			runID: "run-not-exists",
			payload: map[string]string{
				"feedback": "Test feedback",
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:  "empty feedback",
			runID: "run-123",
			setupFunc: func() {
				server.plans["run-123"] = &Plan{
					RunID:  "run-123",
					Status: "pending",
				}
			},
			payload: map[string]string{
				"feedback": "",
			},
			expectedCode: http.StatusOK, // Handler doesn't validate empty feedback
		},
		{
			name:  "invalid JSON",
			runID: "run-123",
			setupFunc: func() {
				server.plans["run-123"] = &Plan{
					RunID:  "run-123",
					Status: "pending",
				}
			},
			payload:      "invalid json",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:  "plan already approved",
			runID: "run-456",
			setupFunc: func() {
				server.plans["run-456"] = &Plan{
					RunID:  "run-456",
					Status: "approved",
				}
			},
			payload: map[string]string{
				"feedback": "Test feedback",
			},
			expectedCode: http.StatusOK, // Handler doesn't check plan status
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset server state
			server.plans = make(map[string]*Plan)

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			var body []byte
			if str, ok := tt.payload.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/plans/"+tt.runID+"/feedback", bytes.NewReader(body))
			req.SetPathValue("runId", tt.runID)
			w := httptest.NewRecorder()

			server.planFeedbackHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestPlanGetHandlerCoverage tests edge cases for planGetHandler
func TestPlanGetHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		name         string
		runID        string
		setupFunc    func()
		expectedCode int
	}{
		{
			name:         "empty run ID",
			runID:        "",
			expectedCode: http.StatusNotFound, // Handler doesn't validate empty ID separately
		},
		{
			name:         "plan not found",
			runID:        "run-not-exists",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset server state
			server.plans = make(map[string]*Plan)

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			req := httptest.NewRequest(http.MethodGet, "/plans/"+tt.runID, nil)
			req.SetPathValue("runId", tt.runID)
			w := httptest.NewRecorder()

			server.planGetHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestRunDetailsHandlerCoverage tests edge cases for runDetailsHandler
func TestRunDetailsHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		name         string
		runID        string
		expectedCode int
	}{
		{
			name:         "empty run ID",
			runID:        "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "run not found",
			runID:        "run-not-exists",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/runs/"+tt.runID, nil)
			req.SetPathValue("id", tt.runID)
			w := httptest.NewRecorder()

			server.runDetailsHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestRunCancelHandlerCoverage tests edge cases for runCancelHandler
func TestRunCancelHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		name         string
		runID        string
		expectedCode int
	}{
		{
			name:         "empty run ID",
			runID:        "",
			expectedCode: http.StatusNotFound, // Handler doesn't validate empty ID separately
		},
		{
			name:         "run not found",
			runID:        "run-not-exists",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/runs/"+tt.runID+"/cancel", nil)
			req.SetPathValue("id", tt.runID)
			w := httptest.NewRecorder()

			server.runCancelHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestRunEventsHandlerCoverage tests edge cases for runEventsHandler
func TestRunEventsHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	// Test empty run ID
	t.Run("empty run ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/runs//events", nil)
		req.SetPathValue("id", "")
		w := httptest.NewRecorder()

		server.runEventsHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	// Test run not found
	t.Run("run not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/runs/run-not-exists/events", nil)
		req.SetPathValue("id", "run-not-exists")
		w := httptest.NewRecorder()

		server.runEventsHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	// Test workflow engine not set
	t.Run("workflow engine not set", func(t *testing.T) {
		server.runs["run-123"] = &Run{
			ID:     "run-123",
			Status: "running",
		}

		req := httptest.NewRequest(http.MethodGet, "/runs/run-123/events", nil)
		req.SetPathValue("id", "run-123")
		w := httptest.NewRecorder()

		server.runEventsHandler(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}

// TestPlanApproveHandlerCoverage tests edge cases for planApproveHandler
func TestPlanApproveHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		name         string
		runID        string
		setupFunc    func()
		expectedCode int
	}{
		{
			name:         "empty run ID",
			runID:        "",
			expectedCode: http.StatusNotFound, // Handler doesn't validate empty ID separately
		},
		{
			name:         "plan not found",
			runID:        "run-not-exists",
			expectedCode: http.StatusNotFound,
		},
		{
			name:  "workflow engine not set",
			runID: "run-123",
			setupFunc: func() {
				server.plans["run-123"] = &Plan{
					RunID:  "run-123",
					Status: "pending",
				}
			},
			expectedCode: http.StatusOK, // Handler doesn't check for workflow engine
		},
		{
			name:  "plan already approved",
			runID: "run-456",
			setupFunc: func() {
				server.plans["run-456"] = &Plan{
					RunID:  "run-456",
					Status: "approved",
				}
			},
			expectedCode: http.StatusOK, // Handler doesn't check plan status
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset server state
			server.plans = make(map[string]*Plan)

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			req := httptest.NewRequest(http.MethodPost, "/plans/"+tt.runID+"/approve", nil)
			req.SetPathValue("runId", tt.runID)
			w := httptest.NewRecorder()

			server.planApproveHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestAgentsRunHandlerCoverage tests edge cases for agentsRunHandler
func TestAgentsRunHandlerCoverage(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		name         string
		payload      interface{}
		expectedCode int
	}{
		{
			name:         "invalid JSON",
			payload:      "invalid json",
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "missing github_issue_url",
			payload: map[string]string{
				"agent_id": "alpine-agent",
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if str, ok := tt.payload.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
			w := httptest.NewRecorder()

			server.agentsRunHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}

// TestModelsCoverage tests edge cases for model methods
func TestModelsCoverage(t *testing.T) {
	t.Run("GenerateID retry logic", func(t *testing.T) {
		// This test ensures the retry logic in GenerateID is covered
		// by calling it multiple times
		ids := make(map[string]bool)
		for i := 0; i < 10; i++ {
			id := GenerateID("test")
			if ids[id] {
				t.Errorf("duplicate ID generated: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("CanTransitionTo edge cases", func(t *testing.T) {
		// Test run status transitions
		run := &Run{Status: "unknown"}
		if run.CanTransitionTo("running") {
			t.Error("should not allow transition from unknown status")
		}

		// Test plan status transitions
		plan := &Plan{Status: "unknown"}
		if plan.CanTransitionTo("approved") {
			t.Error("should not allow transition from unknown status")
		}
	})
}

// TestServerMethodsCoverage tests edge cases for server methods
func TestServerMethodsCoverage(t *testing.T) {
	t.Run("Address when server not started", func(t *testing.T) {
		server := NewServer(0)
		addr := server.Address()
		if addr != "" {
			t.Errorf("expected empty address, got %s", addr)
		}
	})

	t.Run("BroadcastEvent with no clients", func(t *testing.T) {
		server := NewServer(0)
		// This should not panic even with no clients
		server.BroadcastEvent(WorkflowEvent{
			Type:      "test",
			RunID:     "run-123",
			Timestamp: time.Now(),
		})
	})

	t.Run("respondWithError coverage", func(t *testing.T) {
		server := NewServer(0)
		w := httptest.NewRecorder()

		// Test with error message
		server.respondWithError(w, http.StatusBadRequest, "test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var response map[string]string
		json.NewDecoder(w.Body).Decode(&response)
		if response["error"] != "test error" {
			t.Errorf("expected error 'test error', got '%s'", response["error"])
		}
	})
}

// TestWorkflowIntegrationCoverage tests edge cases for workflow integration
func TestWorkflowIntegrationCoverage(t *testing.T) {
	t.Run("GetWorkflowState with invalid state file", func(t *testing.T) {
		tempDir := t.TempDir()
		stateFile := filepath.Join(tempDir, "agent_state", "agent_state.json")
		os.MkdirAll(filepath.Dir(stateFile), 0755)

		// Write invalid JSON
		os.WriteFile(stateFile, []byte("invalid json"), 0644)

		engine := NewAlpineWorkflowEngine(
			&MockClaudeExecutor{},
			&MockWorktreeManager{},
			&config.Config{},
		)

		engine.workflows["run-123"] = &workflowInstance{
			worktreeDir: tempDir,
			events:      make(chan WorkflowEvent, 1),
		}

		// GetWorkflowState returns empty state on error
		state, err := engine.GetWorkflowState(context.Background(), "run-123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if state == nil {
			t.Error("expected empty state, got nil")
		}
	})

	t.Run("createWorkflowDirectory with worktree error", func(t *testing.T) {
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return nil, errors.New("worktree creation failed")
			},
		}

		engine := NewAlpineWorkflowEngine(
			&MockClaudeExecutor{},
			mockWtMgr,
			&config.Config{
				Git: config.GitConfig{
					WorktreeEnabled: true,
				},
			},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := engine.createWorkflowDirectory(ctx, "run-123", cancel)
		if err == nil {
			t.Error("expected error when worktree creation fails")
		}
	})

	t.Run("SubscribeToEvents with closed workflow", func(t *testing.T) {
		engine := NewAlpineWorkflowEngine(
			&MockClaudeExecutor{},
			&MockWorktreeManager{},
			&config.Config{},
		)

		// Create workflow with closed channel
		closedChan := make(chan WorkflowEvent)
		close(closedChan)

		engine.workflows["run-123"] = &workflowInstance{
			events: closedChan,
		}

		// This should handle the closed channel gracefully
		_, err := engine.SubscribeToEvents(context.Background(), "run-123")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
