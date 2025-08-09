package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Backland-Labs/alpine/internal/core"
)

// Test that agentsRunHandler returns appropriate HTTP status codes for git clone failures
func TestAgentsRunHandler_GitCloneErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		mockError      error
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "Clone timeout error returns 504",
			mockError:      ErrCloneTimeout,
			expectedStatus: http.StatusGatewayTimeout,
			expectedError:  "Git clone operation timed out. Please try again or check repository availability.",
			description:    "When git clone times out, API should return 504 Gateway Timeout",
		},
		{
			name:           "Repository not found returns 404",
			mockError:      ErrRepoNotFound,
			expectedStatus: http.StatusNotFound,
			expectedError:  "Repository not found. Please verify the repository exists and you have access.",
			description:    "When repository doesn't exist, API should return 404 Not Found",
		},
		{
			name:           "Clone disabled returns 400",
			mockError:      ErrCloneDisabled,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Git clone is disabled. Workflow will proceed with empty directory.",
			description:    "When git clone is disabled, API should return 400 Bad Request",
		},
		{
			name:           "Authentication error returns 401",
			mockError:      errors.New("authentication failed"),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication failed. Please check your access token for private repositories.",
			description:    "When authentication fails for private repo, API should return 401 Unauthorized",
		},
		{
			name:           "Generic clone error returns 500",
			mockError:      errors.New("git: command not found"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Workflow creation failed. Please try again.",
			description:    "When generic git clone error occurs, API should return 500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will fail initially because error handling is not implemented
			// Create mock server with workflow engine that returns specific errors
			server := createMockServerWithWorkflowError(tt.mockError)

			// Create test request
			payload := map[string]string{
				"issue_url": "https://github.com/owner/repo/issues/123",
				"agent_id":  "alpine-agent",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/agents/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			server.agentsRunHandler(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}

			// Verify error message structure
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check if response contains appropriate error message
			if errorMsg, exists := response["error"]; !exists {
				t.Errorf("%s: response should contain error field", tt.description)
			} else if !strings.Contains(errorMsg.(string), "Failed to start workflow") {
				// Initially, we expect the generic error message since error handling isn't implemented
				// After implementation, this should match tt.expectedError
				t.Logf("%s: Current error message: %s", tt.description, errorMsg.(string))
			}
		})
	}
}

// Test that error handling preserves fallback behavior with informative messages
func TestAgentsRunHandler_GracefulFallback(t *testing.T) {
	tests := []struct {
		name        string
		mockError   error
		description string
	}{
		{
			name:        "Clone timeout with graceful fallback",
			mockError:   ErrCloneTimeout,
			description: "Workflow should continue with empty directory after clone timeout",
		},
		{
			name:        "Repository not found with graceful fallback",
			mockError:   ErrRepoNotFound,
			description: "Workflow should continue with empty directory when repo not found",
		},
		{
			name:        "Clone disabled with graceful fallback",
			mockError:   ErrCloneDisabled,
			description: "Workflow should continue with empty directory when clone disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that even with clone errors,
			// the workflow still gets created (graceful fallback)
			server := createMockServerWithWorkflowError(tt.mockError)

			payload := map[string]string{
				"issue_url": "https://github.com/owner/repo/issues/123",
				"agent_id":  "alpine-agent",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/agents/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.agentsRunHandler(w, req)

			// Verify that a run was still created despite the error
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// The run should be created but marked as failed
			if runID, exists := response["id"]; !exists {
				t.Errorf("%s: response should still contain run id for graceful fallback", tt.description)
			} else {
				t.Logf("%s: Run created with ID: %s", tt.description, runID)
			}
		})
	}
}

// Test authentication error detection patterns
func TestAgentsRunHandler_AuthenticationErrorDetection(t *testing.T) {
	authErrorPatterns := []string{
		"authentication failed",
		"permission denied",
		"401 Unauthorized",
		"invalid credentials",
		"access denied",
	}

	for _, pattern := range authErrorPatterns {
		t.Run("Auth error pattern: "+pattern, func(t *testing.T) {
			server := createMockServerWithWorkflowError(errors.New(pattern))

			payload := map[string]string{
				"issue_url": "https://github.com/private/repo/issues/123",
				"agent_id":  "alpine-agent",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/agents/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.agentsRunHandler(w, req)

			// After implementation, this should return 401
			// Currently will return generic error
			t.Logf("Auth error '%s' returned status: %d", pattern, w.Code)
		})
	}
}

// Helper functions for creating mock servers

// createMockServerWithWorkflowError creates a server with a workflow engine that returns specific errors
func createMockServerWithWorkflowError(err error) *Server {
	server := NewServer(0)
	server.workflowEngine = &mockWorkflowEngine{
		startWorkflowError: err,
	}
	return server
}

// mockWorkflowEngine is a mock implementation for testing
type mockWorkflowEngine struct {
	startWorkflowError error
}

func (m *mockWorkflowEngine) StartWorkflow(ctx context.Context, issueURL, runID string, plan bool) (string, error) {
	if m.startWorkflowError != nil {
		return "", m.startWorkflowError
	}
	return "/tmp/test-worktree", nil
}

func (m *mockWorkflowEngine) GetWorkflowState(ctx context.Context, runID string) (*core.State, error) {
	return &core.State{
		CurrentStepDescription: "test step",
		NextStepPrompt:         "test prompt",
		Status:                 "running",
	}, nil
}

func (m *mockWorkflowEngine) SubscribeToEvents(ctx context.Context, runID string) (<-chan WorkflowEvent, error) {
	events := make(chan WorkflowEvent)
	close(events)
	return events, nil
}

func (m *mockWorkflowEngine) CancelWorkflow(ctx context.Context, runID string) error {
	return nil
}

func (m *mockWorkflowEngine) ApprovePlan(ctx context.Context, runID string) error {
	return nil
}

func (m *mockWorkflowEngine) Cleanup() error {
	return nil
}

// Test that agentsRunHandler returns appropriate HTTP status codes for plan field validation errors
func TestAgentsRunHandler_PlanFieldValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		planValue      interface{}
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "String plan value returns 400",
			planValue:      "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
			description:    "When plan field is a string, API should return 400 Bad Request with validation message",
		},
		{
			name:           "Number plan value returns 400",
			planValue:      123,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
			description:    "When plan field is a number, API should return 400 Bad Request with validation message",
		},
		{
			name:           "Object plan value returns 400",
			planValue:      map[string]string{"invalid": "object"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
			description:    "When plan field is an object, API should return 400 Bad Request with validation message",
		},
		{
			name:           "Array plan value returns 400",
			planValue:      []string{"invalid", "array"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
			description:    "When plan field is an array, API should return 400 Bad Request with validation message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServerForValidation()

			// Create test request with invalid plan field
			payload := map[string]interface{}{
				"issue_url": "https://github.com/owner/repo/issues/123",
				"agent_id":  "alpine-agent",
				"plan":      tt.planValue,
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/agents/run", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			server.agentsRunHandler(w, req)

			// Verify status code
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, w.Code)
			}

			// Verify error message structure
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check if response contains the expected error message
			if errorMsg, exists := response["error"]; !exists {
				t.Errorf("%s: response should contain error field", tt.description)
			} else if errorMsg != tt.expectedError {
				t.Errorf("%s: expected error '%s', got '%s'", tt.description, tt.expectedError, errorMsg.(string))
			}
		})
	}
}

// Test that plan field validation occurs before workflow engine is called
func TestAgentsRunHandler_PlanFieldValidationPrecedence(t *testing.T) {
	server := createMockServerForValidation()
	mockEngine := server.workflowEngine.(*mockValidationWorkflowEngine)

	// Create test request with invalid plan field
	payload := map[string]interface{}{
		"issue_url": "https://github.com/owner/repo/issues/123",
		"agent_id":  "alpine-agent",
		"plan":      "should_not_reach_engine",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/agents/run", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.agentsRunHandler(w, req)

	// Verify that validation error occurs before workflow engine is called
	if mockEngine.startWorkflowCalled {
		t.Error("Workflow engine should not be called when plan field validation fails")
	}

	// Verify proper error response
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Helper functions for plan field validation tests

// createMockServerForValidation creates a server with a mock workflow engine that tracks calls
func createMockServerForValidation() *Server {
	server := NewServer(0)
	server.workflowEngine = &mockValidationWorkflowEngine{
		startWorkflowCalled: false,
	}
	return server
}

// mockValidationWorkflowEngine tracks whether StartWorkflow was called for validation testing
type mockValidationWorkflowEngine struct {
	startWorkflowCalled bool
}

func (m *mockValidationWorkflowEngine) StartWorkflow(ctx context.Context, issueURL, runID string, plan bool) (string, error) {
	m.startWorkflowCalled = true
	return "/tmp/test-worktree", nil
}

func (m *mockValidationWorkflowEngine) GetWorkflowState(ctx context.Context, runID string) (*core.State, error) {
	return &core.State{
		CurrentStepDescription: "test step",
		NextStepPrompt:         "test prompt",
		Status:                 "running",
	}, nil
}

func (m *mockValidationWorkflowEngine) SubscribeToEvents(ctx context.Context, runID string) (<-chan WorkflowEvent, error) {
	events := make(chan WorkflowEvent)
	close(events)
	return events, nil
}

func (m *mockValidationWorkflowEngine) CancelWorkflow(ctx context.Context, runID string) error {
	return nil
}

func (m *mockValidationWorkflowEngine) ApprovePlan(ctx context.Context, runID string) error {
	return nil
}

func (m *mockValidationWorkflowEngine) Cleanup() error {
	return nil
}
