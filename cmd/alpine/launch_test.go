package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunLaunch(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "launch", RunE: runLaunch}
		cmd.SetContext(context.Background())
		return cmd
	}

	setup := func(t *testing.T, yaml string) {
		t.Helper()
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(origWD) })

		if yaml != "" {
			if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(yaml), 0644); err != nil {
				t.Fatalf("write yaml: %v", err)
			}
		}
		if err := os.Setenv("ALPINE_STATE_PATH", filepath.Join(dir, "state.json")); err != nil {
			t.Fatalf("set state path: %v", err)
		}
		t.Cleanup(func() { _ = os.Unsetenv("ALPINE_STATE_PATH") })
	}

	configYAML := strings.Join([]string{
		"repo:",
		"  default: https://github.com/acme/repo.git",
		"sandbox:",
		"  image_profile: default",
		"  image_profiles:",
		"    default: opencode-default",
		"  web_base_url: https://sandbox.example.com",
		"durability:",
		"  bucket: checkpoints",
		"  checkpoint_prefix: sandboxes",
		"github:",
		"  branch_prefix: alpine",
		"  require_auth: false",
	}, "\n")

	t.Run("fails on invalid name", func(t *testing.T) {
		setup(t, configYAML)
		err := runLaunch(newCmd(), []string{"@@@"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("fails unknown image profile before provisioning", func(t *testing.T) {
		setup(t, configYAML)
		launchImageProfile = "unknown"
		err := runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "image_profile_unknown" {
			t.Fatalf("reason = %q", ee.reasonCode)
		}
	})

	t.Run("fails when repo cannot resolve", func(t *testing.T) {
		setup(t, "")
		launchRepo = ""
		err := runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "repo_resolution_failed" {
			t.Fatalf("reason = %q", ee.reasonCode)
		}
	})

	t.Run("fails when repo unreachable", func(t *testing.T) {
		setup(t, configYAML)
		mockRun(t, []cmdResult{errResult("auth failed")})
		err := runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 || ee.reasonCode != "repo_unreachable" {
			t.Fatalf("unexpected error contract: %#v", ee)
		}
	})

	t.Run("successful launch json", func(t *testing.T) {
		setup(t, configYAML)
		jsonOutput = true
		mockRun(t, []cmdResult{{stdout: "ok"}})
		out := captureStdout(t, func() {
			if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runLaunch: %v", err)
			}
		})

		var got launchOutput
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("json: %v", err)
		}
		if got.Name != "alpha" || got.State != stateRunning {
			t.Fatalf("unexpected output: %+v", got)
		}
		if got.ContainerClass != "opencode-default" {
			t.Fatalf("container class = %q", got.ContainerClass)
		}
	})

	t.Run("identity mismatch requires force recreate", func(t *testing.T) {
		setup(t, configYAML)
		mockRun(t, []cmdResult{{stdout: "ok"}})
		if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
			t.Fatalf("first launch: %v", err)
		}

		launchRepo = "https://github.com/acme/another.git"
		err := runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected mismatch error")
		}
		ee := err.(*exitError)
		if ee.code != 1 {
			t.Fatalf("code=%d", ee.code)
		}
	})

	t.Run("force recreate succeeds", func(t *testing.T) {
		setup(t, configYAML)
		mockRun(t, []cmdResult{{stdout: "ok"}, {stdout: "ok"}})
		if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
			t.Fatalf("first launch: %v", err)
		}

		launchRepo = "https://github.com/acme/another.git"
		launchForceRecreate = true
		if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
			t.Fatalf("force recreate launch: %v", err)
		}
	})

	t.Run("human output includes task accepted", func(t *testing.T) {
		setup(t, configYAML)
		launchTask = "implement feature"
		jsonOutput = false
		mockRun(t, []cmdResult{{stdout: "ok"}})
		out := captureStdout(t, func() {
			if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runLaunch: %v", err)
			}
		})
		if !strings.Contains(out, "Initial task accepted") {
			t.Fatalf("unexpected output: %s", out)
		}
	})

	t.Run("identity read parse failure returns system error", func(t *testing.T) {
		setup(t, configYAML)
		if err := os.WriteFile("state.json", []byte("bad"), 0644); err != nil {
			t.Fatalf("write state: %v", err)
		}
		err := runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("code=%d", ee.code)
		}
	})

	t.Run("sanitizes incoming name", func(t *testing.T) {
		setup(t, configYAML)
		mockRun(t, []cmdResult{{stdout: "ok"}})
		if err := runLaunch(newCmd(), []string{"Alpha/Feature"}); err != nil {
			t.Fatalf("runLaunch: %v", err)
		}
	})

	t.Run("checkpoint mismatch from orchestrator bubbles reason", func(t *testing.T) {
		setup(t, configYAML)
		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("seed launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("seed teardown: %v", err)
		}
		store, err := orch.loadStore()
		if err != nil {
			t.Fatalf("loadStore: %v", err)
		}
		store.Sandboxes["alpha"].Checkpoint.Manifest.ContentHash = "bad"
		if err := orch.saveStore(store); err != nil {
			t.Fatalf("saveStore: %v", err)
		}

		mockRun(t, []cmdResult{{stdout: "ok"}})
		err = runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected checksum mismatch")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "checkpoint_checksum_mismatch" {
			t.Fatalf("reason=%q", ee.reasonCode)
		}
	})

	t.Run("human output shows reused identity", func(t *testing.T) {
		setup(t, configYAML)
		jsonOutput = false
		mockRun(t, []cmdResult{{stdout: "ok"}, {stdout: "ok"}})
		if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
			t.Fatalf("first launch: %v", err)
		}
		out := captureStdout(t, func() {
			if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("second launch: %v", err)
			}
		})
		if !strings.Contains(out, "Reused existing identity") {
			t.Fatalf("expected reused message, got: %s", out)
		}
	})

	t.Run("human output shows resumed checkpoint", func(t *testing.T) {
		setup(t, configYAML)
		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("seed launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("seed teardown: %v", err)
		}

		mockRun(t, []cmdResult{{stdout: "ok"}})
		out := captureStdout(t, func() {
			if err := runLaunch(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runLaunch: %v", err)
			}
		})
		if !strings.Contains(out, "Resumed from durable checkpoint") {
			t.Fatalf("expected resumed message, got: %s", out)
		}
	})
}

func TestRunLaunch_ControlPlane(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	origWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	configYAML := strings.Join([]string{
		"repo:",
		"  default: https://github.com/acme/repo.git",
		"sandbox:",
		"  image_profile: default",
		"  image_profiles:",
		"    default: opencode-default",
		"  web_base_url: https://sandbox.example.com",
		"  control_plane_url: http://127.0.0.1:9999",
		"durability:",
		"  bucket: checkpoints",
		"  checkpoint_prefix: sandboxes",
		"github:",
		"  branch_prefix: alpine",
		"  require_auth: false",
	}, "\n")

	if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "launch", RunE: runLaunch}
		cmd.SetContext(context.Background())
		return cmd
	}

	t.Run("control plane unreachable returns system error", func(t *testing.T) {
		resetFlags(t)
		err := runLaunch(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("expected system error code 2, got %d", ee.code)
		}
	})
}
