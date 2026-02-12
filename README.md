# Alpine

Ephemeral dev environments for parallel AI coding agents.

Alpine creates fully isolated, containerized development environments. Each environment gets its own repo clone, git branch, services, and Claude Code instance.

## Install

```bash
go install github.com/max/alpine/cmd/alpine@latest
```

Or build from source:

```bash
make build        # outputs to bin/alpine
make install      # installs to $GOPATH/bin
```

## Prerequisites

- [Docker](https://www.docker.com/) (Docker Desktop on macOS, or Docker Engine on Linux)
- Git remote `origin` configured on the repository
- Git authentication via SSH agent (`SSH_AUTH_SOCK`) or `GITHUB_TOKEN`/`GH_TOKEN`

## Quick Start

```bash
cd your-project
alpine create my-feature
```

This will:

1. Build a dev container from `ubuntu:24.04` (or your configured base image)
2. Clone the repo and create a `feature/my-feature` branch
3. Copy env files and run your install command (if configured)
4. Drop you into a shell inside the container

## Configuration

### alpine.yaml

Place an `alpine.yaml` in your project root to configure environments. All fields are optional.

```yaml
# Base Docker image (default: ubuntu:24.04)
base_image: node:20

# Post-clone install command (default: none)
install: npm install

# Files to copy into the container (default: none)
env_files:
  - .env
  - .env.local

# Sidecar services (default: none)
# Supported: postgres, redis
services:
  - postgres
  - redis
```

See `alpine.yaml.example` for the full template with comments.

**Fields:**

| Field | Default | Description |
|---|---|---|
| `base_image` | `ubuntu:24.04` | Docker image for the dev container |
| `install` | *(none)* | Shell command to run after cloning |
| `env_files` | *(none)* | Files to copy from host into `/workspace/` |
| `services` | *(none)* | Sidecar services: `postgres`, `redis` |

If `alpine.yaml` is missing, Alpine uses defaults and logs a warning.

### Docker Compose

Alpine generates a Docker Compose project for each environment at runtime. No static `docker-compose.yml` is needed.

The generated compose project includes:

- **Dev container** -- your configured base image with git, curl, SSH, and Claude CLI pre-installed. Runs as a non-root `claude` user with `cap_drop: ALL` and `no-new-privileges`.
- **PostgreSQL** (if configured) -- `postgres:16` with `trust` auth and tmpfs storage.
- **Redis** (if configured) -- `redis:7` with persistence disabled.

**Environment variables** are passed through from the host using Docker's passthrough syntax (no literal values in the generated YAML):

- `ANTHROPIC_API_KEY`
- `CLAUDE_CODE_OAUTH_TOKEN`
- `GITHUB_TOKEN`
- `GH_TOKEN`
- `SSH_AUTH_SOCK`

Set these on the host before running `alpine create`. The container inherits them automatically.

**Service aliases** for use inside the container:

| Service | Hostname | Port |
|---|---|---|
| PostgreSQL | `db` | 5432 |
| Redis | `cache` | 6379 |

### Environment Variables

Alpine itself does not read a `.env` file. Set these in your shell:

```bash
# Required: at least one of these for git auth
export SSH_AUTH_SOCK=...           # SSH agent socket (usually already set)
export GITHUB_TOKEN=ghp_...       # or GH_TOKEN

# Required: at least one of these for Claude
export ANTHROPIC_API_KEY=sk-...
export CLAUDE_CODE_OAUTH_TOKEN=...     # alternative to API key
```

The `env_files` field in `alpine.yaml` controls which files are copied into the container -- it does not load variables into Alpine itself.

## Commands

### create

```bash
alpine create <name> [flags]
```

Create an isolated dev environment.

| Flag | Description |
|---|---|
| `--from <branch>` | Base branch (default: current branch) |
| `-d, --detach` | Return immediately after environment is ready |
| `--json` | Machine-readable JSON output |

### list

```bash
alpine list [flags]
```

Show all active environments.

| Flag | Description |
|---|---|
| `--json` | Machine-readable JSON output |

### status

```bash
alpine status <name> [flags]
```

Show details for a specific environment.

| Flag | Description |
|---|---|
| `--json` | Machine-readable JSON output |

### Global Flags

| Flag | Description |
|---|---|
| `-v, --verbose` | Verbose output |
| `--json` | Machine-readable JSON output |

## Development

```bash
make build     # build binary
make test      # run tests
make lint      # vet + golangci-lint
make clean     # remove build artifacts
```

Requires Go 1.22+.
