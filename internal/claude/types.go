package claude

import (
	"errors"
	"fmt"
)

// CommandType represents the type of Claude command
type CommandType string

const (
	// CommandTypePlan represents a plan command
	CommandTypePlan CommandType = "plan"
	// CommandTypeContinue represents a continue command
	CommandTypeContinue CommandType = "continue"
)

// IssueID represents a Linear issue identifier
type IssueID string

// Validate checks if the IssueID is valid
func (id IssueID) Validate() error {
	if id == "" {
		return errors.New("issue ID cannot be empty")
	}
	return nil
}

// Command represents a Claude CLI command configuration
type Command struct {
	// Type specifies whether this is a plan or continue command
	Type CommandType
	// Prompt is the user prompt for the command
	Prompt string
	// Content is an alternative to Prompt for newer command style
	Content string
	// SessionID identifies the session for continuing conversations
	SessionID string
	// OutputFormat specifies the output format (e.g., "json")
	OutputFormat string
	// SystemPrompt provides system-level instructions to Claude
	SystemPrompt string
	// AllowedTools lists the tools Claude is allowed to use
	AllowedTools []string
}

// Validate checks if the Command is valid
func (c *Command) Validate() error {
	switch c.Type {
	case CommandTypePlan, CommandTypeContinue:
		return nil
	default:
		return fmt.Errorf("invalid command type: %s", c.Type)
	}
}

// Response represents a response from Claude
type Response struct {
	// Content contains the response text
	Content string
	// ContinueFlag indicates whether Claude wants to continue
	ContinueFlag bool
	// Error contains any error message
	Error string
}

// HasError returns true if the response contains an error
func (r *Response) HasError() bool {
	return r.Error != ""
}

// CommandOptions represents execution options for a Claude command
type CommandOptions struct {
	// Stream enables streaming output
	Stream bool
	// Timeout specifies the command timeout in seconds
	Timeout int
	// WorkingDir specifies the working directory for command execution
	WorkingDir string
}