---
module: Alpine CLI
date: 2026-02-12
problem_type: runtime_error
component: tooling
symptoms:
  - "chown: cannot access '/home/claude/.claude': Permission denied"
  - "warning: failed to copy ~/.claude after docker cp into container"
root_cause: missing_permission
resolution_type: config_change
severity: medium
tags: [docker, capabilities, dac-override, chown, permission-denied, docker-cp]
---

# Troubleshooting: chown Permission Denied After docker cp Into Capability-Restricted Container

## Problem
After `docker cp` copies `~/.claude` into a container that drops all Linux capabilities, the subsequent `chown -R claude:claude` fails with "Permission denied" because root inside the container lacks `DAC_OVERRIDE`.

## Environment
- Module: Alpine CLI (create command)
- Affected Component: `cmd/alpine/docker.go` (compose template `cap_add`, `copyPathToContainer`)
- Date: 2026-02-12

## Symptoms
- `warning: failed to copy ~/.claude: chown failed: chown: cannot access '/home/claude/.claude': Permission denied`
- `~/.claude` config not present in container after `alpine create`, requiring manual Claude Code setup

## What Didn't Work

**Direct solution:** The problem was identified and fixed on the first attempt.

The root cause was clear from the error message once the container's capability configuration was examined.

## Solution

Added `DAC_OVERRIDE` to the compose template's `cap_add` list.

**Code changes:**

```yaml
# Before (broken):
cap_drop:
  - ALL
cap_add:
  - CHOWN
  - FOWNER

# After (fixed):
cap_drop:
  - ALL
cap_add:
  - CHOWN
  - DAC_OVERRIDE
  - FOWNER
```

File: `cmd/alpine/docker.go` (compose template constant)

## Why This Works

1. **Root cause:** The compose template uses `cap_drop: ALL` for security hardening, then selectively adds back only `CHOWN` and `FOWNER`. When `docker cp` copies files from the host into the container, the files retain the host user's UID (e.g., UID 501 on macOS) and permissions (e.g., mode 700). Root inside the container (UID 0) is a different user than the file owner (UID 501). Without `DAC_OVERRIDE`, root cannot traverse or access directories owned by other UIDs when those directories have restrictive permissions.

2. **Why `CHOWN` and `FOWNER` are not enough:** `CHOWN` allows changing file ownership. `FOWNER` bypasses checks that require the process UID to match the file UID (e.g., `chmod`). But neither grants the ability to read or traverse directories with restrictive permissions. That requires `DAC_OVERRIDE`, which bypasses file read/write/execute permission checks.

3. **Why `DAC_OVERRIDE` is acceptable:** These containers are ephemeral development environments, not production workloads. Root already runs inside the container with `CHOWN` and `FOWNER`. Adding `DAC_OVERRIDE` is incrementally more privileged but standard for containers that manage files with arbitrary ownership after `docker cp`.

## Prevention

- When using `cap_drop: ALL` in containers that run `docker cp` followed by `chown`, always include `DAC_OVERRIDE` in `cap_add`. The `docker cp` command preserves host UIDs, which root inside the container cannot access without this capability.
- Test file copy operations with restrictive source permissions (mode 700) to catch capability gaps early.
- Document the purpose of each capability in comments so future changes don't accidentally remove required ones.

## Related Issues

- See also: [git-worktree-docker-cp-broken-alpine-cli-20260212.md](../logic-errors/git-worktree-docker-cp-broken-alpine-cli-20260212.md) - Another `docker cp` issue in the same create flow (worktree pointer files breaking after copy)
