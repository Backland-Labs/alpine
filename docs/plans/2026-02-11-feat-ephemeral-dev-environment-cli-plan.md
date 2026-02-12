---
title: "feat: Ephemeral Dev Environment CLI"
type: feat
date: 2026-02-11
deepened: 2026-02-12
---

# feat: Ephemeral Dev Environment CLI

## Enhancement Summary

**Deepened on:** 2026-02-12
**Research agents used:** architecture-strategist, performance-oracle, security-sentinel, code-simplicity-reviewer, pattern-recognition-specialist, agent-native-reviewer, best-practices-researcher, framework-docs-researcher
**Context7 queries:** Cobra v1.9.1, moby/term

### Critical Fixes from Research
1. **`docker compose ls --filter label=` does not work** -- switched to `--filter name=alpine-` (architecture review)
2. **`git add .` on teardown can commit secrets** -- replaced with `git add -u` + pre-commit secret scan (security audit)
3. **Containers run as root by default** -- added non-root user to Dockerfile generation (security audit)
4. **No agent-native support** -- added `--detach`, `--json`, `alpine status` to Phase 1 (agent-native review)
5. **Health checks use Docker defaults (30s interval)** -- tuned to 2s interval, saves 25-55s per create (performance review)

### Key Simplifications
- Flattened package structure from 10 files to 4
- Removed `CommandRunner` interface (premature abstraction)
- Removed `--remote`, `--no-install`, `--config` flags
- Hardcoded `apt-get` (package manager detection is YAGNI)
- Dropped MySQL support (add when someone asks)
- Single `defer` rollback instead of per-step tracking

---

## Overview

Build `alpine`, a Go CLI that creates fully isolated, containerized development environments for running parallel AI coding agents. A single command (`alpine create <name>`) spins up an entire dev environment inside Docker -- its own repo clone, branch, services, and Claude Code instance -- with zero cross-talk between environments.

## Problem Statement

Running multiple AI coding agents concurrently requires isolated, reproducible development environments. Today, spinning up a new environment involves manual steps -- creating branches, copying config, starting services -- which discourages parallelism and slows iteration. There is no way to run 5 Claude Code instances simultaneously without them stepping on each other's branches, databases, and file state.

## Proposed Solution

A single binary (`alpine`) with subcommands:

| Command | Description |
|---|---|
| `alpine create <name>` | Create an isolated dev environment |
| `alpine teardown <name>` | Auto-commit, push, then destroy the environment |
| `alpine list` | Show all active environments with status |
| `alpine attach <name>` | Reattach terminal to a running environment's Claude session |
| `alpine status <name>` | Show environment status, Claude process state, and recent output |

Each environment is fully containerized: the repo is cloned inside Docker, services run in an isolated Docker network, and code persists only via git.

### Agent-Native Design

All commands support `--json` for machine-readable output. The `create` command supports `--detach` / `-d` for non-interactive use. The `status` command enables polling for completion. This ensures agents have parity with human users.

| Flag | Commands | Purpose |
|---|---|---|
| `--json` | all | Machine-readable JSON output |
| `--detach` / `-d` | create | Return immediately after launch (don't attach) |

**Exit codes:**

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | User error (invalid name, missing config, duplicate env) |
| 2 | System error (Docker not running, network failure, timeout) |
| 3 | Partial success (environment created but install command failed) |

## Technical Approach

### Architecture

```
Host Machine
  |
  +-- alpine CLI (Go binary)
  |     +-- Cobra CLI framework
  |     +-- Shells out to: docker, docker compose, git
  |
  +-- Docker Engine
        |
        +-- Project: alpine-auth-redesign
        |     +-- Container: alpine-auth-redesign-dev-1
        |     |     - Non-root `claude` user
        |     |     - Claude CLI, git, language runtime
        |     |     - Cloned repo on feature/auth-redesign
        |     |     - claude --dangerously-skip-permissions
        |     +-- Container: alpine-auth-redesign-db-1  (postgres:16, tmpfs)
        |     +-- Container: alpine-auth-redesign-cache-1 (redis)
        |     +-- Network: alpine-auth-redesign_default (isolated bridge)
        |
        +-- Project: alpine-billing-fix
              +-- (same structure, fully isolated)
```

### Key Technical Decisions

1. **Shell out to `docker compose`**: Generate a compose YAML to a temp file and run `docker compose -p alpine-<name> -f <file> up -d`. The `docker compose` CLI is the stable public API. This avoids the unstable Compose SDK (`docker/compose/v5`), eliminates 3 heavyweight dependencies and their transitive trees, and removes the `go.mod` version-pinning workaround. Use `os/exec` for all Docker and git operations. **Never use `sh -c`** -- always pass arguments to `exec.Command` directly to prevent shell injection.

2. **Dynamic compose file generation**: The CLI generates a `docker-compose.yml` from a Go template based on `alpine.yaml` config. Simple `text/template` -- no compose-go library needed. Use the `build:` directive in the compose file so `docker compose up` handles image building and service startup in parallel.

3. **State via Docker + name prefix**: No separate state file. `alpine list` runs `docker compose ls --filter name=alpine- --format json`. State lives in Docker. The `alpine-` prefix is the discovery mechanism. Labels (`alpine.managed=true`, `alpine.name=<name>`, etc.) are set on containers for metadata retrieval but are not used for filtering, because `docker compose ls` only supports filtering by name, not arbitrary labels.

4. **SSH agent forwarding for git auth**: Forward the host's SSH agent socket into the container (platform-aware: Linux vs macOS Docker Desktop paths). Also support `GITHUB_TOKEN` / `GH_TOKEN` env vars for HTTPS auth. HTTPS via token is the recommended path (smaller blast radius than SSH agent forwarding). Fail with a clear error if neither is available.

5. **Dockerfile with Docker layer caching**: The CLI writes a Dockerfile that layers Claude CLI + git on top of the user's `base_image`. The image is tagged by content hash of the Dockerfile, so identical configs share a single image and skip the build entirely. Docker's layer cache makes rebuilds fast when only the base image changes.

6. **Teardown safety**: Never destroy a container if git push fails. Require `--force` to override. Use `git add -u` (tracked files only) instead of `git add .` to prevent committing secrets or artifacts. Run a lightweight pre-commit secret scan before pushing.

7. **`alpine-` prefix**: All compose projects use the `alpine-` prefix to avoid collisions with user's existing Docker projects.

8. **Non-root containers**: Generated Dockerfiles create a `claude` user and run as that user. This limits container escape impact and prevents system binary tampering.

9. **Security hardening**: Generated compose YAML includes `cap_drop: [ALL]`, `security_opt: [no-new-privileges:true]`, and `tmpfs` size limits. Environment variables use Docker passthrough syntax (never literal values in the compose file).

### Implementation Phases

#### Phase 1: Create + List + Status (Core Product)

The `create` command is the entire product. Everything else is secondary. This phase delivers a working tool you can use immediately.

**Project structure (flat -- expand only when needed):**

```
alpine/
  cmd/alpine/
    main.go         # Root cobra command, config loading, all subcommand registration
    create.go       # Create subcommand (orchestration + validation)
    list.go         # List subcommand
    status.go       # Status subcommand (agent-native polling)
    docker.go       # All docker/git shell-outs, compose YAML generation, Dockerfile generation
  go.mod
  go.sum
  Makefile
  alpine.yaml.example
```

4 Go files. No `internal/` packages until complexity warrants it. Name validation is a one-liner regex in `create.go`. Config struct (~15 lines) lives in `main.go`. All Docker and git operations are in `docker.go`.

**Tasks:**

- [ ] Initialize Go module and Makefile with `build`, `test`, `lint`, `install` targets
- [ ] Implement root Cobra command with `--verbose` persistent flag and `--json` persistent flag (`cmd/alpine/main.go`)
  - Use `signal.NotifyContext(context.Background(), os.Interrupt)` on the root command for context propagation
  - Set version via `-ldflags` at build time
  - Use `RunE` (not `Run`) on all commands for proper error propagation
  - Never call `os.Exit()` outside of `main.go`
  - Initialize `log/slog` logger (outputs only when `--verbose` is set)
- [ ] Implement config loading and validation (in `main.go`):
  ```go
  type Config struct {
      Install   string   `yaml:"install"`
      EnvFiles  []string `yaml:"env_files"`
      BaseImage string   `yaml:"base_image"`
      Services  []string `yaml:"services"`
  }
  ```
  - Load via `os.ReadFile("alpine.yaml")` + `gopkg.in/yaml.v3`
  - Defaults: no install command, no services, no env files, `ubuntu:24.04` base image
  - Validate: `base_image` not empty, `services` entries are recognized (`postgres`, `redis`)
  - Missing `alpine.yaml` uses defaults with a warning
- [ ] Implement name validation (in `create.go`): regex `^[a-z0-9]([a-z0-9-]{0,48}[a-z0-9])?$`. Unit test with injection payloads: `"; rm -rf /"`, `$(whoami)`, `` `id` ``, `--flag-injection`
- [x] Implement Docker health check (`docker.go`): shell out to `docker info` with a 3-second timeout via `context.WithTimeout`. On macOS, auto-launches Docker Desktop via `open -a Docker` and polls up to 60s for daemon readiness. On Linux, returns a clear error if Docker is not running.
- [ ] Implement compose YAML generation (`docker.go`):
  - Go `text/template` that produces a valid compose file
  - Dev container service: uses `build:` directive (not separate `docker build`), SSH agent mount, env vars forwarded, healthcheck
  - **Env var passthrough syntax only** -- never interpolate literal values:
    ```yaml
    environment:
      - ANTHROPIC_API_KEY   # Docker resolves from host env
      - GITHUB_TOKEN
      - GH_TOKEN
    ```
  - Service containers with **tuned health checks**:
    ```yaml
    services:
      db:
        image: postgres:16
        tmpfs:
          - /var/lib/postgresql/data:size=512M
        healthcheck:
          test: ["CMD-SHELL", "pg_isready -U postgres"]
          interval: 2s
          timeout: 3s
          retries: 15
          start_period: 5s
      cache:
        image: redis:7
        command: redis-server --save "" --appendonly no
        healthcheck:
          test: ["CMD", "redis-cli", "ping"]
          interval: 2s
          timeout: 3s
          retries: 15
          start_period: 3s
    ```
  - Labels on containers: `alpine.managed=true`, `alpine.name=<name>`, `alpine.created=<timestamp>`, `alpine.branch=<branch>`
  - SSH agent: `/run/host-services/ssh-auth.sock` on macOS, `$SSH_AUTH_SOCK` on Linux
  - Security hardening in compose:
    ```yaml
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    ```
  - Forward `ANTHROPIC_API_KEY`, `GITHUB_TOKEN`, `GH_TOKEN` from host env
- [ ] Implement Dockerfile generation (`docker.go`):
  - Hardcode `apt-get` (add apk/dnf support only when someone asks)
  - Add `--no-install-recommends` to reduce build time and image size
  - Create non-root `claude` user:
    ```dockerfile
    FROM ubuntu:24.04
    RUN apt-get update && apt-get install -y --no-install-recommends \
        git curl openssh-client ca-certificates \
        && rm -rf /var/lib/apt/lists/*
    RUN curl -fsSL https://cli.anthropic.com/install.sh | sh
    RUN useradd -m -s /bin/bash claude
    USER claude
    WORKDIR /workspace
    ```
  - Tag by content hash of Dockerfile: `alpine-dev:<first-16-chars-of-sha256>`. Skip build if image already exists.
- [ ] Implement git operations (`docker.go`):
  - `Clone(ctx, container, remoteURL, branch)` -- `git clone --depth 1 --single-branch --branch <base> <remote> /workspace`
  - `CreateBranch(ctx, container, name)` -- `git checkout -b -- feature/<name>` (use `--` to prevent flag injection)
  - `ConfigureUser(ctx, container, name, email)` -- read from host `git config`
  - `AddCommitPush(ctx, container, branch, message)` -- for teardown. Uses `git add -u` (tracked files only, never `git add .`)
  - `HasChanges(ctx, container)` -- `git status --porcelain`
- [ ] Implement exec helper (`docker.go`):
  - `run(ctx, name string, args ...string) (string, string, error)` -- unexported function wrapping `os/exec.CommandContext`
  - Always use `exec.CommandContext` with deadline -- **every call has a timeout from Phase 1**
  - Return structured error with stderr:
    ```go
    type ExecError struct {
        Command string
        Stderr  string
        Err     error
    }
    ```
  - `runAttached(ctx, name string, args ...string) error` -- for interactive terminal
- [ ] Implement create command (`cmd/alpine/create.go`):
  1. Validate name
  2. Verify Docker is running
  3. Verify git repo: walk up from cwd to find `.git`
  4. Check for duplicate: `docker compose -p alpine-<name> ps` (non-zero exit = doesn't exist)
  5. Determine base branch: current branch (or `--from` flag). Validate `--from` value: reject values starting with `-`. Error on detached HEAD without `--from`.
  6. Determine git remote URL: `origin`
  7. Validate prerequisites: `ANTHROPIC_API_KEY` set, git auth available (SSH agent socket exists or GITHUB_TOKEN set)
  8. Generate Dockerfile, check if image exists by content-hash tag, build only if needed
  9. Write compose YAML to temp dir, run `docker compose -p alpine-<name> -f <file> up -d --wait`
  10. **Clean up temp files** (Dockerfile, compose YAML) via `defer os.RemoveAll(tempDir)`
  11. Clone repo inside dev container via `docker exec`
  12. Create and checkout `feature/<name>` branch
  13. Copy env files into container (`docker cp`), then `chown` to `claude` user
  14. Run install command inside dev container (if configured)
  15. If `--detach`: print JSON status and exit. Otherwise: launch `claude --dangerously-skip-permissions` and attach terminal
  - Rollback: single defer pattern:
    ```go
    composed := false
    defer func() {
        if err != nil && composed {
            run(ctx, "docker", "compose", "-p", project, "down", "-v", "--remove-orphans", "-t", "1")
        }
    }()
    ```
- [ ] Implement signal handling: `signal.NotifyContext` on root command. First Ctrl+C cancels context and triggers deferred rollback. Second Ctrl+C force-exits via `os.Exit(1)`.
- [ ] Add flags:
  - `--from <branch>` -- base branch (default: current branch)
  - `--detach` / `-d` -- return immediately after environment is ready (don't attach to Claude)
  - `--json` -- (persistent) machine-readable JSON output for all commands
- [ ] Implement env file handling: `docker cp` each file in `env_files` into the container, then `chown claude:claude`. Warn (don't fail) if a file is missing.
- [ ] Implement list command (`cmd/alpine/list.go`):
  - Run `docker compose ls --filter name=alpine- --format json`
  - Parse JSON, display table: name, branch, status (running/stopped), created time
  - For metadata (branch, created time): run `docker ps --filter label=alpine.managed=true --format json` and correlate by project name
  - `--json` flag outputs raw JSON array
  - Handle: Docker not running (clear error), no environments (friendly message)
- [ ] Implement status command (`cmd/alpine/status.go`):
  - Verify environment exists (discover container via `docker compose -p alpine-<name> ps --format json`)
  - Show: running/stopped, branch, created time, Claude process state (running/exited + exit code)
  - Check Claude process: `docker exec alpine-<name>-dev-1 pgrep -f claude` or inspect process list
  - `--json` outputs structured status for agent polling
- [ ] Implement version: set via `-ldflags` at build time, one line in `main.go` (`rootCmd.Version = version`)
- [ ] Default timeouts on all operations from Phase 1:

  | Operation | Default timeout |
  |---|---|
  | Docker health check | 3s |
  | Image build | 5m |
  | Compose up --wait | 2m |
  | Git clone | 5m |
  | Install command | 10m |
  | Git push (teardown) | 30s |

**Success criteria:**
- `alpine create test-1` and `alpine create test-2` run simultaneously without conflicts
- Each environment has its own branch, database, and network
- Ctrl+C during create cleans up all resources
- `alpine list` shows active environments
- `alpine status test-1 --json` returns machine-readable status (agent-native)
- `alpine create test-1 -d "fix auth bug"` returns immediately (agent-native)
- Missing `alpine.yaml` uses defaults
- Missing `ANTHROPIC_API_KEY` produces a clear error before any Docker operations
- Detached HEAD without `--from` produces a clear error
- Containers run as non-root `claude` user

---

#### Phase 2: Teardown + Attach (Complete Lifecycle)

**Tasks:**

- [ ] Implement teardown command (`cmd/alpine/teardown.go`):
  1. Verify environment exists (discover via `docker compose -p alpine-<name> ps --format json`, not by constructing name)
  2. Check if dev container is running
  3. If running:
     a. Check for uncommitted changes (`git status --porcelain`)
     b. If changes: `git add -u && git commit -m "alpine: auto-save before teardown [<name>]"` (**never `git add .`**)
     c. **Pre-push secret scan**: grep staged diff for `sk-ant-`, `ghp_`, `ghs_`, `-----BEGIN`, common secret patterns. Refuse to push if detected.
     d. `git push origin feature/<name>`
     e. If push fails: print error, refuse to destroy. Print: "Push failed. Fix the issue and retry, or use `--force`."
  4. If container stopped/crashed:
     a. Attempt `docker compose -p alpine-<name> start dev`
     b. If start succeeds: proceed with step 3
     c. If start fails: warn that uncommitted work may be lost, require `--force`
  5. Destroy: `docker compose -p alpine-<name> down -v --remove-orphans -t 1` (fast stop -- no reason for 10s grace on teardown)
  6. Print summary: branch name, commit SHA (if committed), remote URL
  7. `--json` outputs structured result
- [ ] Add `--force` flag: skip commit/push, destroy immediately
- [ ] Signal handling: first Ctrl+C during push is caught and deferred (push must complete). Second Ctrl+C force-kills. Ctrl+C during `down` is allowed.
- [ ] Teardown of nonexistent environment: clear error with suggestion to run `alpine list`
- [ ] Implement terminal attach (`cmd/alpine/attach.go`):
  - Verify environment exists and dev container is running
  - Run `docker exec -it <discovered-container-name> /bin/bash` (use discovered name, not constructed)
  - For v1, simply use `os/exec` with `Stdin`/`Stdout`/`Stderr` connected to `os.Stdin`/`os.Stdout`/`os.Stderr`
  - If Claude has exited, start a shell so user can inspect/recover
- [ ] Implement `alpine exec <name> <cmd>` -- run arbitrary commands inside an environment (agent-native: agents need to run commands)
- [ ] Implement `alpine logs <name>` -- stream Claude/service output without attaching (agent-native: agents need to read output)

**Success criteria:**
- Teardown with uncommitted changes: auto-commits tracked files, pushes, then destroys
- Teardown with push failure: refuses to destroy, prints actionable error
- Teardown with detected secrets in diff: refuses to push, prints warning
- `--force` destroys without commit/push
- Teardown of stopped container: attempts restart first
- No orphaned Docker resources after teardown
- `alpine attach <name>` connects to the running environment
- `alpine exec <name> "go test ./..."` runs command and returns output (agent-native)
- `alpine logs <name>` streams output (agent-native)
- Detach and reattach works

---

#### Phase 3: Polish + Hardening

**Tasks:**

- [ ] Make timeouts configurable (via `alpine.yaml` `timeouts:` section or `--timeout` flag)
- [ ] Add config validation for unknown service names, empty base image
- [ ] Write integration tests:
  - Create two environments simultaneously, verify isolation
  - Create, make changes, teardown, verify branch pushed
  - Create, simulate push failure, verify teardown refused
  - Create, Ctrl+C midway, verify cleanup
  - Teardown a stopped container
  - Create with `--detach`, poll with `status --json`, verify completion detection
  - Pre-commit secret scan blocks push of leaked keys
  - Run tests with `t.Parallel()` where independent; ensure cleanup on failure
- [ ] Write README.md with installation, quickstart, and config reference
- [ ] Handle edge cases:
  - Running outside a git repo
  - Subdirectory of a repo (walk up to find root)
  - No git remote configured
  - Remote-only branches with `--from`
- [ ] Set up GitHub Actions CI with Docker for integration tests

**Success criteria:**
- Timeouts are configurable
- Integration test suite passes
- README covers installation, usage, and configuration
- Edge cases produce clear error messages

---

### Platform-Specific Considerations

**macOS (Docker Desktop):**
- SSH agent forwarding: mount `/run/host-services/ssh-auth.sock` and set `SSH_AUTH_SOCK=/run/host-services/ssh-auth.sock` inside container
- Docker socket: `/var/run/docker.sock`
- Validate SSH agent socket is functional (`ssh-add -l` via forwarded socket) during create; fail with clear error if not

**Linux (Docker Engine):**
- SSH agent forwarding: mount `$SSH_AUTH_SOCK` directly and set `SSH_AUTH_SOCK` to the mounted path
- Docker socket: `/var/run/docker.sock`
- Permissions: user must be in `docker` group or use rootless Docker

### Security Considerations

**Threat model:** This tool runs Claude Code with `--dangerously-skip-permissions` inside containers that have access to API keys, SSH agent, and GitHub tokens. A malicious `CLAUDE.md` or code comment in the cloned repo could instruct Claude to exfiltrate these credentials. Users must trust the repositories they clone.

- `ANTHROPIC_API_KEY` is passed via Docker env var passthrough (never written to compose YAML as literal value). Unit test asserts generated YAML never contains `sk-ant-`.
- SSH agent socket is forwarded into the container, granting the container SSH signing authority for the lifetime of the container. HTTPS auth via `GITHUB_TOKEN` is recommended over SSH as it has a smaller blast radius (scoped to GitHub only).
- Network isolation: each project gets its own bridge network, no inter-project communication. Containers have unrestricted outbound internet access (required for npm install, git clone, etc.).
- Containers run as non-root `claude` user with `cap_drop: [ALL]` and `no-new-privileges`.
- Containers are ephemeral -- destroyed on teardown, no persistent attack surface.
- Teardown uses `git add -u` (tracked files only), never `git add .`. Pre-push secret scan rejects commits containing API key patterns.
- Temp files (compose YAML, Dockerfile) are cleaned up via `defer os.RemoveAll(tempDir)` after compose up succeeds.
- `alpine.yaml` is read from the **host** filesystem (user's current directory), not from inside the container. This is a security invariant.
- Docker socket is NOT mounted into containers. Any process inside the container cannot control Docker.

### Error Handling Strategy

All errors use a structured `ExecError` type that includes the command name and stderr output. In `--verbose` mode, full stderr is displayed. In normal mode, stderr is wrapped into a user-friendly message.

| Scenario | Behavior | Exit Code |
|---|---|---|
| Docker not running (macOS) | Auto-launches Docker Desktop, polls up to 60s for daemon readiness | 2 (if launch fails or times out) |
| Docker not running (Linux) | "Docker is not running. Start Docker and try again." | 2 |
| `ANTHROPIC_API_KEY` not set | Clear error before any Docker operations | 1 |
| Git auth unavailable | "No git auth found. Set up SSH keys or export GITHUB_TOKEN." | 1 |
| Name already taken | "Environment 'foo' already exists. Run `alpine list` to see active environments." | 1 |
| Detached HEAD without `--from` | "HEAD is detached. Use `--from <branch>` to specify a base branch." | 1 |
| Not in a git repo | "Not inside a git repository." | 1 |
| Install command fails | Error message. Environment left running -- user can `alpine attach` and fix. | 3 |
| Push fails on teardown | Refuse to destroy. "Push failed. Fix and retry, or use `--force`." | 2 |
| Secrets detected in diff | Refuse to push. "Potential secrets detected in staged changes. Review and remove before teardown." | 2 |
| Ctrl+C during create | Clean up all resources created so far (single defer rollback) | 2 |
| Missing `alpine.yaml` | Use defaults with a warning | 0 |
| Operation timeout | "Operation timed out after Xs. Use `--verbose` for details." | 2 |

## Acceptance Criteria

### Functional Requirements

- [ ] `alpine create x` and `alpine create y` run simultaneously without conflicts
- [ ] Each environment has its own git branch, Docker network, database, and Claude instance
- [ ] `alpine create <name> -d` returns immediately with JSON status (agent-native)
- [ ] `alpine status <name> --json` returns Claude process state (agent-native)
- [ ] `alpine teardown <name>` auto-commits tracked files, pushes, then destroys all resources
- [ ] `alpine teardown <name>` refuses to destroy if push fails (unless `--force`)
- [ ] `alpine teardown <name>` refuses to push if secrets detected in diff
- [ ] `alpine list` shows all active environments with accurate status
- [ ] `alpine list --json` returns machine-readable output (agent-native)
- [ ] `alpine attach <name>` connects to a running environment
- [ ] Works on macOS (arm64, amd64) and Linux (amd64, arm64)
- [ ] Ctrl+C during create cleans up all resources
- [ ] Missing config file uses sensible defaults
- [ ] Containers run as non-root `claude` user

### Non-Functional Requirements

- [ ] No orphaned Docker resources after teardown
- [ ] No cross-talk between concurrent environments
- [ ] All blocking operations have default timeouts from Phase 1
- [ ] All commands support `--json` output
- [ ] Environment creation completes in <60s for warm starts (cached image, small repo, excluding install command)

### Quality Gates

- [ ] Unit tests for config parsing, name validation, compose YAML generation, Dockerfile content-hash, rollback logic, secret scanning regex
- [ ] Integration tests that create/teardown real Docker environments (requires Docker in CI)
- [ ] `go vet` and `golangci-lint` pass
- [ ] Generated compose YAML never contains literal API key values (unit test)

## Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| Go | 1.22+ | Build language |
| `github.com/spf13/cobra` | v1.9.1 | CLI framework |
| `gopkg.in/yaml.v3` | latest | Config file parsing |
| Docker CLI (`docker`, `docker compose`) | 24.0+ | Runtime -- invoked via `os/exec` |
| Git | 2.x | Repo operations on host |

**2 Go dependencies in Phase 1.** No Docker SDK, no Compose SDK, no Viper, no moby/term (add in Phase 2 if `docker exec -it` proves insufficient).

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SSH agent forwarding fails on macOS | Medium | High | Support both SSH and HTTPS (GITHUB_TOKEN) auth. Validate socket during create. Document macOS setup. |
| Claude CLI install script changes | Medium | Medium | Pin Claude CLI version in the Dockerfile. |
| `docker compose` CLI output format changes | Low | Medium | Parse `--format json` output, which is the stable contract. |
| Large repos slow to clone | High | Low | Use `--depth 1 --single-branch` shallow clones. |
| Prompt injection via malicious repo content | Medium | Critical | Document threat model. Users must trust cloned repos. Consider `--safe` mode in future. |
| Secret exfiltration via auto-commit | Low | Critical | Use `git add -u` only. Pre-push secret scan. Never `git add .`. |
| Container escape with root privileges | Low | Critical | Non-root user, cap_drop: ALL, no-new-privileges from Phase 1. |

## Future Considerations

- `alpine stop <name>` / `alpine start <name>` -- pause without destroying
- `alpine init` -- generate `alpine.yaml` with project type detection
- `.goreleaser.yml` for cross-platform release automation
- Pre-built images published to a registry
- Resource limits per environment (CPU, memory caps via compose `deploy.resources`)
- Configurable branch prefix (default `feature/`, configurable in `alpine.yaml`)
- `--safe` mode that runs Claude without `--dangerously-skip-permissions`
- Scoped/short-lived API tokens instead of forwarding long-lived credentials
- Network egress logging for security monitoring
- Env file filtering to strip production secrets before copying into container
- `apk`/`dnf` package manager support for non-Debian base images
- MySQL service support
- Host-side reference clone (`--reference`) for faster cloning of large repos

## References

### Brainstorm
- `docs/brainstorms/2026-02-11-ephemeral-dev-cli-brainstorm.md`

### Libraries
- [Cobra CLI framework](https://github.com/spf13/cobra)
- [moby/term](https://github.com/moby/term) (Phase 2+)

### Patterns
- Docker CLI source: terminal attach implementation at `docker/cli/cli/command/container/exec.go`
- Tilt (`tilt-dev/tilt`): pragmatic hybrid of Docker SDK + shell-out
- lazydocker (`jesseduffield/lazydocker`): OSCommand pattern for shell-outs with `CommandTemplatesConfig`
- devcontainers CLI: lifecycle hooks (onCreateCommand, postStartCommand), Docker Compose integration, PTY handling

### Research (from deepen-plan)
- Cobra v1.9.1 Context7: `RunE` for error returns, `PersistentFlags` for global flags, `MarkFlagRequired`, version via `-ldflags`
- moby/term Context7: PTY via `creack/pty`, raw mode via `term.MakeRaw`, SIGWINCH for resize, `defer term.Restore` for cleanup
- Docker Compose: `docker compose ls` only supports `--filter name=`, not label filtering. Use `docker ps --filter label=` for container-level metadata.
- Go signal handling: `signal.NotifyContext` (Go 1.16+), `cmd.Cancel`/`cmd.WaitDelay` (Go 1.20+) for child process cleanup
- SSH agent on macOS Docker Desktop: socket at `/run/host-services/ssh-auth.sock`, requires explicit volume mount
