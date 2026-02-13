package main

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents alpine.yaml configuration
type Config struct {
	Install   string   `yaml:"install"`
	EnvFiles  []string `yaml:"env_files"`
	BaseImage string   `yaml:"base_image"`
	Services  []string `yaml:"services"`
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
	return nil
}
