# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

River is a CLI orchestrator for Claude Code that automates iterative AI-assisted development workflows. It fetches Linear issues and runs Claude Code in a loop based on a state-driven workflow.

Current state: Transitioning from Python prototype (main.py) to Go implementation.

## Architecture

The project follows a state-driven architecture where:
1. River fetches a Linear issue
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

## Specifications

Key specifications are located in the `specs/` directory:
- [architecture.md](specs/architecture.md): Single binary Go application, standard project layout
- [cli-commands.md](specs/cli-commands.md): Simple CLI with `river <issue-id>` and `--no-plan` flag
- [state-management.md](specs/state-management.md): JSON state file format and transitions
- [error-handling.md](specs/error-handling.md): Go error handling patterns
- [configuration.md](specs/configuration.md): Environment variable configuration

## Development Commands

### Go Build (future)
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

### Current Python Prototype
```bash
# Run the Python prototype
./main.py
```

## Key Implementation Notes

1. **Single Binary**: All functionality compiled into one executable
2. **Minimal Dependencies**: Only `github.com/spf13/cobra` for CLI
3. **Error Handling**: Explicit error handling, no panics in production
4. **State Management**: Monitor `claude_state.json` for workflow progress
5. **Claude Integration**: Execute `claude` command with specific MCP servers and tools
6. **Code Style**: Write idiomatic Go code following standard conventions
7. **Quality**: Use standard Go tools (`go fmt`, `golangci-lint`) for formatting and linting

## Workflow Integration

River integrates with:
- **Linear**: For fetching issue details (via Claude's MCP server)
- **Claude Code**: Executes with restricted tools and custom system prompt
- **Slash Commands**: `/make_plan` for planning, `/ralph` for direct execution

## References

- Claude Code CLI reference: https://docs.anthropic.com/en/docs/claude-code/cli-reference