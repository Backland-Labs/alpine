package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/maxkrieger/river/internal/git"
	"github.com/maxkrieger/river/internal/runner"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: river <LINEAR-ISSUE-ID>\n")
	os.Exit(1)
}

func main() {
	// Parse command-line arguments
	flag.Parse()
	args := flag.Args()

	// Validate arguments
	if len(args) != 1 {
		usage()
	}

	issueID := args[0]
	if issueID == "" {
		fmt.Fprintf(os.Stderr, "Error: LINEAR-ISSUE-ID cannot be empty\n")
		usage()
	}

	// Sanitize issue ID for directory name
	sanitizedID := sanitizeIssueID(issueID)

	// Create worktree directory path
	worktreePath := fmt.Sprintf("../river-%s", sanitizedID)

	// Main workflow
	if err := runWorkflow(issueID, worktreePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Workflow completed successfully")
}

func runWorkflow(issueID, worktreePath string) error {
	fmt.Printf("Processing Linear issue: %s\n", issueID)
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

	// Run the auto_claude.sh script in the worktree
	scriptPath := filepath.Join(currentDir, "auto_claude.sh")
	if err := runner.RunAutoClaudeScript(worktreeFullPath, issueID, scriptPath); err != nil {
		return fmt.Errorf("failed to run auto_claude.sh: %w", err)
	}

	return nil
}

func sanitizeIssueID(issueID string) string {
	// Use the sanitization from the git package
	return git.SanitizeIssueID(issueID)
}