package workflow

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Backland-Labs/alpine/internal/claude"
	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/core"
	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/gitx/mock"
)

// MockStreamer captures streaming calls for testing
type MockStreamer struct {
	StartCalls   []StreamCall
	ContentCalls []StreamCall
	EndCalls     []StreamCall
}

type StreamCall struct {
	RunID     string
	MessageID string
	Content   string
}

func (m *MockStreamer) StreamStart(runID, messageID string) error {
	m.StartCalls = append(m.StartCalls, StreamCall{RunID: runID, MessageID: messageID})
	return nil
}

func (m *MockStreamer) StreamContent(runID, messageID, content string) error {
	m.ContentCalls = append(m.ContentCalls, StreamCall{RunID: runID, MessageID: messageID, Content: content})
	return nil
}

func (m *MockStreamer) StreamEnd(runID, messageID string) error {
	m.EndCalls = append(m.EndCalls, StreamCall{RunID: runID, MessageID: messageID})
	return nil
}

// MockExecutorWithStreamer is a mock executor that records streamer and runID calls
type MockExecutorWithStreamer struct {
	Streamer     events.Streamer
	RunID        string
	ExecuteCalls []claude.ExecuteConfig
	ExecuteFn    func(ctx context.Context, config claude.ExecuteConfig) (string, error)
}

func (m *MockExecutorWithStreamer) Execute(ctx context.Context, config claude.ExecuteConfig) (string, error) {
	m.ExecuteCalls = append(m.ExecuteCalls, config)
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, config)
	}
	return "mock output", nil
}

func (m *MockExecutorWithStreamer) SetStreamer(streamer events.Streamer) {
	m.Streamer = streamer
}

func (m *MockExecutorWithStreamer) SetRunID(runID string) {
	m.RunID = runID
}

// TestWorkflowStreamingIntegration tests that streamer is passed correctly to executor
func TestWorkflowStreamingIntegration(t *testing.T) {
	// Arrange
	executed := false
	mockExecutor := &MockExecutorWithStreamer{
		ExecuteFn: func(ctx context.Context, config claude.ExecuteConfig) (string, error) {
			executed = true
			// Update state to complete workflow
			state := &core.State{
				CurrentStepDescription: "Completed",
				NextStepPrompt:         "",
				Status:                 "completed",
			}
			return "", state.Save(config.StateFile)
		},
	}
	mockStreamer := &MockStreamer{}
	mockWtMgr := &mock.WorktreeManager{}

	// Create temp dir for state file
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "agent_state", "agent_state.json")

	cfg := &config.Config{
		StateFile: stateFile,
		Git:       config.GitConfig{WorktreeEnabled: false},
	}

	// Create engine with streamer
	engine := NewEngine(mockExecutor, mockWtMgr, cfg, mockStreamer)

	// Act - run the workflow to trigger executor
	ctx := context.Background()
	err := engine.Run(ctx, "test task", false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert - executor should have been called
	if !executed {
		t.Fatal("Expected executor to be called")
	}

	// Assert - executor should have streamer and runID set
	if mockExecutor.Streamer != mockStreamer {
		t.Errorf("Expected executor to have streamer set, got %v", mockExecutor.Streamer)
	}

	if mockExecutor.RunID == "" {
		t.Error("Expected executor to have runID set, but it was empty")
	}

	// Verify runID is a valid UUID format
	if len(mockExecutor.RunID) != 36 { // UUID format: 8-4-4-4-12
		t.Errorf("Expected runID to be UUID format, got %s", mockExecutor.RunID)
	}
}

// TestWorkflowWithoutStreamer tests backward compatibility when no streamer is provided
func TestWorkflowWithoutStreamer(t *testing.T) {
	// Arrange
	mockExecutor := &MockExecutorWithStreamer{}
	mockWtMgr := &mock.WorktreeManager{}
	cfg := &config.Config{
		StateFile: "test_state.json",
		Git:       config.GitConfig{WorktreeEnabled: false},
	}

	// Create engine without streamer (nil)
	engine := NewEngine(mockExecutor, mockWtMgr, cfg, nil)

	// Act
	ctx := context.Background()
	_ = engine.initializeWorkflow(ctx, "test task", false)

	// Assert - executor should not have streamer set
	if mockExecutor.Streamer != nil {
		t.Errorf("Expected executor to have nil streamer, got %v", mockExecutor.Streamer)
	}

	// RunID might still be set for other purposes, that's OK
}

// TestWorkflowRunIDCorrelation tests that run ID is correctly correlated throughout workflow
func TestWorkflowRunIDCorrelation(t *testing.T) {
	// Arrange
	mockExecutor := &MockExecutorWithStreamer{}
	mockStreamer := &MockStreamer{}
	mockWtMgr := &mock.WorktreeManager{}
	cfg := &config.Config{
		StateFile: "test_state.json",
		Git:       config.GitConfig{WorktreeEnabled: false},
	}

	engine := NewEngine(mockExecutor, mockWtMgr, cfg, mockStreamer)

	// Act - run workflow initialization
	ctx := context.Background()
	_ = engine.initializeWorkflow(ctx, "test task", false)

	// Get the runID that was set
	runID := mockExecutor.RunID

	// Assert
	if runID == "" {
		t.Fatal("Expected runID to be set")
	}

	// Verify engine stores the same runID
	if engine.runID != runID {
		t.Errorf("Expected engine runID %s to match executor runID %s", engine.runID, runID)
	}
}

// TestEngineConstructorWithStreamer tests the updated NewEngine constructor
func TestEngineConstructorWithStreamer(t *testing.T) {
	// Test with streamer
	mockExecutor := &MockExecutorWithStreamer{}
	mockWtMgr := &mock.WorktreeManager{}
	cfg := &config.Config{}
	streamer := &MockStreamer{}

	engine := NewEngine(mockExecutor, mockWtMgr, cfg, streamer)

	if engine.streamer != streamer {
		t.Errorf("Expected engine to have streamer set, got %v", engine.streamer)
	}
}

// TestEngineConstructorWithoutStreamer tests backward compatibility
func TestEngineConstructorWithoutStreamer(t *testing.T) {
	// Test without streamer (nil)
	mockExecutor := &MockExecutorWithStreamer{}
	mockWtMgr := &mock.WorktreeManager{}
	cfg := &config.Config{}

	engine := NewEngine(mockExecutor, mockWtMgr, cfg, nil)

	if engine.streamer != nil {
		t.Errorf("Expected engine to have nil streamer, got %v", engine.streamer)
	}
}

// TestStreamerPropagationDuringExecution tests that streamer is propagated during execution
func TestStreamerPropagationDuringExecution(t *testing.T) {
	// This test verifies that the streamer is properly passed to the executor
	// during the workflow execution loop, not just at initialization

	// Arrange
	mockExecutor := &MockExecutorWithStreamer{}
	mockStreamer := &MockStreamer{}
	mockWtMgr := &mock.WorktreeManager{}
	cfg := &config.Config{
		StateFile: "test_state.json",
		Git:       config.GitConfig{WorktreeEnabled: false},
	}

	engine := NewEngine(mockExecutor, mockWtMgr, cfg, mockStreamer)

	// The actual propagation logic will be tested once implementation is done
	// For now, this test documents the expected behavior

	// Act & Assert will be completed during GREEN phase
	_ = engine // Silence unused variable warning
}

// TestMultipleClaudeExecutions tests that each Claude execution gets proper streaming
func TestMultipleClaudeExecutions(t *testing.T) {
	// This test ensures that when Claude is called multiple times in a workflow,
	// each execution maintains proper streaming with unique message IDs

	// Test implementation will be completed during GREEN phase
	// This placeholder documents the expected behavior
}

// TestErrorHandlingWithStreamer tests that streaming errors don't fail workflow
func TestErrorHandlingWithStreamer(t *testing.T) {
	// Arrange
	mockExecutor := &MockExecutorWithStreamer{}
	failingStreamer := &FailingStreamer{
		err: fmt.Errorf("streaming failed"),
	}
	mockWtMgr := &mock.WorktreeManager{}
	cfg := &config.Config{
		StateFile: "test_state.json",
		Git:       config.GitConfig{WorktreeEnabled: false},
	}

	engine := NewEngine(mockExecutor, mockWtMgr, cfg, failingStreamer)

	// Act
	ctx := context.Background()
	err := engine.initializeWorkflow(ctx, "test task", false)

	// Assert - workflow should continue despite streaming errors
	if err != nil {
		t.Errorf("Expected workflow to continue despite streaming errors, got %v", err)
	}
}

// FailingStreamer is a mock that always returns errors
type FailingStreamer struct {
	err error
}

func (f *FailingStreamer) StreamStart(runID, messageID string) error {
	return f.err
}

func (f *FailingStreamer) StreamContent(runID, messageID, content string) error {
	return f.err
}

func (f *FailingStreamer) StreamEnd(runID, messageID string) error {
	return f.err
}
