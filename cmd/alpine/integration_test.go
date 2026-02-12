//go:build integration

package main

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration tests require a running Docker daemon.
// Run with: go test -tags=integration -v ./cmd/alpine/

func TestIntegrationCreateListStatusCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, _, err := defaultRun(ctx, "docker", "info")
	if err != nil {
		t.Skip("Docker daemon not available, skipping integration test")
	}

	_, _, err = defaultRun(ctx, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Skip("Not in a git repository, skipping integration test")
	}

	if os.Getenv("SSH_AUTH_SOCK") == "" && os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("GH_TOKEN") == "" {
		t.Skip("No git auth configured, skipping integration test")
	}

	name := "integration-test"
	_ = composeDown(ctx, name)

	// TODO: Add full create -> list -> status -> cleanup integration test.
	t.Log("Integration test skeleton - extend when CI environment supports it")
}
