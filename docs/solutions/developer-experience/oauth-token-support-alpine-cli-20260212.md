---
module: System
date: 2026-02-12
problem_type: developer_experience
component: tooling
symptoms:
  - "ANTHROPIC_API_KEY required even when user has Claude OAuth token"
  - "No way to authenticate containers without exporting an API key"
root_cause: incomplete_setup
resolution_type: code_fix
severity: medium
tags: [oauth, authentication, docker, claude-oauth-token, api-key]
---

# Troubleshooting: Container Auth Requires ANTHROPIC_API_KEY, No OAuth Support

## Problem
The `alpine create` command hard-required `ANTHROPIC_API_KEY` to be set on the host, failing immediately if missing. Users who authenticate via Claude OAuth (the standard `claude login` flow) had no way to use Alpine without also exporting an API key.

## Environment
- Module: System (Alpine CLI authentication)
- Affected Component: `cmd/alpine/create.go` (prerequisite validation), `cmd/alpine/docker.go` (compose template)
- Date: 2026-02-12

## Symptoms
- `alpine create` fails with "ANTHROPIC_API_KEY is not set" even when user is logged in via Claude OAuth
- No mechanism to forward OAuth credentials to containers
- Users forced to obtain and manage a separate API key

## What Didn't Work

**Direct solution:** The problem was identified and fixed on the first attempt.

## Solution

Three changes across two files:

**1. Remove hard API key requirement** (`cmd/alpine/create.go:137-140`):
```go
// Before (broken):
if os.Getenv("ANTHROPIC_API_KEY") == "" {
    return userErr("ANTHROPIC_API_KEY is not set")
}

// After (fixed):
// Removed entirely. Auth handled at container launch time.
```

**2. Add CLAUDE_CODE_OAUTH_TOKEN to compose passthrough** (`cmd/alpine/docker.go:298-299`):
```yaml
# Before:
environment:
  - ANTHROPIC_API_KEY
  - GITHUB_TOKEN

# After:
environment:
  - ANTHROPIC_API_KEY
  - CLAUDE_CODE_OAUTH_TOKEN
  - GITHUB_TOKEN
```

**3. Run `claude setup-token` when no OAuth token is set** (`cmd/alpine/create.go:370-377`):
```go
// Before launching Claude, authenticate if needed:
if os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") == "" {
    setupErr := runAttached(ctx, "docker", "exec", "-it", container, "claude", "setup-token")
    if setupErr != nil {
        return sysErr(fmt.Sprintf("claude setup-token failed: %v", setupErr))
    }
}
```

## Why This Works

1. **Root cause:** Auth was hardcoded to require `ANTHROPIC_API_KEY` as the only authentication method. Claude Code supports OAuth tokens natively via `CLAUDE_CODE_OAUTH_TOKEN`, but Alpine had no pathway for this.
2. **The solution** adds three auth paths: (a) `CLAUDE_CODE_OAUTH_TOKEN` set on host passes through to container automatically, (b) `ANTHROPIC_API_KEY` still works as a fallback via passthrough, (c) if neither is set, `claude setup-token` runs interactively inside the container so the user can authenticate on the spot.
3. **Security preserved:** Both env vars use Docker passthrough syntax (no literal values in generated YAML).

## Prevention

- When adding new auth methods to Alpine, always use Docker environment passthrough syntax (`- VAR_NAME` not `- VAR_NAME=value`)
- Test the create flow with no auth env vars set to verify the interactive fallback works
- Keep the compose template's security comment updated when adding new env vars

## Related Issues

- See also: [container-root-user-claude-setup-alpine-cli-20260212.md](./container-root-user-claude-setup-alpine-cli-20260212.md) — container runs as root and needs `setup-token` to pre-register OAuth credentials
- See also: [claude-auth-onboarding-skip-alpine-cli-20260212.md](./claude-auth-onboarding-skip-alpine-cli-20260212.md) — fixes three issues preventing programmatic auth: onboarding flag, .env loading, setup-token misuse
