package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestDockerHealthCheck(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: ""}, // docker info succeeds
		})
		err := dockerHealthCheck(context.Background(), "linux")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fail on linux", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult("Cannot connect to the Docker daemon"),
		})
		err := dockerHealthCheck(context.Background(), "linux")
		if err == nil {
			t.Fatal("expected error on linux when docker not running")
		}
		if !strings.Contains(err.Error(), "docker is not running") {
			t.Fatalf("error = %q, want to contain 'docker is not running'", err.Error())
		}
	})

	t.Run("darwin starts Docker Desktop", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true // suppress stderr output
		mockRun(t, []cmdResult{
			errResult("not running"), // initial docker info fails
			{stdout: ""},             // open -a Docker succeeds
			{stdout: ""},             // poll docker info succeeds
		})
		err := dockerHealthCheck(context.Background(), "darwin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("darwin launch fails", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		mockRun(t, []cmdResult{
			errResult("not running"),   // initial docker info fails
			errResult("app not found"), // open -a Docker fails
		})
		err := dockerHealthCheck(context.Background(), "darwin")
		if err == nil {
			t.Fatal("expected error when Docker Desktop launch fails")
		}
		if !strings.Contains(err.Error(), "failed to start Docker Desktop") {
			t.Fatalf("error = %q, want to contain 'failed to start Docker Desktop'", err.Error())
		}
	})

	t.Run("darwin poll timeout", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		// Pre-cancel context so the poll loop exits immediately via context.Done.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		mockRun(t, []cmdResult{
			errResult("not running"), // initial docker info fails
			{stdout: ""},             // open -a Docker succeeds
			// No more responses needed: cancelled context triggers Done branch
			// before the ticker fires.
		})
		err := dockerHealthCheck(ctx, "darwin")
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "did not become ready") {
			t.Fatalf("error = %q, want to contain 'did not become ready'", err.Error())
		}
	})
}

func TestDockerHealthCheck_DarwinPollSuccessNonJSON(t *testing.T) {
	resetFlags(t)
	jsonOutput = false // non-JSON mode to cover the wait/ready stderr output

	callCount := 0
	orig := run
	run = func(_ context.Context, name string, args ...string) (string, string, error) {
		callCount++
		switch callCount {
		case 1:
			// Initial docker info fails
			return "", "not running", fmt.Errorf("exit status 1")
		case 2:
			// open -a Docker succeeds
			return "", "", nil
		case 3:
			// First poll: still not ready (covers the dots/waiting output)
			return "", "still starting", fmt.Errorf("exit status 1")
		case 4:
			// Second poll: succeeds (covers "Docker Desktop is ready" output)
			return "", "", nil
		default:
			t.Fatalf("unexpected call %d", callCount)
			return "", "", nil
		}
	}
	t.Cleanup(func() { run = orig })

	err := dockerHealthCheck(context.Background(), "darwin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 4 {
		t.Fatalf("expected 4 calls, got %d", callCount)
	}
}

func TestDockerHealthCheck_DarwinPollTimeoutNonJSON(t *testing.T) {
	resetFlags(t)
	jsonOutput = false // non-JSON mode to cover the "\n" write on timeout

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel so pollCtx.Done() fires immediately.

	mockRun(t, []cmdResult{
		errResult("not running"), // initial docker info fails
		{stdout: ""},             // open -a Docker succeeds
	})

	err := dockerHealthCheck(ctx, "darwin")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "did not become ready") {
		t.Fatalf("error = %q, want to contain 'did not become ready'", err.Error())
	}
}

func TestCheckDuplicate(t *testing.T) {
	t.Run("no duplicate empty array", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "[]"},
		})
		err := checkDuplicate(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no duplicate empty string", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: ""},
		})
		err := checkDuplicate(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("exists", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: `[{"Name":"alpine-test"}]`},
		})
		err := checkDuplicate(context.Background(), "test")
		if err == nil {
			t.Fatal("expected error for existing environment")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Fatalf("error = %q, want to contain 'already exists'", err.Error())
		}
	})

	t.Run("command fails returns nil", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult("docker not available"),
		})
		// Should not return error (assumes no duplicate).
		err := checkDuplicate(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDiscoverContainer(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: `{"Name":"alpine-test-dev-1","Service":"dev"}`},
		})
		name, err := discoverContainer(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "alpine-test-dev-1" {
			t.Fatalf("container = %q, want %q", name, "alpine-test-dev-1")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: `{"Name":"alpine-test-cache-1","Service":"cache"}`},
		})
		_, err := discoverContainer(context.Background(), "test")
		if err == nil {
			t.Fatal("expected error when dev container not found")
		}
		if !strings.Contains(err.Error(), "dev container not found") {
			t.Fatalf("error = %q, want to contain 'dev container not found'", err.Error())
		}
	})

	t.Run("command fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			errResult("project not found"),
		})
		_, err := discoverContainer(context.Background(), "test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("error = %q, want to contain 'not found'", err.Error())
		}
	})

	t.Run("unparseable NDJSON line still works", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "not valid json\n" + `{"Name":"alpine-test-dev-1","Service":"dev"}`},
		})
		name, err := discoverContainer(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "alpine-test-dev-1" {
			t.Fatalf("container = %q, want %q", name, "alpine-test-dev-1")
		}
	})

	t.Run("blank lines in output", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "\n" + `{"Name":"alpine-test-dev-1","Service":"dev"}` + "\n\n"},
		})
		name, err := discoverContainer(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "alpine-test-dev-1" {
			t.Fatalf("container = %q, want %q", name, "alpine-test-dev-1")
		}
	})
}

func TestComposeUp(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		err := composeUp(context.Background(), "test", "/tmp/compose.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("network error")})
		err := composeUp(context.Background(), "test", "/tmp/compose.yml")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to start environment") {
			t.Fatalf("error = %q, want to contain 'failed to start environment'", err.Error())
		}
	})
}

func TestComposeDown(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		err := composeDown(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("timeout")})
		err := composeDown(context.Background(), "test")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to tear down environment") {
			t.Fatalf("error = %q, want to contain 'failed to tear down environment'", err.Error())
		}
	})
}

func TestImageExists(t *testing.T) {
	t.Run("true exits 0", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		exists, err := imageExists(context.Background(), "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Fatal("expected exists=true")
		}
	})

	t.Run("false exits non-zero", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("no such image")})
		exists, err := imageExists(context.Background(), "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Fatal("expected exists=false")
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		mockRun(t, []cmdResult{errResult("cancelled")})
		_, err := imageExists(ctx, "alpine-dev:abc")
		if err == nil {
			t.Fatal("expected error for cancelled context")
		}
		if !strings.Contains(err.Error(), "checking image") {
			t.Fatalf("error = %q, want to contain 'checking image'", err.Error())
		}
	})
}
