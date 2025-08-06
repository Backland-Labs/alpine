# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Alpine is a CLI orchestrator for Claude Code that automates iterative AI-assisted development workflows. It accepts task descriptions and runs Claude Code in a loop based on a state-driven workflow.

Current state: Go implementation complete (v0.2.0). Linear dependency removed.

## Architecture

The project follows a state-driven architecture where:
1. Alpine accepts a task description (command line or file)
2. Optionally generates a plan using `/make_plan`
3. Executes Claude Code iteratively based on `agent_state.json`
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

When worktrees are enabled (default behavior), Alpine ensures complete isolation:
- Claude commands execute in the worktree directory
- State files (`agent_state.json`) are created in the worktree
- All file operations are confined to the worktree
- The main repository remains unmodified during execution

## Specifications

Key specifications are located in the `specs/` directory:
- [system-design.md](specs/system-design.md): Covers architecture, project structure, configuration, and state management.
- [cli-commands.md](specs/cli-commands.md): Details all CLI commands, flags, and usage examples.
- [code-quality.md](specs/code-quality.md): Defines linting standards and code quality requirements.
- [error-handling.md](specs/error-handling.md): Outlines Go error handling patterns and principles.
- [testing-strategy.md](specs/testing-strategy.md): Explains the project's testing philosophy, including unit, integration, and E2E tests.
- [release-process.md](specs/release-process.md): Provides step-by-step instructions for publishing new releases.
- [claude-code-hooks.md](specs/claude-code-hooks.md): Specification for integrating with Claude Code hooks.
- [gemini-cli.md](specs/gemini-cli.md): Specification for integrating with the Gemini CLI.
- [server.md](specs/server.md): HTTP server functionality with Server-Sent Events for real-time updates.
- [troubleshooting.md](specs/troubleshooting.md): A guide to common issues and their solutions.

## Development Commands

### Go Build
```bash
# Build the binary
go build -o alpine cmd/alpine/main.go

# Run tests
go test ./...

# Format code
go fmt ./...

# Run linter
golangci-lint run
```

### Running Alpine
```bash
# Run with task description
./alpine "Implement user authentication"

# Run without plan generation
./alpine "Fix bug in payment processing" --no-plan

# Bare execution mode - continue from existing state or start fresh
./alpine --no-plan --no-worktree

# Generate plan from GitHub issue
./alpine plan gh-issue https://github.com/owner/repo/issues/123

# Generate plan from GitHub issue using Claude Code
./alpine plan --cc gh-issue https://github.com/owner/repo/issues/123

# Generate plans in parallel using isolated worktrees
./alpine plan --worktree gh-issue https://github.com/owner/repo/issues/123 &
./alpine plan --worktree gh-issue https://github.com/owner/repo/issues/124 &
wait

# Generate plan in worktree and keep it for inspection
./alpine plan --worktree --cleanup=false "Complex feature implementation"

# Run with HTTP server for real-time updates
./alpine --serve "Implement new feature"

# Run server on custom port
./alpine --serve --port 8080 "Add logging feature"

# Run server standalone (no workflow)
./alpine --serve
```

### REST API Server Usage

Alpine provides a REST API for programmatic workflow management when running with the `--serve` flag. The server runs on port 3001 by default (configurable with `--port`). The REST API server will only ever be run in an isolated Docker container. Consider this when making architectural decisions.

#### Starting the Server

```bash
# Build the docker container
docker build -t alpine-code . 

# Start server in the container
docker run -d -p 3001:3001 --env-file .env alpine-code

# Follow/tail logs in real-time
docker logs -f <container_name_or_id>
```

#### REST API Endpoints

**Health Check**
```bash
curl http://localhost:3001/health
```

**List Available Agents**
```bash
curl http://localhost:3001/agents/list
```

**Start Workflow from GitHub Issue**
```bash
curl -X POST http://localhost:3001/agents/run \
  -H "Content-Type: application/json" \
  -d '{"github_issue_url": "https://github.com/owner/repo/issues/123"}'
```

**List All Runs**
```bash
curl http://localhost:3001/runs
```

**Get Run Details**
```bash
curl http://localhost:3001/runs/{run-id}
```

**Cancel Running Workflow**
```bash
curl -X POST http://localhost:3001/runs/{run-id}/cancel
```

**Get Plan Content**
```bash
curl http://localhost:3001/plans/{run-id}
```

**Approve Plan**
```bash
curl -X POST http://localhost:3001/plans/{run-id}/approve
```

**Send Plan Feedback**
```bash
curl -X POST http://localhost:3001/plans/{run-id}/feedback \
  -H "Content-Type: application/json" \
  -d '{"feedback": "Please add error handling for edge cases"}'
```

#### Complete Workflow Example

1. **Start the server**:
```bash
./alpine --serve --port 3001
```

2. **Create a new run** from a GitHub issue:
```bash
curl -X POST http://localhost:3001/agents/run \
  -H "Content-Type: application/json" \
  -d '{"github_issue_url": "https://github.com/myorg/myrepo/issues/42"}'
```
Response:
```json
{
  "run_id": "run_abc123",
  "status": "running",
  "message": "Workflow started successfully"
}
```

3. **Monitor progress** via Server-Sent Events:
```bash
curl http://localhost:3001/runs/run_abc123/events
```
This will stream real-time events as the workflow executes.

4. **Check run status** periodically:
```bash
curl http://localhost:3001/runs/run_abc123
```

5. **Approve plan** when ready:
```bash
curl -X POST http://localhost:3001/plans/run_abc123/approve
```

#### Notes

- The REST API is available only when Alpine is running with the `--serve` flag
- All endpoints return JSON responses (except SSE endpoints)
- The server uses CORS with wildcard origin for development convenience
- In-memory storage means runs are lost when the server restarts
- Currently only one agent ("alpine-agent") is available

#### Integration Examples

**Python Example**
```python
import requests
import json

# Start a workflow
response = requests.post(
    "http://localhost:3001/agents/run",
    headers={"Content-Type": "application/json"},
    data=json.dumps({
        "github_issue_url": "https://github.com/owner/repo/issues/123"
    })
)

run_data = response.json()
run_id = run_data["run_id"]

# Monitor the run
run_details = requests.get(f"http://localhost:3001/runs/{run_id}").json()
print(f"Status: {run_details['status']}")

# Approve the plan
if run_details["status"] == "waiting_for_approval":
    requests.post(f"http://localhost:3001/plans/{run_id}/approve")
```

**JavaScript Example**
```javascript
// Start a workflow
const response = await fetch('http://localhost:3001/agents/run', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    github_issue_url: 'https://github.com/owner/repo/issues/123'
  })
});

const { run_id } = await response.json();

// Monitor events
const events = new EventSource(`http://localhost:3001/runs/${run_id}/events`);
events.onmessage = (event) => {
  console.log('Event:', JSON.parse(event.data));
};

// Approve plan when ready
await fetch(`http://localhost:3001/plans/${run_id}/approve`, {
  method: 'POST'
});
```

## How to Check Your Work

Alpine is a critical automation tool, so verifying changes is essential. Use these methods to ensure your work is correct:

### 1. Compilation Checks
```bash
# Verify the code compiles
go build -o alpine cmd/alpine/main.go

# Check for compilation errors in all packages
go build ./...

# Verify no type errors
go vet ./...
```

### 2. Testing Suite
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific test package
go test ./internal/cli/...

# Run tests with race detector
go test -race ./...
```

### 3. Code Quality
```bash
# Format all Go files
go fmt ./...

# Run comprehensive linting
golangci-lint run

# Check for common mistakes
go vet ./...

# Check for inefficient code
go mod tidy
go mod verify

# Security scanning
gosec ./...
```

### 4. Integration Testing
```bash
# Build and test basic execution
go build -o alpine cmd/alpine/main.go && ./alpine --help

# Test plan generation (requires GEMINI_API_KEY)
./alpine plan "Add error handling to file operations"

# Test gh-issue plan generation (requires gh CLI and authentication)
./alpine plan gh-issue https://github.com/owner/repo/issues/1

# Test worktree creation
./alpine "Add a test function" --no-plan

# Verify state file creation
./alpine "Simple task" && cat agent_state/agent_state.json

```

### 5. State Management Verification
```bash
# Check state file is valid JSON
./alpine "Test task" --no-plan && jq . agent_state/agent_state.json

# Verify state transitions
./alpine "Multi-step task" && grep -E '"status":\s*"(running|completed)"' agent_state/agent_state.json

# Monitor state changes in real-time
watch -n 1 'jq . agent_state/agent_state.json 2>/dev/null || echo "No state file yet"'
```

### 6. Worktree Isolation Verification
```bash
# Verify worktree is created
./alpine "Test worktree" && git worktree list

# Check files are isolated to worktree
find .git/worktrees -name "agent_state.json"

# Verify main repo is unchanged
git status  # Should show no changes after Alpine execution

# Test cleanup behavior
ALPINE_GIT_AUTO_CLEANUP=false ./alpine "Test cleanup" && git worktree list
```

### 7. Logging and Debug Verification
```bash
# Test debug logging
ALPINE_LOG_LEVEL=debug ./alpine "Debug test" 2>&1 | grep -E "(DEBUG|Set Claude working directory)"

# Verify error handling
./alpine ""  # Should show proper error for empty task

# Check log output format
./alpine "Log test" 2>&1 | grep -E "(INFO|ERROR|WARN)"
```

### 8. End-to-End Workflow Testing
```bash
# Full workflow with plan
./alpine "Implement a simple calculator function"
# Verify: plan.md created, worktree created, agent_state.json exists

# Bare mode workflow
./alpine --no-plan --no-worktree "Add comments to main.go"
# Verify: No worktree created, operates in current directory

# Error recovery test
# Interrupt Alpine (Ctrl+C) and restart
./alpine "Long running task" # Ctrl+C after start
./alpine --no-plan --no-worktree # Should continue from state
```

### 9. Performance and Resource Checks
```bash
# Memory usage monitoring
go test -bench=. -benchmem ./...

# Check binary size
go build -o alpine cmd/alpine/main.go && ls -lh alpine

# Verify no goroutine leaks
go test -race ./...
```

### 10. Pre-Commit Checklist
Before committing changes, always run:
```bash
# Essential checks
go fmt ./...
go vet ./...
go test ./...
golangci-lint run
go build -o alpine cmd/alpine/main.go

# Quick smoke test
./alpine --help && echo "CLI works!"
```

### Automated Verification Script
Create a `verify.sh` script for comprehensive checking:
```bash
#!/bin/bash
set -e

echo "🔍 Running comprehensive verification..."

echo "✓ Formatting code..."
go fmt ./...

echo "✓ Building binary..."
go build -o alpine cmd/alpine/main.go

echo "✓ Running tests..."
go test ./...

echo "✓ Running linter..."
golangci-lint run

echo "✓ Checking vet..."
go vet ./...

echo "✓ Verifying CLI..."
./alpine --help > /dev/null

echo "✅ All checks passed!"
```

### Quick Verification Commands
```bash
# One-liner for quick check
go fmt ./... && go test ./... && go build -o alpine cmd/alpine/main.go

# With linting
go fmt ./... && golangci-lint run && go test ./... && go build -o alpine cmd/alpine/main.go
```

## Key Implementation Notes

1. **Single Binary**: All functionality compiled into one executable
2. **Minimal Dependencies**: Only `github.com/spf13/cobra` for CLI
3. **Error Handling**: Explicit error handling, no panics in production
4. **State Management**: Monitor `agent_state.json` for workflow progress
5. **Claude Integration**: Execute `claude` command with specific MCP servers and tools
6. **Code Style**: Write idiomatic Go code following standard conventions
7. **Quality**: Use standard Go tools (`go fmt`, `golangci-lint`) for formatting and linting

## Worktree Directory Isolation

Alpine uses Git worktrees to provide isolated environments for Claude Code execution:

1. **Automatic Directory Context**: When Alpine creates a worktree, all Claude commands automatically execute within that worktree directory, not the original repository.

2. **Complete Isolation**: File operations, state management, and all Claude interactions are confined to the worktree, preventing unintended changes to the main repository.

3. **Working Directory Inheritance**: Alpine ensures Claude inherits the correct working directory through proper `cmd.Dir` configuration in the executor.

4. **Fallback Behavior**: If working directory detection fails, Alpine logs a warning and allows Claude to use its default directory behavior.

### Worktree Usage
```bash
# Default behavior - creates an isolated worktree
./alpine "Implement new feature"

# Disable worktree isolation (work in current directory)
./alpine "Fix bug" --no-worktree

# Control worktree cleanup
export ALPINE_GIT_AUTO_CLEANUP=false  # Preserve worktrees after completion
```

## Workflow Integration

Alpine integrates with:
- **Claude Code**: Executes with restricted tools and custom system prompt
- **Slash Commands**: `/make_plan` for planning, `/run_implementation_loop` for direct execution, `/verify_plan` to verify @plan.md is implemented fully
- **Task Input**: Direct task descriptions or file input (no external API dependencies)

## References

- Claude Code CLI reference: https://docs.anthropic.com/en/docs/claude-code/cli-reference

## Troubleshooting

### Working Directory Issues

**Problem**: Claude commands not executing in the expected directory
- **Symptom**: Files created in wrong location, state file in main repo instead of worktree
- **Solution**: Ensure you're using Alpine v0.2.1+ which includes the working directory fix
- **Debug**: Check Alpine logs for "Set Claude working directory" messages

**Problem**: "Failed to get working directory" warnings
- **Symptom**: Warning logs about working directory detection failure
- **Cause**: Permission issues or invalid current directory
- **Solution**: Alpine will continue with default behavior; ensure you have proper permissions

**Problem**: Worktree not being used despite default settings
- **Check**: Verify Git is installed and repository is initialized
- **Check**: Ensure `--no-worktree` flag is not set
- **Check**: Confirm `ALPINE_GIT_AUTO_WORKTREE` is not set to "false"

### Debug Logging

Enable debug logging to trace directory operations:
```bash
export ALPINE_LOG_LEVEL=debug
./alpine "Your task"
```

Look for these log entries:
- "Set Claude working directory: /path/to/worktree"
- "Creating worktree at: /path/to/worktree"
- "Failed to get working directory" (indicates fallback mode)