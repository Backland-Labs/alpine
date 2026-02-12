---
module: System
date: 2026-02-12
problem_type: developer_experience
component: tooling
symptoms:
  - "Claude Code prompts for interactive onboarding (theme, auth) inside container despite CLAUDE_CODE_OAUTH_TOKEN being set"
  - "CLAUDE_CODE_OAUTH_TOKEN not available in container when only set in .env file"
  - "claude setup-token piped incorrectly; setup-token generates tokens, does not consume them"
root_cause: incomplete_setup
resolution_type: code_fix
severity: high
tags: [oauth, authentication, docker, claude-code, onboarding, dotenv, container-auth, setup-token]
---

# Troubleshooting: Claude Code Requires Manual Auth in Container Despite OAuth Token

## Problem
After `alpine create`, Claude Code inside the container still prompts for interactive onboarding (theme selection, authentication method) even when `CLAUDE_CODE_OAUTH_TOKEN` is set. Three separate issues combined to break programmatic authentication.

## Environment
- Module: System (Alpine CLI create workflow)
- Affected Component: `cmd/alpine/create.go` (Steps 14, 18), `cmd/alpine/docker.go` (loadDotEnv)
- Date: 2026-02-12

## Symptoms
- Running `claude` inside the container triggers the full onboarding wizard (theme picker, auth prompt)
- Warning: `CLAUDE_CODE_OAUTH_TOKEN is not set` even when token exists in project `.env` file
- `claude setup-token` step silently fails (suppressed by `2>/dev/null; true`)

## What Didn't Work

**Attempted Solution 1:** Pipe token into `claude setup-token` (original Step 18)
```go
run(ctx, "docker", "exec", container,
    "sh", "-c", `[ -n "$CLAUDE_CODE_OAUTH_TOKEN" ] && echo "$CLAUDE_CODE_OAUTH_TOKEN" | claude setup-token 2>/dev/null; true`)
```
- **Why it failed:** `claude setup-token` is designed to *generate* long-lived tokens via interactive OAuth login, not *consume* them from stdin. The command silently failed, suppressed by `2>/dev/null; true`.

**Attempted Solution 2:** Rely solely on compose environment passthrough
```yaml
environment:
  - CLAUDE_CODE_OAUTH_TOKEN
```
- **Why it failed:** Two issues: (1) Claude Code still requires `hasCompletedOnboarding: true` in `~/.claude.json` to skip the interactive wizard even when the token is set ([anthropics/claude-code#8938](https://github.com/anthropics/claude-code/issues/8938)). (2) The passthrough only works if the variable is in the Go process environment; tokens in `.env` files aren't loaded automatically.

## Solution

Three changes across two files:

**1. Copy `~/.claude.json` from host into container** (`cmd/alpine/create.go` Step 14):
```go
// Before: only copied ~/.claude/ directory
// After: also copy ~/.claude.json (onboarding flags, theme, preferences)
hostClaudeJSON := filepath.Join(homeDir, ".claude.json")
if _, statErr := os.Stat(hostClaudeJSON); statErr == nil {
    copyPathToContainer(ctx, container, hostClaudeJSON, "/home/claude/.claude.json")
}
```

**2. Ensure `hasCompletedOnboarding` flag and verify token** (`cmd/alpine/create.go` Step 18):
```go
// Ensure ~/.claude.json exists with hasCompletedOnboarding=true
run(ctx, "docker", "exec", container,
    "sh", "-c", `f=/home/claude/.claude.json
if [ -f "$f" ]; then
  grep -q '"hasCompletedOnboarding"' "$f" || \
    sed -i '1s/^{/{"hasCompletedOnboarding":true,/' "$f"
else
  echo '{"hasCompletedOnboarding":true}' > "$f"
fi`)

// Verify the OAuth token is available in the container
tokenCheck, _, _ := run(ctx, "docker", "exec", container,
    "sh", "-c", `[ -n "$CLAUDE_CODE_OAUTH_TOKEN" ] && echo "set" || echo "unset"`)
```

**3. Load `.env` from git root into process environment** (`cmd/alpine/docker.go`):
```go
// loadDotEnv reads a .env file and sets variables not already present
// in the process environment, so compose passthrough picks them up.
func loadDotEnv(path string) error {
    data, _ := os.ReadFile(path)
    for _, line := range strings.Split(string(data), "\n") {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") { continue }
        line = strings.TrimPrefix(line, "export ")
        key, value, ok := strings.Cut(line, "=")
        if !ok { continue }
        // Strip quotes, only set if not already exported
        if os.Getenv(key) == "" {
            os.Setenv(key, value)
        }
    }
    return nil
}
```

Called before compose up in `create.go`:
```go
envPath := filepath.Join(gitRoot, ".env")
if _, statErr := os.Stat(envPath); statErr == nil {
    loadDotEnv(envPath)
}
```

## Why This Works

1. **Root cause (onboarding):** Claude Code has a known issue ([#8938](https://github.com/anthropics/claude-code/issues/8938)) where `CLAUDE_CODE_OAUTH_TOKEN` alone is insufficient -- the CLI still walks through the interactive onboarding flow unless `hasCompletedOnboarding: true` is set in `~/.claude.json`. Copying the host's `~/.claude.json` (which has this flag) into the container solves this. The `sed` fallback handles cases where the host file doesn't exist.

2. **Root cause (.env loading):** Docker compose passthrough syntax (`- CLAUDE_CODE_OAUTH_TOKEN`) reads from the process environment of the `docker compose up` invocation. When the token is only in a `.env` file (not exported in the shell), the Go process doesn't have it. `loadDotEnv` bridges this gap by reading `.env` into the process environment before compose runs.

3. **Root cause (setup-token misuse):** `claude setup-token` is a token *generator* (run interactively on the host to create a long-lived token), not a token *consumer*. The old code piped the token to it on stdin, which does nothing. The correct flow is: user runs `claude setup-token` once on the host, exports the result as `CLAUDE_CODE_OAUTH_TOKEN`, and the env var is all that's needed (plus the onboarding flag).

## Prevention

- When adding headless/CI Claude Code auth: always set BOTH `CLAUDE_CODE_OAUTH_TOKEN` env var AND `hasCompletedOnboarding: true` in `~/.claude.json`
- Never suppress errors with `2>/dev/null; true` in auth-critical steps -- log failures explicitly
- When using Docker compose passthrough for env vars, ensure the source process has access to `.env` files
- Test container auth by running `claude -p "hello" --max-budget-usd 0.10` to verify non-interactive auth works

## Related Issues

- See also: [oauth-token-support-alpine-cli-20260212.md](./oauth-token-support-alpine-cli-20260212.md) -- added `CLAUDE_CODE_OAUTH_TOKEN` to compose passthrough
- See also: [container-root-user-claude-setup-alpine-cli-20260212.md](./container-root-user-claude-setup-alpine-cli-20260212.md) -- container user and setup-token prerequisites
- Upstream: [anthropics/claude-code#8938](https://github.com/anthropics/claude-code/issues/8938) -- `setup-token`/`CLAUDE_CODE_OAUTH_TOKEN` not sufficient without onboarding flag
