package events

// AG-UI Protocol Event Types
// These constants define the exact event type strings required by the AG-UI protocol.
// They ensure consistency across the application and compliance with AG-UI specifications.
const (
	// Lifecycle Events
	AGUIEventRunStarted  = "run_started"   // First event when Alpine workflow begins
	AGUIEventRunFinished = "run_finished"  // Successful workflow completion
	AGUIEventRunError    = "run_error"     // Workflow failure
	
	// Text Streaming Events (Claude output)
	AGUIEventTextMessageStart   = "text_message_start"   // Begin Claude stdout stream
	AGUIEventTextMessageContent = "text_message_content" // Claude stdout chunks (delta=true)
	AGUIEventTextMessageEnd     = "text_message_end"     // Complete Claude stdout stream
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
}

// IsValidAGUIEventType checks if the given event type is a valid AG-UI event
func IsValidAGUIEventType(eventType string) bool {
	return ValidAGUIEventTypes[eventType]
}