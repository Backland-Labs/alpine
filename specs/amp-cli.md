# Amp CLI Integration Specification

## Overview

Alpine can use Amp Code CLI to execute AI-assisted coding tasks by piping text prompts to the `amp` command. This provides an alternative to Claude Code with features like extended thinking and faster execution.

## Simple Integration Approach

### Basic Command Execution

```go
// ExecuteWithAmp sends a text prompt to Amp CLI and streams the output
func ExecuteWithAmp(prompt string, workDir string) error {
    // Amp accepts piped input through stdin
    cmd := exec.Command("amp", "--dangerously-allow-all")
    cmd.Stdin = strings.NewReader(prompt)
    cmd.Stdout = os.Stdout  // Stream output directly to terminal
    cmd.Stderr = os.Stderr  
    cmd.Dir = workDir       // Execute in the target directory
    
    // Pass API key through environment
    cmd.Env = append(os.Environ(), "AMP_API_KEY=" + os.Getenv("ALPINE_AMP_API_KEY"))
    
    return cmd.Run()
}
```

### Practical Usage Pattern

```go
// Example: Using Amp to implement a feature
prompt := `
Task: Implement user authentication with JWT tokens

Please:
1. Create the authentication middleware
2. Add JWT token generation and validation
3. Update the state file (agent_state.json) when done

Set status to "completed" in the state file when finished.
`

err := ExecuteWithAmp(prompt, "/path/to/project")
```

## Key Implementation Details

### API Key Handling

```go
// Simple API key validation before execution
func validateAmpAPIKey() error {
    apiKey := os.Getenv("ALPINE_AMP_API_KEY")
    if apiKey == "" {
        return fmt.Errorf("missing ALPINE_AMP_API_KEY environment variable")
    }
    return nil
}

// Usage in main flow
if err := validateAmpAPIKey(); err != nil {
    log.Fatal("Amp setup error:", err)
}
```

## Practical Examples

### Simple Task Execution

```bash
# Set up API key
export ALPINE_AMP_API_KEY="your_amp_api_key"

# Execute a simple task
echo "Fix the TypeScript errors in src/auth.ts" | amp --dangerously-allow-all
```

### Alpine Integration

```go
// Minimal Amp executor for Alpine
type AmpExecutor struct {
    workDir string
}

func (a *AmpExecutor) Execute(taskPrompt string) error {
    // Build the full prompt with state management instructions
    fullPrompt := fmt.Sprintf(`
Task: %s

Update agent_state.json after each significant step.
Set status to "completed" when done, or "running" with next steps.
`, taskPrompt)
    
    // Create the command
    cmd := exec.Command("amp", "--dangerously-allow-all")
    cmd.Stdin = strings.NewReader(fullPrompt)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Dir = a.workDir
    
    // Set environment
    cmd.Env = append(os.Environ(), 
        "AMP_API_KEY=" + os.Getenv("ALPINE_AMP_API_KEY"),
    )
    
    // Execute and return
    return cmd.Run()
}
```

### Monitoring State Changes

```go
// Watch for state file updates to know when Amp is done
func waitForCompletion(stateFile string) error {
    for {
        data, err := os.ReadFile(stateFile)
        if err != nil {
            return err
        }
        
        var state struct {
            Status string `json:"status"`
        }
        
        if err := json.Unmarshal(data, &state); err != nil {
            return err
        }
        
        if state.Status == "completed" {
            return nil
        }
        
        time.Sleep(2 * time.Second)
    }
}
```

## Utility and Use Cases

### When to Use Amp vs Claude

```go
// Decision logic for CLI selection
func selectCLI(task string) string {
    // Use Amp for tasks that benefit from:
    // - Extended thinking capabilities
    // - Faster execution speed
    // - Subagent spawning for parallel work
    
    complexityIndicators := []string{
        "refactor", "architecture", "complex", "entire", "system",
    }
    
    taskLower := strings.ToLower(task)
    for _, indicator := range complexityIndicators {
        if strings.Contains(taskLower, indicator) {
            return "amp"  // Complex task - use Amp
        }
    }
    
    return "claude"  // Default to Claude for simple tasks
}
```

### Error Handling

```go
// Handle common Amp errors
func executeWithRetry(prompt string, workDir string) error {
    maxRetries := 3
    
    for i := 0; i < maxRetries; i++ {
        err := ExecuteWithAmp(prompt, workDir)
        
        if err == nil {
            return nil
        }
        
        // Check for specific errors
        if strings.Contains(err.Error(), "API key") {
            return fmt.Errorf("amp authentication failed: check ALPINE_AMP_API_KEY")
        }
        
        if strings.Contains(err.Error(), "rate limit") {
            log.Printf("Rate limited, waiting 30 seconds...")
            time.Sleep(30 * time.Second)
            continue
        }
        
        // Unknown error, retry
        log.Printf("Attempt %d failed: %v", i+1, err)
    }
    
    return fmt.Errorf("amp execution failed after %d attempts", maxRetries)
}
```

## Quick Reference

### Environment Setup
```bash
# Required
export ALPINE_AMP_API_KEY="your_key"

# Optional
export ALPINE_USE_AMP=true  # Flag to prefer Amp over Claude
```

### Minimal Usage
```go
// The simplest possible integration
cmd := exec.Command("amp", "--dangerously-allow-all")
cmd.Stdin = strings.NewReader("Your task here")
cmd.Stdout = os.Stdout
cmd.Env = append(os.Environ(), "AMP_API_KEY=" + apiKey)
cmd.Run()
```

### Key Points

1. **Piped Input**: Amp reads prompts from stdin, not command arguments
2. **API Key**: Must be in AMP_API_KEY environment variable
3. **State Updates**: Must explicitly instruct Amp to update state files
4. **Output**: Streams directly to stdout/stderr
5. **Working Directory**: Set cmd.Dir to control where Amp operates

