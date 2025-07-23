package claude

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTodoMonitor_Start(t *testing.T) {
	// Test that TodoMonitor properly monitors file changes and sends updates
	t.Run("detects file changes and sends updates", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "todo.txt")

		monitor := NewTodoMonitor(todoFile)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start monitoring in background
		go monitor.Start(ctx)

		// Write initial task
		err := os.WriteFile(todoFile, []byte("Task 1: Initial setup"), 0644)
		if err != nil {
			t.Fatalf("Failed to write initial task: %v", err)
		}

		// Wait for update
		select {
		case update := <-monitor.Updates():
			if update != "Task 1: Initial setup" {
				t.Errorf("Expected 'Task 1: Initial setup', got '%s'", update)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for initial update")
		}

		// Update task
		err = os.WriteFile(todoFile, []byte("Task 2: Processing data"), 0644)
		if err != nil {
			t.Fatalf("Failed to write updated task: %v", err)
		}

		// Wait for update
		select {
		case update := <-monitor.Updates():
			if update != "Task 2: Processing data" {
				t.Errorf("Expected 'Task 2: Processing data', got '%s'", update)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for task update")
		}
	})

	t.Run("ignores duplicate updates", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "todo.txt")

		monitor := NewTodoMonitor(todoFile)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Write initial task
		err := os.WriteFile(todoFile, []byte("Same task"), 0644)
		if err != nil {
			t.Fatalf("Failed to write task: %v", err)
		}

		// Start monitoring in background
		go monitor.Start(ctx)

		// Wait for initial update
		select {
		case <-monitor.Updates():
			// Expected
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for initial update")
		}

		// Write same content again
		err = os.WriteFile(todoFile, []byte("Same task"), 0644)
		if err != nil {
			t.Fatalf("Failed to rewrite task: %v", err)
		}

		// Should not receive duplicate update
		select {
		case update := <-monitor.Updates():
			t.Errorf("Unexpected duplicate update: %s", update)
		case <-time.After(1 * time.Second):
			// Expected - no update
		}
	})

	t.Run("handles missing file gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "nonexistent.txt")

		monitor := NewTodoMonitor(todoFile)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start monitoring in background
		go monitor.Start(ctx)

		// Should not crash or send updates for missing file
		select {
		case update := <-monitor.Updates():
			t.Errorf("Unexpected update from missing file: %s", update)
		case <-time.After(1 * time.Second):
			// Expected - no update
		}

		// Now create the file
		err := os.WriteFile(todoFile, []byte("New task"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		// Should detect the new file
		select {
		case update := <-monitor.Updates():
			if update != "New task" {
				t.Errorf("Expected 'New task', got '%s'", update)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for new file update")
		}
	})

	t.Run("stops monitoring on context cancellation", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "todo.txt")

		monitor := NewTodoMonitor(todoFile)
		ctx, cancel := context.WithCancel(context.Background())

		// Start monitoring in background
		monitorDone := make(chan struct{})
		go func() {
			monitor.Start(ctx)
			close(monitorDone)
		}()

		// Cancel context
		cancel()

		// Monitor should stop
		select {
		case <-monitorDone:
			// Expected
		case <-time.After(2 * time.Second):
			t.Error("Monitor did not stop after context cancellation")
		}
	})
}

func TestTodoMonitor_readCurrentTask(t *testing.T) {
	// Test the readCurrentTask method
	t.Run("reads and trims file content", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "todo.txt")

		monitor := NewTodoMonitor(todoFile)

		// Write task with whitespace
		err := os.WriteFile(todoFile, []byte("  Task with spaces  \n"), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		task := monitor.readCurrentTask()
		if task != "Task with spaces" {
			t.Errorf("Expected 'Task with spaces', got '%s'", task)
		}
	})

	t.Run("returns empty string for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "missing.txt")

		monitor := NewTodoMonitor(todoFile)

		task := monitor.readCurrentTask()
		if task != "" {
			t.Errorf("Expected empty string, got '%s'", task)
		}
	})

	t.Run("returns empty string for empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		todoFile := filepath.Join(tmpDir, "empty.txt")

		// Create empty file
		err := os.WriteFile(todoFile, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		monitor := NewTodoMonitor(todoFile)

		task := monitor.readCurrentTask()
		if task != "" {
			t.Errorf("Expected empty string, got '%s'", task)
		}
	})
}

func TestNewTodoMonitor(t *testing.T) {
	// Test TodoMonitor constructor
	t.Run("creates monitor with proper initialization", func(t *testing.T) {
		filePath := "/tmp/test.txt"
		monitor := NewTodoMonitor(filePath)

		if monitor.filePath != filePath {
			t.Errorf("Expected filePath '%s', got '%s'", filePath, monitor.filePath)
		}

		if monitor.lastTask != "" {
			t.Errorf("Expected empty lastTask, got '%s'", monitor.lastTask)
		}

		// Test that updates channel is created and buffered
		select {
		case monitor.updates <- "test":
			// Should not block since it's buffered
			<-monitor.updates // Clean up
		default:
			t.Error("Updates channel appears to be unbuffered or not created")
		}
	})
}
