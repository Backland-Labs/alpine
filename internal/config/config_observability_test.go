package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObservabilityConfiguration tests centralized observability configuration
func TestObservabilityConfiguration(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"ALPINE_TOOL_CALL_EVENTS_ENABLED",
		"ALPINE_TOOL_CALL_BATCH_SIZE",
		"ALPINE_TOOL_CALL_SAMPLE_RATE",
		"ALPINE_OBSERVABILITY_ENABLED",
	}

	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	t.Run("observability disabled by default", func(t *testing.T) {
		cfg, err := New()
		require.NoError(t, err)

		assert.False(t, cfg.ToolCallEvents.Enabled, "Tool call events should be disabled by default")
		assert.Equal(t, 10, cfg.ToolCallEvents.BatchSize, "Default batch size should be 10")
		assert.Equal(t, 100, cfg.ToolCallEvents.SampleRate, "Default sample rate should be 100%")
	})

	t.Run("can enable observability via environment variables", func(t *testing.T) {
		os.Setenv("ALPINE_TOOL_CALL_EVENTS_ENABLED", "true")
		os.Setenv("ALPINE_TOOL_CALL_BATCH_SIZE", "25")
		os.Setenv("ALPINE_TOOL_CALL_SAMPLE_RATE", "75")

		cfg, err := New()
		require.NoError(t, err)

		assert.True(t, cfg.ToolCallEvents.Enabled, "Tool call events should be enabled")
		assert.Equal(t, 25, cfg.ToolCallEvents.BatchSize, "Batch size should be configurable")
		assert.Equal(t, 75, cfg.ToolCallEvents.SampleRate, "Sample rate should be configurable")
	})

	t.Run("validates configuration values", func(t *testing.T) {
		// Test invalid batch size
		os.Setenv("ALPINE_TOOL_CALL_BATCH_SIZE", "0")
		_, err := New()
		assert.Error(t, err, "Should reject zero batch size")

		os.Setenv("ALPINE_TOOL_CALL_BATCH_SIZE", "-5")
		_, err = New()
		assert.Error(t, err, "Should reject negative batch size")

		// Test invalid sample rate
		os.Setenv("ALPINE_TOOL_CALL_BATCH_SIZE", "10") // Reset to valid
		os.Setenv("ALPINE_TOOL_CALL_SAMPLE_RATE", "0")
		_, err = New()
		assert.Error(t, err, "Should reject zero sample rate")

		os.Setenv("ALPINE_TOOL_CALL_SAMPLE_RATE", "101")
		_, err = New()
		assert.Error(t, err, "Should reject sample rate over 100")
	})

	t.Run("provides sensible defaults for all observability settings", func(t *testing.T) {
		// Clear any environment variables that might affect the test
		os.Unsetenv("ALPINE_TOOL_CALL_EVENTS_ENABLED")
		os.Unsetenv("ALPINE_TOOL_CALL_BATCH_SIZE")
		os.Unsetenv("ALPINE_TOOL_CALL_SAMPLE_RATE")

		cfg, err := New()
		require.NoError(t, err)

		// Tool call events should be disabled by default for safety
		assert.False(t, cfg.ToolCallEvents.Enabled)

		// But should have reasonable defaults when enabled
		assert.Equal(t, 10, cfg.ToolCallEvents.BatchSize, "Default batch size should prevent overwhelming")
		assert.Equal(t, 100, cfg.ToolCallEvents.SampleRate, "Default sample rate should capture all events")
	})
}
