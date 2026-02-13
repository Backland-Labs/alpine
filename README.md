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
# Supported: postgres, redis, browser
services:
  - postgres
  - redis
  - browser
```

See `alpine.yaml.example` for the full template with comments.

**Fields:**

| Field | Default | Description |
|---|---|---|
| `base_image` | `ubuntu:24.04` | Docker image for the dev container |
| `install` | *(none)* | Shell command to run after cloning |
| `env_files` | *(none)* | Files to copy from host into `/workspace/` |
| `services` | *(none)* | Sidecar services: `postgres`, `redis`, `browser` |

If `alpine.yaml` is missing, Alpine uses defaults and logs a warning.

### Docker Compose

Alpine generates a Docker Compose project for each environment at runtime. No static `docker-compose.yml` is needed.

The generated compose project includes:

- **Dev container** -- your configured base image with git, curl, SSH, and Claude CLI pre-installed. Runs as a non-root `claude` user with `cap_drop: ALL` and `no-new-privileges`.
- **PostgreSQL** (if configured) -- `postgres:16` with `trust` auth and tmpfs storage.
- **Redis** (if configured) -- `redis:7` with persistence disabled.
- **Browser** (if configured) -- `browserless/chromium:latest` for Playwright/Puppeteer E2E testing. Accessible at `ws://browser:3000` from the dev container via `BROWSER_WS_ENDPOINT`.

**Environment variables** are passed through from the host using Docker's passthrough syntax (no literal values in the generated YAML):

- `ANTHROPIC_API_KEY`
- `CLAUDE_CODE_OAUTH_TOKEN`
- `GITHUB_TOKEN`
- `GH_TOKEN`
- `SSH_AUTH_SOCK`

Set these on the host before running `alpine create`. The container inherits them automatically.

**Service aliases** for use inside the container:

| Service | Hostname | Port | Env var |
|---|---|---|---|
| PostgreSQL | `db` | 5432 | -- |
| Redis | `cache` | 6379 | -- |
| Browser | `browser` | 3000 | `BROWSER_WS_ENDPOINT` |

### Environment Variables

Alpine loads `.env` from the project root (if present) before starting the container. Variables already exported in your shell take precedence.

```bash
# Required: at least one of these for git auth
export SSH_AUTH_SOCK=...           # SSH agent socket (usually already set)
export GITHUB_TOKEN=ghp_...       # or GH_TOKEN

# Required: at least one of these for Claude
export ANTHROPIC_API_KEY=sk-...
export CLAUDE_CODE_OAUTH_TOKEN=...     # alternative to API key (run `claude setup-token` to generate)
```

You can place these in a `.env` file in your project root instead of exporting them in your shell.

The `env_files` field in `alpine.yaml` controls which additional files are copied into the container.

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

## Project Structure

```
cmd/alpine/
├── main.go           CLI root, Cobra setup, flags
├── main_entry.go     OS entrypoint with signal handling
├── create.go         alpine create (19-step workflow)
├── list.go           alpine list
├── status.go         alpine status
├── exec.go           Shell execution (run, runInteractive, ExecError)
├── docker.go         Docker daemon ops (health check, compose up/down)
├── compose.go        Compose YAML generation (templates, service defaults)
├── dockerfile.go     Dockerfile generation and hashing
├── git.go            Git operations (clone, branch, configure, find root)
├── container.go      Container interaction (inspect, copy files, check processes)
├── dotenv.go         .env file loading
├── config.go         Config struct, loadConfig, validate
├── errors.go         exitError, userErr, sysErr
└── output.go         outputJSON, outputError
```

## Development

```bash
make build     # build binary
make test      # run tests
make lint      # vet + golangci-lint
make clean     # remove build artifacts
```

Requires Go 1.22+.
