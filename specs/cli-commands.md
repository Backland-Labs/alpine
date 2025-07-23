# CLI Commands Specification

## Usage

```
river [flags] <task-description>
river [flags] --file <file-path>
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

# Show help
river --help

# Show version
river --version
```

## Flags

- `--no-plan` - Skip plan generation and execute `/run_implementation_loop` directly
- `--file <path>` - Read task description from a file
- `--help` - Show help message
- `--version` - Show version information

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

## Output

- Shows current step being executed
- Displays Claude Code output
- Reports errors clearly

## Interruption

- `Ctrl+C` saves current state and exits cleanly

## Exit Codes

- `0` - Success
- `1` - Error