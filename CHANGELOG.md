# Changelog

All notable changes to the Alpine CLI project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking Changes

#### Removed --continue Flag
- **BREAKING: Removed `--continue` flag** - This flag has been removed to simplify the CLI interface
- **Migration path** - Use `alpine --no-plan --no-worktree` instead of `alpine --continue`
- **Rationale** - The `--continue` flag was redundant as it was equivalent to combining `--no-plan` and `--no-worktree` flags
- **Impact** - Scripts or workflows using `--continue` must be updated to use the new flag combination

### Added

#### Enhanced Logging System with Uber Zap
- **Dual-logger architecture** - Automatic upgrade from simple logger to Uber Zap when configured
- **Structured logging** - Full support for contextual fields and JSON output in production
- **Environment-driven configuration** - All logging configured via environment variables:
  - `ALPINE_LOG_LEVEL` - Set logging level (debug, info, error)
  - `ALPINE_LOG_FORMAT` - Output format (json for production, console for development)
  - `ALPINE_LOG_CALLER` - Include file:line information
  - `ALPINE_LOG_STACKTRACE` - Configure stack trace verbosity
  - `ALPINE_LOG_SAMPLING` - Enable high-volume log sampling
- **Maximum verbosity** - Debug mode provides comprehensive execution details
- **Performance optimized** - Minimal overhead with buffering and sampling capabilities
- **Backward compatible** - Falls back to simple logger when Zap is not configured
- **HTTP request logging** - Added middleware for REST API request/response logging
- **Specialized loggers** - Created dedicated loggers for workflows and server operations

#### Parallel Plan Generation with Git Worktrees
- **New `--worktree` flag** - Enable isolated plan generation using Git worktrees
- **Prevents file conflicts** - Multiple `alpine plan` commands can run concurrently without overwriting `plan.md`
- **Cleanup control** - `--cleanup` flag (default: true) to manage worktree lifecycle
- **Task isolation** - Each plan generation gets its own isolated filesystem environment
- **Example usage**:
  ```bash
  # Generate plans in parallel for multiple issues
  alpine plan --worktree gh-issue https://github.com/owner/repo/issues/123 &
  alpine plan --worktree gh-issue https://github.com/owner/repo/issues/124 &
  wait
  ```

#### Real-time Output Streaming
- **Live Claude Code output** - Stream execution output in real-time instead of buffering
- **Event emitters** - Structured event emission for workflow lifecycle
- **AG UI protocol integration** - Standardized event format for UI consumption
- **Stream writers** - Efficient handling of large output streams

#### REST API for Programmatic Workflow Management
- **Comprehensive REST API** - 10 endpoints for complete workflow management via HTTP
- **Agent Management** - `GET /agents/list` to retrieve available agents
- **Workflow Operations** - Start workflows from GitHub issues via `POST /agents/run`
- **Run Management** - List, retrieve, and cancel workflow runs via `/runs` endpoints
- **Real-time Monitoring** - Run-specific Server-Sent Events at `/runs/{id}/events`
- **Plan Workflow** - Retrieve and approve execution plans via `/plans` endpoints
- **Health Monitoring** - `/health` endpoint for service status checks
- **WorkflowEngine Integration** - Clean abstraction layer connecting API to Alpine's workflow engine
- **Thread-Safe State Management** - In-memory storage with mutex protection for concurrent access
- **87% Test Coverage** - Comprehensive unit and integration tests following TDD methodology

#### HTTP Server with Server-Sent Events (SSE)
- **New HTTP server mode** - Run `alpine --serve` to start a standalone HTTP server
- **Server-Sent Events endpoint** - Real-time event streaming at `/events` endpoint
- **Configurable port** - Use `--port` flag to specify custom port (default: 3001)
- **Standalone operation** - Server can run without requiring a task description
- **Concurrent execution** - Server runs alongside normal workflow when used with tasks
- **Graceful shutdown** - Proper cleanup on interrupt signals
- **Test-Driven Development** - Full TDD implementation with comprehensive test coverage

#### GitHub Issue Integration for Plan Generation (#18)
- **New `gh-issue` subcommand** - Generate implementation plans directly from GitHub issues
- **Command syntax** - `alpine plan gh-issue <github-issue-url>` fetches issue details via `gh` CLI
- **Claude Code support** - Works with both Gemini (default) and Claude Code (`--cc` flag)
- **Comprehensive error handling** - Clear messages for missing `gh` CLI or API failures
- **Full test coverage** - Command structure, integration, and documentation tests
- **Updated documentation** - README.md and specs updated with usage examples and requirements

### Fixed

#### Server-Sent Events and Containerized Workflow Issues
- **Fixed context cancellation issue in containerized workflows** - Resolved race conditions and context handling in Docker environments
- **Fixed deadlock in workflow mutex handling** - Improved synchronization and reduced lock contention
- **Fixed race condition in workflow instance creation** - Enhanced thread-safety for concurrent workflow management
- **Fixed embedded prompt template usage** - Replaced `/make_plan` slash command with proper embedded templates for better reliability

#### Enhanced Branch Publishing for Server Workflows
- **Added GitHub token authentication for branch publishing** - Secure token-based authentication for automated branch operations
- **Enforced branch publishing for server workflows** - Server mode now consistently creates and publishes branches
- **Skip worktree creation in favor of branches** - Streamlined server workflow by using direct branch operations instead of worktrees

#### Allow --serve flag without task description
- **Fixed server-only mode** - The `--serve` flag now works standalone without requiring a task
- **Updated validation logic** - Root command now properly validates --serve usage
- **Server-only workflow** - Added dedicated server mode that doesn't require workflow engine
- **Improved error messages** - Clear error when trying to use --serve with a task description

#### State File Path Mismatch (#16)
- **Fixed Alpine hanging after task completion** - Alpine now correctly monitors state file updates
- **Removed ALPINE_STATE_FILE environment variable** - State file location is now fixed at `agent_state/agent_state.json`
- **Resolved path mismatch** between Alpine's monitoring location and Claude's write location
- **Updated all tests and documentation** to reflect the fixed state file path

### Changed

#### HTTP Server Architecture
- **Extended server package** - Enhanced existing `internal/server` with REST API handlers
- **Data models** - Added `Agent`, `Run`, and `Plan` structs with full validation in `internal/server/models.go`
- **WorkflowEngine abstraction** - Created interface for clean separation between API and workflow execution
- **Event broadcasting** - Enhanced SSE system to support both global and run-specific event streams
- **Documentation updates** - Updated `CLAUDE.md`, `specs/server.md`, and `specs/cli-commands.md` with REST API documentation

#### Configuration
- **Removed StateFile customization** - State file path is no longer configurable via environment variable
- **Simplified state management** - Both worktree and bare modes use the same relative path

## [0.5.0] - 2025-07-23

### Added

#### Claude TODO Visibility
- **Real-time TODO tracking** - Shows Claude's current task instead of generic "Executing Claude" spinner
- **PostToolUse hook integration** - Monitors TodoWrite tool calls via Claude Code hooks
- **Graceful fallback** - Falls back to spinner if hook setup fails
- **Configurable display** - Can be disabled via `ALPINE_SHOW_TODO_UPDATES=false`
- **File-based monitoring** - Efficient file polling system for task updates
- **Comprehensive test coverage** - Tests for hooks, monitoring, and display functionality

### Changed

#### Configuration
- **Added ShowTodoUpdates option** - New configuration field with environment variable support
- **Updated state file location** - Changed default location to `agent_state/agent_state.json`

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
- **Bare execution mode** allowing `alpine --no-plan --no-worktree` without task description
- **Automatic state continuation** from existing `agent_state.json` when present
- **Fresh workflow initialization** with `/run_implementation_loop` command when no state exists
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
- **Removed `--continue` flag** - Use `alpine --no-plan --no-worktree` instead. See breaking changes section above for migration details.

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
- `ALPINE_GIT_ENABLED` - Enable/disable git worktree support (default: true)
- `ALPINE_GIT_BASE_BRANCH` - Base branch for creating worktrees (default: "main") 
- `ALPINE_GIT_AUTO_CLEANUP` - Auto-cleanup worktrees after completion (default: true)
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
- Worktrees created as `../repo-alpine-<task>` alongside main repository
- Branches follow `alpine/<sanitized-task-name>` naming convention
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
- **Removed `--continue` flag** - Use `alpine --no-plan --no-worktree` instead. See breaking changes section above for migration details.

---

## Previous Versions

### [0.2.0] - Previous Release
- Complete Go implementation of Alpine CLI
- State-driven workflow architecture
- Claude Code integration
- Linear dependency removal