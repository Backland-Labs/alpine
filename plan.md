# Implementation Plan: Retry Mechanism for plan.md Generation

## Overview

Currently, Alpine trusts that Gemini will create a plan.md file when executed with the proper prompt. However, this doesn't always happen reliably. This plan implements a simple retry mechanism that validates plan.md existence after each Gemini execution and retries up to 3 times if the file doesn't exist.

### Objectives
- Ensure plan.md is reliably created when using `alpine plan` command
- Maintain simplicity - just retry the same command up to 3 times
- Provide clear user feedback during retry attempts
- Exit with simple error message if all attempts fail

## Prioritized Features

### P0: Core Retry Implementation

#### Task 1: Add Plan File Validation ✅ **IMPLEMENTED** (2025-07-27)
**Acceptance Criteria:**
- Function returns nil if plan.md exists and has content > 0 bytes
- Function returns error if plan.md doesn't exist
- Function returns error if plan.md exists but is empty

**Test Cases:**
```go
// Test validatePlanFile returns nil when plan.md exists with content
// Test validatePlanFile returns error when plan.md doesn't exist
// Test validatePlanFile returns error when plan.md is empty (0 bytes)
```

**Implementation:**
- Create `validatePlanFile() error` function in plan.go
- Use `os.Stat("plan.md")` to check existence
- Check `FileInfo.Size() > 0` for non-empty validation
- Return descriptive error using `fmt.Errorf`

**Integration Points:**
- Called after each Gemini execution attempt in generatePlan()

#### Task 2: Implement Retry Loop ✅ **IMPLEMENTED** (2025-07-27)
**Acceptance Criteria:**
- Executes Gemini command up to 3 times
- Stops on first successful plan.md creation
- Shows attempt number for each try
- Returns original implementation's success message on success

**Test Cases:**
```go
// Test successful generation on first attempt (no retries)
// Test successful generation on second attempt (1 retry)
// Test successful generation on third attempt (2 retries)
// Test failure after 3 attempts returns error
```

**Implementation:**
- Wrap existing Gemini execution in `for i := 1; i <= 3; i++` loop
- After each `cmd.Run()`, call `validatePlanFile()`
- If validation succeeds, break loop and return success
- If validation fails and i < 3, show retry message
- Continue to next iteration

**Integration Points:**
- Modifies the existing generatePlan() function flow
- Preserves all existing Gemini command setup

### P1: User Feedback Enhancement

#### Task 3: Add Progress Messages ✅ **IMPLEMENTED** (2025-07-27)
**Acceptance Criteria:**
- Shows "Attempt 1 of 3..." before first execution
- Shows "Attempt 2 of 3..." before retry
- Shows "Attempt 3 of 3..." before final retry
- Original "Generating plan..." message shown only on first attempt

**Test Cases:**
```go
// Test correct messages shown for each attempt
// Test no duplicate "Generating plan..." messages
// Test printer.Info() called with correct attempt messages
```

**Implementation:**
- Move `printer.Info("Generating plan...")` inside loop, conditional on i==1
- Add `printer.Info("Attempt %d of 3...", i)` at start of each iteration
- Use existing printer instance for consistency

**Integration Points:**
- Uses existing output.Printer for all messages

### P2: Error Handling

#### Task 4: Final Failure Handling ✅ **IMPLEMENTED** (2025-07-27)
**Acceptance Criteria:**
- After 3 failed attempts, shows error "Gemini failed to create plan"
- Returns error with same message
- No stack traces or technical details in user-facing error

**Test Cases:**
```go
// Test exact error message after 3 failures
// Test printer.Error() called with correct message
// Test function returns matching error
```

**Implementation:**
- After loop completes without success:
  - `printer.Error("Gemini failed to create plan")`
  - `return fmt.Errorf("gemini failed to create plan")`
- Remove or conditionalize existing error messages inside loop

**Integration Points:**
- Replaces current error handling for final failure case

## Success Criteria

- [x] plan.md is created successfully when Gemini works on any attempt
- [x] Retry mechanism activates only when plan.md is missing
- [x] User sees clear progress messages during retries
- [x] Final error message is exactly "Gemini failed to create plan"
- [x] No changes to Gemini command construction or prompt
- [x] No changes to environment filtering or other setup
- [x] All existing tests continue to pass
- [x] New tests cover all retry scenarios

## Implementation Notes

- Keep the implementation minimal - no exponential backoff, no error classification
- Preserve existing stdout/stderr piping for Gemini output
- Don't capture or suppress Gemini's output during retries
- File validation should use standard os.Stat pattern seen elsewhere in codebase
- Follow Alpine's error handling patterns (wrap with context, no panics)