# Prompts Module

This module manages prompt templates and generation for Claude Code interactions, including plan generation and task execution prompts.

## Module Overview

The prompts package provides:
- Template management for various prompt types
- Variable substitution and formatting
- Plan generation prompt construction
- Implementation loop prompts
- State transition prompts

## Key Components

### Prompts (`prompts.go`)
- Embedded prompt templates using `go:embed`
- Template variable substitution
- Prompt type enumeration
- Context-aware prompt generation

### Prompt Templates (`prompt-plan.md`)
- Markdown-based prompt templates
- Variable placeholders for dynamic content
- Structured output requirements
- Claude-specific instructions

## Prompt Types

### Plan Generation Prompt
Used for `/make_plan` slash command:
- Accepts task description
- Optionally includes GitHub issue context
- Generates structured markdown plan
- Includes implementation steps and success criteria

### Implementation Loop Prompt
Used for `/run_implementation_loop`:
- References generated plan
- Includes current state context
- Drives iterative implementation
- Updates state file for progress tracking

### Verification Prompt
Used for `/verify_plan`:
- Checks plan completion
- Validates success criteria
- Identifies remaining tasks
- Provides completion summary

## Template Variables

Common variables used in prompts:
```go
type PromptVars struct {
    TaskDescription string
    IssueContext    string
    PlanContent     string
    CurrentState    string
    WorkingDir      string
}
```

## Prompt Engineering Guidelines

### Structure
1. **Context Setting**: Provide necessary background
2. **Clear Instructions**: Explicit, unambiguous directions
3. **Output Format**: Specify expected response structure
4. **Constraints**: Define boundaries and limitations
5. **Examples**: Include when format is critical

### Best Practices
- Keep prompts concise but complete
- Use markdown for better readability
- Include error handling instructions
- Specify output format explicitly
- Test with edge cases

### Token Optimization
- Remove redundant instructions
- Use references instead of repetition
- Compress context when possible
- Prioritize essential information
- Monitor token usage in testing

## Implementation Patterns

### Loading Templates
```go
//go:embed prompt-plan.md
var planPromptTemplate string

func GetPlanPrompt(vars PromptVars) string {
    return substituteVars(planPromptTemplate, vars)
}
```

### Variable Substitution
```go
func substituteVars(template string, vars PromptVars) string {
    result := template
    result = strings.ReplaceAll(result, "{{TASK}}", vars.TaskDescription)
    result = strings.ReplaceAll(result, "{{ISSUE}}", vars.IssueContext)
    return result
}
```

### Dynamic Prompt Construction
```go
func BuildPrompt(taskType string, context map[string]string) string {
    base := getBasePrompt(taskType)
    for key, value := range context {
        base = strings.ReplaceAll(base, fmt.Sprintf("{{%s}}", key), value)
    }
    return base
}
```

## Prompt Templates

### Plan Generation Template Structure
```markdown
# Task
{{TASK_DESCRIPTION}}

# Context
{{ISSUE_CONTEXT}}

# Instructions
Generate a detailed implementation plan...

# Output Format
## Plan
### Steps
1. ...
2. ...

### Success Criteria
- [ ] ...
```

### State Update Template
```markdown
# Current State
{{CURRENT_STATE}}

# Next Action
{{NEXT_STEP}}

Update the state file with:
- current_step_description
- next_step_prompt
- status (running/completed)
```

## Testing Requirements

### Prompt Validation
- Ensure all variables are substituted
- Verify markdown formatting
- Check token count limits
- Test with various input lengths

### Output Testing
- Validate Claude responses match expected format
- Test error cases and edge conditions
- Verify state transitions work correctly
- Check plan structure compliance

## Quality Guidelines

### Clarity
- Avoid ambiguous language
- Use specific, actionable instructions
- Define technical terms when needed
- Provide clear success criteria

### Consistency
- Maintain uniform style across prompts
- Use consistent variable naming
- Follow established patterns
- Keep formatting standardized

### Maintainability
- Document prompt purpose and usage
- Version control prompt changes
- Test prompt modifications thoroughly
- Keep templates modular and reusable

## Performance Considerations

- Template loading happens once at startup
- Variable substitution is lightweight
- Avoid runtime template compilation
- Cache processed prompts when possible
- Monitor prompt token usage

## Common Patterns

### Conditional Sections
```go
if includeIssue {
    prompt += "\n## GitHub Issue Context\n" + issueContent
}
```

### Multi-step Prompts
```go
prompts := []string{
    getPlanningPrompt(),
    getImplementationPrompt(),
    getVerificationPrompt(),
}
```

### Error Recovery Prompts
```go
if lastAttemptFailed {
    prompt = getErrorRecoveryPrompt(lastError)
}
```

## Integration Notes

- Prompts module is used by CLI and workflow packages
- No external dependencies (pure Go)
- Templates embedded in binary at compile time
- Supports both Gemini and Claude Code execution

## Future Enhancements

- Template versioning system
- A/B testing for prompt variations
- Dynamic prompt optimization
- Multi-language prompt support
- Prompt effectiveness metrics