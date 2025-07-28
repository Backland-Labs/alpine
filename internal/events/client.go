// Package events provides event emission functionality for Alpine HTTP server mode.
// It includes clients for posting events to UI endpoints following the ag-ui protocol.
package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client handles posting events to a UI endpoint following the ag-ui protocol.
// It provides both synchronous and asynchronous event posting with automatic retry logic.
type Client struct {
	endpoint   string
	runID      string
	httpClient *http.Client
}

const (
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 10 * time.Second

	// DefaultMaxRetries is the default number of retry attempts
	DefaultMaxRetries = 3

	// InitialBackoff is the initial retry backoff duration
	InitialBackoff = 100 * time.Millisecond
)

// NewClient creates a new event client that posts events to the specified endpoint.
// All events will include the provided runID in their data payload.
func NewClient(endpoint, runID string) *Client {
	return &Client{
		endpoint: endpoint,
		runID:    runID,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// PostEvent posts an event synchronously to the configured endpoint.
// It will retry up to DefaultMaxRetries times on failure with exponential backoff.
// The event will automatically include the runID and a timestamp.
func (c *Client) PostEvent(eventType string, eventData map[string]interface{}) error {
	event := c.formatEvent(eventType, eventData)
	return c.postWithRetry(event, DefaultMaxRetries)
}

// PostEventAsync posts an event asynchronously without waiting for response.
// Errors are silently ignored as the posting happens in a background goroutine.
// Use this for non-critical events where delivery is best-effort.
func (c *Client) PostEventAsync(eventType string, eventData map[string]interface{}) error {
	event := c.formatEvent(eventType, eventData)

	// Start goroutine for async posting
	go func() {
		// Ignore errors in async mode
		_ = c.postWithRetry(event, DefaultMaxRetries)
	}()

	return nil
}

// formatEvent creates an ag-ui protocol compliant event
func (c *Client) formatEvent(eventType string, eventData map[string]interface{}) map[string]interface{} {
	// Merge runId into event data
	data := make(map[string]interface{})
	if eventData != nil {
		for k, v := range eventData {
			data[k] = v
		}
	}
	data["runId"] = c.runID

	return map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}

// postWithRetry attempts to post an event with exponential backoff retry
func (c *Client) postWithRetry(event map[string]interface{}, maxAttempts int) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := c.post(event)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt < maxAttempts {
			// Exponential backoff: 100ms, 200ms, 400ms...
			backoff := InitialBackoff * time.Duration(attempt)
			time.Sleep(backoff)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}

// post sends a single event to the endpoint
func (c *Client) post(event map[string]interface{}) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

