//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// GitTestRepo represents a temporary git repository for testing
type GitTestRepo struct {
	RootDir string
	t       *testing.T
}

// NewGitTestRepo creates a new git repository for testing
func NewGitTestRepo(t *testing.T) *GitTestRepo {
	t.Helper()

	// Create temp directory
	tempDir := t.TempDir()
	
	repo := &GitTestRepo{
		RootDir: tempDir,
		t:       t,
	}

	// Initialize git repo
	repo.runGitCommand("init")
	repo.runGitCommand("config", "user.email", "test@example.com")
	repo.runGitCommand("config", "user.name", "Test User")
	
	// Create initial commit
	initialFile := filepath.Join(tempDir, "README.md")
	err := os.WriteFile(initialFile, []byte("# Test Repository\n"), 0644)
	require.NoError(t, err)
	
	repo.runGitCommand("add", ".")
	repo.runGitCommand("commit", "-m", "Initial commit")
	
	// Create main branch (some git versions default to master)
	repo.runGitCommand("branch", "-M", "main")
	
	return repo
}

// runGitCommand runs a git command in the test repository
func (r *GitTestRepo) runGitCommand(args ...string) string {
	r.t.Helper()
	
	cmd := exec.Command("git", args...)
	cmd.Dir = r.RootDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
	
	return string(output)
}

// GetWorktrees returns a list of all worktrees for the repository
func (r *GitTestRepo) GetWorktrees() []string {
	output := r.runGitCommand("worktree", "list", "--porcelain")
	
	var worktrees []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			worktree := strings.TrimPrefix(line, "worktree ")
			worktrees = append(worktrees, worktree)
		}
	}
	
	return worktrees
}

// GetBranches returns a list of all branches in the repository
func (r *GitTestRepo) GetBranches() []string {
	output := r.runGitCommand("branch", "-a")
	
	var branches []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			branches = append(branches, branch)
		}
	}
	
	return branches
}

// AssertWorktreeExists checks that a worktree exists at the given path
func (r *GitTestRepo) AssertWorktreeExists(t *testing.T, path string) {
	t.Helper()
	
	worktrees := r.GetWorktrees()
	for _, wt := range worktrees {
		if filepath.Clean(wt) == filepath.Clean(path) {
			return
		}
	}
	
	t.Errorf("Worktree not found at path: %s\nExisting worktrees: %v", path, worktrees)
}

// AssertBranchExists checks that a branch exists
func (r *GitTestRepo) AssertBranchExists(t *testing.T, branchName string) {
	t.Helper()
	
	branches := r.GetBranches()
	for _, branch := range branches {
		if branch == branchName {
			return
		}
	}
	
	t.Errorf("Branch not found: %s\nExisting branches: %v", branchName, branches)
}

// CreateTestFile creates a file in the repository with the given content
func (r *GitTestRepo) CreateTestFile(filename, content string) error {
	path := filepath.Join(r.RootDir, filename)
	dir := filepath.Dir(path)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	return os.WriteFile(path, []byte(content), 0644)
}

// BuildAlpineBinary builds the alpine binary for testing
func BuildAlpineBinary(t *testing.T) string {
	t.Helper()
	
	// Build alpine binary in temp location
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "alpine")
	
	// Get the project root directory (where this test file is at test/e2e/)
	projectRoot, err := filepath.Abs(filepath.Join(filepath.Dir(""), "..", ".."))
	require.NoError(t, err)
	
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/alpine")
	cmd.Dir = projectRoot
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build alpine binary: %v\nOutput: %s", err, output)
	}
	
	return binaryPath
}

// RunAlpineCommand runs the alpine CLI with the given arguments
func RunAlpineCommand(ctx context.Context, binaryPath string, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = workDir
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"ALPINE_CLAUDE_COMMAND=echo", // Use echo as mock claude command
		"ALPINE_TEST_MODE=true",
	)
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunAlpineWithMockClaude runs the alpine CLI with a mock claude script
func RunAlpineWithMockClaude(ctx context.Context, t *testing.T, binaryPath string, workDir string, mockScript string, args ...string) (string, error) {
	t.Helper()
	
	// Create a directory for our mock claude
	mockDir := t.TempDir()
	
	// Create a symlink named 'claude' to our mock script
	claudeLink := filepath.Join(mockDir, "claude")
	err := os.Symlink(mockScript, claudeLink)
	require.NoError(t, err)
	
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = workDir
	
	// Prepend our mock directory to PATH
	newPath := mockDir + ":" + os.Getenv("PATH")
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"PATH="+newPath,
		"ALPINE_TEST_MODE=true",
	)
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// WaitForFile waits for a file to exist with a timeout
func WaitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	
	return fmt.Errorf("timeout waiting for file: %s", path)
}

// MockClaudeScript creates a script that simulates Claude behavior
func MockClaudeScript(t *testing.T, responses []MockResponse) string {
	t.Helper()
	
	scriptPath := filepath.Join(t.TempDir(), "mock-claude.sh")
	
	script := `#!/bin/bash
# Mock Claude script for e2e testing

# Get the prompt from command line
PROMPT="$@"

# Use ALPINE_STATE_FILE if set, otherwise default to claude_state.json
STATE_FILE="${ALPINE_STATE_FILE:-claude_state.json}"

# Write state based on prompt
case "$PROMPT" in
`
	
	for _, resp := range responses {
		script += fmt.Sprintf(`  *"%s"*)
    cat > "$STATE_FILE" <<EOF
%s
EOF
    echo "Mock Claude: Processed %s"
    ;;
`, resp.PromptContains, resp.StateJSON, resp.PromptContains)
	}
	
	script += `  *)
    echo "Mock Claude: Unknown prompt"
    exit 1
    ;;
esac
`
	
	err := os.WriteFile(scriptPath, []byte(script), 0755)
	require.NoError(t, err)
	
	return scriptPath
}

// MockResponse represents a mock Claude response
type MockResponse struct {
	PromptContains string
	StateJSON      string
}