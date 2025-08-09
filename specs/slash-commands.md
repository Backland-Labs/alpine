# Slash Commands Specification

## Overview

Alpine uses slash commands to coordinate different phases of AI-assisted development workflows. These commands integrate with Claude Code to provide structured automation for planning, implementation, and verification phases. Slash commands are executed within Claude Code sessions and guide the workflow state transitions.

## Core Workflow Commands

### `/make_plan <task-description>`

**Purpose**: Generates a detailed implementation plan based on a task description.

**Usage Context**: 
- Initial workflow step when plan generation is enabled
- Automatically triggered when Alpine is run with planning enabled (default behavior)
- Can be used to re-plan or update existing plans

**Behavior**:
- Creates a `plan.md` file in the project root with structured implementation tasks
- Analyzes existing codebase and specifications from `/specs` directory
- Breaks down the task into atomic, TDD-friendly implementation units
- Sets up the workflow to transition to `/start` for implementation

**Example**:
```
/make_plan Implement user authentication with JWT tokens
```

**Output**: Creates a structured plan.md file with features broken down into testable tasks.

**State Transition**: After completion, workflow moves to `/start` or `/run_implementation_loop`

### `/start <task-description>`

**Purpose**: Begins direct implementation without generating a plan.

**Usage Context**:
- Used when Alpine is run with `--no-plan` flag
- Bare mode initialization when no existing state file exists
- Direct implementation of simple tasks that don't require formal planning

**Behavior**:
- Skips plan generation and moves directly to implementation
- Initializes workflow state for immediate development work
- Suitable for bug fixes, small features, or continuing existing work

**Example**:
```
/start Fix database connection timeout issue
```

**State Transition**: Typically transitions to `/run_implementation_loop` or `/continue`

### `/continue`

**Purpose**: Continues workflow execution from the current state.

**Usage Context**:
- Resuming interrupted workflows
- Continuing multi-step implementation processes  
- Used by the workflow engine when state indicates more work remains

**Behavior**:
- Loads current workflow state from `agent_state.json`
- Continues from the last completed step
- Maintains workflow context and progress tracking

**Example**:
```
/continue
```

**State Transition**: Depends on current workflow state and remaining tasks

### `/run_implementation_loop`

**Purpose**: Implements features from `plan.md` using Test-Driven Development methodology.

**Usage Context**:
- Primary implementation command after plan generation
- Iterative development cycles following RED-GREEN-REFACTOR pattern
- Continues until all planned features are implemented

**Behavior**:
- Analyzes `plan.md` to identify unimplemented features
- Selects highest priority feature for implementation
- Follows TDD cycle: write tests first, implement minimal code, refactor
- Updates `plan.md` with implementation status
- Creates commits with structured messages
- Updates workflow state to indicate progress or completion

**Key Features**:
- **Feature Selection**: Analyzes dependencies and complexity to choose optimal implementation order
- **TDD Enforcement**: Requires tests before implementation code
- **Minimal Implementation**: Focuses on making tests pass without over-engineering
- **Progress Tracking**: Updates plan.md status and creates detailed commit messages
- **Quality Assurance**: Includes build verification and test execution

**Example Implementation Cycle**:
1. RED Phase: Write focused tests for core business logic
2. GREEN Phase: Implement minimal code to make tests pass
3. REFACTOR Phase: Improve code quality and apply design patterns
4. Update plan.md status to "implemented"
5. Create structured commit with technical notes

**State Transition**: 
- Returns to `/run_implementation_loop` if more features remain
- Transitions to `/verify_plan` when all features are implemented

### `/verify_plan`

**Purpose**: Verifies that all features in `plan.md` have been successfully implemented and creates a pull request.

**Usage Context**:
- Final verification step after all implementation cycles
- Quality assurance and completeness checking
- Automated when all plan.md features are marked as implemented

**Behavior**:
- Reviews `plan.md` for completeness and accuracy
- Verifies implementation status of all features
- Runs comprehensive tests and build verification
- Creates GitHub pull request if not on main/develop branch
- Sets workflow status to "completed"

**Example**:
```
/verify_plan
```

**State Transition**: Sets workflow status to "completed"

## Custom Commands

Alpine provides custom Claude commands defined in `.claude/commands/` directory:

### `/docker_debug`

**Purpose**: Validates Docker container functionality and identifies deployment issues.

**Allowed Tools**: 
- Bash commands (grep, ls, tree, git, find, curl, docker)

**Usage Context**:
- Debugging Docker deployments
- Validating containerized application functionality
- Monitoring application logs and behavior

**Behavior**:
- Analyzes project technical constraints and dependencies
- Maps application architectural patterns
- Executes Docker commands and monitors container logs
- Sends test requests to verify functionality
- Provides detailed error analysis if issues are found

**Key Features**:
- **Constraint Analysis**: Reviews runtime requirements, environment variables, dependencies
- **Pattern Analysis**: Maps project structure, design patterns, middleware chains
- **Live Testing**: Executes containers and performs integration testing
- **Error Reporting**: Provides detailed error analysis with recommended fixes

**Limitations**: Read-only analysis - does not modify source code directly

## Workflow State Management

Slash commands work with Alpine's state management system through `agent_state.json`:

### State File Structure
```json
{
  "current_step_description": "Brief description of completed work",
  "next_step_prompt": "Next command to execute",
  "status": "running" | "completed"
}
```

### Command Transitions

**Planning Phase**:
- `/make_plan` → `/run_implementation_loop`

**Implementation Phase**:
- `/run_implementation_loop` → `/run_implementation_loop` (more features remain)
- `/run_implementation_loop` → `/verify_plan` (all features implemented)

**Verification Phase**:
- `/verify_plan` → workflow completion (status: "completed")

**Direct Implementation**:
- `/start` → `/run_implementation_loop` or `/continue`

## Integration with Claude Code

### Allowed Tools

Different commands have access to different tool sets:

**Planning Commands** (`/make_plan`):
- Read-only tools: Read, Grep, Glob, LS
- Web research: WebSearch, WebFetch
- Context tools: mcp__context7__*

**Implementation Commands** (`/run_implementation_loop`, `/start`, `/continue`):
- File operations: Read, Write, Bash
- Git operations: git commands through Bash
- Build tools: Language-specific build and test commands
- Development tools: Linting, formatting, debugging

**Debug Commands** (`/docker_debug`):
- System tools: Bash (grep, ls, tree, git, find)
- Container tools: docker commands
- Network tools: curl for API testing

### System Prompts

Each command category uses specialized system prompts:

- **Planning**: Technical Product Manager persona focused on analysis and plan creation
- **Implementation**: Senior Software Engineer persona following TDD methodology  
- **Debug**: Cloud-native specialist focused on containerization and deployment

## Command Usage Patterns

### Sequential Workflow
```bash
# Full workflow with planning
alpine "Implement user authentication"
# Internally executes: /make_plan → /run_implementation_loop → /verify_plan
```

### Direct Implementation  
```bash
# Skip planning phase
alpine --no-plan "Fix login bug"
# Internally executes: /start → /run_implementation_loop
```

### Bare Mode Continuation
```bash
# Continue from existing state
alpine --no-plan --no-worktree
# Internally executes: /continue or /start (if no state exists)
```

### Plan-Only Mode
```bash
# Generate plan without implementation
alpine plan "Add caching layer"
# Internally executes: /make_plan only
```

## Error Handling

### Command Failures
- Failed commands maintain workflow state for recovery
- Error messages include context for debugging
- Workflow can be resumed from last successful state

### State Recovery
- Interrupted workflows preserve state in `agent_state.json`
- Commands validate state consistency before execution
- Invalid states are logged with recovery suggestions

## Best Practices

### Command Selection
- Use `/make_plan` for complex features requiring analysis
- Use `/start` for simple bug fixes or direct implementation
- Use `/continue` to resume interrupted work
- Let `/run_implementation_loop` handle iterative development

### State Management
- Monitor `agent_state.json` for workflow progress
- Preserve state files during development
- Clean up completed workflows to avoid confusion

### Integration Testing
- Use `/docker_debug` for deployment validation
- Combine with REST API for programmatic workflow control
- Leverage worktree isolation for parallel development

## Technical Implementation

### Command Parsing
- Commands are parsed by Claude Code's built-in slash command system
- Alpine provides command definitions in `.claude/commands/` directory
- Custom commands specify allowed tools and execution context

### State Synchronization
- Commands update `agent_state.json` to coordinate workflow transitions
- State file changes trigger workflow engine state transitions
- File modification times are monitored for state update detection

### Tool Restrictions
- Each command specifies allowed tools for security and functionality
- Tool access is enforced by Claude Code's execution environment
- Restricted tools prevent unintended system modifications

This specification provides comprehensive documentation of Alpine's slash command system, enabling effective workflow automation and AI-assisted development coordination.
EOF < /dev/null