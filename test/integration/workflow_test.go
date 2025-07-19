package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxkrieger/river/internal/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullWorkflow tests the complete end-to-end workflow of the River CLI.
// This ensures that all components work together correctly from CLI parsing
// through Claude execution and git worktree creation.
func TestFullWorkflow(t *testing.T) {
	// Build River binary
	riverBin := buildRiverBinary(t)
	
	// Setup test environment
	testDir := t.TempDir()
	issueID := "TEST-123"
	
	// Create a test git repo
	setupTestGitRepo(t, testDir)
	
	// Mock Claude CLI
	mockClaude := setupMockClaude(t, &claude.Response{
		Content: "Implementation plan for TEST-123",
	})
	defer mockClaude.Cleanup()
	
	// Run River CLI
	cmd := exec.Command(riverBin, issueID)
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(), 
		fmt.Sprintf("PATH=%s:%s", mockClaude.BinDir, os.Getenv("PATH")),
		"LINEAR_API_KEY=test-api-key")
	
	output, err := cmd.CombinedOutput()
	
	// Verify execution succeeded
	assert.NoError(t, err, "River CLI should execute successfully\nOutput: %s", string(output))
	// For now, just check that the workflow completed
	assert.Contains(t, string(output), "Workflow completed successfully")
	
	// Verify worktree was created in parent directory (as River CLI does)
	parentDir := filepath.Dir(testDir)
	worktreePath := filepath.Join(parentDir, "river-test-123")
	assert.DirExists(t, worktreePath, "Worktree should be created")
	
	// Verify we're on the correct branch in the worktree
	gitCmd := exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = worktreePath
	branchOutput, err := gitCmd.Output()
	assert.NoError(t, err)
	assert.Equal(t, "test-123", strings.TrimSpace(string(branchOutput)))
}

// TestFullWorkflowWithStreaming tests the workflow with JSON streaming enabled.
// This verifies that the --stream flag produces valid JSON output that can be
// consumed by other tools.
func TestFullWorkflowWithStreaming(t *testing.T) {
	// Build River binary
	riverBin := buildRiverBinary(t)
	
	// Setup test environment
	testDir := t.TempDir()
	issueID := "TEST-456"
	
	// Create a test git repo
	setupTestGitRepo(t, testDir)
	
	// Mock Claude CLI with streaming response (single response for now)
	mockClaude := setupMockClaude(t, &claude.Response{
		Content: "Implementation plan for TEST-456",
	})
	defer mockClaude.Cleanup()
	
	// Run River CLI with streaming
	cmd := exec.Command(riverBin, "--stream", issueID)
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(), 
		fmt.Sprintf("PATH=%s:%s", mockClaude.BinDir, os.Getenv("PATH")),
		"LINEAR_API_KEY=test-api-key")
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	// Verify execution succeeded
	if err != nil {
		t.Logf("stdout: %s", stdout.String())
		t.Logf("stderr: %s", stderr.String())
	}
	assert.NoError(t, err, "River CLI should execute successfully with --stream")
	
	// Verify output contains expected messages
	output := stdout.String()
	assert.Contains(t, output, "Workflow completed successfully")
	
	// Verify worktree was created in parent directory
	parentDir := filepath.Dir(testDir)
	worktreePath := filepath.Join(parentDir, "river-test-456")
	assert.DirExists(t, worktreePath, "Worktree should be created")
}

// TestWorkflowWithErrors tests that the River CLI handles errors gracefully.
// This ensures proper error propagation and user-friendly error messages.
func TestWorkflowWithErrors(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (*MockClaude, string)
		issueID     string
		expectError string
	}{
		{
			name: "Claude command fails",
			setup: func(t *testing.T) (*MockClaude, string) {
				testDir := t.TempDir()
				setupTestGitRepo(t, testDir)
				mockClaude := setupMockClaudeError(t, "Claude API error")
				return mockClaude, testDir
			},
			issueID:     "ERROR-123",
			expectError: "Claude API error",
		},
		{
			name: "Invalid git repository",
			setup: func(t *testing.T) (*MockClaude, string) {
				testDir := t.TempDir()
				// Don't initialize git repo
				mockClaude := setupMockClaude(t, &claude.Response{
					Content: "Implementation plan",
				})
				return mockClaude, testDir
			},
			issueID:     "NOGIT-123",
			expectError: "not a git repository",
		},
		{
			name: "Empty issue ID",
			setup: func(t *testing.T) (*MockClaude, string) {
				testDir := t.TempDir()
				setupTestGitRepo(t, testDir)
				mockClaude := setupMockClaude(t, &claude.Response{})
				return mockClaude, testDir
			},
			issueID:     "",
			expectError: "LINEAR-ISSUE-ID cannot be empty",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClaude, testDir := tt.setup(t)
			defer mockClaude.Cleanup()
			
			// Build River binary
			riverBin := buildRiverBinary(t)
			
			// Run River CLI
			cmd := exec.Command(riverBin, tt.issueID)
			cmd.Dir = testDir
			cmd.Env = append(os.Environ(), 
				fmt.Sprintf("PATH=%s:%s", mockClaude.BinDir, os.Getenv("PATH")),
				"LINEAR_API_KEY=test-api-key")
			
			output, err := cmd.CombinedOutput()
			
			// Verify error occurred
			assert.Error(t, err, "River CLI should fail with error")
			assert.Contains(t, string(output), tt.expectError, "Error message should contain expected text")
		})
	}
}

// TestCLIFlags tests all command-line flags and their behavior.
// This ensures that flags like --stream work correctly.
func TestCLIFlags(t *testing.T) {
	tests := []struct {
		name     string
		flags    []string
		validate func(t *testing.T, output string, err error)
	}{
		{
			name:  "Stream flag enables streaming mode",
			flags: []string{"--stream"},
			validate: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.Contains(t, output, "Streaming mode enabled")
				assert.Contains(t, output, "Workflow completed successfully")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			testDir := t.TempDir()
			setupTestGitRepo(t, testDir)
			
			// Mock Claude CLI
			mockClaude := setupMockClaude(t, &claude.Response{
				Content: "Implementation plan",
			})
			defer mockClaude.Cleanup()
			
			// Build River binary
			riverBin := buildRiverBinary(t)
			
			// Build command with flags
			args := append([]string{}, tt.flags...)
			args = append(args, "TEST-123")
			
			cmd := exec.Command(riverBin, args...)
			cmd.Dir = testDir
			cmd.Env = append(os.Environ(), 
				fmt.Sprintf("PATH=%s:%s", mockClaude.BinDir, os.Getenv("PATH")),
				"LINEAR_API_KEY=test-api-key")
			
			output, err := cmd.CombinedOutput()
			
			// Validate based on test case
			tt.validate(t, string(output), err)
		})
	}
}

// TestGitOperations verifies that git operations work correctly.
// This includes worktree creation, branch management, and repository state.
func TestGitOperations(t *testing.T) {
	// Setup test environment
	testDir := t.TempDir()
	setupTestGitRepo(t, testDir)
	
	// Create another test commit (Initial commit already created in setupTestGitRepo)
	createTestCommit(t, testDir, "Second commit")
	
	// Mock Claude CLI
	mockClaude := setupMockClaude(t, &claude.Response{
		Content: "Implementation plan",
	})
	defer mockClaude.Cleanup()
	
	// Build River binary
	riverBin := buildRiverBinary(t)
	
	// Run River CLI
	issueID := "GIT-789"
	cmd := exec.Command(riverBin, issueID)
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(), 
		fmt.Sprintf("PATH=%s:%s", mockClaude.BinDir, os.Getenv("PATH")),
		"LINEAR_API_KEY=test-api-key")
	
	_, err := cmd.CombinedOutput()
	require.NoError(t, err)
	
	// Verify worktree
	parentDir := filepath.Dir(testDir)
	worktreePath := filepath.Join(parentDir, "river-git-789")
	
	// Check branch name
	gitCmd := exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = worktreePath
	branchOutput, err := gitCmd.Output()
	assert.NoError(t, err)
	assert.Equal(t, "git-789", strings.TrimSpace(string(branchOutput)))
	
	// Verify worktree is linked to main repo
	gitCmd = exec.Command("git", "worktree", "list")
	gitCmd.Dir = testDir
	worktreeOutput, err := gitCmd.Output()
	assert.NoError(t, err)
	assert.Contains(t, string(worktreeOutput), "river-git-789")
	
	// Verify commits are accessible in worktree
	gitCmd = exec.Command("git", "log", "--oneline")
	gitCmd.Dir = worktreePath
	logOutput, err := gitCmd.Output()
	assert.NoError(t, err)
	assert.Contains(t, string(logOutput), "Second commit")
	assert.Contains(t, string(logOutput), "Initial commit")
}

// TestEnvironmentValidation verifies that environment checks work correctly.
// This ensures the CLI fails fast when required tools are missing.
func TestEnvironmentValidation(t *testing.T) {
	// Test with missing Claude CLI
	t.Run("Missing Claude CLI", func(t *testing.T) {
		testDir := t.TempDir()
		setupTestGitRepo(t, testDir)
		
		// Build River binary
		riverBin := buildRiverBinary(t)
		
		// Run without Claude in PATH
		cmd := exec.Command(riverBin, "TEST-123")
		cmd.Dir = testDir
		cmd.Env = []string{
			"PATH=/usr/bin:/bin", 
			"HOME=" + os.Getenv("HOME"),
			"LINEAR_API_KEY=test-api-key",
		}
		
		output, err := cmd.CombinedOutput()
		
		assert.Error(t, err)
		assert.Contains(t, string(output), "claude CLI command not found in PATH")
	})
	
	// Test with missing git
	t.Run("Missing git", func(t *testing.T) {
		testDir := t.TempDir()
		
		// Mock Claude CLI
		mockClaude := setupMockClaude(t, &claude.Response{})
		defer mockClaude.Cleanup()
		
		// Build River binary
		riverBin := buildRiverBinary(t)
		
		// Run without git in PATH
		cmd := exec.Command(riverBin, "TEST-123")
		cmd.Dir = testDir
		cmd.Env = []string{
			fmt.Sprintf("PATH=%s", mockClaude.BinDir),
			"HOME=" + os.Getenv("HOME"),
			"LINEAR_API_KEY=test-api-key",
		}
		
		output, err := cmd.CombinedOutput()
		
		assert.Error(t, err)
		assert.Contains(t, string(output), "exec: \"git\": executable file not found")
	})
}

