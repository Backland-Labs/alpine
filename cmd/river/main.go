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
	IssueID string
	Stream  bool
}

// parseArguments parses and validates command-line arguments
func parseArguments() (*Config, error) {
	var stream bool
	
	// Define the --stream flag
	flag.BoolVar(&stream, "stream", false, "Enable JSON streaming output")
	
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
	
	return &Config{
		IssueID: issueID,
		Stream:  stream,
	}, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: river [OPTIONS] <LINEAR-ISSUE-ID>\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	// Parse and validate command-line arguments
	config, err := parseArguments()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		usage()
	}

	// Sanitize issue ID for directory name
	sanitizedID := sanitizeIssueID(config.IssueID)

	// Create worktree directory path
	worktreePath := fmt.Sprintf("../river-%s", sanitizedID)

	// Main workflow
	if err := runWorkflow(config.IssueID, worktreePath, config.Stream); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Workflow completed successfully")
}

func runWorkflow(issueID, worktreePath string, stream bool) error {
	fmt.Printf("Processing Linear issue: %s\n", issueID)
	if stream {
		fmt.Println("Streaming mode enabled")
	}
	fmt.Printf("Creating worktree at: %s\n", worktreePath)

	// Get absolute path for the parent directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	parentDir := filepath.Dir(currentDir)

	// Create git worktree
	worktreeFullPath, err := git.CreateWorktree(parentDir, issueID)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	fmt.Printf("Created worktree at: %s\n", worktreeFullPath)

	// Create Claude executor
	ctx := context.Background()
	executor := claude.New()

	// Execute the workflow using Claude
	if err := executeClaudeWorkflow(ctx, executor, issueID, worktreeFullPath, stream); err != nil {
		return fmt.Errorf("failed to execute Claude workflow: %w", err)
	}

	return nil
}

// executeClaudeWorkflow runs the main Claude workflow with plan/continue loop
func executeClaudeWorkflow(ctx context.Context, executor claude.Claude, issueID, workingDir string, stream bool) error {
	// Initial plan command
	fmt.Println("Creating initial implementation plan...")
	
	planCmd := claude.Command{
		Type:         claude.CommandTypePlan,
		Content:      fmt.Sprintf("Process Linear issue %s following TDD methodology", issueID),
		OutputFormat: "json",
		AllowedTools: []string{"linear-server", "code-editing"},
		SystemPrompt: "You are an AI assistant helping to implement features from Linear issues using Test-Driven Development. Follow the red-green-refactor cycle strictly.",
	}

	opts := claude.CommandOptions{
		Stream:     stream,
		WorkingDir: workingDir,
	}

	// Execute initial plan
	response, err := executor.Execute(ctx, planCmd, opts)
	if err != nil {
		return fmt.Errorf("failed to execute initial plan: %w", err)
	}

	// Continue loop
	sessionID := "" // Will be set after first response if needed
	iteration := 1
	
	for response.ContinueFlag {
		fmt.Printf("Continuing workflow (iteration %d)...\n", iteration)
		
		continueCmd := claude.Command{
			Type:         claude.CommandTypeContinue,
			SessionID:    sessionID,
			OutputFormat: "json",
		}

		response, err = executor.Execute(ctx, continueCmd, opts)
		if err != nil {
			return fmt.Errorf("failed to execute continue command (iteration %d): %w", iteration, err)
		}

		iteration++
		
		// Safety check to prevent infinite loops
		if iteration > 50 {
			return fmt.Errorf("workflow exceeded maximum iterations (50)")
		}
	}

	fmt.Printf("Workflow completed after %d iteration(s)\n", iteration)
	return nil
}

func sanitizeIssueID(issueID string) string {
	// Use the sanitization from the git package
	return git.SanitizeIssueID(issueID)
}