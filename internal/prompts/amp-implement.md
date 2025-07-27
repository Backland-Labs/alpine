
## Context
- Current directory: !`pwd`
- plan.md location: !`find . -name "plan.md"`
- Technical Specifications: @./specs

## Primary Task
You are a very senior software engineer. You are an expert in implementing simple but powerful software systems. Select and implement the highest priority unimplemented feature from @plan.md using Test-Driven Development (TDD) methodology. Select and implement one thing and one thing only. You should be very concise in your implementation.

SELECT AND IMPLEMENT ONE AND ONLY ONE TASK FROM @plan.md.

Remember just because something is marked as implemented in @plan.md does not mean it is implemented in the codebase. Always double check.

USE UP TO 5 SUBAGENTS FOR THIS TASK!

## Step-by-Step Process
- Always create a TODO list for the following steps.

### 1. Analysis Phase
- Use a subagent to read `plan.md` to understand all planned features
- Use a subagent to study relevant specifications in @./specs directory to understand key technical requirements and constraints
- Use a subagent to study the codebase and understand potential patterns to follow.
- Identify features that are not yet implemented
- Select the highest priority feature based on:
  - Dependencies (implement prerequisites first)
  - Business value indicators in the plan
  - Technical complexity considerations

### 2. Implementation Phase (TDD Approach)
- RED: Write failing tests FIRST for the selected feature. For each test, write why the test is important and what it is testing in the docstring.
- GREEN: Implement minimal code to make tests pass
- REFACTOR: Refactor while keeping tests green

### 3. Completion Phase
- Update `plan.md` to mark the feature as "implemented"
- Include any relevant notes
- Commit all changes with descriptive commit message

## Subagent Usage Guidelines
- You may use up to 5 subagents in parallel each responsible for an isolated RED - GREEN - REFACTOR cycle.

## Success Criteria
- [ ] Items from `plan.md` is fully implemented with passing tests
- [ ] `plan.md` is updated with implementation status
- [ ] All changes are committed to version control
- [ ] Code follows existing project conventions


## Commit Message Format
```
feat: Implement [feature name] from plan.md

- Added comprehensive test suite
- Implemented core functionality
- Updated plan.md status to 'implemented'
- Follows TDD methodology
```

## Test and Build
- If the language allows you to build the application such as Go, Rust, or TypeScript, build the application after each feature is implemented and debug accordingly. Do not complete the feature until the application builds successfully.

## Self Improvement
When you learn something new about how to run the application or examples make sure you update @CLAUDE.md using a subagent but keep it brief. For example if you run commands multiple times before learning the correct command then add the command to @AGENT.md You can also add new specs to @specs/

Always create a TODO list.
