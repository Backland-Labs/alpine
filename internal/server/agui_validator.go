package server

import (
	"errors"
	"fmt"

	"github.com/Backland-Labs/alpine/internal/events"
)

// AG-UI validation errors
var (
	ErrInvalidEventSequence = errors.New("invalid AG-UI event sequence")
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidEventType     = errors.New("invalid AG-UI event type")
)

// validateEventSequence validates that a sequence of events follows AG-UI protocol rules
func validateEventSequence(eventList []WorkflowEvent) error {
	if len(eventList) == 0 {
		return nil
	}

	// First event must be run_started
	if eventList[0].Type != events.AGUIEventRunStarted {
		return fmt.Errorf("%w: first event must be run_started, got %s", ErrInvalidEventSequence, eventList[0].Type)
	}

	// Track state
	runStarted := false
	textMessageStarted := make(map[string]bool) // messageId -> started

	for i, event := range eventList {
		switch event.Type {
		case events.AGUIEventRunStarted:
			if i != 0 {
				return fmt.Errorf("%w: run_started must be the first event", ErrInvalidEventSequence)
			}
			runStarted = true

		case events.AGUIEventTextMessageStart:
			if !runStarted {
				return fmt.Errorf("%w: text_message_start before run_started", ErrInvalidEventSequence)
			}
			if event.MessageID == "" {
				return fmt.Errorf("%w: text_message_start missing messageId", ErrMissingRequiredField)
			}
			textMessageStarted[event.MessageID] = true

		case events.AGUIEventTextMessageContent:
			if !runStarted {
				return fmt.Errorf("%w: text_message_content before run_started", ErrInvalidEventSequence)
			}
			if !textMessageStarted[event.MessageID] {
				return fmt.Errorf("%w: text_message_content without matching text_message_start", ErrInvalidEventSequence)
			}

		case events.AGUIEventTextMessageEnd:
			if !textMessageStarted[event.MessageID] {
				return fmt.Errorf("%w: text_message_end without matching text_message_start", ErrInvalidEventSequence)
			}
			delete(textMessageStarted, event.MessageID)

		case events.AGUIEventRunFinished, events.AGUIEventRunError:
			// Check if any text messages are still open
			if len(textMessageStarted) > 0 {
				return fmt.Errorf("%w: %s before text_message_end", ErrInvalidEventSequence, event.Type)
			}
		}
	}

	return nil
}

// validateEventFields validates that an event has all required fields per AG-UI spec
func validateEventFields(event WorkflowEvent) error {
	// All events require type, runId, and timestamp
	if event.Type == "" {
		return fmt.Errorf("%w: type", ErrMissingRequiredField)
	}
	if event.RunID == "" {
		return fmt.Errorf("%w: runId", ErrMissingRequiredField)
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp", ErrMissingRequiredField)
	}

	// Validate event type
	if !events.IsValidAGUIEventType(event.Type) {
		return fmt.Errorf("%w: %s", ErrInvalidEventType, event.Type)
	}

	// Type-specific validation
	switch event.Type {
	case events.AGUIEventTextMessageStart:
		if event.MessageID == "" {
			return fmt.Errorf("%w: messageId", ErrMissingRequiredField)
		}
		if event.Source == "" {
			return fmt.Errorf("%w: source", ErrMissingRequiredField)
		}

	case events.AGUIEventTextMessageContent:
		if event.MessageID == "" {
			return fmt.Errorf("%w: messageId", ErrMissingRequiredField)
		}
		if event.Source == "" {
			return fmt.Errorf("%w: source", ErrMissingRequiredField)
		}
		// Content and Delta are validated in the test itself

	case events.AGUIEventTextMessageEnd:
		if event.MessageID == "" {
			return fmt.Errorf("%w: messageId", ErrMissingRequiredField)
		}
		if event.Source == "" {
			return fmt.Errorf("%w: source", ErrMissingRequiredField)
		}
		// Complete flag is validated in the test itself
	}

	return nil
}
