package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/gitx"
)

// Extend Server struct for testing workflow integration
// This simulates adding the workflowEngine field that will be implemented
type ServerWithWorkflow struct {
	*Server
	workflowEngine WorkflowEngine
}

// BroadcastEvent is a method that needs to be implemented on Server
// This is a placeholder for the integration test
func (s *ServerWithWorkflow) BroadcastEvent(event WorkflowEvent) {
	// TODO: Implement event broadcasting to SSE clients
}

// The WorkflowEngine interface and WorkflowEvent struct are now defined in interfaces.go

// MockWorkflowEngine is a mock implementation for testing
type MockWorkflowEngine struct {
	StartWorkflowFunc     func(ctx context.Context, issueURL string, runID string) (string, error)
	CancelWorkflowFunc    func(ctx context.Context, runID string) error
	GetWorkflowStateFunc  func(ctx context.Context, runID string) (*core.State, error)
	ApprovePlanFunc       func(ctx context.Context, runID string) error
	SubscribeToEventsFunc func(ctx context.Context, runID string) (<-chan WorkflowEvent, error)
}

func (m *MockWorkflowEngine) StartWorkflow(ctx context.Context, issueURL string, runID string) (string, error) {
	if m.StartWorkflowFunc != nil {
		return m.StartWorkflowFunc(ctx, issueURL, runID)
	}
	return "", fmt.Errorf("not implemented")
}

func (m *MockWorkflowEngine) CancelWorkflow(ctx context.Context, runID string) error {
	if m.CancelWorkflowFunc != nil {
		return m.CancelWorkflowFunc(ctx, runID)
	}
	return fmt.Errorf("not implemented")
}

func (m *MockWorkflowEngine) GetWorkflowState(ctx context.Context, runID string) (*core.State, error) {
	if m.GetWorkflowStateFunc != nil {
		return m.GetWorkflowStateFunc(ctx, runID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *MockWorkflowEngine) ApprovePlan(ctx context.Context, runID string) error {
	if m.ApprovePlanFunc != nil {
		return m.ApprovePlanFunc(ctx, runID)
	}
	return fmt.Errorf("not implemented")
}

func (m *MockWorkflowEngine) SubscribeToEvents(ctx context.Context, runID string) (<-chan WorkflowEvent, error) {
	if m.SubscribeToEventsFunc != nil {
		return m.SubscribeToEventsFunc(ctx, runID)
	}
	return nil, fmt.Errorf("not implemented")
}

// Helper function to create handlers that work with ServerWithWorkflow
func (s *ServerWithWorkflow) agentsRunHandler(w http.ResponseWriter, r *http.Request) {
	s.Server.agentsRunHandler(w, r)
}

func (s *ServerWithWorkflow) runCancelHandler(w http.ResponseWriter, r *http.Request) {
	s.Server.runCancelHandler(w, r)
}

func (s *ServerWithWorkflow) runEventsHandler(w http.ResponseWriter, r *http.Request) {
	s.Server.runEventsHandler(w, r)
}

func (s *ServerWithWorkflow) planApproveHandler(w http.ResponseWriter, r *http.Request) {
	s.Server.planApproveHandler(w, r)
}

func (s *ServerWithWorkflow) runDetailsHandler(w http.ResponseWriter, r *http.Request) {
	s.Server.runDetailsHandler(w, r)
}

func (s *ServerWithWorkflow) sseHandler(w http.ResponseWriter, r *http.Request) {
	s.Server.sseHandler(w, r)
}

// TestAgentsRunWorkflowIntegration tests that the agentsRunHandler properly starts a workflow
// through the workflow engine and updates the run state accordingly.
func TestAgentsRunWorkflowIntegration(t *testing.T) {
	t.Run("successful workflow start", func(t *testing.T) {
		// Create mock workflow engine
		mockEngine := &MockWorkflowEngine{
			StartWorkflowFunc: func(ctx context.Context, issueURL string, runID string) (string, error) {
				// Verify correct parameters are passed
				if issueURL != "https://github.com/owner/repo/issues/123" {
					t.Errorf("expected issue URL https://github.com/owner/repo/issues/123, got %s", issueURL)
				}
				if !strings.HasPrefix(runID, "run-") {
					t.Errorf("expected run ID to start with 'run-', got %s", runID)
				}
				// Return worktree directory
				return "/tmp/alpine-worktree-123", nil
			},
		}

		// Create server with workflow engine
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}

		// Create request
		payload := map[string]string{
			"issue_url": "https://github.com/owner/repo/issues/123",
			"agent_id":  "alpine-agent",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
		w := httptest.NewRecorder()

		// Execute handler
		server.agentsRunHandler(w, req)

		// Verify response
		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}

		var response Run
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Verify run was created with correct values
		if response.Status != "running" {
			t.Errorf("expected status 'running', got %s", response.Status)
		}
		if response.WorktreeDir != "/tmp/alpine-worktree-123" {
			t.Errorf("expected worktree dir '/tmp/alpine-worktree-123', got %s", response.WorktreeDir)
		}
		if response.Issue != "https://github.com/owner/repo/issues/123" {
			t.Errorf("expected issue URL to be stored, got %s", response.Issue)
		}
	})

	t.Run("workflow start failure", func(t *testing.T) {
		// Create mock workflow engine that fails
		mockEngine := &MockWorkflowEngine{
			StartWorkflowFunc: func(ctx context.Context, issueURL string, runID string) (string, error) {
				return "", fmt.Errorf("failed to parse GitHub issue")
			},
		}

		// Create server with workflow engine
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}

		// Create request
		payload := map[string]string{
			"issue_url": "https://github.com/owner/repo/issues/invalid",
			"agent_id":  "alpine-agent",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
		w := httptest.NewRecorder()

		// Execute handler
		server.agentsRunHandler(w, req)

		// Should still create run but with failed status
		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}

		var response Run
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Verify run was created with failed status
		if response.Status != "failed" {
			t.Errorf("expected status 'failed', got %s", response.Status)
		}
	})
}

// TestRunCancelWorkflowIntegration tests that the runCancelHandler properly cancels
// a running workflow through the workflow engine.
func TestRunCancelWorkflowIntegration(t *testing.T) {
	t.Run("successful workflow cancellation", func(t *testing.T) {
		cancelCalled := false
		mockEngine := &MockWorkflowEngine{
			CancelWorkflowFunc: func(ctx context.Context, runID string) error {
				cancelCalled = true
				if runID != "run_123" {
					t.Errorf("expected run ID 'run_123', got %s", runID)
				}
				return nil
			},
		}

		// Create server with workflow engine and existing run
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}
		server.Server.runs["run_123"] = &Run{
			ID:      "run_123",
			Status:  "running",
			Created: time.Now(),
			Updated: time.Now(),
		}

		// Create request
		req := httptest.NewRequest(http.MethodPost, "/runs/run_123/cancel", nil)
		req.SetPathValue("id", "run_123")
		w := httptest.NewRecorder()

		// Execute handler
		server.runCancelHandler(w, req)

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Verify workflow engine was called
		if !cancelCalled {
			t.Error("expected workflow engine CancelWorkflow to be called")
		}

		// Verify run status was updated
		if server.Server.runs["run_123"].Status != "cancelled" {
			t.Errorf("expected run status to be 'cancelled', got %s", server.Server.runs["run_123"].Status)
		}
	})

	t.Run("cancellation of non-running workflow", func(t *testing.T) {
		mockEngine := &MockWorkflowEngine{
			CancelWorkflowFunc: func(ctx context.Context, runID string) error {
				t.Error("CancelWorkflow should not be called for completed workflow")
				return nil
			},
		}

		// Create server with completed run
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}
		server.Server.runs["run_123"] = &Run{
			ID:      "run_123",
			Status:  "completed",
			Created: time.Now(),
			Updated: time.Now(),
		}

		// Create request
		req := httptest.NewRequest(http.MethodPost, "/runs/run_123/cancel", nil)
		req.SetPathValue("id", "run_123")
		w := httptest.NewRecorder()

		// Execute handler
		server.runCancelHandler(w, req)

		// Should return error
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

// TestWorkflowEventBroadcasting tests that workflow events are properly broadcast
// to both global and run-specific SSE clients during workflow execution.
func TestWorkflowEventBroadcasting(t *testing.T) {
	t.Run("events broadcast to run-specific SSE endpoint", func(t *testing.T) {
		// Create event channel for mock
		eventChan := make(chan WorkflowEvent, 10)

		mockEngine := &MockWorkflowEngine{
			SubscribeToEventsFunc: func(ctx context.Context, runID string) (<-chan WorkflowEvent, error) {
				if runID != "run_123" {
					t.Errorf("expected run ID 'run_123', got %s", runID)
				}
				return eventChan, nil
			},
		}

		// Create server with workflow engine
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}
		server.Server.runs["run_123"] = &Run{
			ID:     "run_123",
			Status: "running",
		}

		// Create SSE request
		req := httptest.NewRequest(http.MethodGet, "/runs/run_123/events", nil)
		req.SetPathValue("id", "run_123")
		w := httptest.NewRecorder()

		// Start handler in goroutine
		handlerDone := make(chan bool)
		go func() {
			server.runEventsHandler(w, req)
			handlerDone <- true
		}()

		// Give handler time to set up
		time.Sleep(50 * time.Millisecond)

		// Send test events
		eventChan <- WorkflowEvent{
			Type:      "state_changed",
			RunID:     "run_123",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"status": "planning",
			},
		}

		eventChan <- WorkflowEvent{
			Type:      "log",
			RunID:     "run_123",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"message": "Generating plan from GitHub issue",
			},
		}

		// Close event channel to end handler
		close(eventChan)

		// Wait for handler to complete
		select {
		case <-handlerDone:
		case <-time.After(1 * time.Second):
			t.Fatal("handler did not complete in time")
		}

		// Verify SSE response
		response := w.Body.String()
		if !strings.Contains(response, "event: state_changed") {
			t.Error("expected state_changed event in response")
		}
		if !strings.Contains(response, "event: log") {
			t.Error("expected log event in response")
		}
		if !strings.Contains(response, "data: {") {
			t.Error("expected JSON data in SSE response")
		}
	})

	t.Run("events broadcast to global SSE endpoint", func(t *testing.T) {
		// Create server with workflow engine
		baseServer := NewServer(0)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: &MockWorkflowEngine{},
		}

		// Create request with cancelable context
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		// Start handler in goroutine
		handlerDone := make(chan bool)
		go func() {
			server.sseHandler(w, req)
			handlerDone <- true
		}()

		// Give handler time to set up
		time.Sleep(50 * time.Millisecond)

		// Broadcast global event to the embedded Server
		server.Server.BroadcastEvent(WorkflowEvent{
			Type:      "workflow_started",
			RunID:     "run_456",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"issue": "https://github.com/owner/repo/issues/456",
			},
		})

		// Give time for event to be processed
		time.Sleep(50 * time.Millisecond)

		// Cancel request context to end handler
		cancel()

		// Wait for handler to complete
		select {
		case <-handlerDone:
		case <-time.After(1 * time.Second):
			t.Fatal("handler did not complete in time")
		}

		// Verify SSE response contains global event
		response := w.Body.String()
		if !strings.Contains(response, "event: workflow_started") {
			t.Error("expected workflow_started event in response")
		}
	})
}

// TestPlanApprovalWorkflowIntegration tests that plan approval properly triggers
// workflow continuation through the workflow engine.
func TestPlanApprovalWorkflowIntegration(t *testing.T) {
	t.Run("successful plan approval continues workflow", func(t *testing.T) {
		approveCalled := false
		mockEngine := &MockWorkflowEngine{
			ApprovePlanFunc: func(ctx context.Context, runID string) error {
				approveCalled = true
				if runID != "run_123" {
					t.Errorf("expected run ID 'run_123', got %s", runID)
				}
				return nil
			},
			GetWorkflowStateFunc: func(ctx context.Context, runID string) (*core.State, error) {
				return &core.State{
					CurrentStepDescription: "Plan approved, continuing implementation",
					NextStepPrompt:         "/run_implementation_loop",
					Status:                 core.StatusRunning,
				}, nil
			},
		}

		// Create server with workflow engine and existing plan
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}
		server.Server.runs["run_123"] = &Run{
			ID:     "run_123",
			Status: "planning",
		}
		server.Server.plans["run_123"] = &Plan{
			RunID:   "run_123",
			Content: "Test plan content",
			Status:  "pending",
			Created: time.Now(),
			Updated: time.Now(),
		}

		// Create request
		req := httptest.NewRequest(http.MethodPost, "/plans/run_123/approve", nil)
		req.SetPathValue("runId", "run_123")
		w := httptest.NewRecorder()

		// Execute handler
		server.planApproveHandler(w, req)

		// Verify response
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Verify workflow engine was called
		if !approveCalled {
			t.Error("expected workflow engine ApprovePlan to be called")
		}

		// Verify plan status was updated
		if server.Server.plans["run_123"].Status != "approved" {
			t.Errorf("expected plan status to be 'approved', got %s", server.Server.plans["run_123"].Status)
		}

		// Verify run status was updated
		if server.Server.runs["run_123"].Status != "running" {
			t.Errorf("expected run status to be 'running', got %s", server.Server.runs["run_123"].Status)
		}
	})

	t.Run("plan approval with workflow error", func(t *testing.T) {
		mockEngine := &MockWorkflowEngine{
			ApprovePlanFunc: func(ctx context.Context, runID string) error {
				return fmt.Errorf("workflow execution failed")
			},
		}

		// Create server with workflow engine and existing plan
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}
		server.Server.plans["run_123"] = &Plan{
			RunID:  "run_123",
			Status: "pending",
		}

		// Create request
		req := httptest.NewRequest(http.MethodPost, "/plans/run_123/approve", nil)
		req.SetPathValue("runId", "run_123")
		w := httptest.NewRecorder()

		// Execute handler
		server.planApproveHandler(w, req)

		// Should return error
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}

		// Plan status should remain pending
		if server.Server.plans["run_123"].Status != "pending" {
			t.Errorf("expected plan status to remain 'pending', got %s", server.Server.plans["run_123"].Status)
		}
	})
}

// TestWorkflowStateSync tests that the REST API properly syncs with workflow state changes
func TestWorkflowStateSync(t *testing.T) {
	t.Run("run details reflect current workflow state", func(t *testing.T) {
		mockEngine := &MockWorkflowEngine{
			GetWorkflowStateFunc: func(ctx context.Context, runID string) (*core.State, error) {
				return &core.State{
					CurrentStepDescription: "Implementing user authentication",
					NextStepPrompt:         "/continue",
					Status:                 core.StatusRunning,
				}, nil
			},
		}

		// Create server with workflow engine
		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}
		server.Server.runs["run_123"] = &Run{
			ID:     "run_123",
			Status: "running",
		}

		// Before getting run details, sync with workflow state
		state, _ := mockEngine.GetWorkflowState(context.Background(), "run_123")
		if state.Status == core.StatusCompleted {
			server.runs["run_123"].Status = "completed"
		}

		// Create request
		req := httptest.NewRequest(http.MethodGet, "/runs/run_123", nil)
		req.SetPathValue("id", "run_123")
		w := httptest.NewRecorder()

		// Execute handler
		server.runDetailsHandler(w, req)

		// Verify response includes workflow state
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should include current workflow state
		if response["current_step"] != "Implementing user authentication" {
			t.Errorf("expected current_step in response, got %v", response["current_step"])
		}
	})
}

// TestConcurrentWorkflowOperations tests thread-safety of workflow operations
func TestConcurrentWorkflowOperations(t *testing.T) {
	t.Run("concurrent workflow starts", func(t *testing.T) {
		var startCount int
		var mu sync.Mutex
		mockEngine := &MockWorkflowEngine{
			StartWorkflowFunc: func(ctx context.Context, issueURL string, runID string) (string, error) {
				mu.Lock()
				startCount++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond) // Simulate work
				return fmt.Sprintf("/tmp/worktree-%s", runID), nil
			},
		}

		baseServer := NewServer(0)
		baseServer.SetWorkflowEngine(mockEngine)
		server := &ServerWithWorkflow{
			Server:         baseServer,
			workflowEngine: mockEngine,
		}

		// Start multiple workflows concurrently
		done := make(chan bool, 3)
		for i := 0; i < 3; i++ {
			go func(index int) {
				payload := map[string]string{
					"issue_url": fmt.Sprintf("https://github.com/owner/repo/issues/%d", index),
					"agent_id":  "alpine-agent",
				}
				body, _ := json.Marshal(payload)
				req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
				w := httptest.NewRecorder()

				server.agentsRunHandler(w, req)

				if w.Code != http.StatusCreated {
					t.Errorf("request %d: expected status 201, got %d", index, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 3; i++ {
			<-done
		}

		// Verify all workflows were started
		mu.Lock()
		finalCount := startCount
		mu.Unlock()
		if finalCount != 3 {
			t.Errorf("expected 3 workflow starts, got %d", finalCount)
		}

		// Verify all runs were created
		if len(server.Server.runs) != 3 {
			t.Errorf("expected 3 runs, got %d", len(server.Server.runs))
		}
	})
}

// TestCreateWorkflowDirectoryWithGitHubClone tests GitHub URL detection and clone integration
// in the createWorkflowDirectory method. This tests Task 4 implementation.
func TestCreateWorkflowDirectoryWithGitHubClone(t *testing.T) {
	t.Run("creates worktree in cloned repository for GitHub issue URL", func(t *testing.T) {
		// This test should FAIL initially (RED phase of TDD)
		// because the GitHub URL detection and clone integration is not yet implemented
		
		// Create mock configuration with git clone enabled
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled:   true,
					AuthToken: "",
					Timeout:   time.Duration(30) * time.Second,
					Depth:     1,
				},
			},
		}

		// Track whether clone logic was invoked
		cloneLogicInvoked := false
		
		// Create mock worktree manager that can detect clone logic
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, name string) (*gitx.Worktree, error) {
				// Check if we are in a cloned repository context by checking if
				// the worktree name has the "cloned-" prefix indicating clone logic was invoked
				if strings.HasPrefix(name, "cloned-") {
					cloneLogicInvoked = true
				}
				
				return &gitx.Worktree{
					Path:       "/path/to/cloned/repo/.git/worktrees/" + name,
					Branch:     "alpine/" + name,
					ParentRepo: "/path/to/cloned/repo",
				}, nil
			},
		}

		// Create workflow engine with mock components
		engine := NewAlpineWorkflowEngine(nil, mockWtMgr, cfg)
		
		// Create context with GitHub issue URL (use a public repo that actually exists)
		ctx := context.WithValue(context.Background(), "issue_url", "https://github.com/microsoft/vscode/issues/123")
		cancel := func() {}
		
		// Call createWorkflowDirectory - this should detect GitHub URL and clone repository
		worktreeDir, err := engine.createWorkflowDirectory(ctx, "test-run-123", cancel)
		
		// Verify that the method detects GitHub URL and attempts clone
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		
		// This test will FAIL because the GitHub URL detection is not implemented yet
		if !cloneLogicInvoked {
			t.Errorf("expected GitHub URL detection to trigger clone logic, but clone logic was not invoked")
		}
		
		// Verify that worktree was created in cloned repository context
		if !strings.Contains(worktreeDir, "cloned") {
			t.Errorf("expected worktree directory to indicate cloned repository context, got: %s", worktreeDir)
		}
	})

	t.Run("falls back to regular worktree when clone disabled", func(t *testing.T) {
		// This test should PASS initially as it tests existing behavior
		
		// Create mock configuration with git clone disabled
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled: false, // Clone disabled
				},
			},
		}

		// Create mock worktree manager
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, name string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       "/path/to/regular/worktree/" + name,
					Branch:     "alpine/" + name,
					ParentRepo: "/path/to/repo",
				}, nil
			},
		}

		// Create workflow engine
		engine := NewAlpineWorkflowEngine(nil, mockWtMgr, cfg)
		
		// Create context with GitHub issue URL (should be ignored when clone disabled)
		ctx := context.WithValue(context.Background(), "issue_url", "https://github.com/owner/repo/issues/123")
		cancel := func() {}
		
		// Call createWorkflowDirectory
		worktreeDir, err := engine.createWorkflowDirectory(ctx, "test-run-123", cancel)
		
		// Should use regular worktree creation (existing behavior)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		
		// Should create regular worktree, not attempt clone
		if !strings.Contains(worktreeDir, "regular") {
			t.Errorf("expected regular worktree path, got: %s", worktreeDir)
		}
	})

	t.Run("falls back to regular worktree when clone fails", func(t *testing.T) {
		// This test should FAIL initially (RED phase)
		// because clone failure handling is not yet implemented
		
		// Create mock configuration with git clone enabled
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled:   true,
					AuthToken: "",
					Timeout:   time.Duration(30) * time.Second,
					Depth:     1,
				},
			},
		}

		// Create mock worktree manager
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, name string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       "/path/to/fallback/worktree/" + name,
					Branch:     "alpine/" + name,
					ParentRepo: "/path/to/repo",
				}, nil
			},
		}

		// Create workflow engine
		engine := NewAlpineWorkflowEngine(nil, mockWtMgr, cfg)
		
		// Create context with GitHub issue URL that will fail to clone
		ctx := context.WithValue(context.Background(), "issue_url", "https://github.com/nonexistent/repo/issues/123")
		cancel := func() {}
		
		// Call createWorkflowDirectory
		worktreeDir, err := engine.createWorkflowDirectory(ctx, "test-run-123", cancel)
		
		// Should not return error (graceful fallback)
		if err != nil {
			t.Errorf("expected no error with fallback, got: %v", err)
		}
		
		// Should fall back to regular worktree creation
		// (This assertion will fail initially because fallback logic is not implemented)
		if !strings.Contains(worktreeDir, "fallback") {
			t.Errorf("expected fallback worktree path, got: %s", worktreeDir)
		}
	})

	t.Run("handles non-GitHub URLs by using regular worktree", func(t *testing.T) {
		// This test should PASS initially as it tests existing behavior
		
		// Create mock configuration
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled: true,
				},
			},
		}

		// Create mock worktree manager
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, name string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       "/path/to/regular/worktree/" + name,
					Branch:     "alpine/" + name,
					ParentRepo: "/path/to/repo",
				}, nil
			},
		}

		// Create workflow engine
		engine := NewAlpineWorkflowEngine(nil, mockWtMgr, cfg)
		
		// Create context with non-GitHub URL
		ctx := context.WithValue(context.Background(), "issue_url", "https://example.com/some/task")
		cancel := func() {}
		
		// Call createWorkflowDirectory
		worktreeDir, err := engine.createWorkflowDirectory(ctx, "test-run-123", cancel)
		
		// Should use regular worktree creation
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		
		// Should create regular worktree, not attempt clone
		if !strings.Contains(worktreeDir, "regular") {
			t.Errorf("expected regular worktree path, got: %s", worktreeDir)
		}
	})
}

