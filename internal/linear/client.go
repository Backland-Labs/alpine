package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/maxmcd/river/internal/workflow"
)

const (
	linearAPIURL = "https://api.linear.app/graphql"
	timeout      = 30 * time.Second
)

// Issue represents a Linear issue
type Issue struct {
	ID          string
	Identifier  string
	Title       string
	Description string
}

// ToWorkflowIssue converts Linear Issue to workflow.LinearIssue
func (i *Issue) ToWorkflowIssue() *workflow.LinearIssue {
	return &workflow.LinearIssue{
		ID:          i.Identifier,
		Title:       i.Title,
		Description: i.Description,
	}
}

// Client interface for Linear API operations
type Client interface {
	FetchIssue(ctx context.Context, issueID string) (*Issue, error)
}

// linearClient implements the Client interface
type linearClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Linear API client
func NewClient(apiKey string) (Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &linearClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: linearAPIURL,
	}, nil
}

// graphQLRequest represents a GraphQL request
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

// graphQLResponse represents a GraphQL response
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchIssue fetches an issue from Linear by ID or identifier
func (c *linearClient) FetchIssue(ctx context.Context, issueID string) (*Issue, error) {
	if issueID == "" {
		return nil, fmt.Errorf("issue ID is required")
	}

	query := `
		query GetIssue($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
			}
		}
	`

	req := graphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"id": issueID,
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/graphql", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var graphQLResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphQLResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(graphQLResp.Errors) > 0 {
		return nil, fmt.Errorf("%s", graphQLResp.Errors[0].Message)
	}

	var data struct {
		Issue *struct {
			ID          string  `json:"id"`
			Identifier  string  `json:"identifier"`
			Title       string  `json:"title"`
			Description *string `json:"description"`
		} `json:"issue"`
	}

	if err := json.Unmarshal(graphQLResp.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal issue data: %w", err)
	}

	if data.Issue == nil {
		return nil, fmt.Errorf("issue not found")
	}

	issue := &Issue{
		ID:         data.Issue.ID,
		Identifier: data.Issue.Identifier,
		Title:      data.Issue.Title,
	}

	if data.Issue.Description != nil {
		issue.Description = *data.Issue.Description
	}

	return issue, nil
}