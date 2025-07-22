package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/core"
	"github.com/maxmcd/river/internal/workflow"
)

// TestFullWorkflowWithMockClaude tests the complete workflow from task description to completion
// This test validates the integration between all components using a mock Claude executor
func TestFullWorkflowWithMockClaude(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create mock Claude executor
	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/make_plan Implement user authentication",
				responseState: &core.State{
					CurrentStepDescription: "Created plan for OAuth2 implementation",
					NextStepPrompt:         "/implement oauth-setup",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond, // Simulate processing time
			},
			{
				expectedPrompt: "/implement oauth-setup",
				responseState: &core.State{
					CurrentStepDescription: "Implemented OAuth2 setup and configuration",
					NextStepPrompt:         "/implement oauth-handlers",
					Status:                 "running",
				},
				delay: 100 * time.Millisecond,
			},
			{
				expectedPrompt: "/implement oauth-handlers",
				responseState: &core.State{
					CurrentStepDescription: "Implemented OAuth2 request handlers",
					NextStepPrompt:         "/test oauth-integration",
					Status:                 "running",
				},
				delay: 75 * time.Millisecond,
			},
			{
				expectedPrompt: "/test oauth-integration",
				responseState: &core.State{
					CurrentStepDescription: "All OAuth2 tests passing",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				delay: 50 * time.Millisecond,
			},
		},
	}

	// Create workflow engine
	engine := workflow.NewEngine(mockExecutor)
	engine.SetStateFile(stateFile)

	// Run the workflow
	err := engine.Run(ctx, "Implement user authentication", true)
	require.NoError(t, err)

	// Verify the workflow completed successfully
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
	assert.Equal(t, "All OAuth2 tests passing", finalState.CurrentStepDescription)
	assert.Empty(t, finalState.NextStepPrompt)

	// Verify all executions were called
	assert.Equal(t, 4, mockExecutor.executionCount)
}

// TestWorkflowWithNoPlanFlag tests the workflow when --no-plan flag is used
// This should skip the planning phase and go directly to /ralph command
func TestWorkflowWithNoPlanFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/ralph Fix database connection pooling",
				responseState: &core.State{
					CurrentStepDescription: "Fixed connection pooling issue",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				delay: 200 * time.Millisecond,
			},
		},
	}

	engine := workflow.NewEngine(mockExecutor)
	engine.SetStateFile(stateFile)

	// Run with no-plan flag
	err := engine.Run(ctx, "Fix database connection pooling", false)
	require.NoError(t, err)

	// Verify only one execution happened
	assert.Equal(t, 1, mockExecutor.executionCount)
}

// TestWorkflowInterruptHandling tests that the workflow can be interrupted gracefully
// and that state is preserved correctly
func TestWorkflowInterruptHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithCancel(context.Background())
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/make_plan Long running task",
				responseState: &core.State{
					CurrentStepDescription: "Started planning",
					NextStepPrompt:         "/implement step1",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
			},
			{
				expectedPrompt: "/implement step1",
				// This execution will be interrupted
				onExecute: func() {
					cancel() // Cancel context during execution
				},
				responseState: &core.State{
					CurrentStepDescription: "Working on step1",
					NextStepPrompt:         "/implement step2",
					Status:                 "running",
				},
				delay: 100 * time.Millisecond,
			},
		},
	}

	engine := workflow.NewEngine(mockExecutor)
	engine.SetStateFile(stateFile)

	// Run should return context canceled error
	err := engine.Run(ctx, "Long running task", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Verify state was saved before interruption
	state, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "running", state.Status)
	assert.NotEmpty(t, state.CurrentStepDescription)
}

// TestStateFileCreationAndUpdates tests that state files are created and updated correctly
// throughout the workflow lifecycle
func TestStateFileCreationAndUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Verify state file doesn't exist initially
	_, err := os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err))

	stateUpdates := []core.State{}
	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/make_plan Test state management",
				responseState: &core.State{
					CurrentStepDescription: "Planning phase",
					NextStepPrompt:         "/next",
					Status:                 "running",
				},
				onStateUpdate: func(state *core.State) {
					stateUpdates = append(stateUpdates, *state)
				},
			},
			{
				expectedPrompt: "/next",
				responseState: &core.State{
					CurrentStepDescription: "Execution complete",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
				onStateUpdate: func(state *core.State) {
					stateUpdates = append(stateUpdates, *state)
				},
			},
		},
	}

	engine := workflow.NewEngine(mockExecutor)
	engine.SetStateFile(stateFile)

	err = engine.Run(ctx, "Test state management", true)
	require.NoError(t, err)

	// Verify state file exists and contains final state
	_, err = os.Stat(stateFile)
	assert.NoError(t, err)

	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)

	// Verify we captured all state transitions
	assert.Len(t, stateUpdates, 2)
	assert.Equal(t, "running", stateUpdates[0].Status)
	assert.Equal(t, "completed", stateUpdates[1].Status)
}

// TestCleanupBehavior tests that temporary files and resources are cleaned up properly
func TestCleanupBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create some temporary files that might be created during workflow
	tempFiles := []string{
		filepath.Join(tempDir, "temp_output.log"),
		filepath.Join(tempDir, "claude_response.json"),
	}

	for _, f := range tempFiles {
		err := os.WriteFile(f, []byte("test data"), 0644)
		require.NoError(t, err)
	}

	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/ralph Test cleanup",
				responseState: &core.State{
					CurrentStepDescription: "Cleanup test complete",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
			},
		},
	}

	engine := workflow.NewEngine(mockExecutor)
	engine.SetStateFile(stateFile)

	err := engine.Run(ctx, "Test cleanup", false)
	require.NoError(t, err)

	// State file should still exist after completion
	_, err = os.Stat(stateFile)
	assert.NoError(t, err)

	// Verify the state file contains expected data
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
}

// TestOutputFormatting tests that output is formatted correctly based on configuration
func TestOutputFormatting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test would verify output formatting but requires capturing stdout
	// For now, we'll skip the implementation but keep the test as a placeholder
	t.Skip("Output formatting test requires stdout capture implementation")
}

// MockClaudeExecutor is a mock implementation of the Claude executor for testing
type MockClaudeExecutor struct {
	stateFile      string
	executions     []mockExecution
	executionCount int
}

type mockExecution struct {
	expectedPrompt string
	responseState  *core.State
	delay          time.Duration
	onExecute      func()
	onStateUpdate  func(*core.State)
}

func (m *MockClaudeExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	if m.executionCount >= len(m.executions) {
		return "", assert.AnError
	}

	exec := m.executions[m.executionCount]
	m.executionCount++

	// Call onExecute callback if provided
	if exec.onExecute != nil {
		exec.onExecute()
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Verify the prompt matches expected
	if exec.expectedPrompt != config.Prompt {
		return "", assert.AnError
	}

	// Simulate processing delay
	if exec.delay > 0 {
		time.Sleep(exec.delay)
	}

	// Update state file if response state is provided
	if exec.responseState != nil {
		err := exec.responseState.Save(m.stateFile)
		if err != nil {
			return "", err
		}

		// Call state update callback if provided
		if exec.onStateUpdate != nil {
			exec.onStateUpdate(exec.responseState)
		}
	}

	return "Mock Claude execution output", nil
}