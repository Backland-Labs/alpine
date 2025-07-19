// Package claude provides an interface and types for interacting with the Claude CLI.
//
// This package defines the core abstractions for building and executing Claude commands,
// parsing responses, and managing the workflow of plan and continue operations.
//
// The package follows these design principles:
//   - Type safety through custom types (IssueID, CommandType)
//   - Explicit validation at boundaries
//   - Clean separation between command building, execution, and parsing
//   - Support for both streaming and non-streaming operations
//
// Basic usage:
//
//	cmd := claude.Command{
//	    Type:         claude.CommandTypePlan,
//	    Prompt:       "Create a new feature",
//	    OutputFormat: "json",
//	    SystemPrompt: "You are a helpful assistant",
//	    AllowedTools: []string{"read", "write"},
//	}
//
//	// Validate the command
//	if err := cmd.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use with a Claude implementation to execute
//	// response, err := claudeImpl.Execute(ctx, cmd, opts)
package claude
