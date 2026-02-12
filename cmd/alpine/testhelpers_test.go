package main

// Tests mutate package-level variables and must NOT use t.Parallel().

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// cmdResult is a canned response for the mock run function.
type cmdResult struct {
	stdout, stderr string
	err            error
}

// mockRun replaces the package-level run variable with a sequential mock.
// Each call to run() consumes the next response from the slice. Calls beyond
// the slice length cause a test failure.
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
	t.Cleanup(func() { run = orig })
}

// cmdCall records a single invocation of run().
type cmdCall struct {
	name string
	args []string
}

// mockRunRecording replaces run with a sequential mock that also records
// every call for later inspection.
func mockRunRecording(t *testing.T, responses []cmdResult) *[]cmdCall {
	t.Helper()
	orig := run
	var calls []cmdCall
	i := 0
	run = func(_ context.Context, name string, args ...string) (string, string, error) {
		calls = append(calls, cmdCall{name, args})
		if i >= len(responses) {
			t.Fatalf("unexpected call to run(): %s %v (call %d, only %d responses)", name, args, i+1, len(responses))
		}
		r := responses[i]
		i++
		return r.stdout, r.stderr, r.err
	}
	t.Cleanup(func() { run = orig })
	return &calls
}

// mockRunInteractive replaces the package-level runInteractive with a no-op.
func mockRunInteractive(t *testing.T, retErr error) {
	t.Helper()
	orig := runInteractive
	runInteractive = func(_ string, _ ...string) error {
		return retErr
	}
	t.Cleanup(func() { runInteractive = orig })
}

// captureStdout captures everything written to os.Stdout during fn().
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
		io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	fn()

	w.Close()
	os.Stdout = origStdout
	return <-outCh
}

// resetFlags saves and restores Cobra flag globals that tests may modify.
func resetFlags(t *testing.T) {
	t.Helper()
	origJSON := jsonOutput
	origVerbose := verbose
	origFrom := fromBranch
	origDetach := detach
	t.Cleanup(func() {
		jsonOutput = origJSON
		verbose = origVerbose
		fromBranch = origFrom
		detach = origDetach
	})
}

// happyCreateResponses returns the full sequence of run() responses for a
// successful runCreate in detach+json mode. Tests copy this slice and replace
// specific entries to test error paths.
//
// Call sequence (16 calls):
//
//	 0: docker info                          (Step 2: health check)
//	 1: git rev-parse --show-toplevel        (Step 3: find git root)
//	 2: docker compose ls ...                (Step 4: check duplicate)
//	 3: git rev-parse --abbrev-ref HEAD      (Step 5: current branch)
//	 4: docker image inspect ...             (Step 7: image exists)
//	 5: docker compose ... up -d --wait      (Step 8: compose up)
//	 6: docker compose ... ps --format json  (Step 9: discover container)
//	 7: git remote get-url origin            (Step 10: remote URL)
//	 8: docker exec ... git clone ...        (Step 11: clone)
//	 9: docker exec ... git checkout -b ...  (Step 12: create branch)
//	10: git config user.name                 (Step 13: host git name)
//	11: git config user.email                (Step 13: host git email)
//	12: docker exec ... git config user.name (Step 13: container git name)
//	13: docker exec ... git config user.email(Step 13: container git email)
//	14: docker exec ... sh -c (setup)        (Step 18: claude setup)
//	15: docker exec ... sh -c (token)        (Step 18: token check)
func happyCreateResponses(name string) []cmdResult {
	container := fmt.Sprintf("alpine-%s-dev-1", name)
	return []cmdResult{
		{stdout: ""},                 // 0: docker info
		{stdout: "/tmp/fakegitroot"}, // 1: git rev-parse --show-toplevel
		{stdout: "[]"},               // 2: docker compose ls
		{stdout: "main"},             // 3: git rev-parse --abbrev-ref HEAD
		{stdout: ""},                 // 4: docker image inspect (exists)
		{stdout: ""},                 // 5: docker compose up
		{stdout: fmt.Sprintf(`{"Name":"%s","Service":"dev"}`, container)}, // 6: docker compose ps
		{stdout: "git@github.com:user/repo.git"},                          // 7: git remote get-url origin
		{stdout: ""},                                                      // 8: docker exec git clone
		{stdout: ""},                                                      // 9: docker exec git checkout -b
		{stdout: "Test User"},                                             // 10: git config user.name
		{stdout: "test@example.com"},                                      // 11: git config user.email
		{stdout: ""},                                                      // 12: docker exec git config user.name
		{stdout: ""},                                                      // 13: docker exec git config user.email
		{stdout: ""},                                                      // 14: docker exec sh -c (claude setup)
		{stdout: "set"},                                                   // 15: docker exec sh -c (token check)
	}
}

// newTestCreateCmd builds a minimal cobra.Command suitable for calling runCreate
// directly. This avoids Cobra state accumulation from rootCmd.
// Flags (fromBranch, detach, jsonOutput) are set as package-level variables
// before calling this -- do NOT re-register them here (StringVar resets to default).
func newTestCreateCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create",
		RunE: runCreate,
	}
	cmd.SetContext(ctx)
	return cmd
}

// errResult is a convenience for creating an error cmdResult.
func errResult(stderr string) cmdResult {
	return cmdResult{stderr: stderr, err: fmt.Errorf("exit status 1")}
}
