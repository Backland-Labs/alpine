---
module: Alpine CLI
date: 2026-02-12
problem_type: logic_error
component: tooling
symptoms:
  - "Git worktree .git pointer file breaks after docker cp into container"
  - "Container git commands fail with missing gitdir reference"
  - "git checkout -b fails inside container after worktree copy"
root_cause: config_error
resolution_type: workflow_improvement
severity: high
tags: [git-worktree, docker-cp, container-isolation, architecture]
---

# Troubleshooting: Git Worktrees Are Incompatible with docker cp

## Problem
Using `git worktree add` to create a repo copy on the host, then `docker cp` into a container, results in a broken git repository inside the container because worktrees use `.git` pointer files, not standalone `.git` directories.

## Environment
- Module: Alpine CLI (create command)
- Affected Component: `cmd/alpine/docker.go` (gitCreateWorktree, copyToContainer), `cmd/alpine/create.go` (steps 9-10)
- Date: 2026-02-12

## Symptoms
- After `docker cp` of a worktree into a container, the `.git` file contains a host path like `gitdir: /Users/max/code/alpine/.git/worktrees/my-feature`
- All git operations inside the container fail because the referenced path does not exist
- `git checkout -b`, `git commit`, `git push` all break with missing object store errors

## What Didn't Work

**Attempted Solution 1:** `git worktree add --detach .worktrees/<name> <branch>` + `docker cp`
- **Why it failed:** Git worktrees are not standalone repositories. The `.git` entry is a text file containing `gitdir: <path-to-main-repo>/.git/worktrees/<name>`. After `docker cp`, this pointer references a host filesystem path that does not exist inside the container. The worktree shares the parent repo's object store, refs, and config -- none of which are copied.

**Attempted Solution 2:** `git clone --local --branch <branch> <gitRoot> /tmp/alpine-<name>` + `docker cp`
- **Why it failed:** This actually works (creates a standalone `.git` directory), but introduces unnecessary complexity. The local clone sets `origin` to the host filesystem path, requiring a `git remote set-url origin <real-url>` fixup inside the container. This adds a temp directory, cleanup defer, and remote fixup step -- all avoidable.

## Solution

Clone directly from the remote inside the container. The SSH agent is already forwarded via the Docker Compose volume mount, so authentication works transparently.

**Code changes:**

```go
// Before (broken -- worktree approach):
worktreePath, err := gitCreateWorktree(ctx, gitRoot, name, branch)
defer gitRemoveWorktree(ctx, gitRoot, worktreePath)
copyToContainer(ctx, worktreePath, container, "/workspace")

// After (fixed -- remote clone inside container):
remoteURL, err := gitGetRemoteURL(ctx, "origin")
gitClone(ctx, container, remoteURL, branch)
```

Additional changes for auth to work inside the container:

```dockerfile
# Pre-populate GitHub SSH host keys (no interactive prompt)
RUN mkdir -p /home/claude/.ssh \
    && ssh-keyscan -t ed25519,rsa github.com >> /home/claude/.ssh/known_hosts 2>/dev/null \
    && chown -R claude:claude /home/claude/.ssh

# Git credential helper for HTTPS remotes using GITHUB_TOKEN
RUN git config --global credential.helper \
    '!f() { echo "username=x-access-token"; echo "password=${GITHUB_TOKEN:-${GH_TOKEN}}"; }; f'
```

## Why This Works

1. **Root cause:** Git worktrees are designed for shared-filesystem use. The `.git` pointer file creates a dependency on the parent repo's `.git` directory, which makes worktrees fundamentally incompatible with cross-boundary copies (`docker cp`, `scp`, etc.).

2. **Why remote clone is correct:** The container is meant to be fully isolated. Cloning from the remote is the cleanest expression of that isolation -- the container's git state is completely self-contained from creation. Origin URL is correct, object store is complete, no fixups needed.

3. **Why this is simpler:** The remote clone approach eliminates five operations from the pipeline: host-side worktree creation, worktree cleanup defer, `docker cp` of workspace, `chown -R` of workspace (clone runs as `claude` user so ownership is correct by default), and remote URL fixup.

4. **Auth is not a new constraint:** The container already needs SSH agent access for `git push`. Requiring it at clone time surfaces auth failures earlier (at create time) rather than later (at push time), which is better UX.

## Prevention

- **Never `docker cp` a git worktree.** Worktrees are filesystem-local constructs. Always verify that `.git` is a directory (not a file) before copying a repo across boundaries.
- **Prefer the simplest approach that satisfies all constraints.** Before adding intermediate steps (worktree + copy + fixup), check if the end-state tooling (remote clone) already handles the requirement.
- **Use an architecture review agent** to evaluate approaches before implementing. The worktree bug was caught during review, not during initial implementation.

## Related Issues

- See also: [chown-permission-denied-docker-cp-alpine-cli-20260212.md](../runtime-errors/chown-permission-denied-docker-cp-alpine-cli-20260212.md) - `chown` fails after `docker cp` when container drops `DAC_OVERRIDE` capability
