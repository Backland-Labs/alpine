# CLI Commands Specification

## Usage

```
alpine [flags] <task-description>
alpine [flags]
alpine plan [flags] <task-description>
alpine plan [flags] gh-issue <github-issue-url>
alpine --help
alpine --version
```

## Examples

```bash
# Run workflow with task description (with planning)
alpine "Implement user authentication"

# Skip planning phase
alpine --no-plan "Fix bug in payment processing"

# Run with HTTP server for real-time updates
alpine --serve "Implement user dashboard"

# Run with server on custom port
alpine --serve --port 8080 "Add search functionality"

# Run server standalone (no workflow)
alpine --serve

# Generate plan using Gemini (default)
alpine plan "Implement caching layer"

# Generate plan using Claude Code
alpine plan --cc "Implement caching layer"

# Show help
alpine --help

# Show version
alpine --version
```

## Flags

### alpine command
- `--no-plan` - Skip plan generation and execute `/run_implementation_loop` directly
- `--serve` - Enable HTTP server for real-time updates
- `--port` - Port for HTTP server (default: 3001)
- `--help` - Show help message
- `--version` - Show version information

### alpine plan command
- `--cc` - Use Claude Code instead of Gemini for plan generation
- `--help` - Show help message

### alpine plan gh-issue subcommand
- Accepts a GitHub issue URL as the sole argument
- Inherits `--cc` flag from parent `plan` command
- `--help` - Show help message

## Behavior

### Default (with planning)
1. Accepts task description from command line
2. Generates an execution plan using `/make_plan`
3. Runs Claude Code iteratively based on state
4. Updates `agent_state.json` after each step
5. Continues until status is "completed"

### With --no-plan
1. Accepts task description from command line
2. Executes `/run_implementation_loop` command directly
3. Runs Claude Code iteratively based on state
4. Updates `agent_state.json` after each step
5. Continues until status is "completed"

### With --serve (concurrent mode)
1. Starts HTTP server on specified port (default: 3001)
2. Server runs in background, non-blocking
3. Executes workflow normally (with or without planning)
4. Server provides SSE endpoint at `/events` for real-time updates
5. Server shuts down gracefully when workflow completes

### With --serve (standalone mode, no task)
1. Starts HTTP server on specified port (default: 3001)
2. Server runs in foreground, blocking
3. No workflow execution occurs
4. Server continues until interrupted (Ctrl+C)
5. Useful for development and testing

### alpine plan command
1. Accepts task description from command line
2. By default, uses Gemini CLI for plan generation (requires GEMINI_API_KEY)
3. With `--cc` flag, uses Claude Code for plan generation
4. Reads prompt template from `prompts/prompt-plan.md`
5. Outputs plan.md file in the current directory
6. Claude Code execution includes:
   - Read-only tools (Read, Grep, Glob, LS, WebSearch, WebFetch)
   - Full codebase context via `--add-dir .`
   - 5-minute timeout
   - Planning-specific system prompt

### alpine plan gh-issue subcommand
1. Accepts a GitHub issue URL as the sole argument
2. Uses `gh issue view <url> --json title,body` to fetch issue data
3. Requires `gh` CLI to be installed and authenticated
4. Combines issue title and body into a task description format: `Task: <title>\n\n<body>`
5. Passes the combined task description to the plan generation engine
6. Respects the `--cc` flag from parent command for engine selection
7. Outputs plan.md file based on the GitHub issue content
8. Error handling includes:
   - Clear message if `gh` CLI is not found
   - Proper error propagation from `gh` command failures
   - JSON parsing error handling

## Output

- Shows current step being executed
- Displays Claude Code output
- Reports errors clearly

## Interruption

- `Ctrl+C` saves current state and exits cleanly

## Exit Codes

- `0` - Success
- `1` - Error