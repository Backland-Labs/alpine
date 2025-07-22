// Package gitx provides git worktree management functionality for River CLI.
// It enables isolated task execution in separate git worktrees.
package gitx

import "context"

// WorktreeManager provides an interface for managing git worktrees.
// Implementations can use different backends (CLI, go-git, etc).
type WorktreeManager interface {
	// Create creates a new worktree for the given task name.
	// It returns a Worktree struct with the path, branch name, and parent repo.
	Create(ctx context.Context, taskName string) (*Worktree, error)

	// Cleanup removes the worktree directory and prunes it from git.
	// It should handle both the filesystem cleanup and git worktree removal.
	Cleanup(ctx context.Context, wt *Worktree) error
}

// Worktree represents a git worktree created for a task.
type Worktree struct {
	// Path is the absolute path to the worktree directory
	// Format: ../repo-river-<task>
	Path string

	// Branch is the branch name created for this worktree
	// Format: river/<sanitized-task>
	Branch string

	// ParentRepo is the absolute path to the main repository
	ParentRepo string
}
