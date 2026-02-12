package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"text/template"
	"time"
)

// ---------------------------------------------------------------------------
// Timeouts
// ---------------------------------------------------------------------------

const (
	timeoutDockerHealth = 3 * time.Second
	timeoutImageBuild   = 5 * time.Minute
	timeoutComposeUp    = 2 * time.Minute
	timeoutGitClone     = 5 * time.Minute
	timeoutInstall      = 10 * time.Minute
	timeoutGitPush      = 30 * time.Second
)

// ---------------------------------------------------------------------------
// ExecError
// ---------------------------------------------------------------------------

// ExecError wraps errors from shell-out commands with stderr context.
type ExecError struct {
	Command string
	Stderr  string
	Err     error
}

func (e *ExecError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s: %v\nstderr: %s", e.Command, e.Err, e.Stderr)
	}
	return fmt.Sprintf("%s: %v", e.Command, e.Err)
}

func (e *ExecError) Unwrap() error {
	return e.Err
}

// ---------------------------------------------------------------------------
// ServiceConfig
// ---------------------------------------------------------------------------

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
}

// ---------------------------------------------------------------------------
// Shell-out helpers
// ---------------------------------------------------------------------------

// run executes a command and returns stdout and stderr as strings.
// It uses exec.CommandContext with the provided context for timeout support.
// Every external command invocation must go through this function.
// Never use "sh -c" -- always pass arguments directly to prevent shell injection.
func run(ctx context.Context, name string, args ...string) (string, string, error) {
	slog.Debug("exec", "cmd", name, "args", args)

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	outStr := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	if err != nil {
		return outStr, errStr, &ExecError{
			Command: name + " " + strings.Join(args, " "),
			Stderr:  errStr,
			Err:     err,
		}
	}
	return outStr, errStr, nil
}

// runAttached executes a command with stdin/stdout/stderr connected to the
// terminal. Used for interactive sessions (e.g., docker exec -it).
func runAttached(ctx context.Context, name string, args ...string) error {
	slog.Debug("exec (attached)", "cmd", name, "args", args)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return &ExecError{
			Command: name + " " + strings.Join(args, " "),
			Err:     err,
		}
	}
	return nil
}

// runInteractive executes an interactive command with stdin/stdout/stderr
// connected to the terminal. Unlike runAttached, it ignores SIGINT in the
// Go process so Ctrl-C is handled only by the child. This prevents the
// parent from killing the child when the user presses Ctrl-C (e.g., to
// exit Claude Code while remaining in a container shell).
func runInteractive(name string, args ...string) error {
	slog.Debug("exec (interactive)", "cmd", name, "args", args)

	// Ignore SIGINT in the Go process so Ctrl-C passes only to the child.
	signal.Ignore(os.Interrupt)

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return &ExecError{
			Command: name + " " + strings.Join(args, " "),
			Err:     err,
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Docker operations
// ---------------------------------------------------------------------------

// dockerHealthCheck verifies the Docker daemon is running by executing
// "docker info" with a 3-second timeout. On macOS, if Docker is not
// running it attempts to launch Docker Desktop and waits up to 60s
// for the daemon to become ready.
func dockerHealthCheck(ctx context.Context) error {
	hctx, cancel := context.WithTimeout(ctx, timeoutDockerHealth)
	defer cancel()

	_, stderr, err := run(hctx, "docker", "info")
	if err == nil {
		return nil
	}

	// Docker is not running. On macOS, try to start Docker Desktop.
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("Docker is not running. Start Docker and try again.\n(detail: %s)", stderr)
	}

	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "Docker is not running. Starting Docker Desktop...\n")
	}
	slog.Info("Docker is not running, starting Docker Desktop")
	if _, _, launchErr := run(ctx, "open", "-a", "Docker"); launchErr != nil {
		return fmt.Errorf("Docker is not running and failed to start Docker Desktop.\n(detail: %s)", stderr)
	}

	// Poll until the daemon is ready or we time out.
	const pollTimeout = 60 * time.Second
	pollCtx, pollCancel := context.WithTimeout(ctx, pollTimeout)
	defer pollCancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	dots := 0
	for {
		select {
		case <-pollCtx.Done():
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "\n")
			}
			return fmt.Errorf("Docker Desktop started but daemon did not become ready within %s", pollTimeout)
		case <-ticker.C:
			checkCtx, checkCancel := context.WithTimeout(pollCtx, timeoutDockerHealth)
			_, _, checkErr := run(checkCtx, "docker", "info")
			checkCancel()
			if checkErr == nil {
				if !jsonOutput {
					fmt.Fprintf(os.Stderr, "\nDocker Desktop is ready.\n")
				}
				slog.Info("Docker Desktop is ready")
				return nil
			}
			dots++
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "  Waiting for Docker daemon to be ready%s\r", strings.Repeat(".", dots%4+1)+"   ")
			}
		}
	}
}

// checkDuplicate returns an error if an environment with the given name
// already exists (i.e., the compose project is listed by docker compose ls).
func checkDuplicate(ctx context.Context, name string) error {
	project := "alpine-" + name
	// docker compose ls lists known projects. We filter by exact name and check
	// whether the result is non-empty. This is reliable across Docker Compose
	// versions (unlike "docker compose ps" which may return exit 0 with empty
	// output for non-existent projects).
	stdout, _, err := run(ctx, "docker", "compose", "ls", "--filter", "name="+project, "--format", "json")
	if err != nil {
		// If ls itself fails, assume no duplicate (create will fail later if
		// Docker is truly broken).
		return nil
	}
	stdout = strings.TrimSpace(stdout)
	if stdout == "" || stdout == "[]" {
		return nil
	}
	return fmt.Errorf("environment %q already exists", name)
}

// composePSEntry is a helper struct for parsing docker compose ps JSON output.
// This represents a single service entry from "docker compose ps --format json".
type composePSEntry struct {
	Name    string `json:"Name"`
	Service string `json:"Service"`
}

// discoverContainer finds the dev container name for an environment by
// running "docker compose ps --format json" and looking for the "dev" service.
func discoverContainer(ctx context.Context, name string) (string, error) {
	project := "alpine-" + name
	stdout, _, err := run(ctx, "docker", "compose", "-p", project, "ps", "--format", "json")
	if err != nil {
		return "", fmt.Errorf("environment %q not found. Run `alpine list` to see active environments", name)
	}

	// docker compose ps --format json outputs one JSON object per line.
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry composePSEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			slog.Debug("skipping unparseable compose ps line", "line", line, "err", err)
			continue
		}
		if entry.Service == "dev" {
			return entry.Name, nil
		}
	}

	return "", fmt.Errorf("dev container not found for environment %q", name)
}

// composeUp runs "docker compose up -d --wait" with the given compose file
// and a timeout.
func composeUp(ctx context.Context, name string, composeFile string) error {
	project := "alpine-" + name
	upCtx, cancel := context.WithTimeout(ctx, timeoutComposeUp)
	defer cancel()

	slog.Info("starting environment", "name", name)
	_, stderr, err := run(upCtx, "docker", "compose", "-p", project, "-f", composeFile, "up", "-d", "--wait")
	if err != nil {
		return fmt.Errorf("failed to start environment %q: %s", name, stderr)
	}
	return nil
}

// composeDown tears down an environment with -t 1 for fast stop.
func composeDown(ctx context.Context, name string) error {
	project := "alpine-" + name
	slog.Info("tearing down environment", "name", name)
	_, stderr, err := run(ctx, "docker", "compose", "-p", project, "down", "-v", "--remove-orphans", "-t", "1")
	if err != nil {
		return fmt.Errorf("failed to tear down environment %q: %s", name, stderr)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Compose/Dockerfile generation
// ---------------------------------------------------------------------------

// composeTemplate is the Go text/template for generating docker-compose.yml.
// Key features:
//   - Dev service uses build: directive for image building
//   - SSH agent mount is platform-aware (macOS vs Linux)
//   - Environment vars use passthrough syntax only (no literal values)
//   - Labels for metadata discovery
//   - Tuned health checks (interval: 2s)
//   - Security hardening: cap_drop ALL, no-new-privileges
//   - tmpfs with size limits for services that need them
const composeTemplate = `services:
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
`

// serviceTemplate generates the YAML block for a supporting service.
const serviceTemplate = `  {{ .Alias }}:
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
`

// serviceAlias maps service names to their compose service aliases.
var serviceAlias = map[string]string{
	"postgres": "db",
	"redis":    "cache",
}

// serviceTemplateData holds data for rendering a service block.
type serviceTemplateData struct {
	Alias           string
	Image           string
	ExtraCmd        string
	Tmpfs           string
	Healthcheck     string
	UseCMDShell     bool
	HealthcheckParts string
	StartPeriod     string
	EnvName         string
}

// composeData holds all the data needed to render the compose template.
type composeData struct {
	ImageTag string
	Project  string
	Name     string
	SSHSocket string
	SSHTarget string
	Created  string
	Branch   string
	Services []string
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
			return nil, fmt.Errorf("unsupported service: %q (supported: postgres, redis)", svc)
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

		// Determine health check format.
		// postgres uses CMD-SHELL (pg_isready -U postgres), redis uses CMD with split parts.
		if svc == "postgres" {
			data.UseCMDShell = true
			data.StartPeriod = "5s"
		} else {
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

		tmpl, err := template.New("service").Parse(serviceTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to parse service template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to render service %q: %w", svc, err)
		}
		serviceBlocks = append(serviceBlocks, buf.String())
	}

	data := composeData{
		ImageTag: imageTag,
		Project:  project,
		Name:         name,
		SSHSocket:    sshSocket,
		SSHTarget:    sshTarget,
		Created:      created,
		Branch:       branch,
		Services:     serviceBlocks,
	}

	tmpl, err := template.New("compose").Parse(composeTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render compose YAML: %w", err)
	}

	return buf.Bytes(), nil
}

// generateDockerfile produces a Dockerfile that layers Claude CLI and git
// on top of the provided base image. It creates a non-root "claude" user
// and hardcodes apt-get with --no-install-recommends.
func generateDockerfile(baseImage string) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "FROM %s\n", baseImage)
	buf.WriteString(`
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl openssh-client ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash claude

# Pre-populate GitHub SSH host keys so git clone over SSH does not prompt.
RUN mkdir -p /home/claude/.ssh \
    && ssh-keyscan -t ed25519,rsa github.com >> /home/claude/.ssh/known_hosts 2>/dev/null \
    && chown -R claude:claude /home/claude/.ssh \
    && chmod 700 /home/claude/.ssh \
    && chmod 600 /home/claude/.ssh/known_hosts

ENV PATH="/home/claude/.local/bin:${PATH}"

# Configure git to use GITHUB_TOKEN for HTTPS auth when available.
USER claude
RUN git config --global credential.helper \
    '!f() { echo "username=x-access-token"; echo "password=${GITHUB_TOKEN:-${GH_TOKEN}}"; }; f'

RUN bash -c 'set -o pipefail && curl -fsSL https://claude.ai/install.sh | bash'
USER root
RUN ln -s /home/claude/.local/bin/claude /usr/local/bin/claude

WORKDIR /workspace
RUN chown claude:claude /workspace
USER claude
`)
	return buf.Bytes()
}

// dockerfileHash returns the first 16 characters of the SHA-256 hash of
// the Dockerfile content. Used as a Docker image tag so identical
// Dockerfiles share a single image.
func dockerfileHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)[:16]
}

// imageExists checks if a Docker image with the given tag exists locally.
func imageExists(ctx context.Context, tag string) (bool, error) {
	_, _, err := run(ctx, "docker", "image", "inspect", tag)
	if err != nil {
		// Check if the context was cancelled (Docker unreachable, timeout, etc.)
		if ctx.Err() != nil {
			return false, fmt.Errorf("checking image: %w", ctx.Err())
		}
		// docker image inspect exits non-zero if image does not exist.
		return false, nil
	}
	return true, nil
}

// ---------------------------------------------------------------------------
// Environment helpers
// ---------------------------------------------------------------------------

// loadDotEnv reads a .env file and sets variables in the process environment.
// Variables already set in the environment are NOT overwritten, so explicit
// exports always take precedence. This ensures compose passthrough variables
// (e.g., CLAUDE_CODE_OAUTH_TOKEN) pick up values from .env files.
func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip optional "export " prefix.
		line = strings.TrimPrefix(line, "export ")

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Remove surrounding quotes.
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already in environment.
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
			slog.Debug("loaded env var from .env", "key", key)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// File copy helpers
// ---------------------------------------------------------------------------

// copyPathToContainer copies a file or directory from the host into the container
// and chowns it to the claude user. Works for both files and directories.
func copyPathToContainer(ctx context.Context, container, srcPath, destPath string) error {
	// Use docker cp (handles both files and directories)
	_, stderr, err := run(ctx, "docker", "cp", srcPath, container+":"+destPath)
	if err != nil {
		return fmt.Errorf("docker cp failed: %s", stderr)
	}

	// chown to claude user (use -R for directories)
	_, stderr, err = run(ctx, "docker", "exec", "--user", "root", container, "chown", "-R", "claude:claude", destPath)
	if err != nil {
		return fmt.Errorf("chown failed: %s", stderr)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Git operations (all via docker exec into container)
// ---------------------------------------------------------------------------

// gitClone clones a repo inside a container. Uses a full clone so the
// container has enough history for rebases, amends, and pushes.
func gitClone(ctx context.Context, container, remoteURL, branch string) error {
	cloneCtx, cancel := context.WithTimeout(ctx, timeoutGitClone)
	defer cancel()

	slog.Info("cloning repository", "container", container, "branch", branch)
	_, stderr, err := run(cloneCtx, "docker", "exec", container,
		"git", "clone", "--branch", branch, remoteURL, "/workspace")
	if err != nil {
		return fmt.Errorf("git clone failed: %s", stderr)
	}
	return nil
}

// gitCreateBranch creates and checks out a new branch inside a container.
// Uses the -- separator before the branch name to prevent flag injection.
func gitCreateBranch(ctx context.Context, container, name string) error {
	_, stderr, err := run(ctx, "docker", "exec", "-w", "/workspace", container,
		"git", "checkout", "-b", "feature/"+name)
	if err != nil {
		return fmt.Errorf("git checkout -b failed: %s", stderr)
	}
	return nil
}

// gitConfigureUser sets git user.name and user.email inside the container
// by reading the values from the host's git configuration.
func gitConfigureUser(ctx context.Context, container string) error {
	// Read host git config.
	hostName, _, err := run(ctx, "git", "config", "user.name")
	if err != nil {
		hostName = "alpine"
	}
	hostEmail, _, err := run(ctx, "git", "config", "user.email")
	if err != nil {
		hostEmail = "alpine@localhost"
	}

	_, _, err = run(ctx, "docker", "exec", "-w", "/workspace", container,
		"git", "config", "user.name", hostName)
	if err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	_, _, err = run(ctx, "docker", "exec", "-w", "/workspace", container,
		"git", "config", "user.email", hostEmail)
	if err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	return nil
}

// gitHasChanges returns true if there are uncommitted changes in the
// container's working tree.
func gitHasChanges(ctx context.Context, container string) (bool, error) {
	stdout, _, err := run(ctx, "docker", "exec", "-w", "/workspace", container,
		"git", "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return stdout != "", nil
}

// gitAddCommitPush stages tracked files (git add -u, NEVER git add .),
// commits with the given message, and pushes to the remote branch.
func gitAddCommitPush(ctx context.Context, container, branch, message string) error {
	// Stage tracked files only.
	_, _, err := run(ctx, "docker", "exec", "-w", "/workspace", container,
		"git", "add", "-u")
	if err != nil {
		return fmt.Errorf("git add -u failed: %w", err)
	}

	// Commit.
	_, _, err = run(ctx, "docker", "exec", "-w", "/workspace", container,
		"git", "commit", "-m", message)
	if err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	// Push with timeout.
	pushCtx, cancel := context.WithTimeout(ctx, timeoutGitPush)
	defer cancel()

	_, stderr, err := run(pushCtx, "docker", "exec", "-w", "/workspace", container,
		"git", "push", "origin", branch)
	if err != nil {
		return fmt.Errorf("git push failed: %s", stderr)
	}
	return nil
}

// gitGetCurrentBranch returns the current branch name on the host.
func gitGetCurrentBranch(ctx context.Context) (string, error) {
	stdout, _, err := run(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to determine current branch: %w", err)
	}
	if stdout == "HEAD" {
		return "", fmt.Errorf("HEAD is detached. Use --from <branch> to specify a base branch")
	}
	return stdout, nil
}

// gitGetRemoteURL returns the URL for the given remote (e.g., "origin").
func gitGetRemoteURL(ctx context.Context, remote string) (string, error) {
	stdout, _, err := run(ctx, "git", "remote", "get-url", remote)
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL for %q: %w", remote, err)
	}
	return stdout, nil
}

// gitFindRoot returns the repository root by running git rev-parse.
// This handles worktrees, $GIT_DIR overrides, and .git files correctly.
func gitFindRoot() (string, error) {
	stdout, _, err := run(context.Background(), "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return stdout, nil
}


// ---------------------------------------------------------------------------
// Output helpers
// ---------------------------------------------------------------------------

// outputJSON marshals v to JSON and writes it to stdout. Used by all commands
// when the --json flag is set.
func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// outputError writes a structured error to stderr. When --json is set, it
// outputs a JSON object with "error" and "exit_code" fields. Otherwise it
// prints a plain text error message.
func outputError(msg string, exitCode int) {
	if jsonOutput {
		_ = outputJSON(map[string]interface{}{
			"error":     msg,
			"exit_code": exitCode,
		})
		return
	}
	fmt.Fprintln(os.Stderr, "Error: "+msg)
}


