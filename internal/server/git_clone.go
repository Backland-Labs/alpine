package server

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Backland-Labs/alpine/internal/config"
	"github.com/Backland-Labs/alpine/internal/logger"
)

var (
	// ErrCloneTimeout indicates the git clone operation timed out
	ErrCloneTimeout = errors.New("git clone operation timed out")

	// ErrRepoNotFound indicates the repository was not found
	ErrRepoNotFound = errors.New("repository not found")

	// ErrCloneDisabled indicates git clone is disabled in configuration
	ErrCloneDisabled = errors.New("git clone is disabled")
)

// cloneRepository clones a git repository to a temporary directory.
// It supports authentication, timeout handling, and shallow clones.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - repoURL: The repository URL to clone
//   - config: Git clone configuration with timeout, depth, and auth settings
//
// Returns:
//   - string: Path to the cloned repository directory
//   - error: Any error that occurred during cloning
//
// The function performs the following operations:
//  1. Validates that cloning is enabled in configuration
//  2. Creates a timeout context based on config.Timeout
//  3. Creates a temporary directory for the clone
//  4. Builds an authenticated URL if auth token is provided
//  5. Executes git clone with specified depth
//  6. Handles various error conditions with appropriate error types
//  7. Logs all operations for debugging and monitoring
func cloneRepository(ctx context.Context, repoURL string, config *config.GitCloneConfig) (string, error) {
	sanitizedURL := sanitizeURLForLogging(repoURL)

	log := logger.WithFields(map[string]interface{}{
		"repository_url": sanitizedURL,
		"clone_depth":    config.Depth,
		"timeout":        config.Timeout,
		"auth_enabled":   config.AuthToken != "",
	})

	if !config.Enabled {
		log.Debug("Git clone operation disabled by configuration")
		return "", fmt.Errorf("clone operation disabled: %w", ErrCloneDisabled)
	}

	log.Info("Starting git clone operation")
	start := time.Now()

	// Create timeout context
	cloneCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Create temporary directory for the clone
	tempDir, err := os.MkdirTemp("", "alpine-clone-*")
	if err != nil {
		log.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to create temporary directory for git clone")
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"temp_directory": tempDir,
	}).Debug("Created temporary directory for clone")

	// Build authenticated URL if token is provided
	cloneURL := repoURL
	if config.AuthToken != "" {
		cloneURL = buildAuthenticatedURL(repoURL, config.AuthToken)
		log.Debug("Built authenticated URL for private repository clone")
	}

	// Build git clone command
	args := []string{"clone", "--depth", fmt.Sprintf("%d", config.Depth), cloneURL, tempDir}
	cmd := exec.CommandContext(cloneCtx, "git", args...)

	log.WithFields(map[string]interface{}{
		"git_args": []string{"clone", "--depth", fmt.Sprintf("%d", config.Depth), sanitizedURL, tempDir},
	}).Debug("Executing git clone command")

	// Run the clone command
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		// Clean up temp directory on failure
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			log.WithFields(map[string]interface{}{
				"temp_directory": tempDir,
				"remove_error":   removeErr.Error(),
			}).Warn("Failed to clean up temporary directory after clone failure")
		}

		outputStr := string(output)

		// Check for specific error types and log accordingly
		if cloneCtx.Err() == context.DeadlineExceeded {
			log.WithFields(map[string]interface{}{
				"duration": duration,
				"timeout":  config.Timeout,
			}).Error("Git clone operation timed out")
			return "", fmt.Errorf("git clone timed out after %v: %w", config.Timeout, ErrCloneTimeout)
		}

		// Check if it's a repository not found error
		if strings.Contains(outputStr, "repository not found") ||
			strings.Contains(outputStr, "not found") ||
			strings.Contains(outputStr, "404") {
			log.WithFields(map[string]interface{}{
				"duration": duration,
				"output":   outputStr,
			}).Error("Repository not found during clone")
			return "", fmt.Errorf("repository not found: %w", ErrRepoNotFound)
		}

		// Generic clone error
		log.WithFields(map[string]interface{}{
			"duration": duration,
			"error":    err.Error(),
			"output":   outputStr,
		}).Error("Git clone failed")
		return "", fmt.Errorf("git clone failed: %w", err)
	}

	log.WithFields(map[string]interface{}{
		"clone_directory": tempDir,
		"duration":        duration,
	}).Info("Git clone completed successfully")

	return tempDir, nil
}

// sanitizeURLForLogging removes authentication information from URLs for safe logging.
// This prevents sensitive authentication tokens from being logged in plaintext.
//
// Parameters:
//   - repoURL: The repository URL that may contain authentication information
//
// Returns:
//   - string: The sanitized URL with authentication info replaced by "***"
//
// Examples:
//   - "https://token@github.com/owner/repo.git" -> "https://***@github.com/owner/repo.git"
//   - "https://github.com/owner/repo.git" -> "https://github.com/owner/repo.git"
//   - "" -> ""
func sanitizeURLForLogging(repoURL string) string {
	if repoURL == "" {
		return ""
	}

	// Use regex to replace any auth info with ***
	// Pattern matches: ://[anything]@ and replaces with ://***@
	re := regexp.MustCompile(`://[^@]+@`)
	return re.ReplaceAllString(repoURL, "://***@")
}

// buildAuthenticatedURL adds authentication token to a git URL.
// This is used to inject authentication credentials into the clone URL.
//
// Parameters:
//   - repoURL: The base repository URL
//   - authToken: The authentication token to inject
//
// Returns:
//   - string: The URL with authentication token embedded, or original URL if token is empty/parsing fails
//
// Examples:
//   - buildAuthenticatedURL("https://github.com/owner/repo.git", "token") -> "https://token@github.com/owner/repo.git"
//   - buildAuthenticatedURL("https://github.com/owner/repo.git", "") -> "https://github.com/owner/repo.git"
//
// Note: If URL parsing fails, the original URL is returned to prevent clone failures
// due to URL manipulation errors.
func buildAuthenticatedURL(repoURL, authToken string) string {
	if authToken == "" {
		return repoURL
	}

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		// Log warning but return original URL to allow clone to proceed
		logger.WithFields(map[string]interface{}{
			"repository_url": sanitizeURLForLogging(repoURL),
			"error":          err.Error(),
		}).Warn("Failed to parse repository URL for authentication, using original URL")
		return repoURL
	}

	// Set the auth token as the user info
	parsedURL.User = url.User(authToken)

	return parsedURL.String()
}
