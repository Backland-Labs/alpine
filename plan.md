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

### Task 4: Update Server Workflow Directory Creation

**Acceptance Criteria**:
- Server detects GitHub issue URLs from REST API and triggers clone
- Falls back to empty directory for non-GitHub workflows
- Passes cloned directory path to Claude Code

**Test Cases**:
- Test GitHub URL detection in server workflow
- Test fallback when clone is disabled
- Test error handling and recovery

**Implementation Steps**:
1. Modify `internal/server/workflow_integration.go`:
   - Update `createWorkflowDirectory()` to detect GitHub URLs
   - Parse GitHub URL to extract repository information
   - Clone repository before creating worktree
   - Create worktree in cloned repository
   - Fall back to temp directory on failure
2. Pass issue URL context through workflow creation

**Integration Points**:
- Called from server's `StartWorkflow()` method
- Integrates with existing worktree manager

### Task 5: Add Server Clone Operation Logging

**Acceptance Criteria**:
- Server logs clone operations with context
- Includes repository URL and run ID
- Logs errors with full details

**Test Cases**:
- Verify log output for server operations
- Test error logging in server context

**Implementation Steps**:
1. Add logging in server git clone operations:
   - Log clone start with repository URL and run ID
   - Log clone completion with duration
   - Log errors with full context
2. Use existing server logger patterns

**Integration Points**:
- Uses server's logging infrastructure

### Task 6: Implement Server Clone Cleanup

**Acceptance Criteria**:
- Server cleans up cloned repositories after workflow completion
- Respects server cleanup configuration
- Handles cleanup failures gracefully

**Test Cases**:
- Test automatic cleanup after server workflow
- Test cleanup disable via configuration
- Test cleanup error handling

**Implementation Steps**:
1. Track cloned repositories in server workflow context
2. Add cleanup in `AlpineWorkflowEngine` cleanup phase:
   - Remove entire cloned repository directory
   - Handle cleanup errors without failing workflow
3. Respect `ALPINE_GIT_AUTO_CLEANUP` setting

**Integration Points**:
- Called from server workflow cleanup phase

### Task 7: Add Server-Specific Error Handling

**Acceptance Criteria**:
- REST API returns appropriate HTTP status codes for clone failures
- Provides clear error messages in API responses
- Falls back gracefully with informative messages

**Test Cases**:
- Test API error responses for clone failures
- Test authentication failure responses
- Test timeout error responses

**Implementation Steps**:
1. Add server error types:
   - Clone timeout errors (return 504 Gateway Timeout)
   - Authentication errors (return 401 Unauthorized)
   - Repository not found (return 404 Not Found)
2. Update REST API handler error responses
3. Add fallback with warning in response

**Integration Points**:
- REST API error handling

## Success Criteria Checklist

- [ ] Server REST API successfully parses GitHub issue URLs
- [ ] Server can clone public repositories without authentication
- [ ] Server can clone private repositories with authentication token
- [ ] Server creates worktrees in cloned repositories
- [ ] Claude Code receives proper working directory in server workflows
- [ ] Server falls back gracefully when clone fails
- [ ] Clone operations timeout after configurable duration
- [ ] Server cleans up cloned repositories after workflow
- [ ] REST API returns appropriate error codes for failures
- [ ] Server logs clone operations for debugging
- [ ] All server code has appropriate test coverage

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