package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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

// Logger provides structured logging for the TODO monitor hook
type Logger struct {
	runID   string
	verbose bool
}

func newLogger() *Logger {
	runID := os.Getenv("ALPINE_RUN_ID")
	if runID == "" {
		runID = "unknown"
	}
	verbose := os.Getenv("ALPINE_HOOK_VERBOSE") == "true"
	return &Logger{
		runID:   runID,
		verbose: verbose,
	}
}

func (l *Logger) logJSON(level string, message string, data map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"component": "todo-monitor",
		"run_id":    l.runID,
		"message":   message,
	}

	// Add additional data
	for k, v := range data {
		logEntry[k] = v
	}

	jsonBytes, _ := json.Marshal(logEntry)
	fmt.Fprintln(os.Stderr, string(jsonBytes))
}

func (l *Logger) Info(message string, data map[string]interface{}) {
	l.logJSON("INFO", message, data)
}

func (l *Logger) Debug(message string, data map[string]interface{}) {
	if l.verbose {
		l.logJSON("DEBUG", message, data)
	}
}

func (l *Logger) Error(message string, data map[string]interface{}) {
	l.logJSON("ERROR", message, data)
}

func (l *Logger) Warn(message string, data map[string]interface{}) {
	l.logJSON("WARN", message, data)
}

func main() {
	startTime := time.Now()
	logger := newLogger()

	logger.Info("TODO monitor hook execution started", map[string]interface{}{
		"hook_type": "todo-monitor",
		"pid":       os.Getpid(),
	})

	// Read JSON input from Claude Code
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		logger.Error("Failed to read input from stdin", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	logger.Debug("Raw input received", map[string]interface{}{
		"input_size":    len(input),
		"input_preview": string(input)[:min(len(input), 200)] + "...",
	})

	var data HookData
	if err := json.Unmarshal(input, &data); err != nil {
		logger.Error("Failed to parse hook data JSON", map[string]interface{}{
			"error": err.Error(),
			"input": string(input),
		})
		return
	}

	logger.Info("Hook data parsed successfully", map[string]interface{}{
		"hook_event_name":  data.HookEventName,
		"tool_name":        data.ToolName,
		"tool":             data.Tool,
		"session_id":       data.SessionID,
		"transcript_path":  data.TranscriptPath,
		"stop_hook_active": data.StopHookActive,
		"has_tool_input":   len(data.ToolInput) > 0,
		"has_args":         len(data.Args) > 0,
	})

	// Handle subagent:stop events
	if data.HookEventName == "subagent:stop" {
		logger.Info("Subagent stop event received", map[string]interface{}{
			"session_id": data.SessionID,
		})
		fmt.Fprintln(os.Stderr, "🔄 Subagent completed")
		duration := time.Since(startTime)
		logger.Info("TODO monitor hook completed", map[string]interface{}{
			"duration":   duration.String(),
			"event_type": "subagent_stop",
		})
		return
	}

	// Process tool-specific events
	toolName := data.ToolName
	if toolName == "" {
		toolName = data.Tool // Fallback to legacy field
	}

	if toolName == "" {
		logger.Debug("No tool name found, skipping processing", nil)
		return
	}

	logger.Info("Processing tool event", map[string]interface{}{
		"tool_name":  toolName,
		"event_name": data.HookEventName,
	})

	// Handle TodoWrite tool specifically
	if toolName == "todowrite" {
		handleTodoWrite(data, logger)
	} else {
		// Handle other tools with general monitoring
		handleGeneralTool(data, logger)
	}

	duration := time.Since(startTime)
	logger.Info("TODO monitor hook completed", map[string]interface{}{
		"duration":  duration.String(),
		"tool_name": toolName,
	})
}

func handleTodoWrite(data HookData, logger *Logger) {
	logger.Debug("Processing TodoWrite tool", map[string]interface{}{
		"tool_input_size": len(data.ToolInput),
		"args_size":       len(data.Args),
	})

	// Parse tool input to extract todos
	var input TodoWriteInput
	var inputData json.RawMessage

	// Try tool_input first, then args (for backward compatibility)
	if len(data.ToolInput) > 0 {
		inputData = data.ToolInput
	} else if len(data.Args) > 0 {
		inputData = data.Args
	} else {
		logger.Debug("No input data found for TodoWrite", nil)
		return
	}

	if err := json.Unmarshal(inputData, &input); err != nil {
		logger.Error("Failed to parse TodoWrite input", map[string]interface{}{
			"error": err.Error(),
			"input": string(inputData),
		})
		return
	}

	logger.Info("TodoWrite input parsed", map[string]interface{}{
		"todo_count": len(input.Todos),
	})

	// Display current task progress
	if len(input.Todos) > 0 {
		displayTaskProgress(input.Todos, logger)

		// Write current task to file if configured
		if taskFile := os.Getenv("ALPINE_TODO_FILE"); taskFile != "" {
			writeCurrentTaskToFile(input.Todos, taskFile, logger)
		}
	}
}

func handleGeneralTool(data HookData, logger *Logger) {
	logger.Debug("Processing general tool", map[string]interface{}{
		"tool_name": data.ToolName,
		"tool":      data.Tool,
	})

	// Parse tool input for context
	var toolInput ToolInput
	var inputData json.RawMessage

	if len(data.ToolInput) > 0 {
		inputData = data.ToolInput
	} else if len(data.Args) > 0 {
		inputData = data.Args
	}

	if len(inputData) > 0 {
		if err := json.Unmarshal(inputData, &toolInput); err == nil {
			logger.Debug("Tool input parsed", map[string]interface{}{
				"file_path":   toolInput.FilePath,
				"command":     toolInput.Command,
				"pattern":     toolInput.Pattern,
				"path":        toolInput.Path,
				"url":         toolInput.URL,
				"query":       toolInput.Query,
				"description": toolInput.Description,
			})

			// Display tool usage information
			displayToolUsage(data.ToolName, toolInput, logger)
		} else {
			logger.Debug("Failed to parse tool input as ToolInput struct", map[string]interface{}{
				"error":     err.Error(),
				"raw_input": sanitizeToolData(inputData),
			})
		}
	}
}

func displayTaskProgress(todos []TodoItem, logger *Logger) {
	inProgress := 0
	completed := 0
	pending := 0

	for _, todo := range todos {
		switch strings.ToLower(todo.Status) {
		case "in_progress":
			inProgress++
		case "completed":
			completed++
		case "pending":
			pending++
		}
	}

	logger.Info("Task progress summary", map[string]interface{}{
		"total_tasks":     len(todos),
		"in_progress":     inProgress,
		"completed":       completed,
		"pending":         pending,
		"completion_rate": float64(completed) / float64(len(todos)) * 100,
	})

	// Display current task to console
	for _, todo := range todos {
		if strings.ToLower(todo.Status) == "in_progress" {
			fmt.Fprintf(os.Stderr, "🔄 Current Task: %s\n", todo.Content)
			logger.Info("Current active task", map[string]interface{}{
				"task_content": todo.Content,
				"status":       todo.Status,
			})
			break
		}
	}

	// Show progress bar
	progressBar := createProgressBar(completed, len(todos))
	fmt.Fprintf(os.Stderr, "📊 Progress: %s (%d/%d)\n", progressBar, completed, len(todos))
}

func displayToolUsage(toolName string, input ToolInput, logger *Logger) {
	var displayText string

	switch toolName {
	case "read":
		if input.FilePath != "" {
			displayText = fmt.Sprintf("📖 Reading: %s", input.FilePath)
		}
	case "write":
		if input.FilePath != "" {
			displayText = fmt.Sprintf("✏️  Writing: %s", input.FilePath)
		}
	case "bash":
		if input.Command != "" {
			displayText = fmt.Sprintf("⚡ Running: %s", truncateString(input.Command, 50))
		}
	case "grep":
		if input.Pattern != "" && input.Path != "" {
			displayText = fmt.Sprintf("🔍 Searching: %s in %s", input.Pattern, input.Path)
		}
	case "glob":
		if input.Pattern != "" {
			displayText = fmt.Sprintf("📁 Finding: %s", input.Pattern)
		}
	case "webfetch":
		if input.URL != "" {
			displayText = fmt.Sprintf("🌐 Fetching: %s", input.URL)
		}
	default:
		displayText = fmt.Sprintf("🔧 Using: %s", toolName)
	}

	if displayText != "" {
		fmt.Fprintln(os.Stderr, displayText)
		logger.Info("Tool usage displayed", map[string]interface{}{
			"tool_name":    toolName,
			"display_text": displayText,
		})
	}
}

func writeCurrentTaskToFile(todos []TodoItem, filename string, logger *Logger) {
	for _, todo := range todos {
		if strings.ToLower(todo.Status) == "in_progress" {
			if err := os.WriteFile(filename, []byte(todo.Content), 0644); err != nil {
				logger.Error("Failed to write current task to file", map[string]interface{}{
					"error":    err.Error(),
					"filename": filename,
					"task":     todo.Content,
				})
			} else {
				logger.Debug("Current task written to file", map[string]interface{}{
					"filename": filename,
					"task":     todo.Content,
				})
			}
			break
		}
	}
}

func createProgressBar(completed, total int) string {
	if total == 0 {
		return "▓▓▓▓▓▓▓▓▓▓"
	}

	barLength := 10
	filledLength := int(float64(completed) / float64(total) * float64(barLength))

	bar := strings.Repeat("▓", filledLength) + strings.Repeat("░", barLength-filledLength)
	return bar
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// sanitizeToolData removes sensitive information from tool data for logging
func sanitizeToolData(data json.RawMessage) interface{} {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return string(data)[:min(len(data), 100)] + "..."
	}

	// If it's a map, sanitize sensitive fields
	if m, ok := parsed.(map[string]interface{}); ok {
		sanitized := make(map[string]interface{})
		for k, v := range m {
			key := strings.ToLower(k)
			if strings.Contains(key, "password") ||
				strings.Contains(key, "token") ||
				strings.Contains(key, "secret") ||
				strings.Contains(key, "key") {
				sanitized[k] = "[REDACTED]"
			} else {
				sanitized[k] = v
			}
		}
		return sanitized
	}

	return parsed
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
