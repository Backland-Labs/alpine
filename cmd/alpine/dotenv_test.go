package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		preSet   map[string]string // env vars to set before loading
		wantVars map[string]string
		wantErr  bool
	}{
		{
			name:     "basic key=value",
			content:  "FOO=bar\nBAZ=qux\n",
			wantVars: map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:     "comments and blanks skipped",
			content:  "# comment\n\n   \nFOO=bar\n",
			wantVars: map[string]string{"FOO": "bar"},
		},
		{
			name:     "export prefix stripped",
			content:  "export FOO=bar\n",
			wantVars: map[string]string{"FOO": "bar"},
		},
		{
			name:     "double quoted values",
			content:  `FOO="hello world"` + "\n",
			wantVars: map[string]string{"FOO": "hello world"},
		},
		{
			name:     "single quoted values",
			content:  `FOO='hello world'` + "\n",
			wantVars: map[string]string{"FOO": "hello world"},
		},
		{
			name:     "existing env not overwritten",
			content:  "FOO=new\n",
			preSet:   map[string]string{"FOO": "existing"},
			wantVars: map[string]string{"FOO": "existing"},
		},
		{
			name:     "lines without = skipped",
			content:  "NOEQUALS\nFOO=bar\n",
			wantVars: map[string]string{"FOO": "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean env for each sub-test.
			for k := range tt.wantVars {
				t.Setenv(k, "")
				_ = os.Unsetenv(k)
			}
			for k, v := range tt.preSet {
				t.Setenv(k, v)
			}

			dir := t.TempDir()
			path := filepath.Join(dir, ".env")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("writing .env: %v", err)
			}

			err := loadDotEnv(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, want := range tt.wantVars {
				got := os.Getenv(k)
				if got != want {
					t.Errorf("env %s = %q, want %q", k, got, want)
				}
			}
		})
	}

	t.Run("file not found returns error", func(t *testing.T) {
		err := loadDotEnv("/nonexistent/path/.env")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func TestLoadDotEnv_SetenvError(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	// A null byte in the key causes os.Setenv to return EINVAL.
	if err := os.WriteFile(envFile, []byte("FOO\x00BAR=value\n"), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	err := loadDotEnv(envFile)
	if err == nil {
		t.Fatal("expected error from os.Setenv with null byte in key")
	}
	if !strings.Contains(err.Error(), "failed to set env var") {
		t.Fatalf("error = %q, want to contain 'failed to set env var'", err.Error())
	}
}
