package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// dockerHealthCheck verifies the Docker daemon is running by executing
// "docker info" with a 3-second timeout. On macOS, if Docker is not
// running it attempts to launch Docker Desktop and waits up to 60s
// for the daemon to become ready.
func dockerHealthCheck(ctx context.Context, platform string) error {
	hctx, cancel := context.WithTimeout(ctx, timeoutDockerHealth)
	defer cancel()

	_, stderr, err := run(hctx, "docker", "info")
	if err == nil {
		return nil
	}

	// Docker is not running. On macOS, try to start Docker Desktop.
	if platform != "darwin" {
		return fmt.Errorf("docker is not running; start Docker and try again (detail: %s)", stderr)
	}

	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "Docker is not running. Starting Docker Desktop...\n")
	}
	slog.Info("Docker is not running, starting Docker Desktop")
	if _, _, launchErr := run(ctx, "open", "-a", "Docker"); launchErr != nil {
		return fmt.Errorf("docker is not running and failed to start Docker Desktop (detail: %s)", stderr)
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
			return fmt.Errorf("docker Desktop started but daemon did not become ready within %s", pollTimeout)
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
	stdout, _, err := run(ctx, "docker", "compose", "ls", "--filter", "name="+project, "--format", "json")
	if err != nil {
		return nil
	}
	stdout = strings.TrimSpace(stdout)
	if stdout == "" || stdout == "[]" {
		return nil
	}
	return fmt.Errorf("environment %q already exists", name)
}

// composePSEntry is a helper struct for parsing docker compose ps JSON output.
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

// imageExists checks if a Docker image with the given tag exists locally.
func imageExists(ctx context.Context, tag string) (bool, error) {
	_, _, err := run(ctx, "docker", "image", "inspect", tag)
	if err != nil {
		if ctx.Err() != nil {
			return false, fmt.Errorf("checking image: %w", ctx.Err())
		}
		return false, nil
	}
	return true, nil
}
