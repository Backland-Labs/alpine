package main

import (
	"context"
	"strings"
	"testing"
)

func TestCopyPathToContainer(t *testing.T) {
	t.Run("success 2 calls", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{
			{stdout: ""}, // docker cp
			{stdout: ""}, // chown
		})
		err := copyPathToContainer(context.Background(), "ctr", "/host/path", "/dest/path")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(*calls) != 2 {
			t.Fatalf("expected 2 calls, got %d", len(*calls))
		}
		// First call: docker cp
		if (*calls)[0].name != "docker" || (*calls)[0].args[0] != "cp" {
			t.Errorf("call 0: expected docker cp, got %s %v", (*calls)[0].name, (*calls)[0].args)
		}
		// Second call: docker exec chown
		if (*calls)[1].name != "docker" || (*calls)[1].args[0] != "exec" {
			t.Errorf("call 1: expected docker exec, got %s %v", (*calls)[1].name, (*calls)[1].args)
		}
	})

	t.Run("cp fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult("no such file"),
		})
		err := copyPathToContainer(context.Background(), "ctr", "/src", "/dest")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "docker cp failed") {
			t.Fatalf("error = %q, want 'docker cp failed'", err.Error())
		}
	})

	t.Run("chown fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: ""},            // docker cp succeeds
			errResult("permission"), // chown fails
		})
		err := copyPathToContainer(context.Background(), "ctr", "/src", "/dest")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "chown failed") {
			t.Fatalf("error = %q, want 'chown failed'", err.Error())
		}
	})
}

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
