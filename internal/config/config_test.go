package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewConfig tests the creation of a new Config instance with default values
func TestNewConfig(t *testing.T) {
	// Clear all environment variables to test defaults
	envVars := []string{
		"RIVER_WORKDIR",
		"RIVER_VERBOSITY",
		"RIVER_SHOW_OUTPUT",
		"RIVER_AUTO_CLEANUP",
		"RIVER_GIT_ENABLED",
		"RIVER_GIT_BASE_BRANCH",
		"RIVER_GIT_AUTO_CLEANUP",
		"RIVER_SHOW_TODO_UPDATES",
		"RIVER_SHOW_TOOL_UPDATES",
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

	expectedStateFile := filepath.Join(".claude", "river", "claude_state.json")
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
	_ = os.Setenv("RIVER_WORKDIR", testWorkDir)
	_ = os.Setenv("RIVER_VERBOSITY", "debug")
	_ = os.Setenv("RIVER_SHOW_OUTPUT", "false")
	// RIVER_STATE_FILE is no longer configurable
	_ = os.Setenv("RIVER_AUTO_CLEANUP", "false")
	_ = os.Setenv("RIVER_GIT_ENABLED", "false")
	_ = os.Setenv("RIVER_GIT_BASE_BRANCH", "develop")
	_ = os.Setenv("RIVER_GIT_AUTO_CLEANUP", "false")
	_ = os.Setenv("RIVER_SHOW_TODO_UPDATES", "false")
	_ = os.Setenv("RIVER_SHOW_TOOL_UPDATES", "false")

	defer func() {
		// Clean up
		_ = os.Unsetenv("RIVER_WORKDIR")
		_ = os.Unsetenv("RIVER_VERBOSITY")
		_ = os.Unsetenv("RIVER_SHOW_OUTPUT")
		// RIVER_STATE_FILE is no longer used
		_ = os.Unsetenv("RIVER_AUTO_CLEANUP")
		_ = os.Unsetenv("RIVER_GIT_ENABLED")
		_ = os.Unsetenv("RIVER_GIT_BASE_BRANCH")
		_ = os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")
		_ = os.Unsetenv("RIVER_SHOW_TODO_UPDATES")
		_ = os.Unsetenv("RIVER_SHOW_TOOL_UPDATES")
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

	expectedStateFile := filepath.Join(".claude", "river", "claude_state.json")
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
}

// TestConfig_ShowToolUpdatesDefault tests that ShowToolUpdates defaults to true
func TestConfig_ShowToolUpdatesDefault(t *testing.T) {
	// Clear the environment variable to test default
	_ = os.Unsetenv("RIVER_SHOW_TOOL_UPDATES")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// ShowToolUpdates should default to true
	if !cfg.ShowToolUpdates {
		t.Error("ShowToolUpdates = false, want true (default)")
	}
}

// TestConfig_ShowToolUpdatesEnvVar tests that RIVER_SHOW_TOOL_UPDATES correctly sets the config value
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
				_ = os.Unsetenv("RIVER_SHOW_TOOL_UPDATES")
			} else {
				_ = os.Setenv("RIVER_SHOW_TOOL_UPDATES", tt.envValue)
			}

			// Clean up after test
			defer func() {
				_ = os.Unsetenv("RIVER_SHOW_TOOL_UPDATES")
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
			_ = os.Setenv("RIVER_WORKDIR", tt.workDir)
			defer func() {
				_ = os.Unsetenv("RIVER_WORKDIR")
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
			_ = os.Setenv("RIVER_VERBOSITY", tt.verbosity)
			defer func() {
				_ = os.Unsetenv("RIVER_VERBOSITY")
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
			envVar:     "RIVER_SHOW_OUTPUT",
			value:      "true",
			wantErr:    false,
			wantResult: true,
		},
		{
			name:       "false value",
			envVar:     "RIVER_SHOW_OUTPUT",
			value:      "false",
			wantErr:    false,
			wantResult: false,
		},
		{
			name:    "invalid value",
			envVar:  "RIVER_SHOW_OUTPUT",
			value:   "yes",
			wantErr: true,
		},
		{
			name:       "auto cleanup true",
			envVar:     "RIVER_AUTO_CLEANUP",
			value:      "true",
			wantErr:    false,
			wantResult: true,
		},
		{
			name:       "auto cleanup false",
			envVar:     "RIVER_AUTO_CLEANUP",
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
				case "RIVER_SHOW_OUTPUT":
					result = cfg.ShowOutput
				case "RIVER_AUTO_CLEANUP":
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
	// Try to set RIVER_STATE_FILE - it should be ignored
	_ = os.Setenv("RIVER_STATE_FILE", "/custom/path/state.json")
	defer os.Unsetenv("RIVER_STATE_FILE")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	// State file should always be at the fixed location
	expectedStateFile := filepath.Join(".claude", "river", "claude_state.json")
	if cfg.StateFile != expectedStateFile {
		t.Errorf("StateFile = %q, want %q (should ignore RIVER_STATE_FILE env var)", cfg.StateFile, expectedStateFile)
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
	_ = os.Unsetenv("RIVER_GIT_ENABLED")
	_ = os.Unsetenv("RIVER_GIT_BASE_BRANCH")
	_ = os.Unsetenv("RIVER_GIT_AUTO_CLEANUP")

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
				"RIVER_GIT_ENABLED":      "false",
				"RIVER_GIT_BASE_BRANCH":  "master",
				"RIVER_GIT_AUTO_CLEANUP": "false",
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
				"RIVER_GIT_ENABLED":      "true",
				"RIVER_GIT_BASE_BRANCH":  "develop",
				"RIVER_GIT_AUTO_CLEANUP": "false",
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
				"RIVER_GIT_ENABLED": "yes",
			},
			wantErr: true,
		},
		{
			name: "invalid boolean for auto cleanup",
			envVars: map[string]string{
				"RIVER_GIT_AUTO_CLEANUP": "1",
			},
			wantErr: true,
		},
		{
			name: "empty base branch uses default",
			envVars: map[string]string{
				"RIVER_GIT_BASE_BRANCH": "",
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
