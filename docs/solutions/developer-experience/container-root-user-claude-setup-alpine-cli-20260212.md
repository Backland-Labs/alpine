---
module: System
date: 2026-02-12
problem_type: developer_experience
component: tooling
symptoms:
  - "root@container:/workspace# claude → --dangerously-skip-permissions cannot be used with root/sudo privileges"
  - "Claude Code requires interactive setup/login inside container despite CLAUDE_CODE_OAUTH_TOKEN being set"
root_cause: incomplete_setup
resolution_type: code_fix
severity: high
tags: [docker, dockerfile, user-root, claude-code, setup-token, oauth, container-auth]
---

# Troubleshooting: Container Runs as Root and Claude Code Requires Interactive Setup

## Problem
After `alpine create`, the container drops the user into a root shell. Claude Code refuses to run because `bypassPermissions` mode (set in project settings) is blocked for root. Even after fixing the user, Claude Code still requires interactive login because macOS Keychain credentials don't transfer to Linux containers and the OAuth token isn't automatically registered.

## Environment
- Module: System (Alpine CLI container setup)
- Affected Component: `cmd/alpine/docker.go` (Dockerfile generation), `cmd/alpine/create.go` (container setup workflow)
- Date: 2026-02-12

## Symptoms
- Container prompt shows `root@<name>-dev:/workspace#` instead of `claude@<name>-dev:/workspace$`
- Running `claude` inside container errors: `--dangerously-skip-permissions cannot be used with root/sudo privileges for security reasons`
- After fixing the root user issue, `claude` still launches the interactive setup/login sequence
- `CLAUDE_CODE_OAUTH_TOKEN` is present in the container environment (via compose passthrough) but Claude Code doesn't use it automatically for auth

## What Didn't Work

**Direct solution:** Both root causes were identified from the code on the first attempt.

## Solution

Three changes across two files:

**1. Fix Dockerfile to end as `claude` user** (`cmd/alpine/docker.go:533-537`):
```go
// Before (broken) — Dockerfile ends with:
USER root
RUN ln -s /home/claude/.local/bin/claude /usr/local/bin/claude
WORKDIR /workspace

// After (fixed) — added ownership and user switch:
USER root
RUN ln -s /home/claude/.local/bin/claude /usr/local/bin/claude
WORKDIR /workspace
RUN chown claude:claude /workspace
USER claude
```

**2. Explicit `--user claude` on interactive shell exec** (`cmd/alpine/create.go:418`):
```go
// Before:
shellErr := runInteractive("docker", "exec", "-it", "-w", "/workspace", container, "/bin/bash")

// After:
shellErr := runInteractive("docker", "exec", "-it", "--user", "claude", "-w", "/workspace", container, "/bin/bash")
```

**3. Pre-register OAuth token via `claude setup-token`** (`cmd/alpine/create.go:382-394`):
```go
// New Step 18: pipe the OAuth token into claude setup-token inside the container
_, _, setupErr := run(ctx, "docker", "exec", container,
    "sh", "-c", `[ -n "$CLAUDE_CODE_OAUTH_TOKEN" ] && echo "$CLAUDE_CODE_OAUTH_TOKEN" | claude setup-token 2>/dev/null; true`)
```

## Why This Works

1. **Root user bug:** The generated Dockerfile switched to `USER root` to create a symlink at `/usr/local/bin/claude` but never switched back. Since Docker images inherit the last `USER` directive as the default, all `docker exec` commands ran as root. Adding `USER claude` at the end restores the intended non-root default.

2. **Auth not pre-configured:** On macOS, Claude Code stores OAuth credentials in the Keychain. Copying `~/.claude/` into the container transfers settings and config but NOT auth credentials. The `CLAUDE_CODE_OAUTH_TOKEN` env var is available in the container (via compose passthrough) but Claude Code doesn't consume it automatically for authentication — it needs to be registered via `claude setup-token`. Piping the token from the container's own environment into `setup-token` during creation completes the auth flow non-interactively.

3. **Non-fatal design:** The `setup-token` step uses `; true` to ensure it never blocks container creation. If the token isn't set or `setup-token` fails, the user can still authenticate manually.

## Prevention

- Always ensure generated Dockerfiles end with the intended runtime `USER` directive — audit the last `USER` line after any Dockerfile changes
- When adding auth mechanisms that work on macOS via Keychain, remember that Linux containers need explicit credential registration
- Test `alpine create` end-to-end: verify the shell prompt shows `claude@` not `root@`, and that `claude` launches without a setup sequence

## Related Issues

- See also: [oauth-token-support-alpine-cli-20260212.md](./oauth-token-support-alpine-cli-20260212.md) — adding `CLAUDE_CODE_OAUTH_TOKEN` passthrough to the compose template
