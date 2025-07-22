package workflow

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/maxmcd/river/internal/claude"
	"github.com/maxmcd/river/internal/core"
	"github.com/maxmcd/river/internal/output"
)

// MockCommandRunner mocks the claude.CommandRunner interface
type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	args := m.Called(ctx, config)
	return args.String(0), args.Error(1)
}

// MockLinearClient mocks the Linear API client
type MockLinearClient struct {
	mock.Mock
}

func (m *MockLinearClient) FetchIssue(ctx context.Context, issueID string) (*LinearIssue, error) {
	args := m.Called(ctx, issueID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*LinearIssue), args.Error(1)
}

// createMockExecutor creates a claude.Executor with a mock command runner
func createMockExecutor(mockRunner claude.CommandRunner) *claude.Executor {
	executor := claude.NewExecutor()
	// Use reflection to set the private commandRunner field
	// In a real scenario, we'd have a setter or constructor parameter
	return executor
}

func TestNewEngine(t *testing.T) {
	// Test that NewEngine creates a valid workflow engine
	executor := claude.NewExecutor()
	mockLinear := &MockLinearClient{}

	engine := NewEngine(executor, mockLinear)

	assert.NotNil(t, engine)
	assert.Equal(t, executor, engine.claudeExecutor)
	assert.Equal(t, mockLinear, engine.linearClient)
	assert.Equal(t, "claude_state.json", engine.stateFile)
}

func TestEngine_Run_WithPlan(t *testing.T) {
	// Test the full workflow with plan generation
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Set up mocks
	mockLinear := &MockLinearClient{}

	// Mock Linear issue fetch
	issue := &LinearIssue{
		ID:          "TEST-123",
		Title:       "Test Issue",
		Description: "Test description",
	}
	mockLinear.On("FetchIssue", ctx, "TEST-123").Return(issue, nil)

	// Create a test executor that simulates Claude behavior
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/make_plan Test Issue\n\nTest description",
			stateUpdate: &core.State{
				CurrentStepDescription: "Generated plan for Test Issue",
				NextStepPrompt:         "/implement step1",
				Status:                 "running",
			},
		},
		{
			expectedPrompt: "/implement step1",
			stateUpdate: &core.State{
				CurrentStepDescription: "Implemented step1",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	// Create engine and run
	engine := NewEngine(executor, mockLinear)
	engine.SetStateFile(stateFile)

	err := engine.Run(ctx, "TEST-123", false)

	assert.NoError(t, err)
	mockLinear.AssertExpectations(t)
	assert.Equal(t, 2, executor.callCount)

	// Verify final state
	finalState, err := core.LoadState(stateFile)
	assert.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
}

func TestEngine_Run_NoPlan(t *testing.T) {
	// Test the workflow with --no-plan flag
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockLinear := &MockLinearClient{}

	// Mock Linear issue fetch
	issue := &LinearIssue{
		ID:          "TEST-456",
		Title:       "Another Test Issue",
		Description: "Another test description",
	}
	mockLinear.On("FetchIssue", ctx, "TEST-456").Return(issue, nil)

	// Create test executor
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/ralph Another Test Issue\n\nAnother test description",
			stateUpdate: &core.State{
				CurrentStepDescription: "Executed ralph command",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	engine := NewEngine(executor, mockLinear)
	engine.SetStateFile(stateFile)

	err := engine.Run(ctx, "TEST-456", true)

	assert.NoError(t, err)
	mockLinear.AssertExpectations(t)
}

func TestEngine_Run_InvalidIssueID(t *testing.T) {
	// Test that invalid issue IDs are rejected
	ctx := context.Background()

	executor := claude.NewExecutor()
	mockLinear := &MockLinearClient{}

	engine := NewEngine(executor, mockLinear)

	err := engine.Run(ctx, "", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "issue ID cannot be empty")
}

func TestEngine_Run_LinearFetchError(t *testing.T) {
	// Test error handling when Linear fetch fails
	ctx := context.Background()

	executor := claude.NewExecutor()
	mockLinear := &MockLinearClient{}

	mockLinear.On("FetchIssue", ctx, "TEST-789").Return(nil, assert.AnError)

	engine := NewEngine(executor, mockLinear)

	err := engine.Run(ctx, "TEST-789", false)

	assert.Error(t, err)
	mockLinear.AssertExpectations(t)
}

func TestEngine_Run_ClaudeExecutionError(t *testing.T) {
	// Test error handling when Claude execution fails
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockLinear := &MockLinearClient{}

	issue := &LinearIssue{
		ID:          "TEST-999",
		Title:       "Error Test Issue",
		Description: "This will fail",
	}
	mockLinear.On("FetchIssue", ctx, "TEST-999").Return(issue, nil)

	// Create executor that returns error
	executor := newTestExecutor(t, stateFile)
	executor.returnError = assert.AnError

	engine := NewEngine(executor, mockLinear)
	engine.SetStateFile(stateFile)

	err := engine.Run(ctx, "TEST-999", false)

	assert.Error(t, err)
	mockLinear.AssertExpectations(t)
}

func TestEngine_Run_StateFileMonitoring(t *testing.T) {
	// Test that the engine properly monitors state file changes
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockLinear := &MockLinearClient{}

	issue := &LinearIssue{
		ID:          "TEST-MON",
		Title:       "Monitor Test",
		Description: "Test state monitoring",
	}
	mockLinear.On("FetchIssue", ctx, "TEST-MON").Return(issue, nil)

	// Create executor with delayed state updates
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/make_plan Monitor Test\n\nTest state monitoring",
			stateUpdate: &core.State{
				CurrentStepDescription: "Plan created",
				NextStepPrompt:         "/next",
				Status:                 "running",
			},
			delay: 100 * time.Millisecond,
		},
		{
			expectedPrompt: "/next",
			stateUpdate: &core.State{
				CurrentStepDescription: "Completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	engine := NewEngine(executor, mockLinear)
	engine.SetStateFile(stateFile)

	err := engine.Run(ctx, "TEST-MON", false)
	assert.NoError(t, err)

	mockLinear.AssertExpectations(t)
}

func TestEngine_Run_ContextCancellation(t *testing.T) {
	// Test that the engine respects context cancellation
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	mockLinear := &MockLinearClient{}

	issue := &LinearIssue{
		ID:          "TEST-CTX",
		Title:       "Context Test",
		Description: "Test context cancellation",
	}
	mockLinear.On("FetchIssue", ctx, "TEST-CTX").Return(issue, nil)

	// Create executor that cancels context during execution
	executor := newTestExecutor(t, stateFile)
	executor.returnError = context.Canceled

	engine := NewEngine(executor, mockLinear)
	engine.SetStateFile(stateFile)

	err := engine.Run(ctx, "TEST-CTX", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestEngine_initializeWorkflow(t *testing.T) {
	// Test workflow initialization creates correct initial state
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create engine with no-op printer for test
	engine := &Engine{
		stateFile: stateFile,
		printer:   output.NewPrinterWithWriters(io.Discard, io.Discard, false),
	}

	issue := &LinearIssue{
		ID:          "TEST-INIT",
		Title:       "Init Test Issue",
		Description: "Test initialization",
	}

	// Test with plan
	err := engine.initializeWorkflow(issue, false)
	assert.NoError(t, err)

	state, err := core.LoadState(stateFile)
	assert.NoError(t, err)
	assert.Equal(t, "Initializing workflow for Linear issue TEST-INIT", state.CurrentStepDescription)
	assert.Equal(t, "/make_plan Init Test Issue\n\nTest initialization", state.NextStepPrompt)
	assert.Equal(t, "running", state.Status)

	// Clean up for next test
	os.Remove(stateFile)

	// Test without plan
	err = engine.initializeWorkflow(issue, true)
	assert.NoError(t, err)

	state, err = core.LoadState(stateFile)
	assert.NoError(t, err)
	assert.Equal(t, "/ralph Init Test Issue\n\nTest initialization", state.NextStepPrompt)
}

// testExecutor wraps claude.Executor for testing
type testExecutor struct {
	*claude.Executor
	t           *testing.T
	stateFile   string
	executions  []testExecution
	callCount   int
	returnError error
	onExecute   func()
}

type testExecution struct {
	expectedPrompt string
	stateUpdate    *core.State
	delay          time.Duration
}

// newTestExecutor creates a new test executor
func newTestExecutor(t *testing.T, stateFile string) *testExecutor {
	return &testExecutor{
		Executor:  claude.NewExecutor(),
		t:         t,
		stateFile: stateFile,
	}
}

func (e *testExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	if e.onExecute != nil {
		e.onExecute()
	}

	if e.returnError != nil {
		return "", e.returnError
	}

	if e.callCount >= len(e.executions) {
		e.t.Fatalf("Unexpected execution call %d", e.callCount)
	}

	exec := e.executions[e.callCount]
	e.callCount++

	// Verify the prompt matches
	assert.Equal(e.t, exec.expectedPrompt, config.Prompt)
	assert.Equal(e.t, e.stateFile, config.StateFile)

	// Simulate Claude updating the state file
	if exec.stateUpdate != nil {
		if exec.delay > 0 {
			time.Sleep(exec.delay)
		}
		err := exec.stateUpdate.Save(e.stateFile)
		require.NoError(e.t, err)
	}

	return "Mock execution output", nil
}
