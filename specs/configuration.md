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

### State File Settings

**RIVER_STATE_FILE**
- Path to state file
- Default: `./claude_state.json`
- Can be relative or absolute path

**RIVER_AUTO_CLEANUP**
- Delete state file on successful completion
- Values: `true`, `false`
- Default: `true`

## Examples

```bash
# Debug mode with custom state file
export RIVER_VERBOSITY=debug
export RIVER_STATE_FILE=/tmp/river-state.json
river ABC-123

# Quiet mode, keep state file
export RIVER_SHOW_OUTPUT=false
export RIVER_AUTO_CLEANUP=false
river ABC-123

# Run in different directory
export RIVER_WORKDIR=/path/to/project
river ABC-123
```