package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const controlPlaneTimeout = 30 * time.Second

type controlPlaneClient struct {
	baseURL    string
	httpClient *http.Client
}

func newControlPlaneClient(baseURL string) *controlPlaneClient {
	return &controlPlaneClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: controlPlaneTimeout,
		},
	}
}

type cpError struct {
	Message    string `json:"message"`
	ReasonCode string `json:"reason_code"`
	Retryable  bool   `json:"retryable"`
}

func (e *cpError) Error() string {
	return e.Message
}

type cpLaunchRequest struct {
	Repo          string `json:"repo"`
	ImageProfile  string `json:"image_profile"`
	Task          string `json:"task,omitempty"`
	ForceRecreate bool   `json:"force_recreate,omitempty"`
}

type cpLaunchResponse struct {
	Name           string         `json:"name"`
	State          lifecycleState `json:"state"`
	Repo           string         `json:"repo"`
	ImageProfile   string         `json:"image_profile"`
	ContainerClass string         `json:"container_class"`
	WebURL         string         `json:"web_url"`
	Reused         bool           `json:"reused"`
	Resumed        bool           `json:"resumed"`
	TaskAccepted   bool           `json:"task_accepted"`
	OperationID    string         `json:"operation_id"`
	Error          *cpError       `json:"error,omitempty"`
}

type cpSandboxListItem struct {
	Name         string         `json:"name"`
	Repo         string         `json:"repo"`
	ImageProfile string         `json:"image_profile"`
	State        lifecycleState `json:"state"`
	LastActivity string         `json:"last_activity"`
}

type cpListResponse struct {
	Sandboxes []cpSandboxListItem `json:"sandboxes"`
	Error     *cpError            `json:"error,omitempty"`
}

type cpStatusIdentity struct {
	Repo         string `json:"repo"`
	ImageProfile string `json:"image_profile"`
}

type cpStatusDurability struct {
	CheckpointID string `json:"checkpoint_id,omitempty"`
	Verified     bool   `json:"verified"`
	ContentHash  string `json:"content_hash,omitempty"`
}

type cpStatusRuntime struct {
	WebURL     string `json:"web_url"`
	ProbeState string `json:"probe_state"`
	ProbeAt    string `json:"probe_at"`
	ProbeStale bool   `json:"probe_stale"`
}

type cpStatusOperation struct {
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

type cpStatusTeardown struct {
	AutoTeardown bool     `json:"auto_teardown"`
	Blockers     []string `json:"blockers,omitempty"`
}

type cpStatusResponse struct {
	Name         string             `json:"name"`
	State        lifecycleState     `json:"state"`
	Identity     cpStatusIdentity   `json:"identity"`
	Durability   cpStatusDurability `json:"durability"`
	Runtime      cpStatusRuntime    `json:"runtime"`
	Operation    cpStatusOperation  `json:"operation"`
	Teardown     cpStatusTeardown   `json:"teardown"`
	LastActivity string             `json:"last_activity"`
	ErrorReason  string             `json:"error_reason,omitempty"`
	Error        *cpError           `json:"error,omitempty"`
}

type cpOpenURLResponse struct {
	Name   string   `json:"name"`
	WebURL string   `json:"web_url"`
	Error  *cpError `json:"error,omitempty"`
}

type cpTeardownRequest struct {
	Force bool `json:"force"`
}

type cpTeardownResponse struct {
	Name         string         `json:"name"`
	State        lifecycleState `json:"state"`
	CheckpointID string         `json:"checkpoint_id,omitempty"`
	Destroyed    bool           `json:"destroyed"`
	Error        *cpError       `json:"error,omitempty"`
}

type cpHealthResponse struct {
	Status string `json:"status"`
}

func (c *controlPlaneClient) LaunchSandbox(ctx context.Context, name string, req cpLaunchRequest) (*cpLaunchResponse, error) {
	var resp cpLaunchResponse
	if err := c.doPost(ctx, fmt.Sprintf("/v1/sandboxes/%s/launch", name), req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, cpErrToExitErr(resp.Error)
	}
	return &resp, nil
}

func (c *controlPlaneClient) ListSandboxes(ctx context.Context) (*cpListResponse, error) {
	var resp cpListResponse
	if err := c.doGet(ctx, "/v1/sandboxes", &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, cpErrToExitErr(resp.Error)
	}
	return &resp, nil
}

func (c *controlPlaneClient) GetSandboxStatus(ctx context.Context, name string) (*cpStatusResponse, error) {
	var resp cpStatusResponse
	if err := c.doGet(ctx, fmt.Sprintf("/v1/sandboxes/%s/status", name), &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, cpErrToExitErr(resp.Error)
	}
	return &resp, nil
}

func (c *controlPlaneClient) GetSandboxOpenURL(ctx context.Context, name string) (*cpOpenURLResponse, error) {
	var resp cpOpenURLResponse
	if err := c.doGet(ctx, fmt.Sprintf("/v1/sandboxes/%s/open-url", name), &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, cpErrToExitErr(resp.Error)
	}
	return &resp, nil
}

func (c *controlPlaneClient) TeardownSandbox(ctx context.Context, name string, force bool) (*cpTeardownResponse, error) {
	var resp cpTeardownResponse
	if err := c.doPost(ctx, fmt.Sprintf("/v1/sandboxes/%s/teardown", name), cpTeardownRequest{Force: force}, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, cpErrToExitErr(resp.Error)
	}
	return &resp, nil
}

func (c *controlPlaneClient) HealthCheck(ctx context.Context) error {
	var resp cpHealthResponse
	if err := c.doGet(ctx, "/healthz", &resp); err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("health check returned status: %s", resp.Status)
	}
	return nil
}

func (c *controlPlaneClient) doGet(ctx context.Context, path string, out interface{}) error {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("Accept", "application/json")
	return c.doRequest(req, out)
}

func (c *controlPlaneClient) doPost(ctx context.Context, path string, body interface{}, out interface{}) error {
	url := c.baseURL + path
	payload, err := json.Marshal(body)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to encode request body: %v", err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return sysErr(fmt.Sprintf("failed to create request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.doRequest(req, out)
}

func (c *controlPlaneClient) doRequest(req *http.Request, out interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return sysErrReason(fmt.Sprintf("control plane request failed: %v", err), "control_plane_unreachable", true)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to read response body: %v", err))
	}

	if resp.StatusCode >= 400 {
		var cpErr cpError
		if json.Unmarshal(data, &cpErr) == nil && cpErr.Message != "" {
			return cpErrToExitErr(&cpErr)
		}
		return sysErrReason(fmt.Sprintf("control plane error (status %d): %s", resp.StatusCode, string(data)), "control_plane_error", resp.StatusCode >= 500)
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return sysErr(fmt.Sprintf("failed to decode response: %v", err))
		}
	}
	return nil
}

func cpErrToExitErr(cpErr *cpError) error {
	if cpErr == nil {
		return nil
	}
	code := 1
	if cpErr.Retryable {
		code = 2
	}
	return &exitError{
		msg:        cpErr.Message,
		code:       code,
		reasonCode: cpErr.ReasonCode,
		retryable:  cpErr.Retryable,
	}
}
