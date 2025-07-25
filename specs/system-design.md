# System Design Specification

This document defines the system design for the Alpine project, covering architecture, project structure, configuration, and state management.

## 1. Architecture

### 1.1. Single Binary

The application will be distributed as a single, self-contained executable with no external runtime dependencies. All functionality is compiled directly into the binary.

**Benefits:**
- Simple deployment - just copy and run
- No dependency conflicts
- Predictable behavior across environments

### 1.2. Dependencies

**Core Principles:**
- Minimize third-party dependencies
- Prefer standard library solutions
- Audit all dependencies for necessity

**Allowed Dependencies:**
- CLI framework: `github.com/spf13/cobra`

## 2. Project Structure

### 2.1. Directory Layout

This is the project layout after consolidation of the specs:
```
alpine/
├── cmd/
│   └── alpine/
│       └── main.go
├── internal/
│   ├── cli/
│   ├── config/
│   ├── core/
│   ├── gitx/
│   ├── hooks/
│   ├── logger/
│   ├── output/
│   ├── performance/
│   ├── quality/
│   ├── validation/
│   └── workflow/
├── specs/
│   ├── system-design.md
│   ├── claude-code-hooks.md
│   ├── cli-commands.md
│   ├── code-quality.md
│   ├── error-handling.md
│   ├── gemini-cli.md
│   └── troubleshooting.md
├── agent_state/
│   └── agent_state.json
├── .claude/
├── .gemini/
├── .git/
├── .github/
├── prompts/
├── release/
├── scripts/
├── test/
├── .gitignore
├── go.mod
├── go.sum
├── README.md
└── CLAUDE.md
```

### 2.2. Naming Conventions

#### Go Files
- Use lowercase with underscores for multi-word files: `state_manager.go`
- Test files follow Go convention: `<name>_test.go`
- Main package files should be minimal, typically just `main.go`

#### Packages
- Package names are short, lowercase, singular nouns
- Avoid stuttering: prefer `executor.Run()` over `executor.ExecuteCommand()`
- Internal packages go under `internal/` to prevent external imports

#### Directories
- Use lowercase, singular nouns for package directories
- Group related functionality together
- Keep directory nesting shallow (max 3 levels deep)

#### Documentation Files
- Specifications use kebab-case: `system-design.md`
- All specs go in the `specs/` directory
- README files are uppercase: `README.md`, `CLAUDE.md`

#### Configuration and State
- State files use snake_case: `agent_state.json`
- Configuration via environment variables: `ALPINE_<SETTING>`
- Runtime directories use snake_case: `agent_state/`

### 2.3. File Organization Principles

#### Separation of Concerns
- `cmd/`: Entry points only, minimal logic
- `internal/`: All application logic, organized by domain
- `specs/`: Living documentation, kept in sync with code

#### Package Boundaries
- Each package has a clear, single responsibility
- Dependencies flow inward (CLI → executor → state)
- Avoid circular dependencies between packages

#### Test Organization
- Unit tests live alongside implementation files
- Integration tests can go in a `_test` package suffix
- Test fixtures and data in `testdata/` directories

#### Generated Files
- Generated files should have a `_gen.go` suffix
- Include generation commands in comments
- Never edit generated files directly

### 2.4. Special Files

#### CLAUDE.md
- Project-specific instructions for Claude Code
- Kept at repository root
- Contains context and conventions for AI assistance

#### agent_state.json
- Runtime state file created by Alpine
- Located in `agent_state/` directory
- Never committed to version control

#### Worktree Directories
- Created under `.git/worktrees/`
- Named with pattern `alpine-task-<timestamp>`
- Cleaned up based on configuration

### 2.5. Import Organization

Go imports should be organized in groups:
1. Standard library imports
2. Third-party imports
3. Internal project imports

Example:
```go
import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    
    "alpine/internal/executor"
    "alpine/internal/state"
)
```

## 3. Configuration

Alpine configuration controls runtime behavior and output settings. Configuration is set via environment variables.

### 3.1. Configuration Options

#### Execution Settings

**ALPINE_WORKDIR**
- Working directory for Claude execution
- Default: Current directory
- Must be an absolute path

#### Output Settings

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

#### State File Settings

**State File Location**
- Fixed at `agent_state/agent_state.json`
- Directory created automatically when workflow starts
- No longer configurable via environment variable

**ALPINE_AUTO_CLEANUP**
- Delete state file on successful completion
- Values: `true`, `false`
- Default: `true`

### 3.2. Examples

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

## 4. State Management

Alpine uses a JSON state file (`agent_state.json`) to track workflow progress and enable iteration between Claude Code executions.

### 4.1. State File Location

- Fixed location: `agent_state/agent_state.json`
- Directory created automatically when workflow starts
- No longer configurable via environment variable

#### Location Behavior by Mode

- **Worktree Mode (default)**: State file is created in the worktree directory at `agent_state/agent_state.json`
- **Bare Mode (`--no-worktree`)**: State file is created at `agent_state/agent_state.json` in the current directory

### 4.2. Schema

```json
{
  "current_step_description": "string",
  "next_step_prompt": "string",
  "status": "string"
}
```

### 4.3. Fields

#### current_step_description
- Human-readable description of what was just completed
- Updated by Claude Code during execution
- Example: `"Implemented task 1 from plan.md"`

#### next_step_prompt
- The command/prompt to execute in the next iteration
- Set by Claude Code at the end of each step
- Common values: `/run_implementation_loop`, `/continue`, or custom prompts
- Empty string or missing when workflow is complete

#### status
- Workflow state indicator
- Valid values:
  - `"running"` - Actively executing
  - `"completed"` - Workflow finished

### 4.4. State Transitions

1. **Initialize**: Create file with initial prompt and status `"running"`
2. **Iterate**: Claude Code updates all fields during execution
3. **Complete**: When status becomes `"completed"`, workflow ends

#### Bare Mode Behavior

When running with `--no-plan --no-worktree`:
- Uses state file at `agent_state/agent_state.json`
- If state file exists: Continue from existing workflow
- If no state file exists: Initialize new workflow with `/run_implementation_loop`

### 4.5. File Operations

- **Read**: Parse JSON, validate schema
- **Write**: Pretty-print JSON with 2-space indentation
- **Watch**: Monitor file for changes during Claude execution

### 4.6. Error Handling

- Missing file: Create new workflow
- Invalid JSON: Report error and exit
- Missing required fields: Report error and exit
