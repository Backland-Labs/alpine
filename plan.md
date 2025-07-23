# Fix Claude Commands Not Executing in Git Worktree Directory

## Overview

**GitHub Issue**: #7  
**Problem**: River creates Git worktrees for isolation but Claude commands execute in the original repository directory instead of the worktree, defeating the purpose of isolation.

**Root Cause**: The `exec.Command` in `internal/claude/executor.go` doesn't set the `Dir` field, so Claude inherits the directory from when River started, not River's current working directory.

**Objective**: Ensure Claude commands execute in the correct worktree directory to maintain proper isolation.

## Prioritized Features

### P0: Core Working Directory Fix
**Must Have** - Critical bug fix for worktree isolation

### P1: Testing Infrastructure  
**Should Have** - Ensure the fix works and prevent regressions

### P2: Error Handling & Validation
**Should Have** - Robust handling of edge cases

### P3: Documentation Updates
**Nice to Have** - Update documentation to reflect correct behavior

## Implementation Tasks

### P0-1: Set Working Directory in Claude Executor ✅ IMPLEMENTED
**File**: `internal/claude/executor.go`

**Acceptance Criteria**:
- Claude commands execute in the current working directory ✅
- Worktree isolation works correctly ✅
- Backward compatibility maintained for non-worktree usage ✅

**Test Cases**:
```go
func TestBuildCommand_SetsWorkingDirectory(t *testing.T)
func TestBuildCommand_HandlesGetWdError(t *testing.T)
```

**Implementation Steps**:
1. Modify `buildCommand()` method around line 132-138
2. Add `os.Getwd()` call to get current working directory
3. Set `cmd.Dir = workDir` in the command construction
4. Handle `os.Getwd()` error gracefully

**Code Change**:
```go
// Create command
cmd := exec.Command("claude", args...)

// Set working directory to current directory (enables worktree isolation)
if workDir, err := os.Getwd(); err == nil {
    cmd.Dir = workDir
}

// Existing environment setup...
```

**Integration Points**:
- Works with existing workflow engine directory management
- Compatible with `RIVER_WORKDIR` configuration
- No changes needed in worktree manager

### P0-2: Update Command Runner Working Directory ✅ IMPLEMENTED
**File**: `internal/claude/executor.go`

**Acceptance Criteria**:
- `defaultCommandRunner.Run()` preserves working directory from `buildCommand` ✅
- Command execution inherits correct directory context ✅

**Test Cases**:
```go
func TestCommandRunner_PreservesWorkingDirectory(t *testing.T)
```

**Implementation Steps**:
1. Update `defaultCommandRunner.Run()` method around line 163
2. Ensure `cmd.Dir` is preserved when creating `CommandContext`
3. Verify environment and directory are both inherited

**Code Change**:
```go
cmd := exec.CommandContext(ctx, baseCmd.Path, baseCmd.Args[1:]...)
cmd.Env = baseCmd.Env
cmd.Dir = baseCmd.Dir  // Preserve working directory from buildCommand
```

### P1-1: Unit Tests for Working Directory
**File**: `internal/claude/executor_test.go`

**Acceptance Criteria**:
- Tests verify `cmd.Dir` is set correctly
- Tests handle `os.Getwd()` errors
- Tests verify backward compatibility

**Test Cases**:
```go
func TestExecutor_BuildCommand_SetsWorkingDirectory(t *testing.T) {
    // Verify cmd.Dir equals os.Getwd()
}

func TestExecutor_BuildCommand_WorkingDirectoryError(t *testing.T) {
    // Mock os.Getwd() to return error, verify graceful handling
}

func TestExecutor_CommandRunner_PreservesDirectory(t *testing.T) {
    // Verify working directory flows through command runner
}
```

**Implementation Steps**:
1. Add test for successful working directory setting
2. Add test for `os.Getwd()` error handling  
3. Add test for command runner directory preservation
4. Update existing tests if needed

### P1-2: Integration Test for Worktree Execution ✅ IMPLEMENTED
**File**: `test/e2e/worktree_test.go`

**Acceptance Criteria**:
- End-to-end test verifies Claude executes in worktree ✅
- Test confirms file operations happen in correct directory ✅
- Test validates state file is in worktree ✅

**Test Cases**:
```go
func TestWorktree_ClaudeExecutesInCorrectDirectory(t *testing.T) {
    // Create worktree, run Claude command, verify working directory
}

func TestWorktree_FileOperationsIsolated(t *testing.T) {
    // Verify file created by Claude appears in worktree, not main repo
}
```

**Implementation Steps**:
1. Extended existing worktree test file ✅
2. Added test that creates a worktree and runs a Claude command ✅
3. Verified Claude operations happen in worktree directory ✅
4. Added assertions for file isolation ✅

**Implementation Notes**:
- Created custom mock Claude scripts that record working directory and perform file operations
- Tests use `RIVER_GIT_AUTO_CLEANUP=false` to preserve worktrees for inspection
- Both tests verify complete isolation between main repo and worktree
- All e2e tests pass, confirming the fix works correctly

### P2-1: Error Handling for Working Directory ✅ IMPLEMENTED
**File**: `internal/claude/executor.go`

**Acceptance Criteria**:
- Graceful fallback when `os.Getwd()` fails ✅
- Clear error messages for directory-related issues ✅
- Logging for debugging working directory issues ✅

**Test Cases**:
```go
func TestExecutor_WorkingDirectoryFallback(t *testing.T) ✅
```

**Implementation Steps**:
1. Add error handling for `os.Getwd()` failure ✅
2. Add optional logging for working directory debugging ✅
3. Ensure fallback behavior is consistent ✅

**Code Enhancement**:
```go
workDir, err := os.Getwd()
if err != nil {
    // Log warning but continue without setting Dir
    // Claude will use default behavior
    logger.WithField("error", err).Info("Failed to get working directory, Claude will use default directory")
} else {
    cmd.Dir = workDir
    logger.WithField("workDir", workDir).Debug("Set Claude working directory")
}
```

**Implementation Notes**:
- Enhanced error handling with informative logging
- Graceful fallback maintains command execution even when working directory fails
- Added both error and success logging for debugging
- Tests verify fallback behavior works correctly

### P2-2: Working Directory Validation
**File**: `internal/claude/executor.go`

**Acceptance Criteria**:
- Validate working directory exists and is accessible
- Handle edge cases like permission issues
- Provide helpful error messages

**Test Cases**:
```go
func TestExecutor_ValidatesWorkingDirectory(t *testing.T)
```

**Implementation Steps**:
1. Add validation that working directory exists
2. Check directory permissions if needed
3. Add appropriate error handling

### P3-1: Update Documentation
**File**: `CLAUDE.md`

**Acceptance Criteria**:
- Document corrected worktree behavior
- Update architecture notes about directory isolation
- Add troubleshooting notes for directory issues

**Implementation Steps**:
1. Add section about worktree directory isolation
2. Update workflow documentation
3. Add troubleshooting section

## Success Criteria

- [x] Claude commands execute in worktree directory when worktrees are enabled
- [x] File operations by Claude are isolated to the worktree
- [x] State file management works correctly in worktree context  
- [x] Backward compatibility maintained for `--no-worktree` usage
- [x] All existing tests pass
- [x] New tests verify the fix works correctly
- [x] `golangci-lint` passes without new warnings

## Risk Assessment

**Low Risk** - This is a targeted fix that:
- Only affects where Claude commands execute
- Follows established patterns in the codebase
- Is backward compatible
- Changes are isolated to the Claude executor module

## Testing Strategy

1. **Unit tests**: Verify working directory is set correctly
2. **Integration tests**: End-to-end worktree workflow validation  
3. **Manual testing**: Verify actual Claude commands execute in worktree
4. **Regression testing**: Ensure non-worktree usage still works

## Implementation Order

1. **P0 tasks**: Core fix for working directory inheritance
2. **P1 tasks**: Testing infrastructure to validate the fix
3. **P2 tasks**: Enhanced error handling and edge cases
4. **P3 tasks**: Documentation updates