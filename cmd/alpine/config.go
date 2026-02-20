package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents alpine.yaml configuration
type Config struct {
	Install   string   `yaml:"install"`
	EnvFiles  []string `yaml:"env_files"`
	BaseImage string   `yaml:"base_image"`
	Services  []string `yaml:"services"`

	Repo       RepoConfig       `yaml:"repo"`
	Sandbox    SandboxConfig    `yaml:"sandbox"`
	Durability DurabilityConfig `yaml:"durability"`
	GitHub     GitHubConfig     `yaml:"github"`
}

type RepoConfig struct {
	Default string `yaml:"default"`
}

type SandboxConfig struct {
	ImageProfile           string            `yaml:"image_profile"`
	ImageProfiles          map[string]string `yaml:"image_profiles"`
	WebBaseURL             string            `yaml:"web_base_url"`
	AutoTeardown           bool              `yaml:"auto_teardown"`
	CompletedRetentionMins int               `yaml:"completed_retention_minutes"`
}

type DurabilityConfig struct {
	Bucket           string `yaml:"bucket"`
	CheckpointPrefix string `yaml:"checkpoint_prefix"`
}

type GitHubConfig struct {
	BranchPrefix string `yaml:"branch_prefix"`
	RequireAuth  bool   `yaml:"require_auth"`
}

// validServices is the set of recognized service names
var validServices = map[string]bool{
	"postgres": true,
	"redis":    true,
	"browser":  true,
}

// loadConfig reads and validates alpine.yaml from the current directory.
// Returns default config with a warning if file is missing.
func loadConfig() (*Config, error) {
	cfg := &Config{
		BaseImage: "ubuntu:24.04",
		Sandbox: SandboxConfig{
			ImageProfile: "default",
			ImageProfiles: map[string]string{
				"default": "opencode-default",
			},
			WebBaseURL:             "https://opencode.cloudflare.com",
			AutoTeardown:           true,
			CompletedRetentionMins: 60,
		},
		Durability: DurabilityConfig{
			Bucket:           "alpine-checkpoints",
			CheckpointPrefix: "sandboxes",
		},
		GitHub: GitHubConfig{
			BranchPrefix: "alpine",
			RequireAuth:  true,
		},
	}

	data, err := os.ReadFile("alpine.yaml")
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("alpine.yaml not found, using defaults")
			return cfg, nil
		}
		return nil, fmt.Errorf("reading alpine.yaml: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing alpine.yaml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.BaseImage == "" {
		return fmt.Errorf("base_image cannot be empty")
	}
	for _, svc := range c.Services {
		if !validServices[svc] {
			return fmt.Errorf("unknown service %q (supported: postgres, redis, browser)", svc)
		}
	}

	if c.Sandbox.ImageProfile == "" {
		return fmt.Errorf("sandbox.image_profile cannot be empty")
	}
	if len(c.Sandbox.ImageProfiles) == 0 {
		return fmt.Errorf("sandbox.image_profiles must include at least one profile")
	}
	if _, ok := c.Sandbox.ImageProfiles[c.Sandbox.ImageProfile]; !ok {
		return fmt.Errorf("sandbox.image_profile %q not found in sandbox.image_profiles", c.Sandbox.ImageProfile)
	}
	if c.Sandbox.WebBaseURL == "" {
		return fmt.Errorf("sandbox.web_base_url cannot be empty")
	}
	if _, err := url.ParseRequestURI(c.Sandbox.WebBaseURL); err != nil {
		return fmt.Errorf("sandbox.web_base_url is invalid: %w", err)
	}
	if c.Sandbox.CompletedRetentionMins < 0 {
		return fmt.Errorf("sandbox.completed_retention_minutes cannot be negative")
	}

	if c.Durability.Bucket == "" {
		return fmt.Errorf("durability.bucket cannot be empty")
	}
	if c.Durability.CheckpointPrefix == "" {
		return fmt.Errorf("durability.checkpoint_prefix cannot be empty")
	}

	if c.GitHub.BranchPrefix == "" {
		return fmt.Errorf("github.branch_prefix cannot be empty")
	}

	return nil
}

func (c *Config) resolveRepo(override string) (string, error) {
	repo := strings.TrimSpace(override)
	if repo == "" {
		repo = strings.TrimSpace(c.Repo.Default)
	}
	if repo == "" {
		return "", fmt.Errorf("repository is required (use --repo or alpine.yaml repo.default)")
	}
	if err := validateRepoURL(repo); err != nil {
		return "", err
	}
	return repo, nil
}

func (c *Config) resolveImageProfile(override string) (string, string, error) {
	profile := strings.TrimSpace(override)
	if profile == "" {
		profile = c.Sandbox.ImageProfile
	}
	containerClass, ok := c.Sandbox.ImageProfiles[profile]
	if !ok {
		return "", "", fmt.Errorf("unknown image profile %q", profile)
	}
	return profile, containerClass, nil
}

func validateRepoURL(repo string) error {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return fmt.Errorf("repository cannot be empty")
	}
	if strings.Contains(repo, " ") {
		return fmt.Errorf("repository cannot contain spaces")
	}

	if strings.HasPrefix(repo, "git@") {
		if !strings.Contains(repo, ":") {
			return fmt.Errorf("invalid git SSH repository %q", repo)
		}
		return nil
	}

	u, err := url.Parse(repo)
	if err != nil {
		return fmt.Errorf("invalid repository URL %q", repo)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid repository URL %q", repo)
	}
	if u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "ssh" {
		return fmt.Errorf("unsupported repository URL scheme %q", u.Scheme)
	}
	return nil
}
