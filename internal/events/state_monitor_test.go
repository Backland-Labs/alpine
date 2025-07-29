package events

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/core"
	"fmt"
)

func TestStateMonitor_DetectsFileChanges(t *testing.T) {
	// This test verifies that the StateMonitor detects when agent_state.json changes
	// and emits a StateSnapshot event with the updated state content
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "agent_state.json")

	// Create initial state file
	initialState := core.State{
		CurrentStepDescription: "Initial step",
		NextStepPrompt:        "/continue",
		Status:                "running",
	}
	writeStateFile(t, stateFile, initialState)

	// Create mock emitter to capture events
	mockEmitter := NewMockEmitter()

	// Create and start state monitor
	monitor := NewStateMonitor(stateFile, mockEmitter, "test-run-123")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}

	// Give monitor time to start watching
	time.Sleep(100 * time.Millisecond)

	// Update the state file
	updatedState := core.State{
		CurrentStepDescription: "Updated step",
		NextStepPrompt:        "/verify",
		Status:                "completed",
	}
	writeStateFile(t, stateFile, updatedState)

	// Wait for event to be emitted
	time.Sleep(200 * time.Millisecond)

	// Verify StateSnapshot events were emitted (initial + update)
	events := mockEmitter.GetStateSnapshots()
	if len(events) < 2 {
		t.Fatalf("Expected at least 2 StateSnapshot events, got %d", len(events))
	}

	// Check the last event (the update)
	snapshot := events[len(events)-1]
	if snapshot.RunID != "test-run-123" {
		t.Errorf("Expected RunID 'test-run-123', got %s", snapshot.RunID)
	}

	// Verify snapshot contains updated state
	stateData, ok := snapshot.Snapshot.(core.State)
	if !ok {
		t.Fatalf("Snapshot is not core.State type")
	}

	if stateData.CurrentStepDescription != "Updated step" {
		t.Errorf("Expected CurrentStepDescription 'Updated step', got %s", stateData.CurrentStepDescription)
	}
	if stateData.Status != "completed" {
		t.Errorf("Expected Status 'completed', got %s", stateData.Status)
	}
}

func TestStateMonitor_EmitsCorrectEventFormat(t *testing.T) {
	// This test verifies that StateSnapshot events follow the ag-ui protocol format
	// with proper type and data structure
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "agent_state.json")

	// Create state file
	testState := core.State{
		CurrentStepDescription: "Test step",
		NextStepPrompt:        "/test",
		Status:                "running",
	}
	writeStateFile(t, stateFile, testState)

	// Create mock emitter
	mockEmitter := NewMockEmitter()

	// Create and start monitor
	monitor := NewStateMonitor(stateFile, mockEmitter, "format-test-run")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Trigger a change
	testState.Status = "completed"
	writeStateFile(t, stateFile, testState)
	time.Sleep(200 * time.Millisecond)

	// Get the raw event data
	rawEvents := mockEmitter.GetRawEvents()
	if len(rawEvents) == 0 {
		t.Fatal("No events emitted")
	}

	// Verify event structure matches ag-ui spec
	event := rawEvents[len(rawEvents)-1]
	if event["type"] != "StateSnapshot" {
		t.Errorf("Expected event type 'StateSnapshot', got %v", event["type"])
	}

	data, ok := event["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Event data is not a map")
	}

	if data["runId"] != "format-test-run" {
		t.Errorf("Expected runId 'format-test-run', got %v", data["runId"])
	}

	if _, exists := data["snapshot"]; !exists {
		t.Error("Event data missing 'snapshot' field")
	}
}

func TestStateMonitor_HandlesMissingFile(t *testing.T) {
	// This test verifies that the monitor handles gracefully when agent_state.json
	// doesn't exist initially and starts monitoring when it's created
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "agent_state.json")

	mockEmitter := NewMockEmitter()
	monitor := NewStateMonitor(stateFile, mockEmitter, "missing-file-run")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitoring non-existent file
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Monitor should start even with missing file: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Create the file
	newState := core.State{
		CurrentStepDescription: "Created",
		NextStepPrompt:        "/new",
		Status:                "running",
	}
	writeStateFile(t, stateFile, newState)

	// Wait for detection
	time.Sleep(300 * time.Millisecond)

	// Should emit event for newly created file
	events := mockEmitter.GetStateSnapshots()
	if len(events) != 1 {
		t.Fatalf("Expected 1 event after file creation, got %d", len(events))
	}
}

func TestStateMonitor_StopsOnContextCancel(t *testing.T) {
	// This test verifies that the monitor stops watching when context is cancelled
	// and doesn't emit events after stopping
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "agent_state.json")
	
	writeStateFile(t, stateFile, core.State{
		CurrentStepDescription: "Initial",
		NextStepPrompt:        "/start",
		Status:                "running",
	})

	mockEmitter := NewMockEmitter()
	monitor := NewStateMonitor(stateFile, mockEmitter, "cancel-test")

	ctx, cancel := context.WithCancel(context.Background())
	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop monitoring
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Clear any existing events
	mockEmitter.Reset()

	// Update file after stopping
	writeStateFile(t, stateFile, core.State{
		CurrentStepDescription: "Should not detect",
		NextStepPrompt:        "/ignored",
		Status:                "completed",
	})

	time.Sleep(200 * time.Millisecond)

	// Should not emit any new events
	events := mockEmitter.GetStateSnapshots()
	if len(events) != 0 {
		t.Errorf("Expected no events after stopping, got %d", len(events))
	}
}

func TestStateMonitor_HandlesRapidChanges(t *testing.T) {
	// This test verifies that the monitor can handle rapid successive changes
	// to the state file and emits events for each change
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "agent_state.json")

	mockEmitter := NewMockEmitter()
	monitor := NewStateMonitor(stateFile, mockEmitter, "rapid-changes")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := monitor.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start monitor: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Make rapid changes
	for i := 0; i < 5; i++ {
		writeStateFile(t, stateFile, core.State{
			CurrentStepDescription: fmt.Sprintf("Step %d", i),
			NextStepPrompt:        "/continue",
			Status:                "running",
		})
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for all events
	time.Sleep(500 * time.Millisecond)

	events := mockEmitter.GetStateSnapshots()
	if len(events) < 3 {
		t.Errorf("Expected at least 3 events from rapid changes, got %d", len(events))
	}
}

// Helper function to write state file
func writeStateFile(t *testing.T, path string, state core.State) {
	t.Helper()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal state: %v", err)
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write state file: %v", err)
	}
}