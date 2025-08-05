package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Backland-Labs/alpine/internal/config"
)

// Test helper to create a basic GitCloneConfig
func createTestConfig() *config.GitCloneConfig {
	return &config.GitCloneConfig{
		Enabled:   true,
		AuthToken: "",
		Timeout:   30 * time.Second,
		Depth:     1,
	}
}

func TestCloneRepository(t *testing.T) {
	tests := []struct {
		name        string
		repoURL     string
		config      *config.GitCloneConfig
		expectError bool
		errorType   error
		setupFunc   func(t *testing.T) context.Context
	}{
		{
			name:        "successful clone of public repository",
			repoURL:     "https://github.com/octocat/Hello-World.git",
			config:      createTestConfig(),
			expectError: false,
			setupFunc: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				t.Cleanup(cancel)
				return ctx
			},
		},
		{
			name:    "clone with authentication token",
			repoURL: "https://github.com/octocat/Hello-World.git",
			config: &config.GitCloneConfig{
				Enabled:   true,
				AuthToken: "fake_token_for_test",
				Timeout:   30 * time.Second,
				Depth:     1,
			},
			expectError: false,
			setupFunc: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				t.Cleanup(cancel)
				return ctx
			},
		},
		{
			name:    "timeout error when timeout is very short",
			repoURL: "https://github.com/octocat/Hello-World.git",
			config: &config.GitCloneConfig{
				Enabled:   true,
				AuthToken: "",
				Timeout:   1 * time.Nanosecond, // Extremely short timeout
				Depth:     1,
			},
			expectError: true,
			errorType:   ErrCloneTimeout,
			setupFunc: func(t *testing.T) context.Context {
				return context.Background()
			},
		},
		{
			name:        "repository not found error",
			repoURL:     "https://github.com/nonexistent/nonexistent-repo.git",
			config:      createTestConfig(),
			expectError: true,
			errorType:   ErrRepoNotFound,
			setupFunc: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				t.Cleanup(cancel)
				return ctx
			},
		},
		{
			name:        "invalid repository URL",
			repoURL:     "not-a-valid-url",
			config:      createTestConfig(),
			expectError: true,
			setupFunc: func(t *testing.T) context.Context {
				return context.Background()
			},
		},
		{
			name:    "clone disabled in configuration",
			repoURL: "https://github.com/octocat/Hello-World.git",
			config: &config.GitCloneConfig{
				Enabled:   false,
				AuthToken: "",
				Timeout:   30 * time.Second,
				Depth:     1,
			},
			expectError: true,
			setupFunc: func(t *testing.T) context.Context {
				return context.Background()
			},
		},
		{
			name:        "context cancellation",
			repoURL:     "https://github.com/octocat/Hello-World.git",
			config:      createTestConfig(),
			expectError: true,
			setupFunc: func(t *testing.T) context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			},
		},
		{
			name:    "custom clone depth",
			repoURL: "https://github.com/octocat/Hello-World.git",
			config: &config.GitCloneConfig{
				Enabled:   true,
				AuthToken: "",
				Timeout:   30 * time.Second,
				Depth:     5, // Custom depth
			},
			expectError: false,
			setupFunc: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				t.Cleanup(cancel)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupFunc(t)

			result, err := cloneRepository(ctx, tt.repoURL, tt.config)

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none")
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType, "Expected specific error type")
				}
				assert.Empty(t, result, "Expected empty result on error")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
				assert.NotEmpty(t, result, "Expected non-empty result directory")

				// Verify directory exists and contains .git
				_, err := os.Stat(result)
				assert.NoError(t, err, "Clone directory should exist")

				gitDir := filepath.Join(result, ".git")
				_, err = os.Stat(gitDir)
				assert.NoError(t, err, "Clone directory should contain .git directory")

				// Cleanup the cloned directory
				t.Cleanup(func() {
					os.RemoveAll(result)
				})
			}
		})
	}
}

func TestSanitizeURLForLogging(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with token",
			input:    "https://ghp_xxxxxxxxxxxxxxxxxxxx@github.com/owner/repo.git",
			expected: "https://***@github.com/owner/repo.git",
		},
		{
			name:     "URL without token",
			input:    "https://github.com/owner/repo.git",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "URL with different auth format",
			input:    "https://user:pass@github.com/owner/repo.git",
			expected: "https://***@github.com/owner/repo.git",
		},
		{
			name:     "empty URL",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeURLForLogging(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildAuthenticatedURL(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		authToken string
		expected  string
	}{
		{
			name:      "add token to public URL",
			repoURL:   "https://github.com/owner/repo.git",
			authToken: "ghp_xxxxxxxxxxxxxxxxxxxx",
			expected:  "https://ghp_xxxxxxxxxxxxxxxxxxxx@github.com/owner/repo.git",
		},
		{
			name:      "empty token returns original URL",
			repoURL:   "https://github.com/owner/repo.git",
			authToken: "",
			expected:  "https://github.com/owner/repo.git",
		},
		{
			name:      "URL already has auth gets replaced",
			repoURL:   "https://existing@github.com/owner/repo.git",
			authToken: "new_token",
			expected:  "https://new_token@github.com/owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAuthenticatedURL(tt.repoURL, tt.authToken)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration test that verifies the clone operation creates a proper git repository
func TestCloneRepositoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	config := createTestConfig()

	// Use a small, reliable public repository for testing
	repoURL := "https://github.com/octocat/Hello-World.git"

	clonedDir, err := cloneRepository(ctx, repoURL, config)
	require.NoError(t, err, "Clone operation should succeed")
	require.NotEmpty(t, clonedDir, "Should return non-empty directory path")

	// Cleanup
	defer os.RemoveAll(clonedDir)

	// Verify it's a proper git repository
	gitDir := filepath.Join(clonedDir, ".git")
	stat, err := os.Stat(gitDir)
	require.NoError(t, err, ".git directory should exist")
	require.True(t, stat.IsDir(), ".git should be a directory")

	// Verify some expected files exist (this is a known repository)
	readmeFile := filepath.Join(clonedDir, "README")
	_, err = os.Stat(readmeFile)
	assert.NoError(t, err, "README file should exist in Hello-World repository")
}
