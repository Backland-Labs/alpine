# Repository Cleanup and Refactoring Plan

## Overview

This plan outlines the steps to clean up the River CLI repository, remove dead code, refactor for higher code quality, and ensure all documentation and specifications are up-to-date. The primary goal is to streamline the codebase, improve maintainability, and accurately reflect the current state of the project.

## Guiding Principles

All tasks will be implemented following Test-Driven Development (TDD) principles. For each task, tests will be written or verified first, followed by the minimal implementation to make the tests pass, and finally, refactoring to improve code quality.

## P0: Code Cleanup & Dead Code Removal

### Task 1: Remove `test/validate_workflows.go` and `internal/validation` package

- **Priority**: P0
- **Status**: ✅ IMPLEMENTED (2025-07-24)
- **Acceptance Criteria**: The specified files and directories are removed, and the project still builds and passes all tests.
- **Test Cases**:
    - Run `go build ./...` to ensure the project compiles without errors.
    - Run `go test ./...` to ensure all existing tests pass.
- **Implementation Steps**:
    1. Delete the file `test/validate_workflows.go`.
    2. Delete the directory `internal/validation`.
    3. Run `go mod tidy` to clean up dependencies.
    4. Execute the build and test commands to verify that the removal did not break the build or existing functionality.

### Task 2: Remove unused prompt and spec files

- **Priority**: P0
- **Status**: ✅ IMPLEMENTED (2025-07-24)
- **Acceptance Criteria**: The specified files related to Amp and Gemini CLI are removed, as they are not part of the core River functionality.
- **Test Cases**: N/A (file removal).
- **Implementation Steps**:
    1. Delete `prompts/amp-implement.md`.
    2. Delete `specs/amp-cli.md`.
    3. Delete `specs/gemini-cli.md`.

## P1: Refactoring & Quality Improvements

### Task 3: Refactor `internal/claude/executor.go`

- **Priority**: P1
- **Status**: ✅ IMPLEMENTED (2025-07-24)
- **Acceptance Criteria**: The `executor.go` file is refactored for improved clarity and reduced complexity. The logic for different execution paths (with/without monitoring, with/without stderr capture) should be streamlined. All existing tests must pass.
- **Test Cases**:
    - All existing tests in `executor_test.go`, `executor_stderr_test.go`, and `executor_todo_test.go` must pass after refactoring.
    - New unit tests should be added to cover any new or modified logic.
- **Implementation Steps**:
    1. Analyze the `Execute` method and its helpers (`executeWithTodoMonitoring`, `executeWithoutMonitoring`, `executeClaudeCommand`, `executeWithStderrCapture`).
    2. Consolidate the various execution paths into a single, more configurable execution function.
    3. Use a struct for execution options to simplify the function signatures and improve readability.
    4. Ensure that all existing functionality, including TODO monitoring and stderr capture, is preserved.
    5. Run all related tests to verify the correctness of the refactoring.

## P2: Documentation & Spec Updates

### Task 4: Update `README.md` and `CLAUDE.md`

- **Priority**: P2
- **Status**: ✅ IMPLEMENTED (2025-07-24)
- **Acceptance Criteria**: The `README.md` and `CLAUDE.md` files are updated to accurately reflect the current state of the project, with all references to removed features (like Gemini plan generation) deleted.
- **Test Cases**: N/A (documentation).
- **Implementation Steps**:
    1. Review `README.md` and remove the "Plan Generation" section, including the comparison table and examples related to the `plan` command.
    2. Review `CLAUDE.md` and remove references to `amp-cli.md` and `gemini-cli.md` in the "Specifications" section.
    3. Read through both files to identify and correct any other outdated information.

### Task 5: Update `specs` directory

- **Priority**: P2
- **Status**: ✅ IMPLEMENTED (2025-07-24)
- **Acceptance Criteria**: The `specs` directory is cleaned up, and the remaining specification files are updated to be consistent with the current codebase.
- **Test Cases**: N/A (documentation).
- **Implementation Steps**:
    1. Review all files in the `specs/` directory.
    2. Update `cli-commands.md` to remove documentation for the `plan` and `gh-issue` subcommands.
    3. Update `configuration.md` to remove any configuration options that are no longer relevant.
    4. Review the remaining specification files for accuracy and consistency with the current implementation.

## Success Criteria

- [ ] All dead code and unused files listed in P0 tasks are removed.
- [ ] The `internal/claude/executor.go` file is successfully refactored, and all tests pass.
- [x] `README.md` and `CLAUDE.md` are updated and accurate.
- [x] The `specs` directory is cleaned up and all remaining files are up-to-date.
- [ ] The entire project builds successfully (`go build ./...`) and all tests pass (`go test ./...`).
- [ ] The linter passes without any warnings (`golangci-lint run`).
