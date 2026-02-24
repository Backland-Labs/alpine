package remote

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitSetupScriptUsesGHTokenForGitHubHTTPS(t *testing.T) {
	result := runGitSetupScript(t, "https://github.com/example/private-repo.git", "GH_TOKEN=test-gh-token\n")
	if result.err != nil {
		t.Fatalf("expected script to succeed, got error: %v\noutput:\n%s", result.err, result.output)
	}

	if result.askpassPath == "" {
		t.Fatal("expected GIT_ASKPASS to be set for GitHub HTTPS remote when GH_TOKEN is present")
	}
	if got, want := result.terminalPrompt, "0"; got != want {
		t.Fatalf("expected GIT_TERMINAL_PROMPT=%q, got %q", want, got)
	}

	if _, err := os.Stat(result.askpassPath); !os.IsNotExist(err) {
		t.Fatalf("expected askpass helper to be cleaned up, stat err=%v", err)
	}
}

func TestGitSetupScriptUsesGitHubTokenFallback(t *testing.T) {
	result := runGitSetupScript(t, "https://github.com/example/private-repo.git", "GITHUB_TOKEN=test-github-token\n")
	if result.err != nil {
		t.Fatalf("expected script to succeed, got error: %v\noutput:\n%s", result.err, result.output)
	}

	if result.askpassPath == "" {
		t.Fatal("expected GIT_ASKPASS to be set for GitHub HTTPS remote when GITHUB_TOKEN is present")
	}
}

func TestGitSetupScriptSkipsAskPassForNonGitHubRemote(t *testing.T) {
	result := runGitSetupScript(t, "https://gitlab.com/example/private-repo.git", "GH_TOKEN=test-gh-token\n")
	if result.err != nil {
		t.Fatalf("expected script to succeed, got error: %v\noutput:\n%s", result.err, result.output)
	}

	if result.askpassPath != "" {
		t.Fatalf("expected empty GIT_ASKPASS for non-GitHub remote, got %q", result.askpassPath)
	}
	if result.terminalPrompt != "" {
		t.Fatalf("expected empty GIT_TERMINAL_PROMPT for non-GitHub remote, got %q", result.terminalPrompt)
	}
}

type gitSetupResult struct {
	askpassPath    string
	terminalPrompt string
	output         string
	err            error
}

func runGitSetupScript(t *testing.T, repoURL, envContent string) gitSetupResult {
	t.Helper()

	home := t.TempDir()
	repoDir := filepath.Join(home, "code", "repo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("create repo dir: %v", err)
	}

	if envContent != "" {
		envPath := filepath.Join(home, ".env")
		if err := os.WriteFile(envPath, []byte(envContent), 0o600); err != nil {
			t.Fatalf("write .env: %v", err)
		}
	}

	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}

	fakeGit := `#!/bin/sh
printf '%s' "${GIT_ASKPASS:-}" > "$HOME/last-git-askpass"
printf '%s' "${GIT_TERMINAL_PROMPT:-}" > "$HOME/last-git-terminal-prompt"
exit 0
`
	gitPath := filepath.Join(binDir, "git")
	if err := os.WriteFile(gitPath, []byte(fakeGit), 0o700); err != nil {
		t.Fatalf("write fake git: %v", err)
	}

	cmd := exec.Command("sh", "-c", gitSetupScript)
	cmd.Env = []string{
		"HOME=" + home,
		"PATH=" + strings.Join([]string{binDir, "/usr/bin", "/bin", "/usr/sbin", "/sbin"}, ":"),
		"SPRITE_REPO_URL=" + repoURL,
		"SPRITE_REPO_DIR=" + repoDir,
		"SPRITE_TARGET_BRANCH=feat/test-token-auth",
	}
	out, err := cmd.CombinedOutput()

	askpassBytes, _ := os.ReadFile(filepath.Join(home, "last-git-askpass"))
	promptBytes, _ := os.ReadFile(filepath.Join(home, "last-git-terminal-prompt"))

	return gitSetupResult{
		askpassPath:    strings.TrimSpace(string(askpassBytes)),
		terminalPrompt: strings.TrimSpace(string(promptBytes)),
		output:         strings.TrimSpace(string(out)),
		err:            err,
	}
}
