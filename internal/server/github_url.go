package server

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// gitHubIssueURLPattern matches GitHub issue URLs in the format:
	// https://github.com/owner/repo/issues/123
	gitHubIssueURLPattern = `^https://github\.com/([^/]+)/([^/]+)/issues/(\d+)$`

	// gitCloneURLTemplate is the template for building git clone URLs
	gitCloneURLTemplate = "https://github.com/%s/%s.git"
)

var (
	// ErrInvalidGitHubURL indicates the provided URL is not a valid GitHub issue URL
	ErrInvalidGitHubURL = errors.New("invalid GitHub issue URL")

	// gitHubIssueRegex is a compiled regex for parsing GitHub issue URLs
	gitHubIssueRegex = regexp.MustCompile(gitHubIssueURLPattern)
)

// parseGitHubIssueURL parses a GitHub issue URL and extracts owner, repository name, and issue number.
//
// The function expects URLs in the format: https://github.com/owner/repo/issues/123
// and returns the extracted components along with any parsing errors.
//
// Parameters:
//   - issueURL: The GitHub issue URL to parse
//
// Returns:
//   - owner: The repository owner/organization name
//   - repo: The repository name
//   - issueNum: The issue number as an integer
//   - err: Any error encountered during parsing
//
// Example:
//
//	owner, repo, issue, err := parseGitHubIssueURL("https://github.com/owner/repo/issues/123")
//	// Returns: "owner", "repo", 123, nil
func parseGitHubIssueURL(issueURL string) (owner, repo string, issueNum int, err error) {
	// Validate input
	if strings.TrimSpace(issueURL) == "" {
		return "", "", 0, fmt.Errorf("empty or whitespace-only URL: %w", ErrInvalidGitHubURL)
	}

	// Parse URL using pre-compiled regex
	matches := gitHubIssueRegex.FindStringSubmatch(issueURL)
	if len(matches) != 4 {
		return "", "", 0, fmt.Errorf("URL does not match GitHub issue format 'https://github.com/owner/repo/issues/NUMBER': %w", ErrInvalidGitHubURL)
	}

	// Extract components
	owner = strings.TrimSpace(matches[1])
	repo = strings.TrimSpace(matches[2])
	issueNumberStr := strings.TrimSpace(matches[3])

	// Additional validation for empty components after regex match
	if owner == "" {
		return "", "", 0, fmt.Errorf("repository owner cannot be empty: %w", ErrInvalidGitHubURL)
	}
	if repo == "" {
		return "", "", 0, fmt.Errorf("repository name cannot be empty: %w", ErrInvalidGitHubURL)
	}

	// Parse issue number
	issueNumber, err := strconv.Atoi(issueNumberStr)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid issue number '%s': %w", issueNumberStr, err)
	}

	// Validate issue number is positive
	if issueNumber <= 0 {
		return "", "", 0, fmt.Errorf("issue number must be positive, got %d: %w", issueNumber, ErrInvalidGitHubURL)
	}

	return owner, repo, issueNumber, nil
}

// buildGitCloneURL constructs a git clone URL from repository owner and name.
//
// Parameters:
//   - owner: The repository owner/organization name
//   - repo: The repository name
//
// Returns:
//   - The complete git clone URL in HTTPS format
//
// Example:
//
//	url := buildGitCloneURL("owner", "repo")
//	// Returns: "https://github.com/owner/repo.git"
func buildGitCloneURL(owner, repo string) string {
	return fmt.Sprintf(gitCloneURLTemplate, owner, repo)
}

// isGitHubIssueURL checks whether the provided URL is a valid GitHub issue URL.
//
// This function uses parseGitHubIssueURL internally and returns true if the URL
// can be successfully parsed as a GitHub issue URL.
//
// Parameters:
//   - url: The URL to validate
//
// Returns:
//   - true if the URL is a valid GitHub issue URL, false otherwise
//
// Example:
//
//	valid := isGitHubIssueURL("https://github.com/owner/repo/issues/123")
//	// Returns: true
func isGitHubIssueURL(url string) bool {
	_, _, _, err := parseGitHubIssueURL(url)
	return err == nil
}
