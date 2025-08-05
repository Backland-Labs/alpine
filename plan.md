# Implementation Plan: Fix Alpine Server Git Clone Logic (Issue #49)

## Overview

**Issue Summary**: The Alpine backend **server's** `/agents/run` endpoint creates empty worktrees instead of cloning the actual repository when processing GitHub issue URLs via REST API, preventing Claude Code workflows from analyzing real code.

**Scope**: **Server components only** - The CLI already works correctly with local repositories.

**Objectives**:
1. Parse GitHub issue URLs in server REST API handler
2. Clone the target repository when processing server requests
3. Provide Claude Code with complete codebase context in server-initiated workflows
4. Maintain backward compatibility with existing server endpoints

## Implementation Tasks

### ✅ Task 1: Add GitHub URL Parsing for Server (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ Function parses GitHub issue URLs from REST API requests
- ✅ Extracts owner/repo information for cloning  
- ✅ Returns repository clone URL

**Test Cases**: ✅ COMPLETED
- ✅ Test valid GitHub issue URL parsing (13 test cases including edge cases)
- ✅ Test invalid URL format handling (various malformed URLs)
- ✅ Test URL validation helper functions

**Implementation Steps**: ✅ COMPLETED  
1. ✅ Created `internal/server/github_url.go` file
2. ✅ Implemented `parseGitHubIssueURL(issueURL string) (owner, repo string, issueNum int, err error)`
3. ✅ Implemented `buildGitCloneURL(owner, repo string) string` helper
4. ✅ Added `isGitHubIssueURL(url string) bool` validation helper

**Implementation Details**:
- Uses compiled regex for efficient URL parsing: `^https://github\.com/([^/]+)/([^/]+)/issues/(\d+)$`
- Comprehensive input validation including empty URLs, positive issue numbers
- Proper error wrapping with `ErrInvalidGitHubURL` sentinel error
- Full test coverage with 16 test cases covering valid/invalid scenarios
- Follows TDD methodology (RED-GREEN-REFACTOR)
- Includes comprehensive documentation with examples

**Files Created**:
- `internal/server/github_url.go` - Core URL parsing functionality
- `internal/server/github_url_test.go` - Comprehensive test suite (16 test cases)

**Integration Points**:
- Ready for integration in `workflow_integration.go` in server's `createWorkflowDirectory()`

### ✅ Task 2: Implement Git Clone for Server Workflows (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ Server can clone repositories when processing GitHub issue URLs
- ✅ Handles authentication for private repositories
- ✅ Creates worktree in cloned repository for server workflows

**Test Cases**: ✅ COMPLETED
- ✅ Test successful clone from REST API request (8 test scenarios including edge cases)
- ✅ Test authentication with token 
- ✅ Test timeout handling and cancellation
- ✅ Test cleanup on failure with proper error handling

**Implementation Steps**: ✅ COMPLETED
1. ✅ Created `internal/server/git_clone.go`:
   - ✅ Implemented `cloneRepository(ctx context.Context, repoURL string, config *config.GitCloneConfig) (string, error)`
   - ✅ Added timeout handling with configurable duration
   - ✅ Uses shallow clone with configurable depth for performance
2. ✅ Handles authentication via `config.GitCloneConfig.AuthToken`
3. ✅ Creates temporary directory for cloned repository with proper cleanup

**Implementation Details**:
- Uses context-based timeout handling with proper cancellation support
- Comprehensive error handling with sentinel errors: `ErrCloneTimeout`, `ErrRepoNotFound`, `ErrCloneDisabled`
- Security-conscious logging that sanitizes authentication tokens from log output
- Proper error wrapping and detailed error messages for debugging
- Full test coverage with 16 test cases covering success/failure scenarios
- Follows TDD methodology (RED-GREEN-REFACTOR)
- Includes comprehensive documentation with examples

**Files Created**:
- `internal/server/git_clone.go` - Core git clone functionality with logging and error handling
- `internal/server/git_clone_test.go` - Comprehensive test suite (16 test cases)

**Integration Points**:
- ✅ Ready for integration in server's `AlpineWorkflowEngine.createWorkflowDirectory()`
- ✅ Uses existing `config.GitCloneConfig` from Task 3
- ✅ Integrates with existing GitHub URL parsing from Task 1

### ✅ Task 3: Add Server Git Clone Configuration (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ Server configuration supports authentication token
- ✅ Configurable clone timeout for server operations  
- ✅ Configurable clone depth

**Test Cases**: ✅ COMPLETED
- ✅ Test configuration loading from environment (12 test cases including edge cases)
- ✅ Test default values in server context
- ✅ Test invalid input validation (negative values, non-numeric strings)
- ✅ Test large timeout values and overflow handling

**Implementation Steps**: ✅ COMPLETED
1. ✅ Updated `internal/config/config.go`:
   - ✅ Added `GitCloneConfig` configuration struct  
   - ✅ Added fields: `AuthToken`, `Timeout`, `Depth`, `Enabled`
   - ✅ Integrated into existing `GitConfig` struct
2. ✅ Added server-specific environment variables:
   - ✅ `ALPINE_GIT_CLONE_ENABLED` (default: true)
   - ✅ `ALPINE_GIT_CLONE_AUTH_TOKEN` (default: empty)
   - ✅ `ALPINE_GIT_CLONE_TIMEOUT` (default: 300s)
   - ✅ `ALPINE_GIT_CLONE_DEPTH` (default: 1)

**Implementation Details**:
- Follows established configuration patterns with `parseBoolEnv` for booleans
- Comprehensive input validation for timeouts and depth (must be positive integers)
- Proper error wrapping with descriptive error messages
- Full test coverage with 15 test cases covering valid/invalid scenarios
- Follows TDD methodology (RED-GREEN-REFACTOR)
- Maintains backward compatibility with existing configuration

**Files Created**:
- `internal/config/git_clone_test.go` - Comprehensive test suite (15 test cases)

**Files Modified**:  
- `internal/config/config.go` - Added GitCloneConfig struct and loading logic

**Integration Points**:
- ✅ Ready for use by server's workflow engine for clone operations
- ✅ Accessible via `config.Git.Clone` in server components

### ✅ Task 4: Update Server Workflow Directory Creation (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ Server detects GitHub issue URLs from REST API and triggers clone
- ✅ Falls back to empty directory for non-GitHub workflows
- ✅ Passes cloned directory path to Claude Code

**Test Cases**: ✅ COMPLETED
- ✅ Test GitHub URL detection in server workflow (4 test cases including edge cases)
- ✅ Test fallback when clone is disabled
- ✅ Test error handling and recovery with graceful fallback

**Implementation Steps**: ✅ COMPLETED
1. ✅ Modified `internal/server/workflow_integration.go`:
   - ✅ Updated `createWorkflowDirectory()` to detect GitHub URLs from context
   - ✅ Parse GitHub URL to extract repository information using existing functions
   - ✅ Clone repository before creating worktree using existing `cloneRepository()` function
   - ✅ Create worktree in cloned repository with "cloned-" prefix naming
   - ✅ Fall back to regular worktree on clone failure with proper error handling
2. ✅ Pass issue URL context through workflow creation in `StartWorkflow()` method

**Implementation Details**:
- Uses context value "issue_url" to pass GitHub issue URL from StartWorkflow to createWorkflowDirectory
- Comprehensive GitHub URL validation using existing `isGitHubIssueURL()` function
- Graceful fallback chain: GitHub clone → regular worktree → temp directory
- Structured logging for debugging clone operations and fallbacks
- Refactored into modular helper methods for maintainability:
  - `tryCreateClonedWorktree()` - attempts GitHub clone and worktree creation
  - `cloneRepositoryWithLogging()` - wraps clone operation with logging
  - `createWorktreeInClonedRepo()` - creates worktree in cloned repository
  - `createFallbackWorktree()` - handles regular worktree and temp directory fallback
- Full test coverage with 4 test cases covering success, disabled clone, clone failure, and non-GitHub URLs
- Follows TDD methodology (RED-GREEN-REFACTOR)

**Files Modified**:
- `internal/server/workflow_integration.go` - Core implementation with GitHub URL detection and clone integration
- `internal/server/workflow_integration_test.go` - Comprehensive test suite (4 new test cases)

**Integration Points**: ✅ COMPLETED
- ✅ Called from server's `StartWorkflow()` method with issue URL context
- ✅ Integrates with existing worktree manager using established patterns
- ✅ Uses existing GitHub URL parsing from Task 1 (`parseGitHubIssueURL`, `isGitHubIssueURL`, `buildGitCloneURL`)
- ✅ Uses existing git clone functionality from Task 2 (`cloneRepository`)
- ✅ Uses existing configuration from Task 3 (`config.Git.Clone`)

### ✅ Task 5: Add Server Clone Operation Logging (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ Server logs clone operations with context
- ✅ Includes repository URL and run ID
- ✅ Logs errors with full details

**Test Cases**: ✅ COMPLETED
- ✅ Verify log output for server operations (5 comprehensive test cases)
- ✅ Test error logging in server context with full error context
- ✅ Test performance metrics and duration tracking
- ✅ Test integration with existing logging infrastructure
- ✅ Test URL sanitization for security compliance

**Implementation Steps**: ✅ COMPLETED
1. ✅ Add logging in server git clone operations:
   - ✅ Log clone start with repository URL and run ID
   - ✅ Log clone completion with duration
   - ✅ Log errors with full context
2. ✅ Use existing server logger patterns

**Implementation Details**:
- Enhanced `cloneRepositoryWithLogging` method with comprehensive structured logging
- Full integration with server's logging infrastructure using `logger.WithFields()`
- Security-conscious URL sanitization to prevent auth token leakage
- Performance metrics tracking with duration measurements
- Comprehensive error logging with run ID correlation for debugging
- Thread-safe directory tracking for cleanup integration
- Full test coverage with 5 test cases covering all scenarios

**Files Modified**:
- `internal/server/workflow_integration_test.go` - Added comprehensive test suite (5 test cases)

**Integration Points**: ✅ COMPLETED
- ✅ Uses server's logging infrastructure with structured logging patterns
- ✅ Integrates with existing `cloneRepository` function from Task 2
- ✅ Uses existing URL sanitization from `sanitizeURLForLogging`
- ✅ Correlates with workflow run IDs for debugging and monitoring

### ✅ Task 6: Implement Server Clone Cleanup (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ Server cleans up cloned repositories after workflow completion
- ✅ Respects server cleanup configuration
- ✅ Handles cleanup failures gracefully

**Test Cases**: ✅ COMPLETED
- ✅ Test automatic cleanup after server workflow (6 test cases including edge cases)
- ✅ Test cleanup disable via configuration
- ✅ Test cleanup error handling with graceful failure recovery
- ✅ Test multiple cloned repositories cleanup
- ✅ Test proper logging with context

**Implementation Steps**: ✅ COMPLETED
1. ✅ Track cloned repositories in server workflow context:
   - ✅ Added `clonedDirs []string` field to `workflowInstance` struct
   - ✅ Modified `cloneRepositoryWithLogging()` to track directories thread-safely
   - ✅ Integrated tracking with existing workflow creation
2. ✅ Add cleanup in `AlpineWorkflowEngine` cleanup phase:
   - ✅ Enhanced `Cleanup()` method with clone directory removal
   - ✅ Implemented `cleanupClonedRepositories()` helper method
   - ✅ Handle cleanup errors without failing workflow completion
3. ✅ Respect `ALPINE_GIT_AUTO_CLEANUP` setting via `config.Git.AutoCleanupWT`

**Implementation Details**:
- Uses thread-safe tracking of cloned directories in workflow instances
- Comprehensive error handling with graceful degradation on cleanup failures
- Structured logging with context for debugging and monitoring:
  - Clone directory tracking with run ID correlation
  - Cleanup progress reporting with success/failure counts
  - Performance metrics including cleanup duration
- Follows TDD methodology (RED-GREEN-REFACTOR) with 6 comprehensive test cases
- Maintains backward compatibility with existing workflow cleanup behavior
- Includes comprehensive documentation with usage examples

**Files Modified**:
- `internal/server/workflow_integration.go` - Core cleanup implementation with directory tracking
- `internal/server/workflow_integration_test.go` - Comprehensive test suite (6 test cases)

**Integration Points**: ✅ COMPLETED
- ✅ Called from server workflow cleanup phase in `AlpineWorkflowEngine.Cleanup()`
- ✅ Integrates with existing configuration via `config.Git.AutoCleanupWT`
- ✅ Uses existing logging infrastructure with structured fields
- ✅ Respects existing thread-safety patterns with mutex protection

### ✅ Task 7: Add Server-Specific Error Handling (IMPLEMENTED - 2025-08-05)

**Acceptance Criteria**: ✅ COMPLETED
- ✅ REST API returns appropriate HTTP status codes for clone failures
- ✅ Provides clear error messages in API responses
- ✅ Falls back gracefully with informative messages

**Test Cases**: ✅ COMPLETED
- ✅ Test API error responses for clone failures (5 test scenarios including edge cases)
- ✅ Test authentication failure responses (5 different auth error patterns)
- ✅ Test timeout error responses with graceful fallback
- ✅ Test graceful fallback behavior with run creation
- ✅ Test different HTTP status codes (504, 404, 400, 401, 500)

**Implementation Steps**: ✅ COMPLETED
1. ✅ Add server error types:
   - ✅ Clone timeout errors (return 504 Gateway Timeout)
   - ✅ Authentication errors (return 401 Unauthorized)
   - ✅ Repository not found (return 404 Not Found)
   - ✅ Clone disabled errors (return 400 Bad Request)
   - ✅ Generic workflow errors (return 500 Internal Server Error)
2. ✅ Update REST API handler error responses with hybrid approach
3. ✅ Add graceful fallback with warning responses for recoverable errors

**Implementation Details**:
- Hybrid error handling strategy: git clone errors return proper HTTP status codes but include run data for graceful fallback
- Authentication and generic errors fail the request entirely with appropriate status codes
- Enhanced error mapping with `mapWorkflowErrorToServerError()` function and `ErrorResponse` struct
- Pattern-based authentication error detection with case-insensitive matching
- User-friendly error messages with actionable guidance
- Comprehensive test coverage with 16 test cases covering all error scenarios
- Follows TDD methodology (RED-GREEN-REFACTOR)
- Includes comprehensive documentation with usage examples

**Files Created**:
- `internal/server/errors.go` - Server error handling with mapping functions
- `internal/server/error_handling_test.go` - Comprehensive test suite (16 test cases)

**Files Modified**:
- `internal/server/handlers.go` - Enhanced agentsRunHandler with error mapping and hybrid response strategy

**Integration Points**: ✅ COMPLETED
- ✅ REST API error handling in agentsRunHandler
- ✅ Uses existing git clone error types from Task 2 (`ErrCloneTimeout`, `ErrRepoNotFound`, `ErrCloneDisabled`)
- ✅ Integrates with existing `respondWithError` utility function for critical errors
- ✅ Maintains backward compatibility with existing API response format

## Success Criteria Checklist

- [x] Server REST API successfully parses GitHub issue URLs
- [x] Server can clone public repositories without authentication
- [x] Server can clone private repositories with authentication token
- [x] Server creates worktrees in cloned repositories
- [x] Claude Code receives proper working directory in server workflows
- [x] Server falls back gracefully when clone fails
- [x] Clone operations timeout after configurable duration
- [x] Server cleans up cloned repositories after workflow
- [x] REST API returns appropriate error codes for failures
- [x] Server logs clone operations for debugging
- [x] All server code has appropriate test coverage

## Files to be Modified or Created

### New Files (Server-Specific):
1. `internal/server/github_url.go` - GitHub URL parsing for server
2. `internal/server/git_clone.go` - Git clone operations for server

### Modified Files (Server Components Only):
1. `internal/server/workflow_integration.go` - Update createWorkflowDirectory for clone
2. `internal/config/config.go` - Add server git clone configuration
3. `internal/server/handlers.go` - Enhanced error responses for clone failures

### Test Files (Server Tests):
1. `internal/server/github_url_test.go` - URL parsing tests
2. `internal/server/git_clone_test.go` - Clone operation tests
3. `internal/server/workflow_integration_test.go` - Server workflow tests

## Implementation Notes

- This fix is **server-only** - CLI functionality is not affected
- Use shallow clones (`--depth=1`) by default for server performance
- Implement proper timeout handling for server operations
- Ensure clone operations use `CommandContext` for cancellation
- Follow existing server error handling patterns
- Maintain REST API backward compatibility
- Consider rate limiting for server clone operations in future