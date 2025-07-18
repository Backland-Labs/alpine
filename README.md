# River

River is a CLI tool that automates software development workflows by integrating Linear (project management) with Claude Code to implement features using Test-Driven Development.

## Overview

River processes Linear sub-issues by:
1. Creating isolated git worktrees for each issue
2. Generating implementation plans using Claude Code
3. Implementing features following TDD methodology
4. Updating Linear issues upon completion

## Prerequisites

- Go 1.21 or later
- `claude` CLI (Claude Code) installed and configured
- `jq` command-line JSON processor
- Git repository initialized
- Linear API key set as `LINEAR_API_KEY` environment variable

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

# The tool will:
# 1. Create a worktree at ../river-linear-123
# 2. Copy auto_claude.sh to the worktree
# 3. Execute the automation workflow
```

## How It Works

1. **Worktree Creation**: River creates an isolated git worktree in the parent directory (e.g., `../river-linear-123`) to keep the main repository clean.

2. **Script Execution**: The `auto_claude.sh` script is copied to the worktree and executed, which:
   - Fetches issue details from Linear
   - Creates a detailed implementation plan
   - Implements features using TDD
   - Updates the Linear issue status

3. **Isolation**: Each issue gets its own worktree, allowing parallel development on multiple issues.

## Development

```bash
# Run tests
make test

# Build for multiple platforms
make build-all

# Clean build artifacts
make clean
```

## Environment Variables

- `LINEAR_API_KEY`: Required for Linear API access
- `PARENT_ISSUE_ID`: (Optional) Can be set to specify the parent issue when running auto_claude.sh directly

## Project Structure

```
river/
├── cmd/river/          # CLI entry point
├── internal/
│   ├── git/           # Git worktree management
│   └── runner/        # Script execution
├── auto_claude.sh     # Core automation script
├── specs/             # Specifications
└── CLAUDE.md          # Claude Code instructions
```
