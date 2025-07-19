# River

River is a CLI tool that automates software development workflows by integrating Linear (project management) with Claude Code to implement features using Test-Driven Development.

## Overview

River processes Linear issues by:
1. Creating isolated git worktrees for each issue
2. Generating implementation plans using Claude Code
3. Implementing features following TDD methodology
4. Updating Linear issues upon completion

## Prerequisites

- Go 1.21 or later
- `claude` CLI (Claude Code) installed and configured
- Git repository initialized

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/river.git
cd river

# Build the binary
make build

# Or install to $GOPATH/bin
make install
```

## Usage

```bash
# Run River with a Linear issue ID
river LINEAR-123

# Run with streaming output (JSON format)
river --stream LINEAR-123

# The tool will:
# 1. Create a worktree at ../river-linear-123
# 2. Switch to a new branch named 'linear-123'
# 3. Execute Claude workflow with TDD methodology
# 4. Process until completion or max iterations
```

## How It Works

1. **Environment Validation**: River first checks that all required dependencies (Claude CLI) are available.

2. **Worktree Creation**: Creates an isolated git worktree in the parent directory (e.g., `../river-linear-123`) with a dedicated branch.

3. **Claude Integration**: Uses the Claude CLI directly to:
   - Create implementation plans with `/make_plan`
   - Continue workflow iterations with `/continue`
   - Process Linear issues following TDD methodology

4. **Iterative Processing**: Automatically handles continue loops up to 50 iterations for complex tasks.

## Command-Line Options

- `--stream`: Enable JSON streaming output for real-time progress monitoring

## Development

```bash
# Run tests
make test

# Run integration tests
make test-integration

# Build for current platform
make build

# Build for multiple platforms
make build-all

# Clean build artifacts
make clean
```

## Environment Variables

No environment variables are required for River itself. Linear API access is handled through the Claude Code MCP integration.

## Project Structure

```
river/
├── cmd/river/          # CLI entry point
│   ├── main.go        # Main application logic
│   └── validation.go  # Environment validation
├── internal/
│   ├── claude/        # Claude CLI integration
│   │   ├── command.go # Command building
│   │   ├── executor.go# Command execution
│   │   ├── types.go   # Type definitions
│   │   └── interface.go
│   ├── git/           # Git worktree management
│   └── runner/        # Workflow orchestration
├── test/
│   └── integration/   # Integration tests
├── specs/             # Specifications
├── plan.md           # Implementation plan
└── CLAUDE.md         # Claude Code instructions
```

## Architecture

River follows a clean architecture with clear separation of concerns:

- **CLI Layer** (`cmd/river`): Handles command-line parsing and validation
- **Claude Integration** (`internal/claude`): Provides type-safe interface to Claude CLI
- **Git Operations** (`internal/git`): Manages worktree creation and branch switching
- **Runner** (`internal/runner`): Orchestrates the workflow execution

## Error Handling

River implements fail-fast principles:
- Environment validation happens before any operations
- Clear error messages guide users to resolve issues
- Timeouts prevent hanging on long-running commands (default: 120 seconds)
