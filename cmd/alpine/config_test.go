package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string // empty means no file
		wantErr   string
		wantImage string
	}{
		{
			name:      "file not found uses defaults",
			yaml:      "",
			wantImage: "ubuntu:24.04",
		},
		{
			name:      "valid yaml",
			yaml:      "base_image: node:20\nservices:\n  - postgres\n",
			wantImage: "node:20",
		},
		{
			name:    "invalid yaml",
			yaml:    ":\n  :\n  invalid",
			wantErr: "parsing alpine.yaml",
		},
		{
			name:    "validation failure empty base_image",
			yaml:    "base_image: \"\"\n",
			wantErr: "base_image cannot be empty",
		},
		{
			name:    "unknown service",
			yaml:    "base_image: ubuntu:24.04\nservices:\n  - mysql\n",
			wantErr: "unknown service",
		},
		{
			name:    "read error (directory instead of file)",
			yaml:    "DIRECTORY", // sentinel: create a directory named alpine.yaml
			wantErr: "reading alpine.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			origDir, _ := os.Getwd()
			os.Chdir(dir)
			t.Cleanup(func() { os.Chdir(origDir) })

			if tt.yaml == "DIRECTORY" {
				os.Mkdir(filepath.Join(dir, "alpine.yaml"), 0755)
			} else if tt.yaml != "" {
				os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(tt.yaml), 0644)
			}

			cfg, err := loadConfig()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.BaseImage != tt.wantImage {
				t.Fatalf("BaseImage = %q, want %q", cfg.BaseImage, tt.wantImage)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "empty base_image",
			cfg:     Config{BaseImage: ""},
			wantErr: "base_image cannot be empty",
		},
		{
			name:    "unknown service",
			cfg:     Config{BaseImage: "ubuntu:24.04", Services: []string{"mysql"}},
			wantErr: "unknown service",
		},
		{
			name: "valid postgres",
			cfg:  Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres"}},
		},
		{
			name: "valid redis",
			cfg:  Config{BaseImage: "ubuntu:24.04", Services: []string{"redis"}},
		},
		{
			name: "both services",
			cfg:  Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres", "redis"}},
		},
		{
			name: "no services",
			cfg:  Config{BaseImage: "ubuntu:24.04"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
