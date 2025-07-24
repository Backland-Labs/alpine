# plan.md

## Overview

This document outlines the implementation plan for a new `gh-issue` subcommand for the `river plan` command. This feature will allow users to generate an implementation plan directly from a GitHub issue URL, streamlining the workflow from issue tracking to development planning.

**Issue Summary**: Users want to generate a `plan.md` file by providing a link to a GitHub issue, which will be fetched using the `gh` CLI and used as the task description.

**Objectives**:
- Create a new subcommand: `river plan gh-issue <github-issue-url>`
- Use the `gh` CLI to fetch the issue's title and body.
- Combine the title and body to form a task description.
- Pass the task description to the existing plan generation logic.
- Ensure robust error handling (e.g., `gh` not installed, invalid URL).
- Follow TDD principles throughout implementation.

## P0: Core `gh-issue` Subcommand Implementation

This priority covers the essential functionality to make the feature work end-to-end.

### Task 1: Add `gh-issue` subcommand to `plan` command (TDD Cycle) ✅ IMPLEMENTED

- **Acceptance Criteria**:
    - A new subcommand `gh-issue` is available under `river plan`.
    - The command accepts exactly one argument (the GitHub issue URL).
    - The command has appropriate `Use`, `Short`, and `Long` descriptions.
    - The command fails gracefully if more or less than one argument is provided.
- **Test Cases**:
    - `TestGhIssueCommand_Exists`: Verify the `gh-issue` command is registered under `plan`.
    - `TestGhIssueCommand_RequiresOneArgument`: Test that the command returns an error if zero or more than one argument is provided.
    - `TestGhIssueCommand_HelpText`: Verify the help text is informative.
- **Implementation Steps**:
    1. In `internal/cli/plan.go`, create a `newGhIssueCmd()` function that returns a `*cobra.Command`.
    2. Define the command structure with `Use: "gh-issue <url>"`, `Short`, `Long`, and `Args: cobra.ExactArgs(1)`.
    3. In `newPlanCmd()`, add the new command using `planCmd.AddCommand(newGhIssueCmd())`.
    4. Write tests in `internal/cli/plan_test.go` to validate the command's structure and argument handling.

### Task 2: Implement GitHub Issue Fetching Logic (TDD Cycle)

- **Acceptance Criteria**:
    - The command correctly calls the `gh issue view <url> --json title,body` command.
    - The JSON output from the `gh` CLI is successfully parsed.
    - The issue title and body are extracted into separate variables.
    - The command handles errors from the `gh` CLI (e.g., issue not found, not authenticated).
- **Test Cases**:
    - `TestFetchGitHubIssue_Success`: Test with a mock `gh` command that returns valid JSON, and verify the title and body are extracted correctly.
    - `TestFetchGitHubIssue_GhNotFound`: Test the scenario where the `gh` command is not in the system's PATH.
    - `TestFetchGitHubIssue_ApiError`: Test with a mock `gh` command that returns an error exit code and stderr message.
- **Implementation Steps**:
    1. Create a new function `fetchGitHubIssue(url string) (title, body string, err error)`.
    2. Use `exec.Command` to run `gh issue view <url> --json title,body`.
    3. Capture the stdout and stderr of the command.
    4. If the command fails, return a descriptive error.
    5. If successful, unmarshal the JSON output into a struct.
    6. Return the title and body.
    7. Create an interface for command execution to allow for mocking in tests.

### Task 3: Integrate Issue Fetching with Plan Generation (TDD Cycle)

- **Acceptance Criteria**:
    - The fetched issue title and body are combined into a single task description string.
    - The combined task description is passed to the existing `generatePlan` or `generatePlanWithClaude` functions.
    - The `--cc` flag from the parent `plan` command is respected.
- **Test Cases**:
    - `TestGhIssueCommand_Integration_Gemini`: Test the full flow, mocking the `gh` CLI and verifying that the combined task description is passed to a mocked `generatePlan` function.
    - `TestGhIssueCommand_Integration_Claude`: Test the same flow but with the `--cc` flag, verifying the call to a mocked `generatePlanWithClaude` function.
- **Implementation Steps**:
    1. In the `RunE` function of the `gh-issue` command, call `fetchGitHubIssue`.
    2. Format the title and body into a single string, for example: `Task: <title>\n\n<body>`.
    3. Access the `--cc` flag from the parent command's flags.
    4. Call the appropriate plan generation function (`generatePlan` or `generatePlanWithClaude`) with the new task description.
    5. Propagate any errors from the plan generation functions.

## P1: Documentation and Usability

### Task 4: Update Documentation and CLI Help

- **Acceptance Criteria**:
    - The `README.md` file is updated to include instructions for the `river plan gh-issue` command.
    - The `specs/cli-commands.md` file is updated with the new command specification.
    - The command's help text is clear and provides an example.
- **Implementation Steps**:
    1. Add a new section in `README.md` under "Plan Generation" for the `gh-issue` subcommand.
    2. Update the command list in `specs/cli-commands.md`.
    3. Refine the `Long` description in `internal/cli/plan.go` for the `gh-issue` command.

## Success Criteria Checklist

- [ ] A new `river plan gh-issue <url>` command is implemented and functional.
- [ ] The command correctly fetches issue data from GitHub using the `gh` CLI.
- [ ] The fetched data is used to generate a `plan.md` file.
- [ ] The `--cc` flag is respected for choosing the planning engine.
- [ ] Errors (e.g., `gh` not found, invalid URL) are handled gracefully.
- [ ] Unit tests are written for all new logic, with external commands mocked.
- [ ] Documentation is updated to reflect the new feature.
- [ ] All existing tests continue to pass.
- [ ] `go fmt` and `golangci-lint` pass without issues.