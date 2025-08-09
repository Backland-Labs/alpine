package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupToolCallEventHooks tests setting up hooks for tool call event capture
func TestSetupToolCallEventHooks(t *testing.T) {
	// Create a mock hook script for testing
	createMockHookScript := func(dir string) {
		hooksDir := filepath.Join(dir, "hooks")
		err := os.MkdirAll(hooksDir, 0755)
		require.NoError(t, err)

		hookScript := filepath.Join(hooksDir, "alpine-ag-ui-emitter.rs")
		err = os.WriteFile(hookScript, []byte("#!/usr/bin/env rust-script\n// Mock hook script for testing"), 0755)
		require.NoError(t, err)
	}

	t.Run("configures PreToolUse and PostToolUse hooks when tool call events enabled", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory to test settings creation
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalWd) }()

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		// Setup tool call event hooks
		runID := "test-run-123"
		eventEndpoint := "http://localhost:9090/runs/" + runID + "/events"
		cleanup, err := e.SetupToolCallEventHooks(eventEndpoint, runID, 10, 100)
		require.NoError(t, err)
		defer cleanup()

		// Verify .claude directory was created
		_, err = os.Stat(".claude")
		assert.NoError(t, err, ".claude directory should be created")

		// Verify settings.json was created
		settingsPath := filepath.Join(".claude", "settings.json")
		_, err = os.Stat(settingsPath)
		assert.NoError(t, err, "settings.json should be created")

		// Read and verify settings content
		settingsData, err := os.ReadFile(settingsPath)
		require.NoError(t, err)

		var settings map[string]interface{}
		err = json.Unmarshal(settingsData, &settings)
		require.NoError(t, err)

		// Verify hooks structure
		hooks, ok := settings["hooks"].(map[string]interface{})
		assert.True(t, ok, "settings should have hooks object")

		// Verify PreToolUse hook exists
		preToolUse, ok := hooks["PreToolUse"].([]interface{})
		assert.True(t, ok, "PreToolUse hook should exist")
		assert.Len(t, preToolUse, 1, "Should have one PreToolUse matcher")

		// Verify PostToolUse hook exists
		postToolUse, ok := hooks["PostToolUse"].([]interface{})
		assert.True(t, ok, "PostToolUse hook should exist")
		assert.Len(t, postToolUse, 1, "Should have one PostToolUse matcher")

		// Verify both hooks use the same script
		preMatcher := preToolUse[0].(map[string]interface{})
		postMatcher := postToolUse[0].(map[string]interface{})

		assert.Equal(t, ".*", preMatcher["matcher"], "PreToolUse matcher should be .* for all tools")
		assert.Equal(t, ".*", postMatcher["matcher"], "PostToolUse matcher should be .* for all tools")
	})

	t.Run("sets tool call event environment variables", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalWd) }()

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		runID := "unique-run-456"
		eventEndpoint := "http://localhost:8080/runs/" + runID + "/events"
		batchSize := 25
		sampleRate := 75

		// Setup hooks
		cleanup, err := e.SetupToolCallEventHooks(eventEndpoint, runID, batchSize, sampleRate)
		require.NoError(t, err)
		defer cleanup()

		// Verify environment variables are set in executor
		assert.Equal(t, eventEndpoint, e.envVars["ALPINE_EVENTS_ENDPOINT"])
		assert.Equal(t, runID, e.envVars["ALPINE_RUN_ID"])
		assert.Equal(t, "25", e.envVars["ALPINE_TOOL_CALL_BATCH_SIZE"])
		assert.Equal(t, "75", e.envVars["ALPINE_TOOL_CALL_SAMPLE_RATE"])
	})

	t.Run("cleanup removes generated files and environment variables", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalWd) }()

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		// Setup hooks
		cleanup, err := e.SetupToolCallEventHooks("http://localhost:9090/events", "run-cleanup", 10, 100)
		require.NoError(t, err)

		// Verify files exist before cleanup
		_, err = os.Stat(filepath.Join(".claude", "settings.json"))
		assert.NoError(t, err, "settings.json should exist before cleanup")

		// Verify environment variables are set
		assert.NotEmpty(t, e.envVars["ALPINE_EVENTS_ENDPOINT"])
		assert.NotEmpty(t, e.envVars["ALPINE_RUN_ID"])

		// Run cleanup
		cleanup()

		// Verify settings.json is removed
		_, err = os.Stat(filepath.Join(".claude", "settings.json"))
		assert.True(t, os.IsNotExist(err), "settings.json should be removed after cleanup")

		// Verify environment variables are cleared
		_, exists := e.envVars["ALPINE_EVENTS_ENDPOINT"]
		assert.False(t, exists, "ALPINE_EVENTS_ENDPOINT should be cleared")
		_, exists = e.envVars["ALPINE_RUN_ID"]
		assert.False(t, exists, "ALPINE_RUN_ID should be cleared")
		_, exists = e.envVars["ALPINE_TOOL_CALL_BATCH_SIZE"]
		assert.False(t, exists, "ALPINE_TOOL_CALL_BATCH_SIZE should be cleared")
		_, exists = e.envVars["ALPINE_TOOL_CALL_SAMPLE_RATE"]
		assert.False(t, exists, "ALPINE_TOOL_CALL_SAMPLE_RATE should be cleared")
	})
}
