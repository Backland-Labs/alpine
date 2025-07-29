package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestNewConfig tests the creation of a new Config instance with default values
func TestNewConfig(t *testing.T) {
	// Clear all environment variables to test defaults
	envVars := []string{
		"ALPINE_WORKDIR",
		"ALPINE_VERBOSITY",
		"ALPINE_SHOW_OUTPUT",
		"ALPINE_AUTO_CLEANUP",
		"ALPINE_GIT_ENABLED",
		"ALPINE_GIT_BASE_BRANCH",
		"ALPINE_GIT_AUTO_CLEANUP",
		"ALPINE_SHOW_TODO_UPDATES",
		"ALPINE_SHOW_TOOL_UPDATES",
		"ALPINE_HTTP_ENABLED",
		"ALPINE_HTTP_PORT",
	}
	for _, env := range envVars {
		_ = os.Unsetenv(env)
	}

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Test default values
	cwd, _ := os.Getwd()
	if cfg.WorkDir != cwd {
		t.Errorf("WorkDir = %q, want %q", cfg.WorkDir, cwd)
	}

	if cfg.Verbosity != VerbosityNormal {
		t.Errorf("Verbosity = %q, want %q", cfg.Verbosity, VerbosityNormal)
	}

	if !cfg.ShowOutput {
		t.Error("ShowOutput = false, want true")
	}

	expectedStateFile := filepath.Join("agent_state", "agent_state.json")
	if cfg.StateFile != expectedStateFile {
		t.Errorf("StateFile = %q, want %q", cfg.StateFile, expectedStateFile)
	}

	if !cfg.AutoCleanup {
		t.Error("AutoCleanup = false, want true")
	}

	// Test Git defaults
	if !cfg.Git.WorktreeEnabled {
		t.Error("Git.WorktreeEnabled = false, want true")
	}

	if cfg.Git.BaseBranch != "main" {
		t.Errorf("Git.BaseBranch = %q, want %q", cfg.Git.BaseBranch, "main")
	}

	if !cfg.Git.AutoCleanupWT {
		t.Error("Git.AutoCleanupWT = false, want true")
	}

	// Test ShowTodoUpdates default
	if !cfg.ShowTodoUpdates {
		t.Error("ShowTodoUpdates = false, want true")
	}

	// Test ShowToolUpdates default
	if !cfg.ShowToolUpdates {
		t.Error("ShowToolUpdates = false, want true")
	}
}

// TestConfigFromEnvironment tests loading configuration from environment variables
func TestConfigFromEnvironment(t *testing.T) {
	// Set up test environment
	testWorkDir := "/test/work/dir"
	_ = os.Setenv("ALPINE_WORKDIR", testWorkDir)
	_ = os.Setenv("ALPINE_VERBOSITY", "debug")
	_ = os.Setenv("ALPINE_SHOW_OUTPUT", "false")
	// ALPINE_STATE_FILE is no longer configurable
	_ = os.Setenv("ALPINE_AUTO_CLEANUP", "false")
	_ = os.Setenv("ALPINE_GIT_ENABLED", "false")
	_ = os.Setenv("ALPINE_GIT_BASE_BRANCH", "develop")
	_ = os.Setenv("ALPINE_GIT_AUTO_CLEANUP", "false")
	_ = os.Setenv("ALPINE_SHOW_TODO_UPDATES", "false")
	_ = os.Setenv("ALPINE_SHOW_TOOL_UPDATES", "false")
	_ = os.Setenv("ALPINE_HTTP_ENABLED", "true")
	_ = os.Setenv("ALPINE_HTTP_PORT", "9090")

	defer func() {
		// Clean up
		_ = os.Unsetenv("ALPINE_WORKDIR")
		_ = os.Unsetenv("ALPINE_VERBOSITY")
		_ = os.Unsetenv("ALPINE_SHOW_OUTPUT")
		// ALPINE_STATE_FILE is no longer used
		_ = os.Unsetenv("ALPINE_AUTO_CLEANUP")
		_ = os.Unsetenv("ALPINE_GIT_ENABLED")
		_ = os.Unsetenv("ALPINE_GIT_BASE_BRANCH")
		_ = os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")
		_ = os.Unsetenv("ALPINE_SHOW_TODO_UPDATES")
		_ = os.Unsetenv("ALPINE_SHOW_TOOL_UPDATES")
		_ = os.Unsetenv("ALPINE_HTTP_ENABLED")
		_ = os.Unsetenv("ALPINE_HTTP_PORT")
	}()

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	if cfg.WorkDir != testWorkDir {
		t.Errorf("WorkDir = %q, want %q", cfg.WorkDir, testWorkDir)
	}

	if cfg.Verbosity != VerbosityDebug {
		t.Errorf("Verbosity = %q, want %q", cfg.Verbosity, VerbosityDebug)
	}

	if cfg.ShowOutput {
		t.Error("ShowOutput = true, want false")
	}

	expectedStateFile := filepath.Join("agent_state", "agent_state.json")
	if cfg.StateFile != expectedStateFile {
		t.Errorf("StateFile = %q, want %q", cfg.StateFile, expectedStateFile)
	}

	if cfg.AutoCleanup {
		t.Error("AutoCleanup = true, want false")
	}

	// Test Git configuration
	if cfg.Git.WorktreeEnabled {
		t.Error("Git.WorktreeEnabled = true, want false")
	}

	if cfg.Git.BaseBranch != "develop" {
		t.Errorf("Git.BaseBranch = %q, want %q", cfg.Git.BaseBranch, "develop")
	}

	if cfg.Git.AutoCleanupWT {
		t.Error("Git.AutoCleanupWT = true, want false")
	}

	// Test ShowTodoUpdates
	if cfg.ShowTodoUpdates {
		t.Error("ShowTodoUpdates = true, want false")
	}

	// Test ShowToolUpdates
	if cfg.ShowToolUpdates {
		t.Error("ShowToolUpdates = true, want false")
	}

	// Test HTTPEnabled
	if !cfg.HTTPEnabled {
		t.Error("HTTPEnabled = false, want true")
	}

	// Test HTTPPort
	if cfg.HTTPPort != 9090 {
		t.Errorf("HTTPPort = %d, want 9090", cfg.HTTPPort)
	}
}

// TestConfig_ShowToolUpdatesDefault tests that ShowToolUpdates defaults to true
func TestConfig_ShowToolUpdatesDefault(t *testing.T) {
	// Clear the environment variable to test default
	_ = os.Unsetenv("ALPINE_SHOW_TOOL_UPDATES")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// ShowToolUpdates should default to true
	if !cfg.ShowToolUpdates {
		t.Error("ShowToolUpdates = false, want true (default)")
	}
}

// TestConfig_ShowToolUpdatesEnvVar tests that ALPINE_SHOW_TOOL_UPDATES correctly sets the config value
func TestConfig_ShowToolUpdatesEnvVar(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		wantValue bool
		wantErr   bool
	}{
		{
			name:      "set to false",
			envValue:  "false",
			wantValue: false,
			wantErr:   false,
		},
		{
			name:      "set to true",
			envValue:  "true",
			wantValue: true,
			wantErr:   false,
		},
		{
			name:     "invalid value",
			envValue: "maybe",
			wantErr:  true,
		},
		{
			name:      "empty value uses default",
			envValue:  "",
			wantValue: true, // Should use default (true)
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset the environment variable
			if tt.envValue == "" {
				_ = os.Unsetenv("ALPINE_SHOW_TOOL_UPDATES")
			} else {
				_ = os.Setenv("ALPINE_SHOW_TOOL_UPDATES", tt.envValue)
			}

			// Clean up after test
			defer func() {
				_ = os.Unsetenv("ALPINE_SHOW_TOOL_UPDATES")
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && cfg.ShowToolUpdates != tt.wantValue {
				t.Errorf("ShowToolUpdates = %v, want %v", cfg.ShowToolUpdates, tt.wantValue)
			}
		})
	}
}

// TestValidateWorkDir tests validation of the WorkDir configuration
func TestValidateWorkDir(t *testing.T) {
	tests := []struct {
		name    string
		workDir string
		wantErr bool
	}{
		{
			name:    "absolute path",
			workDir: "/absolute/path",
			wantErr: false,
		},
		{
			name:    "relative path",
			workDir: "./relative/path",
			wantErr: true,
		},
		{
			name:    "empty path",
			workDir: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("ALPINE_WORKDIR", tt.workDir)
			defer func() {
				_ = os.Unsetenv("ALPINE_WORKDIR")
			}()

			_, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateVerbosity tests validation of the Verbosity configuration
func TestValidateVerbosity(t *testing.T) {
	tests := []struct {
		name      string
		verbosity string
		want      Verbosity
		wantErr   bool
	}{
		{
			name:      "normal",
			verbosity: "normal",
			want:      VerbosityNormal,
			wantErr:   false,
		},
		{
			name:      "verbose",
			verbosity: "verbose",
			want:      VerbosityVerbose,
			wantErr:   false,
		},
		{
			name:      "debug",
			verbosity: "debug",
			want:      VerbosityDebug,
			wantErr:   false,
		},
		{
			name:      "invalid",
			verbosity: "invalid",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("ALPINE_VERBOSITY", tt.verbosity)
			defer func() {
				_ = os.Unsetenv("ALPINE_VERBOSITY")
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && cfg.Verbosity != tt.want {
				t.Errorf("Verbosity = %q, want %q", cfg.Verbosity, tt.want)
			}
		})
	}
}

// TestValidateBooleanFields tests validation of boolean configuration fields
func TestValidateBooleanFields(t *testing.T) {
	tests := []struct {
		name       string
		envVar     string
		value      string
		wantErr    bool
		wantResult bool
	}{
		{
			name:       "true value",
			envVar:     "ALPINE_SHOW_OUTPUT",
			value:      "true",
			wantErr:    false,
			wantResult: true,
		},
		{
			name:       "false value",
			envVar:     "ALPINE_SHOW_OUTPUT",
			value:      "false",
			wantErr:    false,
			wantResult: false,
		},
		{
			name:    "invalid value",
			envVar:  "ALPINE_SHOW_OUTPUT",
			value:   "yes",
			wantErr: true,
		},
		{
			name:       "auto cleanup true",
			envVar:     "ALPINE_AUTO_CLEANUP",
			value:      "true",
			wantErr:    false,
			wantResult: true,
		},
		{
			name:       "auto cleanup false",
			envVar:     "ALPINE_AUTO_CLEANUP",
			value:      "false",
			wantErr:    false,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(tt.envVar, tt.value)
			defer func() {
				_ = os.Unsetenv(tt.envVar)
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				var result bool
				switch tt.envVar {
				case "ALPINE_SHOW_OUTPUT":
					result = cfg.ShowOutput
				case "ALPINE_AUTO_CLEANUP":
					result = cfg.AutoCleanup
				}

				if result != tt.wantResult {
					t.Errorf("%s = %v, want %v", tt.envVar, result, tt.wantResult)
				}
			}
		})
	}
}

// TestStateFileIsFixed tests that state file is always at a fixed location
func TestStateFileIsFixed(t *testing.T) {
	// Try to set ALPINE_STATE_FILE - it should be ignored
	_ = os.Setenv("ALPINE_STATE_FILE", "/custom/path/state.json")
	defer func() { _ = os.Unsetenv("ALPINE_STATE_FILE") }()

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// State file should always be at the fixed location
	expectedStateFile := filepath.Join("agent_state", "agent_state.json")
	if cfg.StateFile != expectedStateFile {
		t.Errorf("StateFile = %q, want %q (should ignore ALPINE_STATE_FILE env var)", cfg.StateFile, expectedStateFile)
	}
}

// TestConfigMethods tests helper methods on the Config struct
func TestConfigMethods(t *testing.T) {
	t.Run("IsVerbose", func(t *testing.T) {
		tests := []struct {
			verbosity Verbosity
			want      bool
		}{
			{VerbosityNormal, false},
			{VerbosityVerbose, true},
			{VerbosityDebug, true},
		}

		for _, tt := range tests {
			cfg := &Config{Verbosity: tt.verbosity}
			if got := cfg.IsVerbose(); got != tt.want {
				t.Errorf("IsVerbose() with %q = %v, want %v", tt.verbosity, got, tt.want)
			}
		}
	})

	t.Run("IsDebug", func(t *testing.T) {
		tests := []struct {
			verbosity Verbosity
			want      bool
		}{
			{VerbosityNormal, false},
			{VerbosityVerbose, false},
			{VerbosityDebug, true},
		}

		for _, tt := range tests {
			cfg := &Config{Verbosity: tt.verbosity}
			if got := cfg.IsDebug(); got != tt.want {
				t.Errorf("IsDebug() with %q = %v, want %v", tt.verbosity, got, tt.want)
			}
		}
	})
}

// TestConfigGitDefaults tests default values for Git configuration
func TestConfigGitDefaults(t *testing.T) {
	// Clear all Git-related environment variables
	_ = os.Unsetenv("ALPINE_GIT_ENABLED")
	_ = os.Unsetenv("ALPINE_GIT_BASE_BRANCH")
	_ = os.Unsetenv("ALPINE_GIT_AUTO_CLEANUP")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// Test Git defaults
	if !cfg.Git.WorktreeEnabled {
		t.Error("Git.WorktreeEnabled = false, want true (default)")
	}

	if cfg.Git.BaseBranch != "main" {
		t.Errorf("Git.BaseBranch = %q, want %q (default)", cfg.Git.BaseBranch, "main")
	}

	if !cfg.Git.AutoCleanupWT {
		t.Error("Git.AutoCleanupWT = false, want true (default)")
	}
}

// TestConfigGitEnvironmentVariables tests Git configuration from environment variables
func TestConfigGitEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    GitConfig
		wantErr bool
	}{
		{
			name: "all false",
			envVars: map[string]string{
				"ALPINE_GIT_ENABLED":      "false",
				"ALPINE_GIT_BASE_BRANCH":  "master",
				"ALPINE_GIT_AUTO_CLEANUP": "false",
			},
			want: GitConfig{
				WorktreeEnabled: false,
				BaseBranch:      "master",
				AutoCleanupWT:   false,
			},
		},
		{
			name: "mixed values",
			envVars: map[string]string{
				"ALPINE_GIT_ENABLED":      "true",
				"ALPINE_GIT_BASE_BRANCH":  "develop",
				"ALPINE_GIT_AUTO_CLEANUP": "false",
			},
			want: GitConfig{
				WorktreeEnabled: true,
				BaseBranch:      "develop",
				AutoCleanupWT:   false,
			},
		},
		{
			name: "invalid boolean for enabled",
			envVars: map[string]string{
				"ALPINE_GIT_ENABLED": "yes",
			},
			wantErr: true,
		},
		{
			name: "invalid boolean for auto cleanup",
			envVars: map[string]string{
				"ALPINE_GIT_AUTO_CLEANUP": "1",
			},
			wantErr: true,
		},
		{
			name: "empty base branch uses default",
			envVars: map[string]string{
				"ALPINE_GIT_BASE_BRANCH": "",
			},
			want: GitConfig{
				WorktreeEnabled: true,   // default
				BaseBranch:      "main", // default when empty
				AutoCleanupWT:   true,   // default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
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

			if err == nil {
				if cfg.Git.WorktreeEnabled != tt.want.WorktreeEnabled {
					t.Errorf("Git.WorktreeEnabled = %v, want %v", cfg.Git.WorktreeEnabled, tt.want.WorktreeEnabled)
				}
				if cfg.Git.BaseBranch != tt.want.BaseBranch {
					t.Errorf("Git.BaseBranch = %q, want %q", cfg.Git.BaseBranch, tt.want.BaseBranch)
				}
				if cfg.Git.AutoCleanupWT != tt.want.AutoCleanupWT {
					t.Errorf("Git.AutoCleanupWT = %v, want %v", cfg.Git.AutoCleanupWT, tt.want.AutoCleanupWT)
				}
			}
		})
	}
}

// TestHTTPConfigDefaults tests that HTTP server configuration defaults to disabled
// This test verifies the acceptance criteria that HTTP mode is disabled by default with port 8080
func TestHTTPConfigDefaults(t *testing.T) {
	// Clear HTTP-related environment variables to test defaults
	_ = os.Unsetenv("ALPINE_HTTP_ENABLED")
	_ = os.Unsetenv("ALPINE_HTTP_PORT")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// HTTP should be disabled by default
	if cfg.HTTPEnabled {
		t.Error("HTTPEnabled = true, want false (default)")
	}

	// Default port should be 8080
	if cfg.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080 (default)", cfg.HTTPPort)
	}
}

// TestHTTPConfigEnvironmentVariables tests loading HTTP configuration from environment
// This test verifies that ALPINE_HTTP_ENABLED and ALPINE_HTTP_PORT are parsed correctly
func TestHTTPConfigEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name        string
		httpEnabled string
		httpPort    string
		wantEnabled bool
		wantPort    int
		wantErr     bool
	}{
		{
			name:        "HTTP enabled with custom port",
			httpEnabled: "true",
			httpPort:    "9090",
			wantEnabled: true,
			wantPort:    9090,
			wantErr:     false,
		},
		{
			name:        "HTTP disabled with default port",
			httpEnabled: "false",
			httpPort:    "",
			wantEnabled: false,
			wantPort:    8080,
			wantErr:     false,
		},
		{
			name:        "Invalid boolean for enabled",
			httpEnabled: "yes",
			httpPort:    "8080",
			wantEnabled: false,
			wantPort:    0,
			wantErr:     true,
		},
		{
			name:        "Invalid port number",
			httpEnabled: "true",
			httpPort:    "not-a-number",
			wantEnabled: false,
			wantPort:    0,
			wantErr:     true,
		},
		{
			name:        "Port out of range (negative)",
			httpEnabled: "true",
			httpPort:    "-1",
			wantEnabled: false,
			wantPort:    0,
			wantErr:     true,
		},
		{
			name:        "Port out of range (too high)",
			httpEnabled: "true",
			httpPort:    "70000",
			wantEnabled: false,
			wantPort:    0,
			wantErr:     true,
		},
		{
			name:        "Empty values use defaults",
			httpEnabled: "",
			httpPort:    "",
			wantEnabled: false,
			wantPort:    8080,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.httpEnabled != "" {
				_ = os.Setenv("ALPINE_HTTP_ENABLED", tt.httpEnabled)
			} else {
				_ = os.Unsetenv("ALPINE_HTTP_ENABLED")
			}
			if tt.httpPort != "" {
				_ = os.Setenv("ALPINE_HTTP_PORT", tt.httpPort)
			} else {
				_ = os.Unsetenv("ALPINE_HTTP_PORT")
			}

			// Clean up after test
			defer func() {
				_ = os.Unsetenv("ALPINE_HTTP_ENABLED")
				_ = os.Unsetenv("ALPINE_HTTP_PORT")
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if cfg.HTTPEnabled != tt.wantEnabled {
					t.Errorf("HTTPEnabled = %v, want %v", cfg.HTTPEnabled, tt.wantEnabled)
				}
				if cfg.HTTPPort != tt.wantPort {
					t.Errorf("HTTPPort = %d, want %d", cfg.HTTPPort, tt.wantPort)
				}
			}
		})
	}
}

// TestHTTPPortValidation tests specific port validation edge cases
// This test ensures we handle invalid port numbers gracefully
func TestHTTPPortValidation(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "port 0 is invalid",
			port:    "0",
			wantErr: true,
			errMsg:  "must be between 1 and 65535",
		},
		{
			name:    "port 1 is valid",
			port:    "1",
			wantErr: false,
		},
		{
			name:    "port 65535 is valid",
			port:    "65535",
			wantErr: false,
		},
		{
			name:    "port 65536 is invalid",
			port:    "65536",
			wantErr: true,
			errMsg:  "must be between 1 and 65535",
		},
		{
			name:    "empty string uses default",
			port:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("ALPINE_HTTP_PORT", tt.port)
			defer func() {
				_ = os.Unsetenv("ALPINE_HTTP_PORT")
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error message %q does not contain %q", err.Error(), tt.errMsg)
			}

			if err == nil && tt.port != "" {
				portNum, _ := strconv.Atoi(tt.port)
				if cfg.HTTPPort != portNum {
					t.Errorf("HTTPPort = %d, want %d", cfg.HTTPPort, portNum)
				}
			}
		})
	}
}
