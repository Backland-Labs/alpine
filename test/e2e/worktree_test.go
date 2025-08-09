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

	"github.com/Backland-Labs/alpine/internal/core"
)

// TestAlpineCreatesWorktree tests the complete workflow in an isolated worktree
func TestAlpineCreatesWorktree(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

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
	os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
	defer os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

	// Run alpine with worktree enabled
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, mockClaude,
		"implement new feature")
	t.Logf("Alpine output: %s", output)
	require.NoError(t, err, "Alpine command failed: %s", output)

	// Verify worktree was created
	worktrees := repo.GetWorktrees()
	t.Logf("Worktrees found: %v", worktrees)
	assert.Len(t, worktrees, 2, "Should have main repo and one worktree")

	// Find the worktree path (should be outside main repo)
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "alpine") {
			worktreePath = wt
			break
		}
	}

	// If we didn't find a worktree in the list, check if it exists but wasn't listed
	if worktreePath == "" {
		// The worktree should be in the parent directory
		parentDir := filepath.Dir(repo.RootDir)
		expectedName := filepath.Base(repo.RootDir) + "-alpine-implement-new-feature"
		worktreePath = filepath.Join(parentDir, expectedName)

		// Check if the directory exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Fatalf("Worktree directory not found at expected path: %s", worktreePath)
		}
	}

	// Verify worktree directory structure
	t.Logf("Worktree path: %s", worktreePath)
	expectedName := filepath.Base(repo.RootDir) + "-alpine-implement-new-feature"
	assert.Contains(t, filepath.Base(worktreePath), expectedName)

	// Verify branch was created
	branches := repo.GetBranches()
	t.Logf("Branches found: %v", branches)
	var foundBranch bool
	for _, branch := range branches {
		if strings.Contains(branch, "alpine/implement-new-feature") {
			foundBranch = true
			break
		}
	}
	assert.True(t, foundBranch, "Alpine branch not found")

	// Verify state file exists in worktree
	stateDir := filepath.Join(worktreePath, "agent_state")
	stateFile := filepath.Join(stateDir, "agent_state.json")
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

// TestAlpineWorktreeCleanup tests proper cleanup after completion/failure
func TestAlpineWorktreeCleanup(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Test successful completion with auto-cleanup
	t.Run("SuccessfulCompletionWithAutoCleanup", func(t *testing.T) {
		mockClaude := MockClaudeScript(t, []MockResponse{
			{
				PromptContains: "/run_implementation_loop",
				StateJSON: `{
  "current_step_description": "Task completed",
  "next_step_prompt": "",
  "status": "completed"
}`,
			},
		})

		ctx := context.Background()

		// Run with auto-cleanup enabled (default)
		output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, mockClaude,
			"--no-plan",
			"quick fix")
		require.NoError(t, err, "Alpine command failed: %s", output)

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
		os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
		defer os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

		// Run alpine expecting failure
		output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, failScript,
			"--no-plan",
			"failing task")
		t.Logf("Alpine output (failure): %s", output)
		assert.Error(t, err, "Expected error from failing Claude")

		// Verify worktree is still present (not cleaned up on failure)
		worktrees := repo.GetWorktrees()
		t.Logf("Worktrees after failure: %v", worktrees)
		assert.Greater(t, len(worktrees), 1, "Worktree should remain after failure")
	})
}

// TestAlpineWorktreeDisabled tests the --no-worktree flag behavior
func TestAlpineWorktreeDisabled(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Create mock Claude response
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/run_implementation_loop",
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
	output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, mockClaude,
		"--no-plan",
		"--no-worktree",
		"task without worktree")
	t.Logf("Alpine output: %s", output)
	require.NoError(t, err, "Alpine command failed: %s", output)

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
	stateDir := filepath.Join(repo.RootDir, "agent_state")
	stateFile := filepath.Join(stateDir, "agent_state.json")
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "State file should exist in main repo")
}

// TestAlpineWorktreeIsolation tests that worktrees provide proper isolation
func TestAlpineWorktreeIsolation(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Create a test file in main repo
	testFile := "test.txt"
	err := repo.CreateTestFile(testFile, "original content")
	require.NoError(t, err)
	repo.runGitCommand("add", testFile)
	repo.runGitCommand("commit", "-m", "Add test file")

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Note: mockClaude is created but not used, we use modifyScript instead
	// This is intentional as modifyScript needs to perform file modifications

	// Also create a script that modifies the file
	modifyScript := filepath.Join(t.TempDir(), "modify-claude.sh")
	err = os.WriteFile(modifyScript, []byte(fmt.Sprintf(`#!/bin/bash
# Mock Claude that modifies file and updates state

# Use ALPINE_STATE_FILE if set, otherwise default to agent_state/agent_state.json
mkdir -p agent_state
STATE_FILE="${ALPINE_STATE_FILE:-agent_state/agent_state.json}"

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
	os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
	defer os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

	// Run alpine (should create worktree)
	output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, modifyScript,
		"--no-plan",
		"modify test file")
	t.Logf("Alpine output: %s", output)
	require.NoError(t, err, "Alpine command failed: %s", output)

	// Check content in main repo (should be unchanged)
	mainContent, err := os.ReadFile(filepath.Join(repo.RootDir, testFile))
	require.NoError(t, err)
	assert.Equal(t, "original content", string(mainContent), "Main repo file should be unchanged")

	// Find worktree path
	worktrees := repo.GetWorktrees()
	t.Logf("Worktrees found: %v", worktrees)
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "alpine") {
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

// TestAlpineWorktreeBranchNaming tests branch name generation and collision handling
func TestAlpineWorktreeBranchNaming(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Create mock Claude
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/run_implementation_loop",
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
			expectedBranch: "alpine/fix-bug",
		},
		{
			name:           "Task with special characters",
			taskDesc:       "implement feature #123!",
			expectedBranch: "alpine/implement-feature-123",
		},
		{
			name:           "Long task name",
			taskDesc:       "this is a very long task description that should be truncated to a reasonable length",
			expectedBranch: "alpine/this-is-a-very-long-task-description-that-should-b",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Set environment to disable auto-cleanup for inspection
			os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
			defer os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

			// Run alpine
			output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, mockClaude,
				"--no-plan",
				tc.taskDesc)
			require.NoError(t, err, "Alpine command failed: %s", output)

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

// TestAlpineWorktreeEnvironmentVariables tests git configuration via environment variables
func TestAlpineWorktreeEnvironmentVariables(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Create a custom base branch
	repo.runGitCommand("checkout", "-b", "develop")
	repo.CreateTestFile("develop.txt", "develop branch file")
	repo.runGitCommand("add", ".")
	repo.runGitCommand("commit", "-m", "Add develop file")
	repo.runGitCommand("checkout", "main")

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Create mock Claude
	mockClaude := MockClaudeScript(t, []MockResponse{
		{
			PromptContains: "/run_implementation_loop",
			StateJSON: `{
  "current_step_description": "Task completed",
  "next_step_prompt": "",
  "status": "completed"
}`,
		},
	})

	ctx := context.Background()

	// Set custom environment variables
	os.Setenv("ALPINE_GIT_BASE_BRANCH", "develop")
	os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
	defer func() {
		os.Unsetenv("ALPINE_GIT_BASE_BRANCH")
		os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")
	}()

	// Run alpine
	output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, mockClaude,
		"--no-plan",
		"test from develop")
	require.NoError(t, err, "Alpine command failed: %s", output)

	// Find worktree
	worktrees := repo.GetWorktrees()
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "alpine") {
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

// TestWorktree_ClaudeExecutesInCorrectDirectory verifies that Claude commands
// execute in the worktree directory, not the original repository directory.
// This test ensures the fix for issue #7 works correctly.
func TestWorktree_ClaudeExecutesInCorrectDirectory(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Create a marker file in the main repo to detect wrong execution directory
	mainRepoMarker := filepath.Join(repo.RootDir, "main-repo-marker.txt")
	require.NoError(t, os.WriteFile(mainRepoMarker, []byte("main repo"), 0644))

	// Commit the marker file
	repo.runGitCommand("add", ".")
	repo.runGitCommand("commit", "-m", "Add marker file")

	// Create a custom mock Claude script that writes its working directory
	scriptPath := filepath.Join(t.TempDir(), "mock-claude-pwd.sh")
	script := `#!/bin/bash
# Mock Claude script that records working directory

# Get the prompt from command line
PROMPT="$@"

# Use ALPINE_STATE_FILE if set, otherwise default to agent_state/agent_state.json
mkdir -p agent_state
STATE_FILE="${ALPINE_STATE_FILE:-agent_state/agent_state.json}"

# Write working directory to a file
pwd > claude-working-dir.txt

# Write state based on prompt
case "$PROMPT" in
  *"/run_implementation_loop"*)
    cat > "$STATE_FILE" <<EOF
{
  "current_step_description": "Executed task and recorded working directory",
  "next_step_prompt": "",
  "status": "completed"
}
EOF
    echo "Mock Claude: Processed /run_implementation_loop"
    ;;
  *)
    echo "Mock Claude: Unknown prompt: $PROMPT"
    exit 1
    ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	// Disable auto-cleanup to inspect worktree after completion
	os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
	defer os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

	// Run alpine with worktree enabled (default)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, scriptPath,
		"--no-plan",
		"Check working directory and create marker file")
	require.NoError(t, err, "Alpine command failed: %s", output)

	// Find the worktree directory
	worktrees := repo.GetWorktrees()
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "alpine") {
			worktreePath = wt
			break
		}
	}
	require.NotEmpty(t, worktreePath, "No worktree found")

	// Read the working directory file created by Claude
	workingDirFile := filepath.Join(worktreePath, "claude-working-dir.txt")
	workingDirBytes, err := os.ReadFile(workingDirFile)
	require.NoError(t, err, "Claude should have created working directory file")

	claudeWorkingDir := strings.TrimSpace(string(workingDirBytes))

	// Verify Claude executed in the worktree directory
	assert.Equal(t, worktreePath, claudeWorkingDir,
		"Claude should execute in worktree directory, not main repo")

	// Verify the marker file exists only in main repo, not in worktree
	// (unless Claude copied it, which would be wrong)
	_, mainErr := os.Stat(mainRepoMarker)
	assert.NoError(t, mainErr, "Marker should exist in main repo")

	worktreeMarker := filepath.Join(worktreePath, "main-repo-marker.txt")
	_, wtErr := os.Stat(worktreeMarker)
	assert.NoError(t, wtErr, "Marker should also exist in worktree (from git)")
}

// TestWorktree_FileOperationsIsolated verifies that file operations performed
// by Claude in the worktree are isolated from the main repository.
func TestWorktree_FileOperationsIsolated(t *testing.T) {
	// Create test git repository
	repo := NewGitTestRepo(t)

	// Build alpine binary
	alpineBinary := BuildAlpineBinary(t)

	// Create initial file in main repo
	testFile := filepath.Join(repo.RootDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("original content"), 0644))
	repo.runGitCommand("add", ".")
	repo.runGitCommand("commit", "-m", "Add test file")

	// Create a custom mock Claude script that modifies files
	scriptPath := filepath.Join(t.TempDir(), "mock-claude-files.sh")
	script := `#!/bin/bash
# Mock Claude script that modifies files

# Get the prompt from command line
PROMPT="$@"

# Use ALPINE_STATE_FILE if set, otherwise default to agent_state/agent_state.json
mkdir -p agent_state
STATE_FILE="${ALPINE_STATE_FILE:-agent_state/agent_state.json}"

# Write state based on prompt
case "$PROMPT" in
  *"/run_implementation_loop"*)
    # Modify existing file and create new file
    echo "modified content" > test.txt
    echo "new file content" > new-file.txt
    
    cat > "$STATE_FILE" <<EOF
{
  "current_step_description": "Modified files in worktree",
  "next_step_prompt": "",
  "status": "completed"
}
EOF
    echo "Mock Claude: Processed /run_implementation_loop"
    ;;
  *)
    echo "Mock Claude: Unknown prompt: $PROMPT"
    exit 1
    ;;
esac
`
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)

	// Disable auto-cleanup to inspect worktree after completion
	os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
	defer os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

	// Run alpine with worktree enabled
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := RunAlpineWithMockClaude(ctx, t, alpineBinary, repo.RootDir, scriptPath,
		"--no-plan",
		"Modify files to test isolation")
	require.NoError(t, err, "Alpine command failed: %s", output)

	// Find the worktree directory
	worktrees := repo.GetWorktrees()
	var worktreePath string
	for _, wt := range worktrees {
		if wt != repo.RootDir && strings.Contains(wt, "alpine") {
			worktreePath = wt
			break
		}
	}
	require.NotEmpty(t, worktreePath, "No worktree found")

	// Verify main repo file is unchanged
	mainContent, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(mainContent),
		"Main repo file should be unchanged")

	// Verify worktree has modified file
	worktreeTestFile := filepath.Join(worktreePath, "test.txt")
	worktreeContent, err := os.ReadFile(worktreeTestFile)
	require.NoError(t, err)
	assert.Equal(t, "modified content\n", string(worktreeContent),
		"Worktree file should be modified")

	// Verify new file exists only in worktree
	mainNewFile := filepath.Join(repo.RootDir, "new-file.txt")
	_, err = os.Stat(mainNewFile)
	assert.True(t, os.IsNotExist(err),
		"New file should not exist in main repo")

	worktreeNewFile := filepath.Join(worktreePath, "new-file.txt")
	newFileContent, err := os.ReadFile(worktreeNewFile)
	require.NoError(t, err, "New file should exist in worktree")
	assert.Equal(t, "new file content\n", string(newFileContent))
}
