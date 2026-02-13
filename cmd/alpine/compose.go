package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

// ServiceConfig holds defaults for a supported service.
type ServiceConfig struct {
	Image       string
	Healthcheck string
	Tmpfs       string // empty if no tmpfs needed
	ExtraCmd    string // extra command args (e.g. redis --save "")
}

// serviceDefaults maps service names to their default configuration.
var serviceDefaults = map[string]ServiceConfig{
	"postgres": {
		Image:       "postgres:16",
		Healthcheck: "pg_isready -U postgres",
		Tmpfs:       "/var/lib/postgresql/data:size=512M",
	},
	"redis": {
		Image:       "redis:7",
		Healthcheck: "redis-cli ping",
		ExtraCmd:    `redis-server --save "" --appendonly no`,
	},
	"browser": {
		Image:       "browserless/chromium:latest",
		Healthcheck: "wget -q --spider http://localhost:3000/json/version",
	},
}

// serviceAlias maps service names to their compose service aliases.
var serviceAlias = map[string]string{
	"postgres": "db",
	"redis":    "cache",
	"browser":  "browser",
}

// composeTmpl is the parsed Go text/template for generating docker-compose.yml.
// Key features:
//   - Dev service uses build: directive for image building
//   - SSH agent mount is platform-aware (macOS vs Linux)
//   - Environment vars use passthrough syntax only (no literal values)
//   - Labels for metadata discovery
//   - Tuned health checks (interval: 2s)
//   - Security hardening: cap_drop ALL, no-new-privileges
//   - tmpfs with size limits for services that need them
var composeTmpl = template.Must(template.New("compose").Parse(`services:
  dev:
    image: {{ .ImageTag }}
    container_name: {{ .Project }}-dev-1
    hostname: {{ .Name }}-dev
    stdin_open: true
    tty: true
    volumes:
      - {{ .SSHSocket }}:{{ .SSHTarget }}
    environment:
      - ANTHROPIC_API_KEY
      - CLAUDE_CODE_OAUTH_TOKEN
      - GITHUB_TOKEN
      - GH_TOKEN
      - SSH_AUTH_SOCK={{ .SSHTarget }}
{{- range .ExtraEnv }}
      - {{ . }}
{{- end }}
    labels:
      alpine.managed: "true"
      alpine.name: "{{ .Name }}"
      alpine.created: "{{ .Created }}"
      alpine.branch: "{{ .Branch }}"
    healthcheck:
      test: ["CMD-SHELL", "test -f /usr/bin/git"]
      interval: 2s
      timeout: 3s
      retries: 15
      start_period: 5s
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - DAC_OVERRIDE
      - FOWNER
    security_opt:
      - no-new-privileges:true
{{- range .Services }}
{{ . }}
{{- end }}
`))

// serviceTmpl generates the YAML block for a supporting service.
var serviceTmpl = template.Must(template.New("service").Parse(`  {{ .Alias }}:
    image: {{ .Image }}
{{- if .ExtraCmd }}
    command: {{ .ExtraCmd }}
{{- end }}
{{- if .Tmpfs }}
    tmpfs:
      - {{ .Tmpfs }}
{{- end }}
    environment:
{{- if eq .Alias "db" }}
      - POSTGRES_HOST_AUTH_METHOD=trust
{{- end }}
    healthcheck:
{{- if .UseCMDShell }}
      test: ["CMD-SHELL", "{{ .Healthcheck }}"]
{{- else }}
      test: ["CMD", {{ .HealthcheckParts }}]
{{- end }}
      interval: 2s
      timeout: 3s
      retries: 15
      start_period: {{ .StartPeriod }}
    labels:
      alpine.managed: "true"
      alpine.name: "{{ .EnvName }}"
`))

// serviceTemplateData holds data for rendering a service block.
type serviceTemplateData struct {
	Alias            string
	Image            string
	ExtraCmd         string
	Tmpfs            string
	Healthcheck      string
	UseCMDShell      bool
	HealthcheckParts string
	StartPeriod      string
	EnvName          string
}

// composeData holds all the data needed to render the compose template.
type composeData struct {
	ImageTag  string
	Project   string
	Name      string
	SSHSocket string
	SSHTarget string
	Created   string
	Branch    string
	Services  []string
	ExtraEnv  []string
}

// generateComposeYAML produces a docker-compose.yml from the given config.
//
// CRITICAL: environment variables use passthrough syntax only
// (e.g., "- ANTHROPIC_API_KEY" not "- ANTHROPIC_API_KEY=value").
// The generated YAML never contains literal secret values.
func generateComposeYAML(cfg *Config, name, branch, platform, imageTag string) ([]byte, error) {
	project := "alpine-" + name
	created := time.Now().UTC().Format(time.RFC3339)

	// Determine SSH socket paths based on platform.
	var sshSocket, sshTarget string
	if platform == "darwin" {
		sshSocket = "/run/host-services/ssh-auth.sock"
		sshTarget = "/run/host-services/ssh-auth.sock"
	} else {
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			sock = "/tmp/ssh-agent.sock"
		}
		sshSocket = sock
		sshTarget = sock
	}

	// Generate service blocks for each configured service.
	var serviceBlocks []string
	for _, svc := range cfg.Services {
		defaults, ok := serviceDefaults[svc]
		if !ok {
			return nil, fmt.Errorf("unsupported service: %q (supported: postgres, redis, browser)", svc)
		}
		alias := serviceAlias[svc]

		data := serviceTemplateData{
			Alias:       alias,
			Image:       defaults.Image,
			ExtraCmd:    defaults.ExtraCmd,
			Tmpfs:       defaults.Tmpfs,
			Healthcheck: defaults.Healthcheck,
			EnvName:     name,
		}

		// Determine health check format and start period.
		switch svc {
		case "postgres":
			data.UseCMDShell = true
			data.StartPeriod = "5s"
		case "browser":
			data.UseCMDShell = true
			data.StartPeriod = "10s"
		default:
			data.UseCMDShell = false
			// Split the healthcheck command into JSON array parts for CMD format.
			parts := strings.Fields(defaults.Healthcheck)
			quoted := make([]string, len(parts))
			for i, p := range parts {
				quoted[i] = fmt.Sprintf("%q", p)
			}
			data.HealthcheckParts = strings.Join(quoted, ", ")
			data.StartPeriod = "3s"
		}

		var buf bytes.Buffer
		if err := serviceTmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to render service %q: %w", svc, err)
		}
		serviceBlocks = append(serviceBlocks, buf.String())
	}

	// Inject extra environment variables into the dev container based on enabled services.
	var extraEnv []string
	for _, svc := range cfg.Services {
		if svc == "browser" {
			extraEnv = append(extraEnv, "BROWSER_WS_ENDPOINT=ws://browser:3000")
		}
	}

	data := composeData{
		ImageTag:  imageTag,
		Project:   project,
		Name:      name,
		SSHSocket: sshSocket,
		SSHTarget: sshTarget,
		Created:   created,
		Branch:    branch,
		Services:  serviceBlocks,
		ExtraEnv:  extraEnv,
	}

	var buf bytes.Buffer
	if err := composeTmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render compose YAML: %w", err)
	}

	return buf.Bytes(), nil
}
