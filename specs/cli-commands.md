# CLI Commands Specification

## Usage

```
river [flags] <linear-issue-id>
river --help
river --version
```

## Examples

```bash
# Run workflow from Linear issue (with planning)
river ABC-123

# Skip planning phase
river --no-plan ABC-123

# Show help
river --help

# Show version
river --version
```

## Flags

- `--no-plan` - Skip plan generation and execute `/ralph` directly
- `--help` - Show help message
- `--version` - Show version information

## Behavior

### Default (with planning)
1. Validates Linear issue ID format
2. Fetches the issue from Linear
3. Generates an execution plan using `/make_plan`
4. Runs Claude Code iteratively based on state
5. Updates `claude_state.json` after each step
6. Continues until status is "completed"

### With --no-plan
1. Validates Linear issue ID format
2. Fetches the issue from Linear
3. Executes `/ralph` command directly
4. Runs Claude Code iteratively based on state
5. Updates `claude_state.json` after each step
6. Continues until status is "completed"

## Output

- Shows current step being executed
- Displays Claude Code output
- Reports errors clearly

## Interruption

- `Ctrl+C` saves current state and exits cleanly

## Exit Codes

- `0` - Success
- `1` - Error