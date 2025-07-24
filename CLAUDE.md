# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

River is a CLI orchestrator for Claude Code that automates iterative AI-assisted development workflows. It accepts task descriptions and runs Claude Code in a loop based on a state-driven workflow.

Current state: Go implementation complete (v0.2.0). Linear dependency removed.

## Architecture

The project follows a state-driven architecture where:
1. River accepts a task description (command line or file)
2. Optionally generates a plan using `/make_plan`
3. Executes Claude Code iteratively based on `claude_state.json`
4. Continues until status is "completed"

### State File Schema
```json
{
  "current_step_description": "string",
  "next_step_prompt": "string", 
  "status": "running" | "completed"
}
```

### Directory Isolation

When worktrees are enabled (default behavior), River ensures complete isolation:
- Claude commands execute in the worktree directory
- State files (`claude_state.json`) are created in the worktree
- All file operations are confined to the worktree
- The main repository remains unmodified during execution

## Specifications

Key specifications are located in the `specs/` directory:
- [architecture.md](specs/architecture.md): Single binary Go application, standard project layout
- [cli-commands.md](specs/cli-commands.md): Simple CLI with `river <task-description>` and `--no-plan` flag
- [state-management.md](specs/state-management.md): JSON state file format and transitions
- [error-handling.md](specs/error-handling.md): Go error handling patterns
- [configuration.md](specs/configuration.md): Environment variable configuration
- [code-quality.md](specs/code-quality.md): Linting standards and code quality requirements
- [claude-code-hooks](specs/claude-code-hooks.md): Hook scripts for Claude Code integration
- [amp-cli.md](specs/amp-cli.md): Amp Code CLI integration as alternative to Claude Code
- [gemini-cli.md](specs/gemini-cli.md): Gemini CLI integration for non-interactive AI assistance

## Development Commands

### Go Build
```bash
# Build the binary
go build -o river cmd/river/main.go

# Run tests
go test ./...

# Format code
go fmt ./...

# Run linter
golangci-lint run
```

### Running River
```bash
# Run with task description
./river "Implement user authentication"

# Run without plan generation
./river "Fix bug in payment processing" --no-plan

# Run with task from file
./river --file task.md

# Bare execution mode - continue from existing state or start fresh
./river --no-plan --no-worktree
```

## Key Implementation Notes

1. **Single Binary**: All functionality compiled into one executable
2. **Minimal Dependencies**: Only `github.com/spf13/cobra` for CLI
3. **Error Handling**: Explicit error handling, no panics in production
4. **State Management**: Monitor `claude_state.json` for workflow progress
5. **Claude Integration**: Execute `claude` command with specific MCP servers and tools
6. **Code Style**: Write idiomatic Go code following standard conventions
7. **Quality**: Use standard Go tools (`go fmt`, `golangci-lint`) for formatting and linting

## Worktree Directory Isolation

River uses Git worktrees to provide isolated environments for Claude Code execution:

1. **Automatic Directory Context**: When River creates a worktree, all Claude commands automatically execute within that worktree directory, not the original repository.

2. **Complete Isolation**: File operations, state management, and all Claude interactions are confined to the worktree, preventing unintended changes to the main repository.

3. **Working Directory Inheritance**: River ensures Claude inherits the correct working directory through proper `cmd.Dir` configuration in the executor.

4. **Fallback Behavior**: If working directory detection fails, River logs a warning and allows Claude to use its default directory behavior.

### Worktree Usage
```bash
# Default behavior - creates an isolated worktree
./river "Implement new feature"

# Disable worktree isolation (work in current directory)
./river "Fix bug" --no-worktree

# Control worktree cleanup
export RIVER_GIT_AUTO_CLEANUP=false  # Preserve worktrees after completion
```

## Workflow Integration

River integrates with:
- **Claude Code**: Executes with restricted tools and custom system prompt
- **Slash Commands**: `/make_plan` for planning, `/run_implementation_loop` for direct execution, `/verify_plan` to verify @plan.md is implemented fully
- **Task Input**: Direct task descriptions or file input (no external API dependencies)

## References

- Claude Code CLI reference: https://docs.anthropic.com/en/docs/claude-code/cli-reference

## Troubleshooting

### Working Directory Issues

**Problem**: Claude commands not executing in the expected directory
- **Symptom**: Files created in wrong location, state file in main repo instead of worktree
- **Solution**: Ensure you're using River v0.2.1+ which includes the working directory fix
- **Debug**: Check River logs for "Set Claude working directory" messages

**Problem**: "Failed to get working directory" warnings
- **Symptom**: Warning logs about working directory detection failure
- **Cause**: Permission issues or invalid current directory
- **Solution**: River will continue with default behavior; ensure you have proper permissions

**Problem**: Worktree not being used despite default settings
- **Check**: Verify Git is installed and repository is initialized
- **Check**: Ensure `--no-worktree` flag is not set
- **Check**: Confirm `RIVER_GIT_AUTO_WORKTREE` is not set to "false"

### Debug Logging

Enable debug logging to trace directory operations:
```bash
export RIVER_LOG_LEVEL=debug
./river "Your task"
```

Look for these log entries:
- "Set Claude working directory: /path/to/worktree"
- "Creating worktree at: /path/to/worktree"
- "Failed to get working directory" (indicates fallback mode)