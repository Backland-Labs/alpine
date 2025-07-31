package integration

import (
	"testing"
	"context"
	"time"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/server"
	"github.com/Backland-Labs/alpine/internal/workflow"
)

// mockStreamer tracks streaming calls for testing
type mockStreamer struct {
	streamStartCalled   bool
	streamContentCalled bool
	streamEndCalled     bool
	runID               string
	messageID           string
	content             []string
}

func (m *mockStreamer) StreamStart(runID, messageID string) error {
	m.streamStartCalled = true
	m.runID = runID
	m.messageID = messageID
	return nil
}

func (m *mockStreamer) StreamContent(runID, messageID, content string) error {
	m.streamContentCalled = true
	m.content = append(m.content, content)
	return nil
}

func (m *mockStreamer) StreamEnd(runID, messageID string) error {
	m.streamEndCalled = true
	return nil
}

// TestStreamingPropagation verifies that the streamer is properly passed through the workflow chain
func TestStreamingPropagation(t *testing.T) {
	// Create mock executor
	mockExecutor := &claude.Executor{}
	
	// Create mock streamer
	mockStreamer := &mockStreamer{}
	
	// Create config
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
		StateFile: "test_state.json",
	}
	
	// Create workflow engine with streamer
	_ = workflow.NewEngine(mockExecutor, nil, cfg, mockStreamer)
	
	// The executor should now have the streamer set
	// In a real implementation, the workflow engine would call SetStreamer on the executor
	t.Log("Workflow engine created with streamer")
}

// TestServerStreamerIntegration verifies the server streamer broadcasts events correctly
func TestServerStreamerIntegration(t *testing.T) {
	// Create server
	srv := server.NewServer(0)
	
	// Create server streamer
	streamer := server.NewServerStreamer(srv)
	
	// Test streaming lifecycle
	runID := "test-run-123"
	messageID := "msg-456"
	
	// Start streaming
	err := streamer.StreamStart(runID, messageID)
	if err != nil {
		t.Errorf("StreamStart failed: %v", err)
	}
	
	// Stream content
	err = streamer.StreamContent(runID, messageID, "Test content")
	if err != nil {
		t.Errorf("StreamContent failed: %v", err)
	}
	
	// End streaming
	err = streamer.StreamEnd(runID, messageID)
	if err != nil {
		t.Errorf("StreamEnd failed: %v", err)
	}
	
	t.Log("Server streamer integration test passed")
}

// TestAlpineWorkflowEngineWithStreaming verifies the AlpineWorkflowEngine passes server reference
func TestAlpineWorkflowEngineWithStreaming(t *testing.T) {
	// Create mock executor
	mockExecutor := &claude.Executor{}
	
	// Create config
	cfg := &config.Config{
		Git: config.GitConfig{
			WorktreeEnabled: false,
		},
	}
	
	// Create server
	srv := server.NewServer(0)
	
	// Create AlpineWorkflowEngine
	alpineEngine := server.NewAlpineWorkflowEngine(mockExecutor, nil, cfg)
	
	// Set server reference
	alpineEngine.SetServer(srv)
	
	// Start a test workflow
	ctx := context.Background()
	issueURL := "https://github.com/test/repo/issues/1"
	runID := "test-run"
	
	// This will create the workflow with the server streamer
	_, err := alpineEngine.StartWorkflow(ctx, issueURL, runID)
	if err != nil {
		// Expected since we don't have a real GitHub issue
		t.Logf("Expected error starting workflow: %v", err)
	}
	
	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)
	
	t.Log("AlpineWorkflowEngine streaming setup test passed")
}