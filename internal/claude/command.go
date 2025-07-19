package claude

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const (
	// claudeBinary is the name of the Claude CLI executable
	claudeBinary = "claude"
	
	// Command line flags
	flagSystemPrompt = "-s"
	flagPrompt       = "-p"
	flagOutputFormat = "--output-format"
	flagAllowedTools = "--allowed-tools"
	
	// Slash commands
	slashCommandPlan     = "/make_plan"
	slashCommandContinue = "/continue"
	
	// Default values
	defaultOutputFormat = "json"
)

// commandBuilder implements the Claude interface for building CLI commands
type commandBuilder struct{}

// NewCommandBuilder creates a new command builder instance that implements
// the Claude interface. The returned builder can construct CLI arguments
// for both plan and continue commands.
func NewCommandBuilder() Claude {
	return &commandBuilder{}
}

// New creates a new command builder instance that implements
// the Claude interface. The returned builder can construct CLI arguments
// for both plan and continue commands.
func New() Claude {
	return &commandBuilder{}
}

// BuildCommand constructs the CLI arguments for a Claude command.
// It validates the command structure and builds the appropriate arguments
// based on the command type (plan or continue).
//
// The returned slice contains the complete command line arguments ready
// for execution, including the binary name and all flags.
func (b *commandBuilder) BuildCommand(ctx context.Context, cmd Command) ([]string, error) {
	// Validate the command structure
	if err := cmd.Validate(); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}
	
	// Use Content field if Prompt is empty (for backward compatibility)
	prompt := cmd.Prompt
	if prompt == "" {
		prompt = cmd.Content
	}
	
	// Validate prompt is not empty
	if strings.TrimSpace(prompt) == "" {
		return nil, errors.New("prompt cannot be empty")
	}
	
	// Start building the command arguments
	args := []string{claudeBinary}
	
	// Add system prompt if provided
	if cmd.SystemPrompt != "" {
		args = append(args, flagSystemPrompt, cmd.SystemPrompt)
	}
	
	// Add the main prompt with appropriate command prefix
	promptPrefix := b.getPromptPrefix(cmd.Type)
	args = append(args, flagPrompt, fmt.Sprintf("%s %s", promptPrefix, prompt))
	
	// Add output format (default to json if not specified)
	outputFormat := cmd.OutputFormat
	if outputFormat == "" {
		outputFormat = defaultOutputFormat
	}
	args = append(args, flagOutputFormat, outputFormat)
	
	// Add allowed tools if specified
	if len(cmd.AllowedTools) > 0 {
		args = append(args, flagAllowedTools, strings.Join(cmd.AllowedTools, ","))
	}
	
	return args, nil
}

// getPromptPrefix returns the appropriate slash command prefix for the given command type
func (b *commandBuilder) getPromptPrefix(cmdType CommandType) string {
	switch cmdType {
	case CommandTypePlan:
		return slashCommandPlan
	case CommandTypeContinue:
		return slashCommandContinue
	default:
		// This should never happen due to validation, but provides safety
		return ""
	}
}

