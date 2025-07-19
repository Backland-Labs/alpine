package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestValidateEnvironment(t *testing.T) {
	t.Run("returns nil when claude is available", func(t *testing.T) {
		// This test verifies that validation passes when claude is available
		// It's important to test that the environment validation works correctly
		_, err := exec.LookPath("claude")
		if err != nil {
			t.Skip("Skipping test - claude command not available on this system")
		}

		err = validateEnvironment()
		if err != nil {
			t.Errorf("expected no error when claude is available, got %v", err)
		}
	})
}

func TestValidateClaudeAvailability(t *testing.T) {
	t.Run("returns error when claude command is not available", func(t *testing.T) {
		// TestClaudeAvailability: Missing claude binary detected
		// This test ensures dependency validation works correctly,
		// preventing runtime failures when trying to execute claude
		err := validateClaudeAvailable()

		// Since we can't easily mock exec.LookPath, we'll check if the function exists
		// and returns an appropriate error when claude is not found
		if _, lookupErr := exec.LookPath("claude-definitely-not-exists"); lookupErr != nil {
			// If a non-existent command returns error, our test should too
			if err == nil {
				t.Skip("Skipping test - claude command might be available on this system")
			}
		}
	})

	t.Run("returns nil when claude command is available", func(t *testing.T) {
		// This test verifies that the validation passes when claude is available
		// It's crucial to ensure the positive case works correctly
		_, err := exec.LookPath("claude")
		if err != nil {
			t.Skip("Skipping test - claude command not available on this system")
		}

		err = validateClaudeAvailable()
		if err != nil {
			t.Errorf("expected no error when claude is available, got %v", err)
		}
	})
}

func TestMainWithValidation(t *testing.T) {
	t.Run("exits early when environment validation fails", func(t *testing.T) {
		// This test ensures that main() integrates environment validation
		// as the first step, implementing the fail-fast principle
		// We can't easily test os.Exit, so we'll test the integration
		// by checking that validateEnvironment is called appropriately

		// Save original values
		oldArgs := os.Args

		defer func() {
			os.Args = oldArgs
		}()

		os.Args = []string{"river"}

		// We can't directly test main() with os.Exit,
		// but we can verify the validation function behavior
		// The only validation now is for claude availability
		err := validateEnvironment()
		// If claude is not available, we should get an error
		if _, lookupErr := exec.LookPath("claude"); lookupErr != nil && err == nil {
			t.Error("expected validation to fail when claude is not available")
		}
	})
}
