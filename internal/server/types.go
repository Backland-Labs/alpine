package server

import "time"

// RunRequest represents the JSON request to start a new run
type RunRequest struct {
	Task          string `json:"task"`
	EventEndpoint string `json:"eventEndpoint,omitempty"`
}

// RunResponse represents the JSON response when creating a new run
type RunResponse struct {
	RunID string `json:"runId"`
}

// RunStatusResponse represents the JSON response for run status queries
type RunStatusResponse struct {
	RunID  string `json:"runId"`
	Status string `json:"status"`
	Task   string `json:"task"`
}

// Run represents an active or completed workflow run
type Run struct {
	ID            string
	Task          string
	Status        string
	EventEndpoint string
	StartTime     time.Time
	EndTime       *time.Time
	Error         error
}