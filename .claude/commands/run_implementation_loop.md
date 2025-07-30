---
allowed-tools: Bash(grep:*), Bash(ls:*), Bash(tree), Bash(git:*), Bash(find:*)
description: Implement and issue from plan.md
---
# Test-Driven Development Feature Implementation Guide

## Overview and Context

You are a senior software engineer tasked with implementing features from a project plan using Test-Driven Development (TDD) methodology. Your primary objective is to select and implement exactly ONE high-priority unimplemented feature from the `plan.md` file. This focused approach ensures quality over quantity and maintains a clear development cycle.

### Environment Setup
- **Current working directory**: !`pwd`
- **Project plan location**: !`find . -name "plan.md"`
- **Technical specifications**: Review documentation in @/specs
- **Available tools**: Bash commands including `grep`, `ls`, `tree`, `git`, and `find`

### Subagents
When applicable delegate tasks to subagents to decrease implementation time. Always be thinking about how we can do this work faster. Leverage subagents so that we can execute more work and do it quicker and more efficiently.

## Detailed Implementation Process

### Phase 1: Comprehensive Analysis

Begin by creating a detailed TODO list for your implementation. This list should guide your work and ensure nothing is overlooked. Use TodoWrite to track all subtasks.

#### 1.1 Understanding the Project Plan
Thoroughly read and analyze `plan.md`. You should:
- Document all features listed in the plan
- Note their current implementation status
- Identify dependencies between features
- Extract any priority indicators or business value metrics

#### 1.2 Specification Review
Deploy the spec-implementation-reviewer agent to study the technical specifications in the @/specs directory. This agent should:
- Map specifications to planned features
- Identify technical constraints or requirements
- Note any architectural decisions that impact implementation
- Document API contracts or interface definitions

#### 1.3 Feature Selection Criteria
Select the highest priority unimplemented feature based on:

**Dependencies First**: Implement prerequisite features before dependent ones. For example, if Feature B requires Feature A, implement Feature A first regardless of other factors.

**Technical Complexity Evaluation**: Consider:
- Estimated implementation time
- Risk of breaking existing functionality
- Required architectural changes
- Testing complexity

### Phase 2: Test-Driven Development Implementation
YOU CAN ASSIGN UP TO 5 SUBAGENTS TO WORK ON CONCURRENT TASKS FROM PLAN.MD IF THEY ARE NOT DEPENDENT ON EACH OTHER.

Once you've selected a feature, follow the TDD cycle rigorously. Remember: speed is important, but not at the expense of quality. Do the minimum necessary to accomplish the task effectively.

#### 2.1 RED Phase - Write Failing Tests First

Before writing any implementation code, create comprehensive tests that will fail. For each test:
- Write a clear docstring explaining **why** this test is important
- Document **what specific behavior** is being tested
- Include edge cases and error conditions
- Ensure tests are isolated and independent

#### 2.2 GREEN Phase - Minimal Implementation

Write the simplest code that makes all tests pass. This means:
- Avoid premature optimization
- Don't add features not required by current tests
- Focus on correctness over elegance
- Keep the implementation straightforward

#### 2.3 REFACTOR Phase - Improve Code Quality

With all tests passing, refactor to:
- Improve code readability
- Extract common functionality
- Apply design patterns where appropriate
- Ensure consistent coding style
- Add necessary documentation

### Phase 3: Project Completion and Documentation

#### 3.1 Update Project Plan
Modify `plan.md` to reflect the completed implementation:
- Change status from "pending" to "implemented"
- Add implementation date
- Include brief notes about any decisions or trade-offs made
- Document any discovered dependencies or follow-up tasks

#### 3.2 Version Control
Create a comprehensive commit with the following format:
```
feat: Implement [feature name] from plan.md

- Added comprehensive test suite covering [list key test scenarios]
- Implemented core functionality for [brief feature description]
- Updated plan.md status to 'implemented'
- Follows TDD methodology (RED-GREEN-REFACTOR)

Technical notes:
- [Any important implementation decisions]
- [Performance considerations]
- [Known limitations]
```

#### 3.3 Build Verification
For compiled languages (Go, Rust, TypeScript, etc.):
- Build the entire application after implementation
- Fix any compilation errors
- Run the full test suite
- Verify the feature works in the built application
- Do not consider the feature complete until the build succeeds

## Progress Tracking and State Management

After completing the feature implementation and committing all changes, create a state tracking file at `./agent_state/agent_state.json`:

```json
{
    "current_step_description": "Implemented user authentication feature from plan.md with full test coverage",
    "next_step_prompt": "/run_implementation_loop",
    "status": "running"
}
```

### Command Options Explained:

**`/run_implementation_loop`** - Default command to continue with the next unimplemented feature from plan.md. Use this when there are more features to implement.

**`/verify_plan`** - Use only when ALL features in plan.md have been implemented. This triggers a comprehensive verification cycle to ensure nothing was missed.

**`/other <detailed description>`** - Use for special cases that don't fit the standard flow, such as:
- Addressing critical bugs discovered during implementation
- Refactoring required before continuing
- Infrastructure changes needed

### Status Rules:
- Use `"running"` when more work remains (with `/run_implementation_loop` or `/other`)
- Use `"completed"` only when the entire plan.md is fully implemented and verified
- Never combine `"completed"` status with `/run_implementation_loop` or `/verify_plan`

## Continuous Learning and Documentation

When you discover new patterns, commands, or solutions during implementation:

### Update CLAUDE.md
Use a subagent to document:
- Correct command syntax you've learned through trial
- Build or test commands specific to this project
- Environment setup steps
- Common error resolutions

### Add a spec
- If a new design pattern is emerging consider writing a new spec in @/specs. Leep is concise.

## Critical Success Factors

1. **Single Feature Focus**: Implement exactly ONE feature per cycle. Resist the temptation to implement multiple features, even if they seem related.

2. **Verification Over Trust**: Always verify implementation status in the codebase. The plan.md status might be outdated or incorrect.

3. **Test-First Discipline**: Never write implementation code before tests. This ensures you understand the requirements fully.

4. **Minimal Effective Implementation**: Write the least code necessary to fulfill the feature requirements. You can always enhance later.

5. **Clear Documentation**: Your future self (or another developer) should understand what was implemented and why certain decisions were made.

## Common Pitfalls to Avoid

- Don't implement features not listed in plan.md
- Don't skip the test-writing phase
- Don't over-engineer the solution
- Don't forget to update plan.md after implementation
- Don't mark status as "completed" prematurely