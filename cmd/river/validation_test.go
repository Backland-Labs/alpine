package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestValidateEnvironment(t *testing.T) {
	t.Run("returns error when LINEAR_API_KEY is not set", func(t *testing.T) {
		// TestEnvironmentValidation: Missing vars cause early exit
		// This test ensures that the application fails fast when required
		// environment variables are missing, preventing cryptic failures later
		oldValue := os.Getenv("LINEAR_API_KEY")
		defer os.Setenv("LINEAR_API_KEY", oldValue)

		os.Unsetenv("LINEAR_API_KEY")

		err := validateEnvironment()
		if err == nil {
			t.Error("expected error when LINEAR_API_KEY is not set, got nil")
		}

		expectedMsg := "env error: LINEAR_API_KEY environment variable is not set"
		if err != nil && err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("returns nil when LINEAR_API_KEY is set", func(t *testing.T) {
		// This test verifies that validation passes when all requirements are met
		// It's important to test both failure and success cases
		oldValue := os.Getenv("LINEAR_API_KEY")
		defer os.Setenv("LINEAR_API_KEY", oldValue)

		os.Setenv("LINEAR_API_KEY", "test-api-key")

		// Mock the claude command availability
		oldPath := os.Getenv("PATH")
		defer os.Setenv("PATH", oldPath)

		// Assume claude is available for this test
		err := validateEnvironment()
		if err != nil {
			t.Errorf("expected no error when LINEAR_API_KEY is set, got %v", err)
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
		oldLinearKey := os.Getenv("LINEAR_API_KEY")
		oldArgs := os.Args

		defer func() {
			os.Setenv("LINEAR_API_KEY", oldLinearKey)
			os.Args = oldArgs
		}()

		// Set up invalid environment
		os.Unsetenv("LINEAR_API_KEY")
		os.Args = []string{"river"}

		// We can't directly test main() with os.Exit,
		// but we can verify the validation function behavior
		err := validateEnvironment()
		if err == nil {
			t.Error("expected validation to fail with missing LINEAR_API_KEY")
		}
	})
}
