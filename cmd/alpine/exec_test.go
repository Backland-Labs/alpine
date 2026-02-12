package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"testing"
)

func TestExecError(t *testing.T) {
	t.Run("with stderr", func(t *testing.T) {
		e := &ExecError{Command: "docker build", Stderr: "no space left on device", Err: fmt.Errorf("exit 1")}
		msg := e.Error()
		if !strings.Contains(msg, "docker build") {
			t.Errorf("Error() missing command, got %q", msg)
		}
		if !strings.Contains(msg, "no space left on device") {
			t.Errorf("Error() missing stderr, got %q", msg)
		}
		if !strings.Contains(msg, "stderr:") {
			t.Errorf("Error() missing 'stderr:' prefix, got %q", msg)
		}
	})

	t.Run("without stderr", func(t *testing.T) {
		e := &ExecError{Command: "docker build", Err: fmt.Errorf("exit 1")}
		msg := e.Error()
		if !strings.Contains(msg, "docker build") {
			t.Errorf("Error() missing command, got %q", msg)
		}
		if strings.Contains(msg, "stderr") {
			t.Errorf("Error() should not mention stderr when empty, got %q", msg)
		}
	})

	t.Run("unwrap returns inner error", func(t *testing.T) {
		inner := fmt.Errorf("inner error")
		e := &ExecError{Command: "cmd", Err: inner}
		if e.Unwrap() != inner {
			t.Fatal("Unwrap() did not return inner error")
		}
	})
}

func TestDefaultRun_Success(t *testing.T) {
	stdout, stderr, err := defaultRun(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "hello" {
		t.Errorf("stdout = %q, want %q", stdout, "hello")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestDefaultRun_Error(t *testing.T) {
	_, _, err := defaultRun(context.Background(), "false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
	var execErr *ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected *ExecError, got %T", err)
	}
	if !strings.HasPrefix(execErr.Command, "false") {
		t.Errorf("Command = %q, want prefix %q", execErr.Command, "false")
	}
}

func TestDefaultRun_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := defaultRun(ctx, "sleep", "10")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestDefaultRunInteractive_Success(t *testing.T) {
	// Restore SIGINT handling after test (defaultRunInteractive calls signal.Ignore).
	t.Cleanup(func() { signal.Reset(os.Interrupt) })

	err := defaultRunInteractive("true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultRunInteractive_Error(t *testing.T) {
	t.Cleanup(func() { signal.Reset(os.Interrupt) })

	err := defaultRunInteractive("false")
	if err == nil {
		t.Fatal("expected error from 'false' command")
	}
	var execErr *ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected *ExecError, got %T", err)
	}
}
