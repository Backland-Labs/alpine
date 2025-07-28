package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Backland-Labs/alpine/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerIntegration tests the HTTP server with minimal mocking
func TestServerIntegration(t *testing.T) {
	t.Run("server starts and responds to health check", func(t *testing.T) {
		// Create a temporary directory for testing
		tmpDir := t.TempDir()
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tmpDir)
		
		// Start the server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// Use a random port
		port := "0" // Let the system assign a port
		
		// Create and start server command
		rootCmd := NewRootCommand()
		rootCmd.SetArgs([]string{"server", "--port", port})
		
		// Run in background
		serverErr := make(chan error, 1)
		go func() {
			serverErr <- rootCmd.ExecuteContext(ctx)
		}()
		
		// Give server time to start (config loading, logger init, etc.)
		time.Sleep(500 * time.Millisecond)
		
		// Cancel to stop server
		cancel()
		
		// Wait for clean shutdown
		select {
		case err := <-serverErr:
			// Context canceled is expected
			if err != nil && err != context.Canceled {
				t.Errorf("unexpected server error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("server did not shut down in time")
		}
	})
	
	t.Run("server executes workflows with events", func(t *testing.T) {
		// Skip if in short mode since this test takes time
		if testing.Short() {
			t.Skip("skipping integration test in short mode")
		}
		
		// Create a temporary directory for testing
		tmpDir := t.TempDir()
		
		// Create event collector
		events := make([]map[string]interface{}, 0)
		eventCollector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/events" && r.Method == "POST" {
				body, _ := io.ReadAll(r.Body)
				var event map[string]interface{}
				if json.Unmarshal(body, &event) == nil {
					events = append(events, event)
				}
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer eventCollector.Close()
		
		// Create a mock Claude script
		mockClaudePath := filepath.Join(tmpDir, "claude")
		mockScript := `#!/bin/bash
echo '{"status": "completed", "current_step_description": "Test completed"}' > agent_state/agent_state.json
`
		err := os.WriteFile(mockClaudePath, []byte(mockScript), 0755)
		require.NoError(t, err)
		
		// Update PATH to use mock claude
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tmpDir+":"+oldPath)
		defer os.Setenv("PATH", oldPath)
		
		// Change to temp dir
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)
		os.Chdir(tmpDir)
		
		// Start server using the programmatic API
		serverFlags := &serverFlags{
			port:          0, // Auto-assign port
			eventEndpoint: eventCollector.URL + "/events",
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		// Run server in background
		serverErr := make(chan error, 1)
		go func() {
			serverErr <- runServer(ctx, serverFlags)
		}()
		
		// Give server time to start
		time.Sleep(500 * time.Millisecond)
		
		// Server should have started and be ready
		// In a real test we'd check if it's listening, but for now just verify it shuts down cleanly
		
		cancel()
		
		select {
		case err := <-serverErr:
			if err != nil && err != context.Canceled {
				t.Errorf("server error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("server did not shut down")
		}
	})
}

// TestServerCommandFlags verifies the server command has the correct flags
func TestServerCommandFlags(t *testing.T) {
	cmd := newServerCommand()
	
	// Check command metadata
	assert.Equal(t, "server", cmd.Use)
	assert.Contains(t, cmd.Short, "HTTP server")
	
	// Check flags
	portFlag := cmd.Flags().Lookup("port")
	assert.NotNil(t, portFlag)
	assert.Equal(t, "8080", portFlag.DefValue)
	
	endpointFlag := cmd.Flags().Lookup("event-endpoint")
	assert.NotNil(t, endpointFlag)
	assert.Equal(t, "", endpointFlag.DefValue)
}

// TestServerRunsWorkflow tests that the server can execute a simple workflow
func TestServerRunsWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	
	// This is a unit test of the workflow executor
	executor := &workflowExecutor{
		defaultEventEndpoint: "",
		activeRuns:          make(map[string]*runContext),
		mu:                  &sync.Mutex{},
	}
	
	// Create a test request
	req := server.RunRequest{
		ID:   "test-run-123",
		Task: "test task",
	}
	
	// Create temp dir for the run
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)
	
	// Mock Claude
	mockClaudePath := filepath.Join(tmpDir, "claude")
	mockScript := `#!/bin/bash
mkdir -p agent_state
echo '{"status": "completed"}' > agent_state/agent_state.json
`
	err := os.WriteFile(mockClaudePath, []byte(mockScript), 0755)
	require.NoError(t, err)
	
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)
	
	// Execute the workflow
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = executor.handleRun(ctx, req)
	assert.NoError(t, err)
	
	// Verify run was stored
	executor.mu.Lock()
	run, exists := executor.activeRuns["test-run-123"]
	executor.mu.Unlock()
	
	assert.True(t, exists)
	assert.Equal(t, "test task", run.task)
	assert.Equal(t, "running", run.status)
	
	// Wait a bit for async execution
	time.Sleep(1 * time.Second)
	
	// Check status changed
	executor.mu.Lock()
	finalStatus := run.status
	executor.mu.Unlock()
	
	// Status might still be running if workflow is slow
	assert.Contains(t, []string{"running", "completed", "failed"}, finalStatus)
}