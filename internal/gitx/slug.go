package gitx

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// nonAlphanumericRegex matches any character that is not alphanumeric or hyphen
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9-]+`)

	// multipleHyphensRegex matches multiple consecutive hyphens
	multipleHyphensRegex = regexp.MustCompile(`-+`)
)

// sanitizeTaskName converts a task name into a valid git branch component.
// It replaces spaces and special characters with hyphens, converts to lowercase,
// and ensures the result is a valid git branch name component.
func sanitizeTaskName(taskName string) string {
	// Convert to lowercase
	slug := strings.ToLower(taskName)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove any non-alphanumeric characters (except hyphens)
	slug = nonAlphanumericRegex.ReplaceAllString(slug, "-")

	// Replace multiple consecutive hyphens with single hyphen
	slug = multipleHyphensRegex.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// If empty after sanitization, use a default
	if slug == "" {
		slug = "task"
	}

	// Ensure it doesn't exceed git's branch name length limits
	// Git allows very long branch names, but let's be reasonable
	const maxLength = 50
	if len(slug) > maxLength {
		slug = slug[:maxLength]
		// Trim any trailing hyphen from truncation
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// generateUniqueBranchName generates a unique branch name by appending a counter
// if the base branch name already exists.
func generateUniqueBranchName(baseBranch string, existingBranches []string) string {
	// Check if base branch exists
	exists := false
	for _, branch := range existingBranches {
		if branch == baseBranch {
			exists = true
			break
		}
	}

	if !exists {
		return baseBranch
	}

	// Try with incrementing counter
	for i := 2; i <= 100; i++ {
		candidate := fmt.Sprintf("%s-%d", baseBranch, i)
		exists := false
		for _, branch := range existingBranches {
			if branch == candidate {
				exists = true
				break
			}
		}
		if !exists {
			return candidate
		}
	}

	// Fallback: use timestamp if we somehow have 100 branches
	return fmt.Sprintf("%s-%d", baseBranch, int(timeNow().Unix()))
}

// timeNow is a variable to allow mocking in tests
var timeNow = func() time.Time {
	return time.Now()
}
