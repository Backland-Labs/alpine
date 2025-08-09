package events

import (
	"encoding/json"
	"fmt"
	"time"
)

// AG-UI Protocol Event Types
// These constants define the exact event type strings required by the AG-UI protocol.
// They ensure consistency across the application and compliance with AG-UI specifications.
const (
	// Lifecycle Events
	AGUIEventRunStarted  = "run_started"  // First event when Alpine workflow begins
	AGUIEventRunFinished = "run_finished" // Successful workflow completion
	AGUIEventRunError    = "run_error"    // Workflow failure

	// Text Streaming Events (Claude output)
	AGUIEventTextMessageStart   = "text_message_start"   // Begin Claude stdout stream
	AGUIEventTextMessageContent = "text_message_content" // Claude stdout chunks (delta=true)
	AGUIEventTextMessageEnd     = "text_message_end"     // Complete Claude stdout stream

	// Tool Call Events (Claude tool execution)
	AGUIEventToolCallStarted  = "tool_call_started"  // Tool execution begins
	AGUIEventToolCallFinished = "tool_call_finished" // Tool execution completes
	AGUIEventToolCallError    = "tool_call_error"    // Tool execution fails
)

// PascalCase event type constants for external API compatibility
const (
	ToolCallStart = "ToolCallStart"
	ToolCallEnd   = "ToolCallEnd"
	ToolCallError = "ToolCallError"
)

// AGUISourceClaude identifies Claude as the source of text messages
const AGUISourceClaude = "claude"

// ValidAGUIEventTypes contains all valid AG-UI event type strings
var ValidAGUIEventTypes = map[string]bool{
	AGUIEventRunStarted:         true,
	AGUIEventRunFinished:        true,
	AGUIEventRunError:           true,
	AGUIEventTextMessageStart:   true,
	AGUIEventTextMessageContent: true,
	AGUIEventTextMessageEnd:     true,
	AGUIEventToolCallStarted:    true,
	AGUIEventToolCallFinished:   true,
	AGUIEventToolCallError:      true,
}

// IsValidAGUIEventType checks if the given event type is a valid AG-UI event
func IsValidAGUIEventType(eventType string) bool {
	return ValidAGUIEventTypes[eventType]
}

// BaseEvent interface defines common fields and methods for all AG-UI events.
// All AG-UI events must implement this interface to ensure consistency
// and compatibility with the event streaming system.
type BaseEvent interface {
	// GetType returns the AG-UI event type string (e.g., "tool_call_started")
	GetType() string

	// GetRunID returns the workflow run identifier
	GetRunID() string

	// GetTimestamp returns when the event occurred
	GetTimestamp() time.Time

	// Validate checks if the event has all required fields and valid values
	Validate() error
}

// ToolCallStartEvent represents the beginning of a tool execution
type ToolCallStartEvent struct {
	Type       string    `json:"type"`       // "tool_call_started"
	RunID      string    `json:"runId"`      // Workflow run identifier
	Timestamp  time.Time `json:"timestamp"`  // Event timestamp
	ToolCallID string    `json:"toolCallId"` // Unique tool call identifier
	ToolName   string    `json:"toolName"`   // Name of the tool being executed
}

// GetType returns the event type
func (e *ToolCallStartEvent) GetType() string {
	return e.Type
}

// GetRunID returns the run identifier
func (e *ToolCallStartEvent) GetRunID() string {
	return e.RunID
}

// GetTimestamp returns the event timestamp
func (e *ToolCallStartEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// Validate checks if the event has required fields
func (e *ToolCallStartEvent) Validate() error {
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.Type != AGUIEventToolCallStarted {
		return fmt.Errorf("invalid type for ToolCallStartEvent: %s", e.Type)
	}
	if e.RunID == "" {
		return fmt.Errorf("runId is required")
	}
	if e.ToolCallID == "" {
		return fmt.Errorf("toolCallId is required")
	}
	if e.ToolName == "" {
		return fmt.Errorf("toolName is required")
	}
	return nil
}

// MarshalJSON implements custom JSON marshaling for AG-UI protocol compliance
func (e *ToolCallStartEvent) MarshalJSON() ([]byte, error) {
	// Convert internal snake_case to AG-UI PascalCase format
	type Alias ToolCallStartEvent
	return json.Marshal(&struct {
		Type         string    `json:"type"`
		RunID        string    `json:"runId"`
		Timestamp    time.Time `json:"timestamp"`
		ToolCallID   string    `json:"toolCallId"`
		ToolCallName string    `json:"toolCallName"` // AG-UI protocol uses toolCallName
		*Alias
	}{
		Type:         ToolCallStart, // Use PascalCase constant
		RunID:        e.RunID,
		Timestamp:    e.Timestamp,
		ToolCallID:   e.ToolCallID,
		ToolCallName: e.ToolName, // Map ToolName to toolCallName
		Alias:        (*Alias)(e),
	})
}

// ToolCallEndEvent represents the completion of a tool execution
type ToolCallEndEvent struct {
	Type       string    `json:"type"`       // "tool_call_finished"
	RunID      string    `json:"runId"`      // Workflow run identifier
	Timestamp  time.Time `json:"timestamp"`  // Event timestamp
	ToolCallID string    `json:"toolCallId"` // Unique tool call identifier
	ToolName   string    `json:"toolName"`   // Name of the tool that was executed
	Duration   string    `json:"duration"`   // Execution duration
}

// GetType returns the event type
func (e *ToolCallEndEvent) GetType() string {
	return e.Type
}

// GetRunID returns the run identifier
func (e *ToolCallEndEvent) GetRunID() string {
	return e.RunID
}

// GetTimestamp returns the event timestamp
func (e *ToolCallEndEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// Validate checks if the event has required fields
func (e *ToolCallEndEvent) Validate() error {
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.Type != AGUIEventToolCallFinished {
		return fmt.Errorf("invalid type for ToolCallEndEvent: %s", e.Type)
	}
	if e.RunID == "" {
		return fmt.Errorf("runId is required")
	}
	if e.ToolCallID == "" {
		return fmt.Errorf("toolCallId is required")
	}
	if e.ToolName == "" {
		return fmt.Errorf("toolName is required")
	}
	return nil
}

// ToolCallErrorEvent represents a failed tool execution
type ToolCallErrorEvent struct {
	Type       string    `json:"type"`       // "tool_call_error"
	RunID      string    `json:"runId"`      // Workflow run identifier
	Timestamp  time.Time `json:"timestamp"`  // Event timestamp
	ToolCallID string    `json:"toolCallId"` // Unique tool call identifier
	ToolName   string    `json:"toolName"`   // Name of the tool that failed
	Error      string    `json:"error"`      // Error message
}

// GetType returns the event type
func (e *ToolCallErrorEvent) GetType() string {
	return e.Type
}

// GetRunID returns the run identifier
func (e *ToolCallErrorEvent) GetRunID() string {
	return e.RunID
}

// GetTimestamp returns the event timestamp
func (e *ToolCallErrorEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// Validate checks if the event has required fields
func (e *ToolCallErrorEvent) Validate() error {
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.Type != AGUIEventToolCallError {
		return fmt.Errorf("invalid type for ToolCallErrorEvent: %s", e.Type)
	}
	if e.RunID == "" {
		return fmt.Errorf("runId is required")
	}
	if e.ToolCallID == "" {
		return fmt.Errorf("toolCallId is required")
	}
	if e.ToolName == "" {
		return fmt.Errorf("toolName is required")
	}
	return nil
}
