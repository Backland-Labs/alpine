package main

import (
	"context"
	"strings"
	"testing"
)

func TestGitClone(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		err := gitClone(context.Background(), "container", "git@github.com:u/r.git", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("authentication failed")})
		err := gitClone(context.Background(), "container", "git@github.com:u/r.git", "main")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "git clone failed") {
			t.Fatalf("error = %q, want to contain 'git clone failed'", err.Error())
		}
	})
}

func TestGitCreateBranch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		err := gitCreateBranch(context.Background(), "container", "my-feature")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("branch already exists")})
		err := gitCreateBranch(context.Background(), "container", "my-feature")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "git checkout -b failed") {
			t.Fatalf("error = %q, want to contain 'git checkout -b failed'", err.Error())
		}
	})
}

func TestGitConfigureUser(t *testing.T) {
	t.Run("success 4 calls", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{
			{stdout: "Test User"},     // host: git config user.name
			{stdout: "test@test.com"}, // host: git config user.email
			{stdout: ""},              // container: git config user.name
			{stdout: ""},              // container: git config user.email
		})
		err := gitConfigureUser(context.Background(), "container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(*calls) != 4 {
			t.Fatalf("expected 4 calls, got %d", len(*calls))
		}
		// Verify host reads use "git" directly.
		if (*calls)[0].name != "git" {
			t.Errorf("call 0: expected git, got %s", (*calls)[0].name)
		}
		if (*calls)[1].name != "git" {
			t.Errorf("call 1: expected git, got %s", (*calls)[1].name)
		}
		// Verify container writes use "docker exec".
		if (*calls)[2].name != "docker" {
			t.Errorf("call 2: expected docker, got %s", (*calls)[2].name)
		}
		if (*calls)[3].name != "docker" {
			t.Errorf("call 3: expected docker, got %s", (*calls)[3].name)
		}
	})

	t.Run("host config missing uses fallback", func(t *testing.T) {
		calls := mockRunRecording(t, []cmdResult{
			errResult("not set"), // git config user.name fails
			errResult("not set"), // git config user.email fails
			{stdout: ""},         // docker exec git config user.name "alpine"
			{stdout: ""},         // docker exec git config user.email "alpine@localhost"
		})
		err := gitConfigureUser(context.Background(), "container")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Verify the fallback values were passed in the container config calls.
		call2 := (*calls)[2]
		found := false
		for _, a := range call2.args {
			if a == "alpine" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected fallback name 'alpine' in call args: %v", call2.args)
		}
		call3 := (*calls)[3]
		found = false
		for _, a := range call3.args {
			if a == "alpine@localhost" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected fallback email 'alpine@localhost' in call args: %v", call3.args)
		}
	})

	t.Run("container config fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "Test User"},
			{stdout: "test@test.com"},
			errResult("permission denied"), // docker exec git config user.name fails
		})
		err := gitConfigureUser(context.Background(), "container")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to set git user.name") {
			t.Fatalf("error = %q, want to contain 'failed to set git user.name'", err.Error())
		}
	})

	t.Run("container email config fails", func(t *testing.T) {
		mockRun(t, []cmdResult{
			{stdout: "Test User"},
			{stdout: "test@test.com"},
			{stdout: ""},                   // user.name succeeds
			errResult("permission denied"), // user.email fails
		})
		err := gitConfigureUser(context.Background(), "container")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to set git user.email") {
			t.Fatalf("error = %q, want to contain 'failed to set git user.email'", err.Error())
		}
	})
}

func TestGitGetCurrentBranch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "main"}})
		branch, err := gitGetCurrentBranch(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branch != "main" {
			t.Fatalf("branch = %q, want %q", branch, "main")
		}
	})

	t.Run("detached HEAD returns error", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "HEAD"}})
		_, err := gitGetCurrentBranch(context.Background())
		if err == nil {
			t.Fatal("expected error for detached HEAD")
		}
		if !strings.Contains(err.Error(), "detached") {
			t.Fatalf("error = %q, want to contain 'detached'", err.Error())
		}
	})

	t.Run("error", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("not a git repo")})
		_, err := gitGetCurrentBranch(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to determine current branch") {
			t.Fatalf("error = %q, want to contain 'failed to determine current branch'", err.Error())
		}
	})
}

func TestGitGetRemoteURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "git@github.com:user/repo.git"}})
		url, err := gitGetRemoteURL(context.Background(), "origin")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "git@github.com:user/repo.git" {
			t.Fatalf("url = %q, want %q", url, "git@github.com:user/repo.git")
		}
	})

	t.Run("error", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("no such remote")})
		_, err := gitGetRemoteURL(context.Background(), "origin")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "failed to get remote URL") {
			t.Fatalf("error = %q, want to contain 'failed to get remote URL'", err.Error())
		}
	})
}

func TestGitFindRoot(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: "/home/user/project"}})
		root, err := gitFindRoot()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if root != "/home/user/project" {
			t.Fatalf("root = %q, want %q", root, "/home/user/project")
		}
	})

	t.Run("error", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("not a git repo")})
		_, err := gitFindRoot()
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not inside a git repository") {
			t.Fatalf("error = %q, want to contain 'not inside a git repository'", err.Error())
		}
	})
}
