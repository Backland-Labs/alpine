// Package github provides utilities for interacting with GitHub issues
package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// GitHubIssue represents the structure of a GitHub issue response
type GitHubIssue struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// IsGitHubIssueURL checks if a URL is a valid GitHub issue URL
func IsGitHubIssueURL(url string) bool {
	pattern := `^https://github\.com/[^/]+/[^/]+/issues/\d+$`
	matched, _ := regexp.MatchString(pattern, url)
	return matched
}

// FetchIssueDescription fetches issue data from GitHub using the gh CLI
// Returns formatted task description with title and body
func FetchIssueDescription(url string) (string, error) {
	if !IsGitHubIssueURL(url) {
		return "", fmt.Errorf("invalid GitHub issue URL: %s", url)
	}

	// Execute gh command
	cmd := exec.Command("gh", "issue", "view", url, "--json", "title,body")

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		// Check if gh is not found
		if strings.Contains(err.Error(), "executable file not found") {
			return "", fmt.Errorf("gh CLI not found. Please install from https://cli.github.com")
		}

		// Check for exit error with stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			return "", fmt.Errorf("gh command failed: %s", stderr)
		}

		return "", fmt.Errorf("gh command execution failed: %w", err)
	}

	// Parse JSON response
	var issue GitHubIssue
	if err := json.Unmarshal(output, &issue); err != nil {
		return "", fmt.Errorf("failed to parse gh response: %w", err)
	}

	// Validate required fields
	if issue.Title == "" {
		return "", fmt.Errorf("empty issue title received from GitHub")
	}

	// Format task description (same format as used in plan.go)
	taskDesc := fmt.Sprintf("Task: %s\n\n%s", issue.Title, issue.Body)
	
	return taskDesc, nil
}