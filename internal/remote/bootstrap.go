package remote

import (
	"context"
	"fmt"
	"io"

	sprites "github.com/superfly/sprites-go"
)

const bootstrapScript = `set -e
opencode_bin_dir="$HOME/.opencode/bin"
local_bin_dir="$HOME/.local/bin"
opencode_path_line='export PATH="$HOME/.opencode/bin:$PATH"'
local_bin_path_line='export PATH="$HOME/.local/bin:$PATH"'
env_line='if [ -f "$HOME/.env" ]; then set -a; . "$HOME/.env"; set +a; fi'

mkdir -p "$local_bin_dir"
export PATH="$local_bin_dir:$opencode_bin_dir:$PATH"

if [ ! -x "$HOME/.opencode/bin/opencode" ] && ! command -v opencode >/dev/null 2>&1; then
  tmp_installer="$(mktemp /tmp/opencode-install-XXXXXX.sh)"
  curl -fsSL -o "$tmp_installer" https://opencode.ai/install
  bash "$tmp_installer"
  rm -f "$tmp_installer"
fi

install_ast_grep() {
  if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
    return 0
  fi

  if command -v npm >/dev/null 2>&1; then
    npm i -g @ast-grep/cli --prefix "$HOME/.local" || true
    hash -r 2>/dev/null || true
    if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
      return 0
    fi
  fi

  if command -v pip3 >/dev/null 2>&1; then
    pip3 install --user ast-grep-cli || true
    hash -r 2>/dev/null || true
    if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
      return 0
    fi
  fi

  if command -v pip >/dev/null 2>&1; then
    pip install --user ast-grep-cli || true
    hash -r 2>/dev/null || true
    if command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1; then
      return 0
    fi
  fi

  platform="$(uname -s)"
  arch="$(uname -m)"
  target=""

  if [ "$platform" = "Linux" ]; then
    case "$arch" in
      x86_64|amd64) target="x86_64-unknown-linux-gnu" ;;
      aarch64|arm64) target="aarch64-unknown-linux-gnu" ;;
    esac
  elif [ "$platform" = "Darwin" ]; then
    case "$arch" in
      x86_64|amd64) target="x86_64-apple-darwin" ;;
      aarch64|arm64) target="aarch64-apple-darwin" ;;
    esac
  fi

  if [ -z "$target" ]; then
    printf "Unable to install ast-grep for %s/%s.\n" "$platform" "$arch" >&2
    return 1
  fi

  tmp_zip="/tmp/ast-grep-$RANDOM-$RANDOM.zip"
  curl -fsSL -o "$tmp_zip" "https://github.com/ast-grep/ast-grep/releases/latest/download/app-$target.zip"

  if command -v unzip >/dev/null 2>&1; then
    unzip -qo "$tmp_zip" -d "$local_bin_dir"
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c "import sys,zipfile; z=zipfile.ZipFile(sys.argv[1]); [z.extract(name, sys.argv[2]) for name in ('ast-grep','sg')]" "$tmp_zip" "$local_bin_dir"
  else
    rm -f "$tmp_zip"
    printf "Need unzip or python3 to install ast-grep.\n" >&2
    return 1
  fi

  rm -f "$tmp_zip"
  chmod +x "$local_bin_dir/ast-grep" "$local_bin_dir/sg" 2>/dev/null || true
  hash -r 2>/dev/null || true
  command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1
}

install_ast_grep

for rc_file in "$HOME/.zshrc" "$HOME/.zprofile" "$HOME/.bashrc" "$HOME/.profile"; do
  [ -f "$rc_file" ] || touch "$rc_file"
  grep -Fqx "$local_bin_path_line" "$rc_file" || printf "\n%s\n" "$local_bin_path_line" >> "$rc_file"
  grep -Fqx "$opencode_path_line" "$rc_file" || printf "%s\n" "$opencode_path_line" >> "$rc_file"
  grep -Fqx "$env_line" "$rc_file" || printf "%s\n" "$env_line" >> "$rc_file"
done

if [ -f "$HOME/.env" ]; then
  set -a
  . "$HOME/.env"
  set +a
fi
command -v opencode >/dev/null 2>&1
command -v ast-grep >/dev/null 2>&1 || command -v sg >/dev/null 2>&1
`

func Bootstrap(ctx context.Context, sp *sprites.Sprite, stdout, stderr io.Writer) error {
	cmd := sp.CommandContext(ctx, "zsh", "-lc", bootstrapScript)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bootstrap tooling: %w", err)
	}
	return nil
}
