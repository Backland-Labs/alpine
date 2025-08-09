package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type HookData struct {
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	Tool           string          `json:"tool"`
	ToolInput      json.RawMessage `json:"tool_input"`
	Args           json.RawMessage `json:"args"`
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	StopHookActive bool            `json:"stop_hook_active"`
}

type TodoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

type ToolInput struct {
	FilePath    string `json:"file_path"`
	Command     string `json:"command"`
	Pattern     string `json:"pattern"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	Query       string `json:"query"`
	Description string `json:"description"`
}

func main() {
	// Read JSON input from Claude Code
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return // Exit gracefully
	}

	var data HookData
	if err := json.Unmarshal(input, &data); err != nil {
		return // Exit gracefully on invalid JSON
	}

	// Get timestamp
	timestamp := time.Now().Format("15:04:05")

	// Check if this is a subagent:stop event
	if data.HookEventName == "SubagentStop" {
		handleSubagentStop(&data, timestamp)
		return
	}

	// Get tool name from either field
	toolName := data.ToolName
	if toolName == "" {
		toolName = data.Tool
	}

	// Get tool input from either field
	var toolInputRaw json.RawMessage
	if len(data.ToolInput) > 0 {
		toolInputRaw = data.ToolInput
	} else if len(data.Args) > 0 {
		toolInputRaw = data.Args
	}

	// Process and display all tool calls
	switch toolName {
	case "TodoWrite":
		handleTodoWrite(toolInputRaw, timestamp)
	case "Read":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.FilePath != "" {
			fmt.Fprintf(os.Stderr, "[%s] [READ] Reading file: %s\n", timestamp, input.FilePath)
		}
	case "Write":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.FilePath != "" {
			fmt.Fprintf(os.Stderr, "[%s] [WRITE] Writing file: %s\n", timestamp, input.FilePath)
		}
	case "Edit", "MultiEdit":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.FilePath != "" {
			fmt.Fprintf(os.Stderr, "[%s] [EDIT] Editing file: %s\n", timestamp, input.FilePath)
		}
	case "Bash":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.Command != "" {
			fmt.Fprintf(os.Stderr, "[%s] [BASH] Executing: %s\n", timestamp, input.Command)
		}
	case "Grep":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.Pattern != "" {
			path := input.Path
			if path == "" {
				path = "."
			}
			fmt.Fprintf(os.Stderr, "[%s] [GREP] Searching for '%s' in %s\n", timestamp, input.Pattern, path)
		}
	case "Glob":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.Pattern != "" {
			path := input.Path
			if path == "" {
				path = "."
			}
			fmt.Fprintf(os.Stderr, "[%s] [GLOB] Finding files matching '%s' in %s\n", timestamp, input.Pattern, path)
		}
	case "LS":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.Path != "" {
			fmt.Fprintf(os.Stderr, "[%s] [LS] Listing directory: %s\n", timestamp, input.Path)
		}
	case "WebFetch":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.URL != "" {
			fmt.Fprintf(os.Stderr, "[%s] [WEB] Fetching: %s\n", timestamp, input.URL)
		}
	case "WebSearch":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.Query != "" {
			fmt.Fprintf(os.Stderr, "[%s] [SEARCH] Searching web for: %s\n", timestamp, input.Query)
		}
	case "Task":
		var input ToolInput
		if err := json.Unmarshal(toolInputRaw, &input); err == nil && input.Description != "" {
			fmt.Fprintf(os.Stderr, "[%s] [TASK] Launching agent: %s\n", timestamp, input.Description)
		}
	case "":
		// No tool name, ignore
	default:
		// Other tools - show generic message
		fmt.Fprintf(os.Stderr, "[%s] [TOOL] Using: %s\n", timestamp, toolName)
	}
}

func handleTodoWrite(toolInputRaw json.RawMessage, timestamp string) {
	var input TodoWriteInput
	if err := json.Unmarshal(toolInputRaw, &input); err != nil {
		return
	}

	// Count todo statuses
	var pending, inProgress, completed int
	var currentTask string

	for _, todo := range input.Todos {
		switch todo.Status {
		case "pending":
			pending++
		case "in_progress":
			inProgress++
			if currentTask == "" {
				currentTask = todo.Content
			}
		case "completed":
			completed++
		}
	}

	// Display todo summary
	fmt.Fprintf(os.Stderr, "[%s] [TODO] Updated - Completed: %d, In Progress: %d, Pending: %d\n",
		timestamp, completed, inProgress, pending)

	// Display current task if any
	if currentTask != "" {
		fmt.Fprintf(os.Stderr, "[%s] [TODO] Current task: %s\n", timestamp, currentTask)

		// Write to todo file if environment variable is set
		if todoFile := os.Getenv("ALPINE_TODO_FILE"); todoFile != "" {
			os.WriteFile(todoFile, []byte(currentTask), 0644)
		}
	}
}

func handleSubagentStop(data *HookData, timestamp string) {
	// Extract subagent stop information
	sessionID := data.SessionID
	if sessionID == "" {
		sessionID = "unknown"
	}

	transcriptPath := data.TranscriptPath
	if transcriptPath == "" {
		transcriptPath = "unknown"
	}

	fmt.Fprintf(os.Stderr, "[%s] [AGENT] Subagent completed - Session: %s\n", timestamp, sessionID)

	// Only process transcript if stop_hook_active is false to prevent loops
	if !data.StopHookActive && transcriptPath != "unknown" {
		// Could process the transcript file here if needed
		fmt.Fprintf(os.Stderr, "[%s] [AGENT] Transcript saved to: %s\n", timestamp, transcriptPath)
	}
}
