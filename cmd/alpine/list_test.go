package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunList(t *testing.T) {
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

	t.Run("empty list human", func(t *testing.T) {
		_ = setup(t)
		jsonOutput = false

		out := captureStdout(t, func() {
			if err := runList(newCmd(), nil); err != nil {
				t.Fatalf("runList error: %v", err)
			}
		})

		if !strings.Contains(out, "No managed sandboxes") {
			t.Fatalf("expected empty message, got: %s", out)
		}
	})

	t.Run("empty list json", func(t *testing.T) {
		_ = setup(t)
		jsonOutput = true

		out := captureStdout(t, func() {
			if err := runList(newCmd(), nil); err != nil {
				t.Fatalf("runList error: %v", err)
			}
		})

		var items []listOutput
		if err := json.Unmarshal([]byte(out), &items); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("expected 0 entries, got %d", len(items))
		}
	})

	t.Run("lists active sandboxes", func(t *testing.T) {
		cfg := setup(t)
		orch := newOrchestrator(cfg)

		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch alpha: %v", err)
		}
		if _, err := orch.launch(launchOptions{Name: "beta", Repo: "https://github.com/acme/repo2.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch beta: %v", err)
		}

		jsonOutput = true
		out := captureStdout(t, func() {
			if err := runList(newCmd(), nil); err != nil {
				t.Fatalf("runList error: %v", err)
			}
		})

		var items []listOutput
		if err := json.Unmarshal([]byte(out), &items); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(items))
		}
		if items[0].Name != "alpha" {
			t.Fatalf("expected alpha first, got %s", items[0].Name)
		}
		if items[0].State != stateRunning {
			t.Fatalf("expected running, got %s", items[0].State)
		}
	})

	t.Run("human table for active sandboxes", func(t *testing.T) {
		cfg := setup(t)
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch alpha: %v", err)
		}

		jsonOutput = false
		out := captureStdout(t, func() {
			if err := runList(newCmd(), nil); err != nil {
				t.Fatalf("runList error: %v", err)
			}
		})

		for _, want := range []string{"NAME", "STATE", "PROFILE", "alpha"} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q in output: %s", want, out)
			}
		}
	})

	t.Run("returns user error on invalid config", func(t *testing.T) {
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(":\n  :\n  invalid"), 0644); err != nil {
			t.Fatalf("write yaml: %v", err)
		}

		err := runList(newCmd(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if _, ok := err.(*exitError); !ok {
			t.Fatalf("expected exitError, got %T", err)
		}
	})

	t.Run("returns system error on unreadable state", func(t *testing.T) {
		cfg := setup(t)
		orch := newOrchestrator(cfg)
		if err := os.WriteFile(orch.statePath, []byte("bad"), 0644); err != nil {
			t.Fatalf("write state: %v", err)
		}
		err := runList(newCmd(), nil)
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("code=%d", ee.code)
		}
	})
}
