package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maxkrieger/river/internal/claude"
	"github.com/maxkrieger/river/internal/git"
)

// Config holds the parsed command-line configuration
type Config struct {
	IssueID      string
	Stream       bool
	NoPlan       bool
	OutputFormat string
}

// parseArguments parses and validates command-line arguments
func parseArguments() (*Config, error) {
	var stream bool
	var noPlan bool
	var outputFormat string

	// Define flags
	flag.BoolVar(&stream, "stream", false, "Enable JSON streaming output")
	flag.BoolVar(&noPlan, "no-plan", false, "Skip initial plan generation")
	flag.StringVar(&outputFormat, "output-format", "text", "Output format: json or text")

	// Parse flags
	flag.Parse()

	// Get remaining arguments
	args := flag.Args()

	// Validate that we have exactly one argument (the issue ID)
	if len(args) != 1 {
		return nil, errors.New("missing required LINEAR-ISSUE-ID argument")
	}

	issueID := args[0]
	if issueID == "" {
		return nil, errors.New("LINEAR-ISSUE-ID cannot be empty")
	}

	// Validate output format
	if outputFormat != "json" && outputFormat != "text" {
		return nil, errors.New("output-format must be 'json' or 'text'")
	}

	return &Config{
		IssueID:      issueID,
		Stream:       stream,
		NoPlan:       noPlan,
		OutputFormat: outputFormat,
	}, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: river [OPTIONS] <LINEAR-ISSUE-ID>\n\n")
	fmt.Fprintf(os.Stderr, "River automates software development workflows by integrating Linear with Claude Code.\n")
	fmt.Fprintf(os.Stderr, "By default, River generates an implementation plan and then continues until completion.\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	os.Exit(1)
}

const (
	// maxWorkflowIterations prevents infinite loops in the workflow (matching Python script)
	maxWorkflowIterations = 20

	// worktreePathFormat defines the format for worktree directory names
	worktreePathFormat = "../river-%s"
)

func main() {
	// Validate environment before anything else
	if err := validateEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nPlease ensure:\n")
		fmt.Fprintf(os.Stderr, "- claude CLI is installed and available in PATH\n")
		os.Exit(1)
	}

	// Parse and validate command-line arguments
	config, err := parseArguments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		usage()
	}

	// Main workflow
	if err := runWorkflow(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Workflow completed successfully")
}

func runWorkflow(config *Config) error {
	fmt.Printf("Processing Linear issue: %s\n", config.IssueID)
	if config.Stream {
		fmt.Println("Streaming mode enabled")
	}
	fmt.Printf("Output format: %s\n", config.OutputFormat)
	if config.NoPlan {
		fmt.Println("Skipping plan generation")
	}
	
	// Get absolute path for the parent directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	parentDir := filepath.Dir(currentDir)

	// Create git worktree
	worktreeFullPath, err := git.CreateWorktree(parentDir, config.IssueID)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	fmt.Printf("Created worktree at: %s\n", worktreeFullPath)

	// Create Claude executor
	ctx := context.Background()
	executor := claude.New()

	// Execute the workflow using Claude
	if err := executeClaudeWorkflow(ctx, executor, config, worktreeFullPath); err != nil {
		return fmt.Errorf("failed to execute Claude workflow: %w", err)
	}

	return nil
}

// executeClaudeWorkflow runs the main Claude workflow with plan/continue loop
func executeClaudeWorkflow(ctx context.Context, executor claude.Claude, config *Config, workingDir string) error {
	opts := claude.CommandOptions{
		Stream:     config.Stream,
		WorkingDir: workingDir,
	}

	// Initial plan command (unless --no-plan is specified)
	if !config.NoPlan {
		fmt.Println("Creating initial implementation plan...")

		planCmd := claude.Command{
			Type:         claude.CommandTypePlan,
			Content:      fmt.Sprintf("Process Linear issue %s following TDD methodology", config.IssueID),
			OutputFormat: config.OutputFormat,
			AllowedTools: []string{"linear-server", "code-editing"},
			SystemPrompt: "You are an AI assistant helping to implement features from Linear issues using Test-Driven Development. Follow the red-green-refactor cycle strictly.",
		}

		// Execute initial plan
		_, err := executor.Execute(ctx, planCmd, opts)
		if err != nil {
			return fmt.Errorf("failed to execute initial plan: %w", err)
		}
	}

	// Continue loop using status file monitoring (matching Python script behavior)
	sessionID := "" // Will be set after first response if needed
	iteration := 0
	continueFlag := true

	// Ensure cleanup happens even if there's an error
	defer claude.CleanupStatusFile(workingDir)

	for continueFlag && iteration < maxWorkflowIterations {
		iteration++
		fmt.Printf("Starting command... (Iteration %d)\n", iteration)

		continueCmd := claude.Command{
			Type:         claude.CommandTypeContinue,
			Content:      "Continue implementation", // Basic prompt for continue command
			SessionID:    sessionID,
			OutputFormat: config.OutputFormat,
		}

		_, err := executor.Execute(ctx, continueCmd, opts)
		if err != nil {
			return fmt.Errorf("failed to execute continue command (iteration %d): %w", iteration, err)
		}

		// Check status file for continuation (matching Python script)
		continueFlag = claude.CheckStatusFile(workingDir)
		if !continueFlag {
			fmt.Println("Status file indicates completion. Stopping.")
		}
	}

	if iteration >= maxWorkflowIterations {
		fmt.Printf("Reached maximum iterations (%d). Stopping.\n", maxWorkflowIterations)
	}

	fmt.Printf("Workflow completed after %d iteration(s)\n", iteration)
	return nil
}

func sanitizeIssueID(issueID string) string {
	// Use the sanitization from the git package
	return git.SanitizeIssueID(issueID)
}
