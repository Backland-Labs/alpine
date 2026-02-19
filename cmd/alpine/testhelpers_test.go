package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type cmdResult struct {
	stdout, stderr string
	err            error
}

func mockRun(t *testing.T, responses []cmdResult) {
	t.Helper()
	orig := run
	i := 0
	run = func(_ context.Context, _ string, _ ...string) (string, string, error) {
		if i >= len(responses) {
			t.Fatalf("unexpected call to run() (call %d, only %d responses registered)", i+1, len(responses))
		}
		r := responses[i]
		i++
		return r.stdout, r.stderr, r.err
	}
	t.Cleanup(func() {
		run = orig
		if i < len(responses) {
			t.Errorf("mockRun: %d of %d responses consumed", i, len(responses))
		}
	})
}

type cmdCall struct {
	name string
	args []string
}

func mockRunRecording(t *testing.T, responses []cmdResult) *[]cmdCall {
	t.Helper()
	orig := run
	var calls []cmdCall
	i := 0
	run = func(_ context.Context, name string, args ...string) (string, string, error) {
		calls = append(calls, cmdCall{name: name, args: args})
		if i >= len(responses) {
			t.Fatalf("unexpected call to run(): %s %v (call %d, only %d responses)", name, args, i+1, len(responses))
		}
		r := responses[i]
		i++
		return r.stdout, r.stderr, r.err
	}
	t.Cleanup(func() {
		run = orig
		if i < len(responses) {
			t.Errorf("mockRunRecording: %d of %d responses consumed", i, len(responses))
		}
	})
	return &calls
}

func mockOpenBrowser(t *testing.T, opened bool) {
	t.Helper()
	orig := openBrowser
	openBrowser = func(_ context.Context, _ string, _ string) bool { return opened }
	t.Cleanup(func() { openBrowser = orig })
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	outCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r) //nolint:errcheck
		outCh <- buf.String()
	}()

	fn()

	w.Close() //nolint:errcheck
	os.Stdout = origStdout
	return <-outCh
}

func captureOutputs(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}

	origStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outCh := make(chan string)
	errCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rOut) //nolint:errcheck
		outCh <- buf.String()
	}()
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rErr) //nolint:errcheck
		errCh <- buf.String()
	}()

	fn()

	wOut.Close() //nolint:errcheck
	wErr.Close() //nolint:errcheck
	os.Stdout = origStdout
	os.Stderr = origStderr

	return <-outCh, <-errCh
}

func resetFlags(t *testing.T) {
	t.Helper()
	origJSON := jsonOutput
	origVerbose := verbose
	origListAllOwners := listAllOwners
	origDownRetryPersist := downRetryPersist
	origUpBranch := upBranch
	origUpClientRequestID := upClientRequestID
	origUpPublicRepo := upPublicRepo
	origUpNoBrowser := upNoBrowser
	t.Cleanup(func() {
		jsonOutput = origJSON
		verbose = origVerbose
		listAllOwners = origListAllOwners
		downRetryPersist = origDownRetryPersist
		upBranch = origUpBranch
		upClientRequestID = origUpClientRequestID
		upPublicRepo = origUpPublicRepo
		upNoBrowser = origUpNoBrowser
	})
}

func errResult(stderr string) cmdResult {
	return cmdResult{stderr: stderr, err: fmt.Errorf("exit status 1")}
}

func setupLedgerPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sessions.json")
	t.Setenv("ALPINE_LEDGER_PATH", path)
	return path
}
