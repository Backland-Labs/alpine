package output

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// TestAddToolLog_Empty tests adding a log to an empty buffer
func TestAddToolLog_Empty(t *testing.T) {
	// Create a printer with custom writer for testing
	var buf bytes.Buffer
	printer := NewPrinterWithWriters(&buf, &buf, false)

	// Add a single log message
	printer.AddToolLog("First tool log message")

	// Verify the tool log was added
	if len(printer.toolLogs) != 1 {
		t.Errorf("Expected 1 tool log, got %d", len(printer.toolLogs))
	}

	if printer.toolLogs[0] != "First tool log message" {
		t.Errorf("Expected 'First tool log message', got '%s'", printer.toolLogs[0])
	}
}

// TestAddToolLog_Append tests adding logs without reaching capacity
func TestAddToolLog_Append(t *testing.T) {
	// Create a printer with custom writer for testing
	var buf bytes.Buffer
	printer := NewPrinterWithWriters(&buf, &buf, false)

	// Add multiple logs (less than max capacity)
	messages := []string{
		"Log 1: Starting operation",
		"Log 2: Processing file",
		"Log 3: Operation complete",
	}

	for _, msg := range messages {
		printer.AddToolLog(msg)
	}

	// Verify all logs were added
	if len(printer.toolLogs) != 3 {
		t.Errorf("Expected 3 tool logs, got %d", len(printer.toolLogs))
	}

	// Verify logs are in correct order
	for i, msg := range messages {
		if printer.toolLogs[i] != msg {
			t.Errorf("Expected log[%d] to be '%s', got '%s'", i, msg, printer.toolLogs[i])
		}
	}
}

// TestAddToolLog_CircularBuffer tests that adding a new log when buffer is full correctly evicts the oldest log
func TestAddToolLog_CircularBuffer(t *testing.T) {
	// Create a printer with custom writer for testing
	var buf bytes.Buffer
	printer := NewPrinterWithWriters(&buf, &buf, false)

	// Add logs to fill the buffer (assuming maxToolLogs is 4)
	initialLogs := []string{
		"Log 1: First",
		"Log 2: Second",
		"Log 3: Third",
		"Log 4: Fourth",
	}

	for _, msg := range initialLogs {
		printer.AddToolLog(msg)
	}

	// Add one more log to trigger circular buffer behavior
	printer.AddToolLog("Log 5: Fifth")

	// Verify buffer is still at max capacity
	if len(printer.toolLogs) != 4 {
		t.Errorf("Expected 4 tool logs (max capacity), got %d", len(printer.toolLogs))
	}

	// Verify oldest log was evicted and new logs are in correct order
	expectedLogs := []string{
		"Log 2: Second",
		"Log 3: Third",
		"Log 4: Fourth",
		"Log 5: Fifth",
	}

	for i, expected := range expectedLogs {
		if printer.toolLogs[i] != expected {
			t.Errorf("Expected log[%d] to be '%s', got '%s'", i, expected, printer.toolLogs[i])
		}
	}
}

// TestAddToolLog_Concurrency tests adding logs from multiple goroutines concurrently to ensure thread safety
func TestAddToolLog_Concurrency(t *testing.T) {
	// Create a printer with custom writer for testing
	var buf bytes.Buffer
	printer := NewPrinterWithWriters(&buf, &buf, false)

	// Number of goroutines and logs per goroutine
	numGoroutines := 10
	logsPerGoroutine := 5

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines to add logs concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				msg := strings.Repeat("X", goroutineID+1) // Create unique message based on goroutine ID
				printer.AddToolLog(msg)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify we have exactly maxToolLogs entries (4)
	if len(printer.toolLogs) != 4 {
		t.Errorf("Expected 4 tool logs after concurrent access, got %d", len(printer.toolLogs))
	}

	// Verify no data corruption (all entries should be valid strings of X's)
	for i, log := range printer.toolLogs {
		if log == "" {
			t.Errorf("Found empty log at position %d, indicating data corruption", i)
		}
		// Check that the log contains only X characters
		for _, ch := range log {
			if ch != 'X' {
				t.Errorf("Found corrupted data in log[%d]: %s", i, log)
				break
			}
		}
	}
}

