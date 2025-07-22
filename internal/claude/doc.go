// Package claude provides functionality for executing Claude commands
// as part of the River workflow automation system.
//
// This package handles:
//   - Building Claude command-line invocations with appropriate flags
//   - Managing MCP server connections
//   - Enforcing tool restrictions for safe execution
//   - Passing environment variables and system prompts
//   - Handling command timeouts and error reporting
//
// Example usage:
//
//	executor := claude.NewExecutor()
//	config := claude.ExecuteConfig{
//		Prompt:      "/make_plan ISSUE-123",
//		StateFile:   "claude_state.json",
//		LinearIssue: "ISSUE-123",
//		MCPServers:  []string{"linear-server"},
//		Timeout:     5 * time.Minute,
//	}
//	output, err := executor.Execute(context.Background(), config)
package claude
