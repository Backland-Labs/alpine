package main

import (
	"context"
	"os"
	"testing"
)

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

	t.Run("exitError returns custom code via validateName", func(t *testing.T) {
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
