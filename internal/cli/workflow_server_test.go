package cli

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServeFlagStartsServer verifies that the server is started when --serve is used.
// This test ensures that when the --serve flag is provided, the HTTP server
// starts in the background and is accessible on the specified port.
func TestServeFlagStartsServer(t *testing.T) {

	// Create test dependencies with a quick-completing workflow
	mockEngine := &mockWorkflowEngine{
		err: nil, // Workflow completes immediately
	}
	deps := &Dependencies{
		FileReader:     &mockFileReader{},
		ConfigLoader:   &mockConfigLoader{},
		WorkflowEngine: mockEngine,
	}

	// Use a specific test port to avoid conflicts
	testPort := 8765

	// Create a context with the serve flag set to true and test port
	ctx := context.Background()
	ctx = context.WithValue(ctx, serveKey, true)
	ctx = context.WithValue(ctx, portKey, testPort)

	// Create a context with cancel to control test lifecycle
	testCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Run workflow with serve flag in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- runWorkflowWithDependencies(testCtx, []string{"test task"}, false, false, false, deps)
	}()

	// Give the server time to start
	time.Sleep(200 * time.Millisecond)

	// Try to connect to the server - this SHOULD work when implemented
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/events", testPort))
	require.NoError(t, err, "Server should be accessible when --serve flag is used")

	// Verify it's an SSE endpoint
	contentType := resp.Header.Get("Content-Type")
	assert.Equal(t, "text/event-stream", contentType, "Should be an SSE endpoint")

	// Close the response before canceling context
	_ = resp.Body.Close()

	// Cancel the context to stop the server
	cancel()

	// Wait for workflow to complete
	select {
	case err := <-errChan:
		// Context cancellation might cause an error, which is OK
		if err != nil && err != context.Canceled {
			assert.NoError(t, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Workflow did not shut down in time")
	}
}

// TestWorkflowRunsConcurrentlyWithServer verifies that the main task is executed
// while the server is active. This ensures the server doesn't block the main workflow.
func TestWorkflowRunsConcurrentlyWithServer(t *testing.T) {
	// Create a custom mock that tracks execution
	mockEngine := &mockWorkflowEngine{}

	// Create test dependencies with our mock workflow engine
	deps := &Dependencies{
		FileReader:     &mockFileReader{},
		ConfigLoader:   &mockConfigLoader{},
		WorkflowEngine: mockEngine,
	}

	// Create a context with the serve flag
	ctx := context.Background()
	ctx = context.WithValue(ctx, serveKey, true)
	ctx = context.WithValue(ctx, portKey, 0) // Use port 0 for dynamic assignment

	// Run workflow
	err := runWorkflowWithDependencies(ctx, []string{"test task"}, false, false, false, deps)
	require.NoError(t, err)

	// Verify the workflow was executed by checking the mock was called
	assert.Equal(t, "test task", mockEngine.lastTaskDescription, "Workflow should have been executed with correct task")
	assert.True(t, mockEngine.lastGeneratePlan, "Generate plan should be true when no-plan=false")

	// TODO: Once implemented, also verify the server is running concurrently
}

// TestServerShutdownOnWorkflowComplete verifies that the server is gracefully
// shut down when the main workflow completes or is interrupted.
func TestServerShutdownOnWorkflowComplete(t *testing.T) {
	// Create test dependencies
	deps := &Dependencies{
		FileReader:     &mockFileReader{},
		ConfigLoader:   &mockConfigLoader{},
		WorkflowEngine: &mockWorkflowEngine{},
	}

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, serveKey, true)
	ctx = context.WithValue(ctx, portKey, 0) // Use dynamic port to avoid conflicts

	// Run workflow in a goroutine
	done := make(chan bool)
	go func() {
		err := runWorkflowWithDependencies(ctx, []string{"test task"}, false, false, false, deps)
		assert.NoError(t, err)
		done <- true
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the context to simulate interruption
	cancel()

	// Wait for workflow to complete
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Workflow did not complete in time")
	}

	// TODO: Once implemented, verify the server has stopped
	// For now, this test just ensures the workflow completes when context is cancelled
}
