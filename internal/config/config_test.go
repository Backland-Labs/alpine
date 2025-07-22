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
		"RIVER_STATE_FILE",
		"RIVER_AUTO_CLEANUP",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
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

	expectedStateFile := filepath.Join(".", "claude_state.json")
	if cfg.StateFile != expectedStateFile {
		t.Errorf("StateFile = %q, want %q", cfg.StateFile, expectedStateFile)
	}

	if !cfg.AutoCleanup {
		t.Error("AutoCleanup = false, want true")
	}
}

// TestConfigFromEnvironment tests loading configuration from environment variables
func TestConfigFromEnvironment(t *testing.T) {
	// Set up test environment
	testWorkDir := "/test/work/dir"
	os.Setenv("RIVER_WORKDIR", testWorkDir)
	os.Setenv("RIVER_VERBOSITY", "debug")
	os.Setenv("RIVER_SHOW_OUTPUT", "false")
	os.Setenv("RIVER_STATE_FILE", "/custom/state.json")
	os.Setenv("RIVER_AUTO_CLEANUP", "false")

	defer func() {
		// Clean up
		os.Unsetenv("RIVER_WORKDIR")
		os.Unsetenv("RIVER_VERBOSITY")
		os.Unsetenv("RIVER_SHOW_OUTPUT")
		os.Unsetenv("RIVER_STATE_FILE")
		os.Unsetenv("RIVER_AUTO_CLEANUP")
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

	if cfg.StateFile != "/custom/state.json" {
		t.Errorf("StateFile = %q, want %q", cfg.StateFile, "/custom/state.json")
	}

	if cfg.AutoCleanup {
		t.Error("AutoCleanup = true, want false")
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
			os.Setenv("RIVER_WORKDIR", tt.workDir)
			defer func() {
				os.Unsetenv("RIVER_WORKDIR")
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
			os.Setenv("RIVER_VERBOSITY", tt.verbosity)
			defer func() {
				os.Unsetenv("RIVER_VERBOSITY")
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
			os.Setenv(tt.envVar, tt.value)
			defer func() {
				os.Unsetenv(tt.envVar)
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

// TestStateFilePath tests that state file paths can be both relative and absolute
func TestStateFilePath(t *testing.T) {
	tests := []struct {
		name      string
		stateFile string
		wantErr   bool
	}{
		{
			name:      "relative path",
			stateFile: "./my_state.json",
			wantErr:   false,
		},
		{
			name:      "absolute path",
			stateFile: "/tmp/my_state.json",
			wantErr:   false,
		},
		{
			name:      "nested relative path",
			stateFile: "./nested/dir/state.json",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("RIVER_STATE_FILE", tt.stateFile)
			defer func() {
				os.Unsetenv("RIVER_STATE_FILE")
			}()

			cfg, err := New()
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && cfg.StateFile != tt.stateFile {
				t.Errorf("StateFile = %q, want %q", cfg.StateFile, tt.stateFile)
			}
		})
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