package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunOpen(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "open", RunE: runOpen}
		cmd.SetContext(context.Background())
		return cmd
	}

	setup := func(t *testing.T) {
		t.Helper()
		resetFlags(t)
		dir := t.TempDir()
		origWD, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(origWD) })
		_ = os.Setenv("ALPINE_STATE_PATH", filepath.Join(dir, "state.json"))
		t.Cleanup(func() { _ = os.Unsetenv("ALPINE_STATE_PATH") })
	}

	t.Run("prints URL in json mode", func(t *testing.T) {
		setup(t)
		cfg, _ := loadConfig()
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		jsonOutput = true
		out := captureStdout(t, func() {
			if err := runOpen(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runOpen: %v", err)
			}
		})

		var got openOutput
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("json: %v", err)
		}
		if got.WebURL == "" {
			t.Fatal("expected web url")
		}
	})

	t.Run("opens browser in human mode when command succeeds", func(t *testing.T) {
		setup(t)
		cfg, _ := loadConfig()
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		mockRun(t, []cmdResult{{stdout: ""}})
		out := captureStdout(t, func() {
			if err := runOpen(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runOpen: %v", err)
			}
		})
		if out == "" {
			t.Fatal("expected output")
		}
	})

	t.Run("print-only mode skips browser command", func(t *testing.T) {
		setup(t)
		cfg, _ := loadConfig()
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		openPrintOnly = true
		out := captureStdout(t, func() {
			if err := runOpen(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runOpen: %v", err)
			}
		})
		if out == "" {
			t.Fatal("expected url output")
		}
	})

	t.Run("fallback to printing URL when browser command fails", func(t *testing.T) {
		setup(t)
		cfg, _ := loadConfig()
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		mockRun(t, []cmdResult{{err: errors.New("no browser")}})
		out := captureStdout(t, func() {
			if err := runOpen(newCmd(), []string{"alpha"}); err != nil {
				t.Fatalf("runOpen: %v", err)
			}
		})
		if out == "" {
			t.Fatal("expected fallback URL output")
		}
	})

	t.Run("non-running returns deterministic reason", func(t *testing.T) {
		setup(t)
		cfg, _ := loadConfig()
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alpha", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}
		if _, err := orch.teardown(teardownOptions{Name: "alpha", Force: true}); err != nil {
			t.Fatalf("teardown: %v", err)
		}

		err := runOpen(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.reasonCode != "sandbox_not_running" {
			t.Fatalf("reason = %q", ee.reasonCode)
		}
	})

	t.Run("sanitizes name before open", func(t *testing.T) {
		setup(t)
		cfg, _ := loadConfig()
		orch := newOrchestrator(cfg)
		if _, err := orch.launch(launchOptions{Name: "alphafeature", Repo: "https://github.com/acme/repo.git", ImageProfile: "default"}); err != nil {
			t.Fatalf("launch: %v", err)
		}

		out := captureStdout(t, func() {
			if err := runOpen(newCmd(), []string{"Alpha Feature"}); err != nil {
				t.Fatalf("runOpen: %v", err)
			}
		})
		if !strings.Contains(out, "https://") {
			t.Fatalf("expected url output, got: %s", out)
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
		err := runOpen(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestOpenBrowserURLForOS(t *testing.T) {
	resetFlags(t)

	t.Run("darwin", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{{stdout: ""}})
		if err := openBrowserURLForOS("darwin", "https://example.com"); err != nil {
			t.Fatalf("openBrowserURLForOS: %v", err)
		}
		if len(*calls) != 1 || (*calls)[0].name != "open" {
			t.Fatalf("unexpected call: %+v", *calls)
		}
	})

	t.Run("linux", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{{stdout: ""}})
		if err := openBrowserURLForOS("linux", "https://example.com"); err != nil {
			t.Fatalf("openBrowserURLForOS: %v", err)
		}
		if len(*calls) != 1 || (*calls)[0].name != "xdg-open" {
			t.Fatalf("unexpected call: %+v", *calls)
		}
	})

	t.Run("windows", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{{stdout: ""}})
		if err := openBrowserURLForOS("windows", "https://example.com"); err != nil {
			t.Fatalf("openBrowserURLForOS: %v", err)
		}
		if len(*calls) != 1 || (*calls)[0].name != "rundll32" {
			t.Fatalf("unexpected call: %+v", *calls)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		if err := openBrowserURLForOS("plan9", "https://example.com"); err == nil {
			t.Fatal("expected unsupported platform error")
		}
	})

	t.Run("propagates command error", func(t *testing.T) {
		mockRun(t, []cmdResult{{err: errors.New("boom")}})
		if err := openBrowserURLForOS("linux", "https://example.com"); err == nil {
			t.Fatal("expected command error")
		}
	})
}

func TestRunOpen_ControlPlane(t *testing.T) {
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
		cmd := &cobra.Command{Use: "open", RunE: runOpen}
		cmd.SetContext(context.Background())
		return cmd
	}

	t.Run("control plane unreachable returns system error", func(t *testing.T) {
		resetFlags(t)
		err := runOpen(newCmd(), []string{"alpha"})
		if err == nil {
			t.Fatal("expected error")
		}
		ee := err.(*exitError)
		if ee.code != 2 {
			t.Fatalf("expected system error code 2, got %d", ee.code)
		}
	})
}
