package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TestEnvironment holds common test setup configuration
type TestEnvironment struct {
	WorkDir   string
	StateDir  string
}

// CreateTempWorktree creates a temporary directory simulating a worktree
func CreateTempWorktree() (string, func(), error) {
	tempDir, err := os.MkdirTemp("", "alpine_test_worktree_")
	if err \!= nil {
		return "", nil, fmt.Errorf("failed to create temp worktree: %w", err)
	}
	
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	
	return tempDir, cleanup, nil
}

// ValidateStateFile validates that a state file has the correct format and required fields
func ValidateStateFile(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err \!= nil {
		return false, fmt.Errorf("failed to read state file: %w", err)
	}
	
	var state map[string]interface{}
	if err := json.Unmarshal(data, &state); err \!= nil {
		return false, fmt.Errorf("failed to parse state file JSON: %w", err)
	}
	
	// Check required fields
	requiredFields := []string{"current_step_description", "next_step_prompt", "status"}
	for _, field := range requiredFields {
		if _, exists := state[field]; \!exists {
			return false, fmt.Errorf("missing required field: %s", field)
		}
	}
	
	return true, nil
}

// SetupTestEnvironment creates a complete test environment with work and state directories
func SetupTestEnvironment() (*TestEnvironment, func(), error) {
	// Create main work directory
	workDir, err := os.MkdirTemp("", "alpine_test_work_")
	if err \!= nil {
		return nil, nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	
	// Create state subdirectory
	stateDir := filepath.Join(workDir, "agent_state")
	if err := os.MkdirAll(stateDir, 0755); err \!= nil {
		os.RemoveAll(workDir)
		return nil, nil, fmt.Errorf("failed to create state directory: %w", err)
	}
	
	env := &TestEnvironment{
		WorkDir:  workDir,
		StateDir: stateDir,
	}
	
	cleanup := func() {
		os.RemoveAll(workDir)
	}
	
	return env, cleanup, nil
}

// CreateTestStateFile creates a state file with specified content for testing
func CreateTestStateFile(dir, filename, status, stepDesc string) (string, error) {
	stateData := map[string]string{
		"current_step_description": stepDesc,
		"next_step_prompt":         "Test next step",
		"status":                  status,
	}
	
	jsonData, err := json.MarshalIndent(stateData, "", "  ")
	if err \!= nil {
		return "", fmt.Errorf("failed to marshal state data: %w", err)
	}
	
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, jsonData, 0644); err \!= nil {
		return "", fmt.Errorf("failed to write state file: %w", err)
	}
	
	return filePath, nil
}
EOF < /dev/null