package gitx

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CLIWorktreeManager implements WorktreeManager using git CLI commands.
type CLIWorktreeManager struct {
	parentRepo string
	baseBranch string
}

// NewCLIWorktreeManager creates a new WorktreeManager that uses git CLI.
func NewCLIWorktreeManager(parentRepo, baseBranch string) WorktreeManager {
	return &CLIWorktreeManager{
		parentRepo: parentRepo,
		baseBranch: baseBranch,
	}
}

// Create creates a new worktree for the given task.
func (m *CLIWorktreeManager) Create(ctx context.Context, taskName string) (*Worktree, error) {
	// Sanitize task name for branch and directory
	sanitized := sanitizeTaskName(taskName)

	// Get existing branches to check for conflicts
	branches, err := m.listBranches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Generate unique branch name
	baseBranchName := fmt.Sprintf("river/%s", sanitized)
	branch := generateUniqueBranchName(baseBranchName, branches)

	// Create worktree path - include branch suffix if present
	repoName := filepath.Base(m.parentRepo)
	var wtDir string
	if branch != baseBranchName {
		// Extract suffix from branch name for directory
		suffix := strings.TrimPrefix(branch, baseBranchName)
		wtDir = fmt.Sprintf("%s-river-%s%s", repoName, sanitized, suffix)
	} else {
		wtDir = fmt.Sprintf("%s-river-%s", repoName, sanitized)
	}
	wtPath := filepath.Join(filepath.Dir(m.parentRepo), wtDir)

	// Create worktree with new branch
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-B", branch, wtPath, m.baseBranch)
	cmd.Dir = m.parentRepo

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}

	return &Worktree{
		Path:       wtPath,
		Branch:     branch,
		ParentRepo: m.parentRepo,
	}, nil
}

// Cleanup removes the worktree and cleans up git references.
func (m *CLIWorktreeManager) Cleanup(ctx context.Context, wt *Worktree) error {
	// Remove the worktree
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", wt.Path, "--force")
	cmd.Dir = m.parentRepo

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the worktree is already gone, try to prune it
		if _, statErr := os.Stat(wt.Path); os.IsNotExist(statErr) {
			// Directory doesn't exist, just prune
			pruneCmd := exec.CommandContext(ctx, "git", "worktree", "prune")
			pruneCmd.Dir = m.parentRepo
			if pruneErr := pruneCmd.Run(); pruneErr != nil {
				return fmt.Errorf("failed to prune worktree: %w", pruneErr)
			}
			return nil
		}
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	// Prune any remaining references
	pruneCmd := exec.CommandContext(ctx, "git", "worktree", "prune")
	pruneCmd.Dir = m.parentRepo
	if err := pruneCmd.Run(); err != nil {
		return fmt.Errorf("failed to prune worktree: %w", err)
	}

	return nil
}

// listBranches returns all branch names in the repository.
func (m *CLIWorktreeManager) listBranches(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "-a", "--format=%(refname:short)")
	cmd.Dir = m.parentRepo

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	branches := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove origin/ prefix if present
			line = strings.TrimPrefix(line, "origin/")
			branches = append(branches, line)
		}
	}

	return branches, nil
}
