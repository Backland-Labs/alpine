package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunStatus(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		return cmd
	}

	setup := func(t *testing.T) *Config {
		t.Helper()
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir temp dir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(origWD) })

		statePath := filepath.Join(dir, "state.json")
		if err := os.Setenv("ALPINE_STATE_PATH", statePath); err != nil {
			t.Fatalf("set env: %v", err)
		}
		t.Cleanup(func() { _ = os.Unsetenv("ALPINE_STATE_PATH") })

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		return cfg
	}

	t.Run("json status includes lifecycle and durability", func(t *testing.T) {
		cfg := setup(t)
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("teardown: %v", err)
		}

		jsonOutput = true
		out := captureStdout(t, func() {
			if err := runStatus(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runStatus: %v", err)
			}
		})

		var got statusOutput
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("unmarshal status: %v", err)
		}
		if got.Name != "alpha" {
			t.Fatalf("name = %q", got.Name)
		}
		if got.State != stateDestroyed {
			t.Fatalf("state = %q", got.State)
		}
		if got.Durability.CheckpointID == "" || !got.Durability.Verified {
			t.Fatalf("expected verified checkpoint: %+v", got.Durability)
		}
	})

	t.Run("human status prints key fields", func(t *testing.T) {
		cfg := setup(t)
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "beta", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		store, err := orch.loadStore()
		if err != nil {
			t.Fatalf("loadStore: %v", err)
		}
		store.Sandboxes["beta"].TeardownBlockers = []string{"active_export_lock"}
		store.Sandboxes["beta"].ErrorReason = "checkpoint_checksum_mismatch"
		if err := orch.saveStore(store); err != nil {
			t.Fatalf("saveStore: %v", err)
		}

		jsonOutput = false
		out := captureStdout(t, func() {
			if err := runStatus(newCmd(), []string{"beta"}); err != nil {
				t.Fatalf("runStatus: %v", err)
			}
		})

		for _, want := range []string{"Sandbox:", "State:", "Repo:", "Image profile:", "Blockers:", "Error reason:"} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q in output: %s", want, out)
			}
		}
	})

	t.Run("stale parsing fallback branch", func(t *testing.T) {
		cfg := setup(t)
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "gamma", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		store, err := orch.loadStore()
		if err != nil {
			t.Fatalf("loadStore: %v", err)
		}
		store.Sandboxes["gamma"].RuntimeProbe.CheckedAt = "not-a-time"
		if err := orch.saveStore(store); err != nil {
			t.Fatalf("saveStore: %v", err)
		}

		jsonOutput = true
		out := captureStdout(t, func() {
			if err := runStatus(newCmd(), []string{"gamma"}); err != nil {
				t.Fatalf("runStatus: %v", err)
			}
		})

		var got statusOutput
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("unmarshal status: %v", err)
		}
		if !got.Runtime.ProbeStale {
			t.Fatal("expected stale=true when probe timestamp invalid")
		}
	})

	t.Run("not found returns reason coded error", func(t *testing.T) {
		_ = setup(t)
		err := runStatus(newCmd(), []string{"missing"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee, ok := err.(*exitError)
		if !ok {
			t.Fatalf("expected exitError, got %T", err)
		}
		if ee.code != 1 {
			t.Fatalf("code = %d", ee.code)
		}
		if ee.reasonCode != "sandbox_not_found" {
			t.Fatalf("reason = %q", ee.reasonCode)
		}
	})

	t.Run("invalid config returns user error", func(t *testing.T) {
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(":\n  :\n  invalid"), 0644); err != nil {
			t.Fatalf("write yaml: %v", err)
		}
		err := runStatus(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("state read failure returns system error", func(t *testing.T) {
		_ = setup(t)
		if err := os.WriteFile("state.json", []byte("bad"), 0644); err != nil {
			t.Fatalf("write state: %v", err)
		}
		err := runStatus(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("code=%d", ee.code)
		}
	})
}

func TestRunStatus_ControlPlane(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	origWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWD) })

	configYAML := strings.Join([]string{
		"sandbox:",
		"  control_plane_url: http://127.0.0.1:9999",
	}, "\n")

	if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	newCmd := func() *cobra.Command {
		return &cobra.Command{}
	}

	t.Run("control plane unreachable returns system error", func(t *testing.T) {
		resetFlags(t)
		err := runStatus(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("expected system error code 2, got %d", ee.code)
		}
	})
}
