# Sprite OpenCode Setup

`sprite-opencode` is a basic CLI that creates a ready-to-code Sprite environment for the current repository.

## What it does

- Creates a new Sprite with a unique name based on repo and branch
- Installs OpenCode, Plannotator, and ast-grep inside the Sprite
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
./sprite-opencode setup --branch <branch-name> [--org <org-name>]
./sprite-opencode --branch <branch-name> [--org <org-name>]
./sprite-opencode install
./sprite-opencode version
```

Install globally with Make:

```bash
make install
sprite-opencode --help
```

Examples:

```bash
./sprite-opencode setup --branch feat/my-change
./sprite-opencode setup --branch fix/login-timeout --org my-org
```

## Notes

- Run the CLI from inside the git repository you want to clone into Sprite.
- `--branch` is required.
- If `expect` is installed, the script auto-enters `sprite console`; otherwise it falls back to direct `sprite exec` launch.
