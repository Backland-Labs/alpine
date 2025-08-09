package server

import (
	"context"
	"testing"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowIntegrationWithToolCallHooks tests that tool call hooks are properly
// integrated with workflow execution when the feature is enabled
func TestWorkflowIntegrationWithToolCallHooks(t *testing.T) {
	t.Run("enables tool call hooks when feature is enabled in server mode", func(t *testing.T) {
		// Create config with tool call events enabled
		cfg := &config.Config{
			ToolCallEvents: config.ToolCallEventsConfig{
				Enabled:    true,
				BatchSize:  10,
				SampleRate: 100,
			},
			Server: config.ServerConfig{
				Enabled: true,
				Port:    3001,
			},
		}

		// Create mock executor that tracks hook setup calls
		mockExecutor := &MockClaudeExecutorWithHooks{}

		// Create workflow engine
		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)

		// Mock server for event endpoint
		mockServer := &Server{
			port: 3001,
		}
		engine.SetServer(mockServer)

		// Start workflow - this should trigger hook setup
		ctx := context.Background()
		_, err := engine.StartWorkflow(ctx, "https://github.com/test/repo/issues/1", "test-run", false)

		// Should not error and should have called hook setup
		require.NoError(t, err)
		assert.True(t, mockExecutor.setupHooksCalled, "Tool call hooks should be set up when feature is enabled")
		assert.Contains(t, mockExecutor.eventEndpoint, "/events/tool-calls", "Event endpoint should be configured")
		assert.Equal(t, "test-run", mockExecutor.runID, "Run ID should be passed to hooks")
	})

	t.Run("skips tool call hooks when feature is disabled", func(t *testing.T) {
		// Create config with tool call events disabled
		cfg := &config.Config{
			ToolCallEvents: config.ToolCallEventsConfig{
				Enabled: false,
			},
			Server: config.ServerConfig{
				Enabled: true,
				Port:    3001,
			},
		}

		mockExecutor := &MockClaudeExecutorWithHooks{}

		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)
		mockServer := &Server{port: 3001}
		engine.SetServer(mockServer)

		ctx := context.Background()
		_, err := engine.StartWorkflow(ctx, "https://github.com/test/repo/issues/1", "test-run", false)

		require.NoError(t, err)
		assert.False(t, mockExecutor.setupHooksCalled, "Tool call hooks should not be set up when feature is disabled")
	})

	t.Run("cleans up hooks after workflow completion", func(t *testing.T) {
		cfg := &config.Config{
			ToolCallEvents: config.ToolCallEventsConfig{
				Enabled:    true,
				BatchSize:  5,
				SampleRate: 50,
			},
			Server: config.ServerConfig{
				Enabled: true,
				Port:    3001,
			},
		}

		mockExecutor := &MockClaudeExecutorWithHooks{}

		engine := NewAlpineWorkflowEngine(mockExecutor, nil, cfg)
		mockServer := &Server{port: 3001}
		engine.SetServer(mockServer)

		ctx := context.Background()
		_, err := engine.StartWorkflow(ctx, "https://github.com/test/repo/issues/1", "test-run", false)
		require.NoError(t, err)

		// Verify hooks were set up
		assert.True(t, mockExecutor.setupHooksCalled, "Tool call hooks should be set up")

		// Simulate workflow completion by calling cleanup directly
		// In a real scenario, this would be called when the workflow completes
		engine.Cleanup("test-run")

		// Cleanup should be called after workflow completion
		assert.True(t, mockExecutor.cleanupCalled, "Hook cleanup should be called after workflow completion")
	})
}

// MockClaudeExecutorWithHooks extends MockClaudeExecutor with hook functionality for testing
type MockClaudeExecutorWithHooks struct {
	MockClaudeExecutor
	setupHooksCalled bool
	cleanupCalled    bool
	eventEndpoint    string
	runID            string
	batchSize        int
	sampleRate       int
}

func (m *MockClaudeExecutorWithHooks) SetupToolCallEventHooks(eventEndpoint, runID string, batchSize, sampleRate int) (func(), error) {
	m.setupHooksCalled = true
	m.eventEndpoint = eventEndpoint
	m.runID = runID
	m.batchSize = batchSize
	m.sampleRate = sampleRate

	return func() {
		m.cleanupCalled = true
	}, nil
}
