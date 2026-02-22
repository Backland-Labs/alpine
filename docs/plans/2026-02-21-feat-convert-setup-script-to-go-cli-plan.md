# feat: convert setup-sprite-opencode.sh to a basic Go CLI with sprites-go

Type: feature
Detail level: comprehensive

## Overview

Build a parity-first Go CLI that replaces `setup-sprite-opencode.sh` while preserving current onboarding behavior. The new CLI uses `github.com/superfly/sprites-go` for sprite lifecycle, command execution, and filesystem operations.

## Problem statement / motivation

`setup-sprite-opencode.sh:1` currently handles argument parsing, preflight checks, sprite lifecycle, remote bootstrap, file transfer, git branch setup, and interactive launch in one large Bash file. This makes behavior difficult to test and risky to evolve.

Migrating to Go provides typed SDK calls, clearer error boundaries, deterministic exit-code contracts, and better maintainability for future setup changes.

## Stakeholders

- Developers running local setup for Sprite coding sessions.
- Repository maintainers responsible for onboarding reliability.
- Security-sensitive users relying on safe handling of `auth.json`, `.env`, and `~/.claude` contents.

## Detailed background

- Current usage contract is documented in `README.md:3` and `README.md:24`.
- Current behavior to preserve is implemented in `setup-sprite-opencode.sh:83`, `setup-sprite-opencode.sh:119`, `setup-sprite-opencode.sh:173`, `setup-sprite-opencode.sh:206`, `setup-sprite-opencode.sh:337`, `setup-sprite-opencode.sh:366`, and `setup-sprite-opencode.sh:417`.
- No `docs/brainstorms/` notes exist to reuse.
- No `docs/solutions/` entries exist yet, so this plan establishes the first migration baseline.

## Research notes

### Key references

- Local baseline: `setup-sprite-opencode.sh:1`, `README.md:1`
- SDK repository: `https://github.com/superfly/sprites-go`
- SDK files: `https://github.com/superfly/sprites-go/blob/main/management.go`, `https://github.com/superfly/sprites-go/blob/main/exec.go`, `https://github.com/superfly/sprites-go/blob/main/filesystem.go`, `https://github.com/superfly/sprites-go/blob/main/client.go`, `https://github.com/superfly/sprites-go/blob/main/test-cli/main.go`
- API docs: `https://docs.sprites.dev/api/v001-rc30/`
- DeepWiki index: `https://deepwiki.com/superfly/sprites-go`

### Reusable learnings from `docs/solutions/`

- None found. `docs/solutions/` is not present in this repository.

### Open questions still needing decisions

- Token precedence contract: use `SPRITES_TOKEN` first, then credential-file fallback; if both differ, warn and prefer `SPRITES_TOKEN`.
- `--org` fail-closed behavior: if org scoping cannot be guaranteed for required operations, should command hard-fail by default.
- Cleanup default: on failure after sprite creation, default to delete-or-keep policy.
- Transfer overwrite policy: backup-before-overwrite plus restore-on-failure retention window.

## Acceptance criteria

- [x] Argument parsing is deterministic: `--branch` required, `--org` optional, `--help` exits `0`, invalid arguments exit `2`.
- [x] Preflight runs before any remote mutation and validates git context plus required local assets (`auth.json`, `~/.config/opencode`, `~/.claude`, `.env`).
- [x] Branch input passes `git check-ref-format --branch` validation before create/bootstrap/transfer steps.
- [x] Core workflow uses `sprites-go` APIs for create/list/exec/filesystem operations (no `sprite` CLI fallback for core steps).
- [x] Naming preserves current slug + adjective/noun logic, retries boundedly on collision/transient errors, and fails with actionable output after max attempts.
- [x] Bootstrap is idempotent for supported shells (`bash`, `zsh`) and verifies `opencode` and `ast-grep` presence.
- [x] Transfer writes only to an explicit destination allowlist, rejects traversal/symlink-escape paths, and enforces secret file modes (`0600`) and directory modes (`0700`) where applicable.
- [x] Overwrite behavior is deterministic: backup-before-overwrite and restore-on-failure are implemented and tested.
- [ ] Git setup follows the branch decision table in this plan and is tested for: local branch exists, remote-only branch exists, and missing branch with/without resolvable base branch.
- [x] Non-TTY mode never attempts interactive attach and always prints `sprite_id` plus a reconnect command.
- [x] Exit code mapping is stable and documented (`0`, `1`, `2`, `3`, `4`, `5`) with tests for each failure class.
- [x] `README.md` is updated to describe Go CLI usage and migration notes from script usage.

## Proposed solution

Implement a single-command Go CLI with parity-first flow:

1. Parse/validate flags and environment.
2. Run local preflight checks.
3. Resolve deterministic sprite name with collision handling.
4. Create sprite via `CreateSprite`.
5. Bootstrap remote tooling via `CommandContext` (`opencode`, `ast-grep`, profile updates).
6. Transfer local artifacts via `Filesystem` APIs with safe path handling.
7. Clone/fetch/checkout branch in remote repo.
8. Launch interactive session in TTY mode, or print deterministic non-TTY reconnect output.

Suggested package layout:

- `cmd/setup-sprite-opencode/main.go`: entrypoint and exit-code mapping.
- `internal/cli/config.go`: arguments, validation, auth/org resolution.
- `internal/flow/run.go`: orchestration state machine.
- `internal/sprites/client.go`: typed wrappers for SDK interactions.
- `internal/remote/bootstrap.go`: install/verify tooling and shell profile idempotency.
- `internal/remote/transfer.go`: tar/write helpers, path canonicalization, permissions.
- `internal/remote/gitsetup.go`: clone/fetch/branch decision implementation.
- `internal/remote/launch.go`: TTY attach and non-TTY contract output.

## Technical considerations

- SDK pinning: `sprites-go` currently has no stable tag; pin a commit/pseudo-version in `go.mod`.
- Deprecated API avoidance: use `CreateSprite`, `ListSprites`/`ListAllSprites`, and `SetTTYSize` (not deprecated methods).
- Auth contract: resolve token precedence deterministically and validate credentials before remote mutations.
- Org contract: if `--org` is provided but end-to-end org enforcement is uncertain, fail closed with remediation guidance.
- Path safety: canonicalize remote destinations and reject absolute paths, `..`, and symlink escapes.
- Command safety: avoid unsafe interpolation for branch/org/path values in remote command construction.
- Idempotency: shell profile changes must be append-once and rerunnable.
- Retries/timeouts: use context timeouts and bounded retries for transient failures only.
- Observability: structured logs include `phase`, `step`, `sprite_id`, `attempt`, and redact secrets centrally.

## Success metrics

- 100% pass on parity matrix scenarios compared with current script behavior.
- 0 secret leaks in logs during unit/integration/e2e verification.
- Deterministic outcomes across branch decision scenarios.
- Successful run on a clean setup plus successful rerun proving idempotency.

## Dependencies and risks

- Dependency: `github.com/superfly/sprites-go` API semantics for auth/org, exec, and filesystem.
- Dependency: remote runtime capabilities (`git`, shell utilities, package install prerequisites).
- Risk: ambiguous auth source selection can target unintended account/org.
- Risk: path traversal or symlink collisions can cause unintended remote writes.
- Risk: partial failures can leave orphaned sprites or half-applied config.
- Risk: TTY/non-TTY behavior divergence can break local and CI usage.
- Risk: default-branch resolution can vary if remote HEAD metadata is absent.

Mitigations:

- Define explicit auth precedence and org fail-closed behavior.
- Implement allowlist + canonicalization for transfers, with backup/restore.
- Add phase-scoped rollback matrix with cleanup guidance.
- Test interactive and non-interactive flows separately.

## Implementation plan

### Phase 1 - Scaffold and contracts

- [x] Create Go module and CLI entrypoint in `cmd/setup-sprite-opencode/`.
- [x] Implement argument parsing (`--branch`, `--org`, `--help`) and exit-code taxonomy.
- [x] Implement preflight checks with deterministic error ordering.
- [x] Implement auth precedence and org resolution contract.
- [x] Implement slug + random name generation with bounded collision retries.

### Branch decision table (normative behavior)

1. If local branch `<branch>` exists in remote checkout, check out `<branch>`.
2. Else if `origin/<branch>` exists, create local tracking branch from `origin/<branch>`.
3. Else resolve base branch in order: `origin/HEAD`, then `origin/main`, then `origin/master`.
4. If no base branch resolves, fail with remediation text and no implicit guessing.
5. Emit selected branch path in logs for deterministic debugging.

### Phase 2 - Sprite and remote workflow

- [x] Implement SDK wrapper for create/list/get/exec/filesystem operations.
- [x] Implement remote bootstrap with idempotent profile updates and tool verification.
- [x] Implement transfer allowlist and canonicalization checks.
- [x] Implement traversal and symlink-escape rejection for all writes.
- [x] Implement backup-before-overwrite and restore-on-failure behavior.
- [x] Implement remote permission enforcement for sensitive files/directories.
- [x] Implement git clone/fetch/branch setup per decision table.

### Phase 3 - Launch, rollback, and docs

- [x] Implement interactive TTY launch path.
- [x] Implement deterministic non-TTY output contract (`sprite_id`, reconnect command, status line).
- [x] Implement phase-scoped rollback behavior and cleanup-failure guidance.
- [x] Implement structured logging with redaction checks.
- [x] Update `README.md` for Go CLI usage and migration notes.

### Phase 4 - Verification and release readiness

- [x] Add unit tests for parsing, validation, naming, retries, and exit-code mapping.
- [ ] Add integration tests for preflight failure ordering and branch matrix cases.
- [x] Add tests for traversal rejection, symlink collisions, and backup/restore boundaries.
- [ ] Add tests proving no secrets from `.env`, `auth.json`, and `.claude` are logged.
- [ ] Run e2e smoke tests in interactive and non-interactive modes.
- [x] Publish a parity matrix artifact for old-script vs new-CLI outcomes.
- [x] Capture post-implementation learnings in `docs/solutions/`.

## References

- Baseline behavior: `setup-sprite-opencode.sh:1`
- Current user docs: `README.md:1`
- Existing plan context: `docs/plans/2026-02-21-feat-convert-setup-script-to-go-cli-plan.md:1`
- Sprites Go SDK: `https://github.com/superfly/sprites-go`
- SDK examples: `https://github.com/superfly/sprites-go/blob/main/examples/exec.go`, `https://github.com/superfly/sprites-go/blob/main/examples/sprite_create.go`, `https://github.com/superfly/sprites-go/blob/main/examples/sprite_list.go`
- SDK internals: `https://github.com/superfly/sprites-go/blob/main/client.go`, `https://github.com/superfly/sprites-go/blob/main/management.go`, `https://github.com/superfly/sprites-go/blob/main/exec.go`, `https://github.com/superfly/sprites-go/blob/main/filesystem.go`
- API docs: `https://docs.sprites.dev/api/v001-rc30/`
- DeepWiki repo docs: `https://deepwiki.com/superfly/sprites-go`
