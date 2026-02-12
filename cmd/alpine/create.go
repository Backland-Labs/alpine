package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	fromBranch string
	detach     bool
)

// nameRegex validates environment names: 1-50 lowercase alphanumeric chars or
// hyphens, no leading/trailing hyphens.
var nameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,48}[a-z0-9])?$`)

func validateName(name string) error {
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("invalid name %q: must be 1-50 lowercase alphanumeric chars or hyphens, no leading/trailing hyphens", name)
	}
	return nil
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an isolated dev environment",
	Long:  "Create a fully isolated dev environment with its own repo clone, branch, services, and Claude Code instance.",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("requires a name argument\n\nUsage: alpine create <name>\n\nExample:\n  alpine create my-feature")
		}
		if len(args) > 1 {
			return fmt.Errorf("accepts 1 argument, received %d\n\nUsage: alpine create <name>", len(args))
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVar(&fromBranch, "from", "", "base branch (default: current branch)")
	createCmd.Flags().BoolVarP(&detach, "detach", "d", false, "return immediately after environment is ready")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) (err error) {
	ctx := cmd.Context()
	name := args[0]
	project := "alpine-" + name

	// ---------------------------------------------------------------
	// Step 1: Validate name
	// ---------------------------------------------------------------
	slog.Debug("validating environment name", "name", name)
	if err := validateName(name); err != nil {
		return userErr(err.Error())
	}

	// ---------------------------------------------------------------
	// Step 2: Docker health check
	// ---------------------------------------------------------------
	slog.Debug("checking Docker health")
	if err := dockerHealthCheck(ctx, runtime.GOOS); err != nil {
		return sysErr(err.Error())
	}

	// ---------------------------------------------------------------
	// Step 3: Find git root
	// ---------------------------------------------------------------
	slog.Debug("finding git repository root")
	gitRoot, err := gitFindRoot()
	if err != nil {
		return userErr("not inside a git repository")
	}
	slog.Debug("git root found", "path", gitRoot)

	// Load .env from the git root so compose passthrough variables
	// (CLAUDE_CODE_OAUTH_TOKEN, ANTHROPIC_API_KEY, etc.) are available
	// even when the user hasn't exported them in their shell.
	envPath := filepath.Join(gitRoot, ".env")
	if _, statErr := os.Stat(envPath); statErr == nil {
		if loadErr := loadDotEnv(envPath); loadErr != nil {
			slog.Debug("failed to load .env", "error", loadErr)
		} else {
			slog.Debug("loaded .env from git root", "path", envPath)
		}
	}

	// ---------------------------------------------------------------
	// Step 4: Check for duplicate environment
	// ---------------------------------------------------------------
	slog.Debug("checking for duplicate environment", "name", name)
	if err := checkDuplicate(ctx, name); err != nil {
		return userErr(fmt.Sprintf("environment %q already exists. Run `alpine list` to see active environments", name))
	}

	// ---------------------------------------------------------------
	// Step 5: Determine base branch
	// ---------------------------------------------------------------
	var branch string
	if fromBranch != "" {
		// Reject values starting with "-" to prevent flag injection.
		if strings.HasPrefix(fromBranch, "-") {
			return userErr(fmt.Sprintf("invalid branch name %q: must not start with '-'", fromBranch))
		}
		branch = fromBranch
		slog.Debug("using --from branch", "branch", branch)
	} else {
		branch, err = gitGetCurrentBranch(ctx)
		if err != nil {
			return userErr("HEAD is detached. Use `--from <branch>` to specify a base branch")
		}
		slog.Debug("using current branch", "branch", branch)
	}

	// ---------------------------------------------------------------
	// Step 6: Validate git auth prerequisites
	// ---------------------------------------------------------------
	slog.Debug("validating git auth prerequisites")
	hasGitAuth := false
	if os.Getenv("SSH_AUTH_SOCK") != "" {
		hasGitAuth = true
		slog.Debug("git auth via SSH agent")
	}
	if os.Getenv("GITHUB_TOKEN") != "" || os.Getenv("GH_TOKEN") != "" {
		hasGitAuth = true
		slog.Debug("git auth via GITHUB_TOKEN/GH_TOKEN")
	}
	if !hasGitAuth {
		return userErr("no git auth found. Set up SSH keys (SSH_AUTH_SOCK) or export GITHUB_TOKEN")
	}

	// ---------------------------------------------------------------
	// Step 7: Generate Dockerfile, compute hash, build if needed
	// ---------------------------------------------------------------
	slog.Debug("loading configuration")
	cfg, err := loadConfig()
	if err != nil {
		return userErr(fmt.Sprintf("failed to load config: %v", err))
	}

	dockerfile := generateDockerfile(cfg.BaseImage)
	hash := dockerfileHash(dockerfile)
	imageTag := "alpine-dev:" + hash
	slog.Debug("Dockerfile generated", "tag", imageTag)

	exists, err := imageExists(ctx, imageTag)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to check image: %v", err))
	}

	if exists {
		slog.Debug("image already exists, skipping build", "tag", imageTag)
	} else {
		slog.Debug("building image", "tag", imageTag)
		tempBuildDir, err := os.MkdirTemp("", "alpine-build-*")
		if err != nil {
			return sysErr(fmt.Sprintf("failed to create temp dir: %v", err))
		}
		defer os.RemoveAll(tempBuildDir) //nolint:errcheck // best-effort cleanup

		dockerfilePath := filepath.Join(tempBuildDir, "Dockerfile")
		if err := os.WriteFile(dockerfilePath, dockerfile, 0644); err != nil {
			return sysErr(fmt.Sprintf("failed to write Dockerfile: %v", err))
		}

		buildCtx, buildCancel := context.WithTimeout(ctx, 5*time.Minute)
		defer buildCancel()

		_, stderr, err := run(buildCtx, "docker", "build", "-t", imageTag, tempBuildDir)
		if err != nil {
			return sysErr(fmt.Sprintf("Docker image build failed: %s", stderr))
		}
		slog.Debug("image built successfully", "tag", imageTag)
	}

	// ---------------------------------------------------------------
	// Step 8: Generate compose YAML, write to temp dir, compose up
	// ---------------------------------------------------------------
	composeYAML, err := generateComposeYAML(cfg, name, branch, runtime.GOOS, imageTag)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to generate compose YAML: %v", err))
	}

	tempDir, err := os.MkdirTemp("", "alpine-compose-*")
	if err != nil {
		return sysErr(fmt.Sprintf("failed to create temp dir: %v", err))
	}

	// ---------------------------------------------------------------
	// Step 9: defer cleanup of temp dir
	// ---------------------------------------------------------------
	defer os.RemoveAll(tempDir) //nolint:errcheck // best-effort cleanup

	composeFile := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, composeYAML, 0644); err != nil {
		return sysErr(fmt.Sprintf("failed to write compose file: %v", err))
	}
	slog.Debug("compose file written", "path", composeFile)

	// Rollback defer: tear down compose project if we started it and hit an error.
	// Uses context.Background() so cleanup works even after Ctrl+C cancels ctx.
	// Skip rollback for exit code 3 (install failure) -- leave env running so
	// the user can attach and fix.
	composed := false
	defer func() {
		if err != nil && composed {
			if ee, ok := err.(*exitError); ok && ee.code == 3 {
				return
			}
			slog.Debug("rolling back: tearing down compose project", "project", project)
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cleanupCancel()
			_ = composeDown(cleanupCtx, name)
		}
	}()

	slog.Debug("starting compose project", "project", project)
	if err = composeUp(ctx, name, composeFile); err != nil {
		return sysErr(fmt.Sprintf("failed to start environment: %v", err))
	}
	composed = true
	slog.Debug("compose project started", "project", project)

	// Discover the dev container name.
	container, err := discoverContainer(ctx, name)
	if err != nil {
		return sysErr(fmt.Sprintf("failed to discover dev container: %v", err))
	}
	slog.Debug("dev container discovered", "container", container)

	// ---------------------------------------------------------------
	// Step 10: Get remote URL
	// ---------------------------------------------------------------
	slog.Debug("resolving git remote URL")
	remoteURL, err := gitGetRemoteURL(ctx, "origin")
	if err != nil {
		return userErr("no git remote 'origin' configured. Add a remote and try again")
	}
	slog.Debug("remote URL resolved", "url", remoteURL)

	// ---------------------------------------------------------------
	// Step 11: Clone repo inside container
	// ---------------------------------------------------------------
	slog.Debug("cloning repository into container", "remote", remoteURL, "branch", branch)
	if err = gitClone(ctx, container, remoteURL, branch); err != nil {
		return sysErr(fmt.Sprintf("failed to clone repository: %v", err))
	}

	// ---------------------------------------------------------------
	// Step 12: Create feature/<name> branch inside container
	// ---------------------------------------------------------------
	featureBranch := "feature/" + name
	slog.Debug("creating feature branch", "branch", featureBranch)
	if err = gitCreateBranch(ctx, container, name); err != nil {
		return sysErr(fmt.Sprintf("failed to create branch: %v", err))
	}

	// ---------------------------------------------------------------
	// Step 13: Configure git user
	// ---------------------------------------------------------------
	slog.Debug("configuring git user in container")
	if err = gitConfigureUser(ctx, container); err != nil {
		return sysErr(fmt.Sprintf("failed to configure git user: %v", err))
	}

	// ---------------------------------------------------------------
	// Step 14: Copy host ~/.claude and ~/.claude.json into container
	// ---------------------------------------------------------------
	if homeDir, homeErr := os.UserHomeDir(); homeErr == nil {
		hostClaudeDir := filepath.Join(homeDir, ".claude")
		if _, statErr := os.Stat(hostClaudeDir); statErr == nil {
			slog.Debug("copying host ~/.claude into container home", "src", hostClaudeDir)
			if cpErr := copyPathToContainer(ctx, container, hostClaudeDir, "/home/claude/.claude"); cpErr != nil {
				slog.Debug("failed to copy host ~/.claude", "error", cpErr)
				if !jsonOutput {
					fmt.Fprintf(os.Stderr, "warning: failed to copy ~/.claude: %v\n", cpErr)
				}
			} else {
				slog.Debug("copied host ~/.claude into container home")
			}
		}

		// Copy ~/.claude.json (onboarding flags, theme, preferences) so
		// Claude Code skips the interactive setup flow in the container.
		hostClaudeJSON := filepath.Join(homeDir, ".claude.json")
		if _, statErr := os.Stat(hostClaudeJSON); statErr == nil {
			slog.Debug("copying host ~/.claude.json into container home", "src", hostClaudeJSON)
			if cpErr := copyPathToContainer(ctx, container, hostClaudeJSON, "/home/claude/.claude.json"); cpErr != nil {
				slog.Debug("failed to copy host ~/.claude.json", "error", cpErr)
			} else {
				slog.Debug("copied host ~/.claude.json into container home")
			}
		}
	}

	// ---------------------------------------------------------------
	// Step 15: Auto-detect and copy common config files into container
	// ---------------------------------------------------------------
	autoConfigPaths := []string{".claude", ".env", ".tool-versions", ".node-version", ".ruby-version", ".python-version"}
	copiedPaths := make(map[string]bool)

	for _, configPath := range autoConfigPaths {
		srcPath := filepath.Join(gitRoot, configPath)
		if _, statErr := os.Stat(srcPath); statErr != nil {
			// File/directory does not exist -- skip silently.
			continue
		}

		destPath := "/workspace/" + configPath
		slog.Debug("auto-copying config into container", "src", srcPath, "dest", destPath)

		if cpErr := copyPathToContainer(ctx, container, srcPath, destPath); cpErr != nil {
			slog.Debug("failed to auto-copy config", "path", configPath, "error", cpErr)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "warning: failed to copy %q: %v\n", configPath, cpErr)
			}
			continue
		}

		copiedPaths[configPath] = true
		slog.Debug("auto-copied config into container", "path", configPath)
	}

	// ---------------------------------------------------------------
	// Step 16: Copy env files into container
	// ---------------------------------------------------------------
	for _, envFile := range cfg.EnvFiles {
		// Skip if already copied in step 14.
		if copiedPaths[envFile] {
			slog.Debug("env file already copied in auto-detect, skipping", "file", envFile)
			continue
		}

		srcPath := filepath.Join(gitRoot, envFile)
		absPath, _ := filepath.Abs(srcPath)
		if !strings.HasPrefix(absPath, gitRoot+string(filepath.Separator)) && absPath != gitRoot {
			slog.Debug("env file escapes git root, skipping", "file", envFile)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "warning: env file %q escapes repository root, skipping\n", envFile)
			}
			continue
		}
		if _, statErr := os.Stat(srcPath); os.IsNotExist(statErr) {
			slog.Debug("env file not found, skipping", "file", envFile)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "warning: env file %q not found, skipping\n", envFile)
			}
			continue
		}

		destPath := "/workspace/" + envFile
		slog.Debug("copying env file into container", "src", srcPath, "dest", destPath)

		if cpErr := copyPathToContainer(ctx, container, srcPath, destPath); cpErr != nil {
			slog.Debug("failed to copy env file", "file", envFile, "error", cpErr)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "warning: failed to copy env file %q: %v\n", envFile, cpErr)
			}
			continue
		}

		copiedPaths[envFile] = true
	}

	// ---------------------------------------------------------------
	// Step 17: Run install command if configured
	// ---------------------------------------------------------------
	installFailed := false
	if cfg.Install != "" {
		slog.Debug("running install command", "command", cfg.Install)
		installCtx, installCancel := context.WithTimeout(ctx, 10*time.Minute)
		defer installCancel()

		_, stderr, installErr := run(installCtx, "docker", "exec", container, "sh", "-c", "cd /workspace && "+cfg.Install)
		if installErr != nil {
			installFailed = true
			slog.Debug("install command failed", "error", stderr)
			if !jsonOutput {
				fmt.Fprintf(os.Stderr, "warning: install command failed: %s\n", stderr)
				fmt.Fprintf(os.Stderr, "Environment is running. Use `alpine attach %s` to inspect and fix.\n", name)
			}
			// Do NOT return error -- leave env running so user can attach and fix.
			// Set exit code 3 (partial success) handled below.
		} else {
			slog.Debug("install command completed successfully")
		}
	}

	// ---------------------------------------------------------------
	// Step 18: Ensure Claude Code auth is configured inside container
	// ---------------------------------------------------------------
	// CLAUDE_CODE_OAUTH_TOKEN is passed through via the compose environment.
	// Claude Code also requires hasCompletedOnboarding=true in ~/.claude.json
	// to skip the interactive setup wizard (theme picker, auth prompt).
	// See: https://github.com/anthropics/claude-code/issues/8938
	slog.Debug("configuring Claude Code auth in container")

	// Ensure ~/.claude.json exists with hasCompletedOnboarding=true.
	// If the file was already copied from the host (Step 14), it almost certainly
	// has the flag. If not, we create a minimal file.
	_, _, setupErr := run(ctx, "docker", "exec", container,
		"sh", "-c", `f=/home/claude/.claude.json
if [ -f "$f" ]; then
  grep -q '"hasCompletedOnboarding"' "$f" || \
    sed -i '1s/^{/{"hasCompletedOnboarding":true,/' "$f"
else
  echo '{"hasCompletedOnboarding":true}' > "$f"
fi`)
	if setupErr != nil {
		slog.Debug("claude onboarding setup failed (non-fatal)", "error", setupErr)
	}

	// Verify the OAuth token is available in the container environment.
	tokenCheck, _, _ := run(ctx, "docker", "exec", container,
		"sh", "-c", `[ -n "$CLAUDE_CODE_OAUTH_TOKEN" ] && echo "set" || echo "unset"`)
	if tokenCheck == "set" {
		slog.Debug("CLAUDE_CODE_OAUTH_TOKEN is set in container")
	} else {
		slog.Debug("CLAUDE_CODE_OAUTH_TOKEN is NOT set in container")
		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "warning: CLAUDE_CODE_OAUTH_TOKEN is not set. Run `claude setup-token` on the host and export the token to enable auto-auth.\n")
		}
	}

	// ---------------------------------------------------------------
	// Step 19: Detach or attach
	// ---------------------------------------------------------------
	if detach || jsonOutput {
		// Output status JSON and return.
		status := map[string]interface{}{
			"name":      name,
			"branch":    featureBranch,
			"status":    "running",
			"container": container,
			"project":   project,
		}
		if installFailed {
			status["install_status"] = "failed"
			status["warning"] = "install command failed; environment is running"
		} else if cfg.Install != "" {
			status["install_status"] = "success"
		}

		if jsonOutput {
			if err := outputJSON(status); err != nil {
				return err
			}
		} else {
			// Pretty print for non-JSON detach mode.
			data, _ := json.MarshalIndent(status, "", "  ")
			fmt.Println(string(data))
		}

		if installFailed {
			return &exitError{msg: "install command failed; environment is running", code: 3}
		}
		return nil
	}

	// Interactive mode: drop into a shell inside the container.
	slog.Debug("attaching to container shell", "container", container)
	shellErr := runInteractive("docker", "exec", "-it", "--user", "claude", "-w", "/workspace", container, "/bin/bash")
	if shellErr != nil {
		slog.Debug("shell session ended", "error", shellErr)
	}

	if installFailed {
		return &exitError{msg: "install command failed; environment is running", code: 3}
	}

	// Clear the named return so rollback defer does not fire.
	err = nil
	return nil
}
