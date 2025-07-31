// Package events provides streaming interfaces and implementations for real-time output
package events

// Streamer defines the interface for streaming operations.
// It provides methods to start, send content, and end a streaming session.
// Implementations must be thread-safe for concurrent use.
type Streamer interface {
	// StreamStart begins a new streaming session
	StreamStart(runID, messageID string) error

	// StreamContent sends a chunk of content during streaming
	StreamContent(runID, messageID, content string) error

	// StreamEnd completes the streaming session
	StreamEnd(runID, messageID string) error
}

// NoOpStreamer is a no-operation implementation of the Streamer interface.
// It's used for backward compatibility when streaming is disabled.
// All methods return nil without performing any operations.
type NoOpStreamer struct{}

// StreamStart for NoOpStreamer does nothing and returns nil
func (n *NoOpStreamer) StreamStart(runID, messageID string) error {
	return nil
}

// StreamContent for NoOpStreamer does nothing and returns nil
func (n *NoOpStreamer) StreamContent(runID, messageID, content string) error {
	return nil
}

// StreamEnd for NoOpStreamer does nothing and returns nil
func (n *NoOpStreamer) StreamEnd(runID, messageID string) error {
	return nil
}
