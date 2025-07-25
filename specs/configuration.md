# Configuration Specification

## Overview

Alpine configuration controls runtime behavior and output settings. Configuration is set via environment variables.

## Configuration Options

### Execution Settings

**ALPINE_WORKDIR**
- Working directory for Claude execution
- Default: Current directory
- Must be an absolute path

### Output Settings

**ALPINE_VERBOSITY**
- Output verbosity level
- Values: `normal`, `verbose`, `debug`
- Default: `normal`
- Behavior:
  - `normal`: Show essential output only
  - `verbose`: Include step descriptions and timing
  - `debug`: Full debug logging

**ALPINE_SHOW_OUTPUT**
- Display Claude command output
- Values: `true`, `false`
- Default: `true`

**ALPINE_SHOW_TOOL_UPDATES**
- Display real-time tool usage information from Claude
- Values: `true`, `false`
- Default: `true`
- Behavior:
  - `true`: Shows sticky header with current task and scrolling log of recent tool calls
  - `false`: Disables real-time tool usage display

### State File Settings

**State File Location**
- Fixed at `.claude/alpine/claude_state.json`
- Directory created automatically when workflow starts
- No longer configurable via environment variable

**ALPINE_AUTO_CLEANUP**
- Delete state file on successful completion
- Values: `true`, `false`
- Default: `true`

## Examples

```bash
# Debug mode
export ALPINE_VERBOSITY=debug
alpine ABC-123

# Quiet mode, keep state file
export ALPINE_SHOW_OUTPUT=false
export ALPINE_AUTO_CLEANUP=false
alpine ABC-123

# Run in different directory
export ALPINE_WORKDIR=/path/to/project
alpine ABC-123

# Disable real-time tool updates
export ALPINE_SHOW_TOOL_UPDATES=false
alpine ABC-123
```