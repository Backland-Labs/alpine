package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestRunExport(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "export", RunE: runExport}
		cmd.SetContext(context.Background())
		return cmd
	}

	setup := func(t *testing.T, requireAuth bool) *orchestrator {
		t.Helper()
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Setenv("ALPINE_STATE_PATH", filepath.Join(dir, "state.json"))
		t.Cleanup(func() { _ = os.Unsetenv("ALPINE_STATE_PATH") })

		yaml := strings.Join([]string{
			"repo:",
			"  default: https://github.com/acme/repo.git",
			"sandbox:",
			"  image_profile: default",
			"  image_profiles:",
			"    default: class-a",
			"  web_base_url: https://sandbox.example.com",
			"durability:",
			"  bucket: b",
			"  checkpoint_prefix: p",
			"github:",
			"  branch_prefix: alpine",
			fmt.Sprintf("  require_auth: %t", requireAuth),
		}, "\n")
		if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(yaml), 0644); err != nil {
			t.Fatalf("write yaml: %v", err)
		}

		cfg := &Config{
			Sandbox:    SandboxConfig{WebBaseURL: "https://sandbox.example.com", AutoTeardown: true, ImageProfile: "default", ImageProfiles: map[string]string{"default": "class-a"}},
			GitHub:     GitHubConfig{BranchPrefix: "alpine", RequireAuth: requireAuth},
			Durability: DurabilityConfig{Bucket: "b", CheckpointPrefix: "p"},
			BaseImage:  "ubuntu:24.04",
		}
		orch := newOrchestrator(cfg)
		if err := orch.saveStore(&orchestratorStore{Sandboxes: map[string]*sandboxRecord{}}); err != nil {
			t.Fatalf("seed store: %v", err)
		}
		return orch
	}

	t.Run("auth required contract", func(t *testing.T) {
		orch := setup(t, true)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("teardown: %v", err)
		}

		_ = os.Unsetenv("GH_TOKEN")
		_ = os.Unsetenv("GITHUB_TOKEN")

		err := runExport(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected auth error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "github_auth_missing" {
			t.Fatalf("reason = %q", ee.reasonCode)
		}
	})

	t.Run("json output success", func(t *testing.T) {
		orch := setup(t, false)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("teardown: %v", err)
		}

		jsonOutput = true
		out := captureStdout(t, func() {
			if err := runExport(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runExport: %v", err)
			}
		})

		var got exportResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("json: %v", err)
		}
		if got.Branch == "" || got.Source != "checkpoint" {
			t.Fatalf("unexpected export output: %+v", got)
		}
	})

	t.Run("human output path", func(t *testing.T) {
		orch := setup(t, false)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("teardown: %v", err)
		}

		jsonOutput = false
		out := captureStdout(t, func() {
			if err := runExport(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runExport: %v", err)
			}
		})
		if !strings.Contains(out, "Exported alpha") {
			t.Fatalf("unexpected output: %s", out)
		}
	})

	t.Run("human output includes auto teardown message", func(t *testing.T) {
		orch := setup(t, false)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		store, err := orch.loadStore()
		if err != nil {
			t.Fatalf("loadStore: %v", err)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		store.Sandboxes["alpha"].State = stateCompleted
		store.Sandboxes["alpha"].Checkpoint = &checkpointPointer{Verified: true, Manifest: newCheckpointManifest("https://github.com/acme/repo.git", "default", now)}
		if err := orch.saveStore(store); err != nil {
			t.Fatalf("saveStore: %v", err)
		}

		out := captureStdout(t, func() {
			if err := runExport(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runExport: %v", err)
			}
		})
		if !strings.Contains(out, "auto-tore down") {
			t.Fatalf("expected auto teardown message, got: %s", out)
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
		err := runExport(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing sandbox returns reason code", func(t *testing.T) {
		_ = setup(t, false)
		err := runExport(newCmd(), []string{"missing"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "sandbox_not_found" {
			t.Fatalf("reason=%q", ee.reasonCode)
		}
	})
}
