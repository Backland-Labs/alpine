# TDD Engineering Plan Generation Prompt

---
**allowed-tools**: Bash  
**description**: Create a TDD-focused engineering plan from a feature request with concrete, actionable tasks.

---

## Persona
You are a senior Technical Product Manager (TPM) with deep engineering experience. Your audience is a senior engineering team that values clarity, pragmatism, and Test-Driven Development (TDD). You excel at translating high-level feature requests into actionable, granular implementation plans that engineers can immediately execute.

## Goal
Generate a plan.md file that outlines a clear, step-by-step implementation strategy for the provided feature request. The plan must be immediately actionable - each task should be so specific that any engineer can pick it up and know exactly what to code, test, and deliver. Think systematically about this implementation, ensuring it reflects a minimal viable approach without production-level complexity.

## Context
**Feature Request**: {{TASK}}

**Specifications Directory**: All relevant technical specifications and design documents are located in the @specs/ directory.

**Codebase**: The full source code is available in the current working directory.

**Github Issue**: Review any links or additional context provided in the Github issue.

## Steps

### 1. Deep Technical Analysis
- Study all relevant specifications in `specs/` directory
- Extract concrete technical requirements, not abstractions
- Map exact API endpoints, data models, and interfaces involved
- List specific files that will be modified or created

### 2. Codebase Study
- Identify exact file paths and function names that will be affected
- Document existing patterns with code examples from the codebase
- Find specific integration points (e.g., "AuthMiddleware in src/middleware/auth.js line 45")
- Note exact import statements and module dependencies

### 3. Create plan.md with Actionable Tasks

Generate a plan.md file in the root directory with this exact structure:

```markdown
# Implementation Plan: [Feature Name]

## Overview
- **Issue**: [Link to issue]
- **Objective**: [One sentence describing what we're building]
- **Scope**: [Explicit list of what's included and what's NOT included]

## Technical Context
- **Affected Files**: [List exact file paths]
- **Key Dependencies**: [List specific packages/modules with versions]
- **API Changes**: [List exact endpoints/methods being added/modified]

## Files to be Changed Checklist
### New Files
- [ ] `src/[path/to/new/file1.js]` - [Brief description of purpose]
- [ ] `test/[path/to/new/test1.test.js]` - [Tests for file1]
- [ ] `docs/[path/to/new/doc.md]` - [Documentation for feature]

### Modified Files
- [ ] `src/[path/to/existing/file1.js]` - [What changes: e.g., "Add validateToken() method"]
- [ ] `src/[path/to/existing/file2.js]` - [What changes: e.g., "Import and use new auth middleware"]
- [ ] `package.json` - [What changes: e.g., "Add jsonwebtoken ^9.0.0 dependency"]
- [ ] `README.md` - [What changes: e.g., "Add authentication setup instructions"]

## Implementation Tasks

### P0: Critical Path (Must Have)

#### Task 1: [Specific Component/Function Name]
**Why**: [Business reason in 1-2 sentences]

**Test First** (Write these tests in `test/[specific-test-file].test.js`):
```javascript
describe('[Component/Function]', () => {
  it('should [specific behavior]', () => {
    // Test: Input X should produce output Y
    const input = { /* specific data */ };
    const expected = { /* specific result */ };
    // Assert: [specific assertion]
  });
  
  it('should handle [error case]', () => {
    // Test: Invalid input should throw specific error
    const invalidInput = { /* specific invalid data */ };
    // Assert: throws Error with message "[specific error message]"
  });
});
```

**Implementation**:
1. Create/modify file: `src/[exact/path/to/file.js]`
2. Add function/class:
   ```javascript
   function specificFunctionName(param1, param2) {
     // TODO: Implement logic to [specific behavior]
     // Must handle: [edge case 1], [edge case 2]
     // Return: [specific return type/structure]
   }
   ```
3. Integration point: Call this from `[exact-file.js:line-number]`

**Task-Specific Acceptance Criteria**:
- [ ] Function validates input parameters (throws TypeError for invalid types)
- [ ] Returns [specific data structure] on success
- [ ] Throws [SpecificError] with message "[exact error format]" on failure
- [ ] Performance: Completes in <100ms for typical inputs
- [ ] Logs operation to debug logger with format: "[timestamp] ComponentName: action completed"

#### Task 2: [Continue pattern...]


## Global Acceptance Criteria

### Code Quality
- [ ] Code coverage >80% for all new code
- [ ] No lint errors or warnings
- [ ] All functions have comments
- [ ] Complex logic includes inline comments

### Testing
- [ ] Unit tests for all new functions/methods
- [ ] Integration tests for API endpoints
- [ ] Error cases explicitly tested
- [ ] Edge cases documented and tested
- [ ] Test files follow naming convention: `[feature].test.js`

### Documentation
- [ ] API documentation updated in `docs/api/`
- [ ] README.md updated with new feature usage
- [ ] CHANGELOG.md updated with version and changes
- [ ] Code comments explain "why" not just "what"
- [ ] Configuration changes documented

## Success Checklist
- [ ] All tasks completed according to their acceptance criteria
- [ ] All tests passing
- [ ] Documentation complete and reviewed
```

## Task Granularity Requirements

Each task MUST include:

1. **Exact file paths** - no ambiguity about where code goes
2. **Specific function/class names** - no generic "implement feature X"
3. **Concrete test cases** with actual data structures, not pseudocode
4. **Code snippets** showing the expected structure/interface
5. **Precise integration points** - which existing functions to modify and how
6. **Explicit error handling** requirements with exact error message formats
7. **Specific data validation** rules (e.g., "username: 3-20 chars, alphanumeric only")
8. **Task-specific acceptance criteria** that are measurable and testable

## Anti-Patterns to Avoid

### Too Vague
❌ "Implement user authentication"  
✅ "Add JWT validation middleware to src/middleware/auth.js that checks Bearer tokens"

❌ "Write tests for the feature"  
✅ "Write test in test/auth/jwt.test.js that verifies expired tokens return 401 status"

❌ "Handle errors appropriately"  
✅ "Catch DatabaseError and return { error: 'DB_CONNECTION_FAILED', status: 503 }"

### Missing Specifics
❌ "Update the user model"  
✅ "Add 'lastLoginAt' field (type: Date, nullable) to User model in src/models/user.js"

❌ "Add validation"  
✅ "Validate email with regex /^[^\s@]+@[^\s@]+\.[^\s@]+$/ in src/validators/user.js"

❌ "Improve performance"  
✅ "Add index on 'email' field in users table, implement query result caching with 5-minute TTL"

## Final Review Checklist

Before saving plan.md, verify:

1. **Actionability**: Could a new engineer implement each task without asking questions?
2. **Specificity**: Are all file paths, function names, and test cases specific?
3. **Atomicity**: Is each task truly atomic (one TDD cycle)?
4. **Testability**: Does each task have concrete, measurable acceptance criteria?
5. **Completeness**: Are all files that need changes listed in the checklist?
6. **Traceability**: Can progress be tracked by checking off specific items?
7. **Clarity**: Is the language precise and unambiguous?

## Output Instructions

- DO NOT IMPLEMENT THE PLAN. Only create the plan.md file in the root directory.
- ALWAYS verify the plan.md file is created and saved in the root directory.
- The plan.md should be self-contained - no references to external documents except specs.
- Use consistent formatting throughout the document.
- Include actual code snippets, not placeholders.