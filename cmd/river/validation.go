package main

import (
	"fmt"
	"os/exec"
)

// validateEnvironment checks that all required dependencies are available.
// This implements the fail-fast principle to prevent cryptic failures later in execution.
func validateEnvironment() error {
	// Check claude availability
	if err := validateClaudeAvailable(); err != nil {
		return fmt.Errorf("env error: %w", err)
	}

	return nil
}

// validateClaudeAvailable checks if the claude CLI command is available in PATH.
// This prevents runtime failures when trying to execute claude commands.
func validateClaudeAvailable() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI command not found in PATH: %w", err)
	}
	return nil
}
