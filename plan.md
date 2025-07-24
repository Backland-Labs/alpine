# River Multi-Agent - Simplified Implementation Plan

## Overview
Enable River to run multiple agents in different codebases simultaneously. Keep it dead simple - just spawn multiple River processes and let them run.

## Core Command

```bash
# Run multiple agents
river multi \
  ~/code/frontend "upgrade to React 18" \
  ~/code/backend "add authentication" \
  ~/code/mobile "implement push notifications"

# Continue a paused agent (in any project directory)
river --continue
```

## Implementation Plan

### P0: Minimal Multi-Agent Support

#### Task 1: Add --continue Flag ✅ IMPLEMENTED
**Test Cases:**
- `river --continue` resumes from existing state file
- Works if state exists with status != "completed"
- Shows clear error if no state file found

**Implementation:**
- Add `--continue` flag to root command
- When set, skip task argument requirement
- Equivalent to `--no-plan --no-worktree` but cleaner

**Status:** Implemented in v0.2.0
- Added --continue flag to root command
- Flag automatically sets --no-plan and --no-worktree
- Validates that no task argument is provided with --continue
- Checks for existing state file and shows appropriate error if missing
- Comprehensive test suite added

#### Task 2: Add Multi Command ✅ IMPLEMENTED
**Test Cases:**
- `river multi <path> <task> <path> <task>...` parses pairs
- Each pair spawns a River process
- Parent waits for all children to complete

**Implementation:**
- Simple argument parsing (path, task, path, task...)
- Spawn each River process with `os/exec`
- No complex monitoring - just wait for completion

**Status:** Implemented in v0.3.0
- Added multi subcommand to root command
- Implemented argument validation and parsing
- SpawnRiverProcesses function handles parallel execution
- Comprehensive test suite with mocked river execution
- Follows the simple approach outlined in the plan

#### Task 3: Parallel Execution
**Test Cases:**
- All agents start simultaneously
- Output is interleaved (that's OK)
- Parent exits when all children done

**Implementation:**
- Start all processes without waiting
- Use WaitGroup to track completion
- Let stdout/stderr pass through naturally

### P1: Basic Usability

#### Task 4: Prefixed Output ✅ IMPLEMENTED
**Test Cases:**
- Each agent's output prefixed with project name
- `[frontend] Starting upgrade to React 18...`
- Colors to distinguish agents (if terminal supports)

**Implementation:**
- Wrap stdout/stderr with prefix writer
- Extract project name from path
- Add basic ANSI colors

**Status:** Implemented in v0.3.0
- Added PrefixWriter to wrap stdout/stderr streams
- Automatic project name extraction from directory paths
- Color support with cycling through 6 distinct ANSI colors
- Integrated with SpawnRiverProcesses for automatic prefixing
- Comprehensive test coverage for prefix formatting and color assignment
- Thread-safe implementation for concurrent output streams

#### Task 5: Ctrl+C Handling
**Test Cases:**
- Ctrl+C stops all agents gracefully
- Each agent preserves state
- Can resume with `river --continue`

**Implementation:**
- Parent catches SIGINT
- Forwards signal to all children
- Waits for graceful shutdown

### P2: Polish

#### Task 6: Simple Status Check
**Test Cases:**
- `river status` shows any River processes
- Just lists: project, PID, current task
- No complex state tracking needed

**Implementation:**
- Write PID file when River starts
- Status command finds PID files
- Show basic ps-style output

## What We're NOT Building
- Complex state synchronization
- Inter-agent communication  
- Fancy progress bars
- Web dashboards
- Configuration files
- Resource limits

## Success Criteria
- [ ] Can run multiple agents with one command
- [ ] Each runs in its own directory
- [ ] `river --continue` resumes work
- [ ] Ctrl+C stops all agents cleanly
- [ ] Output is readable (prefixed)
- [ ] No feature creep

## Technical Notes

1. **Simplest Thing That Works**: Just spawn processes
2. **No New Dependencies**: Use Go's standard library
3. **No State Coordination**: Each agent is independent
4. **No Breaking Changes**: Single agent mode unchanged

## Example Usage

```bash
# Start three agents
river multi \
  ~/web "add dark mode" \
  ~/api "add caching" \
  ~/cli "update docs"

# Output (interleaved but prefixed)
[web] Starting: add dark mode
[api] Starting: add caching  
[cli] Starting: update docs
[web] Created plan.md
[api] Created plan.md
[web] Implementing theme provider...
[cli] Updating README.md...
...

# If interrupted, resume individually
cd ~/web && river --continue
```

This is the minimal viable feature - run multiple Rivers at once. Simple, useful, done.