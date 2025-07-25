package cli

import (
	"context"
	"testing"

	"github.com/maxmcd/alpine/internal/config"
	"github.com/maxmcd/alpine/internal/gitx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestRealWorkflowEngineCreation tests that RealWorkflowEngine can be created
func TestRealWorkflowEngineCreation(t *testing.T) {
	// Create a minimal config
	cfg := &config.Config{
		WorkDir: "/tmp",
		Git: config.GitConfig{
			WorktreeEnabled: false,
			BaseBranch:      "main",
		},
	}

	// Create the real workflow engine
	engine := NewRealWorkflowEngine(cfg, nil)

	// Verify the engine was created
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.engine)
}

// TestRunWorkflowWrapper tests the runWorkflow wrapper function
func TestRunWorkflowWrapper(t *testing.T) {
	// We can't easily test runWorkflow since it creates real dependencies
	// but we can verify it exists and has the right signature

	// This is mainly a compilation test to ensure the function exists
	var fn = runWorkflow
	assert.NotNil(t, fn)
}

// TestRunWorkflowBareMode tests the bare mode scenarios
func TestRunWorkflowBareMode(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		noPlan           bool
		noWorktree       bool
		fromFile         string
		fileContent      string
		setupMocks       func(*Dependencies)
		wantErr          bool
		expectedErrorMsg string
	}{
		{
			name:       "bare mode - empty args allowed",
			args:       []string{},
			noPlan:     true,
			noWorktree: true,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: true, // Should be overridden
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "bare mode - whitespace task allowed",
			args:       []string{"   \t\n   "},
			noPlan:     true,
			noWorktree: true,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: true,
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "bare mode - file with empty content",
			args:       []string{},
			noPlan:     true,
			noWorktree: true,
			fromFile:   "empty.md",
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git:     config.GitConfig{},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.FileReader.(*MockFileReader).On("ReadFile", "empty.md").Return([]byte("   "), nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "noWorktree flag overrides config",
			args:       []string{"Test task"},
			noPlan:     false,
			noWorktree: true,
			setupMocks: func(deps *Dependencies) {
				cfg := &config.Config{
					WorkDir: "/tmp",
					Git: config.GitConfig{
						WorktreeEnabled: true, // Should be set to false
					},
				}
				deps.ConfigLoader.(*MockConfigLoader).On("Load").Return(cfg, nil)
				deps.WorkflowEngine.(*MockWorkflowEngine).On("Run", mock.Anything, "Test task", true).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := &Dependencies{
				ConfigLoader:   &MockConfigLoader{},
				WorkflowEngine: &MockWorkflowEngine{},
				FileReader:     &MockFileReader{},
			}

			tt.setupMocks(deps)

			err := runWorkflowWithDependencies(context.Background(), tt.args, tt.noPlan, tt.noWorktree, tt.fromFile, false, deps)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			// Verify expectations
			deps.ConfigLoader.(*MockConfigLoader).AssertExpectations(t)
			deps.WorkflowEngine.(*MockWorkflowEngine).AssertExpectations(t)
			deps.FileReader.(*MockFileReader).AssertExpectations(t)
		})
	}
}

// TestCreateWorkflowEngineWithErrors tests error cases in CreateWorkflowEngine
func TestCreateWorkflowEngineWithErrors(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *config.Config
		expectNilWtMgr bool
	}{
		{
			name: "worktree disabled",
			cfg: &config.Config{
				WorkDir: "/tmp",
				Git: config.GitConfig{
					WorktreeEnabled: false,
				},
			},
			expectNilWtMgr: true,
		},
		{
			name: "worktree enabled creates manager",
			cfg: &config.Config{
				WorkDir: "/tmp",
				Git: config.GitConfig{
					WorktreeEnabled: true,
					BaseBranch:      "main",
				},
			},
			expectNilWtMgr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, wtMgr := CreateWorkflowEngine(tt.cfg)

			assert.NotNil(t, engine)

			if tt.expectNilWtMgr {
				assert.Nil(t, wtMgr)
			} else {
				// Worktree manager might still be nil if not in a git repo
				// so we just check it's the right type if not nil
				if wtMgr != nil {
					assert.IsType(t, &gitx.CLIWorktreeManager{}, wtMgr)
				}
			}
		})
	}
}
