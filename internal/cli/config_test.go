package cli

import (
	"os"
	"path/filepath"
	"testing"
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
