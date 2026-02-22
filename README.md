# Sprite OpenCode Setup

`setup-sprite-opencode` creates a ready-to-code Sprite environment for the current repository.

## What it does

- Creates a new Sprite with a unique name based on repo and branch
- Installs OpenCode and ast-grep inside the Sprite
- Copies local auth/config files (`~/.local/share/opencode/auth.json`, `~/.config/opencode`, `~/.claude`)
- Copies repository `.env` to the Sprite and loads its variables
- Clones the current repo in the Sprite and checks out (or creates) the target branch
- Opens a Sprite shell and launches `opencode`

## Requirements

- `sprite` CLI installed and authenticated
- `git` available locally
- Local files/directories present:
  - `~/.local/share/opencode/auth.json`
  - `~/.config/opencode`
  - `~/.claude`
  - `<repo>/.env`

## Usage

```bash
go run ./cmd/setup-sprite-opencode --branch <branch-name>
```

To install globally:

```bash
make install
```

Run Opencode in an interactive Sprite session:

```bash
sprite exec -dir /home/sprite/code/alpine -tty bash -lc 'opencode run "/ralph-loop"'
```

Examples:

```bash
go run ./cmd/setup-sprite-opencode --branch feat/my-change
```

To build a standalone binary:

```bash
go build -o setup-sprite-opencode ./cmd/setup-sprite-opencode
./setup-sprite-opencode --branch feat/my-change
```

## Migration notes

- The Bash script remains in the repository for reference during migration.
- The Go CLI uses `sprites-go` APIs for sprite lifecycle, command execution, and filesystem transfer.
- Exit codes are now stable: `0` success, `1` runtime failure, `2` usage, `3` preflight, `4` auth, `5` cleanup failure.
- `--org` is currently fail-closed and exits with code `4` until end-to-end org scoping can be guaranteed.

## Notes

- Run the command from inside the git repository you want to clone into Sprite.
- `--branch` is required.
- In TTY mode, the CLI launches `opencode` directly inside the sprite.
- In non-TTY mode, the CLI prints `sprite_id` and a reconnect command instead of attaching interactively.
