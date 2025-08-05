package config

import (
	"os"
	"testing"
	"time"
)

// TestGitCloneConfigDefaults tests that Git Clone configuration has correct default values
func TestGitCloneConfigDefaults(t *testing.T) {
	// Clear all Git clone-related environment variables to test defaults
	envVars := []string{
		"ALPINE_GIT_CLONE_ENABLED",
		"ALPINE_GIT_CLONE_AUTH_TOKEN",
		"ALPINE_GIT_CLONE_TIMEOUT",
		"ALPINE_GIT_CLONE_DEPTH",
	}
	for _, env := range envVars {
		_ = os.Unsetenv(env)
	}

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Test Git Clone defaults
	if !cfg.Git.Clone.Enabled {
		t.Error("Git.Clone.Enabled = false, want true (default)")
	}

	if cfg.Git.Clone.AuthToken != "" {
		t.Errorf("Git.Clone.AuthToken = %q, want empty string (default)", cfg.Git.Clone.AuthToken)
	}

	expectedTimeout := 300 * time.Second
	if cfg.Git.Clone.Timeout != expectedTimeout {
		t.Errorf("Git.Clone.Timeout = %v, want %v (default)", cfg.Git.Clone.Timeout, expectedTimeout)
	}

	if cfg.Git.Clone.Depth != 1 {
		t.Errorf("Git.Clone.Depth = %d, want 1 (default)", cfg.Git.Clone.Depth)
	}
}

// TestGitCloneConfigEnvironmentVariables tests loading Git Clone configuration from environment
func TestGitCloneConfigEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		want     GitCloneConfig
		wantErr  bool
		errMsg   string
	}{
		{
			name: "all values set",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_ENABLED":    "false",
				"ALPINE_GIT_CLONE_AUTH_TOKEN": "ghp_test123",
				"ALPINE_GIT_CLONE_TIMEOUT":    "600",
				"ALPINE_GIT_CLONE_DEPTH":      "5",
			},
			want: GitCloneConfig{
				Enabled:   false,
				AuthToken: "ghp_test123",
				Timeout:   600 * time.Second,
				Depth:     5,
			},
		},
		{
			name: "enabled true with custom timeout",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_ENABLED": "true",
				"ALPINE_GIT_CLONE_TIMEOUT": "120",
			},
			want: GitCloneConfig{
				Enabled:   true,
				AuthToken: "", // default
				Timeout:   120 * time.Second,
				Depth:     1, // default
			},
		},
		{
			name: "invalid boolean for enabled",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_ENABLED": "yes",
			},
			wantErr: true,
			errMsg:  "ALPINE_GIT_CLONE_ENABLED must be true or false",
		},
		{
			name: "invalid timeout (negative)",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_TIMEOUT": "-60",
			},
			wantErr: true,
			errMsg:  "ALPINE_GIT_CLONE_TIMEOUT must be positive",
		},
		{
			name: "invalid timeout (zero)",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_TIMEOUT": "0",
			},
			wantErr: true,
			errMsg:  "ALPINE_GIT_CLONE_TIMEOUT must be positive",
		},
		{
			name: "invalid timeout (not a number)",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_TIMEOUT": "not-a-number",
			},
			wantErr: true,
			errMsg:  "invalid ALPINE_GIT_CLONE_TIMEOUT",
		},
		{
			name: "invalid depth (negative)",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_DEPTH": "-1",
			},
			wantErr: true,
			errMsg:  "ALPINE_GIT_CLONE_DEPTH must be positive",
		},
		{
			name: "invalid depth (zero)",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_DEPTH": "0",
			},
			wantErr: true,
			errMsg:  "ALPINE_GIT_CLONE_DEPTH must be positive",
		},
		{
			name: "invalid depth (not a number)",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_DEPTH": "not-a-number",
			},
			wantErr: true,
			errMsg:  "invalid ALPINE_GIT_CLONE_DEPTH",
		},
		{
			name: "empty values use defaults",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_ENABLED":    "",
				"ALPINE_GIT_CLONE_AUTH_TOKEN": "",
				"ALPINE_GIT_CLONE_TIMEOUT":    "",
				"ALPINE_GIT_CLONE_DEPTH":      "",
			},
			want: GitCloneConfig{
				Enabled:   true,                // default
				AuthToken: "",                  // default (empty)
				Timeout:   300 * time.Second,   // default
				Depth:     1,                   // default
			},
		},
		{
			name: "auth token with whitespace preserved",
			envVars: map[string]string{
				"ALPINE_GIT_CLONE_AUTH_TOKEN": "  token_with_spaces  ",
			},
			want: GitCloneConfig{
				Enabled:   true,                // default
				AuthToken: "  token_with_spaces  ", // whitespace preserved
				Timeout:   300 * time.Second,   // default
				Depth:     1,                   // default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				if v == "" {
					_ = os.Unsetenv(k)
				} else {
					_ = os.Setenv(k, v)
				}
			}

			// Clean up after test
			defer func() {
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				if tt.errMsg != "" && !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
				}
				return
			}

			// Compare expected values
			if cfg.Git.Clone.Enabled != tt.want.Enabled {
				t.Errorf("Git.Clone.Enabled = %v, want %v", cfg.Git.Clone.Enabled, tt.want.Enabled)
			}
			if cfg.Git.Clone.AuthToken != tt.want.AuthToken {
				t.Errorf("Git.Clone.AuthToken = %q, want %q", cfg.Git.Clone.AuthToken, tt.want.AuthToken)
			}
			if cfg.Git.Clone.Timeout != tt.want.Timeout {
				t.Errorf("Git.Clone.Timeout = %v, want %v", cfg.Git.Clone.Timeout, tt.want.Timeout)
			}
			if cfg.Git.Clone.Depth != tt.want.Depth {
				t.Errorf("Git.Clone.Depth = %d, want %d", cfg.Git.Clone.Depth, tt.want.Depth)
			}
		})
	}
}

// TestGitCloneConfigLargeTimeout tests handling of large timeout values
func TestGitCloneConfigLargeTimeout(t *testing.T) {
	tests := []struct {
		name       string
		timeout    string
		wantSeconds int
		wantErr    bool
	}{
		{
			name:       "max int32 seconds",
			timeout:    "2147483647", // max int32
			wantSeconds: 2147483647,
			wantErr:    false,
		},
		{
			name:    "very large timeout (overflow)",
			timeout: "999999999999999999999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("ALPINE_GIT_CLONE_TIMEOUT", tt.timeout)
			defer func() {
				_ = os.Unsetenv("ALPINE_GIT_CLONE_TIMEOUT")
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				expectedTimeout := time.Duration(tt.wantSeconds) * time.Second
				if cfg.Git.Clone.Timeout != expectedTimeout {
					t.Errorf("Git.Clone.Timeout = %v, want %v", cfg.Git.Clone.Timeout, expectedTimeout)
				}
			}
		})
	}
}

// containsSubstring is a helper function to check if a message contains a substring
func containsSubstring(message, substring string) bool {
	return len(message) >= len(substring) && 
		   message != substring && 
		   (message[:len(substring)] == substring || 
			message[len(message)-len(substring):] == substring ||
			findSubstring(message, substring))
}

// findSubstring checks if substring exists anywhere in the message
func findSubstring(message, substring string) bool {
	if len(substring) > len(message) {
		return false
	}
	for i := 0; i <= len(message)-len(substring); i++ {
		if message[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}