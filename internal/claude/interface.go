package claude

import (
	"context"
)

// Claude defines the interface for interacting with the Claude CLI
type Claude interface {
	// BuildCommand constructs CLI arguments for a Claude command
	BuildCommand(ctx context.Context, cmd Command) ([]string, error)

	// Execute runs a Claude command and returns the response
	Execute(ctx context.Context, cmd Command, opts CommandOptions) (*Response, error)

	// ParseResponse parses the JSON output from Claude
	ParseResponse(ctx context.Context, output string) (*Response, error)
}
