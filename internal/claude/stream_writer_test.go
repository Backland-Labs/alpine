package claude

import (
	"strings"
	"testing"
)

// TestStreamWriterFunctionality tests the StreamWriter in isolation
func TestStreamWriterFunctionality(t *testing.T) {
	streamer := &mockStreamer{}
	runID := "run-test"
	messageID := "msg-test"

	sw := NewStreamWriter(streamer, runID, messageID)

	// Test writing complete lines
	testData := []byte("Hello World\nSecond Line\n")
	n, err := sw.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Check streamed content
	if len(streamer.contentCalls) != 2 {
		t.Errorf("Expected 2 content calls, got %d", len(streamer.contentCalls))
	} else {
		if streamer.contentCalls[0].content != "Hello World\n" {
			t.Errorf("First line: expected %q, got %q", "Hello World\n", streamer.contentCalls[0].content)
		}
		if streamer.contentCalls[1].content != "Second Line\n" {
			t.Errorf("Second line: expected %q, got %q", "Second Line\n", streamer.contentCalls[1].content)
		}
	}

	// Test partial line buffering
	streamer.contentCalls = nil // Reset
	sw.Write([]byte("Partial"))
	if len(streamer.contentCalls) != 0 {
		t.Error("Partial line should not be streamed immediately")
	}

	// Complete the line
	sw.Write([]byte(" Line\n"))
	if len(streamer.contentCalls) != 1 {
		t.Errorf("Expected 1 content call after completing line, got %d", len(streamer.contentCalls))
	} else {
		if streamer.contentCalls[0].content != "Partial Line\n" {
			t.Errorf("Expected %q, got %q", "Partial Line\n", streamer.contentCalls[0].content)
		}
	}

	// Test flush
	streamer.contentCalls = nil
	sw.Write([]byte("No newline"))
	sw.Flush()
	if len(streamer.contentCalls) != 1 {
		t.Errorf("Expected 1 content call after flush, got %d", len(streamer.contentCalls))
	} else {
		if streamer.contentCalls[0].content != "No newline" {
			t.Errorf("Expected %q, got %q", "No newline", streamer.contentCalls[0].content)
		}
	}
}

// TestMultiWriterWithFlush tests the custom multi-writer
func TestMultiWriterWithFlush(t *testing.T) {
	var buf1 strings.Builder
	sw := &mockFlushWriter{}
	
	mw := newMultiWriterWithFlush(&buf1, sw)
	
	// Write some data
	data := []byte("test data")
	n, err := mw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
	
	// Check both writers received the data
	if buf1.String() != "test data" {
		t.Errorf("buf1: expected %q, got %q", "test data", buf1.String())
	}
	if sw.String() != "test data" {
		t.Errorf("sw: expected %q, got %q", "test data", sw.String())
	}
	
	// Test flush
	sw.flushed = false
	err = mw.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
	if !sw.flushed {
		t.Error("Expected flush to be called on flushable writer")
	}
}

type mockFlushWriter struct {
	builder strings.Builder
	flushed bool
}

func (m *mockFlushWriter) Write(p []byte) (n int, err error) {
	return m.builder.Write(p)
}

func (m *mockFlushWriter) String() string {
	return m.builder.String()
}

func (m *mockFlushWriter) Flush() error {
	m.flushed = true
	return nil
}

// TestStreamWriterErrorHandling tests error scenarios
func TestStreamWriterErrorHandling(t *testing.T) {
	// Test with a failing streamer
	testErr := testStreamError{}
	failingStreamer := &mockStreamer{
		streamErr: testErr,
	}
	
	sw := NewStreamWriter(failingStreamer, "run-fail", "msg-fail")
	
	// Write should succeed even if streaming fails (errors are logged)
	data := []byte("Test line\n")
	n, err := sw.Write(data)
	if err != nil {
		t.Errorf("Write should not return streaming errors: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
	
	// Verify streaming was attempted
	if len(failingStreamer.contentCalls) != 1 {
		t.Errorf("Expected 1 content call, got %d", len(failingStreamer.contentCalls))
	}
}

// testStreamError is a test error for simulating streaming failures
type testStreamError struct{}

func (e testStreamError) Error() string {
	return "test stream failure"
}