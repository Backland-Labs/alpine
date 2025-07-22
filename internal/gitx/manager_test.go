package gitx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateWorktree_basic tests creating a worktree with basic functionality
func TestCreateWorktree_basic(t *testing.T) {
	// This test verifies that a worktree is created with:
	// - Correct directory structure
	// - Correct branch name
	// - Proper git worktree registration
	ctx := context.Background()

	// Create a temporary repository for testing
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	// Initialize git repo
	if err := exec.Command("git", "init", repoDir).Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set git config
	gitConfig := [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	}
	for _, args := range gitConfig {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config: %v", err)
		}
	}

	// Create initial commit
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmds := [][]string{
		{"add", "."},
		{"commit", "-m", "Initial commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create initial commit: %v", err)
		}
	}

	// Create manager
	manager := NewCLIWorktreeManager(repoDir, "main")

	// Create worktree
	taskName := "implement feature X"
	wt, err := manager.Create(ctx, taskName)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify worktree properties
	expectedBranch := "river/implement-feature-x"
	if wt.Branch != expectedBranch {
		t.Errorf("Branch = %q, want %q", wt.Branch, expectedBranch)
	}

	if wt.ParentRepo != repoDir {
		t.Errorf("ParentRepo = %q, want %q", wt.ParentRepo, repoDir)
	}

	// Verify directory exists
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Errorf("Worktree directory does not exist: %s", wt.Path)
	}

	// Verify git worktree is registered
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if !strings.Contains(string(output), wt.Path) {
		t.Errorf("Worktree not found in git worktree list")
	}
}

// TestCleanup_removesWorktreeDirAndPrunes tests proper cleanup of worktrees
func TestCleanup_removesWorktreeDirAndPrunes(t *testing.T) {
	// This test verifies that cleanup:
	// - Removes the worktree directory
	// - Prunes the worktree from git's registry
	ctx := context.Background()

	// Setup test repository
	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")

	// Initialize and setup repo (same as above)
	setupTestRepo(t, repoDir)

	// Create manager and worktree
	manager := NewCLIWorktreeManager(repoDir, "main")
	wt, err := manager.Create(ctx, "test cleanup")
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify worktree exists before cleanup
	if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
		t.Fatalf("Worktree should exist before cleanup")
	}

	// Cleanup
	if err := manager.Cleanup(ctx, wt); err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(wt.Path); !os.IsNotExist(err) {
		t.Errorf("Worktree directory should be removed after cleanup")
	}

	// Verify worktree is pruned from git
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list worktrees: %v", err)
	}

	if strings.Contains(string(output), wt.Path) {
		t.Errorf("Worktree should be pruned from git after cleanup")
	}
}

// TestBranchNameCollisionProducesUniqueNames tests handling of branch name conflicts
func TestBranchNameCollisionProducesUniqueNames(t *testing.T) {
	// This test verifies that when a branch name already exists,
	// the manager creates a unique branch name
	ctx := context.Background()

	tempDir := t.TempDir()
	repoDir := filepath.Join(tempDir, "test-repo")
	setupTestRepo(t, repoDir)

	manager := NewCLIWorktreeManager(repoDir, "main")

	// Create first worktree
	taskName := "duplicate task"
	wt1, err := manager.Create(ctx, taskName)
	if err != nil {
		t.Fatalf("First Create() failed: %v", err)
	}

	// Create second worktree with same task name
	wt2, err := manager.Create(ctx, taskName)
	if err != nil {
		t.Fatalf("Second Create() failed: %v", err)
	}

	// Verify branches are different
	if wt1.Branch == wt2.Branch {
		t.Errorf("Branch names should be unique, got %q for both", wt1.Branch)
	}

	// Verify second branch has a suffix
	expectedPrefix := "river/duplicate-task"
	if !strings.HasPrefix(wt2.Branch, expectedPrefix) {
		t.Errorf("Second branch should have prefix %q, got %q", expectedPrefix, wt2.Branch)
	}

	// Cleanup
	_ = manager.Cleanup(ctx, wt1)
	_ = manager.Cleanup(ctx, wt2)
}

// TestErrorPropagation_gitFailure tests error handling for git failures
func TestErrorPropagation_gitFailure(t *testing.T) {
	// This test verifies that git command failures are properly
	// propagated as errors from the manager methods
	ctx := context.Background()

	// Test with non-existent repository
	manager := NewCLIWorktreeManager("/non/existent/repo", "main")

	_, err := manager.Create(ctx, "test task")
	if err == nil {
		t.Error("Create() should fail with non-existent repository")
	}

	// Test cleanup with invalid worktree
	invalidWT := &Worktree{
		Path:       "/non/existent/worktree",
		Branch:     "invalid",
		ParentRepo: "/non/existent/repo",
	}

	err = manager.Cleanup(ctx, invalidWT)
	if err == nil {
		t.Error("Cleanup() should fail with invalid worktree")
	}
}

// setupTestRepo is a helper to set up a git repository for testing
func setupTestRepo(t *testing.T, repoDir string) {
	t.Helper()

	// Initialize git repo
	if err := exec.Command("git", "init", repoDir).Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set git config
	gitConfig := [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
	}
	for _, args := range gitConfig {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config: %v", err)
		}
	}

	// Create initial commit
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmds := [][]string{
		{"add", "."},
		{"commit", "-m", "Initial commit"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create initial commit: %v", err)
		}
	}
}
