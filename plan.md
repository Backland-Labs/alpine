# Implementation Plan: Parallel Plan Generation using Git Worktrees

## Overview
- **Issue**: Support parallel execution of the `alpine plan` command to prevent file conflicts.
- **Objective**: Add a `--worktree` flag to the `alpine plan` command that creates an isolated git worktree for each plan generation, allowing multiple commands to run concurrently without overwriting `plan.md`.
- **Scope**: This change affects the `alpine plan` and `alpine plan gh-issue` commands. It includes adding the new flag, implementing the worktree creation and cleanup logic, and updating all relevant documentation.

## Technical Context
- **Affected Files**: 
  - `internal/cli/plan.go`
  - `internal/cli/plan_test.go`
  - `specs/cli-commands.md`
  - `README.md`
  - `CLAUDE.md`
- **Key Dependencies**: 
  - `github.com/spf13/cobra` for CLI command structure.
  - `github.com/Backland-Labs/alpine/internal/gitx` for existing worktree management logic.
- **API Changes**: No external API changes. CLI interface will be modified.

## Files to be Changed Checklist
### Modified Files
- [x] `internal/cli/plan.go`: Add `--worktree` and `--cleanup` flags; implement the core logic to create, use, and clean up worktrees during plan generation.
- [x] `internal/cli/plan_test.go`: Add unit and integration tests to verify the new flags and worktree functionality.
- [x] `specs/cli-commands.md`: Document the new `--worktree` and `--cleanup` flags and their behavior.
- [x] `README.md`: Add examples for the new parallel plan generation feature.
- [x] `CLAUDE.md`: Update the development commands and examples to reflect the new functionality.

## Implementation Tasks

### P0: Critical Path (Must Have)

#### Task 1: Add `--worktree` and `--cleanup` Flags to the `plan` Command
**Why**: To provide users with the ability to enable isolated plan generation and control the lifecycle of the created worktrees.

**Test First** (Write these tests in `internal/cli/plan_test.go`):
```go
import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestPlanCommand_WorktreeFlags(t *testing.T) {
	planCmd := NewPlanCommand()

	t.Run("should have a --worktree flag", func(t *testing.T) {
		worktreeFlag := planCmd.Flags().Lookup("worktree")
		assert.NotNil(t, worktreeFlag)
		assert.Equal(t, "bool", worktreeFlag.Value.Type(), "should be a boolean flag")
		assert.Equal(t, "false", worktreeFlag.DefValue, "should default to false")
		assert.Contains(t, worktreeFlag.Usage, "Generate the plan in an isolated git worktree")
	})

	t.Run("should have a --cleanup flag", func(t *testing.T) {
		cleanupFlag := planCmd.Flags().Lookup("cleanup")
		assert.NotNil(t, cleanupFlag)
		assert.Equal(t, "bool", cleanupFlag.Value.Type(), "should be a boolean flag")
		assert.Equal(t, "true", cleanupFlag.DefValue, "should default to true")
		assert.Contains(t, cleanupFlag.Usage, "Automatically clean up (remove) the worktree")
	})
}
```

**Implementation**:
1. Modify file: `internal/cli/plan.go`
2. In the `newPlanCmd` function, add the new flags to `pc.cmd.Flags()`:
   ```go
   var worktreeFlag bool
   var cleanupFlag bool
   // ... inside newPlanCmd ...
   pc.cmd.Flags().BoolVar(&worktreeFlag, "worktree", false, "Generate the plan in an isolated git worktree")
   pc.cmd.Flags().BoolVar(&cleanupFlag, "cleanup", true, "Automatically clean up (remove) the worktree after plan generation")
   ```
3. Ensure the `worktreeFlag` and `cleanupFlag` variables are passed into the `RunE` function's scope.

**Task-Specific Acceptance Criteria**:
- [x] The `alpine plan --help` command displays the `--worktree` and `--cleanup` flags with correct descriptions and defaults.
- [x] The flags are correctly parsed when the command is executed.

---

#### Task 2: Implement Worktree Logic for Plan Generation
**Why**: To create an isolated filesystem environment for each plan generation, preventing conflicts with `plan.md` and other files.

**Test First** (Write these tests in `internal/cli/plan_test.go`):
```go
// This will be an integration-style test, mocking the gitx.WorktreeManager
// and os.Chdir to verify the correct sequence of operations.
func TestPlanExecution_WithWorktree(t *testing.T) {
	// 1. Mock gitx.WorktreeManager
	// 2. Mock os.Getwd and os.Chdir to track directory changes
	// 3. Run the plan command with --worktree
	// 4. Assert that wtMgr.Create was called with a sanitized task name.
	// 5. Assert that os.Chdir was called to enter the worktree directory.
	// 6. Assert that the plan generation function (e.g., generatePlan) was called.
	// 7. Assert that os.Chdir was called again to return to the original directory.
	// 8. Assert that wtMgr.Cleanup was called (since --cleanup defaults to true).
}

func TestPlanExecution_WithWorktreeAndNoCleanup(t *testing.T) {
    // Similar to the above test, but run with --cleanup=false
    // and assert that wtMgr.Cleanup is *not* called.
}
```

**Implementation**:
1. Modify file: `internal/cli/plan.go`
2. Update the `RunE` function for both the `plan` command and its `gh-issue` subcommand.
3. Create a new helper function, `runPlanInWorktree`, that wraps the plan generation logic.
   ```go
   func runPlanInWorktree(task string, useClaude bool, cleanup bool) error {
       // 1. Get current working directory (originalDir)
       // 2. Instantiate gitx.NewCLIWorktreeManager
       // 3. Sanitize task to create a unique worktree name, e.g., "plan-implement-feature-x"
       // 4. Create the worktree: wt, err := wtMgr.Create(ctx, sanitizedTask)
       // 5. Defer the cleanup logic:
       defer func() {
           // Chdir back to originalDir
           os.Chdir(originalDir)
           // If cleanup is true, call wtMgr.Cleanup(ctx, wt)
           if cleanup {
               wtMgr.Cleanup(ctx, wt)
               fmt.Printf("Cleaned up worktree: %s\n", wt.Path)
           }
       }()
       // 6. Chdir into the new worktree: os.Chdir(wt.Path)
       // 7. Call the appropriate plan generation function
       if useClaude {
           return generatePlanWithClaude(task)
       }
       return generatePlan(task)
   }
   ```
4. In the `RunE` functions, check if the `worktreeFlag` is set. If true, call `runPlanInWorktree`. Otherwise, call the original `generatePlan` or `generatePlanWithClaude` functions.

**Task-Specific Acceptance Criteria**:
- [x] When `alpine plan --worktree "task"` is run, a new git worktree is created.
- [x] The `plan.md` file is generated inside the worktree directory, not the original directory.
- [x] The command's working directory is restored to its original location after execution.
- [x] By default (`--cleanup=true`), the worktree is removed after the command finishes.
- [x] With `--cleanup=false`, the worktree directory persists after the command finishes.

---

#### Task 3: Update Documentation
**Why**: To ensure users are aware of the new parallel plan generation capability and know how to use it effectively.

**Implementation**:
1. **Modify `specs/cli-commands.md`**:
   - Add the `--worktree` and `--cleanup` flags to the `alpine plan` command section.
   - Provide a clear explanation of what they do and how they enable parallel execution.
2. **Modify `README.md`**:
   - Add a new example under the "Usage" section demonstrating parallel plan generation.
   ```bash
   # Generate plans for multiple issues in parallel
   alpine plan --worktree gh-issue https://github.com/owner/repo/issues/123 &
   alpine plan --worktree gh-issue https://github.com/owner/repo/issues/124 &
   wait
   ```
3. **Modify `CLAUDE.md`**:
   - Update the "Running Alpine" section with an example of the new command usage.

**Task-Specific Acceptance Criteria**:
- [x] All three documentation files (`specs/cli-commands.md`, `README.md`, `CLAUDE.md`) are updated to reflect the new flags and functionality.
- [x] The examples provided are clear, correct, and demonstrate the primary use case of parallel execution.

## Global Acceptance Criteria

### Code Quality
- [x] Code coverage >80% for all new/modified code in `internal/cli/plan.go`.
- [x] No new lint errors or warnings are introduced (`golangci-lint run`).

### Testing
- [x] Unit tests are added for the new flags in `internal/cli/plan_test.go`.
- [x] Integration tests are added to verify the worktree creation and cleanup lifecycle.
- [x] Error cases are tested (e.g., running the command outside of a git repository).
- [x] Build and run the new commands with sample data, if possible.

### Documentation
- [x] All user-facing documentation is updated as specified in Task 3.
- [x] Code comments are added to `internal/cli/plan.go` to explain the new worktree logic.
