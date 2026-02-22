package remote

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	sprites "github.com/superfly/sprites-go"
	"golang.org/x/term"
)

const launchScript = `. ~/.zprofile >/dev/null 2>&1 || true
. ~/.zshrc >/dev/null 2>&1 || true
if [ -f "$HOME/.env" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      ""|\#*)
        continue
        ;;
      *=*)
        key="${line%%=*}"
        val="${line#*=}"
        case "$key" in
          ""|*[!A-Za-z0-9_]*)
            continue
            ;;
        esac
        export "$key=$val"
        ;;
    esac
  done < "$HOME/.env"
fi
cd "$SPRITE_REPO_DIR" >/dev/null 2>&1 || true
hash -r 2>/dev/null || true
if command -v opencode >/dev/null 2>&1; then exec opencode; fi
exec "$HOME/.opencode/bin/opencode"`

func Launch(ctx context.Context, sp *sprites.Sprite, repoDir string, stdout, stderr io.Writer) error {
	if !isTTY() {
		reconnect := "sprite exec -s " + shellQuote(sp.Name()) + " -tty zsh -lc " + shellQuote(launchScriptWithRepoDir(repoDir))
		fmt.Fprintf(stdout, "status=ready\nsprite_id=%s\nreconnect=%s\n", sp.Name(), reconnect)
		return nil
	}

	cmd := sp.CommandContext(ctx, "zsh", "-lc", launchScript)
	cmd.Env = []string{"SPRITE_REPO_DIR=" + repoDir}
	cmd.SetTTY(true)
	if cols, rows, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		_ = cmd.SetTTYSize(uint16(rows), uint16(cols))
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	defer signal.Stop(resize)
	go func() {
		for range resize {
			if cols, rows, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				_ = cmd.SetTTYSize(uint16(rows), uint16(cols))
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launch opencode: %w", err)
	}
	return nil
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func launchScriptWithRepoDir(repoDir string) string {
	return `. ~/.zprofile >/dev/null 2>&1 || true; . ~/.zshrc >/dev/null 2>&1 || true; if [ -f "$HOME/.env" ]; then while IFS= read -r line || [ -n "$line" ]; do case "$line" in ""|\#*) continue ;; *=*) key="${line%%=*}"; val="${line#*=}"; case "$key" in ""|*[!A-Za-z0-9_]*) continue ;; esac; export "$key=$val" ;; esac; done < "$HOME/.env"; fi; cd ` + shellQuote(repoDir) + ` >/dev/null 2>&1 || true; hash -r 2>/dev/null || true; if command -v opencode >/dev/null 2>&1; then exec opencode; fi; exec "$HOME/.opencode/bin/opencode"`
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
