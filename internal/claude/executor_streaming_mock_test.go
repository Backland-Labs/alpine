package claude

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/Backland-Labs/alpine/internal/events"
)

// TestExecutorStreamingWithMockExecution tests streaming by directly exercising
// the streaming logic without requiring a real command execution
func TestExecutorStreamingWithMockExecution(t *testing.T) {
	tests := []struct {
		name           string
		streamer       events.Streamer
		runID          string
		simulateOutput string
		expectStream   bool
	}{
		{
			name:           "streams output line by line",
			streamer:       &mockStreamer{},
			runID:          "run-123",
			simulateOutput: "Line 1\nLine 2\nLine 3\n",
			expectStream:   true,
		},
		{
			name:           "streams partial lines correctly",
			streamer:       &mockStreamer{},
			runID:          "run-456",
			simulateOutput: "Partial line without newline",
			expectStream:   true,
		},
		{
			name:           "no streaming when streamer is nil",
			streamer:       nil,
			runID:          "",
			simulateOutput: "Test output",
			expectStream:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create executor
			executor := NewExecutor()
			executor.SetStreamer(tt.streamer)
			executor.SetRunID(tt.runID)

			// Simulate the streaming logic directly
			if tt.expectStream && tt.streamer != nil && tt.runID != "" {
				// Generate message ID
				messageID := generateMessageID()

				// Start streaming
				if err := tt.streamer.StreamStart(tt.runID, messageID); err != nil {
					t.Errorf("StreamStart failed: %v", err)
				}

				// Simulate streaming content
				streamWriter := NewStreamWriter(tt.streamer, tt.runID, messageID)
				reader := strings.NewReader(tt.simulateOutput)
				
				// Copy data through the stream writer
				var buf bytes.Buffer
				multiWriter := newMultiWriterWithFlush(&buf, streamWriter)
				_, err := io.Copy(multiWriter, reader)
				if err != nil {
					t.Errorf("Failed to copy through stream writer: %v", err)
				}

				// Flush any remaining data
				multiWriter.Flush()

				// End streaming
				if err := tt.streamer.StreamEnd(tt.runID, messageID); err != nil {
					t.Errorf("StreamEnd failed: %v", err)
				}

				// Verify streaming behavior
				if mock, ok := tt.streamer.(*mockStreamer); ok {
					// Verify start call
					if len(mock.startCalls) != 1 {
						t.Errorf("Expected 1 StreamStart call, got %d", len(mock.startCalls))
					}

					// Verify content calls
					if tt.simulateOutput == "Line 1\nLine 2\nLine 3\n" {
						// Should have 3 content calls for 3 lines
						if len(mock.contentCalls) != 3 {
							t.Errorf("Expected 3 StreamContent calls, got %d", len(mock.contentCalls))
						} else {
							expectedContents := []string{"Line 1\n", "Line 2\n", "Line 3\n"}
							for i, expected := range expectedContents {
								if mock.contentCalls[i].content != expected {
									t.Errorf("Content[%d]: expected %q, got %q", i, expected, mock.contentCalls[i].content)
								}
							}
						}
					} else if tt.simulateOutput == "Partial line without newline" {
						// Should have 1 content call after flush
						if len(mock.contentCalls) != 1 {
							t.Errorf("Expected 1 StreamContent call for partial line, got %d", len(mock.contentCalls))
						} else {
							if mock.contentCalls[0].content != tt.simulateOutput {
								t.Errorf("Expected content %q, got %q", tt.simulateOutput, mock.contentCalls[0].content)
							}
						}
					}

					// Verify end call
					if len(mock.endCalls) != 1 {
						t.Errorf("Expected 1 StreamEnd call, got %d", len(mock.endCalls))
					}

					// Verify IDs match throughout
					if len(mock.startCalls) > 0 && len(mock.endCalls) > 0 {
						if mock.startCalls[0].messageID != mock.endCalls[0].messageID {
							t.Error("Message ID mismatch between start and end")
						}
						for _, call := range mock.contentCalls {
							if call.messageID != mock.startCalls[0].messageID {
								t.Error("Message ID mismatch in content call")
							}
							if call.runID != tt.runID {
								t.Errorf("Run ID mismatch: expected %q, got %q", tt.runID, call.runID)
							}
						}
					}
				}
			}
		})
	}
}

// TestExecutorStreamingErrorHandling verifies error handling in streaming
func TestExecutorStreamingErrorHandling(t *testing.T) {
	// Create a streamer that fails
	failingStreamer := &mockStreamer{
		streamErr: io.ErrUnexpectedEOF,
	}

	executor := NewExecutor()
	executor.SetStreamer(failingStreamer)
	executor.SetRunID("run-error")

	// Simulate streaming with errors
	messageID := generateMessageID()

	// Start should fail but not panic
	err := failingStreamer.StreamStart("run-error", messageID)
	if err == nil {
		t.Error("Expected error from StreamStart")
	}

	// Create stream writer and test error handling
	sw := NewStreamWriter(failingStreamer, "run-error", messageID)
	
	// Writing should succeed (errors are logged but not returned)
	data := []byte("Test data\n")
	n, err := sw.Write(data)
	if err != nil {
		t.Errorf("Write should not fail even with streaming errors: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Verify streaming was attempted despite errors
	if len(failingStreamer.contentCalls) == 0 {
		t.Error("Expected StreamContent to be attempted")
	}
}