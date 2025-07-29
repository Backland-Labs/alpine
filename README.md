# Alpine CLI

> [!WARNING]
> **This repository is experimental and under active development.** APIs, commands, and behaviors may change without notice. Use in production at your own risk.

## Overview

Alpine is an experimental multi-coding agent orchestration system. The goal is to automate the feature request to PR pipeline uniquely. Alpine is being used to build Alpine, so every feature implemented in Alpine is implemented via itself. This is a self-building agent orchestration tool.

Think of Alpine as a conductor for AI coding agents - orchestrating complex development tasks by breaking them down, planning execution, and iterating until completion.


## Architecture

Alpine orchestrates AI coding agents through a two-phase approach:

### Planning Phase
- **Gemini** (default): Fast, lightweight plan generation via API
- **Claude Code** (optional): Deep codebase analysis with full context via `--cc` flag

### Execution Phase
- **Claude Code**: Executes the plan iteratively with full tool access
- **State Management**: JSON-based state tracking ensures progress persistence
- **Worktree Isolation**: Git worktrees provide safe, isolated execution environments

The architecture allows different AI models to play to their strengths - Gemini for rapid ideation, Claude for deep implementation work.

## Installation

### Quick Start

```bash
# Clone and build
git clone https://github.com/[username]/alpine.git
cd alpine
go build -o alpine cmd/alpine/main.go

# Install Claude Code CLI (required)
curl -fsSL https://claude.ai/code/install.sh | sh
claude auth login

# Set up Gemini API key (for plan generation)
export GEMINI_API_KEY="your-api-key"

# Run your first task
./alpine "Add a hello world endpoint to main.go"
```

## Usage

```bash
# Basic task execution
alpine "Implement user authentication with JWT tokens"

# Skip planning, execute directly
alpine "Fix the payment processing bug" --no-plan

# Continue from existing state
alpine --continue

# Generate plan from GitHub issue
alpine plan gh-issue https://github.com/owner/repo/issues/123

# Run HTTP server with Server-Sent Events (SSE)
alpine --serve                    # Start server on default port 3001
alpine --serve --port 8080        # Start server on custom port
```

### HTTP Server Mode

Alpine includes a built-in HTTP server with both REST API and Server-Sent Events (SSE) support for programmatic workflow management:

#### Server Modes
- **Standalone mode**: Run `alpine --serve` to start just the HTTP server
- **With workflow**: Run `alpine --serve "Your task"` to execute a task with API access
- **Port configuration**: Default port 3001, configurable with `--port`

#### REST API Endpoints

Alpine provides a comprehensive REST API for programmatic workflow management:

```bash
# Health check
curl http://localhost:3001/health

# Start a workflow from GitHub issue
curl -X POST http://localhost:3001/agents/run \
  -H "Content-Type: application/json" \
  -d '{"github_issue_url": "https://github.com/owner/repo/issues/123"}'

# List all workflow runs
curl http://localhost:3001/runs

# Get specific run details
curl http://localhost:3001/runs/{run-id}

# Monitor run progress via Server-Sent Events
curl http://localhost:3001/runs/{run-id}/events

# Cancel a running workflow
curl -X POST http://localhost:3001/runs/{run-id}/cancel

# Approve an execution plan
curl -X POST http://localhost:3001/plans/{run-id}/approve
```

#### Complete Workflow Example

```bash
# Start the server
./alpine --serve --port 3001

# Create a new run from GitHub issue
curl -X POST http://localhost:3001/agents/run \
  -H "Content-Type: application/json" \
  -d '{"github_issue_url": "https://github.com/myorg/myrepo/issues/42"}'

# Monitor progress in real-time
curl http://localhost:3001/runs/{run-id}/events

# Approve plan when generated
curl -X POST http://localhost:3001/plans/{run-id}/approve
```

The REST API enables integration with CI/CD pipelines, monitoring tools, and custom applications for automated development workflows.


## License

Prosperity Public License 3.0.0 - see LICENSE.md for details.

**Key Points:**
- Free for non-commercial use
- 30-day trial for commercial use
- Contributions back don't count as commercial use
