package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/core"
	gitxmock "github.com/maxmcd/river/internal/gitx/mock"
	"github.com/maxmcd/river/internal/workflow"
	"github.com/maxmcd/river/test/integration/helpers"
)

// TestBareMode_StartsWithRalph tests that bare mode initializes with /ralph when no state exists
// This test validates that the bare execution mode correctly starts a new workflow
// with the /ralph command when both --no-plan and --no-worktree flags are set
func TestBareMode_StartsWithRalph(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create mock Claude executor that expects /ralph command
	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/ralph",
				responseState: &core.State{
					CurrentStepDescription: "Initialized bare mode workflow",
					NextStepPrompt:         "/implement first-task",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
			},
			{
				expectedPrompt: "/implement first-task",
				responseState: &core.State{
					CurrentStepDescription: "Completed first task",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				delay: 50 * time.Millisecond,
			},
		},
	}

	// Create workflow engine with bare mode configuration
	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false, // --no-worktree
		},
	}
	engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Run with empty task description and no plan generation (bare mode)
	err := engine.Run(ctx, "", false) // empty task + no plan = bare mode
	require.NoError(t, err)

	// Verify the workflow completed successfully
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
	assert.Equal(t, "Completed first task", finalState.CurrentStepDescription)
	assert.Empty(t, finalState.NextStepPrompt)

	// Verify /ralph was called first
	assert.Equal(t, 2, mockExecutor.executionCount)
}

// TestBareMode_ContinuesExistingState tests that bare mode continues from existing state
// This test validates that when claude_state.json already exists, bare mode
// continues the existing workflow instead of starting a new one
func TestBareMode_ContinuesExistingState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create an existing state file
	existingState := &core.State{
		CurrentStepDescription: "Previous task completed",
		NextStepPrompt:         "/implement remaining-work",
		Status:                 "running",
	}
	err := existingState.Save(stateFile)
	require.NoError(t, err)

	// Create mock executor that expects continuation from existing state
	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/implement remaining-work",
				responseState: &core.State{
					CurrentStepDescription: "Completed remaining work",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				delay: 50 * time.Millisecond,
			},
		},
	}

	// Create workflow engine
	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}
	engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Run in bare mode
	err = engine.Run(ctx, "", false)
	require.NoError(t, err)

	// Verify it continued from existing state (not /ralph)
	assert.Equal(t, 1, mockExecutor.executionCount)
	
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
	assert.Equal(t, "Completed remaining work", finalState.CurrentStepDescription)
}

// TestBareMode_HandlesInterrupt tests that bare mode saves state correctly on interrupt
// This test validates that the state is properly saved when the workflow is interrupted,
// allowing it to be resumed later
func TestBareMode_HandlesInterrupt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithCancel(context.Background())
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create mock executor that cancels context after first execution
	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/ralph",
				responseState: &core.State{
					CurrentStepDescription: "Started task",
					NextStepPrompt:         "/continue work",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
				onStateUpdate: func(state *core.State) {
					// Cancel context after state is saved
					cancel()
				},
			},
			{
				// This should not be executed due to cancellation
				expectedPrompt: "/continue work",
				responseState: &core.State{
					CurrentStepDescription: "Should not reach here",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
			},
		},
	}

	// Create workflow engine
	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}
	engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Run in bare mode - should be interrupted
	err := engine.Run(ctx, "", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Verify state was saved before interruption
	savedState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "running", savedState.Status)
	assert.Equal(t, "Started task", savedState.CurrentStepDescription)
	assert.Equal(t, "/continue work", savedState.NextStepPrompt)

	// Verify only one execution happened
	assert.Equal(t, 1, mockExecutor.executionCount)
}

// TestBareMode_RequiresBothFlags tests that bare mode is only activated with both flags
// This test validates the error handling when only one flag is provided
func TestBareMode_RequiresBothFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockExecutor := &MockClaudeExecutor{
		stateFile:  stateFile,
		executions: []mockExecution{}, // Should not execute anything
	}

	// Test with only --no-worktree (worktree disabled but plan enabled)
	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}
	engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
	engine.SetStateFile(stateFile)

	// This should fail because empty task is only allowed in bare mode
	err := engine.Run(ctx, "", true) // empty task with plan generation = error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task description cannot be empty")

	// Verify no executions happened
	assert.Equal(t, 0, mockExecutor.executionCount)
}

// TestBareMode_CompleteWorkflow tests a complete bare mode workflow from start to finish
// This test validates the full integration of bare mode features including
// state persistence across multiple invocations
func TestBareMode_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// First invocation - starts with /ralph
	mockExecutor1 := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/ralph",
				responseState: &core.State{
					CurrentStepDescription: "Analyzed codebase and created plan",
					NextStepPrompt:         "/implement feature-a",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
			},
			{
				expectedPrompt: "/implement feature-a",
				responseState: &core.State{
					CurrentStepDescription: "Implemented feature A",
					NextStepPrompt:         "/test feature-a",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
			},
			{
				expectedPrompt: "/test feature-a",
				responseState: &core.State{
					CurrentStepDescription: "Tests pending",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				delay: 50 * time.Millisecond,
			},
		},
	}

	// Run first workflow
	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}
	engine1 := workflow.NewEngine(mockExecutor1, mockWtMgr, cfg)
	engine1.SetStateFile(stateFile)

	err := engine1.Run(ctx, "", false)
	require.NoError(t, err)

	// Verify first run completed successfully
	state1, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", state1.Status)
	assert.Equal(t, "Tests pending", state1.CurrentStepDescription)

	// Manually update state to simulate continuation scenario
	continuationState := &core.State{
		CurrentStepDescription: "Ready to continue",
		NextStepPrompt:         "/implement feature-b",
		Status:                 "running",
	}
	err = continuationState.Save(stateFile)
	require.NoError(t, err)

	// Second invocation - continues from updated state
	mockExecutor2 := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/implement feature-b",
				responseState: &core.State{
					CurrentStepDescription: "Implemented feature B",
					NextStepPrompt:         "/finalize",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
			},
			{
				expectedPrompt: "/finalize",
				responseState: &core.State{
					CurrentStepDescription: "Completed all features",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				delay: 50 * time.Millisecond,
			},
		},
	}

	engine2 := workflow.NewEngine(mockExecutor2, mockWtMgr, cfg)
	engine2.SetStateFile(stateFile)

	err = engine2.Run(ctx, "", false)
	require.NoError(t, err)

	// Verify final state
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
	assert.Equal(t, "Completed all features", finalState.CurrentStepDescription)
	assert.Empty(t, finalState.NextStepPrompt)
}

// TestBareMode_ErrorHandling tests error scenarios in bare mode
// This test validates that errors are properly handled and don't corrupt state
func TestBareMode_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create a mock executor that simulates an error after initial execution
	errorExecutions := []mockExecution{
		{
			expectedPrompt: "/ralph",
			responseState: &core.State{
				CurrentStepDescription: "Started processing",
				NextStepPrompt:         "/continue",
				Status:                 "running",
			},
			delay: 50 * time.Millisecond,
		},
		// Second execution will fail by exceeding the executions array
		// The MockClaudeExecutor.Execute method returns assert.AnError when
		// executionCount >= len(executions)
	}

	mockExecutor := &MockClaudeExecutor{
		stateFile:  stateFile,
		executions: errorExecutions,
	}

	mockWtMgr := &gitxmock.WorktreeManager{}
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}
	engine := workflow.NewEngine(mockExecutor, mockWtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Run should fail on second execution
	err := engine.Run(ctx, "", false)
	assert.Error(t, err)

	// State file should exist with the last successful state
	helpers.AssertFileExists(t, stateFile)
	
	// Verify the state from the first successful execution was preserved
	savedState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "running", savedState.Status)
	assert.Equal(t, "Started processing", savedState.CurrentStepDescription)
	assert.Equal(t, "/continue", savedState.NextStepPrompt)
}

// TestBareMode_StateFilePersistence tests that state files persist between invocations
// This test validates the state file is properly read and written in bare mode
func TestBareMode_StateFilePersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create initial state
	initialState := &core.State{
		CurrentStepDescription: "Task in progress",
		NextStepPrompt:         "/continue",
		Status:                 "running",
	}
	err := initialState.Save(stateFile)
	require.NoError(t, err)

	// Verify file exists
	helpers.AssertFileExists(t, stateFile)

	// Load and verify state
	loadedState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.True(t, helpers.CompareStates(initialState, loadedState))

	// Update state
	loadedState.CurrentStepDescription = "Task completed"
	loadedState.Status = "completed"
	loadedState.NextStepPrompt = ""
	err = loadedState.Save(stateFile)
	require.NoError(t, err)

	// Verify updated state
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "Task completed", finalState.CurrentStepDescription)
	assert.Equal(t, "completed", finalState.Status)
	assert.Empty(t, finalState.NextStepPrompt)
}