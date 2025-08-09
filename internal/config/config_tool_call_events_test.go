package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToolCallEventsConfiguration tests the new tool call events configuration
func TestToolCallEventsConfiguration(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"ALPINE_TOOL_CALL_EVENTS_ENABLED",
		"ALPINE_TOOL_CALL_BATCH_SIZE",
		"ALPINE_TOOL_CALL_SAMPLE_RATE",
	}

	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key)
	}

	// Restore environment after test
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("tool call events disabled by default", func(t *testing.T) {
		cfg, err := New()
		require.NoError(t, err)

		// Tool call events should be disabled by default
		assert.False(t, cfg.ToolCallEvents.Enabled, "Tool call events should be disabled by default")
		assert.Equal(t, 10, cfg.ToolCallEvents.BatchSize, "Default batch size should be 10")
		assert.Equal(t, 100, cfg.ToolCallEvents.SampleRate, "Default sample rate should be 100%")
	})

	t.Run("enables tool call events when environment variable set", func(t *testing.T) {
		os.Setenv("ALPINE_TOOL_CALL_EVENTS_ENABLED", "true")
		defer os.Unsetenv("ALPINE_TOOL_CALL_EVENTS_ENABLED")

		cfg, err := New()
		require.NoError(t, err)

		assert.True(t, cfg.ToolCallEvents.Enabled, "Tool call events should be enabled")
	})

	t.Run("configures custom batch size", func(t *testing.T) {
		os.Setenv("ALPINE_TOOL_CALL_BATCH_SIZE", "25")
		defer os.Unsetenv("ALPINE_TOOL_CALL_BATCH_SIZE")

		cfg, err := New()
		require.NoError(t, err)

		assert.Equal(t, 25, cfg.ToolCallEvents.BatchSize, "Batch size should be configurable")
	})

	t.Run("configures custom sample rate", func(t *testing.T) {
		os.Setenv("ALPINE_TOOL_CALL_SAMPLE_RATE", "50")
		defer os.Unsetenv("ALPINE_TOOL_CALL_SAMPLE_RATE")

		cfg, err := New()
		require.NoError(t, err)

		assert.Equal(t, 50, cfg.ToolCallEvents.SampleRate, "Sample rate should be configurable")
	})

	t.Run("validates batch size bounds", func(t *testing.T) {
		// Test invalid batch size
		os.Setenv("ALPINE_TOOL_CALL_BATCH_SIZE", "0")
		defer os.Unsetenv("ALPINE_TOOL_CALL_BATCH_SIZE")

		_, err := New()
		assert.Error(t, err, "Should error on invalid batch size")
		assert.Contains(t, err.Error(), "ALPINE_TOOL_CALL_BATCH_SIZE", "Error should mention batch size variable")
	})

	t.Run("validates sample rate bounds", func(t *testing.T) {
		// Test invalid sample rate
		os.Setenv("ALPINE_TOOL_CALL_SAMPLE_RATE", "150")
		defer os.Unsetenv("ALPINE_TOOL_CALL_SAMPLE_RATE")

		_, err := New()
		assert.Error(t, err, "Should error on sample rate > 100")
		assert.Contains(t, err.Error(), "ALPINE_TOOL_CALL_SAMPLE_RATE", "Error should mention sample rate variable")
	})
}
