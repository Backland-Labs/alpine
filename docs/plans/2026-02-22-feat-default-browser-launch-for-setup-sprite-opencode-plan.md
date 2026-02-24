# feat: make browser-first launch the default for setup-sprite-opencode

Type: feature
Detail level: standard

## Overview

Switch the default launch behavior of `setup-sprite-opencode` from interactive in-sprite `opencode` to browser-first web usage.

After sprite creation/bootstrap/transfer/git setup completes, TTY runs should:

1. start `opencode serve` inside the sprite in the checked-out repo directory,
2. forward the sprite server to localhost,
3. open the local browser to the forwarded URL.

Non-TTY runs must keep the existing deterministic, machine-readable output contract and remain non-interactive.

## Problem statement / motivation

Today, TTY launch behavior in `internal/remote/launch.go` is terminal-centric (`expect` + shell attach to run `opencode`), while user intent is browser-centric. This creates friction for web-first usage and makes startup behavior inconsistent with a "launch web session" expectation.

The launch phase should still preserve the migration invariants established in this repo: deterministic phase flow, clear output contract in non-TTY contexts, and stable failure classification.

## Stakeholders

- Developers using `setup-sprite-opencode` for local coding sessions.
- Maintainers responsible for parity and reliability of setup flow.
- Automation/CI users depending on non-TTY output stability.

## Detailed background

- Orchestration currently runs in `internal/flow/run.go` and ends with `remote.Launch(...)`.
- Current launch implementation in `internal/remote/launch.go`:
  - non-TTY: emits `status`, `sprite_id`, `reconnect` and exits,
  - TTY: tries `expect` console automation, then falls back to `sprite exec ... opencode`.
- Existing docs and migration artifacts emphasize:
  - preflight-first behavior,
  - non-TTY deterministic contract,
  - parity with shell-era operational semantics.

## Research notes

### Key references and file paths

- `internal/flow/run.go`
- `internal/remote/launch.go`
- `internal/remote/bootstrap.go`
- `internal/remote/gitsetup.go`
- `internal/cli/config.go`
- `README.md`

### Reusable learnings from `docs/solutions/`

- `docs/solutions/integration-issues/setup-sprite-opencode-transfer-hang-parity.md`
  - Preserve parity-focused launch semantics and non-TTY contract.
  - Keep behavior deterministic under integration conditions.
- `docs/solutions/2026-02-21-script-to-go-cli-sprites-go.md`
  - Keep explicit phase boundaries and avoid hidden behavior drift.
- `docs/solutions/2026-02-21-go-cli-migration-baseline.md`
  - Preserve machine-readable non-TTY output and stable exit-code expectations.

### External references used

- DeepWiki: `anomalyco/opencode` command behavior (`serve`, `web`, `attach`).
- DeepWiki: `superfly/sprites-go` port-forwarding APIs (`Sprite.ProxyPort`, `PortMapping`, proxy session lifecycle).

### Open questions still needing decisions

1. Remote port policy when sprite port `4096` is busy.
   - Recommended default: fail fast with clear launch error in first iteration; do not add remote-port fallback yet.
2. Whether to print reconnect details in TTY mode in addition to opening browser.
   - Recommended default: print forwarded URL and reconnect command for observability.
3. Readiness probe strictness.
   - Recommended default: probe proxied localhost root with bounded timeout before declaring launch success.

## Acceptance criteria

- [ ] Default TTY launch starts `opencode serve --hostname 0.0.0.0 --port 4096` inside sprite repo directory.
- [ ] TTY launch establishes localhost forwarding via sprites-go before declaring ready.
- [ ] TTY launch attempts to open the local browser to the forwarded URL.
- [ ] If browser open fails, command remains successful when serve+forward are healthy; URL is printed for manual open.
- [ ] Non-TTY launch remains non-interactive and stdout contract stays deterministic with `status`, `sprite_id`, and `reconnect`.
- [ ] Non-TTY mode does not emit extra stdout noise that breaks machine parsing.
- [ ] Exit code taxonomy remains unchanged (`0/1/2/3/4/5`) for existing failure classes.
- [ ] Launch-phase failures still use existing cleanup-on-error behavior in `flow.Run`.
- [ ] README usage notes describe browser-first default and non-TTY contract.

## Proposed solution

Keep phase orchestration unchanged and update launch behavior only.

1. Keep `internal/flow/run.go` flow as-is (`preflight -> create -> bootstrap -> transfer -> git -> launch`).
2. Replace TTY launch path in `internal/remote/launch.go` with browser-first sequence:
   - spawn remote `opencode serve` command in repo dir,
   - create local proxy session to `127.0.0.1:<local-port>` from sprite port `4096`,
   - wait for bounded readiness,
   - attempt local browser open,
   - print URL and reconnect details.
3. Preserve current non-TTY behavior exactly (machine-readable contract, no browser open).
4. Keep existing auth/token/bootstrap/transfer/git behavior unchanged.
5. Update `README.md` to reflect new default launch semantics.

## Technical considerations

- **TTY detection**: keep current terminal detection behavior to avoid contract drift.
- **Serve lifecycle**: define whether command blocks while remote serve runs or returns after readiness; avoid orphaned local proxy sessions.
- **Signal handling**: ensure Ctrl-C cleanly tears down local proxy and child process context.
- **Port conflicts**:
  - local: deterministic fallback strategy when preferred local port is unavailable,
  - remote: explicit fail policy if `4096` is occupied (unless fallback is later approved).
- **Readiness gate**: avoid fixed sleeps; use bounded retries and timeout.
- **Output channels**:
  - non-TTY stdout stays machine-only,
  - diagnostics may go to stderr.
- **Security/auth**:
  - no change to token resolution precedence,
  - preserve current `.env` and auth transfer behavior.

## Success metrics

- TTY startup opens a usable browser session from a single command in local smoke tests.
- Non-TTY output remains parse-stable across repeated runs.
- `go test ./...` remains green.
- No regression reported in bootstrap/transfer/git phases during smoke verification.

## Dependencies and risks

- Dependency: `opencode` CLI behavior for `serve` endpoint readiness.
- Dependency: `sprites-go` proxy session reliability and local bind behavior.
- Risk: launch hangs from unclear readiness signal.
- Risk: port conflicts in local development environments.
- Risk: headless systems where browser open is unavailable.
- Risk: stdout contract regressions if human logs leak into non-TTY output.

Mitigations:

- Keep readiness timeout bounded and explicit.
- Keep browser-open best-effort with manual URL fallback.
- Add launch-focused unit/integration coverage before flipping defaults in docs.
- Preserve non-TTY serializer contract as a tested invariant.

## Implementation plan

### Phase 1 - Launch contract definition

- [ ] Finalize and document port policy (local fallback, remote fail-fast or fallback).
- [ ] Define launch success criteria: serve started + proxy bound + readiness probe success.
- [ ] Define browser-open failure behavior and stdout/stderr boundaries.

### Phase 2 - Launch implementation

- [ ] Refactor `internal/remote/launch.go` TTY path to run remote `opencode serve`.
- [ ] Add sprites-go proxy wiring for localhost forwarding in launch path.
- [ ] Add readiness probe and timeout handling.
- [ ] Add browser open helper with platform-aware fallback behavior.
- [ ] Preserve existing non-TTY branch contract and reconnect output.

### Phase 3 - Tests and verification

- [ ] Add unit tests for launch helpers (port selection, URL formatting, contract behavior).
- [ ] Add tests that non-TTY output remains deterministic and machine-parseable.
- [ ] Add integration/smoke checks for TTY happy path and browser-open failure path.
- [ ] Verify exit-code behavior unchanged for existing failure classes.

### Phase 4 - Documentation and rollout

- [ ] Update `README.md` launch behavior and usage notes.
- [ ] Add an implementation note under `docs/solutions/` after completion.
- [ ] Run final parity check against previous launch expectations.

## References

- `docs/plans/2026-02-21-feat-convert-setup-script-to-go-cli-plan.md`
- `docs/parity/2026-02-21-setup-sprite-opencode-parity-matrix.md`
- `docs/solutions/integration-issues/setup-sprite-opencode-transfer-hang-parity.md`
- `docs/solutions/2026-02-21-script-to-go-cli-sprites-go.md`
- `docs/solutions/2026-02-21-go-cli-migration-baseline.md`
- `internal/flow/run.go`
- `internal/remote/launch.go`
- `internal/cli/config.go`
- `README.md`
