# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

River is a Go-based CLI tool that automates software development workflows by integrating Linear (project management) with Claude Code to implement features using Test-Driven Development (TDD) methodology.

## Commands

### Building and Running
```bash
# Build the River binary
make build

# Install River to $GOPATH/bin
make install

# Run River with a Linear issue ID
river <LINEAR-ISSUE-ID>

# Run with streaming JSON output
river --stream <LINEAR-ISSUE-ID>

# Build for multiple platforms
make build-all
```

### Testing
```bash
# Run all tests
make test

# Run tests for a specific package
go test -v ./internal/claude/

# Run a specific test
go test -v -run TestExecutorStreaming ./internal/claude/
```

### Development
```bash
# Clean build artifacts
make clean

# Run with arguments via make
make run ARGS="--stream RIV-123"
```

## Architecture

### Core Components

1. **CLI Layer (`cmd/river/`)**
   - `main.go`: Entry point, workflow orchestration, and main logic
   - `validation.go`: Environment validation (Claude CLI)

2. **Claude Integration (`internal/claude/`)**
   - `interface.go`: Claude interface definition
   - `command.go`: Command building logic for plan/continue
   - `executor.go`: Command execution with streaming support
   - `types.go`: Response types and structures

3. **Git Operations (`internal/git/`)**
   - `worktree.go`: Git worktree management for isolated development

4. **Runner Package (`internal/runner/`)**
   - Minimal implementation, designed for future workflow orchestration

### Workflow

1. **Environment Validation**: Checks for `claude` CLI
2. **Worktree Creation**: Creates `../river-<issue-id>` directory with new git branch
3. **Claude Execution**: 
   - Initial `/make_plan` command with TDD instructions
   - Continue loop (up to 50 iterations) until completion
4. **Output**: Stream JSON or standard output based on `--stream` flag

### Key Implementation Details

- **TDD Enforcement**: System prompt requires Test-Driven Development approach
- **Isolated Development**: Each issue gets its own git worktree to avoid conflicts
- **Tool Configuration**: Allows Linear and code-editing tools, disables web tools
- **Error Handling**: Fail-fast approach with clear error messages
- **No External Dependencies**: Uses only Go standard library (testify for tests)

## Environment Requirements

- **Required System Commands**:
  - `claude`: Claude Code CLI must be installed and in PATH
  - `git`: For worktree operations

- **Note**: Linear API access is handled through Claude Code's MCP integration, not through environment variables

## Specs

**IMPORTANT:** Never deleted the `specs/` directory as it contains essential specifications for the River project.

The `specs/` directory contains detailed technical specifications for different aspects of the River system:

### [Overview](specs/overview.md)
High-level overview of all specifications and recommended implementation order.

### [Type System](specs/types.md)
Defines data structures and interfaces for Linear integration, Claude responses, and internal workflow state.

### [Error Handling](specs/error_handling.md)
Patterns for graceful failure management, retry strategies, and recovery mechanisms for API failures.

### [Logging System](specs/logging.md)
Structured logging requirements for debugging and monitoring the automation workflow.

### [Testing Strategy](specs/testing.md)
Test-Driven Development approach, testing patterns, and infrastructure for unit and integration tests.