---
title: Resolve setup-sprite-opencode transfer hang and parity drift
date: 2026-02-21
module: setup-sprite-opencode
component: transfer
problem_type: integration-issues
severity: medium
tags:
  - go
  - cli
  - sprites-go
  - migration
---

## Symptom summary

- `setup-sprite-opencode` appeared to hang during `go run ./cmd/setup-sprite-opencode ...` in logs like `phase=transfer step=start`, even though bootstrap and preflight succeeded.
- The final script parity gap was not limited to transfer: `--org` handling and launch output/console behavior also differed from the legacy shell flow.
- The CLI still needed to avoid `~/.claude` transfer per user requirement while preserving migration semantics.

## Reproduction / observed behavior

- Run `go run ./cmd/setup-sprite-opencode --branch <branch> </dev/null` in a repo with local `~/.config/opencode` that contains many files or symlinks.
- In prior versions, setup logs usually stalled after the transfer start phase and never reached `phase=git setup` or `phase=launch`.
- Manual inspection against `setup-sprite-opencode.sh` showed a behavioral mismatch in data movement and launch fallback semantics.

## Investigation notes

- We already had a successful "don't transfer `~/.claude`" commit (`19ff0c9`), but that did not fully eliminate the observed stall.
- Initial Go parity used per-file copy via `copyTree` in `internal/remote/transfer.go`, including path safety checks and repeated read/write operations to remote FS.
- The legacy shell flow uses tar packing for `~/.config/opencode` and remote extraction; this avoids deep walk overhead and makes symlink-heavy directories much less fragile.
- A second drift check showed Go behavior also diverged from the shell for org-scoped operations and launch behavior, which impacted consistency after transfer.

## Root cause

- **Primary hang cause:** the old Go transfer path copied `~/.config/opencode` file-by-file (`copyTree`) in a large, symlink-sensitive traversal, which could stall relative to the shell’s tar-based approach.
- **Secondary parity cause:** CLI flow mismatched legacy expectations in org scoping, branch preflight strictness, and launch path, creating extra migration risk and inconsistent console output.

## Fix steps

1. Switched transfer implementation to the shell-style tar flow:
   - `internal/remote/transfer.go` now writes local `auth.json`, `.env`, and a tarball of local `~/.config/opencode` to the sprite, then extracts with `tar -xzf`.
   - Added `packLocalConfigTar` and `runTarWithFallback`.
   - `transfer_test` now validates repository-name derivation used for remote repo path and keeps the transfer test surface aligned with the new flow.
2. Kept and reinforced `~/.claude` exclusion in transfer callsites and docs
   - `internal/cli/config.go`, `internal/flow/run.go`, and `internal/remote/transfer.go` now only transfer `auth.json`, `.env`, and `~/.config/opencode`.
3. Restored migration-compatible auth/org behavior:
   - `internal/sprites/client.go` now carries optional org context.
   - `internal/flow/run.go` passes `cfg.Org` into client initialization.
   - `internal/cli/config.go` removes fail-closed `--org` rejection.
4. Restored launch behavior matching shell parity in TTY and non-TTY paths:
   - `internal/remote/launch.go` now uses `expect` when available and falls back to direct `sprite exec`.
   - Reconnect command format now follows the legacy structure and is emitted on non-TTY.
5. Aligned bootstrap env handling with shell semantics:
   - `internal/remote/bootstrap.go` appends the env-loader helper line to shell rc files and uses `set -a; . "$HOME/.env"; set +a`.

## Preventive checks

- Keep regression run: `go test ./...`
- Preserve token resolution tests in `internal/cli/config_test.go` that cover env precedence, auth.json fallback, and sprite-login fallback.
- Add transfer-focused regression tests for tar packaging/extraction and invalid repo naming.
- Add/keep an integration-style smoke run for non-TTY path (status + reconnect command) and TTY path (`expect`/fallback launch order).

## Related references

- `docs/plans/2026-02-21-feat-convert-setup-script-to-go-cli-plan.md`
- `docs/parity/2026-02-21-setup-sprite-opencode-parity-matrix.md`
- `docs/solutions/2026-02-21-script-to-go-cli-sprites-go.md`
- `setup-sprite-opencode.sh`
- Commits: `19ff0c9`, `bf38a7e`, `f1e4a46`
- Current working files: `internal/remote/transfer.go`, `internal/remote/launch.go`, `internal/remote/bootstrap.go`, `internal/cli/config.go`, `internal/flow/run.go`, `internal/sprites/client.go`, `internal/remote/transfer_test.go`
