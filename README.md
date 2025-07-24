# River CLI

River is a CLI orchestrator for Claude Code that automates iterative AI-assisted development workflows. It accepts task descriptions and runs Claude Code in a loop based on a state-driven workflow.

## Features

- **Task-Based Workflow**: Provide task descriptions directly from the command line or file
- **Automated Planning**: Generate execution plans using the `/make_plan` slash command
- **State-Driven Execution**: Monitor and manage workflow progress through a JSON state file
- **Iterative Development**: Automatically continue execution until task completion
- **Enhanced UX**: Colored output, progress indicators, and detailed logging
- **Fast & Efficient**: Written in Go for superior performance compared to the Python version

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/[username]/river.git
cd river

# Build the binary
go build -o river cmd/river/main.go

# Optionally install to your PATH
go install ./cmd/river
```

### Prerequisites

- Go 1.19 or higher
- Claude Code CLI installed and configured (for execution)
- Gemini API key set as `GEMINI_API_KEY` environment variable (for plan generation with Gemini)
- MCP servers configured (if using specific tools)

### Installing Claude Code CLI

To use River with Claude Code (for execution or plan generation with `--cc`), you need to install the Claude Code CLI:

1. Visit the Claude Code website: [claude.ai/code](https://claude.ai/code)
2. Follow the installation instructions for your platform
3. Authenticate with your Anthropic account: `claude auth login`
4. Verify installation: `claude --version`

### Pre-built Binaries

Download pre-built binaries from the [Releases](https://github.com/[username]/river/releases) page for your platform:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Usage

### Basic Usage

Run River with a task description:
```bash
river "Implement user authentication with JWT tokens"
```

### Skip Planning Phase

Execute directly without generating a plan using the `--no-plan` flag:
```bash
river "Fix the bug in payment processing" --no-plan
```

### Read Task from File

For complex task descriptions, you can provide them via a file:
```bash
river --file task.md
```

The file should contain the complete task description in plain text or Markdown format.

### Command Options

```
river [flags] <task-description>

Flags:
  -f, --file string   Read task description from file
  -h, --help          Help for river
      --no-plan       Skip plan generation and execute directly
  -v, --version       Version for river

river plan [flags] <task>

Flags:
      --cc            Use Claude Code instead of Gemini for plan generation
  -h, --help          Help for plan
```

## Plan Generation

River supports two engines for generating implementation plans:

### Gemini (Default)
By default, River uses Gemini for plan generation. This requires a Gemini API key:

```bash
export GEMINI_API_KEY="your-api-key"
river plan "Add user authentication to the web app"
```

### Claude Code (Alternative)
You can use Claude Code for plan generation with the `--cc` flag:

```bash
river plan --cc "Add user authentication to the web app"
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
river plan "Implement caching layer for API responses"

# Generate a plan using Claude Code
river plan --cc "Implement caching layer for API responses"

# Generate a plan from a file description
echo "Refactor the authentication module to use JWT tokens" > task.md
river plan --file task.md

# Use Claude Code with file input
river plan --cc --file task.md

# Generate a plan from a GitHub issue
river plan gh-issue https://github.com/owner/repo/issues/123

# Use Claude Code to generate a plan from a GitHub issue
river plan --cc gh-issue https://github.com/owner/repo/issues/123
```

### GitHub Issue Integration

The `river plan gh-issue` subcommand allows you to generate implementation plans directly from GitHub issues. This feature uses the GitHub CLI (`gh`) to fetch issue details and generate a comprehensive plan.

**Requirements:**
- The `gh` CLI must be installed and authenticated
- You must have access to the specified GitHub issue

**Usage:**
```bash
# Basic usage with Gemini
river plan gh-issue <github-issue-url>

# Use Claude Code for plan generation
river plan --cc gh-issue <github-issue-url>
```

The command will:
1. Fetch the issue title and body using `gh issue view`
2. Combine them into a task description
3. Generate a plan using your chosen engine (Gemini or Claude Code)

## Configuration

River uses environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `RIVER_WORK_DIR` | Working directory for execution | Current directory |
| `RIVER_VERBOSITY` | Logging level (debug, verbose, normal) | normal |
| `RIVER_SHOW_OUTPUT` | Show Claude output (true/false) | true |
| `RIVER_AUTO_CLEANUP` | Auto-cleanup state file on completion | true |

Example configuration:
```bash
export RIVER_VERBOSITY=debug
export RIVER_SHOW_OUTPUT=true
export RIVER_AUTO_CLEANUP=false
river "Implement caching layer"
```

## How It Works

1. **Task Input**: River accepts a task description from the command line or file
2. **Planning Phase** (optional): Generates an execution plan using Claude Code's `/make_plan` command
3. **Execution**: Runs Claude Code iteratively based on the state file (`claude_state.json`)
4. **State Management**: Monitors progress through a JSON state file
5. **Completion**: Continues until the task status is "completed"

### State File Format

River uses a JSON state file to track workflow progress:

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
- **Debug Logging**: Detailed logs with timestamps when `RIVER_VERBOSITY=debug`

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
river/
├── cmd/river/          # Main application entry point
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
- River uses `claude_state.json` to track progress
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

For issues and feature requests, please use the [GitHub Issues](https://github.com/[username]/river/issues) page.