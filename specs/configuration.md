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

**RIVER_SHOW_TODO_UPDATES**
- Display TODO progress tracking from Claude
- Values: `true`, `false`
- Default: `true`
- Behavior:
  - `true`: Shows TODO list changes and progress during execution
  - `false`: Disables TODO progress display

### State File Settings

**State File Location**
- Fixed at `.claude/river/claude_state.json`
- Directory created automatically when workflow starts
- No longer configurable via environment variable

**RIVER_AUTO_CLEANUP**
- Delete state file on successful completion
- Values: `true`, `false`
- Default: `true`

### Git Worktree Settings

**RIVER_GIT_ENABLED**
- Enable Git worktree support for isolated execution
- Values: `true`, `false`
- Default: `true`
- Behavior:
  - `true`: Creates an isolated Git worktree for Claude execution
  - `false`: Executes Claude in the current directory

**RIVER_GIT_BASE_BRANCH**
- Base branch for creating worktrees
- Default: `main`
- Example values: `main`, `master`, `develop`

**RIVER_GIT_AUTO_CLEANUP**
- Automatically cleanup worktrees after completion
- Values: `true`, `false`
- Default: `true`
- Behavior:
  - `true`: Removes worktree when River completes successfully
  - `false`: Preserves worktree for manual inspection

## Examples

```bash
# Debug mode
export RIVER_VERBOSITY=debug
river "Implement feature"

# Quiet mode, keep state file
export RIVER_SHOW_OUTPUT=false
export RIVER_AUTO_CLEANUP=false
river "Fix bug"

# Run in different directory
export RIVER_WORKDIR=/path/to/project
river "Refactor code"

# Disable real-time tool updates
export RIVER_SHOW_TOOL_UPDATES=false
river "Add tests"

# Disable worktree isolation, use develop branch
export RIVER_GIT_ENABLED=false
export RIVER_GIT_BASE_BRANCH=develop
river "Quick fix"

# Keep worktree for inspection
export RIVER_GIT_AUTO_CLEANUP=false
river "Debug issue"
```