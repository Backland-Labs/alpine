package main

import (
	"fmt"
	"os"
	"os/exec"
)

// validateEnvironment checks that all required environment variables and dependencies are available.
// This implements the fail-fast principle to prevent cryptic failures later in execution.
func validateEnvironment() error {
	// Check LINEAR_API_KEY
	if os.Getenv("LINEAR_API_KEY") == "" {
		return fmt.Errorf("env error: LINEAR_API_KEY environment variable is not set")
	}
	
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