# River CLI - Bare Execution Mode Implementation Plan

## Overview
Implement bare execution mode for River CLI: allow `river --no-plan --no-worktree` to run without a task description.

**GitHub Issue**: #6  
**Estimated Time**: 8-10 hours

## Requirements
1. Accept no arguments when both `--no-plan` and `--no-worktree` flags are set
2. Continue from existing `claude_state.json` if present
3. Start new workflow with `/ralph` if no state exists
4. Maintain backwards compatibility

## Implementation Tasks

### 1. CLI Argument Validation (2 hours) ✅ IMPLEMENTED
**File**: `internal/cli/root.go`

**Changes**:
```go
// In rootCmd.PreRunE
if len(args) == 0 && !cmd.Flag("file").Changed {
    noPlan, _ := cmd.Flags().GetBool("no-plan")
    noWorktree, _ := cmd.Flags().GetBool("no-worktree")
    
    if !(noPlan && noWorktree) {
        return fmt.Errorf("either provide a task description or use --file flag")
    }
}
```

**Tests**:
- `TestRootCmd_BareMode_AcceptsNoArgs`
- `TestRootCmd_RequiresArgs_WithSingleFlag`

### 2. Task Description Handling (1 hour) ✅ IMPLEMENTED
**File**: `internal/cli/workflow.go`

**Changes**:
```go
// In extractTaskDescription
taskDescription := strings.TrimSpace(strings.Join(args, " "))

if taskDescription == "" {
    noPlan, _ := cmd.Flags().GetBool("no-plan")
    noWorktree, _ := cmd.Flags().GetBool("no-worktree")
    
    if !(noPlan && noWorktree) {
        return "", fmt.Errorf("task description cannot be empty")
    }
}
```

**Tests**:
- `TestExtractTaskDescription_BareMode` ✅ All tests passing

### 3. Workflow Engine State Handling (3 hours) ✅ IMPLEMENTED
**File**: `internal/workflow/workflow.go`

**Changes**:
```go
// In Run method
isBareMode := taskDescription == "" && !generatePlan && !e.cfg.Git.WorktreeEnabled

if isBareMode {
    if _, err := os.Stat(e.stateFile); err == nil {
        // Continue from existing state
        e.logger.Info("Continuing from existing state file")
        return e.runWorkflowLoop(ctx)
    } else if os.IsNotExist(err) {
        // Initialize with /ralph
        e.logger.Info("Starting bare execution with /ralph")
        if err := e.initializeWorkflow(ctx, "/ralph", false); err != nil {
            return fmt.Errorf("failed to initialize bare workflow: %w", err)
        }
    }
} else {
    // Normal initialization
    if err := e.initializeWorkflow(ctx, taskDescription, generatePlan); err != nil {
        return fmt.Errorf("failed to initialize workflow: %w", err)
    }
}

return e.runWorkflowLoop(ctx)
```

**Tests**:
- `TestEngine_BareMode_ContinuesExistingState` ✅ All tests passing
- `TestEngine_BareMode_InitializesWithRalph` ✅ All tests passing

### 4. Integration Tests (2 hours) ✅ IMPLEMENTED
**File**: `test/integration/bare_mode_test.go`

**Key Tests**:
- Complete bare mode workflow ✅ TestBareMode_CompleteWorkflow
- State continuation after interrupt ✅ TestBareMode_HandlesInterrupt
- Error handling for invalid flags ✅ TestBareMode_RequiresBothFlags
- Bare mode starts with /ralph ✅ TestBareMode_StartsWithRalph
- Continues from existing state ✅ TestBareMode_ContinuesExistingState
- Error handling ✅ TestBareMode_ErrorHandling
- State file persistence ✅ TestBareMode_StateFilePersistence

### 5. Documentation (1-2 hours) ✅ IMPLEMENTED
**Updates**:
1. Help text in `root.go`: ✅ COMPLETED
   ```
   river --no-plan --no-worktree  # Bare execution mode
   ```

2. CLAUDE.md - Add bare mode section ✅ COMPLETED

3. CHANGELOG.md - Document new feature ✅ COMPLETED

## Testing Checklist
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing scenarios:
  - [ ] First run creates state with `/ralph`
  - [ ] Second run continues from state
  - [ ] Single flag shows error
  - [ ] Ctrl+C saves state correctly

## Success Criteria
- [ ] Bare mode works as specified
- [ ] No breaking changes
- [ ] All tests pass
- [ ] golangci-lint clean
- [ ] Documentation updated

## Notes
- This is an advanced feature - requires both flags to prevent accidental use
- State file handling must be robust
- Clear error messages are critical