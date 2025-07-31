package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHealthEndpoint tests the health check endpoint to ensure the server is operational
func TestHealthEndpoint(t *testing.T) {
	server := NewServer(0)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %s", response["status"])
	}
}

// TestAgentsListEndpoint tests retrieving the list of available agents
func TestAgentsListEndpoint(t *testing.T) {
	server := NewServer(0)

	req := httptest.NewRequest(http.MethodGet, "/agents/list", nil)
	w := httptest.NewRecorder()

	server.agentsListHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var agents []Agent
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have at least one agent in MVP
	if len(agents) == 0 {
		t.Error("expected at least one agent")
	}
}

// TestAgentsRunEndpoint tests starting a workflow with a GitHub issue
func TestAgentsRunEndpoint(t *testing.T) {
	server := NewServer(0)

	payload := map[string]string{
		"issue_url": "https://github.com/owner/repo/issues/123",
		"agent_id":  "alpine-agent",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.agentsRunHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var run Run
	if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if run.ID == "" {
		t.Error("expected run ID to be set")
	}
	if run.Status != "running" {
		t.Errorf("expected status 'running', got %s", run.Status)
	}
	if run.Issue != payload["issue_url"] {
		t.Errorf("expected issue URL %s, got %s", payload["issue_url"], run.Issue)
	}
}

// TestRunsListEndpoint tests listing all runs
func TestRunsListEndpoint(t *testing.T) {
	server := NewServer(0)
	// Initialize with some test data
	server.runs = map[string]*Run{
		"run-1": {
			ID:      "run-1",
			AgentID: "alpine-agent",
			Status:  "completed",
			Issue:   "https://github.com/owner/repo/issues/1",
			Created: time.Now().Add(-1 * time.Hour),
			Updated: time.Now().Add(-30 * time.Minute),
		},
		"run-2": {
			ID:      "run-2",
			AgentID: "alpine-agent",
			Status:  "running",
			Issue:   "https://github.com/owner/repo/issues/2",
			Created: time.Now().Add(-10 * time.Minute),
			Updated: time.Now(),
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	w := httptest.NewRecorder()

	server.runsListHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var runs []Run
	if err := json.NewDecoder(w.Body).Decode(&runs); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}

// TestRunDetailsEndpoint tests getting specific run details
func TestRunDetailsEndpoint(t *testing.T) {
	server := NewServer(0)
	testRun := &Run{
		ID:      "test-run-123",
		AgentID: "alpine-agent",
		Status:  "running",
		Issue:   "https://github.com/owner/repo/issues/42",
		Created: time.Now(),
		Updated: time.Now(),
	}
	server.runs = map[string]*Run{
		testRun.ID: testRun,
	}

	tests := []struct {
		name       string
		runID      string
		wantStatus int
	}{
		{
			name:       "existing run",
			runID:      "test-run-123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent run",
			runID:      "does-not-exist",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/runs/"+tt.runID, nil)
			req.SetPathValue("id", tt.runID)
			w := httptest.NewRecorder()

			server.runDetailsHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusOK {
				var run Run
				if err := json.NewDecoder(w.Body).Decode(&run); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if run.ID != tt.runID {
					t.Errorf("expected run ID %s, got %s", tt.runID, run.ID)
				}
			}
		})
	}
}

// TestRunCancelEndpoint tests canceling a running workflow
func TestRunCancelEndpoint(t *testing.T) {
	server := NewServer(0)
	testRun := &Run{
		ID:      "test-run-456",
		AgentID: "alpine-agent",
		Status:  "running",
		Issue:   "https://github.com/owner/repo/issues/99",
		Created: time.Now(),
		Updated: time.Now(),
	}
	server.runs = map[string]*Run{
		testRun.ID: testRun,
	}

	req := httptest.NewRequest(http.MethodPost, "/runs/test-run-456/cancel", nil)
	req.SetPathValue("id", "test-run-456")
	w := httptest.NewRecorder()

	server.runCancelHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "cancelled" {
		t.Errorf("expected status 'cancelled', got %s", response["status"])
	}

	// Verify the run status was updated
	if server.runs["test-run-456"].Status != "cancelled" {
		t.Error("expected run status to be updated to 'cancelled'")
	}
}

// TestPlanGetEndpoint tests retrieving plan content for a run
func TestPlanGetEndpoint(t *testing.T) {
	server := NewServer(0)
	testPlan := &Plan{
		RunID:   "test-run-789",
		Content: "# Test Plan\n\n- Step 1\n- Step 2\n- Step 3",
		Status:  "pending",
		Created: time.Now(),
		Updated: time.Now(),
	}
	server.plans = map[string]*Plan{
		testPlan.RunID: testPlan,
	}

	req := httptest.NewRequest(http.MethodGet, "/plans/test-run-789", nil)
	req.SetPathValue("runId", "test-run-789")
	w := httptest.NewRecorder()

	server.planGetHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var plan Plan
	if err := json.NewDecoder(w.Body).Decode(&plan); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if plan.RunID != testPlan.RunID {
		t.Errorf("expected run ID %s, got %s", testPlan.RunID, plan.RunID)
	}
	if plan.Content != testPlan.Content {
		t.Error("plan content mismatch")
	}
}

// TestPlanApproveEndpoint tests approving a plan to continue workflow
func TestPlanApproveEndpoint(t *testing.T) {
	server := NewServer(0)
	testPlan := &Plan{
		RunID:   "test-run-approve",
		Content: "# Approval Test Plan",
		Status:  "pending",
		Created: time.Now(),
		Updated: time.Now(),
	}
	server.plans = map[string]*Plan{
		testPlan.RunID: testPlan,
	}

	req := httptest.NewRequest(http.MethodPost, "/plans/test-run-approve/approve", nil)
	req.SetPathValue("runId", "test-run-approve")
	w := httptest.NewRecorder()

	server.planApproveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "approved" {
		t.Errorf("expected status 'approved', got %s", response["status"])
	}

	// Verify the plan status was updated
	if server.plans["test-run-approve"].Status != "approved" {
		t.Error("expected plan status to be updated to 'approved'")
	}
}

// TestPlanFeedbackEndpoint tests sending feedback on a plan
func TestPlanFeedbackEndpoint(t *testing.T) {
	server := NewServer(0)
	testPlan := &Plan{
		RunID:   "test-run-feedback",
		Content: "# Feedback Test Plan",
		Status:  "pending",
		Created: time.Now(),
		Updated: time.Now(),
	}
	server.plans = map[string]*Plan{
		testPlan.RunID: testPlan,
	}

	payload := map[string]string{
		"feedback": "Please add more detail to step 2",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/plans/test-run-feedback/feedback", bytes.NewReader(body))
	req.SetPathValue("runId", "test-run-feedback")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.planFeedbackHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "feedback_received" {
		t.Errorf("expected status 'feedback_received', got %s", response["status"])
	}
}

// TestRunEventsSSE tests the run-specific SSE endpoint
func TestRunEventsSSE(t *testing.T) {
	server := NewServer(0)
	testRun := &Run{
		ID:      "test-run-sse",
		AgentID: "alpine-agent",
		Status:  "running",
		Issue:   "https://github.com/owner/repo/issues/sse",
		Created: time.Now(),
		Updated: time.Now(),
	}
	server.runs = map[string]*Run{
		testRun.ID: testRun,
	}

	// Create a request with a short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/runs/test-run-sse/events", nil)
	req = req.WithContext(ctx)
	req.SetPathValue("id", "test-run-sse")
	w := httptest.NewRecorder()

	// Start handler in goroutine since it blocks
	done := make(chan bool)
	go func() {
		server.runEventsHandler(w, req)
		done <- true
	}()

	// Wait for handler to complete
	select {
	case <-done:
		// Handler completed
	case <-time.After(200 * time.Millisecond):
		t.Fatal("handler did not complete in time")
	}

	// Check headers
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %s", w.Header().Get("Content-Type"))
	}

	// Should have received initial connection event
	body := w.Body.String()
	if !strings.Contains(body, "data:") {
		t.Error("expected SSE data in response")
	}
}

// TestInvalidMethods tests that endpoints return 405 for wrong HTTP methods
func TestInvalidMethods(t *testing.T) {
	server := NewServer(0)

	tests := []struct {
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{http.MethodPost, "/health", server.healthHandler},
		{http.MethodPost, "/agents/list", server.agentsListHandler},
		{http.MethodGet, "/agents/run", server.agentsRunHandler},
		{http.MethodPost, "/runs", server.runsListHandler},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.method, tt.path), func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status 405, got %d", w.Code)
			}
		})
	}
}

// TestJSONErrorResponses tests that error responses are properly formatted JSON
func TestJSONErrorResponses(t *testing.T) {
	server := NewServer(0)

	// Test with invalid JSON payload
	req := httptest.NewRequest(http.MethodPost, "/agents/run", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.agentsRunHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response["error"] == "" {
		t.Error("expected error message in response")
	}
}
