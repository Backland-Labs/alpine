// Package mock provides mock implementations of gitx interfaces for testing.
package mock

import (
	"context"
	"fmt"

	"github.com/maxmcd/alpine/internal/gitx"
)

// WorktreeManager is a mock implementation of gitx.WorktreeManager.
type WorktreeManager struct {
	// CreateFunc is called when Create is invoked
	CreateFunc func(ctx context.Context, taskName string) (*gitx.Worktree, error)

	// CleanupFunc is called when Cleanup is invoked
	CleanupFunc func(ctx context.Context, wt *gitx.Worktree) error

	// CreateCalls records calls to Create
	CreateCalls []struct {
		Ctx      context.Context
		TaskName string
	}

	// CleanupCalls records calls to Cleanup
	CleanupCalls []struct {
		Ctx context.Context
		WT  *gitx.Worktree
	}
}

// Create implements gitx.WorktreeManager.
func (m *WorktreeManager) Create(ctx context.Context, taskName string) (*gitx.Worktree, error) {
	m.CreateCalls = append(m.CreateCalls, struct {
		Ctx      context.Context
		TaskName string
	}{ctx, taskName})

	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, taskName)
	}

	// Default implementation
	return &gitx.Worktree{
		Path:       fmt.Sprintf("/tmp/test-alpine-%s", taskName),
		Branch:     fmt.Sprintf("alpine/%s", taskName),
		ParentRepo: "/tmp/test-repo",
	}, nil
}

// Cleanup implements gitx.WorktreeManager.
func (m *WorktreeManager) Cleanup(ctx context.Context, wt *gitx.Worktree) error {
	m.CleanupCalls = append(m.CleanupCalls, struct {
		Ctx context.Context
		WT  *gitx.Worktree
	}{ctx, wt})

	if m.CleanupFunc != nil {
		return m.CleanupFunc(ctx, wt)
	}

	// Default implementation
	return nil
}

// Verify that mock implements the interface
var _ gitx.WorktreeManager = (*WorktreeManager)(nil)
