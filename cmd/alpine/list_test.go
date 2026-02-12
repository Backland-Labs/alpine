package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Pure function tests (table-driven)
// ---------------------------------------------------------------------------

func TestParseContainerLabels(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "normal key=value pairs",
			input: "alpine.name=test,alpine.branch=main,alpine.managed=true",
			want:  map[string]string{"alpine.name": "test", "alpine.branch": "main", "alpine.managed": "true"},
		},
		{
			name:  "empty string",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "no equals sign in pair",
			input: "badlabel",
			want:  map[string]string{},
		},
		{
			name:  "single pair",
			input: "alpine.name=test",
			want:  map[string]string{"alpine.name": "test"},
		},
		{
			name:  "mixed valid and invalid",
			input: "badlabel,alpine.name=test",
			want:  map[string]string{"alpine.name": "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseContainerLabels(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("key %q = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseNDJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // expected number of containerInfo entries
	}{
		{
			name:  "normal multi-line",
			input: `{"Names":"c1","Labels":"","State":"running"}` + "\n" + `{"Names":"c2","Labels":"","State":"running"}`,
			want:  2,
		},
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "invalid JSON line skipped",
			input: "not json\n" + `{"Names":"c1","Labels":"","State":"running"}`,
			want:  1,
		},
		{
			name:  "blank lines skipped",
			input: "\n" + `{"Names":"c1","Labels":"","State":"running"}` + "\n\n",
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNDJSON(tt.input)
			if len(got) != tt.want {
				t.Fatalf("len = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestBuildEnvList(t *testing.T) {
	t.Run("normal with labels", func(t *testing.T) {
		projects := []composeLsEntry{
			{Name: "alpine-test", Status: "running(1)"},
		}
		containers := []containerInfo{
			{Names: "alpine-test-dev-1", Labels: "alpine.name=test,alpine.branch=main,alpine.created=2024-01-01T00:00:00Z", State: "running"},
		}
		envs := buildEnvList(projects, containers)
		if len(envs) != 1 {
			t.Fatalf("len = %d, want 1", len(envs))
		}
		if envs[0].Name != "test" {
			t.Errorf("Name = %q, want %q", envs[0].Name, "test")
		}
		if envs[0].Branch != "main" {
			t.Errorf("Branch = %q, want %q", envs[0].Branch, "main")
		}
		if envs[0].Status != "running" {
			t.Errorf("Status = %q, want %q", envs[0].Status, "running")
		}
		if envs[0].Created != "2024-01-01T00:00:00Z" {
			t.Errorf("Created = %q, want %q", envs[0].Created, "2024-01-01T00:00:00Z")
		}
	})

	t.Run("no containers", func(t *testing.T) {
		projects := []composeLsEntry{
			{Name: "alpine-orphan", Status: "exited(1)"},
		}
		envs := buildEnvList(projects, nil)
		if len(envs) != 1 {
			t.Fatalf("len = %d, want 1", len(envs))
		}
		if envs[0].Name != "orphan" {
			t.Errorf("Name = %q, want %q", envs[0].Name, "orphan")
		}
		if envs[0].Branch != "" {
			t.Errorf("Branch should be empty, got %q", envs[0].Branch)
		}
		if envs[0].Status != "stopped" {
			t.Errorf("Status = %q, want %q", envs[0].Status, "stopped")
		}
	})

	t.Run("container match by prefix", func(t *testing.T) {
		projects := []composeLsEntry{
			{Name: "alpine-noname", Status: "running(1)"},
		}
		// Container without alpine.name label -- matched by name prefix.
		containers := []containerInfo{
			{Names: "alpine-noname-dev-1", Labels: "alpine.branch=develop,alpine.created=2025-06-01T12:00:00Z", State: "running"},
		}
		envs := buildEnvList(projects, containers)
		if len(envs) != 1 {
			t.Fatalf("len = %d, want 1", len(envs))
		}
		if envs[0].Branch != "develop" {
			t.Errorf("Branch = %q, want %q", envs[0].Branch, "develop")
		}
		if envs[0].Created != "2025-06-01T12:00:00Z" {
			t.Errorf("Created = %q, want %q", envs[0].Created, "2025-06-01T12:00:00Z")
		}
	})

	t.Run("empty inputs", func(t *testing.T) {
		envs := buildEnvList(nil, nil)
		if len(envs) != 0 {
			t.Fatalf("len = %d, want 0", len(envs))
		}
	})
}

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"running(2)", "running"},
		{"exited(1)", "stopped"},
		{"dead(1)", "stopped"},
		{"running(1), exited(1)", "partial"},
		{"exited(1), running(1)", "partial"},
		{"created(1)", "created"},
		{"something-unknown", "something-unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeStatus(tt.input)
			if got != tt.want {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Mock-dependent tests (runList)
// ---------------------------------------------------------------------------

// newListCmd builds a minimal cobra.Command for calling runList directly.
func newListCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

func TestRunListNoProjects(t *testing.T) {
	t.Run("human output", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""},   // docker info (health check)
			{stdout: "[]"}, // docker compose ls
		})
		out := captureStdout(t, func() {
			if err := runList(newListCmd(), []string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "No active environments") {
			t.Errorf("expected 'No active environments', got: %s", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		mockRun(t, []cmdResult{
			{stdout: ""},   // docker info
			{stdout: "[]"}, // docker compose ls
		})
		out := captureStdout(t, func() {
			if err := runList(newListCmd(), []string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		var result []envInfo
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
		}
		if len(result) != 0 {
			t.Errorf("expected empty array, got %d items", len(result))
		}
	})
}

func TestRunListWithProjects(t *testing.T) {
	composeLsJSON := `[{"Name":"alpine-myenv","Status":"running(2)","ConfigFiles":"/tmp/compose.yaml"}]`
	containerNDJSON := `{"Names":"alpine-myenv-dev-1","Labels":"alpine.managed=true,alpine.name=myenv,alpine.branch=main,alpine.created=2025-01-15T10:00:00Z","State":"running"}`

	t.Run("human table output", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""},              // docker info
			{stdout: composeLsJSON},   // docker compose ls
			{stdout: containerNDJSON}, // docker ps
		})
		out := captureStdout(t, func() {
			if err := runList(newListCmd(), []string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		for _, hdr := range []string{"NAME", "BRANCH", "STATUS", "CREATED"} {
			if !strings.Contains(out, hdr) {
				t.Errorf("expected header %q in output, got: %s", hdr, out)
			}
		}
		if !strings.Contains(out, "myenv") {
			t.Errorf("expected 'myenv' in output, got: %s", out)
		}
		if !strings.Contains(out, "running") {
			t.Errorf("expected 'running' in output, got: %s", out)
		}
	})

	t.Run("json output", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		mockRun(t, []cmdResult{
			{stdout: ""},              // docker info
			{stdout: composeLsJSON},   // docker compose ls
			{stdout: containerNDJSON}, // docker ps
		})
		out := captureStdout(t, func() {
			if err := runList(newListCmd(), []string{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		var result []envInfo
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 env, got %d", len(result))
		}
		e := result[0]
		if e.Name != "myenv" {
			t.Errorf("Name = %q, want %q", e.Name, "myenv")
		}
		if e.Branch != "main" {
			t.Errorf("Branch = %q, want %q", e.Branch, "main")
		}
		if e.Status != "running" {
			t.Errorf("Status = %q, want %q", e.Status, "running")
		}
		if e.Created != "2025-01-15T10:00:00Z" {
			t.Errorf("Created = %q, want %q", e.Created, "2025-01-15T10:00:00Z")
		}
	})
}

func TestRunListDockerNotRunning(t *testing.T) {
	resetFlags(t)
	// On macOS (darwin), dockerHealthCheck attempts to launch Docker Desktop
	// after the initial docker info fails, so we need a second error response
	// for the "open -a Docker" call.
	mockRun(t, []cmdResult{
		errResult("Cannot connect to the Docker daemon"), // docker info fails
		errResult("open -a Docker failed"),               // open -a Docker fails
	})
	err := runList(newListCmd(), []string{})
	if err == nil {
		t.Fatal("expected error when Docker is not running")
	}
	if !strings.Contains(err.Error(), "docker") && !strings.Contains(err.Error(), "Docker") {
		t.Errorf("expected docker-related error, got: %v", err)
	}
}

func TestRunListComposeLsFails(t *testing.T) {
	resetFlags(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                      // docker info succeeds
		errResult("compose ls not found"), // docker compose ls fails
	})
	err := runList(newListCmd(), []string{})
	if err == nil {
		t.Fatal("expected error when compose ls fails")
	}
	if !strings.Contains(err.Error(), "listing environments") {
		t.Errorf("expected 'listing environments' in error, got: %v", err)
	}
}

func TestRunListDockerPsFailsNonFatal(t *testing.T) {
	resetFlags(t)
	jsonOutput = true
	composeLsJSON := `[{"Name":"alpine-resilient","Status":"running(1)","ConfigFiles":""}]`
	mockRun(t, []cmdResult{
		{stdout: ""},            // docker info
		{stdout: composeLsJSON}, // docker compose ls
		errResult("ps failed"),  // docker ps fails (non-fatal)
	})
	out := captureStdout(t, func() {
		if err := runList(newListCmd(), []string{}); err != nil {
			t.Fatalf("docker ps failure should be non-fatal, got: %v", err)
		}
	})
	var result []envInfo
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\noutput: %s", err, out)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 env, got %d", len(result))
	}
	if result[0].Name != "resilient" {
		t.Errorf("Name = %q, want %q", result[0].Name, "resilient")
	}
	if result[0].Branch != "" {
		t.Errorf("Branch should be empty when ps fails, got %q", result[0].Branch)
	}
	if result[0].Created != "" {
		t.Errorf("Created should be empty when ps fails, got %q", result[0].Created)
	}
}

// ---------------------------------------------------------------------------
// Compose ls returns malformed JSON (list.go lines 180-182)
// ---------------------------------------------------------------------------

func TestRunListComposeLsMalformedJSON(t *testing.T) {
	resetFlags(t)
	mockRun(t, []cmdResult{
		{stdout: ""},                      // docker info (health check)
		{stdout: "not valid json at all"}, // docker compose ls returns garbage
	})
	err := runList(newListCmd(), []string{})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parsing compose output") {
		t.Fatalf("error = %q, want to contain 'parsing compose output'", err.Error())
	}
}
