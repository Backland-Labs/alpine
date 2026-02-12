package main

import (
	"context"
	"fmt"
	"log/slog"
)

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
