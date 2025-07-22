package linear

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockClient is a mock implementation of the Client interface
type mockClient struct {
	mock.Mock
}

func (m *mockClient) FetchIssue(ctx context.Context, issueID string) (*Issue, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Issue), args.Error(1)
}

func TestNewWorkflowAdapter(t *testing.T) {
	// Test successful creation
	adapter, err := NewWorkflowAdapter("test-api-key")
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Test with empty API key
	adapter, err = NewWorkflowAdapter("")
	assert.Error(t, err)
	assert.Nil(t, adapter)
	assert.Contains(t, err.Error(), "API key is required")
}

func TestWorkflowAdapter_FetchIssue(t *testing.T) {
	tests := []struct {
		name          string
		issueID       string
		mockIssue     *Issue
		mockError     error
		expectedError string
	}{
		{
			name:    "successful fetch",
			issueID: "ABC-123",
			mockIssue: &Issue{
				ID:          "test-id",
				Identifier:  "ABC-123",
				Title:       "Test Issue",
				Description: "Test description",
			},
			mockError: nil,
		},
		{
			name:          "fetch error",
			issueID:       "XYZ-999",
			mockIssue:     nil,
			mockError:     errors.New("issue not found"),
			expectedError: "issue not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := new(mockClient)
			adapter := &WorkflowAdapter{client: mockClient}

			ctx := context.Background()
			mockClient.On("FetchIssue", ctx, tt.issueID).Return(tt.mockIssue, tt.mockError)

			// Call FetchIssue
			workflowIssue, err := adapter.FetchIssue(ctx, tt.issueID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, workflowIssue)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, workflowIssue)
				assert.Equal(t, tt.mockIssue.Identifier, workflowIssue.ID)
				assert.Equal(t, tt.mockIssue.Title, workflowIssue.Title)
				assert.Equal(t, tt.mockIssue.Description, workflowIssue.Description)
			}

			mockClient.AssertExpectations(t)
		})
	}
}