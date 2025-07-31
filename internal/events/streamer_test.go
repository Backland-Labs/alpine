package events

import (
	"errors"
	"testing"
)

// TestStreamerInterface verifies that the Streamer interface methods are called correctly
// and follow the expected lifecycle pattern (Start -> Content -> End).
// This test ensures proper contract definition for streaming operations.
func TestStreamerInterface(t *testing.T) {
	t.Run("Complete streaming lifecycle", func(t *testing.T) {
		// Given a mock streamer that tracks method calls
		mock := &mockStreamer{
			calls: make([]string, 0),
		}

		// When we perform a complete streaming lifecycle
		runID := "run-123"
		messageID := "msg-456"

		err := mock.StreamStart(runID, messageID)
		if err != nil {
			t.Fatalf("StreamStart failed: %v", err)
		}

		err = mock.StreamContent(runID, messageID, "Hello, world!")
		if err != nil {
			t.Fatalf("StreamContent failed: %v", err)
		}

		err = mock.StreamContent(runID, messageID, "Second chunk")
		if err != nil {
			t.Fatalf("StreamContent failed: %v", err)
		}

		err = mock.StreamEnd(runID, messageID)
		if err != nil {
			t.Fatalf("StreamEnd failed: %v", err)
		}

		// Then the methods should be called in the correct order
		expectedCalls := []string{
			"StreamStart:run-123:msg-456",
			"StreamContent:run-123:msg-456:Hello, world!",
			"StreamContent:run-123:msg-456:Second chunk",
			"StreamEnd:run-123:msg-456",
		}

		if len(mock.calls) != len(expectedCalls) {
			t.Fatalf("Expected %d calls, got %d", len(expectedCalls), len(mock.calls))
		}

		for i, expected := range expectedCalls {
			if mock.calls[i] != expected {
				t.Errorf("Call %d: expected %q, got %q", i, expected, mock.calls[i])
			}
		}
	})

	t.Run("Error handling in StreamStart", func(t *testing.T) {
		// Given a mock streamer that fails on StreamStart
		mock := &mockStreamer{
			failOnStart: true,
		}

		// When we try to start streaming
		err := mock.StreamStart("run-123", "msg-456")

		// Then it should return an error
		if err == nil {
			t.Error("Expected error from StreamStart, got nil")
		}

		if err.Error() != "failed to start stream" {
			t.Errorf("Expected specific error message, got: %v", err)
		}
	})

	t.Run("Error handling in StreamContent", func(t *testing.T) {
		// Given a mock streamer that fails on StreamContent
		mock := &mockStreamer{
			failOnContent: true,
		}

		// When we try to stream content
		err := mock.StreamContent("run-123", "msg-456", "content")

		// Then it should return an error
		if err == nil {
			t.Error("Expected error from StreamContent, got nil")
		}
	})

	t.Run("Error handling in StreamEnd", func(t *testing.T) {
		// Given a mock streamer that fails on StreamEnd
		mock := &mockStreamer{
			failOnEnd: true,
		}

		// When we try to end streaming
		err := mock.StreamEnd("run-123", "msg-456")

		// Then it should return an error
		if err == nil {
			t.Error("Expected error from StreamEnd, got nil")
		}
	})
}

// TestNoOpStreamer verifies that the NoOpStreamer implementation
// successfully implements the Streamer interface without performing any actions.
// This ensures backward compatibility for non-streaming mode.
func TestNoOpStreamer(t *testing.T) {
	t.Run("NoOpStreamer returns nil for all operations", func(t *testing.T) {
		// Given a NoOpStreamer
		noop := &NoOpStreamer{}

		// When we call all interface methods
		err1 := noop.StreamStart("run-123", "msg-456")
		err2 := noop.StreamContent("run-123", "msg-456", "content")
		err3 := noop.StreamEnd("run-123", "msg-456")

		// Then all methods should return nil (no error)
		if err1 != nil {
			t.Errorf("StreamStart returned error: %v", err1)
		}
		if err2 != nil {
			t.Errorf("StreamContent returned error: %v", err2)
		}
		if err3 != nil {
			t.Errorf("StreamEnd returned error: %v", err3)
		}
	})

	t.Run("NoOpStreamer can be used as Streamer interface", func(t *testing.T) {
		// Given a NoOpStreamer
		var streamer Streamer = &NoOpStreamer{}

		// When we use it through the interface
		err := performStreaming(streamer, "run-123", "msg-456", "test content")

		// Then it should work without errors
		if err != nil {
			t.Errorf("Streaming with NoOpStreamer failed: %v", err)
		}
	})
}

// Helper function to test interface usage
func performStreaming(s Streamer, runID, messageID, content string) error {
	if err := s.StreamStart(runID, messageID); err != nil {
		return err
	}

	if err := s.StreamContent(runID, messageID, content); err != nil {
		return err
	}

	return s.StreamEnd(runID, messageID)
}

// mockStreamer is a test implementation of the Streamer interface
type mockStreamer struct {
	calls         []string
	failOnStart   bool
	failOnContent bool
	failOnEnd     bool
}

func (m *mockStreamer) StreamStart(runID, messageID string) error {
	if m.failOnStart {
		return errors.New("failed to start stream")
	}
	m.calls = append(m.calls, "StreamStart:"+runID+":"+messageID)
	return nil
}

func (m *mockStreamer) StreamContent(runID, messageID, content string) error {
	if m.failOnContent {
		return errors.New("failed to stream content")
	}
	m.calls = append(m.calls, "StreamContent:"+runID+":"+messageID+":"+content)
	return nil
}

func (m *mockStreamer) StreamEnd(runID, messageID string) error {
	if m.failOnEnd {
		return errors.New("failed to end stream")
	}
	m.calls = append(m.calls, "StreamEnd:"+runID+":"+messageID)
	return nil
}
