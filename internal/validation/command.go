package validation

import (
	"sort"
	"strings"
)

// commandValidator implements CommandValidator interface
type commandValidator struct{}

// NewCommandValidator creates a new command validator
func NewCommandValidator() CommandValidator {
	return &commandValidator{}
}

// CompareCommands compares Python and Go command arrays
func (v *commandValidator) CompareCommands(pythonCmd, goCmd []string) ComparisonResult {
	pythonComponents := v.ExtractComponents(pythonCmd)
	goComponents := v.ExtractComponents(goCmd)

	result := ComparisonResult{
		Match:       true,
		Differences: []Difference{},
	}

	// Compare executables
	if pythonComponents.Executable != goComponents.Executable {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "executable",
			PythonValue: pythonComponents.Executable,
			GoValue:     goComponents.Executable,
			Description: "Executable mismatch",
		})
	}

	// Compare MCP servers
	if !mapsEqual(pythonComponents.MCPServers, goComponents.MCPServers) {
		result.Match = false
		for name, pythonPath := range pythonComponents.MCPServers {
			if goPath, exists := goComponents.MCPServers[name]; !exists || goPath != pythonPath {
				result.Differences = append(result.Differences, Difference{
					Type:        "mcp_server",
					PythonValue: name + "=" + pythonPath,
					GoValue:     name + "=" + goPath,
					Description: "MCP server mismatch",
				})
			}
		}
	}

	// Compare tool allowlists (order independent)
	if !slicesEqualUnordered(pythonComponents.ToolAllowlist, goComponents.ToolAllowlist) {
		result.Match = false
		pythonSet := make(map[string]bool)
		goSet := make(map[string]bool)
		
		for _, tool := range pythonComponents.ToolAllowlist {
			pythonSet[tool] = true
		}
		for _, tool := range goComponents.ToolAllowlist {
			goSet[tool] = true
		}

		// Find differences
		for tool := range pythonSet {
			if !goSet[tool] {
				result.Differences = append(result.Differences, Difference{
					Type:        "tool_allowlist",
					PythonValue: tool,
					GoValue:     "",
					Description: "Tool allowlist mismatch",
				})
			}
		}
		for tool := range goSet {
			if !pythonSet[tool] {
				result.Differences = append(result.Differences, Difference{
					Type:        "tool_allowlist",
					PythonValue: "",
					GoValue:     tool,
					Description: "Tool allowlist mismatch",
				})
			}
		}
		
		// Special case: if counts match but tools differ, show the difference
		if len(pythonComponents.ToolAllowlist) == len(goComponents.ToolAllowlist) &&
			len(pythonComponents.ToolAllowlist) > 0 && len(result.Differences) > 0 {
			// Clear and show specific mismatches
			result.Differences = []Difference{}
			for i := range pythonComponents.ToolAllowlist {
				if i < len(goComponents.ToolAllowlist) && 
					pythonComponents.ToolAllowlist[i] != goComponents.ToolAllowlist[i] {
					result.Differences = append(result.Differences, Difference{
						Type:        "tool_allowlist",
						PythonValue: pythonComponents.ToolAllowlist[i],
						GoValue:     goComponents.ToolAllowlist[i],
						Description: "Tool allowlist mismatch",
					})
					break // Show only first difference for clarity
				}
			}
		}
	}

	// Compare output format
	if pythonComponents.OutputFormat != goComponents.OutputFormat {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "output_format",
			PythonValue: pythonComponents.OutputFormat,
			GoValue:     goComponents.OutputFormat,
			Description: "Output format mismatch",
		})
	}

	// Compare system prompts
	if pythonComponents.SystemPrompt != goComponents.SystemPrompt {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "system_prompt",
			PythonValue: pythonComponents.SystemPrompt,
			GoValue:     goComponents.SystemPrompt,
			Description: "System prompt mismatch",
		})
	}

	// Compare user prompts
	if pythonComponents.UserPrompt != goComponents.UserPrompt {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Type:        "user_prompt",
			PythonValue: pythonComponents.UserPrompt,
			GoValue:     goComponents.UserPrompt,
			Description: "User prompt mismatch",
		})
	}

	return result
}

// ExtractComponents parses a command array into its components
func (v *commandValidator) ExtractComponents(cmd []string) CommandComponents {
	components := CommandComponents{
		MCPServers:    make(map[string]string),
		ToolAllowlist: []string{},
	}

	if len(cmd) == 0 {
		return components
	}

	components.Executable = cmd[0]
	
	i := 1
	for i < len(cmd) {
		arg := cmd[i]
		
		if strings.HasPrefix(arg, "--mcp-server-") {
			// Extract MCP server name
			serverName := strings.TrimPrefix(arg, "--mcp-server-")
			if i+1 < len(cmd) {
				components.MCPServers[serverName] = cmd[i+1]
				i += 2
			} else {
				i++
			}
		} else if arg == "--tool-allowlist" {
			if i+1 < len(cmd) {
				components.ToolAllowlist = append(components.ToolAllowlist, cmd[i+1])
				i += 2
			} else {
				i++
			}
		} else if arg == "--output" {
			if i+1 < len(cmd) {
				components.OutputFormat = cmd[i+1]
				i += 2
			} else {
				i++
			}
		} else if arg == "--system" {
			if i+1 < len(cmd) {
				components.SystemPrompt = cmd[i+1]
				i += 2
			} else {
				i++
			}
		} else if !strings.HasPrefix(arg, "--") {
			// This is the user prompt (all remaining non-flag arguments)
			userPromptParts := []string{}
			for j := i; j < len(cmd); j++ {
				if !strings.HasPrefix(cmd[j], "--") {
					userPromptParts = append(userPromptParts, cmd[j])
				}
			}
			components.UserPrompt = strings.Join(userPromptParts, " ")
			break
		} else {
			// Skip unknown flags
			if i+1 < len(cmd) && !strings.HasPrefix(cmd[i+1], "--") {
				i += 2
			} else {
				i++
			}
		}
	}

	return components
}

// Helper functions

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

func slicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	
	// Sort copies to compare
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)
	
	sort.Strings(aCopy)
	sort.Strings(bCopy)
	
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	
	return true
}