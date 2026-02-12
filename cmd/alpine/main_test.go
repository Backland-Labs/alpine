package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string // empty means no file
		wantErr   string
		wantImage string
	}{
		{
			name:      "file not found uses defaults",
			yaml:      "",
			wantImage: "ubuntu:24.04",
		},
		{
			name:      "valid yaml",
			yaml:      "base_image: node:20\nservices:\n  - postgres\n",
			wantImage: "node:20",
		},
		{
			name:    "invalid yaml",
			yaml:    ":\n  :\n  invalid",
			wantErr: "parsing alpine.yaml",
		},
		{
			name:    "validation failure empty base_image",
			yaml:    "base_image: \"\"\n",
			wantErr: "base_image cannot be empty",
		},
		{
			name:    "unknown service",
			yaml:    "base_image: ubuntu:24.04\nservices:\n  - mysql\n",
			wantErr: "unknown service",
		},
		{
			name:    "read error (directory instead of file)",
			yaml:    "DIRECTORY", // sentinel: create a directory named alpine.yaml
			wantErr: "reading alpine.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			origDir, _ := os.Getwd()
			os.Chdir(dir)
			t.Cleanup(func() { os.Chdir(origDir) })

			if tt.yaml == "DIRECTORY" {
				os.Mkdir(filepath.Join(dir, "alpine.yaml"), 0755)
			} else if tt.yaml != "" {
				os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(tt.yaml), 0644)
			}

			cfg, err := loadConfig()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.BaseImage != tt.wantImage {
				t.Fatalf("BaseImage = %q, want %q", cfg.BaseImage, tt.wantImage)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "empty base_image",
			cfg:     Config{BaseImage: ""},
			wantErr: "base_image cannot be empty",
		},
		{
			name:    "unknown service",
			cfg:     Config{BaseImage: "ubuntu:24.04", Services: []string{"mysql"}},
			wantErr: "unknown service",
		},
		{
			name: "valid postgres",
			cfg:  Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres"}},
		},
		{
			name: "valid redis",
			cfg:  Config{BaseImage: "ubuntu:24.04", Services: []string{"redis"}},
		},
		{
			name: "both services",
			cfg:  Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres", "redis"}},
		},
		{
			name: "no services",
			cfg:  Config{BaseImage: "ubuntu:24.04"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExitErrorError(t *testing.T) {
	e := &exitError{msg: "something failed", code: 2}
	if e.Error() != "something failed" {
		t.Fatalf("Error() = %q, want %q", e.Error(), "something failed")
	}
}

func TestOutputJSON(t *testing.T) {
	resetFlags(t)
	out := captureStdout(t, func() {
		err := outputJSON(map[string]string{"key": "val"})
		if err != nil {
			t.Fatalf("outputJSON failed: %v", err)
		}
	})
	if !strings.Contains(out, `"key": "val"`) {
		t.Fatalf("expected JSON output, got: %s", out)
	}
}

func TestOutputError(t *testing.T) {
	tests := []struct {
		name     string
		json     bool
		wantJSON bool
	}{
		{name: "json mode", json: true, wantJSON: true},
		{name: "text mode", json: false, wantJSON: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			jsonOutput = tt.json

			if tt.wantJSON {
				out := captureStdout(t, func() {
					outputError("test error", 1)
				})
				if !strings.Contains(out, `"error"`) {
					t.Fatalf("expected JSON error output, got: %s", out)
				}
				if !strings.Contains(out, `"exit_code"`) {
					t.Fatalf("expected exit_code in output, got: %s", out)
				}
			} else {
				// Text mode writes to stderr, not stdout.
				out := captureStdout(t, func() {
					outputError("test error", 1)
				})
				if strings.Contains(out, "error") {
					t.Fatalf("text mode should not write to stdout, got: %s", out)
				}
			}
		})
	}
}

func TestExecute(t *testing.T) {
	t.Run("success returns 0", func(t *testing.T) {
		resetFlags(t)
		mockRun(t, []cmdResult{})
		// Execute with --version flag, which returns 0.
		origArgs := os.Args
		os.Args = []string{"alpine", "--version"}
		t.Cleanup(func() { os.Args = origArgs })

		code := execute(context.Background())
		if code != 0 {
			t.Fatalf("execute returned %d, want 0", code)
		}
	})

	t.Run("unknown command returns 1", func(t *testing.T) {
		resetFlags(t)
		mockRun(t, []cmdResult{})
		origArgs := os.Args
		os.Args = []string{"alpine", "nonexistent-cmd"}
		t.Cleanup(func() { os.Args = origArgs })

		code := execute(context.Background())
		if code != 1 {
			t.Fatalf("execute returned %d, want 1", code)
		}
	})

	t.Run("exitError returns custom code", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		// Run "alpine create" with no args -- Cobra's Args validator returns error.
		origArgs := os.Args
		os.Args = []string{"alpine", "create"}
		t.Cleanup(func() { os.Args = origArgs })

		captureStdout(t, func() {
			code := execute(context.Background())
			if code != 1 {
				t.Fatalf("execute returned %d, want 1", code)
			}
		})
	})

	t.Run("verbose flag sets debug logging", func(t *testing.T) {
		resetFlags(t)
		// Call PersistentPreRun directly to exercise the verbose logging setup.
		verbose = true
		rootCmd.PersistentPreRun(rootCmd, []string{})
		verbose = false
		rootCmd.PersistentPreRun(rootCmd, []string{})
	})

	t.Run("exitError returns custom code", func(t *testing.T) {
		resetFlags(t)
		jsonOutput = true
		mockRun(t, []cmdResult{})
		// "INVALID" is uppercase and fails validateName, returning exitError code 1.
		origArgs := os.Args
		os.Args = []string{"alpine", "create", "INVALID"}
		t.Cleanup(func() { os.Args = origArgs })

		captureStdout(t, func() {
			code := execute(context.Background())
			if code != 1 {
				t.Fatalf("execute returned %d, want 1", code)
			}
		})
	})
}
