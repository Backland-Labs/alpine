package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupAgUIHooks tests setting up ag-ui event emitter hooks for HTTP mode
func TestSetupAgUIHooks(t *testing.T) {
	// Create a mock hook script for testing
	createMockHookScript := func(dir string) {
		hooksDir := filepath.Join(dir, "hooks")
		err := os.MkdirAll(hooksDir, 0755)
		require.NoError(t, err)

		hookScript := filepath.Join(hooksDir, "alpine-ag-ui-emitter.rs")
		err = os.WriteFile(hookScript, []byte("#!/usr/bin/env rust-script\n// Mock hook script for testing"), 0755)
		require.NoError(t, err)
	}

	// Test cases for ag-ui hook configuration
	t.Run("creates claude settings with ag-ui hook when HTTP mode enabled", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory to test settings creation
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		// Setup ag-ui hooks with event endpoint
		eventEndpoint := "http://localhost:9090/events"
		runID := "test-run-123"
		cleanup, err := e.SetupAgUIHooks(eventEndpoint, runID)
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

		// Verify PostToolUse hook exists
		postToolUse, ok := hooks["PostToolUse"].([]interface{})
		assert.True(t, ok, "PostToolUse hook should exist")
		assert.Len(t, postToolUse, 1, "Should have one PostToolUse matcher")

		// Verify hook configuration
		matcher := postToolUse[0].(map[string]interface{})
		assert.Equal(t, ".*", matcher["matcher"], "Matcher should be .* for all tools")

		hookList := matcher["hooks"].([]interface{})
		assert.Len(t, hookList, 1, "Should have one hook")

		hook := hookList[0].(map[string]interface{})
		assert.Equal(t, "command", hook["type"], "Hook type should be command")

		// Verify hook command path is absolute
		hookCommand := hook["command"].(string)
		assert.True(t, filepath.IsAbs(hookCommand), "Hook command path should be absolute")
		assert.Contains(t, hookCommand, "alpine-ag-ui-emitter", "Hook should use ag-ui emitter script")
	})

	t.Run("resolves hook script path to absolute path", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		// Setup hooks
		cleanup, err := e.SetupAgUIHooks("http://localhost:9090/events", "run-123")
		require.NoError(t, err)
		defer cleanup()

		// Read settings and check hook path
		settingsData, err := os.ReadFile(filepath.Join(".claude", "settings.json"))
		require.NoError(t, err)

		var settings map[string]interface{}
		err = json.Unmarshal(settingsData, &settings)
		require.NoError(t, err)

		// Extract hook command path
		hooks := settings["hooks"].(map[string]interface{})
		postToolUse := hooks["PostToolUse"].([]interface{})
		matcher := postToolUse[0].(map[string]interface{})
		hookList := matcher["hooks"].([]interface{})
		hook := hookList[0].(map[string]interface{})
		hookCommand := hook["command"].(string)

		// Verify path is absolute and exists
		assert.True(t, filepath.IsAbs(hookCommand), "Hook path should be absolute")
		_, err = os.Stat(hookCommand)
		assert.NoError(t, err, "Hook script should exist at specified path")
	})

	t.Run("sets environment variables for hook context", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		eventEndpoint := "http://localhost:8080/events"
		runID := "unique-run-456"

		// Setup hooks
		cleanup, err := e.SetupAgUIHooks(eventEndpoint, runID)
		require.NoError(t, err)
		defer cleanup()

		// Verify environment variables are set in executor
		assert.Equal(t, eventEndpoint, e.envVars["ALPINE_EVENTS_ENDPOINT"])
		assert.Equal(t, runID, e.envVars["ALPINE_RUN_ID"])
	})

	t.Run("handles missing hook script gracefully", func(t *testing.T) {
		// Create a temp directory and move the hooks directory out of reach
		tmpDir := t.TempDir()

		// Change to a new empty directory where hooks won't be found
		emptyDir := filepath.Join(tmpDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		originalWd, _ := os.Getwd()
		err = os.Chdir(emptyDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Create executor in empty directory where hook script won't be found
		e := &Executor{}

		// Attempt to setup hooks with missing script
		_, err = e.SetupAgUIHooks("http://localhost:9090/events", "run-789")
		assert.Error(t, err, "Should error when hook script is missing")
		assert.Contains(t, err.Error(), "hook script", "Error should mention hook script")
	})

	t.Run("cleanup removes generated files", func(t *testing.T) {
		e := &Executor{}
		tmpDir := t.TempDir()

		// Change to temp directory
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Create mock hook script in current directory
		createMockHookScript(tmpDir)

		// Setup hooks
		cleanup, err := e.SetupAgUIHooks("http://localhost:9090/events", "run-cleanup")
		require.NoError(t, err)

		// Verify files exist before cleanup
		_, err = os.Stat(filepath.Join(".claude", "settings.json"))
		assert.NoError(t, err, "settings.json should exist before cleanup")

		// Run cleanup
		cleanup()

		// Verify settings.json is removed
		_, err = os.Stat(filepath.Join(".claude", "settings.json"))
		assert.True(t, os.IsNotExist(err), "settings.json should be removed after cleanup")

		// .claude directory should remain (may contain user settings)
		_, err = os.Stat(".claude")
		assert.NoError(t, err, ".claude directory should remain after cleanup")
	})
}

// TestGetAgUIHookScript tests retrieving the ag-ui hook script
func TestGetAgUIHookScript(t *testing.T) {
	t.Run("returns valid ag-ui hook script path", func(t *testing.T) {
		// Create a temporary directory with hook script for testing
		tmpDir := t.TempDir()
		originalWd, _ := os.Getwd()
		err := os.Chdir(tmpDir)
		require.NoError(t, err)
		defer os.Chdir(originalWd)

		// Create hooks directory and script
		hooksDir := filepath.Join(tmpDir, "hooks")
		err = os.MkdirAll(hooksDir, 0755)
		require.NoError(t, err)

		hookScript := filepath.Join(hooksDir, "alpine-ag-ui-emitter.rs")
		err = os.WriteFile(hookScript, []byte("#!/usr/bin/env rust-script\n// Mock hook script"), 0755)
		require.NoError(t, err)

		e := &Executor{}
		scriptPath, err := e.GetAgUIHookScriptPath()
		require.NoError(t, err)

		// Verify path points to ag-ui emitter script
		assert.Contains(t, scriptPath, "alpine-ag-ui-emitter.rs")

		// Verify script exists
		_, err = os.Stat(scriptPath)
		assert.NoError(t, err, "ag-ui hook script should exist")
	})
}
