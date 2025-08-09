package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/gitx"
	"github.com/Backland-Labs/alpine/internal/workflow"
)

// MockClaudeExecutor is a mock implementation of workflow.ClaudeExecutor
type MockClaudeExecutor struct {
	ExecuteFunc func(ctx context.Context, config claude.ExecuteConfig) (string, error)
}

func (m *MockClaudeExecutor) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, config)
	}
	return "", nil
}

// MockWorktreeManager is a mock implementation of gitx.WorktreeManager
type MockWorktreeManager struct {
	CreateFunc  func(ctx context.Context, taskName string) (*gitx.Worktree, error)
	CleanupFunc func(ctx context.Context, wt *gitx.Worktree) error
}

func (m *MockWorktreeManager) Create(ctx context.Context, taskName string) (*gitx.Worktree, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, taskName)
	}
	return &gitx.Worktree{
		Path:       "/tmp/worktree-" + taskName,
		Branch:     "alpine/" + taskName,
		ParentRepo: "/tmp/repo",
	}, nil
}

func (m *MockWorktreeManager) Cleanup(ctx context.Context, wt *gitx.Worktree) error {
	if m.CleanupFunc != nil {
		return m.CleanupFunc(ctx, wt)
	}
	return nil
}

// TestNewAlpineWorkflowEngine tests the creation of a new AlpineWorkflowEngine
func TestNewAlpineWorkflowEngine(t *testing.T) {
	// Create necessary mocks
	mockExecutor := &MockClaudeExecutor{}
	mockWtMgr := &MockWorktreeManager{}
	cfg := &config.Config{
		WorkDir: "/tmp/test",
		Git: config.GitConfig{
			WorktreeEnabled: true,
		},
	}

	// Create engine
	engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

	// Verify engine is created correctly
	if engine == nil {
		t.Fatal("expected engine to be created")
	}

	// Verify internal state
	if engine.claudeExecutor != mockExecutor {
		t.Error("executor not set correctly")
	}
	if engine.wtMgr != mockWtMgr {
		t.Error("worktree manager not set correctly")
	}
	if engine.cfg != cfg {
		t.Error("config not set correctly")
	}
	if engine.workflows == nil {
		t.Error("workflows map not initialized")
	}
}

// TestStartWorkflow tests the StartWorkflow method
func TestStartWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		issueURL     string
		runID        string
		isGitRepo    bool
		createError  error
		executeError error
		expectError  bool
		expectedDir  string
	}{
		{
			name:        "successful start with git repo",
			issueURL:    "https://github.com/owner/repo/issues/123",
			runID:       "run-123",
			isGitRepo:   true,
			expectedDir: "/tmp/worktree-run-123",
		},
		{
			name:        "successful start without git repo",
			issueURL:    "https://github.com/owner/repo/issues/456",
			runID:       "run-456",
			isGitRepo:   false,
			expectedDir: "/tmp/alpine-run-456",
		},
		{
			name:        "worktree creation failure",
			issueURL:    "https://github.com/owner/repo/issues/789",
			runID:       "run-789",
			isGitRepo:   true,
			createError: errors.New("failed to create worktree"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for non-git tests
			tempDir := t.TempDir()

			// Create mocks
			mockExecutor := &MockClaudeExecutor{
				ExecuteFunc: func(ctx context.Context, cfg claude.ExecuteConfig) (string, error) {
					if tt.executeError != nil {
						return "", tt.executeError
					}
					// Verify prompt contains issue URL
					if !strings.Contains(cfg.Prompt, tt.issueURL) {
						t.Errorf("expected prompt to contain issue URL %s", tt.issueURL)
					}
					return "", nil
				},
			}

			mockWtMgr := &MockWorktreeManager{
				CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
					if tt.createError != nil {
						return nil, tt.createError
					}
					// The worktree name is expected to be "run-" + runID
					expectedPrefix := worktreeNamePrefix + tt.runID
					if taskName != expectedPrefix {
						t.Errorf("expected worktree name to be %s, got %s", expectedPrefix, taskName)
					}
					return &gitx.Worktree{
						Path:       "/tmp/worktree-" + taskName,
						Branch:     "alpine/" + taskName,
						ParentRepo: "/tmp/repo",
					}, nil
				},
			}

			cfg := &config.Config{
				WorkDir: tempDir,
				Git: config.GitConfig{
					WorktreeEnabled: tt.isGitRepo,
				},
			}

			// Create engine
			engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

			// Start workflow
			ctx := context.Background()
			workdir, err := engine.StartWorkflow(ctx, tt.issueURL, tt.runID, true)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// For git repo tests, expect worktree path
				if tt.isGitRepo {
					if !strings.Contains(workdir, "/tmp/worktree-") {
						t.Errorf("expected worktree path, got %s", workdir)
					}
				} else {
					// For non-git tests, expect temp directory
					if !strings.Contains(workdir, tempDirPrefix) {
						t.Errorf("expected temp directory with prefix %s, got %s", tempDirPrefix, workdir)
					}
				}
				// Verify workflow is tracked
				engine.mu.RLock()
				_, exists := engine.workflows[tt.runID]
				engine.mu.RUnlock()
				if !exists {
					t.Error("workflow not tracked after successful start")
				}
			}
		})
	}
}

// TestCancelWorkflow tests the CancelWorkflow method
func TestCancelWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		runID       string
		setupFunc   func(*AlpineWorkflowEngine)
		expectError bool
		errorMsg    string
	}{
		{
			name:  "successful cancellation",
			runID: "run-123",
			setupFunc: func(engine *AlpineWorkflowEngine) {
				// Create a mock workflow instance
				ctx, cancel := context.WithCancel(context.Background())
				engine.workflows["run-123"] = &workflowInstance{
					cancel:      cancel,
					ctx:         ctx,
					worktreeDir: "/tmp/test",
					events:      make(chan WorkflowEvent, 1),
				}
			},
			expectError: false,
		},
		{
			name:        "cancel non-existent workflow",
			runID:       "run-not-exists",
			setupFunc:   func(engine *AlpineWorkflowEngine) {},
			expectError: true,
			errorMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create engine with mocks
			engine := NewAlpineWorkflowEngine(
				&MockClaudeExecutor{},
				&MockWorktreeManager{},
				&config.Config{},
			)

			// Setup test state
			if tt.setupFunc != nil {
				tt.setupFunc(engine)
			}

			// Cancel workflow
			err := engine.CancelWorkflow(context.Background(), tt.runID)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				// Verify workflow was cancelled
				engine.mu.RLock()
				instance, exists := engine.workflows[tt.runID]
				engine.mu.RUnlock()
				if exists {
					select {
					case <-instance.ctx.Done():
						// Context should be cancelled
					default:
						t.Error("workflow context not cancelled")
					}
				}
			}
		})
	}
}

// TestGetWorkflowState tests the GetWorkflowState method
func TestGetWorkflowState(t *testing.T) {
	tests := []struct {
		name        string
		runID       string
		setupFunc   func(*AlpineWorkflowEngine, string)
		expectError bool
		expectState *core.State
	}{
		{
			name:  "successful state retrieval",
			runID: "run-123",
			setupFunc: func(engine *AlpineWorkflowEngine, tempDir string) {
				// Create workflow directory
				workDir := filepath.Join(tempDir, "run-123")
				_ = os.MkdirAll(filepath.Join(workDir, "agent_state"), 0755)

				// Create state file
				state := &core.State{
					CurrentStepDescription: "Testing",
					NextStepPrompt:         "/continue",
					Status:                 core.StatusRunning,
				}
				stateFile := filepath.Join(workDir, stateFileRelativePath)
				_ = state.Save(stateFile)

				// Track workflow
				engine.workflows["run-123"] = &workflowInstance{
					worktreeDir: workDir,
					events:      make(chan WorkflowEvent, 1),
					stateFile:   filepath.Join(workDir, stateFileRelativePath),
				}
			},
			expectState: &core.State{
				CurrentStepDescription: "Testing",
				NextStepPrompt:         "/continue",
				Status:                 core.StatusRunning,
			},
		},
		{
			name:        "workflow not found",
			runID:       "run-not-exists",
			setupFunc:   func(engine *AlpineWorkflowEngine, tempDir string) {},
			expectError: true,
		},
		{
			name:  "state file not found returns empty state",
			runID: "run-456",
			setupFunc: func(engine *AlpineWorkflowEngine, tempDir string) {
				// Track workflow without state file
				engine.workflows["run-456"] = &workflowInstance{
					worktreeDir: filepath.Join(tempDir, "run-456"),
					events:      make(chan WorkflowEvent, 1),
					stateFile:   filepath.Join(filepath.Join(tempDir, "run-456"), stateFileRelativePath),
				}
			},
			expectError: false,
			expectState: &core.State{}, // Empty state is returned when file doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create engine with mocks
			engine := NewAlpineWorkflowEngine(
				&MockClaudeExecutor{},
				&MockWorktreeManager{},
				&config.Config{},
			)

			// Setup test state
			if tt.setupFunc != nil {
				tt.setupFunc(engine, tempDir)
			}

			// Get workflow state
			state, err := engine.GetWorkflowState(context.Background(), tt.runID)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if state == nil {
					t.Fatal("expected state but got nil")
				}
				if state.CurrentStepDescription != tt.expectState.CurrentStepDescription {
					t.Errorf("expected description %s, got %s",
						tt.expectState.CurrentStepDescription,
						state.CurrentStepDescription)
				}
				if state.Status != tt.expectState.Status {
					t.Errorf("expected status %s, got %s",
						tt.expectState.Status,
						state.Status)
				}
			}
		})
	}
}

// TestApprovePlan tests the ApprovePlan method
func TestApprovePlan(t *testing.T) {
	tests := []struct {
		name         string
		runID        string
		setupFunc    func(*AlpineWorkflowEngine, string)
		executeError error
		expectError  bool
	}{
		{
			name:  "successful plan approval",
			runID: "run-123",
			setupFunc: func(engine *AlpineWorkflowEngine, tempDir string) {
				// Create workflow directory with state
				workDir := filepath.Join(tempDir, "run-123")
				_ = os.MkdirAll(filepath.Join(workDir, "agent_state"), 0755)

				// Create initial state
				state := &core.State{
					CurrentStepDescription: "Waiting for plan approval",
					NextStepPrompt:         "/make_plan",
					Status:                 core.StatusRunning,
				}
				stateFile := filepath.Join(workDir, stateFileRelativePath)
				_ = state.Save(stateFile)

				// Track workflow
				ctx, cancel := context.WithCancel(context.Background())
				ctx = context.WithValue(ctx, issueURLKey, "https://github.com/owner/repo/issues/123")
				engine.workflows["run-123"] = &workflowInstance{
					worktreeDir: workDir,
					events:      make(chan WorkflowEvent, 100),
					ctx:         ctx,
					cancel:      cancel,
					stateFile:   stateFile,
				}
			},
			expectError: false,
		},
		{
			name:        "workflow not found",
			runID:       "run-not-exists",
			setupFunc:   func(engine *AlpineWorkflowEngine, tempDir string) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create engine with mocks - ApprovePlan doesn't execute, it just updates state
			mockExecutor := &MockClaudeExecutor{}

			engine := NewAlpineWorkflowEngine(
				mockExecutor,
				&MockWorktreeManager{},
				&config.Config{},
			)

			// Setup test state
			if tt.setupFunc != nil {
				tt.setupFunc(engine, tempDir)
			}

			// Approve plan
			err := engine.ApprovePlan(context.Background(), tt.runID)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// For successful plan approval, verify the state was updated correctly
				if tt.name == "successful plan approval" {
					// Load the state and verify it contains the correct /start command
					if instance, exists := engine.workflows[tt.runID]; exists {
						state, loadErr := core.LoadState(instance.stateFile)
						if loadErr != nil {
							t.Errorf("failed to load state after approval: %v", loadErr)
						} else {
							expectedPrompt := "/start https://github.com/owner/repo/issues/123"
							if state.NextStepPrompt != expectedPrompt {
								t.Errorf("expected NextStepPrompt to be %q, got %q", expectedPrompt, state.NextStepPrompt)
							}
							if state.CurrentStepDescription != "Plan approved, continuing implementation" {
								t.Errorf("expected CurrentStepDescription to be updated")
							}
						}
					}
				}
			}
		})
	}
}

// TestSubscribeToEvents tests the SubscribeToEvents method
func TestSubscribeToEvents(t *testing.T) {
	tests := []struct {
		name        string
		runID       string
		setupFunc   func(*AlpineWorkflowEngine)
		expectError bool
		sendEvents  []WorkflowEvent
	}{
		{
			name:  "successful event subscription",
			runID: "run-123",
			setupFunc: func(engine *AlpineWorkflowEngine) {
				// Create workflow with event channel
				eventChan := make(chan WorkflowEvent, 10)
				engine.workflows["run-123"] = &workflowInstance{
					events:      eventChan,
					worktreeDir: "/tmp/test",
					stateFile:   "/tmp/test/agent_state/agent_state.json", // Will return empty state
				}
			},
			sendEvents: []WorkflowEvent{
				{
					Type:      "state_changed",
					RunID:     "run-123",
					Timestamp: time.Now(),
					Data:      map[string]interface{}{"status": "running"},
				},
				{
					Type:      "log",
					RunID:     "run-123",
					Timestamp: time.Now(),
					Data:      map[string]interface{}{"message": "Processing..."},
				},
			},
		},
		{
			name:        "workflow not found",
			runID:       "run-not-exists",
			setupFunc:   func(engine *AlpineWorkflowEngine) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create engine with mocks
			engine := NewAlpineWorkflowEngine(
				&MockClaudeExecutor{},
				&MockWorktreeManager{},
				&config.Config{},
			)

			// Setup test state
			if tt.setupFunc != nil {
				tt.setupFunc(engine)
			}

			// Subscribe to events
			eventChan, err := engine.SubscribeToEvents(context.Background(), tt.runID)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if eventChan == nil {
					t.Fatal("expected event channel but got nil")
				}

				// First event should be the current state
				select {
				case event := <-eventChan:
					if event.Type != "state_changed" {
						t.Errorf("expected first event to be state_changed, got %s", event.Type)
					}
				case <-time.After(100 * time.Millisecond):
					t.Error("timeout waiting for initial state event")
				}

				// Send test events if workflow exists
				if instance, exists := engine.workflows[tt.runID]; exists {
					for _, event := range tt.sendEvents {
						instance.events <- event
					}
				}

				// Verify events are received
				for i, expectedEvent := range tt.sendEvents {
					select {
					case event := <-eventChan:
						if event.Type != expectedEvent.Type {
							t.Errorf("event %d: expected type %s, got %s",
								i, expectedEvent.Type, event.Type)
						}
					case <-time.After(100 * time.Millisecond):
						t.Errorf("timeout waiting for event %d", i)
					}
				}
			}
		})
	}
}

// TestCleanup tests the Cleanup method
func TestCleanup(t *testing.T) {
	// Create engine with mocks
	engine := NewAlpineWorkflowEngine(
		&MockClaudeExecutor{},
		&MockWorktreeManager{},
		&config.Config{},
	)

	// Create multiple workflows
	workflows := []string{"run-123", "run-456", "run-789"}
	for _, runID := range workflows {
		ctx, cancel := context.WithCancel(context.Background())
		events := make(chan WorkflowEvent, 1)
		engine.workflows[runID] = &workflowInstance{
			ctx:         ctx,
			cancel:      cancel,
			events:      events,
			worktreeDir: "/tmp/" + runID,
			createdAt:   time.Now(),
		}
	}

	// Call cleanup for each workflow
	for _, runID := range workflows {
		engine.Cleanup(runID)
	}

	// Verify workflows map is empty
	engine.mu.RLock()
	defer engine.mu.RUnlock()
	if len(engine.workflows) != 0 {
		t.Errorf("expected workflows map to be empty, got %d entries", len(engine.workflows))
	}
}

// TestCreateWorkflowDirectory tests the createWorkflowDirectory method
func TestCreateWorkflowDirectory(t *testing.T) {
	t.Run("with git repository", func(t *testing.T) {
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       "/tmp/worktree-" + taskName,
					Branch:     "alpine/" + taskName,
					ParentRepo: "/tmp/repo",
				}, nil
			},
		}

		engine := NewAlpineWorkflowEngine(
			&MockClaudeExecutor{},
			mockWtMgr,
			&config.Config{
				Git: config.GitConfig{
					WorktreeEnabled: true,
				},
			},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dir, err := engine.createWorkflowDirectory(ctx, "run-123", cancel)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(dir, "/tmp/worktree-") {
			t.Errorf("expected worktree directory, got %s", dir)
		}
	})

	t.Run("without git repository", func(t *testing.T) {
		engine := NewAlpineWorkflowEngine(
			&MockClaudeExecutor{},
			nil, // No worktree manager
			&config.Config{
				Git: config.GitConfig{
					WorktreeEnabled: false,
				},
			},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dir, err := engine.createWorkflowDirectory(ctx, "run-456", cancel)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(dir, tempDirPrefix) {
			t.Errorf("expected temp directory with prefix %s, got %s", tempDirPrefix, dir)
		}
	})
}

// TestRunWorkflowAsync tests the runWorkflowAsync method
func TestRunWorkflowAsync(t *testing.T) {
	t.Run("successful workflow execution", func(t *testing.T) {
		tempDir := t.TempDir()
		stateDir := filepath.Join(tempDir, "agent_state")
		_ = os.MkdirAll(stateDir, 0755)

		// Create initial state
		initialState := &core.State{
			CurrentStepDescription: "Starting",
			NextStepPrompt:         "/make_plan",
			Status:                 core.StatusRunning,
		}
		stateFile := filepath.Join(tempDir, stateFileRelativePath)
		_ = initialState.Save(stateFile)

		executeCalls := 0
		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, cfg claude.ExecuteConfig) (string, error) {
				executeCalls++
				// Update state to completed after first execution
				if executeCalls == 1 {
					completedState := &core.State{
						CurrentStepDescription: "Completed",
						NextStepPrompt:         "",
						Status:                 core.StatusCompleted,
					}
					_ = completedState.Save(stateFile)
				}
				return "", nil
			},
		}

		engine := NewAlpineWorkflowEngine(
			mockExecutor,
			&MockWorktreeManager{},
			&config.Config{WorkDir: tempDir},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a workflow engine for the instance
		workflowEngine := workflow.NewEngine(mockExecutor, &MockWorktreeManager{}, &config.Config{WorkDir: tempDir}, nil)
		workflowEngine.SetStateFile(stateFile)

		instance := &workflowInstance{
			engine:      workflowEngine,
			ctx:         ctx,
			cancel:      cancel,
			worktreeDir: tempDir,
			events:      make(chan WorkflowEvent, 100),
			stateFile:   stateFile,
			createdAt:   time.Now(),
		}

		// Collect events
		var collectedEvents []WorkflowEvent
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := range instance.events {
				collectedEvents = append(collectedEvents, event)
			}
		}()

		// Run workflow
		go engine.runWorkflowAsync(instance, "Test task", "run-123", true)

		// Wait for the collector goroutine to finish (which happens when channel is closed)
		wg.Wait()

		// Verify execution
		if executeCalls < 1 {
			t.Error("expected at least one execution")
		}

		// Verify events were sent
		foundStart := false
		foundCompletion := false
		for _, event := range collectedEvents {
			if event.Type == "workflow_started" {
				foundStart = true
			}
			if event.Type == "workflow_completed" {
				foundCompletion = true
			}
		}
		if !foundStart {
			t.Error("expected workflow_started event")
		}
		if !foundCompletion {
			t.Error("expected workflow_completed event")
		}
	})

	t.Run("workflow execution with error", func(t *testing.T) {
		tempDir := t.TempDir()
		stateFile := filepath.Join(tempDir, stateFileRelativePath)

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, cfg claude.ExecuteConfig) (string, error) {
				return "", fmt.Errorf("execution failed")
			},
		}

		engine := NewAlpineWorkflowEngine(
			mockExecutor,
			&MockWorktreeManager{},
			&config.Config{WorkDir: tempDir},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create a workflow engine for the instance
		workflowEngine := workflow.NewEngine(mockExecutor, &MockWorktreeManager{}, &config.Config{WorkDir: tempDir}, nil)
		workflowEngine.SetStateFile(stateFile)

		instance := &workflowInstance{
			engine:      workflowEngine,
			ctx:         ctx,
			cancel:      cancel,
			worktreeDir: tempDir,
			events:      make(chan WorkflowEvent, 100),
			stateFile:   stateFile,
			createdAt:   time.Now(),
		}

		// Collect events
		var errorEvent *WorkflowEvent
		done := make(chan bool)
		go func() {
			for event := range instance.events {
				if event.Type == "error" || event.Type == "workflow_failed" {
					errorEvent = &event
					done <- true
					return
				}
			}
		}()

		// Run workflow
		go engine.runWorkflowAsync(instance, "Test task", "run-456", true)

		// Wait for error event
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for error event")
		}

		// Verify error event
		if errorEvent == nil {
			t.Fatal("expected error event")
		}
		if errorEvent.RunID != "run-456" {
			t.Errorf("expected run ID run-456, got %s", errorEvent.RunID)
		}
	})
}

// TestSendEventNonBlocking tests the sendEventNonBlocking method
func TestSendEventNonBlocking(t *testing.T) {
	engine := NewAlpineWorkflowEngine(
		&MockClaudeExecutor{},
		&MockWorktreeManager{},
		&config.Config{},
	)

	t.Run("successful send", func(t *testing.T) {
		eventChan := make(chan WorkflowEvent, 1)
		event := WorkflowEvent{
			Type:      "test",
			RunID:     "run-123",
			Timestamp: time.Now(),
		}

		instance := &workflowInstance{
			events: eventChan,
		}
		engine.sendEventNonBlocking(instance, event)

		// Verify event was sent
		select {
		case received := <-eventChan:
			if received.Type != "test" {
				t.Errorf("expected event type 'test', got '%s'", received.Type)
			}
		default:
			t.Error("expected event in channel")
		}
	})

	t.Run("channel full - should not block", func(t *testing.T) {
		// Create full channel
		eventChan := make(chan WorkflowEvent, 1)
		eventChan <- WorkflowEvent{Type: "existing"}

		// This should not block
		done := make(chan bool)
		go func() {
			instance := &workflowInstance{
				events: eventChan,
			}
			engine.sendEventNonBlocking(instance, WorkflowEvent{Type: "new"})
			done <- true
		}()

		// Should complete quickly without blocking
		select {
		case <-done:
			// Success - didn't block
		case <-time.After(100 * time.Millisecond):
			t.Error("sendEventNonBlocking blocked on full channel")
		}
	})
}

// TestConcurrentOperations tests thread safety of concurrent operations
func TestConcurrentOperations(t *testing.T) {
	engine := NewAlpineWorkflowEngine(
		&MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, cfg claude.ExecuteConfig) (string, error) {
				// Simulate some work
				time.Sleep(10 * time.Millisecond)
				return "", nil
			},
		},
		&MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       "/tmp/" + taskName,
					Branch:     "alpine/" + taskName,
					ParentRepo: "/tmp/repo",
				}, nil
			},
		},
		&config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
			},
		},
	)

	// Start multiple workflows concurrently
	var wg sync.WaitGroup
	runIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		runID := fmt.Sprintf("run-%d", i)
		runIDs[i] = runID
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, err := engine.StartWorkflow(context.Background(),
				"https://github.com/test/repo/issues/1", id, true)
			if err != nil {
				t.Errorf("failed to start workflow %s: %v", id, err)
			}
		}(runID)
	}

	wg.Wait()

	// Verify all workflows were created
	engine.mu.RLock()
	workflowCount := len(engine.workflows)
	engine.mu.RUnlock()

	if workflowCount != 10 {
		t.Errorf("expected 10 workflows, got %d", workflowCount)
	}

	// Cancel workflows concurrently
	for _, runID := range runIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			err := engine.CancelWorkflow(context.Background(), id)
			if err != nil {
				t.Errorf("failed to cancel workflow %s: %v", id, err)
			}
		}(runID)
	}

	wg.Wait()

	// Clean up
	for _, runID := range runIDs {
		engine.Cleanup(runID)
	}
}
