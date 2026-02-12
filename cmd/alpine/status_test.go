package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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
		responses := []cmdResult{
			errResult("Cannot connect to Docker daemon"), // docker info fails
		}
		if runtime.GOOS == "darwin" {
			// On darwin, dockerHealthCheck tries to start Docker Desktop.
			responses = append(responses, errResult("app not found"))
		}
		mockRun(t, responses)
		err := runStatus(newCmd(), []string{"myenv"})
		if err == nil {
			t.Fatal("expected error when Docker is not running")
		}
		if !strings.Contains(err.Error(), "docker") && !strings.Contains(err.Error(), "Docker") {
			t.Errorf("expected docker-related error, got: %v", err)
		}
	})
}
