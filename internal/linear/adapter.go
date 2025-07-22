package linear

import (
	"context"

	"github.com/maxmcd/river/internal/workflow"
)

// WorkflowAdapter adapts the Linear Client to the workflow.LinearClient interface
type WorkflowAdapter struct {
	client Client
}

// NewWorkflowAdapter creates a new adapter that implements workflow.LinearClient
func NewWorkflowAdapter(apiKey string) (workflow.LinearClient, error) {
	client, err := NewClient(apiKey)
	if err != nil {
		return nil, err
	}
	return &WorkflowAdapter{client: client}, nil
}

// FetchIssue implements workflow.LinearClient.FetchIssue
func (a *WorkflowAdapter) FetchIssue(ctx context.Context, issueID string) (*workflow.LinearIssue, error) {
	issue, err := a.client.FetchIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}
	return issue.ToWorkflowIssue(), nil
}