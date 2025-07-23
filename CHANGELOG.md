# Changelog

All notable changes to the River CLI project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2025-07-23

### Added

#### Claude TODO Visibility
- **Real-time TODO tracking** - Shows Claude's current task instead of generic "Executing Claude" spinner
- **PostToolUse hook integration** - Monitors TodoWrite tool calls via Claude Code hooks
- **Graceful fallback** - Falls back to spinner if hook setup fails
- **Configurable display** - Can be disabled via `RIVER_SHOW_TODO_UPDATES=false`
- **File-based monitoring** - Efficient file polling system for task updates
- **Comprehensive test coverage** - Tests for hooks, monitoring, and display functionality

### Changed

#### Configuration
- **Added ShowTodoUpdates option** - New configuration field with environment variable support
- **Updated state file location** - Changed default location to `.claude/river/claude_state.json`

#### Claude Executor
- **Enhanced Execute method** - Now supports TODO monitoring mode alongside traditional spinner
- **Added hook setup functionality** - Creates `.claude/settings.local.json` with PostToolUse configuration
- **Improved error handling** - Graceful degradation when hook setup fails

### Technical Implementation
- **Hook system** in `internal/claude/hooks.go` - Manages Claude Code hook configuration
- **TODO monitor** in `internal/claude/todo_monitor.go` - File-based task monitoring
- **Hook script** in `internal/hooks/todo-monitor.sh` - Bash script for TodoWrite processing
- **Display functions** in `internal/output/color.go` - Real-time task update display

## [0.4.0] - 2025-07-23

### Added

#### Bare Execution Mode
- **Bare execution mode** allowing `river --no-plan --no-worktree` without task description
- **Automatic state continuation** from existing `claude_state.json` when present
- **Fresh workflow initialization** with `/ralph` command when no state exists
- **Advanced flag validation** requiring both `--no-plan` and `--no-worktree` flags

#### Enhanced CLI
- **Flexible argument handling** supporting bare mode execution
- **Comprehensive error messages** for invalid flag combinations
- **State-aware workflow resumption** for interrupted tasks

#### Testing Coverage
- **Complete integration test suite** for bare mode scenarios
- **State persistence validation** across workflow interruptions
- **Error handling verification** for invalid configurations
- **Workflow continuation testing** from existing state files

### Fixed

#### Claude Working Directory Execution (#7)
- **Fixed Claude commands executing in wrong directory** - Claude now properly executes in the worktree directory instead of the original repository
- **Added working directory validation** - Validates directory exists and is accessible before execution
- **Enhanced error handling** - Graceful fallback when working directory detection fails
- **Added comprehensive tests** - Unit and integration tests verify correct directory isolation

### Changed

#### Claude Executor
- **Command builder now sets working directory** - Uses `os.Getwd()` to ensure Claude inherits current directory
- **Added directory validation method** - Checks directory existence and permissions before execution
- **Improved logging** - Debug and warning logs for directory operations

### Technical Implementation
- **CLI argument validation** in `internal/cli/root.go`
- **Task description handling** in `internal/cli/workflow.go`
- **Workflow engine state management** in `internal/workflow/workflow.go`
- **Integration tests** in `test/integration/bare_mode_test.go`

### Breaking Changes
- None. All changes are backward compatible with existing workflows.

---

## [0.3.0] - 2025-07-22

### Added

#### Git Worktree Support
- **Complete git worktree integration** for isolated task execution
- **WorktreeManager interface** with CLI-based implementation using system `git`
- **Automatic worktree creation** at workflow start with branch isolation
- **Task name sanitization** for safe branch and directory naming
- **Configurable worktree behavior** via environment variables and CLI flags

#### New Configuration Options
- `RIVER_GIT_ENABLED` - Enable/disable git worktree support (default: true)
- `RIVER_GIT_BASE_BRANCH` - Base branch for creating worktrees (default: "main") 
- `RIVER_GIT_AUTO_CLEANUP` - Auto-cleanup worktrees after completion (default: true)
- `--no-worktree` CLI flag to disable worktree creation for individual runs

#### New Packages
- `internal/gitx/` - Git worktree management with interfaces, manager, and utilities
- `internal/gitx/mock/` - Mock implementations for testing
- `test/e2e/` - End-to-end integration tests with real git operations

#### Enhanced Testing
- **Comprehensive unit tests** for all worktree functionality
- **Integration tests** with temporary git repositories  
- **End-to-end tests** validating complete worktree workflows
- **Mock implementations** for isolated testing scenarios

### Changed

#### Workflow Engine
- **Engine constructor** now accepts `WorktreeManager` dependency
- **State file location** dynamically updated to worktree path when enabled
- **Working directory management** automatically switches to worktree
- **Dependency injection** throughout the codebase for testability

#### CLI Integration  
- **Root command** enhanced with worktree flag support
- **Dependency creation** updated to instantiate `WorktreeManager`
- **Configuration loading** extended with git-specific settings

### Technical Details

#### Worktree Architecture
- Worktrees created as `../repo-river-<task>` alongside main repository
- Branches follow `river/<sanitized-task-name>` naming convention
- Task names sanitized using URL-safe slug generation
- Branch name collisions handled with numeric suffixes

#### Integration Points
- Pre-workflow: Create worktree and change working directory
- During workflow: All operations execute within worktree context
- Post-workflow: Optional cleanup removes worktree and prunes references

#### Error Handling
- Comprehensive error propagation from git operations
- Graceful fallback when worktree creation fails
- Proper cleanup even when workflow encounters errors

### Files Changed
- **25 files modified** with 2,637 additions and 138 deletions
- **New packages**: `internal/gitx/` with full implementation
- **Enhanced tests**: Comprehensive coverage across unit, integration, and e2e levels
- **Configuration**: Extended config system with git-specific options
- **CLI**: Enhanced with worktree flags and dependency injection

### Breaking Changes
- None. All changes are backward compatible with existing workflows.

---

## Previous Versions

### [0.2.0] - Previous Release
- Complete Go implementation of River CLI
- State-driven workflow architecture
- Claude Code integration
- Linear dependency removal