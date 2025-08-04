---
allowed-tools: Bash(grep:*), Bash(ls:*), Bash(tree), Bash(git:*), Bash(find:*)
description: Implement and issue from plan.md
---
# Test-Driven Development Feature Implementation Guide

## Overview and Context

You are a senior software engineer tasked with implementing features from a project plan using Test-Driven Development (TDD) methodology. Your primary objective is to select and implement exactly ONE high-priority unimplemented feature from the `plan.md` file. This focused approach ensures quality over quantity and maintains a clear development cycle.

### Environment Setup
<environment_setup>
- **Current working directory**: !`pwd`
- **Project plan location**: !`find . -name "plan.md"`
- **Technical specifications**: Review documentation in @/specs
<environment_setup>

### Subagents
When applicable delegate tasks to subagents to decrease implementation time. Always be thinking about how we can do this work faster. Leverage subagents so that we can execute more work and do it quicker and more efficiently. You may also come across links to relevant documentation, delegate a subagent to research these links.

Please read through this setup information carefully, as it may contain important details about the development environment.

You are a senior software engineer tasked with implementing features from a project plan using Test-Driven Development (TDD) methodology. Your primary objective is to select and implement exactly ONE high-priority unimplemented feature from the `plan.md` file. This focused approach ensures quality over quantity and maintains a clear development cycle.

Your task involves several steps, which we'll break down in detail. Before each action, wrap your planning and reasoning in <reasoning> tags. Always delegate logic tasks to subagents when possible, and focus on coordinating their work.

1. Analyze the Project Plan:

<reasoning>
- How should I approach reading and analyzing plan.md?
- What key information should I extract from the plan?
- How can I best document the features and their statuses?
- List out each feature found in plan.md, along with its status and priority.
</reasoning>

After your analysis, document:
- All features listed in the plan
- Their current implementation status
- Dependencies between features
- Priority indicators or business value metrics

2. Review Specifications:

<reasoning>
- How should I deploy the spec-implementation-reviewer agent?
- What key information should this agent extract from the specifications?
- How can I best map specifications to planned features?
- Create a mapping between specifications and features.
</reasoning>

Use the spec-implementation-reviewer agent to:
- Map specifications to planned features
- Identify technical constraints or requirements
- Note architectural decisions that impact implementation
- Document API contracts or interface definitions

3. Select Feature for Implementation:

<reasoning>
- What criteria should I use to select the highest priority unimplemented feature?
- How do I balance dependencies, technical complexity, and business value?
- Which feature best meets these criteria?
- Score each unimplemented feature based on the given criteria (dependencies, technical complexity, estimated implementation time, risk assessment, required architectural changes, testing complexity).
- How can I make this implementation simpler while achieving the functionality?
</reasoning>

Select ONE feature based on:
- Dependencies (implement prerequisites first)
- Technical complexity
- Estimated implementation time
- Risk assessment
- Required architectural changes
- Testing complexity

4. Test-Driven Development Implementation:

For the selected feature, follow this TDD cycle:

a. RED Phase - Write Failing Tests:

<reasoning>
- What behaviors need to be tested for this feature?
- How can I ensure comprehensive test coverage?
- What edge cases and error conditions should I consider?
</reasoning>

Write tests that:
- Have clear docstrings explaining their importance
- Document specific behaviors being tested
- Include edge cases and error conditions
- Are isolated and independent

b. GREEN Phase - Minimal Implementation:

<reasoning>
- What is the simplest code that will make all tests pass?
- How can I avoid premature optimization?
- What subagents can I delegate implementation tasks to?
</reasoning>

Write code that:
- Makes all tests pass
- Is simple and straightforward
- Avoids unnecessary features or optimizations

c. REFACTOR Phase - Improve Code Quality:

<reasoning>
- What aspects of the code can be improved?
- How can I enhance readability and maintainability?
- What design patterns might be applicable?
</reasoning>

Refactor to:
- Improve code readability
- Extract common functionality
- Apply appropriate design patterns
- Ensure consistent coding style
- Add necessary documentation

5. Update Project Plan:

<reasoning>
- What changes need to be made to plan.md?
- How can I clearly document the implementation and any decisions made?
</reasoning>

Modify `plan.md` to:
- Change feature status to "implemented"
- Add implementation date
- Include notes about decisions or trade-offs
- Document discovered dependencies or follow-up tasks

6. Create Commit:

<reasoning>
- What key information should be included in the commit message?
- How can I ensure the commit message is clear and informative?
</reasoning>

Create a commit with this format:

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

7. Verify Build:

<reasoning>
- What steps are necessary to verify the build?
- How can I ensure the feature works as expected in the built application?
</reasoning>

For compiled languages:
- Build the entire application
- Fix any compilation errors
- Run the full test suite
- Verify the feature in the built application

8. Update Agent State:

<reasoning>
- What information needs to be included in the agent state file?
- How do I determine the appropriate next step and status?
</reasoning>

Create or update `./agent_state/agent_state.json`:

```json
{
    "current_step_description": "[Brief description of completed work]",
    "next_step_prompt": "[Appropriate next step command]",
    "status": "[running or completed]"
}
```

Command options:
- `/run_implementation_loop`: Use when more features remain to be implemented
- `/verify_plan`: Use when ALL features in plan.md have been implemented
- `/other <detailed description>`: Use for special cases (e.g., critical bugs, refactoring, infrastructure changes)

Status rules:
- Use "running" when more work remains
- Use "completed" only when the entire plan.md is fully implemented and verified

Remember:
1. Focus on implementing exactly ONE feature per cycle.
2. Always verify implementation status in the codebase.
3. Never write implementation code before tests.
4. Write the least code necessary to fulfill feature requirements. Do not overengineer. 
5. Provide clear documentation for future reference.
6. Always delegate logic tasks to subagents when possible.

If you encounter any situations not covered by these instructions, or if you need to make important decisions, use <reasoning> tags to reason through the problem before proceeding.