package config

import (
	"os"
	"testing"
)

// TestLinearAPIKeyRequired tests that LinearAPIKey is required
func TestLinearAPIKeyRequired(t *testing.T) {
	// Clear all environment variables
	envVars := []string{
		"RIVER_WORKDIR",
		"RIVER_VERBOSITY",
		"RIVER_SHOW_OUTPUT",
		"RIVER_STATE_FILE",
		"RIVER_AUTO_CLEANUP",
		"RIVER_LINEAR_API_KEY",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}

	// Test missing LinearAPIKey
	_, err := New()
	if err == nil {
		t.Error("Expected error when RIVER_LINEAR_API_KEY is not set, got nil")
	}
	if err != nil && err.Error() != "RIVER_LINEAR_API_KEY environment variable is required" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	// Test with LinearAPIKey set
	os.Setenv("RIVER_LINEAR_API_KEY", "test-key-123")
	defer os.Unsetenv("RIVER_LINEAR_API_KEY")

	cfg, err := New()
	if err != nil {
		t.Fatalf("Unexpected error with LinearAPIKey set: %v", err)
	}

	if cfg.LinearAPIKey != "test-key-123" {
		t.Errorf("LinearAPIKey = %q, want %q", cfg.LinearAPIKey, "test-key-123")
	}
}