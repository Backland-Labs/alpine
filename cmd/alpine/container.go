package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

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

// inspectContainer runs docker inspect with the given Go template format string
// and returns the trimmed output.
func inspectContainer(ctx context.Context, container, format string) (string, error) {
	stdout, _, err := run(ctx, "docker", "inspect", "--format", format, container)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout), nil
}

// checkClaudeProcess checks whether a Claude process is running inside the
// container. If the container is not running, it returns (false, nil). If
// Claude has exited, it attempts to read the exit code from
// /tmp/claude-exit-code inside the container.
func checkClaudeProcess(ctx context.Context, container, containerState string) (bool, *int) {
	if containerState != "running" {
		return false, nil
	}

	// pgrep -f claude exits 0 if a matching process is found, non-zero otherwise.
	_, _, err := run(ctx, "docker", "exec", container, "pgrep", "-f", "claude")
	if err == nil {
		// Claude is running
		return true, nil
	}

	// Claude is not running -- try to read the exit code file
	stdout, _, err := run(ctx, "docker", "exec", container, "sh", "-c", "cat /tmp/claude-exit-code 2>/dev/null")
	if err == nil {
		stdout = strings.TrimSpace(stdout)
		if code, parseErr := strconv.Atoi(stdout); parseErr == nil {
			return false, &code
		}
	}

	// Claude exited but no exit code file found
	return false, nil
}
