# Implementation Plan for `river plan` Command

This document outlines the plan to implement the new `river plan` command.

## Development Approach

This implementation will strictly follow **Test-Driven Development (TDD)**. A test file, `internal/cli/plan_test.go`, will be created first. Tests will be written to cover each piece of functionality before the corresponding implementation code is written.

---

## Task 1: Create Command and Test Files (✅ IMPLEMENTED)

**Objective:** Create the necessary files for the command and its tests.

**Action:**
- Create a new file at `internal/cli/plan.go`.
- Create a corresponding test file at `internal/cli/plan_test.go`.

---

## Task 2: Write Initial Test for Command Definition (✅ IMPLEMENTED)

**Objective:** Ensure the `plan` command is correctly defined and registered.

**Action:**
- In `internal/cli/plan_test.go`, write a test that checks:
    - The `plan` command exists as a subcommand of `rootCmd`.
    - The command requires exactly one argument.

---

## Task 3: Define the Cobra Command (✅ IMPLEMENTED)

**Objective:** Define the `plan` command structure to satisfy the initial test.

**Action:**
- In `internal/cli/plan.go`, define a `cobra.Command` variable named `planCmd`.
- Configure `Use`, `Short`, `Long`, and `Args` properties.
- Register the command in an `init()` function by calling `rootCmd.AddCommand(planCmd)`.

---

## Task 4: Write Tests for Plan Generation Logic (✅ IMPLEMENTED)

**Objective:** Define the behavior of the `generatePlan` function through tests.

**Action:**
- In `internal/cli/plan_test.go`, write tests for the `generatePlan` function that cover:
    - **Error Case:** The function returns an error if the `GEMINI_API_KEY` is not set.
    - **Error Case:** The function returns an error if `prompts/prompt-plan.md` cannot be read.
    - **Error Case:** The function returns an error if no spec files are found.
    - **Success Case:** The function correctly constructs the prompt for the Gemini CLI. This may require mocking external dependencies like `os/exec` or filesystem access.
    - **Success Case:** The function correctly writes the output from the Gemini CLI to `plan.md`.

---

## Task 5: Implement the Plan Generation Logic (✅ IMPLEMENTED)

**Objective:** Implement the core `generatePlan` function to make the tests pass.

**Action:**
- Create the `generatePlan(task string) error` function in `internal/cli/plan.go`.
- Implement the logic as previously described (API key check, read template, find specs, build prompt, execute command, save output). This implementation will be guided by the tests written in Task 4.

**Implementation Notes:**
- The `generatePlan` function checks for the `GEMINI_API_KEY` environment variable
- Reads the prompt template from `prompts/prompt-plan.md`
- Finds all spec files in the `specs/` directory
- Constructs a prompt using the @filename syntax for Gemini CLI
- Filters environment variables to remove CI-related ones
- Executes the `gemini -p` command with the constructed prompt
- Writes the output to `plan.md`
- All tests pass and the code is properly formatted