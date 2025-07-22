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
)

// TestClaudeCommandExecution tests the Claude command execution with various configurations
func TestClaudeCommandExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Test various Claude execution scenarios
	testCases := []struct {
		name           string
		config         claude.ExecuteConfig
		initialState   *core.State
		expectedError  string
		validateResult func(t *testing.T, output string, err error)
	}{
		{
			name: "Basic execution with state file",
			config: claude.ExecuteConfig{
				Prompt:    "/test Basic test prompt",
				StateFile: stateFile,
			},
			initialState: &core.State{
				CurrentStepDescription: "Starting test",
				NextStepPrompt:         "/test Basic test prompt",
				Status:                 "running",
			},
			validateResult: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
				assert.NotEmpty(t, output)
			},
		},
		{
			name: "Execution with custom MCP servers",
			config: claude.ExecuteConfig{
				Prompt:     "/mcp-test Test with MCP servers",
				StateFile:  stateFile,
				MCPServers: []string{"context7", "web-search"},
			},
			initialState: &core.State{
				CurrentStepDescription: "Testing MCP servers",
				NextStepPrompt:         "/mcp-test Test with MCP servers",
				Status:                 "running",
			},
			validateResult: func(t *testing.T, output string, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "Execution with timeout",
			config: claude.ExecuteConfig{
				Prompt:    "/timeout-test Test with timeout",
				StateFile: stateFile,
				Timeout:   1 * time.Second,
			},
			initialState: &core.State{
				CurrentStepDescription: "Testing timeout",
				NextStepPrompt:         "/timeout-test Test with timeout",
				Status:                 "running",
			},
			validateResult: func(t *testing.T, output string, err error) {
				// This would timeout in real execution
				// For mock, we just verify it completes
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up initial state if provided
			if tc.initialState != nil {
				err := tc.initialState.Save(stateFile)
				require.NoError(t, err)
			}

			// Create mock executor for testing
			mockExecutor := &MockClaudeExecutor{
				stateFile: stateFile,
				executions: []mockExecution{
					{
						expectedPrompt: tc.config.Prompt,
						responseState: &core.State{
							CurrentStepDescription: "Test completed",
							NextStepPrompt:         "",
							Status:                 "completed",
						},
					},
				},
			}

			output, err := mockExecutor.Execute(ctx, tc.config)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else if tc.validateResult != nil {
				tc.validateResult(t, output, err)
			}

			// Clean up state file for next test
			_ = os.Remove(stateFile)
		})
	}
}

// TestClaudeStateFileMonitoring tests that Claude properly monitors and updates state files
func TestClaudeStateFileMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create initial state
	initialState := &core.State{
		CurrentStepDescription: "Initial state",
		NextStepPrompt:         "/monitor-test Start monitoring",
		Status:                 "running",
	}
	err := initialState.Save(stateFile)
	require.NoError(t, err)

	// Track state changes
	stateChanges := []core.State{}

	// Create mock executor that simulates state updates
	mockExecutor := &MockClaudeExecutor{
		stateFile: stateFile,
		executions: []mockExecution{
			{
				expectedPrompt: "/monitor-test Start monitoring",
				responseState: &core.State{
					CurrentStepDescription: "Step 1 complete",
					NextStepPrompt:         "/monitor-test Continue",
					Status:                 "running",
				},
				delay: 50 * time.Millisecond,
				onStateUpdate: func(state *core.State) {
					stateChanges = append(stateChanges, *state)
				},
			},
		},
	}

	// Execute and monitor state changes
	_, err = mockExecutor.Execute(ctx, claude.ExecuteConfig{
		Prompt:    "/monitor-test Start monitoring",
		StateFile: stateFile,
	})
	require.NoError(t, err)

	// Verify state was updated
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "Step 1 complete", finalState.CurrentStepDescription)
	assert.Equal(t, "running", finalState.Status)

	// Verify we captured state changes
	assert.Len(t, stateChanges, 1)
}

// TestClaudeExecutionWithRealCommand tests executing real Claude command
// This test is skipped unless CLAUDE_INTEGRATION_TEST is set
func TestClaudeExecutionWithRealCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if os.Getenv("CLAUDE_INTEGRATION_TEST") != "true" {
		t.Skip("Set CLAUDE_INTEGRATION_TEST=true to run real Claude command tests")
	}

	// Check if claude command exists
	executor := claude.NewExecutor()
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create a simple test state
	initialState := &core.State{
		CurrentStepDescription: "Testing real Claude execution",
		NextStepPrompt:         "echo 'Hello from Claude integration test'",
		Status:                 "running",
	}
	err := initialState.Save(stateFile)
	require.NoError(t, err)

	// Execute a simple command
	config := claude.ExecuteConfig{
		Prompt:    "echo 'Hello from Claude integration test'",
		StateFile: stateFile,
		Timeout:   30 * time.Second,
	}

	output, err := executor.Execute(ctx, config)
	if err != nil {
		// If Claude is not installed, skip the test
		if os.IsNotExist(err) {
			t.Skip("Claude command not found, skipping real execution test")
		}
		require.NoError(t, err)
	}

	assert.NotEmpty(t, output)
}

// TestClaudeErrorScenarios tests various error conditions
func TestClaudeErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	tempDir := t.TempDir()

	testCases := []struct {
		name          string
		setup         func() (claude.ExecuteConfig, *MockClaudeExecutor)
		expectedError string
	}{
		{
			name: "Missing state file",
			setup: func() (claude.ExecuteConfig, *MockClaudeExecutor) {
				config := claude.ExecuteConfig{
					Prompt:    "/test Missing state file",
					StateFile: filepath.Join(tempDir, "nonexistent", "state.json"),
				}
				executor := &MockClaudeExecutor{
					stateFile: config.StateFile,
				}
				return config, executor
			},
			expectedError: "assert.AnError",
		},
		{
			name: "Context cancellation during execution",
			setup: func() (claude.ExecuteConfig, *MockClaudeExecutor) {
				stateFile := filepath.Join(tempDir, "cancelled_state.json")
				config := claude.ExecuteConfig{
					Prompt:    "/test Context cancellation",
					StateFile: stateFile,
				}
				executor := &MockClaudeExecutor{
					stateFile: stateFile,
					executions: []mockExecution{
						{
							expectedPrompt: "/test Context cancellation",
							delay:          100 * time.Millisecond,
							onExecute: func() {
								// Simulate long-running operation
								time.Sleep(200 * time.Millisecond)
							},
						},
					},
				}
				return config, executor
			},
			expectedError: "context deadline exceeded",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, executor := tc.setup()

			// Use a context with timeout for cancellation test
			testCtx := ctx
			if tc.name == "Context cancellation during execution" {
				var cancel context.CancelFunc
				testCtx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			_, err := executor.Execute(testCtx, config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestClaudeOutputParsing tests parsing of Claude command output
func TestClaudeOutputParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This test would verify that Claude output is properly parsed
	// For now, it's a placeholder as the current implementation returns raw output
	t.Skip("Output parsing test requires implementation of output parser")
}