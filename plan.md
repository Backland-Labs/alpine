# plan.md

## Overview

This document outlines the implementation plan for the `river review` command, a new feature for the River CLI orchestrator. The `review` command will enable users to get an AI-powered review of a `plan.md` file against the current state of the codebase.

**Issue Summary**: Users need a way to validate that a `plan.md` is still relevant and accurately reflects the work needed to be done in the codebase.
**Objectives**:
- Create a new `river review` subcommand.
- The command should accept a path to a `plan.md` file.
- It will use an AI model (Gemini) to analyze the plan against the codebase.
- The analysis will be streamed to the user's console.

## P0: Core `review` Command Implementation

### Task 1: Create the `review` command structure (TDD Cycle)

- **Acceptance Criteria**:
    - A new `review` command is available under `river`.
    - The command is defined in `internal/cli/review.go` and `internal/cli/review_test.go`.
    - It is registered in `internal/cli/root.go`.
    - The command accepts exactly one argument, which is the path to the `plan.md` file.
    - The command fails if the `plan.md` file does not exist.
- **Test Cases**:
    - `TestReviewCommandExists`: Verify the command is registered in the root command.
    - `TestReviewCommand_RequiresOneArgument`: Test that the command returns an error if no arguments or more than one argument are provided.
    - `TestReviewCommand_FileDoesNotExist`: Test that the command returns an error if the provided path to `plan.md` does not exist.
- **Implementation Steps**:
    1. Create `internal/cli/review.go` and `internal/cli/review_test.go`.
    2. In `review.go`, create a `newReviewCmd` function that returns a `*cobra.Command`.
    3. Configure the `cobra.Command` with `Use: "review <plan-file>"`, a `Short` and `Long` description, and `Args: cobra.ExactArgs(1)`.
    4. Add a `RunE` function that checks if the file at the provided path exists.
    5. In `root.go`, add the new command using `cmd.AddCommand(newReviewCmd().Command())`.
    6. Write the corresponding tests in `review_test.go` to satisfy the test cases.
- **Integration Points**:
    - `internal/cli/root.go`: To register the new command.

### Task 2: Implement the plan review logic (TDD Cycle)

- **Acceptance Criteria**:
    - The `review` command reads the content of the `plan.md` file.
    - It constructs a prompt for the Gemini AI model, including the content of the `plan.md` and instructions to review it against the codebase.
    - It executes the `gemini` CLI with the constructed prompt.
    - The output of the `gemini` command is streamed to the console.
- **Test Cases**:
    - `TestGenerateReviewPlan`: Test the logic for generating the review prompt.
    - `TestExecuteReview`: Test the execution of the `gemini` command (this will be an integration test, or will require mocking `exec.Command`).
- **Implementation Steps**:
    1. Create a new function `generateReview` in `internal/cli/review.go` that takes the `plan.md` content as input.
    2. This function will be similar to `generatePlan` in `internal/cli/plan.go`. It will:
        - Check for the `GEMINI_API_KEY`.
        - Create a new prompt template for the review in `prompts/prompt-review.md`.
        - Read the template and inject the `plan.md` content.
        - Execute the `gemini` CLI with the prompt, streaming the output to the console.
    3. Update the `RunE` function in `review.go` to call `generateReview`.
    4. Add tests for the new logic.
- **Integration Points**:
    - `exec.Command`: To call the `gemini` CLI.
    - A new prompt file `prompts/prompt-review.md`.

## Success Criteria Checklist

- [ ] `river review` command is implemented and available.
- [ ] The command correctly handles file existence and argument validation.
- [ ] The command successfully calls the Gemini CLI with the correct prompt.
- [ ] The output from the Gemini CLI is displayed to the user.
- [ ] All new code is covered by unit and integration tests.
- [ ] The implementation follows the existing coding standards and patterns of the River project.
