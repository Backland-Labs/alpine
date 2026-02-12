//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Integration tests require a running Docker daemon.
// Run with: go test -tags=integration -v ./cmd/alpine/

func TestIntegrationDockerOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	_, _, err := defaultRun(ctx, "docker", "info")
	if err != nil {
		t.Skip("Docker daemon not available, skipping integration test")
	}

	name := "integ-test"
	project := "alpine-" + name

	// Clean up any leftover from a previous failed run.
	_ = composeDown(ctx, name)

	// 1. Build image from generated Dockerfile.
	cfg := &Config{BaseImage: "ubuntu:24.04"}
	dockerfile := generateDockerfile(cfg.BaseImage)
	hash := dockerfileHash(dockerfile)
	imageTag := "alpine-dev:" + hash

	exists, err := imageExists(ctx, imageTag)
	if err != nil {
		t.Fatalf("imageExists: %v", err)
	}
	if !exists {
		tmpDir := t.TempDir()
		dfPath := filepath.Join(tmpDir, "Dockerfile")
		if err := os.WriteFile(dfPath, dockerfile, 0644); err != nil {
			t.Fatalf("write Dockerfile: %v", err)
		}

		buildCtx, buildCancel := context.WithTimeout(ctx, 5*time.Minute)
		defer buildCancel()
		_, stderr, err := defaultRun(buildCtx, "docker", "build", "-t", imageTag, tmpDir)
		if err != nil {
			t.Fatalf("docker build failed: %s", stderr)
		}
	}

	// 2. Generate compose YAML and start environment.
	composeYAML, err := generateComposeYAML(cfg, name, "main", runtime.GOOS, imageTag)
	if err != nil {
		t.Fatalf("generateComposeYAML: %v", err)
	}

	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, composeYAML, 0644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	if err := composeUp(ctx, name, composePath); err != nil {
		t.Fatalf("composeUp: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = composeDown(cleanupCtx, name)
	})

	// 3. Discover the dev container.
	container, err := discoverContainer(ctx, name)
	if err != nil {
		t.Fatalf("discoverContainer: %v", err)
	}
	if container != project+"-dev-1" {
		t.Errorf("container = %q, want %q", container, project+"-dev-1")
	}

	// 4. Verify duplicate detection catches the running environment.
	if err := checkDuplicate(ctx, name); err == nil {
		t.Fatal("checkDuplicate should return error for existing environment")
	}

	// 5. Inspect container state.
	state, err := inspectContainer(ctx, container, "{{.State.Status}}")
	if err != nil {
		t.Fatalf("inspectContainer: %v", err)
	}
	if state != "running" {
		t.Errorf("container state = %q, want %q", state, "running")
	}

	// 6. Verify the container runs as the claude user.
	stdout, _, err := defaultRun(ctx, "docker", "exec", container, "whoami")
	if err != nil {
		t.Fatalf("docker exec whoami: %v", err)
	}
	if strings.TrimSpace(stdout) != "claude" {
		t.Errorf("whoami = %q, want %q", stdout, "claude")
	}

	// 7. Verify git is available inside the container.
	stdout, _, err = defaultRun(ctx, "docker", "exec", container, "git", "--version")
	if err != nil {
		t.Fatalf("docker exec git --version: %v", err)
	}
	if !strings.Contains(stdout, "git version") {
		t.Errorf("git --version = %q, expected to contain 'git version'", stdout)
	}

	// 8. Verify compose YAML labels are present.
	labelName, err := inspectContainer(ctx, container, `{{index .Config.Labels "alpine.name"}}`)
	if err != nil {
		t.Fatalf("inspect alpine.name label: %v", err)
	}
	if labelName != name {
		t.Errorf("alpine.name label = %q, want %q", labelName, name)
	}

	// 9. Compose down and verify container is gone.
	if err := composeDown(ctx, name); err != nil {
		t.Fatalf("composeDown: %v", err)
	}
	if err := checkDuplicate(ctx, name); err != nil {
		t.Errorf("environment should not exist after composeDown, but checkDuplicate returned: %v", err)
	}
}

func TestIntegrationListCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	_, _, err := defaultRun(ctx, "docker", "info")
	if err != nil {
		t.Skip("Docker daemon not available, skipping integration test")
	}

	// Run the real list command logic (no environments expected).
	resetFlags(t)
	jsonOutput = true
	origRun := run
	defer func() { run = origRun }()

	// Use real run function (not mocked).
	run = defaultRun

	cmd := newListCmd()
	out := captureStdout(t, func() {
		if err := runList(cmd, []string{}); err != nil {
			t.Fatalf("runList: %v", err)
		}
	})

	var envs []envInfo
	if err := json.Unmarshal([]byte(out), &envs); err != nil {
		t.Fatalf("failed to parse list JSON: %v\noutput: %s", err, out)
	}
	// We can't assert exact count since other environments may exist,
	// but the command should complete without error and return valid JSON.
}
