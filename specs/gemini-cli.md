# Gemini CLI Integration Specification

This specification defines how Alpine can integrate with Gemini CLI as an alternative to Claude Code for AI-assisted development workflows.

## Overview

Gemini CLI is Google's command-line interface for interacting with Gemini models. While primarily designed for interactive use, it supports non-interactive execution through specific flags and patterns.

## Authentication Setup

### API Key Configuration

```bash
# Primary method - Gemini API
export GEMINI_API_KEY="your-gemini-api-key"
```

### Alpine Configuration

```go
// Check for Gemini API key in executor
apiKey := os.Getenv("GEMINI_API_KEY")
if apiKey == "" {
    return fmt.Errorf("GEMINI_API_KEY not set")
}
```

## Non-Interactive Execution Patterns

### Basic Single Prompt Execution

```bash
# Direct prompt flag (REQUIRED for non-interactive mode)
gemini -p "Generate a README for a Go CLI project"

# With specific model
gemini --model gemini-1.5-pro-latest -p "Implement error handling"

# Piping input (alternative method)
echo "Explain this code: $(cat main.go)" | gemini
```

### Alpine Integration Example

```go
func executeGeminiCommand(prompt string, workDir string) error {
    // Build command with non-interactive flag
    cmd := exec.Command("gemini", "-p", prompt)
    cmd.Dir = workDir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    // Set environment to ensure non-interactive mode
    cmd.Env = append(os.Environ(), 
        "GEMINI_API_KEY="+os.Getenv("GEMINI_API_KEY"),
    )
    
    // Execute and check exit code
    err := cmd.Run()
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            return fmt.Errorf("gemini failed with exit code %d", exitErr.ExitCode())
        }
        return err
    }
    
    return nil
}
```

## File Context Handling

### Including Files in Prompts

```bash
# Reference files using @ syntax
gemini -p "@main.go Refactor this code to follow Go best practices"

# Multiple files (concatenate in prompt)
gemini -p "@pkg/server.go @pkg/client.go Add comprehensive error handling"

# With explicit instructions
gemini -p "Review these files for security issues: @cmd/main.go @internal/auth.go"
```

### Alpine File Context Builder

```go
func buildGeminiPromptWithFiles(basePrompt string, files []string) string {
    var parts []string
    
    // Add file references
    for _, file := range files {
        parts = append(parts, fmt.Sprintf("@%s", file))
    }
    
    // Combine with base prompt
    if len(parts) > 0 {
        return fmt.Sprintf("%s %s", strings.Join(parts, " "), basePrompt)
    }
    return basePrompt
}
```

## Working with Output

### Capturing and Processing Output

```bash
# Save output to file
gemini -p "Generate unit tests for calculator.go" > calculator_test.go

# Process output in script
output=$(gemini -p "List all TODO items in the codebase")
echo "$output" | grep -E "TODO:|FIXME:" > todos.txt

# Check success and handle errors
if ! gemini -p "Validate JSON schema" > /dev/null 2>&1; then
    echo "Validation failed"
    exit 1
fi
```

### Alpine Output Handler

```go
func captureGeminiOutput(prompt string) (string, error) {
    cmd := exec.Command("gemini", "-p", prompt)
    
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    err := cmd.Run()
    if err != nil {
        return "", fmt.Errorf("gemini error: %s", stderr.String())
    }
    
    return stdout.String(), nil
}
```

## Limitations and Workarounds

### Key Limitations

1. **No Multi-Turn Conversations**
   ```bash
   # NOT SUPPORTED - Each invocation is independent
   gemini -p "Create a function"
   gemini -p "Now add error handling"  # Has no context of previous command
   ```

2. **No Session Persistence**
   ```bash
   # Workaround: Include context manually
   previous_output=$(gemini -p "Generate base structure")
   gemini -p "Based on this code: $previous_output, add logging"
   ```

3. **CI Environment Detection**
   ```bash
   # CI environments trigger interactive mode - must unset CI vars
   env -u CI -u CI_COMMIT_SHA gemini -p "Run analysis"
   ```

4. **Automatic Tool Execution**
   ```bash
   # Tools run without confirmation in -p mode
   # Use --yolo flag explicitly to acknowledge this behavior
   gemini --yolo -p "Create and modify files for new feature"
   ```

### Alpine-Specific Workarounds

```go
// Handle CI environment
func prepareGeminiEnvironment() []string {
    env := os.Environ()
    
    // Remove CI-related variables that trigger interactive mode
    filtered := []string{}
    for _, e := range env {
        if !strings.HasPrefix(e, "CI_") && !strings.HasPrefix(e, "CI=") {
            filtered = append(filtered, e)
        }
    }
    
    return filtered
}

// Simulate multi-turn with context injection
func executeWithContext(prompt string, previousOutput string) error {
    contextualPrompt := fmt.Sprintf(
        "Previous context:\n%s\n\nNew instruction: %s",
        previousOutput, prompt,
    )
    
    return executeGeminiCommand(contextualPrompt, ".")
}
```

## Comparison with Claude CLI

### Feature Comparison

| Feature | Claude CLI | Gemini CLI |
|---------|-----------|------------|
| Non-interactive flag | `--prompt` | `-p` |
| Continue from state | Not supported | Not supported |
| Session persistence | Built-in | Manual context |
| Multi-turn support | Yes | No |
| Tool confirmation | Yes | Auto-executes with `-p` |
| Output streaming | Yes | Yes |
| Exit codes | Reliable | Reliable |

### When to Use Gemini CLI

```go
// Good use cases for Gemini
func shouldUseGemini(task TaskType) bool {
    switch task {
    case SinglePromptGeneration:    // Simple, one-shot tasks
        return true
    case CodeReview:               // Stateless analysis
        return true
    case DocumentationGeneration:  // Single file outputs
        return true
    default:
        return false
    }
}

// Better suited for Claude
func requiresClaude(task TaskType) bool {
    switch task {
    case IterativeDevelopment:     // Needs context preservation
        return true
    case MultiStepRefactoring:     // Requires state management
        return true
    case InteractiveDebugging:     // Needs back-and-forth
        return true
    default:
        return false
    }
}
```

## Integration with Alpine State Management

### Adapting Alpine's State Model

```go
// Since Gemini doesn't support continuation, simulate it
type GeminiStateAdapter struct {
    stateFile   string
    contextFile string  // Store previous outputs for context
}

func (g *GeminiStateAdapter) ExecuteStep(state *ClaudeState) error {
    // Load previous context
    context, _ := g.loadContext()
    
    // Build prompt with context
    prompt := fmt.Sprintf(
        "Context from previous steps:\n%s\n\nCurrent task: %s",
        context,
        state.NextStepPrompt,
    )
    
    // Execute with gemini
    output, err := captureGeminiOutput(prompt)
    if err != nil {
        return err
    }
    
    // Append to context for next iteration
    return g.appendContext(output)
}
```

## Error Handling

### Robust Error Handling Pattern

```go
func executeGeminiWithRetry(prompt string, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        cmd := exec.Command("gemini", "-p", prompt)
        
        var stderr bytes.Buffer
        cmd.Stderr = &stderr
        
        err := cmd.Run()
        if err == nil {
            return nil
        }
        
        // Check for specific error types
        errMsg := stderr.String()
        
        if strings.Contains(errMsg, "rate limit") {
            time.Sleep(time.Second * time.Duration(i+1))
            continue
        }
        
        if strings.Contains(errMsg, "API key") {
            return fmt.Errorf("authentication error: %s", errMsg)
        }
        
        // Non-retryable error
        return fmt.Errorf("gemini error: %s", errMsg)
    }
    
    return fmt.Errorf("max retries exceeded")
}
```

## Security Considerations

### API Key Management

```bash
# Never hardcode keys
# Bad
gemini -p "Generate code" --api-key="hardcoded-key"

# Good - Use environment variable
export GEMINI_API_KEY="your-key"
gemini -p "Generate code"
```

### Safe Execution Pattern

```go
func safeGeminiExecution(prompt string) error {
    // Validate prompt doesn't contain injection attempts
    if strings.Contains(prompt, "$(") || strings.Contains(prompt, "`") {
        return fmt.Errorf("potential command injection detected")
    }
    
    // Use explicit tool restrictions if needed
    cmd := exec.Command("gemini", 
        "-p", prompt,
        "--extensions", "-all,+read,+write",  // Limit tools
    )
    
    return cmd.Run()
}
```

## Practical Usage Examples

### Complete Alpine Integration

```bash
#!/bin/bash
# alpine-gemini wrapper script

# Check for API key
if [ -z "$GEMINI_API_KEY" ]; then
    echo "Error: GEMINI_API_KEY not set"
    exit 1
fi

# Function to execute gemini with proper error handling
run_gemini() {
    local prompt="$1"
    local output_file="$2"
    
    # Handle CI environment
    env -u CI gemini -p "$prompt" > "$output_file" 2>gemini_error.log
    
    if [ $? -ne 0 ]; then
        echo "Gemini execution failed:"
        cat gemini_error.log
        return 1
    fi
    
    return 0
}

# Example usage
if run_gemini "Generate Go HTTP server boilerplate" server.go; then
    echo "Successfully generated server.go"
    
    # Follow-up with context
    if run_gemini "@server.go Add graceful shutdown handling" server_v2.go; then
        mv server_v2.go server.go
        echo "Enhanced server with graceful shutdown"
    fi
fi
```

## Summary

Gemini CLI can be integrated with Alpine for single-prompt, stateless AI tasks. While it lacks Claude's session management and continuation features, it's suitable for:

- One-shot code generation
- Stateless code review
- Documentation generation
- Simple refactoring tasks

For complex, multi-step workflows requiring state preservation, Claude CLI remains the better choice. Consider Gemini CLI as a complementary tool for specific use cases rather than a complete replacement.