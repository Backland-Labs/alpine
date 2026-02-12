# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**WHY**: Create fully isolated, containerized dev environments for running parallel AI coding agents.

**WHAT**:
- Tech stack: Go 1.22, Cobra, Docker Compose
- Architecture: Single-binary CLI that generates Dockerfiles and Compose YAML at runtime

**HOW**:
```bash
make build                    # build to bin/alpine
make install                  # install to $GOPATH/bin
alpine create my-feature      # create an environment
alpine list                   # show active environments
alpine status my-feature      # show environment details
```

## Key Architecture

### Commands
- `cmd/alpine/main.go` -- CLI root, Cobra setup, flags, execute()
- `cmd/alpine/main_entry.go` -- OS entrypoint with signal handling
- `cmd/alpine/create.go` -- `alpine create` (19-step workflow)
- `cmd/alpine/list.go` -- `alpine list`
- `cmd/alpine/status.go` -- `alpine status`

### Infrastructure
- `cmd/alpine/exec.go` -- Shell execution (`run()`, `runInteractive()`, `ExecError`)
- `cmd/alpine/docker.go` -- Docker daemon ops (health check, compose up/down, image exists)
- `cmd/alpine/compose.go` -- Compose YAML generation (templates, service defaults)
- `cmd/alpine/dockerfile.go` -- Dockerfile generation and hashing
- `cmd/alpine/git.go` -- Git operations (clone, branch, configure user, find root)
- `cmd/alpine/container.go` -- Container interaction (inspect, copy files, check processes)
- `cmd/alpine/dotenv.go` -- .env file loading

### Support
- `cmd/alpine/config.go` -- Config struct, loadConfig(), validate()
- `cmd/alpine/errors.go` -- exitError, userErr(), sysErr()
- `cmd/alpine/output.go` -- outputJSON(), outputError()

## Configuration

- `alpine.yaml` -- per-project config (base_image, install, env_files, services)
- Docker Compose YAML is generated at runtime, never stored as a static file
- Environment variables use passthrough syntax only (no secrets in generated YAML)

## Conventions

- One concern per file, named for what it contains (e.g., `git.go` not `utils.go`). If a file does two unrelated things, split it. Each test file mirrors its source file (`git.go` -> `git_test.go`).
- All external commands go through `run()` in exec.go -- never use `sh -c`
- Error handling: `userErr()` (exit 1) vs `sysErr()` (exit 2)
- Services are aliased: postgres -> `db`, redis -> `cache`
- Container user is always `claude` (non-root)
- Never echo sensitive env vars directly

```bash
# Check if a variable is set without printing its value
[ -n "$VARIABLE" ]
```
