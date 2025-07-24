# plan-claude-cc.md

## Overview

This document outlines the implementation plan for extending the `river plan` command to support Claude Code as an alternative to Gemini for plan generation. When the `--cc` flag is passed, River will use Claude Code instead of the Gemini model to generate plan.md files.

**Issue Summary**: Users want the option to use Claude Code for plan generation, leveraging its advanced code understanding and multi-turn conversation capabilities.

**Objectives**:
- Add a `--cc` flag to the `river plan` command
- When `--cc` is used, execute Claude Code instead of Gemini for plan generation
- Maintain backward compatibility (Gemini remains the default)
- Reuse existing Claude Code integration infrastructure
- Follow River's established patterns and architecture

## P0: Core `--cc` Flag Implementation

### Task 1: Add `--cc` flag to plan command (TDD Cycle)

- **Acceptance Criteria**:
    - The `plan` command accepts an optional `--cc` flag
    - The flag is properly documented in the command help
    - Flag state is accessible within the command's RunE function
    - Default behavior (without flag) remains unchanged
- **Test Cases**:
    - `TestPlanCommand_CCFlagExists`: Verify the flag is registered on the command
    - `TestPlanCommand_CCFlagDefault`: Verify flag defaults to false
    - `TestPlanCommand_ParsesCCFlag`: Test that the flag value is correctly parsed
- **Implementation Steps**:
    1. In `internal/cli/plan.go`, add a `ccFlag` boolean variable
    2. In `newPlanCmd()`, add the flag using `cmd.Flags().BoolVar(&ccFlag, "cc", false, "Use Claude Code instead of Gemini for plan generation")`
    3. Pass the `ccFlag` value to the `RunE` function closure
    4. Write tests in `internal/cli/plan_test.go` to verify flag behavior
- **Integration Points**:
    - `internal/cli/plan.go`: Command definition and flag registration

### Task 2: Implement Claude Code plan generation logic (TDD Cycle)

- **Acceptance Criteria**:
    - When `--cc` flag is set, the command uses Claude Code instead of Gemini
    - Claude Code is executed with appropriate restrictions for planning
    - The same prompt template (`prompts/prompt-plan.md`) is used initially
    - Output is streamed to console similar to Gemini execution
- **Test Cases**:
    - `TestGeneratePlanWithClaude`: Test the Claude plan generation logic
    - `TestPlanCommand_UsesClaudeWithCCFlag`: Integration test verifying Claude is called when flag is set
- **Implementation Steps**:
    1. Create `generatePlanWithClaude()` function in `internal/cli/plan.go`
    2. This function will:
        - Read the prompt template from `prompts/prompt-plan.md`
        - Replace `{{TASK}}` with the user's task
        - Create a restricted Claude executor with planning-appropriate tools
        - Execute Claude CLI with the prompt
        - Stream output to console
    3. Update the `RunE` function to check `ccFlag` and call either `generatePlan()` or `generatePlanWithClaude()`
    4. Add appropriate error handling and logging
    5. Write comprehensive tests
- **Integration Points**:
    - `internal/claude/executor.go`: Reuse existing Claude executor infrastructure
    - `prompts/prompt-plan.md`: Use existing prompt template

### Task 3: Configure Claude Code for planning context (TDD Cycle)

- **Acceptance Criteria**:
    - Claude Code is executed with restricted tools appropriate for planning
    - Read-only tools are allowed: Read, Grep, Glob, LS, WebSearch, WebFetch
    - Modification tools are blocked: Write, Edit, MultiEdit, Bash, TodoWrite
    - Claude has access to codebase context similar to Gemini's `--all-files`
    - System prompt is adjusted to focus on planning tasks
- **Test Cases**:
    - `TestClaudePlanningToolRestrictions`: Verify correct tools are allowed/blocked
    - `TestClaudePlanningSystemPrompt`: Test that appropriate system prompt is used
    - `TestClaudePlanningWorkingDirectory`: Verify Claude executes in correct directory
- **Implementation Steps**:
    1. Define `planningAllowedTools` slice with read-only tools
    2. Create a planning-specific system prompt that emphasizes:
        - Creating comprehensive plan.md files
        - Understanding codebase structure
        - Following River's planning conventions
    3. Configure Claude executor with:
        - Custom allowed tools list
        - Planning-specific system prompt
        - Correct working directory (project root)
        - No session persistence needed (single-shot execution)
    4. Consider using `--add-dir .` to provide codebase context
    5. Add tests to verify configuration
- **Integration Points**:
    - Tool configuration in Claude executor
    - System prompt customization

## P1: Enhanced Features

### Task 4: Add progress indicators for Claude execution

- **Acceptance Criteria**:
    - User sees clear indication when Claude is being used vs Gemini
    - Progress feedback during Claude execution
- **Implementation Steps**:
    1. Add startup message: "Generating plan using Claude Code..."
    2. Consider adding spinner or progress indicator
    3. Clear completion message when done

## Implementation Notes

1. **Backward Compatibility**: Gemini remains the default; `--cc` is opt-in
2. **Error Handling**: Clear error messages for missing API keys or execution failures
3. **Testing Strategy**: Mock CLI executions in unit tests; integration tests optional
4. **Documentation**: Update CLI help text and README with new flag information
5. **Tool Restrictions**: Critical to prevent Claude from modifying files during planning
6. **Performance**: Claude may take longer than Gemini; user should be aware

## Success Criteria Checklist

- [ ] `--cc` flag is implemented and functional on `river plan` command
- [ ] Claude Code executes successfully when flag is used
- [ ] Gemini remains the default when flag is not specified
- [ ] Appropriate tools are restricted during Claude planning
- [ ] All tests pass including new test cases
- [ ] Documentation is updated with new flag information
- [ ] Error messages are clear and helpful
- [ ] Output streaming works correctly for both engines
- [ ] Code follows River's established patterns and quality standards
