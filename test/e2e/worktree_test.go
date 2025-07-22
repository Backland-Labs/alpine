//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maxmcd/river/internal/core"
)

// TestRiverCreatesWorktree tests the complete workflow in an isolated worktree
func TestRiverCreatesWorktree(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)
	
	// Build river binary
	riverBinary := BuildRiverBinary(t)
	
	// Create mock Claude responses
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/make_plan",
			StateJSON: `{
  "current_step_description": "Created plan for implementing feature",
  "next_step_prompt": "/implement feature",
  "status": "running"
}`,
		},
		{
			PromptContains: "/implement feature",
			StateJSON: `{
  "current_step_description": "Implemented the feature successfully",
  "next_step_prompt": "/test feature",
  "status": "running"
}`,
		},
		{
			PromptContains: "/test feature",
			StateJSON: `{
  "current_step_description": "All tests passing",
  "next_step_prompt": "",
  "status": "completed"
}`,
		},
	})
	
	// Disable auto-cleanup for inspection
	os.Setenv("RIVER_GIT_AUTO_CLEANUP", "false")
	defer os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")
	
	// Run river with worktree enabled
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, mockClaude,
		"implement new feature")
	t.Logf("River output: %s", output)
	require.NoError(t, err, "River command failed: %s", output)
	
	// Verify worktree was created
	worktrees := repo.GetWorktrees()
	t.Logf("Worktrees found: %v", worktrees)
	assert.Len(t, worktrees, 2, "Should have main repo and one worktree")
	
	// Find the worktree path (should be outside main repo)
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "river") {
			worktreePath = wt
			break
		}
	}
	
	// If we didn't find a worktree in the list, check if it exists but wasn't listed
	if worktreePath == "" {
		// The worktree should be in the parent directory
		parentDir := filepath.Dir(repo.RootDir)
		expectedName := filepath.Base(repo.RootDir) + "-river-implement-new-feature"
		worktreePath = filepath.Join(parentDir, expectedName)
		
		// Check if the directory exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Fatalf("Worktree directory not found at expected path: %s", worktreePath)
		}
	}
	
	// Verify worktree directory structure
	t.Logf("Worktree path: %s", worktreePath)
	expectedName := filepath.Base(repo.RootDir) + "-river-implement-new-feature"
	assert.Contains(t, filepath.Base(worktreePath), expectedName)
	
	// Verify branch was created
	branches := repo.GetBranches()
	t.Logf("Branches found: %v", branches)
	var foundBranch bool
	for _, branch := range branches {
		if strings.Contains(branch, "river/implement-new-feature") {
			foundBranch = true
			break
		}
	}
	assert.True(t, foundBranch, "River branch not found")
	
	// Verify state file exists in worktree
	stateFile := filepath.Join(worktreePath, "claude_state.json")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "State file should exist in worktree")
	
	// Verify final state
	var state core.State
	data, err := os.ReadFile(stateFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)
	
	assert.Equal(t, "completed", state.Status)
	assert.Equal(t, "All tests passing", state.CurrentStepDescription)
}

// TestRiverWorktreeCleanup tests proper cleanup after completion/failure
func TestRiverWorktreeCleanup(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)
	
	// Build river binary
	riverBinary := BuildRiverBinary(t)
	
	// Test successful completion with auto-cleanup
	t.Run("SuccessfulCompletionWithAutoCleanup", func(t *testing.T) {
		mockClaude := MockClaudeScript(t, []MockResponse{
			{
				PromptContains: "/ralph",
				StateJSON: `{
  "current_step_description": "Task completed",
  "next_step_prompt": "",
  "status": "completed"
}`,
			},
		})
		
		ctx := context.Background()
		
		// Run with auto-cleanup enabled (default)
		output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, mockClaude,
			"--no-plan",
			"quick fix")
		require.NoError(t, err, "River command failed: %s", output)
		
		// Wait a bit for cleanup
		time.Sleep(100 * time.Millisecond)
		
		// Verify worktree was cleaned up
		worktrees := repo.GetWorktrees()
		assert.Len(t, worktrees, 1, "Only main repo should remain after cleanup")
	})
	
	// Test failure scenario
	t.Run("FailureScenario", func(t *testing.T) {
		// Create a script that simulates Claude failure
		failScript := filepath.Join(t.TempDir(), "fail-claude.sh")
		err := os.WriteFile(failScript, []byte(`#!/bin/bash
echo "Mock Claude: Simulating failure"
exit 1
`), 0755)
		require.NoError(t, err)
		
		ctx := context.Background()
		
		// Disable auto-cleanup to ensure worktree remains
		os.Setenv("RIVER_GIT_AUTO_CLEANUP", "false")
		defer os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")
		
		// Run river expecting failure
		output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, failScript,
			"--no-plan",
			"failing task")
		t.Logf("River output (failure): %s", output)
		assert.Error(t, err, "Expected error from failing Claude")
		
		// Verify worktree is still present (not cleaned up on failure)
		worktrees := repo.GetWorktrees()
		t.Logf("Worktrees after failure: %v", worktrees)
		assert.Greater(t, len(worktrees), 1, "Worktree should remain after failure")
	})
}

// TestRiverWorktreeDisabled tests the --no-worktree flag behavior
func TestRiverWorktreeDisabled(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)
	
	// Build river binary
	riverBinary := BuildRiverBinary(t)
	
	// Create mock Claude response
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/ralph",
			StateJSON: `{
  "current_step_description": "Task completed in main repo",
  "next_step_prompt": "",
  "status": "completed"
}`,
		},
	})
	
	ctx := context.Background()
	
	// Get initial branch
	initialBranch := strings.TrimSpace(repo.runGitCommand("branch", "--show-current"))
	
	// Run with --no-worktree flag
	output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, mockClaude,
		"--no-plan",
		"--no-worktree",
		"task without worktree")
	t.Logf("River output: %s", output)
	require.NoError(t, err, "River command failed: %s", output)
	
	// Verify no worktree was created
	worktrees := repo.GetWorktrees()
	assert.Len(t, worktrees, 1, "No additional worktree should be created")
	
	// Verify we're still on the same branch
	currentBranch := strings.TrimSpace(repo.runGitCommand("branch", "--show-current"))
	assert.Equal(t, initialBranch, currentBranch, "Should remain on original branch")
	
	// List files in main repo for debugging
	files, _ := os.ReadDir(repo.RootDir)
	var fileNames []string
	for _, f := range files {
		fileNames = append(fileNames, f.Name())
	}
	t.Logf("Files in main repo: %v", fileNames)
	
	// Verify state file exists in main repo
	stateFile := filepath.Join(repo.RootDir, "claude_state.json")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "State file should exist in main repo")
}

// TestRiverWorktreeIsolation tests that worktrees provide proper isolation
func TestRiverWorktreeIsolation(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)
	
	// Create a test file in main repo
	testFile := "test.txt"
	err := repo.CreateTestFile(testFile, "original content")
	require.NoError(t, err)
	repo.runGitCommand("add", testFile)
	repo.runGitCommand("commit", "-m", "Add test file")
	
	// Build river binary
	riverBinary := BuildRiverBinary(t)
	
	// Note: mockClaude is created but not used, we use modifyScript instead
	// This is intentional as modifyScript needs to perform file modifications
	
	// Also create a script that modifies the file
	modifyScript := filepath.Join(t.TempDir(), "modify-claude.sh")
	err = os.WriteFile(modifyScript, []byte(fmt.Sprintf(`#!/bin/bash
# Mock Claude that modifies file and updates state

# Use RIVER_STATE_FILE if set, otherwise default to claude_state.json
STATE_FILE="${RIVER_STATE_FILE:-claude_state.json}"

# Find the worktree directory (we're already in it)
echo "modified content" > %s

# Update state
cat > "$STATE_FILE" <<EOF
{
  "current_step_description": "Modified test file",
  "next_step_prompt": "",
  "status": "completed"
}
EOF

echo "Mock Claude: Modified file"
`, testFile)), 0755)
	require.NoError(t, err)
	
	ctx := context.Background()
	
	// Disable auto-cleanup for this test
	os.Setenv("RIVER_GIT_AUTO_CLEANUP", "false")
	defer os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")
	
	// Run river (should create worktree)
	output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, modifyScript,
		"--no-plan",
		"modify test file")
	t.Logf("River output: %s", output)
	require.NoError(t, err, "River command failed: %s", output)
	
	// Check content in main repo (should be unchanged)
	mainContent, err := os.ReadFile(filepath.Join(repo.RootDir, testFile))
	require.NoError(t, err)
	assert.Equal(t, "original content", string(mainContent), "Main repo file should be unchanged")
	
	// Find worktree path
	worktrees := repo.GetWorktrees()
	t.Logf("Worktrees found: %v", worktrees)
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "river") {
			worktreePath = wt
			break
		}
	}
	require.NotEmpty(t, worktreePath, "Worktree path should not be empty")
	
	// Check content in worktree (should be modified)
	wtContent, err := os.ReadFile(filepath.Join(worktreePath, testFile))
	require.NoError(t, err)
	assert.Contains(t, string(wtContent), "modified content", "Worktree file should be modified")
}

// TestRiverWorktreeBranchNaming tests branch name generation and collision handling
func TestRiverWorktreeBranchNaming(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)
	
	// Build river binary
	riverBinary := BuildRiverBinary(t)
	
	// Create mock Claude
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/ralph",
			StateJSON: `{
  "current_step_description": "Task done",
  "next_step_prompt": "",
  "status": "completed"
}`,
		},
	})
	
	testCases := []struct {
		name           string
		taskDesc       string
		expectedBranch string
	}{
		{
			name:           "Simple task name",
			taskDesc:       "fix bug",
			expectedBranch: "river/fix-bug",
		},
		{
			name:           "Task with special characters",
			taskDesc:       "implement feature #123!",
			expectedBranch: "river/implement-feature-123",
		},
		{
			name:           "Long task name",
			taskDesc:       "this is a very long task description that should be truncated to a reasonable length",
			expectedBranch: "river/this-is-a-very-long-task-description-that-should-b",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			
			// Set environment to disable auto-cleanup for inspection
			os.Setenv("RIVER_GIT_AUTO_CLEANUP", "false")
			defer os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")
			
			// Run river
			output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, mockClaude,
				"--no-plan",
				tc.taskDesc)
			require.NoError(t, err, "River command failed: %s", output)
			
			// Check that expected branch was created
			branches := repo.GetBranches()
			found := false
			for _, branch := range branches {
				if strings.Contains(branch, tc.expectedBranch) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected branch %s not found in %v", tc.expectedBranch, branches)
		})
	}
}

// TestRiverWorktreeEnvironmentVariables tests git configuration via environment variables
func TestRiverWorktreeEnvironmentVariables(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)
	
	// Create a custom base branch
	repo.runGitCommand("checkout", "-b", "develop")
	repo.CreateTestFile("develop.txt", "develop branch file")
	repo.runGitCommand("add", ".")
	repo.runGitCommand("commit", "-m", "Add develop file")
	repo.runGitCommand("checkout", "main")
	
	// Build river binary
	riverBinary := BuildRiverBinary(t)
	
	// Create mock Claude
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/ralph",
			StateJSON: `{
  "current_step_description": "Task completed",
  "next_step_prompt": "",
  "status": "completed"
}`,
		},
	})
	
	ctx := context.Background()
	
	// Set custom environment variables
	os.Setenv("RIVER_GIT_BASE_BRANCH", "develop")
	os.Setenv("RIVER_GIT_AUTO_CLEANUP", "false")
	defer func() {
		os.Unsetenv("RIVER_GIT_BASE_BRANCH")
		os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")
	}()
	
	// Run river
	output, err := RunRiverWithMockClaude(ctx, t, riverBinary, repo.RootDir, mockClaude,
		"--no-plan",
		"test from develop")
	require.NoError(t, err, "River command failed: %s", output)
	
	// Find worktree
	worktrees := repo.GetWorktrees()
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "river") {
			worktreePath = wt
			break
		}
	}
	require.NotEmpty(t, worktreePath)
	
	// Verify worktree has file from develop branch
	developFile := filepath.Join(worktreePath, "develop.txt")
	_, err = os.Stat(developFile)
	assert.NoError(t, err, "Worktree should be based on develop branch")
	
	// Verify worktree was not auto-cleaned (due to env var)
	time.Sleep(100 * time.Millisecond)
	worktrees = repo.GetWorktrees()
	assert.Greater(t, len(worktrees), 1, "Worktree should not be auto-cleaned")
}