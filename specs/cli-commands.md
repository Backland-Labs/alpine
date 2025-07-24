# CLI Commands Specification

## Usage

```
river [flags] <task-description>
river [flags] --file <file-path>
river plan [flags] <task-description>
river --help
river --version
```

## Examples

```bash
# Run workflow with task description (with planning)
river "Implement user authentication"

# Skip planning phase
river --no-plan "Fix bug in payment processing"

# Read task from file
river --file task.md

# Generate plan using Gemini (default)
river plan "Implement caching layer"

# Generate plan using Claude Code
river plan --cc "Implement caching layer"

# Show help
river --help

# Show version
river --version
```

## Flags

### river command
- `--no-plan` - Skip plan generation and execute `/run_implementation_loop` directly
- `--file <path>` - Read task description from a file
- `--help` - Show help message
- `--version` - Show version information

### river plan command
- `--cc` - Use Claude Code instead of Gemini for plan generation
- `--help` - Show help message

## Behavior

### Default (with planning)
1. Accepts task description from command line or file
2. Generates an execution plan using `/make_plan`
3. Runs Claude Code iteratively based on state
4. Updates `claude_state.json` after each step
5. Continues until status is "completed"

### With --no-plan
1. Accepts task description from command line or file
2. Executes `/run_implementation_loop` command directly
3. Runs Claude Code iteratively based on state
4. Updates `claude_state.json` after each step
5. Continues until status is "completed"

### river plan command
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

## Output

- Shows current step being executed
- Displays Claude Code output
- Reports errors clearly

## Interruption

- `Ctrl+C` saves current state and exits cleanly

## Exit Codes

- `0` - Success
- `1` - Error