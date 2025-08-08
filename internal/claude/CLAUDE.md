# Claude Executor Module

This module handles all interactions with the Claude Code CLI, including command execution, stream processing, and hook integration.

## Module Overview

The `claude` package is responsible for:
- Executing Claude Code commands with proper environment configuration
- Managing stdout/stderr stream processing and buffering
- Integrating with AG-UI hooks for real-time event emission
- Monitoring todo list changes during execution
- Handling working directory isolation for worktree support

## Key Components

### Executor (`executor.go`)
- Primary interface: `Executor` with `Execute(ctx, config) error`
- Stream handling: Buffered processing with 64KB buffer size
- Output modes: Combined, separate stdout/stderr, or custom stream writers
- Working directory management for worktree isolation

### Stream Writer (`stream_writer.go`)
- Real-time output processing with ANSI color preservation
- Line-based buffering for clean output formatting
- Thread-safe concurrent write handling
- Flush mechanisms for ensuring complete output delivery

### Hooks Integration (`hooks.go`, `agui_hooks.go`)
- Claude Code hooks configuration for todo monitoring
- AG-UI protocol event emission via external Rust binary
- JSON-based inter-process communication
- Error handling for hook failures (non-blocking)

### Todo Monitor (`todo_monitor.go`)
- Watches `todo.json` file for changes during execution
- Parses todo list updates and emits events
- Handles file creation, updates, and deletion gracefully
- Non-blocking monitoring to avoid impacting main execution

## Implementation Guidelines

### Stream Processing
```go
// Always use buffered readers for stream processing
scanner := bufio.NewScanner(stdout)
scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

// Handle incomplete lines properly
if !scanner.Scan() && scanner.Err() == nil {
    // Process remaining data
}
```

### Error Handling
- Never panic in production code
- Log errors with appropriate context
- Return wrapped errors for better debugging
- Hook failures should not block main execution

### Testing Patterns
- Use `TestExecutor` for integration tests
- Mock external commands with controlled responses
- Test stream processing edge cases (large output, ANSI codes)
- Verify hook integration without external dependencies

## Working Directory Management

The executor ensures Claude Code runs in the correct directory:
1. Detects current working directory
2. Sets `cmd.Dir` for subprocess execution
3. Falls back gracefully if detection fails
4. Critical for worktree isolation

## Hook Configuration

Claude Code hooks are configured via environment:
- `CLAUDE_CODE_MCP_SETTINGS_SCHEMA`: Hook configuration JSON
- Points to Rust binary in `hooks/` directory
- Monitors `todo.json` for real-time updates

## Performance Considerations

- Stream processing uses 64KB buffers to handle large outputs
- Todo monitoring runs in separate goroutine
- Hook execution is non-blocking
- Proper cleanup of resources in defer blocks

## Common Patterns

### Executing Claude Commands
```go
executor := NewExecutor()
config := Config{
    Command: "claude",
    Args: []string{"--task", "implement feature"},
    WorkingDir: "/path/to/worktree",
}
err := executor.Execute(ctx, config)
```

### Custom Stream Handling
```go
config.StdoutWriter = customWriter
config.StderrWriter = errorWriter
config.OutputMode = SeparateOutput
```

## Testing Requirements

- All new features must include unit tests
- Integration tests for Claude command execution
- Mock testing for external dependencies
- Stream processing edge case coverage
- Hook integration verification

## Dependencies

- Standard library only (no external packages)
- Rust binary for AG-UI hook (`hooks/alpine-ag-ui-emitter.rs`)
- Claude Code CLI (external command)

## Notes

- This module is performance-critical for the Alpine workflow
- Stream processing must handle very large outputs efficiently
- Hook failures should be logged but not block execution
- Working directory isolation is essential for worktree support