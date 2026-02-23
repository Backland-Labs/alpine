# Sprite OpenCode Setup

`sc` creates a ready-to-code Sprite environment in either repo mode or plain mode.

## What it does

- Creates a new Sprite with a unique random name (repo/branch-tagged in repo mode)
- Installs OpenCode and ast-grep inside the Sprite
- Copies local auth/config files (`~/.local/share/opencode/auth.json`, `~/.config/opencode`)
- In repo mode: copies repository `.env`, clones the current repo, and checks out (or creates) the target branch
- In plain mode: skips all repository and git setup
- Opens a Sprite shell and launches `opencode`

## Requirements

- `sprite` CLI installed and authenticated
- `git` available locally (repo mode only)
- Local files/directories present:
  - `~/.local/share/opencode/auth.json`
  - `~/.config/opencode`
  - `<repo>/.env` (repo mode only)

## Usage

```bash
go run ./cmd/sc --branch <branch-name>
go run ./cmd/sc --plain
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
go run ./cmd/sc --branch feat/my-change
go run ./cmd/sc --plain
```

To build a standalone binary:

```bash
go build -o sc ./cmd/sc
./sc --branch feat/my-change
```

## Migration notes

- The Bash script remains in the repository for reference during migration.
- The Go CLI uses `sprites-go` APIs for sprite lifecycle, command execution, and filesystem transfer.
- Exit codes are now stable: `0` success, `1` runtime failure, `2` usage, `3` preflight, `4` auth, `5` cleanup failure.

## Notes

- Repo mode must run from inside the git repository you want to clone into Sprite.
- Use exactly one launch mode: `--branch <branch-name>` or `--plain`.
- In TTY mode, the CLI launches `opencode` directly inside the sprite.
- In non-TTY mode, the CLI prints `sprite_id` and a reconnect command instead of attaching interactively.
