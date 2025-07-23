package helpers

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/maxmcd/river/internal/core"
)

// CreateTestState creates a test state file with the given status
func CreateTestState(t *testing.T, dir string, status string) string {
	t.Helper()

	stateFile := filepath.Join(dir, "claude_state.json")
	state := &core.State{
		CurrentStepDescription: "Test state",
		NextStepPrompt:         "/test",
		Status:                 status,
	}

	err := state.Save(stateFile)
	require.NoError(t, err)

	return stateFile
}

// CaptureOutput captures stdout during function execution
func CaptureOutput(fn func()) string {
	// Save current stdout
	oldStdout := os.Stdout

	// Create pipe for capturing
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function
	fn()

	// Close writer and restore stdout
	_ = w.Close()
	os.Stdout = oldStdout

	// Read captured output
	output, _ := io.ReadAll(r)
	return string(output)
}

// SetupTestEnvironment sets up a test environment and returns a cleanup function
func SetupTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Save current environment
	originalEnv := os.Environ()
	envMap := make(map[string]string)
	for _, env := range originalEnv {
		if len(env) > 0 {
			if idx := len(env) - 1 - len(env[1:]); idx > 0 {
				key := env[:idx]
				value := env[idx+1:]
				envMap[key] = value
			}
		}
	}

	// Set test-specific environment variables
	_ = os.Setenv("RIVER_TEST_MODE", "true")

	// Return cleanup function
	return func() {
		// Clear all environment variables
		os.Clearenv()

		// Restore original environment
		for key, value := range envMap {
			_ = os.Setenv(key, value)
		}
	}
}

// AssertFileExists checks that a file exists
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.NoError(t, err, "File should exist: %s", path)
}

// AssertFileNotExists checks that a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "File should not exist: %s", path)
}

// WriteTestFile writes content to a file for testing
func WriteTestFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

// CompareStates compares two states and returns true if they match
func CompareStates(state1, state2 *core.State) bool {
	if state1 == nil || state2 == nil {
		return state1 == state2
	}

	return state1.CurrentStepDescription == state2.CurrentStepDescription &&
		state1.NextStepPrompt == state2.NextStepPrompt &&
		state1.Status == state2.Status
}
