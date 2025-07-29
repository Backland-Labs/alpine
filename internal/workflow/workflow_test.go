package workflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/gitx"
	gitxmock "github.com/Backland-Labs/alpine/internal/gitx/mock"
	"github.com/Backland-Labs/alpine/internal/output"
)

// MockCommandRunner mocks the claude.CommandRunner interface
type MockCommandRunner struct {
	mock.Mock
}

func (m *MockCommandRunner) Run(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	args := m.Called(ctx, config)
	return args.String(0), args.Error(1)
}

// testConfig creates a test configuration with worktree disabled by default
func testConfig(workTreeEnabled bool) *config.Config {
	return &config.Config{
		WorkDir:     "",
		Verbosity:   config.VerbosityNormal,
		ShowOutput:  false,
		StateFile:   "agent_state/agent_state.json",
		AutoCleanup: true,
		Git: config.GitConfig{
			WorktreeEnabled: workTreeEnabled,
			BaseBranch:      "main",
			AutoCleanupWT:   true,
		},
	}
}

func TestNewEngine(t *testing.T) {
	// Test that NewEngine creates a valid workflow engine
	executor := claude.NewExecutor()
	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)

	engine := NewEngine(executor, wtMgr, cfg)

	assert.Equal(t, executor, engine.claudeExecutor)
	assert.Equal(t, wtMgr, engine.wtMgr)
	assert.Equal(t, cfg, engine.cfg)
	assert.Equal(t, "agent_state/agent_state.json", engine.stateFile)
}

func TestEngine_Run_WithPlan(t *testing.T) {
	// Test the full workflow with plan generation
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create a test executor that simulates Claude behavior
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/make_plan Implement user authentication",
			stateUpdate: &core.State{
				CurrentStepDescription: "Generated plan for user authentication",
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

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Suppress output during tests
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run the workflow
	err = engine.Run(ctx, "Implement user authentication", true)
	require.NoError(t, err)

	// Verify all executions were performed
	assert.Equal(t, 2, executor.executionCount)

	// State file should be cleaned up after successful completion
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "State file should be removed after successful completion")
}

func TestEngine_Run_NoPlan(t *testing.T) {
	// Test direct execution without plan generation
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create a test executor that simulates Claude behavior
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop Fix bug in payment processing",
			stateUpdate: &core.State{
				CurrentStepDescription: "Fixed bug directly",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Suppress output during tests
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run the workflow without plan
	err = engine.Run(ctx, "Fix bug in payment processing", false)
	require.NoError(t, err)

	// Verify only one execution was performed
	assert.Equal(t, 1, executor.executionCount)
}

func TestEngine_Run_EmptyTaskDescription(t *testing.T) {
	// Test validation of empty task description
	ctx := context.Background()
	executor := claude.NewExecutor()
	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)

	// Test empty string
	err := engine.Run(ctx, "", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task description cannot be empty")

	// Test whitespace only
	err = engine.Run(ctx, "   ", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task description cannot be empty")
}

func TestEngine_Run_ContextCancellation(t *testing.T) {
	// Test that context cancellation stops the workflow
	ctx, cancel := context.WithCancel(context.Background())
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/make_plan Test task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Started",
				NextStepPrompt:         "/continue",
				Status:                 "running",
			},
			beforeExecution: func() {
				// Cancel context during first execution
				cancel()
			},
		},
		{
			// This should not be reached
			expectedPrompt: "/continue",
			stateUpdate: &core.State{
				CurrentStepDescription: "Should not reach here",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	err = engine.Run(ctx, "Test task", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")

	// Verify only first execution happened
	assert.Equal(t, 1, executor.executionCount)
}

func TestEngine_Run_StateFileUpdate(t *testing.T) {
	// Test waiting for state file updates
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create initial state
	initialState := &core.State{
		CurrentStepDescription: "Initial",
		NextStepPrompt:         "/start",
		Status:                 "running",
	}
	err = initialState.Save(stateFile)
	require.NoError(t, err)

	// Create a delayed executor that updates state after a delay
	executor := &delayedExecutor{
		t:         t,
		stateFile: stateFile,
		delay:     100 * time.Millisecond,
		newState: &core.State{
			CurrentStepDescription: "Updated",
			NextStepPrompt:         "",
			Status:                 "completed",
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run should wait for state update and complete
	err = engine.Run(ctx, "Test task", false)
	require.NoError(t, err)

	// State file should be cleaned up after successful completion
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "State file should be removed after successful completion")
}

// Helper types for testing

type testExecutor struct {
	t              *testing.T
	stateFile      string
	executions     []testExecution
	executionCount int
}

type testExecution struct {
	expectedPrompt  string
	stateUpdate     *core.State
	error           error
	beforeExecution func()
}

func newTestExecutor(t *testing.T, stateFile string) *testExecutor {
	return &testExecutor{
		t:         t,
		stateFile: stateFile,
	}
}

func (e *testExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	if e.executionCount >= len(e.executions) {
		e.t.Fatalf("Unexpected execution call #%d", e.executionCount+1)
	}

	execution := e.executions[e.executionCount]
	e.executionCount++

	// Run any pre-execution function
	if execution.beforeExecution != nil {
		execution.beforeExecution()
	}

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Verify the prompt matches expected
	assert.Equal(e.t, execution.expectedPrompt, config.Prompt)

	// Update state file if requested
	if execution.stateUpdate != nil {
		// Use the state file from config if provided, otherwise use the executor's state file
		stateFile := config.StateFile
		if stateFile == "" {
			stateFile = e.stateFile
		}
		err := execution.stateUpdate.Save(stateFile)
		require.NoError(e.t, err)
	}

	if execution.error != nil {
		return "", execution.error
	}

	return "Mock execution completed", nil
}

// delayedExecutor simulates an executor that updates state after a delay
type delayedExecutor struct {
	t         *testing.T
	stateFile string
	delay     time.Duration
	newState  *core.State
}

func (e *delayedExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	// Update state after delay in a goroutine
	go func() {
		time.Sleep(e.delay)
		err := e.newState.Save(e.stateFile)
		require.NoError(e.t, err)
	}()

	return "Started", nil
}

func TestEngineCreatesWorktree(t *testing.T) {
	// Test that engine creates worktree when enabled
	ctx := context.Background()
	tempDir := t.TempDir()
	worktreeDir := filepath.Join(tempDir, "test-worktree")

	// Save current directory to restore later
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	// Mock worktree manager
	mockWT := &gitx.Worktree{
		Path:       worktreeDir,
		Branch:     "alpine/test-task",
		ParentRepo: tempDir,
	}

	// Create a simple test executor that saves state relative to current directory
	executor := &testExecutor{
		t: t,
		executions: []testExecution{
			{
				expectedPrompt: "/run_implementation_loop test task",
				beforeExecution: func() {
					// At this point we should be in the worktree directory
					cwd, err := os.Getwd()
					require.NoError(t, err)
					// Resolve symlinks for comparison (macOS has /var -> /private/var)
					resolvedCwd, err := filepath.EvalSymlinks(cwd)
					require.NoError(t, err)
					resolvedWorktreeDir, err := filepath.EvalSymlinks(worktreeDir)
					require.NoError(t, err)
					assert.Equal(t, resolvedWorktreeDir, resolvedCwd)
				},
				stateUpdate: &core.State{
					CurrentStepDescription: "Task completed",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{
		CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
			assert.Equal(t, "test task", taskName)
			// Create the worktree directory
			require.NoError(t, os.MkdirAll(mockWT.Path, 0755))
			return mockWT, nil
		},
	}

	// Enable worktree in config
	cfg := testConfig(true)

	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow
	err = engine.Run(ctx, "test task", false)
	require.NoError(t, err)

	// Verify worktree was created
	assert.Len(t, wtMgr.CreateCalls, 1)
	assert.Equal(t, "test task", wtMgr.CreateCalls[0].TaskName)

	// Verify cleanup was called (auto cleanup is enabled)
	assert.Len(t, wtMgr.CleanupCalls, 1)
	assert.Equal(t, mockWT, wtMgr.CleanupCalls[0].WT)
}

func TestEngineWorktreeDisabled(t *testing.T) {
	// Test that engine doesn't create worktree when disabled
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create a test executor that marks workflow as completed immediately
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop test task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Task completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	// Mock worktree manager
	wtMgr := &gitxmock.WorktreeManager{}

	// Disable worktree in config
	cfg := testConfig(false)

	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow
	err = engine.Run(ctx, "test task", false)
	require.NoError(t, err)

	// Verify worktree was NOT created
	assert.Len(t, wtMgr.CreateCalls, 0)
	assert.Len(t, wtMgr.CleanupCalls, 0)
}

func TestEngineStateFileInWorktree(t *testing.T) {
	// Test that state file is created in worktree when worktree is enabled
	ctx := context.Background()
	tempDir := t.TempDir()
	worktreeDir := filepath.Join(tempDir, "test-worktree")

	// Mock worktree manager
	mockWT := &gitx.Worktree{
		Path:       worktreeDir,
		Branch:     "alpine/test-task",
		ParentRepo: tempDir,
	}

	wtMgr := &gitxmock.WorktreeManager{
		CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
			// Create the worktree directory
			require.NoError(t, os.MkdirAll(mockWT.Path, 0755))
			return mockWT, nil
		},
	}

	// Create a test executor that verifies state file location
	executor := &testExecutor{
		t: t,
		executions: []testExecution{
			{
				expectedPrompt: "/run_implementation_loop test task",
				stateUpdate: &core.State{
					CurrentStepDescription: "Task completed",
					NextStepPrompt:         "",
					Status:                 "completed",
				},
			},
		},
	}

	// Enable worktree in config
	cfg := testConfig(true)

	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow
	err := engine.Run(ctx, "test task", false)
	require.NoError(t, err)

	// State file should be cleaned up after successful completion
	expectedStateDir := filepath.Join(worktreeDir, "agent_state")
	expectedStateFile := filepath.Join(expectedStateDir, "agent_state.json")
	_, err = os.Stat(expectedStateFile)
	assert.True(t, os.IsNotExist(err), "State file should be removed after successful completion")
}

func TestEngine_BareMode_ContinuesExistingState(t *testing.T) {
	// Test that bare mode continues from existing claude_state.json
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create existing state file
	existingState := &core.State{
		CurrentStepDescription: "Previous work done",
		NextStepPrompt:         "/continue previous task",
		Status:                 "running",
	}
	err = existingState.Save(stateFile)
	require.NoError(t, err)

	// Create test executor that expects continuation
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/continue previous task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Continued previous work",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run in bare mode (empty task, no plan, no worktree)
	err = engine.Run(ctx, "", false)
	require.NoError(t, err)

	// Verify execution happened
	assert.Equal(t, 1, executor.executionCount)

	// State file should be cleaned up after successful completion
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "State file should be removed after successful completion")
}

func TestEngine_BareMode_InitializesWithrun_implementation_loop(t *testing.T) {
	// Test that bare mode initializes with /run_implementation_loop when no state exists
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create test executor that expects /run_implementation_loop initialization
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop",
			stateUpdate: &core.State{
				CurrentStepDescription: "Started bare execution",
				NextStepPrompt:         "/continue task",
				Status:                 "running",
			},
		},
		{
			expectedPrompt: "/continue task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Completed bare execution",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run in bare mode (empty task, no plan, no worktree)
	err = engine.Run(ctx, "", false)
	require.NoError(t, err)

	// Verify both executions happened
	assert.Equal(t, 2, executor.executionCount)

	// State file should be cleaned up after successful completion
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "State file should be removed after successful completion")
}

func TestEngine_Run_StateFileCleanup(t *testing.T) {
	// Test that state file is cleaned up on successful completion
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create a test executor that simulates Claude behavior
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop Test cleanup",
			stateUpdate: &core.State{
				CurrentStepDescription: "Task completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Suppress output during tests
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Verify state file exists after initialization
	err = engine.Run(ctx, "Test cleanup", false)
	require.NoError(t, err)

	// Verify state file was cleaned up
	_, err = os.Stat(stateFile)
	assert.True(t, os.IsNotExist(err), "State file should be removed after successful completion")

	// Verify state directory still exists
	_, err = os.Stat(stateDir)
	assert.NoError(t, err, "State directory should remain after completion")
}

func TestEngine_Run_StateFileNotCleanedOnError(t *testing.T) {
	// Test that state file is NOT cleaned up when workflow doesn't complete
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create a test executor that simulates Claude failure
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop Test error",
			error:          fmt.Errorf("simulated Claude error"),
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)

	// Suppress output during tests
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run should fail
	err = engine.Run(ctx, "Test error", false)
	assert.Error(t, err)

	// Verify state file still exists
	_, err = os.Stat(stateFile)
	assert.NoError(t, err, "State file should remain when workflow doesn't complete")
}

// TestEngine_EventEmitter_RunStarted verifies that the workflow engine calls EventEmitter.RunStarted
// when a workflow begins execution. This ensures lifecycle events are properly emitted.
func TestEngine_EventEmitter_RunStarted(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create test executor
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop test task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Task completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	// Create mock event emitter
	mockEmitter := events.NewMockEmitter()

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetEventEmitter(mockEmitter)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow
	err = engine.Run(ctx, "test task", false)
	require.NoError(t, err)

	// Verify RunStarted was called
	calls := mockEmitter.FindCallsByMethod("RunStarted")
	assert.Len(t, calls, 1, "RunStarted should be called once")
	assert.NotEmpty(t, calls[0].RunID, "RunID should not be empty")
	assert.Equal(t, "test task", calls[0].Task, "Task should match input")
}

// TestEngine_EventEmitter_RunFinished verifies that the workflow engine calls EventEmitter.RunFinished
// when a workflow completes successfully.
func TestEngine_EventEmitter_RunFinished(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create test executor
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop test task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Task completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	// Create mock event emitter
	mockEmitter := events.NewMockEmitter()

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetEventEmitter(mockEmitter)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow
	err = engine.Run(ctx, "test task", false)
	require.NoError(t, err)

	// Verify RunFinished was called
	calls := mockEmitter.FindCallsByMethod("RunFinished")
	assert.Len(t, calls, 1, "RunFinished should be called once")
	
	// Verify the same RunID was used for both start and finish
	startCalls := mockEmitter.FindCallsByMethod("RunStarted")
	assert.Equal(t, startCalls[0].RunID, calls[0].RunID, "RunID should be consistent")
	assert.Equal(t, "test task", calls[0].Task, "Task should match input")
}

// TestEngine_EventEmitter_RunError verifies that the workflow engine calls EventEmitter.RunError
// when a workflow encounters an error.
func TestEngine_EventEmitter_RunError(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	testError := errors.New("test execution error")

	// Create test executor that will fail
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop test task",
			error:          testError,
		},
	}

	// Create mock event emitter
	mockEmitter := events.NewMockEmitter()

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	engine.SetEventEmitter(mockEmitter)
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow (expect error)
	err = engine.Run(ctx, "test task", false)
	assert.Error(t, err)

	// Verify RunStarted was called
	startCalls := mockEmitter.FindCallsByMethod("RunStarted")
	assert.Len(t, startCalls, 1, "RunStarted should be called once")

	// Verify RunError was called
	errorCalls := mockEmitter.FindCallsByMethod("RunError")
	assert.Len(t, errorCalls, 1, "RunError should be called once")
	assert.Equal(t, startCalls[0].RunID, errorCalls[0].RunID, "RunID should be consistent")
	assert.Equal(t, "test task", errorCalls[0].Task, "Task should match input")
	assert.NotNil(t, errorCalls[0].Error, "Error should be captured")

	// Verify RunFinished was NOT called
	finishCalls := mockEmitter.FindCallsByMethod("RunFinished")
	assert.Len(t, finishCalls, 0, "RunFinished should not be called on error")
}

// TestEngine_EventEmitter_NilEmitter verifies that the workflow engine works correctly
// when no EventEmitter is provided (nil emitter).
func TestEngine_EventEmitter_NilEmitter(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "agent_state")
	err := os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)
	stateFile := filepath.Join(stateDir, "agent_state.json")

	// Create test executor
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/run_implementation_loop test task",
			stateUpdate: &core.State{
				CurrentStepDescription: "Task completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			},
		},
	}

	wtMgr := &gitxmock.WorktreeManager{}
	cfg := testConfig(false)
	engine := NewEngine(executor, wtMgr, cfg)
	engine.SetStateFile(stateFile)
	// Don't set event emitter - it should be nil
	engine.SetPrinter(output.NewPrinterWithWriters(io.Discard, io.Discard, false))

	// Run workflow - should not panic or error
	err = engine.Run(ctx, "test task", false)
	require.NoError(t, err)
}
