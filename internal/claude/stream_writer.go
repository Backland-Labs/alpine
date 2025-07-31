package claude

import (
	"bytes"
	"io"

	"github.com/Backland-Labs/alpine/internal/events"
	"github.com/Backland-Labs/alpine/internal/logger"
)

// StreamWriter implements io.Writer to bridge stdout streaming to the Streamer interface.
// It buffers data and sends complete lines to the streamer, ensuring real-time
// streaming of Claude's output while handling partial writes gracefully.
type StreamWriter struct {
	streamer  events.Streamer
	runID     string
	messageID string
	buffer    bytes.Buffer
}

// NewStreamWriter creates a new StreamWriter that sends output lines to the provided Streamer
func NewStreamWriter(streamer events.Streamer, runID, messageID string) *StreamWriter {
	return &StreamWriter{
		streamer:  streamer,
		runID:     runID,
		messageID: messageID,
	}
}

// Write implements io.Writer interface. It buffers incoming data and sends complete
// lines to the streamer. Partial lines are held in the buffer until a newline is received.
func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	
	// Add to buffer
	sw.buffer.Write(p)
	
	// Process complete lines
	for {
		line, err := sw.buffer.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				// No more complete lines, put back what we read
				if len(line) > 0 {
					sw.buffer.Write(line)
				}
				break
			}
			return n, err
		}
		
		// Stream the complete line
		if streamErr := sw.streamer.StreamContent(sw.runID, sw.messageID, string(line)); streamErr != nil {
			// Log streaming error but don't fail the write
			logger.WithFields(map[string]interface{}{
				"error":     streamErr,
				"runID":     sw.runID,
				"messageID": sw.messageID,
			}).Debug("Failed to stream content")
		}
	}
	
	return n, nil
}

// Flush sends any remaining buffered data to the streamer.
// This should be called when the writer is done to ensure all data is streamed.
func (sw *StreamWriter) Flush() error {
	if sw.buffer.Len() > 0 {
		remaining := sw.buffer.String()
		if streamErr := sw.streamer.StreamContent(sw.runID, sw.messageID, remaining); streamErr != nil {
			logger.WithFields(map[string]interface{}{
				"error":     streamErr,
				"runID":     sw.runID,
				"messageID": sw.messageID,
			}).Debug("Failed to stream final content")
			return streamErr
		}
		sw.buffer.Reset()
	}
	return nil
}

// multiWriterWithFlush wraps io.MultiWriter to add flush capability
type multiWriterWithFlush struct {
	writers []io.Writer
}

// Write writes to all underlying writers
func (mw *multiWriterWithFlush) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

// Flush flushes all writers that support flushing
func (mw *multiWriterWithFlush) Flush() error {
	for _, w := range mw.writers {
		if flusher, ok := w.(interface{ Flush() error }); ok {
			if err := flusher.Flush(); err != nil {
				return err
			}
		}
	}
	return nil
}

// newMultiWriterWithFlush creates a MultiWriter that supports flushing
func newMultiWriterWithFlush(writers ...io.Writer) *multiWriterWithFlush {
	return &multiWriterWithFlush{writers: writers}
}