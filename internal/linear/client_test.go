package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid API key",
			apiKey:    "lin_api_test123",
			wantError: false,
		},
		{
			name:      "empty API key",
			apiKey:    "",
			wantError: true,
			errorMsg:  "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey)
			
			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_FetchIssue(t *testing.T) {
	tests := []struct {
		name           string
		issueID        string
		mockResponse   interface{}
		mockStatusCode int
		expectedIssue  *Issue
		expectedError  string
	}{
		{
			name:    "successful fetch",
			issueID: "ABC-123",
			mockResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"issue": map[string]interface{}{
						"id":          "test-id-123",
						"identifier":  "ABC-123",
						"title":       "Test Issue",
						"description": "This is a test issue description",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedIssue: &Issue{
				ID:          "test-id-123",
				Identifier:  "ABC-123",
				Title:       "Test Issue",
				Description: "This is a test issue description",
			},
		},
		{
			name:    "issue not found",
			issueID: "INVALID-999",
			mockResponse: map[string]interface{}{
				"errors": []map[string]interface{}{
					{
						"message": "Issue not found",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedError:  "Issue not found",
		},
		{
			name:           "server error",
			issueID:        "ABC-123",
			mockResponse:   "Internal Server Error",
			mockStatusCode: http.StatusInternalServerError,
			expectedError:  "unexpected status code: 500",
		},
		{
			name:          "empty issue ID",
			issueID:       "",
			expectedError: "issue ID is required",
		},
		{
			name:    "null description",
			issueID: "ABC-123",
			mockResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"issue": map[string]interface{}{
						"id":          "test-id-123",
						"identifier":  "ABC-123",
						"title":       "Test Issue",
						"description": nil,
					},
				},
			},
			mockStatusCode: http.StatusOK,
			expectedIssue: &Issue{
				ID:          "test-id-123",
				Identifier:  "ABC-123",
				Title:       "Test Issue",
				Description: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/graphql", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

				// Verify GraphQL query
				var reqBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.NoError(t, err)
				
				query, ok := reqBody["query"].(string)
				require.True(t, ok)
				assert.Contains(t, query, "query GetIssue")
				assert.Contains(t, query, "$id: String!")
				
				variables, ok := reqBody["variables"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.issueID, variables["id"])

				// Send response
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockStatusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.mockResponse)
				} else {
					w.Write([]byte(tt.mockResponse.(string)))
				}
			}))
			defer server.Close()

			client, err := NewClient("test-api-key")
			require.NoError(t, err)
			
			// Override the base URL to use test server
			client.(*linearClient).baseURL = server.URL

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			issue, err := client.FetchIssue(ctx, tt.issueID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, issue)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIssue, issue)
			}
		})
	}
}

func TestClient_FetchIssue_ContextCancellation(t *testing.T) {
	// Test that context cancellation is properly handled
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient("test-api-key")
	require.NoError(t, err)
	client.(*linearClient).baseURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	issue, err := client.FetchIssue(ctx, "ABC-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, issue)
}

func TestIssue_ToWorkflowIssue(t *testing.T) {
	// Test conversion from Linear Issue to workflow.LinearIssue
	tests := []struct {
		name     string
		issue    *Issue
		expected struct {
			ID          string
			Title       string
			Description string
		}
	}{
		{
			name: "full issue",
			issue: &Issue{
				ID:          "test-id-123",
				Identifier:  "ABC-123",
				Title:       "Test Issue",
				Description: "Test description",
			},
			expected: struct {
				ID          string
				Title       string
				Description string
			}{
				ID:          "ABC-123",
				Title:       "Test Issue",
				Description: "Test description",
			},
		},
		{
			name: "issue with empty description",
			issue: &Issue{
				ID:          "test-id-123",
				Identifier:  "XYZ-456",
				Title:       "Another Issue",
				Description: "",
			},
			expected: struct {
				ID          string
				Title       string
				Description string
			}{
				ID:          "XYZ-456",
				Title:       "Another Issue",
				Description: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowIssue := tt.issue.ToWorkflowIssue()
			assert.Equal(t, tt.expected.ID, workflowIssue.ID)
			assert.Equal(t, tt.expected.Title, workflowIssue.Title)
			assert.Equal(t, tt.expected.Description, workflowIssue.Description)
		})
	}
}