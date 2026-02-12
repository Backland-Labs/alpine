package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInspectContainer(t *testing.T) {
	t.Run("success returns trimmed output", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "  running\n"},
		})
		got, err := inspectContainer(context.Background(), "mycontainer", "{{.State.Status}}")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "running" {
			t.Fatalf("got %q, want %q", got, "running")
		}
	})

	t.Run("error returns error", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult("no such container"),
		})
		_, err := inspectContainer(context.Background(), "mycontainer", "{{.State.Status}}")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCheckClaudeProcess(t *testing.T) {
	t.Run("not running state", func(t *testing.T) {
		// Container is not running -- no run calls expected.
		mockRun(t, []cmdResult{})
		running, code := checkClaudeProcess(context.Background(), "ctr", "exited")
		if running {
			t.Fatal("expected running=false")
		}
		if code != nil {
			t.Fatal("expected code=nil")
		}
	})

	t.Run("claude running", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "123"}, // pgrep succeeds
		})
		running, code := checkClaudeProcess(context.Background(), "ctr", "running")
		if !running {
			t.Fatal("expected running=true")
		}
		if code != nil {
			t.Fatal("expected code=nil when running")
		}
	})

	t.Run("exited with code", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult(""), // pgrep fails
			{stdout: "0"}, // cat exit code file succeeds
		})
		running, code := checkClaudeProcess(context.Background(), "ctr", "running")
		if running {
			t.Fatal("expected running=false")
		}
		if code == nil {
			t.Fatal("expected non-nil exit code")
		}
		if *code != 0 {
			t.Fatalf("code = %d, want 0", *code)
		}
	})

	t.Run("exited no code file", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult(""), // pgrep fails
			errResult(""), // cat fails
		})
		running, code := checkClaudeProcess(context.Background(), "ctr", "running")
		if running {
			t.Fatal("expected running=false")
		}
		if code != nil {
			t.Fatal("expected code=nil when no exit code file")
		}
	})

	t.Run("bad code file content", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult(""),   // pgrep fails
			{stdout: "abc"}, // cat returns non-numeric content
		})
		running, code := checkClaudeProcess(context.Background(), "ctr", "running")
		if running {
			t.Fatal("expected running=false")
		}
		if code != nil {
			t.Fatal("expected code=nil when code file is not numeric")
		}
	})
}

func TestRunStatus(t *testing.T) {
	// newCmd creates a minimal cobra.Command for calling runStatus directly.
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())
		return cmd
	}

	t.Run("happy path human", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""}, // 0: docker info
			{stdout: `{"Name":"alpine-myenv-dev-1","Service":"dev"}`}, // 1: compose ps
			{stdout: "running"},              // 2: inspect state
			{stdout: "feature/myenv"},        // 3: inspect branch label
			{stdout: "2024-01-15T10:30:00Z"}, // 4: inspect created label
			{stdout: "42"},                   // 5: pgrep claude
		})
		out := captureStdout(t, func() {
			err := runStatus(newCmd(), []string{"myenv"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		for _, want := range []string{
			"Environment: myenv",
			"Container:   alpine-myenv-dev-1",
			"State:       running",
			"Branch:      feature/myenv",
			"Claude:      running",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q\ngot:\n%s", want, out)
			}
		}
	})

	t.Run("happy path json", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		mockRun(t, []cmdResult{
			{stdout: ""}, // 0: docker info
			{stdout: `{"Name":"alpine-myenv-dev-1","Service":"dev"}`}, // 1: compose ps
			{stdout: "running"},              // 2: inspect state
			{stdout: "feature/myenv"},        // 3: inspect branch label
			{stdout: "2024-01-15T10:30:00Z"}, // 4: inspect created label
			{stdout: "42"},                   // 5: pgrep claude
		})
		out := captureStdout(t, func() {
			err := runStatus(newCmd(), []string{"myenv"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		var status statusOutput
		if err := json.Unmarshal([]byte(out), &status); err != nil {
			t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
		}
		if status.Name != "myenv" {
			t.Errorf("name = %q, want %q", status.Name, "myenv")
		}
		if status.Container != "alpine-myenv-dev-1" {
			t.Errorf("container = %q, want %q", status.Container, "alpine-myenv-dev-1")
		}
		if status.State != "running" {
			t.Errorf("state = %q, want %q", status.State, "running")
		}
		if status.Branch != "feature/myenv" {
			t.Errorf("branch = %q, want %q", status.Branch, "feature/myenv")
		}
		if status.Created != "2024-01-15T10:30:00Z" {
			t.Errorf("created = %q, want %q", status.Created, "2024-01-15T10:30:00Z")
		}
		if !status.ClaudeRunning {
			t.Error("claude_running = false, want true")
		}
		if status.ClaudeExitCode != nil {
			t.Errorf("claude_exit_code = %v, want nil", *status.ClaudeExitCode)
		}
	})

	t.Run("not found human", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""},           // 0: docker info
			errResult("not found"), // 1: compose ps fails
		})
		err := runStatus(newCmd(), []string{"myenv"})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("error = %q, want to contain 'not found'", err.Error())
		}
	})

	t.Run("not found json", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		mockRun(t, []cmdResult{
			{stdout: ""},           // 0: docker info
			errResult("not found"), // 1: compose ps fails
		})
		out := captureStdout(t, func() {
			err := runStatus(newCmd(), []string{"myenv"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		var status statusOutput
		if err := json.Unmarshal([]byte(out), &status); err != nil {
			t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
		}
		if status.State != "not_found" {
			t.Errorf("state = %q, want %q", status.State, "not_found")
		}
	})

	t.Run("inspect fails", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""}, // 0: docker info
			{stdout: `{"Name":"alpine-myenv-dev-1","Service":"dev"}`}, // 1: compose ps
			errResult("inspect failed"),                               // 2: inspect state fails
		})
		err := runStatus(newCmd(), []string{"myenv"})
		if err == nil {
			t.Fatal("expected error when inspect fails")
		}
		if !strings.Contains(err.Error(), "failed to inspect") {
			t.Fatalf("error = %q, want to contain 'failed to inspect'", err.Error())
		}
	})

	t.Run("claude not running", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""}, // 0: docker info
			{stdout: `{"Name":"alpine-myenv-dev-1","Service":"dev"}`}, // 1: compose ps
			{stdout: "running"}, // 2: inspect state
			{stdout: ""},        // 3: inspect branch (empty)
			{stdout: ""},        // 4: inspect created (empty)
			errResult(""),       // 5: pgrep fails
			errResult(""),       // 6: cat exit code fails
		})
		out := captureStdout(t, func() {
			err := runStatus(newCmd(), []string{"myenv"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "Claude:      not running") {
			t.Fatalf("expected 'Claude:      not running'\ngot:\n%s", out)
		}
	})

	t.Run("claude exited with code", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		mockRun(t, []cmdResult{
			{stdout: ""}, // 0: docker info
			{stdout: `{"Name":"alpine-myenv-dev-1","Service":"dev"}`}, // 1: compose ps
			{stdout: "running"},              // 2: inspect state
			{stdout: "feature/myenv"},        // 3: inspect branch
			{stdout: "2024-01-15T10:30:00Z"}, // 4: inspect created
			errResult(""),                    // 5: pgrep fails
			{stdout: "1"},                    // 6: cat exit code returns 1
		})
		out := captureStdout(t, func() {
			err := runStatus(newCmd(), []string{"myenv"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		if !strings.Contains(out, "Claude:      exited (code 1)") {
			t.Fatalf("expected 'Claude:      exited (code 1)'\ngot:\n%s", out)
		}
	})

	t.Run("docker not running", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = false
		// On darwin, dockerHealthCheck tries docker info then open -a Docker.
		mockRun(t, []cmdResult{
			errResult("Cannot connect to Docker daemon"), // docker info fails
			errResult("app not found"),                   // open -a Docker fails (darwin)
		})
		err := runStatus(newCmd(), []string{"myenv"})
		if err == nil {
			t.Fatal("expected error when Docker is not running")
		}
		if !strings.Contains(err.Error(), "docker") && !strings.Contains(err.Error(), "Docker") {
			t.Errorf("expected docker-related error, got: %v", err)
		}
	})
}
