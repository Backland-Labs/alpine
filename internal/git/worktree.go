package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CreateWorktree creates a new git worktree for the given issue ID
func CreateWorktree(baseDir, issueID string) (string, error) {
	// Sanitize the issue ID
	sanitizedID := SanitizeIssueID(issueID)

	// Create worktree directory name
	worktreeName := fmt.Sprintf("river-%s", sanitizedID)
	worktreePath := filepath.Join(baseDir, worktreeName)

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree already exists at %s", worktreePath)
	}

	// Create the worktree with a new branch
	branchName := sanitizedID
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create worktree: %w\nOutput: %s", err, string(output))
	}

	return worktreePath, nil
}

// RemoveWorktree removes the worktree and its associated branch
func RemoveWorktree(worktreePath string) error {
	// First, remove the worktree
	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w\nOutput: %s", err, string(output))
	}

	// Get the branch name from the worktree path
	// Assuming the path ends with river-<branch-name>
	baseName := filepath.Base(worktreePath)
	if strings.HasPrefix(baseName, "river-") {
		branchName := strings.TrimPrefix(baseName, "river-")

		// Delete the branch
		cmd = exec.Command("git", "branch", "-D", branchName)
		output, err = cmd.CombinedOutput()
		if err != nil {
			// Branch deletion failure is not critical, log but don't fail
			// The branch might not exist or might be checked out elsewhere
			return fmt.Errorf("worktree removed but failed to delete branch %s: %w\nOutput: %s", branchName, err, string(output))
		}
	}

	return nil
}

// SanitizeIssueID converts an issue ID to a valid git branch name
func SanitizeIssueID(issueID string) string {
	// Convert to lowercase
	sanitized := strings.ToLower(issueID)

	// Replace non-alphanumeric characters with dashes
	// This regex matches anything that's not a letter, number, or dash
	reg := regexp.MustCompile(`[^a-z0-9-]+`)
	sanitized = reg.ReplaceAllString(sanitized, "-")

	// Remove leading/trailing dashes
	sanitized = strings.Trim(sanitized, "-")

	// Replace multiple consecutive dashes with a single dash
	reg = regexp.MustCompile(`-+`)
	sanitized = reg.ReplaceAllString(sanitized, "-")

	return sanitized
}
