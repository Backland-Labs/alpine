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

- `cmd/alpine/main.go` -- CLI root, config loading, validation
- `cmd/alpine/create.go` -- `alpine create` (15-step workflow)
- `cmd/alpine/docker.go` -- Docker/Compose/Git operations, Dockerfile and Compose YAML generation
- `cmd/alpine/list.go` -- `alpine list`
- `cmd/alpine/status.go` -- `alpine status`

## Configuration

- `alpine.yaml` -- per-project config (base_image, install, env_files, services)
- Docker Compose YAML is generated at runtime, never stored as a static file
- Environment variables use passthrough syntax only (no secrets in generated YAML)

## Conventions

- All external commands go through `run()` in docker.go -- never use `sh -c`
- Error handling: `userErr()` (exit 1) vs `sysErr()` (exit 2)
- Services are aliased: postgres -> `db`, redis -> `cache`
- Container user is always `claude` (non-root)
- Never echo sensitive env vars directly

```bash
# Check if a variable is set without printing its value
[ -n "$VARIABLE" ]
```
