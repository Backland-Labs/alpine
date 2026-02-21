package flow

import (
	"math/rand"
	"strings"
	"testing"
)

func TestRandomSpriteNameWithPrefix(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	name := randomSpriteName("repo-branch", r)
	if !strings.HasPrefix(name, "repo-branch-") {
		t.Fatalf("expected prefix, got %q", name)
	}
	if strings.Count(name, "-") < 3 {
		t.Fatalf("expected adjective/noun suffix, got %q", name)
	}
}

func TestRandomSpriteNameNoPrefix(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	name := randomSpriteName("", r)
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		t.Fatalf("invalid name formatting: %q", name)
	}
	if strings.Count(name, "-") != 1 {
		t.Fatalf("expected adjective-noun form, got %q", name)
	}
}

func TestSanitizeRepoURLHTTPRemovesUserinfo(t *testing.T) {
	g := sanitizeRepoURL("https://user:secret@example.com/repo.git")
	if g != "https://example.com/repo.git" {
		t.Fatalf("unexpected sanitized url: %s", g)
	}
}

func TestSanitizeRepoURLScpStyleRemovesUser(t *testing.T) {
	g := sanitizeRepoURL("git@github.com:owner/repo.git")
	if g != "github.com:owner/repo.git" {
		t.Fatalf("unexpected sanitized url: %s", g)
	}
}
