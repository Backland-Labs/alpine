# State Management Specification

## Overview

River uses a JSON state file (`claude_state.json`) to track workflow progress and enable iteration between Claude Code executions.

## State File Location

- Default: `.claude/river/claude_state.json` (Claude Code's standard location)
- Override with `RIVER_STATE_FILE` environment variable
- Directory created automatically when workflow starts

## Schema

```json
{
  "current_step_description": "string",
  "next_step_prompt": "string",
  "status": "string"
}
```

## Fields

### current_step_description
- Human-readable description of what was just completed
- Updated by Claude Code during execution
- Example: `"Implemented task 1 from plan.md"`

### next_step_prompt
- The command/prompt to execute in the next iteration
- Set by Claude Code at the end of each step
- Common values: `/run_implementation_loop`, `/continue`, or custom prompts
- Empty string or missing when workflow is complete

### status
- Workflow state indicator
- Valid values:
  - `"running"` - Actively executing
  - `"completed"` - Workflow finished

## State Transitions

1. **Initialize**: Create file with initial prompt and status `"running"`
2. **Iterate**: Claude Code updates all fields during execution
3. **Complete**: When status becomes `"completed"`, workflow ends

### Bare Mode Behavior

When running with `--no-plan --no-worktree`:
- If state file exists: Continue from existing workflow
- If no state file exists: Initialize new workflow with `/run_implementation_loop`

## File Operations

- **Read**: Parse JSON, validate schema
- **Write**: Pretty-print JSON with 2-space indentation
- **Watch**: Monitor file for changes during Claude execution

## Error Handling

- Missing file: Create new workflow
- Invalid JSON: Report error and exit
- Missing required fields: Report error and exit