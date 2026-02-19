package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
)

// Timeouts for external commands.
const (
	timeoutDockerHealth = 3 * time.Second
	timeoutImageBuild   = 5 * time.Minute
	timeoutComposeUp    = 2 * time.Minute
	timeoutGitClone     = 5 * time.Minute
	timeoutInstall      = 10 * time.Minute
	timeoutGitPush      = 30 * time.Second
)

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

// Package-level function variables -- tests swap these via t.Cleanup.
var (
	run = defaultRun
)

// defaultRun executes a command and returns stdout and stderr as strings.
// It uses exec.CommandContext with the provided context for timeout support.
// Every external command invocation must go through this function.
// Never use "sh -c" -- always pass arguments directly to prevent shell injection.
func defaultRun(ctx context.Context, name string, args ...string) (string, string, error) {
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

// defaultRunInteractive executes an interactive command with stdin/stdout/stderr
// connected to the terminal. Unlike runAttached, it ignores SIGINT in the
// Go process so Ctrl-C is handled only by the child. This prevents the
// parent from killing the child when the user presses Ctrl-C (e.g., to
// exit Claude Code while remaining in a container shell).
func defaultRunInteractive(name string, args ...string) error {
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
