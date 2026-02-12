# Brainstorm: Ephemeral Dev Environment CLI

**Date:** 2026-02-11
**Status:** Ready for planning

## What We're Building

A Go CLI (`alpine`) that creates fully isolated, containerized development environments for running parallel AI coding agents. A single command spins up an entire dev environment inside Docker -- its own repo clone, branch, services, and Claude Code instance -- with zero cross-talk between environments.

### Commands

| Command | Description |
|---|---|
| `alpine create <name>` | Create a new isolated dev environment |
| `alpine teardown <name>` | Destroy an environment (auto-commits and pushes first) |
| `alpine list` | Show all active environments and their status |

### Workflow

```
alpine create auth-redesign
```

1. Start Docker Desktop if not running
2. Build/pull the dev container image (Claude CLI + language runtimes + git)
3. Start docker compose project `dev-auth-redesign` with dev container + services (db, cache)
4. Clone the repo inside the dev container
5. Create and checkout `feature/auth-redesign` branch (from current branch by default, `--from main` to override)
6. Copy `.env` and `.env.local` into the container
7. Run the configured install command (e.g., `npm install`)
8. Launch `claude --dangerously-skip-permissions` inside the container
9. Attach the user's terminal to the Claude session

```
alpine teardown auth-redesign
```

1. Auto-commit any uncommitted changes inside the container
2. Push the feature branch to remote
3. Destroy Docker containers and volumes
4. Clean up any host-side state

## Why This Approach

### Fully Containerized (No Host Mounts)

We chose full containerization over mounted worktrees for these reasons:

- **Portability**: The environment works identically on any machine with Docker. No dependency on host OS, filesystem, or installed tools.
- **True isolation**: No macOS Docker mount performance issues. No accidental host filesystem interference.
- **Consistency**: The dev container image defines the exact toolchain. No "works on my machine" problems.

The tradeoff is that code changes only persist via git. The CLI handles this by auto-committing and pushing on teardown.

### Single Binary with Subcommands

Standard Go CLI pattern (`alpine create`, `alpine teardown`, `alpine list`). Easier to distribute, extend, and document than separate binaries.

### Configurable Project Types

An `alpine.yaml` config file in the project root defines the install command, compose file, env files, and services. This supports Node, Python, Go, Ruby, or any stack without hardcoding assumptions.

## Key Decisions

1. **CLI structure**: Single binary (`alpine`) with subcommands (`create`, `teardown`, `list`)
2. **Containerization**: Fully containerized -- repo cloned inside Docker, no host mounts
3. **Branch base**: Branch from current branch by default, `--from <branch>` flag to override
4. **Project config**: `alpine.yaml` in project root defines stack-specific settings
5. **Port strategy**: Each environment is fully isolated within its Docker network. Services communicate internally. App port mapped to host for browser access only.
6. **Lifecycle**: `alpine list` included. No auto-teardown -- too dangerous.
7. **Teardown safety**: Auto-commit and push uncommitted changes before destroying the environment
8. **Claude execution**: Runs inside the dev container. User's terminal attaches to the Claude session.

## Config File Shape

```yaml
# alpine.yaml
install: npm install
compose_file: docker-compose.dev.yml
env_files:
  - .env
  - .env.local
worktree_dir: .worktrees
base_image: node:20  # or python:3.12, golang:1.22, etc.
services:
  - db
  - cache
```

## Docker Architecture

```
docker compose -p dev-<name>

  dev-<name>-app (dev container)
    - Claude CLI
    - Language runtime
    - Git
    - Cloned repo on feature/<name> branch
    - Runs: claude --dangerously-skip-permissions

  dev-<name>-db (postgres:16, tmpfs)
  dev-<name>-cache (redis, if configured)

  Network: dev-<name>_default (isolated)
```

## Open Questions (Resolved)

| Question | Decision |
|---|---|
| Non-Node projects? | Configurable via `alpine.yaml` from day one |
| Branch from main or current? | Current branch default, `--from` flag to override |
| `list-features` command? | Yes, as `alpine list` |
| Auto-teardown for stale envs? | No -- too risky to auto-destroy work |
| Port conflicts? | Full network isolation, no host ports except app for browser |
| Claude in container or on host? | Inside container for full portability |

## Remaining Open Questions

- What base images should we pre-build or support? (Node, Python, Go, Ruby)
- Should `alpine.yaml` be auto-generated with `alpine init`?
- How should API keys (ANTHROPIC_API_KEY) be passed into the container? (env var forwarding from host, or config file)
- Should `alpine create` support a `--prompt` flag to pass an initial task to Claude?
- How should the dev container image be distributed? (Build locally from Dockerfile, or pull from a registry)
