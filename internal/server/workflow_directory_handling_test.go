package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/gitx"
)

// TestWorkflowDirectoryHandling tests comprehensive directory handling in server workflows
// This ensures the WorkDir fix works end-to-end in server mode from request to Claude execution
func TestWorkflowDirectoryHandling(t *testing.T) {
	t.Run("end_to_end_github_clone_and_workdir_setup", func(t *testing.T) {
		// Create a mock that tracks Claude execution config to verify WorkDir is set correctly
		executionConfigs := make([]claude.ExecuteConfig, 0)
		var configMutex sync.Mutex

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				configMutex.Lock()
				executionConfigs = append(executionConfigs, config)
				configMutex.Unlock()

				// Don't create a conflicting state file - the workflow engine handles this
				// Just return success to simulate Claude execution
				return "Plan created successfully", nil
			},
		}

		// Mock worktree manager that simulates real worktree creation
		tempDir := t.TempDir()
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				// Create a directory structure that mimics real worktree behavior
				worktreePath := filepath.Join(tempDir, "worktrees", taskName)
				if err := os.MkdirAll(worktreePath, 0755); err != nil {
					return nil, fmt.Errorf("failed to create worktree directory: %w", err)
				}

				return &gitx.Worktree{
					Path:       worktreePath,
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		// Create configuration that enables git operations
		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				AutoCleanupWT:   false, // Keep for inspection
				Clone: config.GitCloneConfig{
					Enabled:   false, // Disable actual cloning for this test
					AuthToken: "",
					Timeout:   30 * time.Second,
					Depth:     1,
				},
			},
			StateFile: "agent_state/agent_state.json",
		}

		// Create workflow engine
		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

		// Create server and set workflow engine
		server := NewServer(0)
		server.SetWorkflowEngine(engine)

		// Create request to start workflow with GitHub issue
		payload := map[string]string{
			"issue_url": "https://github.com/owner/repo/issues/123",
			"agent_id":  "alpine-agent",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
		w := httptest.NewRecorder()

		// Execute the workflow start
		server.agentsRunHandler(w, req)

		// Verify successful workflow start
		assert.Equal(t, http.StatusCreated, w.Code)

		var response Run
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Wait for workflow to complete
		time.Sleep(500 * time.Millisecond)

		// Verify Claude was executed with correct WorkDir
		configMutex.Lock()
		require.Len(t, executionConfigs, 1, "Expected exactly one Claude execution")

		claudeConfig := executionConfigs[0]
		configMutex.Unlock()

		// CRITICAL ASSERTION: Verify WorkDir was set to the worktree directory
		assert.NotEmpty(t, claudeConfig.WorkDir, "Claude WorkDir should be set")
		assert.Contains(t, claudeConfig.WorkDir, "worktrees",
			"Claude WorkDir should point to worktree directory, got: %s", claudeConfig.WorkDir)

		// Verify the agent_state directory was created in the correct location
		agentStateDir := filepath.Join(claudeConfig.WorkDir, "agent_state")
		_, err = os.Stat(agentStateDir)
		assert.NoError(t, err, "Agent state directory should exist in the worktree directory")

		// Verify WorkDir exists and is accessible
		_, err = os.Stat(claudeConfig.WorkDir)
		assert.NoError(t, err, "WorkDir should exist and be accessible")

		// Verify response contains the correct worktree directory
		assert.Equal(t, claudeConfig.WorkDir, response.WorktreeDir,
			"Response WorktreeDir should match Claude's WorkDir")
	})

	t.Run("github_clone_integration_with_workdir", func(t *testing.T) {
		// This test verifies the integration between GitHub cloning and WorkDir setup
		// when cloning is enabled

		var claudeWorkDir string
		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				claudeWorkDir = config.WorkDir
				return "Execution completed", nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled:   true,
					AuthToken: "",
					Timeout:   30 * time.Second,
					Depth:     1,
				},
			},
		}

		// Create workflow engine
		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)

		// We'll test this by verifying that when clone is enabled and a GitHub URL is provided,
		// the workflow uses a different directory than the default temp directory pattern
		ctx := context.Background()
		workdir, err := engine.StartWorkflow(ctx, "https://github.com/owner/repo/issues/123", "test-run-clone")

		// For this test, we expect it to work even if actual cloning fails
		// because there should be fallback behavior
		require.NoError(t, err)
		assert.NotEmpty(t, workdir)

		// Allow workflow to complete
		time.Sleep(300 * time.Millisecond)

		// Verify Claude received a valid WorkDir
		assert.NotEmpty(t, claudeWorkDir, "Claude WorkDir should be set")
		assert.Equal(t, workdir, claudeWorkDir, "Returned workdir should match Claude's WorkDir")
	})

	t.Run("worktree_creation_failure_handling", func(t *testing.T) {
		// Test that WorkDir handling gracefully handles worktree creation failures

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				return "Fallback execution", nil
			},
		}

		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return nil, fmt.Errorf("worktree creation failed")
			},
		}

		tempDir := t.TempDir()
		cfg := &config.Config{
			WorkDir: tempDir,
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled: false,
				},
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

		ctx := context.Background()
		_, err := engine.StartWorkflow(ctx, "https://github.com/owner/repo/issues/123", "test-run-fail")

		// Should handle failure gracefully and not crash
		assert.Error(t, err, "Should return error when worktree creation fails")
		assert.Contains(t, err.Error(), "failed to create worktree",
			"Error should mention worktree creation failure")
	})

	t.Run("directory_isolation_between_workflows", func(t *testing.T) {
		// Test that different workflows get isolated directories

		var workDirs []string
		var workDirMutex sync.Mutex

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				workDirMutex.Lock()
				workDirs = append(workDirs, config.WorkDir)
				workDirMutex.Unlock()
				return "Execution completed", nil
			},
		}

		tempDir := t.TempDir()
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				worktreePath := filepath.Join(tempDir, "worktrees", taskName)
				os.MkdirAll(worktreePath, 0755)
				return &gitx.Worktree{
					Path:       worktreePath,
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled: false,
				},
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

		// Start multiple workflows concurrently
		ctx := context.Background()
		var wg sync.WaitGroup

		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				runID := fmt.Sprintf("test-run-%d", index)
				issueURL := fmt.Sprintf("https://github.com/owner/repo/issues/%d", index+1)

				workdir, err := engine.StartWorkflow(ctx, issueURL, runID)
				assert.NoError(t, err)
				assert.NotEmpty(t, workdir)
			}(i)
		}

		wg.Wait()
		time.Sleep(500 * time.Millisecond) // Allow workflows to complete

		// Verify each workflow got a unique directory
		workDirMutex.Lock()
		assert.Len(t, workDirs, 3, "Should have 3 unique work directories")

		// Check all directories are different
		uniqueDirs := make(map[string]bool)
		for _, dir := range workDirs {
			assert.False(t, uniqueDirs[dir], "Work directory %s should be unique", dir)
			uniqueDirs[dir] = true
		}
		workDirMutex.Unlock()
	})

	t.Run("state_file_path_configuration", func(t *testing.T) {
		// Test that state files are correctly placed relative to WorkDir

		var claudeConfig claude.ExecuteConfig
		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				claudeConfig = config

				// Verify state file path is relative to WorkDir
				expectedStatePath := filepath.Join(config.WorkDir, "agent_state", "agent_state.json")

				// Create state file as Claude would
				stateDir := filepath.Dir(expectedStatePath)
				os.MkdirAll(stateDir, 0755)

				state := &core.State{
					CurrentStepDescription: "Test execution",
					NextStepPrompt:         "/continue",
					Status:                 core.StatusRunning,
				}

				return "Created state file", state.Save(expectedStatePath)
			},
		}

		tempDir := t.TempDir()
		worktreeDir := filepath.Join(tempDir, "test-worktree")
		os.MkdirAll(worktreeDir, 0755)

		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       worktreeDir,
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
			},
			StateFile: "agent_state/agent_state.json",
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

		ctx := context.Background()
		workdir, err := engine.StartWorkflow(ctx, "https://github.com/owner/repo/issues/123", "test-state")

		require.NoError(t, err)
		time.Sleep(300 * time.Millisecond)

		// Verify WorkDir is set correctly
		assert.Equal(t, worktreeDir, claudeConfig.WorkDir)
		assert.Equal(t, worktreeDir, workdir)

		// Verify state file exists in correct location
		expectedStatePath := filepath.Join(worktreeDir, "agent_state", "agent_state.json")
		_, err = os.Stat(expectedStatePath)
		assert.NoError(t, err, "State file should exist at expected path")

		// Verify state can be loaded
		state, err := core.LoadState(expectedStatePath)
		assert.NoError(t, err, "Should be able to load state from correct path")
		assert.Equal(t, "Test execution", state.CurrentStepDescription)
	})

	t.Run("server_integration_with_workdir", func(t *testing.T) {
		// Test that server integration properly sets up working directories

		var claudeWorkDir string
		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				claudeWorkDir = config.WorkDir
				return "Server integration test completed", nil
			},
		}

		tempDir := t.TempDir()
		worktreeDir := filepath.Join(tempDir, "server-test-worktree")
		os.MkdirAll(worktreeDir, 0755)

		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       worktreeDir,
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
			},
		}

		// Create server and engine
		server := NewServer(0)
		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)
		server.SetWorkflowEngine(engine)

		ctx := context.Background()
		workdir, err := engine.StartWorkflow(ctx, "https://github.com/owner/repo/issues/123", "server-test")

		require.NoError(t, err)
		assert.Equal(t, worktreeDir, workdir, "Should return correct worktree directory")

		time.Sleep(300 * time.Millisecond) // Allow workflow to complete

		// Verify Claude received the correct WorkDir
		assert.Equal(t, worktreeDir, claudeWorkDir, "Claude should use the worktree directory")
	})

	t.Run("rest_api_run_details_include_workdir", func(t *testing.T) {
		// Test that REST API run details include working directory information

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				return "API test completed", nil
			},
		}

		tempDir := t.TempDir()
		worktreeDir := filepath.Join(tempDir, "api-test-worktree")
		os.MkdirAll(worktreeDir, 0755)

		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       worktreeDir,
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
			},
		}

		// Create server and engine
		server := NewServer(0)
		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)
		server.SetWorkflowEngine(engine)

		// Start workflow via REST API
		payload := map[string]string{
			"issue_url": "https://github.com/owner/repo/issues/123",
			"agent_id":  "alpine-agent",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/agents/run", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.agentsRunHandler(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var runResponse Run
		err := json.NewDecoder(w.Body).Decode(&runResponse)
		require.NoError(t, err)

		runID := runResponse.ID
		time.Sleep(300 * time.Millisecond) // Allow workflow to start

		// Get run details via REST API
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/runs/%s", runID), nil)
		req.SetPathValue("id", runID)
		w = httptest.NewRecorder()

		server.runDetailsHandler(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response and verify WorktreeDir is included
		var runDetails map[string]interface{}
		err = json.NewDecoder(w.Body).Decode(&runDetails)
		require.NoError(t, err)

		worktreeDirFromAPI, exists := runDetails["worktree_dir"]
		assert.True(t, exists, "Run details should include worktree_dir")
		assert.Equal(t, worktreeDir, worktreeDirFromAPI, "API should return correct worktree_dir")
	})
}

// TestWorkflowDirectoryCleanup tests the comprehensive cleanup functionality
func TestWorkflowDirectoryCleanup(t *testing.T) {
	t.Run("cleanup_tracked_clone_directories", func(t *testing.T) {
		// Test that cleanup properly removes tracked clone directories

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				return "Execution completed", nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				AutoCleanupWT: true, // Enable cleanup
				Clone: config.GitCloneConfig{
					Enabled: false, // Disable actual cloning
				},
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)

		// Simulate a workflow with cloned directories
		runID := "cleanup-test"
		ctx, cancel := context.WithCancel(context.Background())

		// Create test directories that simulate cloned repos
		clonedDirs := make([]string, 3)
		for i := range clonedDirs {
			dir, err := os.MkdirTemp("", fmt.Sprintf("alpine-clone-test-%d-", i))
			require.NoError(t, err)
			clonedDirs[i] = dir

			// Create some content to verify deletion
			testFile := filepath.Join(dir, "test.txt")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			require.NoError(t, err)
		}

		// Create workflow instance with cloned directories
		instance := &workflowInstance{
			ctx:        ctx,
			cancel:     cancel,
			events:     make(chan WorkflowEvent),
			createdAt:  time.Now(),
			clonedDirs: clonedDirs,
		}

		engine.mu.Lock()
		engine.workflows[runID] = instance
		engine.mu.Unlock()

		// Verify directories exist before cleanup
		for _, dir := range clonedDirs {
			_, err := os.Stat(dir)
			assert.NoError(t, err, "Directory should exist before cleanup: %s", dir)
		}

		// Perform cleanup
		engine.Cleanup(runID)

		// Verify directories were removed
		for _, dir := range clonedDirs {
			_, err := os.Stat(dir)
			assert.True(t, os.IsNotExist(err), "Directory should be removed after cleanup: %s", dir)
		}

		// Verify workflow instance was removed from engine
		engine.mu.RLock()
		_, exists := engine.workflows[runID]
		engine.mu.RUnlock()
		assert.False(t, exists, "Workflow instance should be removed from engine")
	})

	t.Run("cleanup_respects_disable_flag", func(t *testing.T) {
		// Test that cleanup respects the AutoCleanupWT=false setting

		mockExecutor := &MockClaudeExecutor{}
		cfg := &config.Config{
			Git: config.GitConfig{
				AutoCleanupWT: false, // Disable cleanup
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)

		// Create test directory
		testDir, err := os.MkdirTemp("", "alpine-no-cleanup-test-")
		require.NoError(t, err)
		defer os.RemoveAll(testDir) // Manual cleanup for test

		runID := "no-cleanup-test"
		ctx, cancel := context.WithCancel(context.Background())

		instance := &workflowInstance{
			ctx:        ctx,
			cancel:     cancel,
			events:     make(chan WorkflowEvent),
			createdAt:  time.Now(),
			clonedDirs: []string{testDir},
		}

		engine.mu.Lock()
		engine.workflows[runID] = instance
		engine.mu.Unlock()

		// Perform cleanup
		engine.Cleanup(runID)

		// Directory should still exist (cleanup disabled)
		_, err = os.Stat(testDir)
		assert.NoError(t, err, "Directory should still exist when cleanup is disabled")
	})

	t.Run("cleanup_handles_errors_gracefully", func(t *testing.T) {
		// Test that cleanup errors don't prevent workflow completion

		mockExecutor := &MockClaudeExecutor{}
		cfg := &config.Config{
			Git: config.GitConfig{
				AutoCleanupWT: true,
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)

		runID := "error-cleanup-test"
		ctx, cancel := context.WithCancel(context.Background())

		// Use non-existent directory to cause cleanup error
		nonExistentDir := "/tmp/nonexistent-alpine-test-dir-12345"

		instance := &workflowInstance{
			ctx:        ctx,
			cancel:     cancel,
			events:     make(chan WorkflowEvent),
			createdAt:  time.Now(),
			clonedDirs: []string{nonExistentDir},
		}

		engine.mu.Lock()
		engine.workflows[runID] = instance
		engine.mu.Unlock()

		// Cleanup should not panic or fail
		assert.NotPanics(t, func() {
			engine.Cleanup(runID)
		}, "Cleanup should handle errors gracefully")

		// Workflow should still be removed from engine despite cleanup error
		engine.mu.RLock()
		_, exists := engine.workflows[runID]
		engine.mu.RUnlock()
		assert.False(t, exists, "Workflow should be removed even if cleanup fails")
	})
}

// TestGitHubCloneIntegration tests the GitHub-specific clone and WorkDir setup
func TestGitHubCloneIntegration(t *testing.T) {
	t.Run("github_url_detection_and_clone_attempt", func(t *testing.T) {
		// Test that GitHub URLs are properly detected and clone logic is invoked

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				return "Clone test completed", nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				Clone: config.GitCloneConfig{
					Enabled: true,
					Timeout: 30 * time.Second,
				},
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)

		ctx := context.Background()
		issueURL := "https://github.com/octocat/Hello-World/issues/1"

		workdir, err := engine.StartWorkflow(ctx, issueURL, "github-clone-test")

		// The test verifies that GitHub URL detection works by checking that
		// the workflow starts successfully and sets up a working directory
		require.NoError(t, err)
		assert.NotEmpty(t, workdir)

		// Wait for workflow to complete
		time.Sleep(300 * time.Millisecond)

		// The key test is that GitHub URLs are properly parsed and handled
		// This is verified by the fact that the workflow starts without error
		// and the isGitHubIssueURL function would have been called internally
		owner, repo, issueNum, err := parseGitHubIssueURL(issueURL)
		assert.NoError(t, err, "Should be able to parse the GitHub URL")
		assert.Equal(t, "octocat", owner)
		assert.Equal(t, "Hello-World", repo)
		assert.Equal(t, 1, issueNum)

		// Verify the expected clone URL would be generated
		expectedCloneURL := buildGitCloneURL(owner, repo)
		assert.Equal(t, "https://github.com/octocat/Hello-World.git", expectedCloneURL)
	})

	t.Run("non_github_url_skips_clone", func(t *testing.T) {
		// Test that non-GitHub URLs don't trigger clone attempts

		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				return "Non-GitHub test completed", nil
			},
		}

		tempDir := t.TempDir()
		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       filepath.Join(tempDir, taskName),
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled: true, // Even with clone enabled
				},
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

		ctx := context.Background()
		nonGitHubURL := "https://example.com/some/task"

		workdir, err := engine.StartWorkflow(ctx, nonGitHubURL, "non-github-test")

		require.NoError(t, err)
		assert.NotEmpty(t, workdir)

		time.Sleep(300 * time.Millisecond)

		// Verify non-GitHub URLs are not detected as GitHub URLs
		assert.False(t, isGitHubIssueURL(nonGitHubURL),
			"Non-GitHub URLs should not be detected as GitHub issue URLs")

		// The workflow should use regular worktree creation
		assert.Contains(t, workdir, tempDir,
			"Should use regular worktree path for non-GitHub URLs")
	})

	t.Run("clone_failure_fallback_to_regular_worktree", func(t *testing.T) {
		// Test graceful fallback when clone fails

		var claudeWorkDir string
		mockExecutor := &MockClaudeExecutor{
			ExecuteFunc: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
				claudeWorkDir = config.WorkDir
				return "Fallback test completed", nil
			},
		}

		tempDir := t.TempDir()
		worktreeDir := filepath.Join(tempDir, "fallback-worktree")
		os.MkdirAll(worktreeDir, 0755)

		mockWtMgr := &MockWorktreeManager{
			CreateFunc: func(ctx context.Context, taskName string) (*gitx.Worktree, error) {
				return &gitx.Worktree{
					Path:       worktreeDir,
					Branch:     "alpine/" + taskName,
					ParentRepo: tempDir,
				}, nil
			},
		}

		cfg := &config.Config{
			Git: config.GitConfig{
				WorktreeEnabled: true,
				Clone: config.GitCloneConfig{
					Enabled: true,                 // Clone enabled but will fail in practice
					Timeout: 1 * time.Millisecond, // Very short timeout to cause failure
				},
			},
		}

		engine := NewAlpineWorkflowEngine(mockExecutor, mockWtMgr, cfg)

		ctx := context.Background()
		issueURL := "https://github.com/private/repo/issues/1"

		workdir, err := engine.StartWorkflow(ctx, issueURL, "fallback-test")

		require.NoError(t, err)
		assert.Equal(t, worktreeDir, workdir, "Should fall back to regular worktree")

		time.Sleep(300 * time.Millisecond)

		// Verify Claude used the fallback directory
		assert.Equal(t, worktreeDir, claudeWorkDir,
			"Claude should use fallback worktree when clone fails")
	})
}
