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

func TestRunTeardown(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "teardown", RunE: runTeardown}
		cmd.SetContext(context.Background())
		return cmd
	}

	setup := func(t *testing.T) *orchestrator {
		t.Helper()
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Setenv("ALPINE_STATE_PATH", filepath.Join(dir, "state.json"))
		t.Cleanup(func() { _ = os.Unsetenv("ALPINE_STATE_PATH") })

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		return newOrchestrator(cfg)
	}

	t.Run("requires force for running sandbox", func(t *testing.T) {
		orch := setup(t)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		teardownForce = false
		err := runTeardown(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "teardown_requires_force" {
			t.Fatalf("reason = %q", ee.reasonCode)
		}
	})

	t.Run("json output includes checkpoint", func(t *testing.T) {
		orch := setup(t)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		teardownForce = true
		jsonOutput = true
		out := captureStdout(t, func() {
			if err := runTeardown(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runTeardown: %v", err)
			}
		})

		var got teardownResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("json: %v", err)
		}
		if !got.Destroyed || got.CheckpointID == "" {
			t.Fatalf("unexpected teardown output: %+v", got)
		}
	})

	t.Run("human output path", func(t *testing.T) {
		orch := setup(t)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		teardownForce = true
		jsonOutput = false
		out := captureStdout(t, func() {
			if err := runTeardown(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runTeardown: %v", err)
			}
		})
		if out == "" {
			t.Fatal("expected output")
		}
	})

	t.Run("invalid config returns user error", func(t *testing.T) {
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(":::"), 0644); err != nil {
			t.Fatalf("write yaml: %v", err)
		}
		err := runTeardown(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing sandbox bubbles reason code", func(t *testing.T) {
		_ = setup(t)
		err := runTeardown(newCmd(), []string{"missing"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "sandbox_not_found" {
			t.Fatalf("reason=%q", ee.reasonCode)
		}
	})
}

func TestRunTeardown_ControlPlane(t *testing.T) {
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
		cmd := &cobra.Command{Use: "teardown", RunE: runTeardown}
		cmd.SetContext(context.Background())
		return cmd
	}

	t.Run("control plane unreachable returns system error", func(t *testing.T) {
		resetFlags(t)
		err := runTeardown(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("expected system error code 2, got %d", ee.code)
		}
	})
}
