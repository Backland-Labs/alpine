package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	setupDir := func(t *testing.T, yaml string) {
		t.Helper()
		dir := t.TempDir()
		orig, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(orig) })
		if yaml != "" {
			if err := os.WriteFile(filepath.Join(dir, "alpine.yaml"), []byte(yaml), 0644); err != nil {
				t.Fatalf("write config: %v", err)
			}
		}
	}

	t.Run("defaults when file missing", func(t *testing.T) {
		setupDir(t, "")
		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		if cfg.Sandbox.ImageProfile != "default" {
			t.Fatalf("image profile = %q", cfg.Sandbox.ImageProfile)
		}
		if !cfg.Sandbox.AutoTeardown {
			t.Fatal("expected auto teardown default true")
		}
		if cfg.GitHub.BranchPrefix != "alpine" {
			t.Fatalf("branch prefix = %q", cfg.GitHub.BranchPrefix)
		}
	})

	t.Run("custom cloud-first schema", func(t *testing.T) {
		yaml := strings.Join([]string{
			"repo:",
			"  default: https://github.com/acme/repo.git",
			"sandbox:",
			"  image_profile: heavy",
			"  image_profiles:",
			"    default: opencode-default",
			"    heavy: opencode-heavy",
			"  web_base_url: https://sandbox.example.com",
			"  auto_teardown: false",
			"  completed_retention_minutes: 30",
			"durability:",
			"  bucket: checkpoints",
			"  checkpoint_prefix: sessions",
			"github:",
			"  branch_prefix: agent",
			"  require_auth: false",
		}, "\n")
		setupDir(t, yaml)
		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("loadConfig: %v", err)
		}
		if cfg.Repo.Default == "" || cfg.Sandbox.ImageProfiles["heavy"] == "" {
			t.Fatalf("expected parsed custom config: %+v", cfg)
		}
		if cfg.Sandbox.AutoTeardown {
			t.Fatal("expected auto_teardown false")
		}
	})

	t.Run("validation error surfaces", func(t *testing.T) {
		setupDir(t, "sandbox:\n  image_profile: gpu\n")
		_, err := loadConfig()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "sandbox.image_profile") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("read error when alpine.yaml is directory", func(t *testing.T) {
		dir := t.TempDir()
		orig, _ := os.Getwd()
		if err := os.Chdir(dir); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(orig) })
		if err := os.Mkdir(filepath.Join(dir, "alpine.yaml"), 0755); err != nil {
			t.Fatalf("mkdir alpine.yaml: %v", err)
		}
		_, err := loadConfig()
		if err == nil || !strings.Contains(err.Error(), "reading alpine.yaml") {
			t.Fatalf("expected reading error, got: %v", err)
		}
	})
}

func TestResolveRepo(t *testing.T) {
	cfg := &Config{}

	t.Run("flag override wins", func(t *testing.T) {
		repo, err := cfg.resolveRepo("https://github.com/acme/repo.git")
		if err != nil {
			t.Fatalf("resolveRepo: %v", err)
		}
		if repo == "" {
			t.Fatal("expected repo")
		}
	})

	t.Run("config default used", func(t *testing.T) {
		cfg.Repo.Default = "git@github.com:acme/repo.git"
		repo, err := cfg.resolveRepo("")
		if err != nil {
			t.Fatalf("resolveRepo: %v", err)
		}
		if repo != cfg.Repo.Default {
			t.Fatalf("repo=%q", repo)
		}
	})

	t.Run("missing repo fails", func(t *testing.T) {
		cfg.Repo.Default = ""
		_, err := cfg.resolveRepo("")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid repo fails validation", func(t *testing.T) {
		_, err := cfg.resolveRepo("file:///tmp/repo")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestResolveImageProfile(t *testing.T) {
	cfg := &Config{Sandbox: SandboxConfig{ImageProfile: "default", ImageProfiles: map[string]string{"default": "class-a", "gpu": "class-b"}}}

	t.Run("default profile", func(t *testing.T) {
		profile, class, err := cfg.resolveImageProfile("")
		if err != nil {
			t.Fatalf("resolveImageProfile: %v", err)
		}
		if profile != "default" || class != "class-a" {
			t.Fatalf("got (%s,%s)", profile, class)
		}
	})

	t.Run("override profile", func(t *testing.T) {
		profile, class, err := cfg.resolveImageProfile("gpu")
		if err != nil {
			t.Fatalf("resolveImageProfile: %v", err)
		}
		if profile != "gpu" || class != "class-b" {
			t.Fatalf("got (%s,%s)", profile, class)
		}
	})

	t.Run("unknown profile", func(t *testing.T) {
		_, _, err := cfg.resolveImageProfile("unknown")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateRepoURL(t *testing.T) {
	for _, tc := range []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{name: "https", repo: "https://github.com/acme/repo.git"},
		{name: "ssh", repo: "ssh://git@github.com/acme/repo.git"},
		{name: "git@", repo: "git@github.com:acme/repo.git"},
		{name: "spaces", repo: "not valid", wantErr: true},
		{name: "empty", repo: "", wantErr: true},
		{name: "bad git@", repo: "git@github.com/acme/repo.git", wantErr: true},
		{name: "bad scheme", repo: "file:///tmp/repo", wantErr: true},
		{name: "missing host", repo: "https:///repo", wantErr: true},
		{name: "parse error", repo: "https://%zz", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRepoURL(tc.repo)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigValidateBranches(t *testing.T) {
	base := Config{
		BaseImage: "ubuntu:24.04",
		Sandbox: SandboxConfig{
			ImageProfile:           "default",
			ImageProfiles:          map[string]string{"default": "class-a"},
			WebBaseURL:             "https://sandbox.example.com",
			CompletedRetentionMins: 10,
		},
		Durability: DurabilityConfig{Bucket: "b", CheckpointPrefix: "p"},
		GitHub:     GitHubConfig{BranchPrefix: "alpine"},
	}

	tests := []struct {
		name string
		edit func(*Config)
	}{
		{name: "empty image profile", edit: func(c *Config) { c.Sandbox.ImageProfile = "" }},
		{name: "missing profile map", edit: func(c *Config) { c.Sandbox.ImageProfiles = map[string]string{} }},
		{name: "profile missing in map", edit: func(c *Config) { c.Sandbox.ImageProfile = "gpu" }},
		{name: "empty web base", edit: func(c *Config) { c.Sandbox.WebBaseURL = "" }},
		{name: "invalid web base", edit: func(c *Config) { c.Sandbox.WebBaseURL = "not a url" }},
		{name: "negative retention", edit: func(c *Config) { c.Sandbox.CompletedRetentionMins = -1 }},
		{name: "empty bucket", edit: func(c *Config) { c.Durability.Bucket = "" }},
		{name: "empty prefix", edit: func(c *Config) { c.Durability.CheckpointPrefix = "" }},
		{name: "empty branch prefix", edit: func(c *Config) { c.GitHub.BranchPrefix = "" }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			cfg.Sandbox.ImageProfiles = map[string]string{"default": "class-a"}
			tc.edit(&cfg)
			if err := cfg.validate(); err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
		})
	}
}

func TestUsesControlPlane(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"empty", "", false},
		{"whitespace", "   ", false},
		{"localhost", "http://127.0.0.1:8787", true},
		{"https url", "https://worker.example.com", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{Sandbox: SandboxConfig{ControlPlaneURL: tc.url}}
			if got := cfg.usesControlPlane(); got != tc.expected {
				t.Errorf("usesControlPlane() = %t, want %t", got, tc.expected)
			}
		})
	}
}

func TestControlPlaneURLValidation(t *testing.T) {
	base := Config{
		BaseImage: "ubuntu:24.04",
		Sandbox: SandboxConfig{
			ImageProfile:           "default",
			ImageProfiles:          map[string]string{"default": "class-a"},
			WebBaseURL:             "https://sandbox.example.com",
			CompletedRetentionMins: 10,
		},
		Durability: DurabilityConfig{Bucket: "b", CheckpointPrefix: "p"},
		GitHub:     GitHubConfig{BranchPrefix: "alpine"},
	}

	t.Run("valid control plane url", func(t *testing.T) {
		cfg := base
		cfg.Sandbox.ControlPlaneURL = "http://127.0.0.1:8787"
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid control plane url", func(t *testing.T) {
		cfg := base
		cfg.Sandbox.ControlPlaneURL = "not a url"
		if err := cfg.validate(); err == nil {
			t.Error("expected error for invalid control plane url")
		}
	})

	t.Run("empty control plane url is valid", func(t *testing.T) {
		cfg := base
		cfg.Sandbox.ControlPlaneURL = ""
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
