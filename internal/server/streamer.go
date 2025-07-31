package server

import (
	"time"
)

// ServerStreamer implements the Streamer interface using the server's BroadcastEvent method.
// It transforms streaming operations into AG-UI protocol compliant SSE events.
// This implementation is thread-safe and supports multiple concurrent streaming sessions.
type ServerStreamer struct {
	server *Server
}

// NewServerStreamer creates a new ServerStreamer instance that uses the provided
// server's BroadcastEvent infrastructure to emit streaming events.
func NewServerStreamer(server *Server) *ServerStreamer {
	return &ServerStreamer{server: server}
}

// StreamStart broadcasts a text_message_start event
func (s *ServerStreamer) StreamStart(runID, messageID string) error {
	event := WorkflowEvent{
		Type:      "text_message_start",
		RunID:     runID,
		MessageID: messageID,
		Timestamp: time.Now(),
		Source:    "claude",
	}
	s.server.BroadcastEvent(event)
	return nil
}

// StreamContent broadcasts a text_message_content event with the content chunk
func (s *ServerStreamer) StreamContent(runID, messageID, content string) error {
	event := WorkflowEvent{
		Type:      "text_message_content",
		RunID:     runID,
		MessageID: messageID,
		Timestamp: time.Now(),
		Content:   content,
		Delta:     true,
		Source:    "claude",
	}
	s.server.BroadcastEvent(event)
	return nil
}

// StreamEnd broadcasts a text_message_end event
func (s *ServerStreamer) StreamEnd(runID, messageID string) error {
	event := WorkflowEvent{
		Type:      "text_message_end",
		RunID:     runID,
		MessageID: messageID,
		Timestamp: time.Now(),
		Complete:  true,
		Source:    "claude",
	}
	s.server.BroadcastEvent(event)
	return nil
}
