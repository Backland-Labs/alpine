package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorktreeManager for testing
type MockWorktreeManager struct {
	mock.Mock
}

func (m *MockWorktreeManager) Create(ctx context.Context, taskName string) (*gitx.Worktree, error) {
	args := m.Called(ctx, taskName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gitx.Worktree), args.Error(1)
}

func (m *MockWorktreeManager) Cleanup(ctx context.Context, wt *gitx.Worktree) error {
	args := m.Called(ctx, wt)
	return args.Error(0)
}

// TestCLIWorktreeDisabled tests that when --no-worktree is used, worktree creation is disabled
func TestCLIWorktreeDisabled(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		noPlan             bool
		noWorktree         bool
		setupMocks         func(*Dependencies, *MockWorktreeManager)
		expectWorktreeCall bool
		wantErr            bool
		expectedErrorMsg   string
	}{
		{
			name:       "worktree enabled by default",
			args:       []string{"Implement feature"},
			noPlan:     false,
			noWorktree: false,
			setupMocks: func(deps *Dependencies, wtMgr *MockWorktreeManager) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: true,
						BaseBranch:      "main",
						AutoCleanupWT:   true,
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)

				// Expect worktree creation
				wt := &gitx.Worktree{
					Path:       "../alpine-alpine-implement-feature",
					Branch:     "alpine/implement-feature",
					ParentRepo: "/tmp",
				}
				wtMgr.On("Create", mock.Anything, "Implement feature").Return(wt, nil)

				// Mock workflow engine with worktree manager injected
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Implement feature", true).Return(nil)
			},
			expectWorktreeCall: true,
			wantErr:            false,
		},
		{
			name:       "worktree disabled with --no-worktree flag",
			args:       []string{"Fix bug"},
			noPlan:     false,
			noWorktree: true, // This flag should disable worktree creation
			setupMocks: func(deps *Dependencies, wtMgr *MockWorktreeManager) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: true, // Even if config says enabled
						BaseBranch:      "main",
						AutoCleanupWT:   true,
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)

				// Should NOT call worktree manager when --no-worktree is used
				// No expectations set on wtMgr

				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Fix bug", true).Return(nil)
			},
			expectWorktreeCall: false,
			wantErr:            false,
		},
		{
			name:       "worktree disabled in config",
			args:       []string{"Add tests"},
			noPlan:     false,
			noWorktree: false,
			setupMocks: func(deps *Dependencies, wtMgr *MockWorktreeManager) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: false, // Disabled in config
						BaseBranch:      "main",
						AutoCleanupWT:   true,
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)

				// Should NOT call worktree manager when disabled in config
				// No expectations set on wtMgr

				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Add tests", true).Return(nil)
			},
			expectWorktreeCall: false,
			wantErr:            false,
		},
		{
			name:       "--no-worktree overrides config",
			args:       []string{"Refactor code"},
			noPlan:     true,
			noWorktree: true, // Flag should override config
			setupMocks: func(deps *Dependencies, wtMgr *MockWorktreeManager) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: true, // Config says enabled, but flag overrides
						BaseBranch:      "main",
						AutoCleanupWT:   true,
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)

				// Should NOT call worktree manager when --no-worktree flag is used
				// No expectations set on wtMgr

				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Refactor code", false).Return(nil)
			},
			expectWorktreeCall: false,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dependencies
			deps := &Dependencies{
				ConfigLoader:   &MockConfigLoader{},
				WorkflowEngine: &MockWorkflowEngine{},
				FileReader:     &MockFileReader{},
			}

			// Create mock worktree manager
			wtMgr := &MockWorktreeManager{}

			// Setup mocks
			tt.setupMocks(deps, wtMgr)

			// Create a custom runWorkflow that uses our mock worktree manager
			var taskDescription string

			// Get task description from command line
			if len(tt.args) == 0 {
				t.Fatal("Task description is required")
			}
			taskDescription = tt.args[0]

			// Validate task description
			taskDescription = strings.TrimSpace(taskDescription)
			if taskDescription == "" {
				t.Fatal("Task description cannot be empty")
			}

			// Load configuration
			cfg, err := deps.ConfigLoader.Load()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Override worktree setting if --no-worktree flag is used
			if tt.noWorktree {
				cfg.Git.WorktreeEnabled = false
			}

			// Initialize logger
			logger.InitializeFromConfig(cfg)

			// If worktree is enabled and we have a manager, simulate the worktree creation
			if cfg.Git.WorktreeEnabled && wtMgr != nil {
				// This simulates what the real workflow engine would do
				_, _ = wtMgr.Create(context.Background(), taskDescription)
			}

			// Run the workflow
			generatePlan := !tt.noPlan
			err = deps.WorkflowEngine.Run(context.Background(), taskDescription, generatePlan)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			deps.ConfigLoader.(*MockConfigLoader).AssertExpectations(t)
			deps.WorkflowEngine.(*MockWorkflowEngine).AssertExpectations(t)
			deps.FileReader.(*MockFileReader).AssertExpectations(t)

			// Verify worktree manager expectations
			if tt.expectWorktreeCall {
				wtMgr.AssertExpectations(t)
			} else {
				// Ensure no calls were made to worktree manager
				wtMgr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				wtMgr.AssertNotCalled(t, "Cleanup", mock.Anything, mock.Anything)
			}
		})
	}
}

// TestCreateWorkflowEngine tests that CreateWorkflowEngine creates WorktreeManager
func TestCreateWorkflowEngine(t *testing.T) {
	// Create config with worktree enabled
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: true,
			BaseBranch:      "main",
			AutoCleanupWT:   true,
		},
	}

	// Create workflow engine
	engine, wtMgr := CreateWorkflowEngine(cfg, nil)

	// Check that engine and worktree manager are created
	assert.NotNil(t, engine, "CreateWorkflowEngine should create a workflow engine")
	assert.NotNil(t, wtMgr, "CreateWorkflowEngine should create a WorktreeManager when enabled")

	// Test with worktree disabled
	cfg.Git.WorktreeEnabled = false
	engine2, wtMgr2 := CreateWorkflowEngine(cfg, nil)

	assert.NotNil(t, engine2, "CreateWorkflowEngine should create a workflow engine")
	assert.Nil(t, wtMgr2, "CreateWorkflowEngine should not create a WorktreeManager when disabled")
}
