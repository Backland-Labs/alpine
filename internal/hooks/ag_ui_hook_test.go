package hooks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAgUIHookScriptParsesToolData tests that the hook script correctly parses tool data
func TestAgUIHookScriptParsesToolData(t *testing.T) {
	hookScript := filepath.Join("..", "..", "hooks", "alpine-ag-ui-emitter.rs")
	
	// Skip if script doesn't exist yet (RED phase)
	if _, err := os.Stat(hookScript); os.IsNotExist(err) {
		t.Skip("Hook script not yet implemented")
	}

	// Test data that Claude would send to the hook
	toolData := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/tmp/test.txt",
			"content":   "Hello, world!",
		},
		"tool_output": map[string]interface{}{
			"success": true,
		},
		"event": "tool_use",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Convert to JSON
	toolJSON, err := json.Marshal(toolData)
	if err != nil {
		t.Fatalf("Failed to marshal tool data: %v", err)
	}

	// Set up environment variables
	env := []string{
		"ALPINE_EVENTS_ENDPOINT=http://localhost:9999/events",
		"ALPINE_RUN_ID=test-run-123",
	}

	// Execute the hook script
	cmd := exec.Command(hookScript)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = bytes.NewReader(toolJSON)
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Hook script failed: %v\nStderr: %s", err, stderr.String())
	}

	// Verify the script logged the tool name
	if !strings.Contains(stderr.String(), "HOOK CALLED: tool=Write") {
		t.Errorf("Expected hook to log tool name, got stderr: %s", stderr.String())
	}
}

// TestAgUIHookScriptGeneratesValidEvents tests that the hook generates valid ag-ui events
func TestAgUIHookScriptGeneratesValidEvents(t *testing.T) {
	hookScript := filepath.Join("..", "..", "hooks", "alpine-ag-ui-emitter.rs")
	
	// Skip if script doesn't exist yet (RED phase)
	if _, err := os.Stat(hookScript); os.IsNotExist(err) {
		t.Skip("Hook script not yet implemented")
	}

	// Set up a test server to receive events
	var receivedEvent map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedEvent)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test data
	toolData := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": "echo 'test'",
		},
		"event": "tool_use",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	toolJSON, _ := json.Marshal(toolData)

	// Execute hook with test server endpoint
	cmd := exec.Command(hookScript)
	cmd.Env = append(os.Environ(),
		"ALPINE_EVENTS_ENDPOINT="+server.URL,
		"ALPINE_RUN_ID=test-run-456",
	)
	cmd.Stdin = bytes.NewReader(toolJSON)

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Hook script failed: %v", err)
	}

	// Give server time to receive the event
	time.Sleep(100 * time.Millisecond)

	// Verify event format matches ag-ui spec
	if receivedEvent == nil {
		t.Fatal("No event received by server")
	}

	if receivedEvent["type"] != "ToolCallStart" {
		t.Errorf("Expected event type 'ToolCallStart', got %v", receivedEvent["type"])
	}

	data, ok := receivedEvent["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Event data is not a map")
	}

	if data["toolCallName"] != "Bash" {
		t.Errorf("Expected toolCallName 'Bash', got %v", data["toolCallName"])
	}

	if data["runId"] != "test-run-456" {
		t.Errorf("Expected runId 'test-run-456', got %v", data["runId"])
	}

	if data["toolCallId"] == nil || data["toolCallId"] == "" {
		t.Error("Expected non-empty toolCallId")
	}
}

// TestAgUIHookScriptHandlesEndpointUnavailable tests graceful handling of endpoint failures
func TestAgUIHookScriptHandlesEndpointUnavailable(t *testing.T) {
	hookScript := filepath.Join("..", "..", "hooks", "alpine-ag-ui-emitter.rs")
	
	// Skip if script doesn't exist yet (RED phase)
	if _, err := os.Stat(hookScript); os.IsNotExist(err) {
		t.Skip("Hook script not yet implemented")
	}

	// Test data
	toolData := map[string]interface{}{
		"tool_name": "Read",
		"tool_input": map[string]interface{}{
			"file_path": "/tmp/test.txt",
		},
		"event": "tool_use",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	toolJSON, _ := json.Marshal(toolData)

	// Execute hook with unreachable endpoint
	cmd := exec.Command(hookScript)
	cmd.Env = append(os.Environ(),
		"ALPINE_EVENTS_ENDPOINT=http://localhost:1/unreachable",
		"ALPINE_RUN_ID=test-run-789",
	)
	cmd.Stdin = bytes.NewReader(toolJSON)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Should not fail even if endpoint is unavailable
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Hook script should not fail on endpoint unavailable: %v", err)
	}

	// Should still log that it was called
	if !strings.Contains(stderr.String(), "HOOK CALLED: tool=Read") {
		t.Errorf("Expected hook to log even with unavailable endpoint, got stderr: %s", stderr.String())
	}
}

// TestAgUIHookScriptGeneratesToolCallEndEvents tests that the hook generates ToolCallEnd events
func TestAgUIHookScriptGeneratesToolCallEndEvents(t *testing.T) {
	hookScript := filepath.Join("..", "..", "hooks", "alpine-ag-ui-emitter.rs")
	
	// Skip if script doesn't exist yet (RED phase)
	if _, err := os.Stat(hookScript); os.IsNotExist(err) {
		t.Skip("Hook script not yet implemented")
	}

	// Set up a test server to receive events
	var receivedEvents []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event map[string]interface{}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &event)
		receivedEvents = append(receivedEvents, event)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test data with tool output (indicates tool completion)
	toolData := map[string]interface{}{
		"tool_name": "Write",
		"tool_input": map[string]interface{}{
			"file_path": "/tmp/test.txt",
			"content":   "Test content",
		},
		"tool_output": map[string]interface{}{
			"success": true,
			"message": "File written successfully",
		},
		"event": "tool_use",
		"timestamp": time.Now().Format(time.RFC3339),
		"tool_call_id": "call-123", // Claude may provide this
	}

	toolJSON, _ := json.Marshal(toolData)

	// Execute hook
	cmd := exec.Command(hookScript)
	cmd.Env = append(os.Environ(),
		"ALPINE_EVENTS_ENDPOINT="+server.URL,
		"ALPINE_RUN_ID=test-run-end",
	)
	cmd.Stdin = bytes.NewReader(toolJSON)

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Hook script failed: %v", err)
	}

	// Give server time to receive events
	time.Sleep(100 * time.Millisecond)

	// Should receive both ToolCallStart and ToolCallEnd events
	if len(receivedEvents) != 2 {
		t.Fatalf("Expected 2 events (Start and End), got %d", len(receivedEvents))
	}

	// Verify ToolCallEnd event
	endEvent := receivedEvents[1]
	if endEvent["type"] != "ToolCallEnd" {
		t.Errorf("Expected second event type 'ToolCallEnd', got %v", endEvent["type"])
	}

	data, _ := endEvent["data"].(map[string]interface{})
	if data["toolCallName"] != "Write" {
		t.Errorf("Expected toolCallName 'Write' in end event, got %v", data["toolCallName"])
	}
}