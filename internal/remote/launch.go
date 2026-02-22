package remote

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	sprites "github.com/superfly/sprites-go"
	"golang.org/x/term"
)

const directLaunchScript = `. ~/.zprofile >/dev/null 2>&1 || true; . ~/.zshrc >/dev/null 2>&1 || true; cd "$SPRITE_REPO_DIR" >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; if command -v opencode >/dev/null 2>&1; then exec opencode; fi; exec "$HOME/.opencode/bin/opencode"`

const expectLaunchScript = `set timeout 20
set cmd [list sprite]
if {[info exists env(SPRITE_ORG)] && $env(SPRITE_ORG) ne ""} {
  lappend cmd -o $env(SPRITE_ORG)
}
lappend cmd console -s $env(SPRITE_NAME)
spawn {*}$cmd
after 1200
send -- ". ~/.profile >/dev/null 2>&1 || true; cd \"$env(SPRITE_REPO_DIR)\" >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; opencode\r"
interact`

func Launch(ctx context.Context, sp *sprites.Sprite, repoDir string, stdout, stderr io.Writer) error {
	org := ""
	if orgInfo := sp.Organization(); orgInfo != nil {
		org = strings.TrimSpace(orgInfo.Name)
	}

	if !isTTY() {
		reconnect := reconnectCommand(sp.Name(), org, repoDir)
		fmt.Fprintf(stdout, "status=ready\nsprite_id=%s\nreconnect=%s\n", sp.Name(), reconnect)
		return nil
	}

	if _, err := exec.LookPath("expect"); err == nil {
		cmd := exec.CommandContext(ctx, "expect", "-c", expectLaunchScript)
		cmd.Env = append(os.Environ(),
			"SPRITE_NAME="+sp.Name(),
			"SPRITE_ORG="+org,
			"SPRITE_REPO_DIR="+repoDir,
		)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
		fmt.Fprintln(stderr, "Automatic console launch failed, falling back to direct launch.")
	} else {
		fmt.Fprintln(stderr, "'expect' is not installed, falling back to direct launch.")
	}

	args := []string{}
	if org != "" {
		args = append(args, "-o", org)
	}
	args = append(args,
		"exec",
		"-s", sp.Name(),
		"-env", "SPRITE_REPO_DIR="+repoDir,
		"-tty",
		"zsh",
		"-lc",
		directLaunchScript,
	)

	cmd := exec.CommandContext(ctx, "sprite", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launch opencode: %w", err)
	}
	return nil
}

func reconnectCommand(spriteName, org, repoDir string) string {
	cmd := "sprite"
	if org != "" {
		cmd += " -o " + shellQuote(org)
	}
	cmd += " exec -s " + shellQuote(spriteName)
	cmd += " -env " + shellQuote("SPRITE_REPO_DIR="+repoDir)
	cmd += " -tty zsh -lc " + shellQuote(launchScriptWithRepoDir(repoDir))
	return cmd
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func launchScriptWithRepoDir(repoDir string) string {
	return `. ~/.zprofile >/dev/null 2>&1 || true; . ~/.zshrc >/dev/null 2>&1 || true; cd ` + shellQuote(repoDir) + ` >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; if command -v opencode >/dev/null 2>&1; then exec opencode; fi; exec "$HOME/.opencode/bin/opencode"`
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
