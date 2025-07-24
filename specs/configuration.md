# Configuration Specification

## Overview

River configuration controls runtime behavior and output settings. Configuration is set via environment variables.

## Configuration Options

### Execution Settings

**RIVER_WORKDIR**
- Working directory for Claude execution
- Default: Current directory
- Must be an absolute path

### Output Settings

**RIVER_VERBOSITY**
- Output verbosity level
- Values: `normal`, `verbose`, `debug`
- Default: `normal`
- Behavior:
  - `normal`: Show essential output only
  - `verbose`: Include step descriptions and timing
  - `debug`: Full debug logging

**RIVER_SHOW_OUTPUT**
- Display Claude command output
- Values: `true`, `false`
- Default: `true`

**RIVER_SHOW_TOOL_UPDATES**
- Display real-time tool usage information from Claude
- Values: `true`, `false`
- Default: `true`
- Behavior:
  - `true`: Shows sticky header with current task and scrolling log of recent tool calls
  - `false`: Disables real-time tool usage display

### State File Settings

**State File Location**
- Fixed at `.claude/river/claude_state.json`
- Directory created automatically when workflow starts
- No longer configurable via environment variable

**RIVER_AUTO_CLEANUP**
- Delete state file on successful completion
- Values: `true`, `false`
- Default: `true`

## Examples

```bash
# Debug mode
export RIVER_VERBOSITY=debug
river ABC-123

# Quiet mode, keep state file
export RIVER_SHOW_OUTPUT=false
export RIVER_AUTO_CLEANUP=false
river ABC-123

# Run in different directory
export RIVER_WORKDIR=/path/to/project
river ABC-123

# Disable real-time tool updates
export RIVER_SHOW_TOOL_UPDATES=false
river ABC-123
```