package main

import (
	"strings"
	"testing"
)

func TestGenerateComposeYAML(t *testing.T) {
	baseCfg := &Config{BaseImage: "ubuntu:24.04"}

	t.Run("darwin SSH path", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "/run/host-services/ssh-auth.sock") {
			t.Error("darwin should use /run/host-services/ssh-auth.sock")
		}
	})

	t.Run("linux SSH path uses SSH_AUTH_SOCK", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "/tmp/test-ssh.sock")
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "linux", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "/tmp/test-ssh.sock") {
			t.Error("linux should use SSH_AUTH_SOCK value")
		}
	})

	t.Run("linux SSH fallback when unset", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "linux", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "/tmp/ssh-agent.sock") {
			t.Error("linux with no SSH_AUTH_SOCK should fallback to /tmp/ssh-agent.sock")
		}
	})

	t.Run("with postgres service", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "db:") {
			t.Error("postgres should produce db: service alias")
		}
		if !strings.Contains(s, "CMD-SHELL") {
			t.Error("postgres healthcheck should use CMD-SHELL")
		}
		if !strings.Contains(s, "pg_isready") {
			t.Error("postgres healthcheck should use pg_isready")
		}
		if !strings.Contains(s, "tmpfs") {
			t.Error("postgres should have tmpfs mount")
		}
		if !strings.Contains(s, "POSTGRES_HOST_AUTH_METHOD=trust") {
			t.Error("postgres should have trust auth method")
		}
	})

	t.Run("with redis service", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"redis"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "cache:") {
			t.Error("redis should produce cache: service alias")
		}
		if !strings.Contains(s, `"CMD"`) {
			t.Error("redis healthcheck should use CMD format")
		}
		if !strings.Contains(s, "redis-cli") {
			t.Error("redis healthcheck should use redis-cli")
		}
		if !strings.Contains(s, "command:") {
			t.Error("redis should have ExtraCmd (command:)")
		}
		if !strings.Contains(s, `--save ""`) {
			t.Error("redis command should contain --save \"\"")
		}
	})

	t.Run("with browser service", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"browser"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "browser:") {
			t.Error("browser should produce browser: service block")
		}
		if !strings.Contains(s, "browserless/chromium:latest") {
			t.Error("browser should use browserless/chromium image")
		}
		if !strings.Contains(s, "CMD-SHELL") {
			t.Error("browser healthcheck should use CMD-SHELL")
		}
		if !strings.Contains(s, "wget") {
			t.Error("browser healthcheck should use wget")
		}
		if !strings.Contains(s, "start_period: 10s") {
			t.Error("browser should have 10s start_period")
		}
	})

	t.Run("browser injects BROWSER_WS_ENDPOINT into dev", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"browser"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "BROWSER_WS_ENDPOINT=ws://browser:3000") {
			t.Error("dev container should have BROWSER_WS_ENDPOINT when browser is enabled")
		}
	})

	t.Run("no BROWSER_WS_ENDPOINT without browser", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if strings.Contains(s, "BROWSER_WS_ENDPOINT") {
			t.Error("dev container should NOT have BROWSER_WS_ENDPOINT when browser is not enabled")
		}
	})

	t.Run("both services", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres", "redis"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "db:") || !strings.Contains(s, "cache:") {
			t.Error("both service aliases should be present")
		}
	})

	t.Run("all three services", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"postgres", "redis", "browser"}}
		yaml, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "db:") || !strings.Contains(s, "cache:") || !strings.Contains(s, "browser:") {
			t.Error("all three service aliases should be present")
		}
		if !strings.Contains(s, "BROWSER_WS_ENDPOINT=ws://browser:3000") {
			t.Error("BROWSER_WS_ENDPOINT should be injected with all three services")
		}
	})

	t.Run("unsupported service returns error", func(t *testing.T) {
		cfg := &Config{BaseImage: "ubuntu:24.04", Services: []string{"mysql"}}
		_, err := generateComposeYAML(cfg, "test", "main", "darwin", "alpine-dev:abc")
		if err == nil {
			t.Fatal("expected error for unsupported service")
		}
		if !strings.Contains(err.Error(), "unsupported service") {
			t.Fatalf("error = %q, want to contain 'unsupported service'", err.Error())
		}
	})

	t.Run("labels present", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "myenv", "feat", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, `alpine.managed: "true"`) {
			t.Error("missing alpine.managed label")
		}
		if !strings.Contains(s, `alpine.name: "myenv"`) {
			t.Error("missing alpine.name label")
		}
		if !strings.Contains(s, `alpine.branch: "feat"`) {
			t.Error("missing alpine.branch label")
		}
	})

	t.Run("cap_drop present", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		if !strings.Contains(s, "cap_drop:") {
			t.Error("missing cap_drop directive")
		}
		if !strings.Contains(s, "ALL") {
			t.Error("cap_drop should include ALL")
		}
		if !strings.Contains(s, "no-new-privileges:true") {
			t.Error("missing no-new-privileges security opt")
		}
	})

	t.Run("passthrough env syntax only", func(t *testing.T) {
		yaml, err := generateComposeYAML(baseCfg, "test", "main", "darwin", "alpine-dev:abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		s := string(yaml)
		// Secrets must use passthrough syntax (no "=value" after the key).
		for _, envVar := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
			if !strings.Contains(s, "- "+envVar) {
				t.Errorf("missing passthrough env var %s", envVar)
			}
			// Must NOT contain a literal secret value.
			if strings.Contains(s, envVar+"=sk-") {
				t.Errorf("env var %s appears to contain a literal secret", envVar)
			}
		}
	})
}
