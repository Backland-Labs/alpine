# Alpine CLI

Alpine is a CLI orchestrator for Claude Code that automates iterative AI-assisted development workflows. It accepts task descriptions and runs Claude Code in a loop based on a state-driven workflow.

## Features

- **Task-Based Workflow**: Provide task descriptions directly from the command line or file
- **Automated Planning**: Generate execution plans using the `/make_plan` slash command
- **State-Driven Execution**: Monitor and manage workflow progress through a JSON state file
- **Iterative Development**: Automatically continue execution until task completion
- **Enhanced UX**: Colored output, progress indicators, and detailed logging
- **Real-time Tool Logging**: Live display of agent operations with sticky header showing current task and scrolling log of recent tool usage
- **Fast & Efficient**: Written in Go for superior performance compared to the Python version

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/[username]/alpine.git
cd alpine

# Build the binary
go build -o alpine cmd/alpine/main.go

# Optionally install to your PATH
go install ./cmd/alpine
```

### Prerequisites

- Go 1.19 or higher
- Claude Code CLI installed and configured (for execution)
- Gemini API key set as `GEMINI_API_KEY` environment variable (for plan generation with Gemini)
- MCP servers configured (if using specific tools)

### Installing Claude Code CLI

To use Alpine with Claude Code (for execution or plan generation with `--cc`), you need to install the Claude Code CLI:

1. Visit the Claude Code website: [claude.ai/code](https://claude.ai/code)
2. Follow the installation instructions for your platform
3. Authenticate with your Anthropic account: `claude auth login`
4. Verify installation: `claude --version`

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/[username]/alpine/releases) page for your platform:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Usage

### Basic Usage

Run Alpine with a task description:
```bash
alpine "Implement user authentication with JWT tokens"
```

### Skip Planning Phase

Execute directly without generating a plan using the `--no-plan` flag:
```bash
alpine "Fix the bug in payment processing" --no-plan
```

### Continue from Existing State

Continue from where you left off using the `--continue` flag:
```bash
alpine --continue
```

### Read Task from File

For complex task descriptions, you can provide them via a file:
```bash
alpine --file task.md
```

The file should contain the complete task description in plain text or Markdown format.

### Command Options

```
alpine [flags] <task-description>

Flags:
      --continue      Continue from existing state (equivalent to --no-plan --no-worktree)
      --file string   Read task description from a file
  -h, --help          help for alpine
      --no-plan       Skip plan generation and execute directly
      --no-worktree   Disable git worktree creation
  -v, --version       Show version information

alpine plan [flags] <task>

Flags:
      --cc     Use Claude Code instead of Gemini for plan generation
  -h, --help   help for plan
```

## Plan Generation

Alpine supports two engines for generating implementation plans:

### Gemini (Default)
By default, Alpine uses Gemini for plan generation. This requires a Gemini API key:

```bash
export GEMINI_API_KEY="your-api-key"
alpine plan "Add user authentication to the web app"
```

### Claude Code (Alternative)
You can use Claude Code for plan generation with the `--cc` flag:

```bash
alpine plan --cc "Add user authentication to the web app"
```

### Comparison: Gemini vs Claude Code

| Feature | Gemini | Claude Code |
|---------|--------|-------------|
| Default option | ✓ | |
| API key required | ✓ (GEMINI_API_KEY) | |
| CLI installation required | | ✓ |
| Output streaming | Real-time | Buffered |
| Codebase context | Limited | Full (via `--add-dir .`) |
| Multi-turn conversation | | ✓ |
| Tool usage | | Read-only tools |
| Typical speed | Fast | Slower (more analysis) |

### Examples

```bash
# Generate a plan using Gemini (default)
alpine plan "Implement caching layer for API responses"

# Generate a plan using Claude Code
alpine plan --cc "Implement caching layer for API responses"

# Generate a plan from a file description
echo "Refactor the authentication module to use JWT tokens" > task.md
alpine plan --file task.md

# Use Claude Code with file input
alpine plan --cc --file task.md

# Generate a plan from a GitHub issue
alpine plan gh-issue https://github.com/owner/repo/issues/123

# Use Claude Code to generate a plan from a GitHub issue
alpine plan --cc gh-issue https://github.com/owner/repo/issues/123
```

### GitHub Issue Integration

The `alpine plan gh-issue` subcommand allows you to generate implementation plans directly from GitHub issues. This feature uses the GitHub CLI (`gh`) to fetch issue details and generate a comprehensive plan.

**Requirements:**
- The `gh` CLI must be installed and authenticated
- You must have access to the specified GitHub issue

**Usage:**
```bash
# Basic usage with Gemini
alpine plan gh-issue <github-issue-url>

# Use Claude Code for plan generation
alpine plan --cc gh-issue <github-issue-url>
```

The command will:
1. Fetch the issue title and body using `gh issue view`
2. Combine them into a task description
3. Generate a plan using your chosen engine (Gemini or Claude Code)

## Configuration

Alpine uses environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `ALPINE_WORK_DIR` | Working directory for execution | Current directory |
| `ALPINE_VERBOSITY` | Logging level (debug, verbose, normal) | normal |
| `ALPINE_SHOW_OUTPUT` | Show Claude output (true/false) | true |
| `ALPINE_AUTO_CLEANUP` | Auto-cleanup state file on completion | true |
| `ALPINE_SHOW_TOOL_UPDATES` | Display real-time tool usage logs | true |

Example configuration:
```bash
export ALPINE_VERBOSITY=debug
export ALPINE_SHOW_OUTPUT=true
export ALPINE_AUTO_CLEANUP=false
alpine "Implement caching layer"
```

## How It Works

1. **Task Input**: Alpine accepts a task description from the command line or file
2. **Planning Phase** (optional): Generates an execution plan using Claude Code's `/make_plan` command
3. **Execution**: Runs Claude Code iteratively based on the state file (`claude_state.json`)
4. **State Management**: Monitors progress through a JSON state file
5. **Completion**: Continues until the task status is "completed"

### State File Format

Alpine uses a JSON state file to track workflow progress:

```json
{
  "current_step_description": "Implementing authentication middleware",
  "next_step_prompt": "Add JWT token validation to the middleware",
  "status": "running"
}
```

Status values:
- `"running"`: Task is in progress
- `"completed"`: Task is finished

## Features

### Enhanced User Experience

- **Colored Output**: Terminal colors for better readability (respects `NO_COLOR` environment variable)
- **Progress Indicators**: Animated spinner with elapsed time during long operations
- **Debug Logging**: Detailed logs with timestamps when `ALPINE_VERBOSITY=debug`
- **Real-time Tool Logging**: Live feed showing the last 3-4 tool operations performed by the agent
  - Sticky header displays the current primary task
  - Scrolling log shows recent tool usage (Read, Edit, Write, etc.)
  - Non-intrusive display that updates without flickering
  - Can be disabled by setting `ALPINE_SHOW_TOOL_UPDATES=false`

### Workflow Automation

- **Iterative Execution**: Automatically continues until task completion
- **State Persistence**: Maintains progress across interruptions
- **Graceful Shutdown**: Handles Ctrl+C cleanly

## Differences from Python Version

The Go implementation (v0.2.0) provides:

1. **Single Binary**: No Python runtime required
2. **Better Performance**: ~5x faster startup, ~50% less memory usage
3. **Enhanced UX**: Colored output, progress indicators, better error messages
4. **Simplified Input**: Direct task descriptions instead of Linear issue IDs
5. **Cross-Platform**: Native binaries for Linux, macOS, and Windows

## Requirements

- Claude Code CLI installed and configured
- Go 1.19+ (for building from source)
- Terminal with UTF-8 support (for colored output)

## Development

### Building from Source

```bash
# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run linter
golangci-lint run

# Format code
go fmt ./...
```

### Project Structure

```
alpine/
├── cmd/alpine/          # Main application entry point
├── internal/           # Internal packages
│   ├── cli/           # CLI command handling
│   ├── claude/        # Claude Code integration
│   ├── config/        # Configuration management
│   ├── logger/        # Logging utilities
│   ├── output/        # Terminal output formatting
│   ├── performance/   # Performance monitoring
│   ├── validation/    # Validation utilities
│   └── workflow/      # Core workflow engine
├── specs/             # Architecture specifications
└── test/              # Integration tests
```

## Troubleshooting

### Plan Generation Issues

**Missing GEMINI_API_KEY error**
```
Error: GEMINI_API_KEY environment variable is not set
```
Solution: Set your Gemini API key:
```bash
export GEMINI_API_KEY="your-api-key"
```

**Claude Code CLI not found error**
```
Error: Claude Code CLI not found. Please install from https://claude.ai/code
```
Solution: Install Claude Code CLI following the instructions in the Prerequisites section.

**Plan generation timeout**
- Claude Code plan generation has a 5-minute timeout
- For complex codebases, this may be exceeded
- Try breaking down your task into smaller, more specific requirements

### Common Issues

**State file conflicts**
- Alpine uses `claude_state.json` to track progress
- If you see unexpected behavior, try removing this file and restarting
```bash
rm claude_state.json
```

**Worktree permission errors**
- Ensure you have write permissions in the Git repository
- Check that Git is properly initialized: `git status`

**MCP server errors**
- Verify MCP servers are properly configured
- Check Claude Code configuration: `claude config list`

## License

[Your License Here]

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) before submitting PRs.

## Support

For issues and feature requests, please use the [GitHub Issues](https://github.com/[username]/alpine/issues) page.