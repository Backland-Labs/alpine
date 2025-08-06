package cli

import (
	"context"
	"testing"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/server"
)

// MockServerStreamer implements both server.Server interface and events.Streamer
type MockServerStreamer struct {
	StreamStartCalls   []StreamCall
	StreamContentCalls []StreamCall
	StreamEndCalls     []StreamCall
	StartCalled        bool
	Address            string
}

type StreamCall struct {
	RunID     string
	MessageID string
	Content   string
}

func (m *MockServerStreamer) StreamStart(runID, messageID string) error {
	m.StreamStartCalls = append(m.StreamStartCalls, StreamCall{RunID: runID, MessageID: messageID})
	return nil
}

func (m *MockServerStreamer) StreamContent(runID, messageID, content string) error {
	m.StreamContentCalls = append(m.StreamContentCalls, StreamCall{RunID: runID, MessageID: messageID, Content: content})
	return nil
}

func (m *MockServerStreamer) StreamEnd(runID, messageID string) error {
	m.StreamEndCalls = append(m.StreamEndCalls, StreamCall{RunID: runID, MessageID: messageID})
	return nil
}

func (m *MockServerStreamer) Start(ctx context.Context) error {
	m.StartCalled = true
	return nil
}

func (m *MockServerStreamer) GetAddress() string {
	return m.Address
}

// TestCLIStreamerIntegrationServerMode tests that server is passed as streamer in serve mode
func TestCLIStreamerIntegrationServerMode(t *testing.T) {
	// This test verifies that when --serve flag is used, the server instance
	// is properly passed as a Streamer to the workflow engine

	// Arrange
	ctx := context.WithValue(context.Background(), serveKey, true)
	mockServer := &MockServerStreamer{Address: "localhost:3001"}

	// We need to test that CreateWorkflowEngine is called with the server as streamer
	// This will be implemented in the GREEN phase

	// For now, this test documents the expected behavior:
	// 1. When serve=true in context, server should be created
	// 2. Server should be wrapped in ServerStreamer
	// 3. ServerStreamer should be passed to workflow engine

	_ = ctx        // Silence unused variable warning
	_ = mockServer // Silence unused variable warning
}

// TestCLIStreamerIntegrationCLIMode tests that no streamer is passed in CLI-only mode
func TestCLIStreamerIntegrationCLIMode(t *testing.T) {
	// This test verifies that when --serve flag is NOT used,
	// no streamer is passed to the workflow engine

	// Arrange
	ctx := context.Background() // No serve flag

	// We need to test that CreateWorkflowEngine is called with nil streamer
	// This will be implemented in the GREEN phase

	// For now, this test documents the expected behavior:
	// 1. When serve is not in context, no server is created
	// 2. Nil streamer should be passed to workflow engine
	// 3. Workflow should work normally without streaming

	_ = ctx // Silence unused variable warning
}

// TestCreateWorkflowEngineWithStreamer tests the updated CreateWorkflowEngine function
func TestCreateWorkflowEngineWithStreamer(t *testing.T) {
	// Test that CreateWorkflowEngine accepts and passes streamer parameter

	// Arrange
	cfg := &config.Config{
		Git: config.GitConfig{WorktreeEnabled: false},
	}
	mockStreamer := &MockServerStreamer{}

	// Act
	engine, wtMgr, _ := CreateWorkflowEngine(cfg, mockStreamer)

	// Assert
	if engine == nil {
		t.Error("Expected workflow engine to be created")
	}

	if wtMgr != nil && !cfg.Git.WorktreeEnabled {
		t.Error("Expected no worktree manager when worktree is disabled")
	}

	// TODO: In GREEN phase, verify that streamer was passed to the engine
}

// TestCreateWorkflowEngineWithoutStreamer tests backward compatibility
func TestCreateWorkflowEngineWithoutStreamer(t *testing.T) {
	// Test that CreateWorkflowEngine works with nil streamer

	// Arrange
	cfg := &config.Config{
		Git: config.GitConfig{WorktreeEnabled: false},
	}

	// Act
	engine, wtMgr, _ := CreateWorkflowEngine(cfg, nil)

	// Assert
	if engine == nil {
		t.Error("Expected workflow engine to be created")
	}

	if wtMgr != nil && !cfg.Git.WorktreeEnabled {
		t.Error("Expected no worktree manager when worktree is disabled")
	}
}

// TestServerAsStreamerWrapper tests that server can be wrapped as a Streamer
func TestServerAsStreamerWrapper(t *testing.T) {
	// This test verifies that we can properly wrap a server instance
	// to implement the Streamer interface using ServerStreamer

	// Arrange
	httpServer := server.NewServer(3001)
	streamer := server.NewServerStreamer(httpServer)

	// Act & Assert
	// Verify the wrapper implements Streamer interface
	var _ events.Streamer = streamer

	// The actual behavior will be tested in integration
}

// TestWorkflowIntegrationEndToEnd tests the complete integration
func TestWorkflowIntegrationEndToEnd(t *testing.T) {
	// This is a placeholder for end-to-end integration test
	// It will verify the complete flow from CLI → Server → Workflow → Executor
	// with proper streaming at each step

	// Will be implemented during GREEN phase
}

// TestRunWorkflowWithServerStreaming tests the full workflow with server streaming
func TestRunWorkflowWithServerStreaming(t *testing.T) {
	// This test will verify that runWorkflowWithDependencies properly
	// detects server mode and passes the server as a streamer

	// Test implementation will be completed during GREEN phase
}

// TestServerLifecycleWithWorkflow tests server starts and stops correctly
func TestServerLifecycleWithWorkflow(t *testing.T) {
	// This test ensures that:
	// 1. Server starts when --serve is set
	// 2. Server instance is available for streaming
	// 3. Server shuts down gracefully after workflow

	// Test implementation will be completed during GREEN phase
}
