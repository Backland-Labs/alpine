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
	"github.com/maxmcd/river/internal/config"
	"github.com/maxmcd/river/internal/core"
	"github.com/maxmcd/river/internal/gitx"
	gitxmock "github.com/maxmcd/river/internal/gitx/mock"
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

// testConfig creates a test configuration with worktree disabled by default
func testConfig(workTreeEnabled bool) *config.Config {
	return &config.Config{
		WorkDir:     "",
		Verbosity:   config.VerbosityNormal,
		ShowOutput:  false,
		StateFile:   "claude_state.json",
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

	assert.NotNil(t, engine)
	assert.Equal(t, executor, engine.claudeExecutor)
	assert.Equal(t, wtMgr, engine.wtMgr)
	assert.Equal(t, cfg, engine.cfg)
	assert.Equal(t, "claude_state.json", engine.stateFile)
}

func TestEngine_Run_WithPlan(t *testing.T) {
	// Test the full workflow with plan generation
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

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
	err := engine.Run(ctx, "Implement user authentication", true)
	require.NoError(t, err)

	// Verify all executions were performed
	assert.Equal(t, 2, executor.executionCount)

	// Verify final state
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
}

func TestEngine_Run_NoPlan(t *testing.T) {
	// Test direct execution without plan generation
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create a test executor that simulates Claude behavior
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/ralph Fix bug in payment processing",
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
	err := engine.Run(ctx, "Fix bug in payment processing", false)
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
	stateFile := filepath.Join(tempDir, "claude_state.json")

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

	err := engine.Run(ctx, "Test task", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	
	// Verify only first execution happened
	assert.Equal(t, 1, executor.executionCount)
}

func TestEngine_Run_StateFileUpdate(t *testing.T) {
	// Test waiting for state file updates
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")

	// Create initial state
	initialState := &core.State{
		CurrentStepDescription: "Initial",
		NextStepPrompt:         "/start",
		Status:                 "running",
	}
	err := initialState.Save(stateFile)
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

	// Verify state was updated
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
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
		Branch:     "river/test-task",
		ParentRepo: tempDir,
	}
	
	// Create a simple test executor that saves state relative to current directory
	executor := &testExecutor{
		t: t,
		executions: []testExecution{
			{
				expectedPrompt: "/ralph test task",
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
	stateFile := filepath.Join(tempDir, "claude_state.json")
	
	// Create a test executor that marks workflow as completed immediately
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/ralph test task",
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
	err := engine.Run(ctx, "test task", false)
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
		Branch:     "river/test-task",
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
				expectedPrompt: "/ralph test task",
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
	
	// Verify state file was created in worktree
	expectedStateFile := filepath.Join(worktreeDir, "claude_state.json")
	assert.FileExists(t, expectedStateFile)
}

func TestEngine_BareMode_ContinuesExistingState(t *testing.T) {
	// Test that bare mode continues from existing claude_state.json
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")
	
	// Create existing state file
	existingState := &core.State{
		CurrentStepDescription: "Previous work done",
		NextStepPrompt:         "/continue previous task",
		Status:                 "running",
	}
	err := existingState.Save(stateFile)
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
	
	// Verify final state
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
	assert.Equal(t, "Continued previous work", finalState.CurrentStepDescription)
}

func TestEngine_BareMode_InitializesWithRalph(t *testing.T) {
	// Test that bare mode initializes with /ralph when no state exists
	ctx := context.Background()
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "claude_state.json")
	
	// Create test executor that expects /ralph initialization
	executor := newTestExecutor(t, stateFile)
	executor.executions = []testExecution{
		{
			expectedPrompt: "/ralph",
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
	err := engine.Run(ctx, "", false)
	require.NoError(t, err)
	
	// Verify both executions happened
	assert.Equal(t, 2, executor.executionCount)
	
	// Verify final state
	finalState, err := core.LoadState(stateFile)
	require.NoError(t, err)
	assert.Equal(t, "completed", finalState.Status)
}