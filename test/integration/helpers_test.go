package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxkrieger/river/internal/claude"
	"github.com/stretchr/testify/require"
)

// MockClaude represents a mock Claude CLI installation
type MockClaude struct {
	BinDir  string
	Cleanup func()
}

// setupTestGitRepo initializes a git repository for testing
func setupTestGitRepo(t *testing.T, dir string) {
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	err := cmd.Run()
	require.NoError(t, err, "Failed to initialize git repo")
	
	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)
	
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)
	
	// Create initial commit
	createTestCommit(t, dir, "Initial commit")
}

// createTestCommit creates a test commit in the repository
func createTestCommit(t *testing.T, dir, message string) {
	// Create a file
	filename := fmt.Sprintf("file-%d.txt", len(message))
	err := os.WriteFile(filepath.Join(dir, filename), []byte(message), 0644)
	require.NoError(t, err)
	
	// Add and commit
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)
	
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	err = cmd.Run()
	require.NoError(t, err)
}

// setupMockClaude creates a mock Claude executable that returns a single response
func setupMockClaude(t *testing.T, response *claude.Response) *MockClaude {
	binDir := filepath.Join(t.TempDir(), "bin")
	err := os.MkdirAll(binDir, 0755)
	require.NoError(t, err)
	
	// Create mock Claude executable
	claudePath := filepath.Join(binDir, "claude")
	mockScript := fmt.Sprintf(`#!/bin/bash
echo '%s'
`, mustMarshalJSON(response))
	
	err = os.WriteFile(claudePath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	return &MockClaude{
		BinDir: binDir,
		Cleanup: func() {
			os.RemoveAll(binDir)
		},
	}
}

// setupMockClaudeStreaming creates a mock Claude that outputs multiple JSON responses
func setupMockClaudeStreaming(t *testing.T, responses []claude.Response) *MockClaude {
	binDir := filepath.Join(t.TempDir(), "bin")
	err := os.MkdirAll(binDir, 0755)
	require.NoError(t, err)
	
	// Create mock Claude executable that outputs multiple JSON lines
	claudePath := filepath.Join(binDir, "claude")
	var lines []string
	for _, resp := range responses {
		lines = append(lines, fmt.Sprintf("echo '%s'", mustMarshalJSON(resp)))
	}
	
	mockScript := fmt.Sprintf(`#!/bin/bash
%s
`, strings.Join(lines, "\n"))
	
	err = os.WriteFile(claudePath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	return &MockClaude{
		BinDir: binDir,
		Cleanup: func() {
			os.RemoveAll(binDir)
		},
	}
}

// setupMockClaudeError creates a mock Claude that returns an error
func setupMockClaudeError(t *testing.T, errorMsg string) *MockClaude {
	binDir := filepath.Join(t.TempDir(), "bin")
	err := os.MkdirAll(binDir, 0755)
	require.NoError(t, err)
	
	// Create mock Claude executable that fails
	claudePath := filepath.Join(binDir, "claude")
	mockScript := fmt.Sprintf(`#!/bin/bash
echo '%s' >&2
exit 1
`, errorMsg)
	
	err = os.WriteFile(claudePath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	return &MockClaude{
		BinDir: binDir,
		Cleanup: func() {
			os.RemoveAll(binDir)
		},
	}
}

// mustMarshalJSON marshals a value to JSON or panics
func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

// contains checks if a slice contains an item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// buildRiverBinary builds the River CLI binary for testing
func buildRiverBinary(t *testing.T) string {
	binPath := filepath.Join(t.TempDir(), "river")
	
	cmd := exec.Command("go", "build", "-o", binPath, "../../cmd/river")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build river binary: %s", string(output))
	
	return binPath
}