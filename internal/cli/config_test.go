package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"alpine/internal/apperr"
)

func noSpriteLookup() (string, error) { return "", nil }

func TestSlugify(t *testing.T) {
	got := slugify("Feat/My.Change Test")
	if got != "feat-my-change-test" {
		t.Fatalf("slugify mismatch: %q", got)
	}
}

func TestResolveTokenPrefersEnv(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"token":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	token, warn, err := resolveTokenWithSpriteLookup([]string{"SPRITES_TOKEN=from-env"}, authPath, noSpriteLookup)
	if err != nil {
		t.Fatal(err)
	}
	if token != "from-env" {
		t.Fatalf("expected env token, got %q", token)
	}
	if !warn {
		t.Fatal("expected warning for token mismatch")
	}
}

func TestResolveTokenFallsBackToAuthJSON(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"token":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	token, warn, err := resolveTokenWithSpriteLookup(nil, authPath, noSpriteLookup)
	if err != nil {
		t.Fatal(err)
	}
	if token != "from-file" {
		t.Fatalf("expected file token, got %q", token)
	}
	if warn {
		t.Fatal("did not expect mismatch warning")
	}
}

func TestResolveTokenMissingEverywhere(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"other":"value"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := resolveTokenWithSpriteLookup(nil, authPath, noSpriteLookup)
	if err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestResolveTokenFallsBackToSpritesLogin(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"other":"value"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	token, warn, err := resolveTokenWithSpriteLookup(nil, authPath, func() (string, error) {
		return "from-sprites-login", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if token != "from-sprites-login" {
		t.Fatalf("expected sprites login token, got %q", token)
	}
	if warn {
		t.Fatal("did not expect mismatch warning")
	}
}

func TestTokenFromAuthJSONNestedAuthObject(t *testing.T) {
	dir := t.TempDir()
	authPath := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"auth":{"access_token":"nested-token"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := tokenFromAuthJSON(authPath)
	if err != nil {
		t.Fatal(err)
	}
	if token != "nested-token" {
		t.Fatalf("expected nested token, got %q", token)
	}
}

func TestParsePlainModeSkipsRepoChecks(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	authPath := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(`{"token":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse([]string{"--plain"}, []string{"SPRITES_TOKEN=from-env"}, io.Discard)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}
	if !cfg.Plain {
		t.Fatal("expected plain mode")
	}
	if cfg.NamePrefix != "plain" {
		t.Fatalf("expected plain name prefix, got %q", cfg.NamePrefix)
	}
	if cfg.LocalEnvFile != "" {
		t.Fatalf("expected no local env file in plain mode, got %q", cfg.LocalEnvFile)
	}
	if cfg.RepoRoot != "" || cfg.RepoURL != "" || cfg.RepoName != "" {
		t.Fatalf("expected repo fields to be empty in plain mode: root=%q url=%q name=%q", cfg.RepoRoot, cfg.RepoURL, cfg.RepoName)
	}
}

func TestParsePlainRejectsBranch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	authPath := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(`{"token":"from-file"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Parse([]string{"--plain", "--branch", "feat/test"}, []string{"SPRITES_TOKEN=from-env"}, io.Discard)
	if !errors.Is(err, apperr.ErrUsage) {
		t.Fatalf("expected usage error, got: %v", err)
	}
}
