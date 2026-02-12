package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Pure function tests (no mocking needed)
// ---------------------------------------------------------------------------

func TestGenerateDockerfile(t *testing.T) {
	tests := []struct {
		name      string
		baseImage string
		markers   []string
	}{
		{
			name:      "ubuntu 24.04",
			baseImage: "ubuntu:24.04",
			markers: []string{
				"FROM ubuntu:24.04",
				"apt-get update",
				"apt-get install -y --no-install-recommends",
				"git",
				"useradd -m -s /bin/bash claude",
				"ssh-keyscan",
				"credential.helper",
				"USER claude",
				"WORKDIR /workspace",
				"claude.ai/install.sh",
			},
		},
		{
			name:      "different base image",
			baseImage: "node:20-bookworm",
			markers: []string{
				"FROM node:20-bookworm",
				"apt-get",
				"useradd",
				"git",
				"credential.helper",
				"USER claude",
				"WORKDIR /workspace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			df := string(generateDockerfile(tt.baseImage))
			for _, m := range tt.markers {
				if !strings.Contains(df, m) {
					t.Errorf("Dockerfile missing %q", m)
				}
			}
		})
	}
}

func TestDockerfileHash(t *testing.T) {
	df1 := generateDockerfile("ubuntu:24.04")
	df2 := generateDockerfile("ubuntu:22.04")

	h1 := dockerfileHash(df1)
	h2 := dockerfileHash(df2)

	// Length must be 16 hex chars.
	if len(h1) != 16 {
		t.Fatalf("hash length = %d, want 16", len(h1))
	}

	// Deterministic: same input -> same hash.
	if dockerfileHash(df1) != h1 {
		t.Fatal("hash is not deterministic")
	}

	// Different input -> different hash.
	if h1 == h2 {
		t.Fatal("different inputs produced same hash")
	}
}

func TestLoadDotEnv(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		preSet   map[string]string // env vars to set before loading
		wantVars map[string]string
		wantErr  bool
	}{
		{
			name:     "basic key=value",
			content:  "FOO=bar\nBAZ=qux\n",
			wantVars: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:     "comments and blanks skipped",
			content:  "# comment\n\n   \nFOO=bar\n",
			wantVars: map[string]string{"FOO": "bar"},
		},
		{
			name:     "export prefix stripped",
			content:  "export FOO=bar\n",
			wantVars: map[string]string{"FOO": "bar"},
		},
		{
			name:     "double quoted values",
			content:  `FOO="hello world"` + "\n",
			wantVars: map[string]string{"FOO": "hello world"},
		},
		{
			name:     "single quoted values",
			content:  `FOO='hello world'` + "\n",
			wantVars: map[string]string{"FOO": "hello world"},
		},
		{
			name:     "existing env not overwritten",
			content:  "FOO=new\n",
			preSet:   map[string]string{"FOO": "existing"},
			wantVars: map[string]string{"FOO": "existing"},
		},
		{
			name:     "lines without = skipped",
			content:  "NOEQUALS\nFOO=bar\n",
			wantVars: map[string]string{"FOO": "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean env for each sub-test.
			for k := range tt.wantVars {
				t.Setenv(k, "")
				os.Unsetenv(k)
			}
			for k, v := range tt.preSet {
				t.Setenv(k, v)
			}

			dir := t.TempDir()
			path := filepath.Join(dir, ".env")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("writing .env: %v", err)
			}

			err := loadDotEnv(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, want := range tt.wantVars {
				got := os.Getenv(k)
				if got != want {
					t.Errorf("env %s = %q, want %q", k, got, want)
				}
			}
		})
	}

	t.Run("file not found returns error", func(t *testing.T) {
		err := loadDotEnv("/nonexistent/path/.env")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestExecError(t *testing.T) {
	t.Run("with stderr", func(t *testing.T) {
		e := &ExecError{Command: "docker build", Stderr: "no space left on device", Err: fmt.Errorf("exit 1")}
		msg := e.Error()
		if !strings.Contains(msg, "docker build") {
			t.Errorf("Error() missing command, got %q", msg)
		}
		if !strings.Contains(msg, "no space left on device") {
			t.Errorf("Error() missing stderr, got %q", msg)
		}
		if !strings.Contains(msg, "stderr:") {
			t.Errorf("Error() missing 'stderr:' prefix, got %q", msg)
		}
	})

	t.Run("without stderr", func(t *testing.T) {
		e := &ExecError{Command: "docker build", Err: fmt.Errorf("exit 1")}
		msg := e.Error()
		if !strings.Contains(msg, "docker build") {
			t.Errorf("Error() missing command, got %q", msg)
		}
		if strings.Contains(msg, "stderr") {
			t.Errorf("Error() should not mention stderr when empty, got %q", msg)
		}
	})

	t.Run("unwrap returns inner error", func(t *testing.T) {
		inner := fmt.Errorf("inner error")
		e := &ExecError{Command: "cmd", Err: inner}
		if e.Unwrap() != inner {
			t.Fatal("Unwrap() did not return inner error")
		}
	})
}

func TestGenerateComposeYAML(t *testing.T) {
	baseCfg := &Config{BaseImage: "ubuntu:24.04"}

	t.Run("darwin SSH path", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "/run/host-services/ssh-auth.sock") {
			t.Error("darwin should use /run/host-services/ssh-auth.sock")
		}
	})

	t.Run("linux SSH path uses SSH_AUTH_SOCK", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "/tmp/test-ssh.sock")
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "linux", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "/tmp/test-ssh.sock") {
			t.Error("linux should use SSH_AUTH_SOCK value")
		}
	})

	t.Run("linux SSH fallback when unset", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "linux", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "/tmp/ssh-agent.sock") {
			t.Error("linux with no SSH_AUTH_SOCK should fallback to /tmp/ssh-agent.sock")
		}
	})

	t.Run("with postgres service", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "db:") {
			t.Error("postgres should produce db: service alias")
		}
		if !strings.Contains(s, "CMD-SHELL") {
			t.Error("postgres healthcheck should use CMD-SHELL")
		}
		if !strings.Contains(s, "pg_isready") {
			t.Error("postgres healthcheck should use pg_isready")
		}
		if !strings.Contains(s, "tmpfs") {
			t.Error("postgres should have tmpfs mount")
		}
		if !strings.Contains(s, "POSTGRES_HOST_AUTH_METHOD=trust") {
			t.Error("postgres should have trust auth method")
		}
	})

	t.Run("with redis service", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"redis"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "cache:") {
			t.Error("redis should produce cache: service alias")
		}
		if !strings.Contains(s, `"CMD"`) {
			t.Error("redis healthcheck should use CMD format")
		}
		if !strings.Contains(s, "redis-cli") {
			t.Error("redis healthcheck should use redis-cli")
		}
		if !strings.Contains(s, "command:") {
			t.Error("redis should have ExtraCmd (command:)")
		}
		if !strings.Contains(s, `--save ""`) {
			t.Error("redis command should contain --save \"\"")
		}
	})

	t.Run("both services", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres", "redis"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "db:") || !strings.Contains(s, "cache:") {
			t.Error("both service aliases should be present")
		}
	})

	t.Run("unsupported service returns error", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"mysql"}}
		_, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err == nil {
			t.Fatal("expected error for unsupported service")
		}
		if !strings.Contains(err.Error(), "unsupported service") {
			t.Fatalf("error = %q, want to contain 'unsupported service'", err.Error())
		}
	})

	t.Run("labels present", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "myenv", "feat", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, `alpine.managed: "true"`) {
			t.Error("missing alpine.managed label")
		}
		if !strings.Contains(s, `alpine.name: "myenv"`) {
			t.Error("missing alpine.name label")
		}
		if !strings.Contains(s, `alpine.branch: "feat"`) {
			t.Error("missing alpine.branch label")
		}
	})

	t.Run("cap_drop present", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "cap_drop:") {
			t.Error("missing cap_drop directive")
		}
		if !strings.Contains(s, "ALL") {
			t.Error("cap_drop should include ALL")
		}
		if !strings.Contains(s, "no-new-privileges:true") {
			t.Error("missing no-new-privileges security opt")
		}
	})

	t.Run("passthrough env syntax only", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		// Secrets must use passthrough syntax (no "=value" after the key).
		for _, envVar := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
			if !strings.Contains(s, "- "+envVar) {
				t.Errorf("missing passthrough env var %s", envVar)
			}
			// Must NOT contain a literal secret value.
			if strings.Contains(s, envVar+"=sk-") {
				t.Errorf("env var %s appears to contain a literal secret", envVar)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Mock-dependent tests
// ---------------------------------------------------------------------------

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

func TestGitClone(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		err := gitClone(context.Background(), "container", "git@github.com:u/r.git", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("authentication failed")})
		err := gitClone(context.Background(), "container", "git@github.com:u/r.git", "main")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "git clone failed") {
			t.Fatalf("error = %q, want to contain 'git clone failed'", err.Error())
		}
	})
}

func TestGitCreateBranch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		err := gitCreateBranch(context.Background(), "container", "my-feature")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("branch already exists")})
		err := gitCreateBranch(context.Background(), "container", "my-feature")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "git checkout -b failed") {
			t.Fatalf("error = %q, want to contain 'git checkout -b failed'", err.Error())
		}
	})
}

func TestGitConfigureUser(t *testing.T) {
	t.Run("success 4 calls", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{
			{stdout: "Test User"},     // host: git config user.name
			{stdout: "test@test.com"}, // host: git config user.email
			{stdout: ""},              // container: git config user.name
			{stdout: ""},              // container: git config user.email
		})
		err := gitConfigureUser(context.Background(), "container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(*calls) != 4 {
			t.Fatalf("expected 4 calls, got %d", len(*calls))
		}
		// Verify host reads use "git" directly.
		if (*calls)[0].name != "git" {
			t.Errorf("call 0: expected git, got %s", (*calls)[0].name)
		}
		if (*calls)[1].name != "git" {
			t.Errorf("call 1: expected git, got %s", (*calls)[1].name)
		}
		// Verify container writes use "docker exec".
		if (*calls)[2].name != "docker" {
			t.Errorf("call 2: expected docker, got %s", (*calls)[2].name)
		}
		if (*calls)[3].name != "docker" {
			t.Errorf("call 3: expected docker, got %s", (*calls)[3].name)
		}
	})

	t.Run("host config missing uses fallback", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{
			errResult("not set"), // git config user.name fails
			errResult("not set"), // git config user.email fails
			{stdout: ""},         // docker exec git config user.name "alpine"
			{stdout: ""},         // docker exec git config user.email "alpine@localhost"
		})
		err := gitConfigureUser(context.Background(), "container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify the fallback values were passed in the container config calls.
		call2 := (*calls)[2]
		found := false
		for _, a := range call2.args {
			if a == "alpine" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected fallback name 'alpine' in call args: %v", call2.args)
		}
		call3 := (*calls)[3]
		found = false
		for _, a := range call3.args {
			if a == "alpine@localhost" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected fallback email 'alpine@localhost' in call args: %v", call3.args)
		}
	})

	t.Run("container config fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "Test User"},
			{stdout: "test@test.com"},
			errResult("permission denied"), // docker exec git config user.name fails
		})
		err := gitConfigureUser(context.Background(), "container")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to set git user.name") {
			t.Fatalf("error = %q, want to contain 'failed to set git user.name'", err.Error())
		}
	})

	t.Run("container email config fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "Test User"},
			{stdout: "test@test.com"},
			{stdout: ""},                   // user.name succeeds
			errResult("permission denied"), // user.email fails
		})
		err := gitConfigureUser(context.Background(), "container")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to set git user.email") {
			t.Fatalf("error = %q, want to contain 'failed to set git user.email'", err.Error())
		}
	})
}

func TestGitGetCurrentBranch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "main"}})
		branch, err := gitGetCurrentBranch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branch != "main" {
			t.Fatalf("branch = %q, want %q", branch, "main")
		}
	})

	t.Run("detached HEAD returns error", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "HEAD"}})
		_, err := gitGetCurrentBranch(context.Background())
		if err == nil {
			t.Fatal("expected error for detached HEAD")
		}
		if !strings.Contains(err.Error(), "detached") {
			t.Fatalf("error = %q, want to contain 'detached'", err.Error())
		}
	})

	t.Run("error", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("not a git repo")})
		_, err := gitGetCurrentBranch(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to determine current branch") {
			t.Fatalf("error = %q, want to contain 'failed to determine current branch'", err.Error())
		}
	})
}

func TestGitGetRemoteURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "git@github.com:user/repo.git"}})
		url, err := gitGetRemoteURL(context.Background(), "origin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "git@github.com:user/repo.git" {
			t.Fatalf("url = %q, want %q", url, "git@github.com:user/repo.git")
		}
	})

	t.Run("error", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("no such remote")})
		_, err := gitGetRemoteURL(context.Background(), "origin")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to get remote URL") {
			t.Fatalf("error = %q, want to contain 'failed to get remote URL'", err.Error())
		}
	})
}

func TestGitFindRoot(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "/home/user/project"}})
		root, err := gitFindRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != "/home/user/project" {
			t.Fatalf("root = %q, want %q", root, "/home/user/project")
		}
	})

	t.Run("error", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("not a git repo")})
		_, err := gitFindRoot()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not inside a git repository") {
			t.Fatalf("error = %q, want to contain 'not inside a git repository'", err.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// Docker health check: darwin poll success in non-JSON mode (docker.go 191-200)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Docker health check: darwin poll timeout in non-JSON mode (docker.go 182-184)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// defaultRun: tests the real exec.CommandContext wrapper (docker.go:93-113)
// ---------------------------------------------------------------------------

func TestDefaultRun_Success(t *testing.T) {
	stdout, stderr, err := defaultRun(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "hello" {
		t.Errorf("stdout = %q, want %q", stdout, "hello")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestDefaultRun_Error(t *testing.T) {
	_, _, err := defaultRun(context.Background(), "false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
	var execErr *ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected *ExecError, got %T", err)
	}
	if !strings.HasPrefix(execErr.Command, "false") {
		t.Errorf("Command = %q, want prefix %q", execErr.Command, "false")
	}
}

func TestDefaultRun_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := defaultRun(ctx, "sleep", "10")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ---------------------------------------------------------------------------
// defaultRunInteractive: tests the real interactive exec wrapper (docker.go:120-137)
// ---------------------------------------------------------------------------

func TestDefaultRunInteractive_Success(t *testing.T) {
	// Restore SIGINT handling after test (defaultRunInteractive calls signal.Ignore).
	t.Cleanup(func() { signal.Reset(os.Interrupt) })

	err := defaultRunInteractive("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultRunInteractive_Error(t *testing.T) {
	t.Cleanup(func() { signal.Reset(os.Interrupt) })

	err := defaultRunInteractive("false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
	var execErr *ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected *ExecError, got %T", err)
	}
}

// ---------------------------------------------------------------------------
// loadDotEnv: os.Setenv failure with null byte in key (docker.go:591-593)
// ---------------------------------------------------------------------------

func TestLoadDotEnv_SetenvError(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	// A null byte in the key causes os.Setenv to return EINVAL.
	if err := os.WriteFile(envFile, []byte("FOO\x00BAR=value\n"), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	err := loadDotEnv(envFile)
	if err == nil {
		t.Fatal("expected error from os.Setenv with null byte in key")
	}
	if !strings.Contains(err.Error(), "failed to set env var") {
		t.Fatalf("error = %q, want to contain 'failed to set env var'", err.Error())
	}
}
