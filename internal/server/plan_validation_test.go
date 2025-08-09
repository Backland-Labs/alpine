package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentsRunHandler_PlanFieldValidation(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "plan field with string value should return 400",
			payload: map[string]interface{}{
				"issue_url": "https://github.com/owner/repo/issues/123",
				"agent_id":  "alpine-agent",
				"plan":      "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(0)

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.agentsRunHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if response["error"] == "" {
				t.Error("expected error message in response")
			}

			if !strings.Contains(response["error"], tt.expectedError) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedError, response["error"])
			}
		})
	}
}

func TestAgentsRunHandler_AdditionalPlanValidation(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "plan field with number value should return 400",
			payload: map[string]interface{}{
				"issue_url": "https://github.com/owner/repo/issues/123",
				"agent_id":  "alpine-agent",
				"plan":      123,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
		},
		{
			name: "plan field with array value should return 400",
			payload: map[string]interface{}{
				"issue_url": "https://github.com/owner/repo/issues/123",
				"agent_id":  "alpine-agent",
				"plan":      []string{"invalid"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "plan field must be a boolean value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(0)

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.agentsRunHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]string
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if response["error"] == "" {
				t.Error("expected error message in response")
			}

			if !strings.Contains(response["error"], tt.expectedError) {
				t.Errorf("expected error message to contain '%s', got '%s'", tt.expectedError, response["error"])
			}
		})
	}
}
