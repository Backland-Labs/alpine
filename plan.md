# plan.md

## Overview

This document outlines the implementation plan for a real-time tool usage logging feature. This will enhance developer experience by providing a live, unobtrusive feed of the agent's operations in the terminal, using a sticky header for the primary task and a scrolling log for tool calls.

**Issue Summary**: Users need better visibility into the agent's real-time activities. The current spinner is uninformative. This feature will display a primary task status and a secondary, scrolling log of the last 3-4 tool calls captured from the `todo-monitor.rs` hook script's `stderr`.

**Objectives**:
- Implement a two-tiered display with a sticky primary status and a scrolling secondary log.
- Capture `stderr` from the `todo-monitor.rs` hook script in real-time.
- Enhance `internal/output/Printer` to manage the state and rendering of the tool log.
- Use ANSI escape codes to create a flicker-free, scrolling UI in the terminal.
- Make the feature enabled by default and configurable via an environment variable.
- Follow Test-Driven Development (TDD) principles throughout implementation.

## P0: Core Real-time Logging Display

This priority covers the essential functionality to capture, manage, and display the real-time tool log.

### Task 1: Enhance `Printer` to Manage Tool Log State (TDD Cycle) ✅ IMPLEMENTED

- **Acceptance Criteria**:
    - The `output.Printer` struct has a new field, such as `toolLogs`, which is a circular buffer (e.g., a slice) that stores the last N (e.g., 4) tool log messages.
    - A new method, `AddToolLog(message string)`, correctly adds a new message to the buffer, evicting the oldest message if the buffer is full.
    - The state management is thread-safe.
- **Test Cases**:
    - `TestAddToolLog_Empty`: Test adding a log to an empty buffer.
    - `TestAddToolLog_Append`: Test adding logs without reaching capacity.
    - `TestAddToolLog_CircularBuffer`: Test that adding a new log when the buffer is full correctly evicts the oldest log.
    - `TestAddToolLog_Concurrency`: Test adding logs from multiple goroutines concurrently to ensure thread safety.
- **Implementation Steps**:
    1. In `internal/output/color.go`, add a `toolLogs []string` slice and a `maxToolLogs int` constant to the `Printer` struct. Add a `sync.Mutex` for thread safety.
    2. Create a new test file `internal/output/tool_log_test.go`.
    3. Write failing tests for the `AddToolLog` method that cover the acceptance criteria.
    4. Implement the `AddToolLog` method in `internal/output/color.go`. It should lock the mutex, append the new log, and trim the slice if it exceeds `maxToolLogs`.
    5. Ensure all tests pass.

### Task 2: Capture Tool Logs from `stderr` in Real-time (TDD Cycle)

- **Acceptance Criteria**:
    - The `claude.Executor` can capture the `stderr` stream of the hook script process separately from `stdout`.
    - Each line from the `stderr` stream is passed to the `output.Printer`'s new `AddToolLog` method.
    - The existing functionality of capturing `stdout` and handling command completion is preserved.
- **Test Cases**:
    - `TestExecute_StderrCapture`: Test that a mock command's `stderr` output is captured line-by-line and passed to a mock printer's `AddToolLog` method.
    - `TestExecute_StdoutUnchanged`: Verify that the command's `stdout` is still captured correctly as the return value.
    - `TestExecute_CommandError`: Ensure that if the command fails, its `stderr` is still captured and the error is propagated correctly.
- **Implementation Steps**:
    1. In `internal/claude/executor.go`, modify the `executeClaudeCommand` function.
    2. Instead of using `cmd.CombinedOutput()`, get separate pipes for `stdout` and `stderr` using `cmd.StdoutPipe()` and `cmd.StderrPipe()`.
    3. Write a failing test in `internal/claude/executor_test.go` that uses a mock command to write to `stderr` and asserts that a mock `Printer`'s `AddToolLog` method is called.
    4. Implement the logic to read from the `stderr` pipe line-by-line in a separate goroutine. Use a `bufio.Scanner`.
    5. In the goroutine, call the `printer.AddToolLog(line)` for each captured line.
    6. Read the `stdout` pipe to completion and wait for the command to exit using `cmd.Wait()`.
    7. Ensure all existing and new tests pass.

### Task 3: Implement Sticky Header Rendering Logic (TDD Cycle)

- **Acceptance Criteria**:
    - A new method on `output.Printer`, `RenderToolLogs()`, uses ANSI escape codes to display the sticky header and scrolling log.
    - The display is updated without flickering.
    - The primary task (from `UpdateCurrentTask`) remains fixed, while the tool logs scroll below it.
    - The tool log is indented and rendered in a dimmer color.
- **Test Cases**:
    - `TestRenderToolLogs_Empty`: Verify that nothing is rendered when the tool log buffer is empty.
    - `TestRenderToolLogs_Partial`: Verify correct rendering when the buffer is not yet full.
    - `TestRenderToolLogs_Full`: Verify correct rendering when the buffer is full.
    - `TestRenderToolLogs_ANSIOutput`: Check that the output string contains the correct ANSI codes for moving the cursor up, clearing lines, and setting colors.
- **Implementation Steps**:
    1. In `internal/output/tool_log_test.go`, write failing tests for a new `RenderToolLogs` method. The tests will check the generated string for correct ANSI escape sequences.
    2. In `internal/output/color.go`, implement `RenderToolLogs`. The logic should:
        a. Move the cursor up N+1 lines (where N is the number of logs).
        b. Re-print the primary task line (from `UpdateCurrentTask`).
        c. Loop through the `toolLogs` buffer and print each log, indented and with a dim color (`colorGray`).
    3. Modify the `stderr` reading goroutine in `claude.Executor` to call `RenderToolLogs` after adding each new log.
    4. Modify `UpdateCurrentTask` to also call `RenderToolLogs` to ensure the display is cohesive.
    5. Ensure all tests pass.

## P1: Configuration and Usability

### Task 4: Add Configuration Flag (TDD Cycle)

- **Acceptance Criteria**:
    - A new configuration option, `ShowToolUpdates` (e.g., `RIVER_SHOW_TOOL_UPDATES`), is available.
    - The feature is enabled by default.
    - If the flag is set to `false`, the real-time tool log is not displayed.
- **Test Cases**:
    - `TestConfig_ShowToolUpdatesDefault`: Verify that `ShowToolUpdates` defaults to `true`.
    - `TestConfig_ShowToolUpdatesEnvVar`: Verify that the `RIVER_SHOW_TOOL_UPDATES` environment variable correctly sets the config value to `false`.
    - `TestExecutor_ToolLogsDisabled`: Verify that if the config flag is false, the `stderr` of the hook is not captured and the printer's `AddToolLog` method is not called.
- **Implementation Steps**:
    1. In `internal/config/config_test.go`, add tests for the new `ShowToolUpdates` field.
    2. In `internal/config/config.go`, add the `ShowToolUpdates bool` field to the `Config` struct and load it from the `RIVER_SHOW_TOOL_UPDATES` environment variable, defaulting to `true`.
    3. In `internal/claude/executor.go`, wrap the `stderr` capturing and rendering logic in a conditional check based on `config.ShowToolUpdates`.
    4. Ensure all tests pass.

### Task 5: Update Documentation

- **Acceptance Criteria**:
    - `README.md` is updated to mention the new real-time tool logging feature.
    - `specs/configuration.md` is updated with the new `RIVER_SHOW_TOOL_UPDATES` environment variable.
- **Implementation Steps**:
    1. Add a section to `README.md` under "Features" describing the new UI.
    2. Add the `RIVER_SHOW_TOOL_UPDATES` variable to the configuration table in `specs/configuration.md`.

## Success Criteria Checklist

- [ ] `output.Printer` can store and manage the last N tool logs.
- [ ] `claude.Executor` captures `stderr` from hook scripts in real-time.
- [ ] A sticky header UI is implemented using ANSI escape codes.
- [ ] The primary task and scrolling tool log are displayed without flickering.
- [ ] The feature can be disabled via the `RIVER_SHOW_TOOL_UPDATES` environment variable.
- [ ] All new logic is covered by unit tests following TDD.
- [ ] `README.md` and `specs/configuration.md` are updated.
- [ ] All existing tests continue to pass.
- [ ] `go fmt` and `golangci-lint` pass without issues.
