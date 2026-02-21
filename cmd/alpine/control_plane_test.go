package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestControlPlaneClient_LaunchSandbox(t *testing.T) {
	tests := []struct {
		name       string
		resp       cpLaunchResponse
		statusCode int
		wantErr    bool
		wantState  lifecycleState
	}{
		{
			name: "success",
			resp: cpLaunchResponse{
				Name:         "test",
				State:        stateRunning,
				Repo:         "https://github.com/example/repo",
				ImageProfile: "default",
				WebURL:       "https://example.com/sandbox/test",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantState:  stateRunning,
		},
		{
			name: "error_response",
			resp: cpLaunchResponse{
				Error: &cpError{
					Message:    "sandbox already exists",
					ReasonCode: "sandbox_exists",
					Retryable:  false,
				},
			},
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/sandboxes/test/launch" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.resp)
			}))
			defer server.Close()

			client := newControlPlaneClient(server.URL)
			resp, err := client.LaunchSandbox(context.Background(), "test", cpLaunchRequest{
				Repo:         "https://github.com/example/repo",
				ImageProfile: "default",
			})

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if resp.State != tt.wantState {
				t.Errorf("expected state %q, got %q", tt.wantState, resp.State)
			}
		})
	}
}

func TestControlPlaneClient_ListSandboxes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpListResponse{
			Sandboxes: []cpSandboxListItem{
				{Name: "alpha", State: stateRunning, Repo: "https://github.com/example/repo"},
				{Name: "beta", State: stateDestroyed, Repo: "https://github.com/example/other"},
			},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	resp, err := client.ListSandboxes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Sandboxes) != 2 {
		t.Errorf("expected 2 sandboxes, got %d", len(resp.Sandboxes))
	}
	if resp.Sandboxes[0].Name != "alpha" {
		t.Errorf("expected first sandbox to be 'alpha', got %q", resp.Sandboxes[0].Name)
	}
}

func TestControlPlaneClient_GetSandboxStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/test/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpStatusResponse{
			Name:  "test",
			State: stateRunning,
			Runtime: cpStatusRuntime{
				WebURL:     "https://example.com/sandbox/test",
				ProbeState: "running",
			},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	resp, err := client.GetSandboxStatus(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Name != "test" {
		t.Errorf("expected name 'test', got %q", resp.Name)
	}
	if resp.State != stateRunning {
		t.Errorf("expected state %q, got %q", stateRunning, resp.State)
	}
}

func TestControlPlaneClient_GetSandboxOpenURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/test/open-url" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpOpenURLResponse{
			Name:   "test",
			WebURL: "https://example.com/sandbox/test",
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	resp, err := client.GetSandboxOpenURL(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.WebURL != "https://example.com/sandbox/test" {
		t.Errorf("expected web URL 'https://example.com/sandbox/test', got %q", resp.WebURL)
	}
}

func TestControlPlaneClient_TeardownSandbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/test/teardown" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpTeardownResponse{
			Name:      "test",
			State:     stateDestroyed,
			Destroyed: true,
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	resp, err := client.TeardownSandbox(context.Background(), "test", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Destroyed {
		t.Error("expected destroyed to be true")
	}
}

func TestControlPlaneClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpHealthResponse{Status: "ok"})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	if err := client.HealthCheck(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestControlPlaneClient_Unreachable(t *testing.T) {
	client := newControlPlaneClient("http://127.0.0.1:1")
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Error("expected error for unreachable server")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Errorf("expected exitError, got %T", err)
		return
	}
	if exitErr.code != 2 {
		t.Errorf("expected exit code 2 (system error), got %d", exitErr.code)
	}
	if exitErr.reasonCode != "control_plane_unreachable" {
		t.Errorf("expected reason_code 'control_plane_unreachable', got %q", exitErr.reasonCode)
	}
}

func TestCpErrToExitErr(t *testing.T) {
	tests := []struct {
		name          string
		cpErr         *cpError
		wantCode      int
		wantReason    string
		wantRetryable bool
	}{
		{
			name:          "nil_error",
			cpErr:         nil,
			wantCode:      0,
			wantReason:    "",
			wantRetryable: false,
		},
		{
			name: "user_error",
			cpErr: &cpError{
				Message:    "not found",
				ReasonCode: "sandbox_not_found",
				Retryable:  false,
			},
			wantCode:      1,
			wantReason:    "sandbox_not_found",
			wantRetryable: false,
		},
		{
			name: "system_error_retryable",
			cpErr: &cpError{
				Message:    "temporary failure",
				ReasonCode: "temporary_error",
				Retryable:  true,
			},
			wantCode:      2,
			wantReason:    "temporary_error",
			wantRetryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cpErrToExitErr(tt.cpErr)
			if tt.cpErr == nil {
				if result != nil {
					t.Error("expected nil result for nil input")
				}
				return
			}
			exitErr, ok := result.(*exitError)
			if !ok {
				t.Errorf("expected exitError, got %T", result)
				return
			}
			if exitErr.code != tt.wantCode {
				t.Errorf("expected code %d, got %d", tt.wantCode, exitErr.code)
			}
			if exitErr.reasonCode != tt.wantReason {
				t.Errorf("expected reason %q, got %q", tt.wantReason, exitErr.reasonCode)
			}
			if exitErr.retryable != tt.wantRetryable {
				t.Errorf("expected retryable %t, got %t", tt.wantRetryable, exitErr.retryable)
			}
		})
	}
}

func TestRunLaunchControlPlane(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sandboxes/test-sandbox/launch" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cpLaunchResponse{
				Name:         "test-sandbox",
				State:        stateRunning,
				Repo:         "https://github.com/example/repo",
				ImageProfile: "default",
				WebURL:       "https://example.com/sandbox/test-sandbox",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test-sandbox", "https://github.com/example/repo", "default", "small")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLaunchControlPlane_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Error: &cpError{Message: "already exists", ReasonCode: "exists", Retryable: false},
		})
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test", "https://github.com/example/repo", "default", "small")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRunListControlPlane(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sandboxes" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cpListResponse{
				Sandboxes: []cpSandboxListItem{
					{Name: "sandbox1", State: stateRunning, Repo: "https://github.com/example/repo"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runListControlPlane(context.Background(), cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetOpenURLControlPlane(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sandboxes/test/open-url" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cpOpenURLResponse{
				Name:   "test",
				WebURL: "https://example.com/sandbox/test",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	url, err := getOpenURLControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if url != "https://example.com/sandbox/test" {
		t.Errorf("expected url 'https://example.com/sandbox/test', got %q", url)
	}
}

func TestRunStatusControlPlane(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sandboxes/test/status" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cpStatusResponse{
				Name:  "test",
				State: stateRunning,
				Identity: cpStatusIdentity{
					Repo:         "https://github.com/example/repo",
					ImageProfile: "default",
				},
				Runtime: cpStatusRuntime{
					WebURL:     "https://example.com/sandbox/test",
					ProbeState: "running",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runStatusControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunTeardownControlPlane(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sandboxes/test/teardown" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(cpTeardownResponse{
				Name:      "test",
				State:     stateDestroyed,
				Destroyed: true,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runTeardownControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCpError_Error(t *testing.T) {
	err := &cpError{Message: "test error message"}
	if err.Error() != "test error message" {
		t.Errorf("expected 'test error message', got %q", err.Error())
	}
}

func TestControlPlaneClient_HealthCheck_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpHealthResponse{Status: "degraded"})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Error("expected error for bad health status")
	}
}

func TestControlPlaneClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Error("expected error for server error")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Errorf("expected exitError, got %T", err)
		return
	}
	if exitErr.code != 2 {
		t.Errorf("expected exit code 2 (system error), got %d", exitErr.code)
	}
}

func TestControlPlaneClient_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestControlPlaneClient_GetSandboxStatus_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(cpStatusResponse{
			Error: &cpError{Message: "not found", ReasonCode: "not_found", Retryable: false},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.GetSandboxStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestControlPlaneClient_GetSandboxOpenURL_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(cpOpenURLResponse{
			Error: &cpError{Message: "not found", ReasonCode: "not_found", Retryable: false},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.GetSandboxOpenURL(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestControlPlaneClient_TeardownSandbox_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(cpTeardownResponse{
			Error: &cpError{Message: "cannot teardown", ReasonCode: "active", Retryable: false},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.TeardownSandbox(context.Background(), "test", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestControlPlaneClient_LaunchSandbox_ErrorInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Error: &cpError{Message: "failed", ReasonCode: "internal", Retryable: true},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.LaunchSandbox(context.Background(), "test", cpLaunchRequest{})
	if err == nil {
		t.Error("expected error")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Errorf("expected exitError, got %T", err)
		return
	}
	if exitErr.code != 2 {
		t.Errorf("expected exit code 2 (retryable), got %d", exitErr.code)
	}
}

func TestControlPlaneClient_ListSandboxes_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpListResponse{
			Error: &cpError{Message: "failed", ReasonCode: "internal", Retryable: false},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Error("expected error")
	}
}

func TestRunLaunchControlPlane_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Name:         "test",
			State:        stateRunning,
			Repo:         "https://github.com/example/repo",
			ImageProfile: "default",
			WebURL:       "https://example.com/test",
			Reused:       true,
			Resumed:      true,
			TaskAccepted: true,
		})
	}))
	defer server.Close()

	jsonOutput = true
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test", "https://github.com/example/repo", "default", "small")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunListControlPlane_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpListResponse{
			Sandboxes: []cpSandboxListItem{
				{Name: "test", State: stateRunning, Repo: "https://github.com/example/repo", ImageProfile: "default"},
			},
		})
	}))
	defer server.Close()

	jsonOutput = true
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runListControlPlane(context.Background(), cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunStatusControlPlane_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpStatusResponse{
			Name:  "test",
			State: stateRunning,
			Identity: cpStatusIdentity{
				Repo:         "https://github.com/example/repo",
				ImageProfile: "default",
			},
			Durability: cpStatusDurability{
				CheckpointID: "cp-123",
				Verified:     true,
			},
			Runtime: cpStatusRuntime{
				WebURL:     "https://example.com/test",
				ProbeState: "running",
			},
			Teardown: cpStatusTeardown{
				AutoTeardown: true,
				Blockers:     []string{"active_export"},
			},
		})
	}))
	defer server.Close()

	jsonOutput = true
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runStatusControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunTeardownControlPlane_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpTeardownResponse{
			Name:         "test",
			State:        stateDestroyed,
			CheckpointID: "cp-123",
			Destroyed:    true,
		})
	}))
	defer server.Close()

	jsonOutput = true
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runTeardownControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetOpenURLControlPlane_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(cpOpenURLResponse{
			Error: &cpError{Message: "not found", ReasonCode: "not_found", Retryable: false},
		})
	}))
	defer server.Close()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	_, err := getOpenURLControlPlane(context.Background(), cfg, "missing")
	if err == nil {
		t.Error("expected error")
	}
}

func TestRunListControlPlane_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpListResponse{Sandboxes: []cpSandboxListItem{}})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runListControlPlane(context.Background(), cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunStatusControlPlane_HumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpStatusResponse{
			Name:  "test",
			State: stateRunning,
			Identity: cpStatusIdentity{
				Repo:         "https://github.com/example/repo",
				ImageProfile: "default",
			},
			Durability: cpStatusDurability{
				CheckpointID: "cp-123",
				Verified:     true,
				ContentHash:  "hash123",
			},
			Runtime: cpStatusRuntime{
				WebURL:     "https://example.com/test",
				ProbeState: "running",
				ProbeStale: true,
			},
			Teardown: cpStatusTeardown{
				AutoTeardown: true,
				Blockers:     []string{"active_export"},
			},
			ErrorReason: "some_error",
		})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runStatusControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLaunchControlPlane_HumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Name:         "test",
			State:        stateRunning,
			Repo:         "https://github.com/example/repo",
			ImageProfile: "default",
			WebURL:       "https://example.com/test",
			Reused:       false,
			Resumed:      false,
			TaskAccepted: false,
		})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test", "https://github.com/example/repo", "default", "small")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunTeardownControlPlane_HumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpTeardownResponse{
			Name:         "test",
			State:        stateDestroyed,
			CheckpointID: "cp-123",
			Destroyed:    true,
		})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runTeardownControlPlane(context.Background(), cfg, "test")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestControlPlaneClient_GetSandboxStatus_ErrorInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpStatusResponse{
			Error: &cpError{Message: "failed", ReasonCode: "internal", Retryable: true},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.GetSandboxStatus(context.Background(), "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestControlPlaneClient_GetSandboxOpenURL_ErrorInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpOpenURLResponse{
			Error: &cpError{Message: "failed", ReasonCode: "internal", Retryable: false},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.GetSandboxOpenURL(context.Background(), "test")
	if err == nil {
		t.Error("expected error")
	}
}

func TestControlPlaneClient_TeardownSandbox_ErrorInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpTeardownResponse{
			Error: &cpError{Message: "failed", ReasonCode: "internal", Retryable: true},
		})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.TeardownSandbox(context.Background(), "test", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestControlPlaneClient_HealthCheck_Unreachable(t *testing.T) {
	client := newControlPlaneClient("http://127.0.0.1:1")
	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestControlPlaneClient_LaunchSandbox_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.LaunchSandbox(context.Background(), "test", cpLaunchRequest{})
	if err == nil {
		t.Error("expected error")
	}
}

func TestControlPlaneClient_LaunchSandbox_JSONEncodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{Name: "test", State: stateRunning})
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.LaunchSandbox(context.Background(), "test", cpLaunchRequest{
		Repo:         "https://github.com/example/repo",
		ImageProfile: "default",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestControlPlaneClient_DoGet_InvalidURL(t *testing.T) {
	client := newControlPlaneClient("://invalid-url")
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestControlPlaneClient_DoPost_InvalidURL(t *testing.T) {
	client := newControlPlaneClient("://invalid-url")
	_, err := client.LaunchSandbox(context.Background(), "test", cpLaunchRequest{})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestRunLaunchControlPlane_TaskAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Name:         "test",
			State:        stateRunning,
			Repo:         "https://github.com/example/repo",
			ImageProfile: "default",
			WebURL:       "https://example.com/test",
			TaskAccepted: true,
		})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()
	launchTask = "some task"
	defer func() { launchTask = "" }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test", "https://github.com/example/repo", "default", "small")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestControlPlaneClient_DoRequest_ReadBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newControlPlaneClient(server.URL)
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Error("expected error for body read issue")
	}
}

func TestRunLaunchControlPlane_Reused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Name:         "test",
			State:        stateRunning,
			Repo:         "https://github.com/example/repo",
			ImageProfile: "default",
			WebURL:       "https://example.com/test",
			Reused:       true,
		})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test", "https://github.com/example/repo", "default", "small")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunLaunchControlPlane_Resumed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cpLaunchResponse{
			Name:         "test",
			State:        stateRunning,
			Repo:         "https://github.com/example/repo",
			ImageProfile: "default",
			WebURL:       "https://example.com/test",
			Resumed:      true,
		})
	}))
	defer server.Close()

	jsonOutput = false
	defer func() { jsonOutput = false }()

	cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: server.URL}}
	err := runLaunchControlPlane(context.Background(), cfg, "test", "https://github.com/example/repo", "default", "small")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
