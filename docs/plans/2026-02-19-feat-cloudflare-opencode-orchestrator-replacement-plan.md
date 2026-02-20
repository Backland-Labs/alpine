---
title: "feat: Replace Alpine with Cloudflare OpenCode Orchestrator"
type: feat
date: 2026-02-19
brainstorm: docs/brainstorms/2026-02-19-cloudflare-opencode-orchestrator-brainstorm.md
detail_level: comprehensive
status: completed
---

# feat: Replace Alpine with Cloudflare OpenCode Orchestrator

## Overview

Replace the current Docker-local `alpine` workflow with a Cloudflare-native orchestrator that manages OpenCode sessions in Sandbox containers.

The new CLI targets a single operator running up to 5 concurrent sandboxes and supports:

- launch with explicit repository selection,
- optional initial task kickoff from CLI,
- browser interaction through OpenCode web UI,
- disconnect/reconnect workflows,
- pragmatic resume using Durable Objects (metadata) + R2 (workspace/checkpoints),
- GitHub branch export,
- and automatic teardown after durable save.

## Problem Statement / Motivation

Current `alpine` is built around local Docker lifecycle. The new goal is to keep AI coding work off the laptop, run cloud sandboxes asynchronously, and return later without losing meaningful progress.

Without a cloud-native orchestrator, the user cannot reliably:

- spin up multiple remote OpenCode instances from one CLI,
- track state transitions for long-running jobs,
- recover from disconnects and container sleeps,
- or safely clean up after successful persistence.

## Stakeholders

- Primary: single local operator using CLI + web UI.
- Secondary: future contributors maintaining command lifecycle and test coverage.
- External systems: Cloudflare Worker + Durable Objects + Sandbox runtime + R2 + GitHub.

## Consolidated Research Notes

### Local repository patterns to preserve

- Keep one concern per file and mirrored tests (`docs/solutions/patterns/critical-patterns.md`).
- Keep Cobra root/subcommand structure (`cmd/alpine/main.go`, `cmd/alpine/create.go`, `cmd/alpine/list.go`, `cmd/alpine/status.go`).
- Preserve JSON/text output split via `outputJSON`/`outputError` (`cmd/alpine/output.go`).
- Preserve error semantics: user failures vs system failures (`cmd/alpine/errors.go`, `userErr`/`sysErr`).
- Preserve shell execution abstraction through `run()` wrappers for testability (`cmd/alpine/exec.go`, `cmd/alpine/testhelpers_test.go`).

### Learnings from prior solutions

- Avoid god-files; keep command and infra concerns separated.
- Do not hide auth failures or setup failures behind suppressed stderr.
- Prefer explicit migration decisions over partial compatibility layers.
- Keep security defaults intentional when moving files/state across boundaries.

### External constraints (Cloudflare Sandbox)

- Sandbox architecture is Worker -> Durable Object -> Container; DO identity is durable, container runtime state is not.
- Session state/process/files inside container are ephemeral across sleep/destroy unless persisted externally.
- Durable storage for resume must use mounted object storage (R2/S3-compatible).
- Terminal reconnect is supported, but should not be treated as full process resurrection guarantee.
- Image selection is configured through container class bindings (Wrangler), so v1 image override should be profile-based (mapped classes), not arbitrary runtime image strings.

## Acceptance Criteria

### Functional

- [x] `alpine launch <name> --repo <url>` creates or reuses a sandbox identity and starts OpenCode for that repo.
- [x] `alpine launch` supports optional initial task (`--task`) without requiring immediate terminal attachment.
- [x] `alpine list` shows all managed sandboxes with lifecycle state and last activity.
- [x] `alpine status <name> --json` returns machine-readable lifecycle and durability metadata.
- [x] `alpine open <name>` opens or prints the OpenCode web URL for that sandbox.
- [x] Disconnecting from CLI/web and reconnecting later preserves pragmatic resume context (workspace/checkpoints/history pointers).
- [x] `alpine export <name>` pushes sandbox work to a GitHub branch.
- [x] Sandbox auto-teardown occurs after successful durable save and completion.
- [x] `alpine teardown <name>` remains available for explicit teardown and supports `--force`.
- [x] Launch supports repo-level defaults from `alpine.yaml` with CLI overrides for repo and image profile.
- [x] Launch with same sandbox name but changed repo or image profile fails with exit code `1` unless `--force-recreate` is provided.
- [x] Unknown image profile fails before provisioning mutates remote lifecycle state.
- [x] Export from `saving` or `tearing_down` fails with retryable error contract (exit code `2` + stable reason code).
- [x] Export from `destroyed` succeeds when a verified checkpoint exists.
- [x] `alpine open <name>` for non-running sandbox returns deterministic guidance and machine-readable reason.

### Non-functional

- [x] Commands are idempotent for retry safety (`launch`, `status`, `teardown`, `export`).
- [x] Concurrency target of 5 active sandboxes is supported without cross-session state contamination.
- [x] Data loss is prevented on normal completion path by durable-save-before-teardown invariant.
- [x] Human-readable output and `--json` output are both available for all lifecycle commands.
- [x] Concurrent duplicate launch requests for same identity converge without duplicate provisioning.
- [x] Checkpoint checksum mismatch transitions sandbox to `error` and blocks auto-teardown.

### Quality gates

- [x] Unit tests cover config parsing/validation, lifecycle transitions, output schema, and failure handling.
- [x] Integration tests cover launch -> task kickoff -> reconnect -> export -> teardown path.
- [x] Regression tests cover sleep/restart resume using DO+R2 restoration path.
- [x] `make test` and `make lint` pass.

## Proposed Solution

### Product surface

Replace command surface with cloud orchestration commands:

- `alpine launch <name>`: provision or resume sandbox, clone repo, optionally send initial task.
- `alpine list`: show all sandboxes and high-level state.
- `alpine status <name>`: detailed lifecycle/durability/export status.
- `alpine open <name>`: open/print OpenCode web UI URL.
- `alpine export <name>`: push branch snapshot to GitHub.
- `alpine teardown <name>`: explicit destroy.

### Lifecycle model

Use explicit lifecycle states in DO metadata:

- `new`, `provisioning`, `running`, `saving`, `completed`, `tearing_down`, `destroyed`, `error`.

Lifecycle operations must be convergent (retries move toward desired state, not duplicate work).

- Sandbox identity is immutable by key `(name, repo, image_profile)` once created.
- `alpine launch <name>` with different repo or profile must fail with `userErr` unless `--force-recreate` is set.
- Lifecycle mutations must use per-sandbox operation locks in DO metadata (`operation_id`, `operation_type`, `started_at`, `expires_at`).
- Retried operations with same intent must converge to the same terminal result.

### Durability model (pragmatic resume)

- DO stores orchestration metadata, state transitions, pointers to latest checkpoint, and operation locks.
- R2 stores workspace snapshots/checkpoints and resume artifacts.
- Resume restores latest durable checkpoint, then continues with new prompts/commands.
- Exact in-flight process continuation is explicitly out of scope for v1.
- Each checkpoint must include a manifest (`schema_version`, `created_at`, `repo_ref`, `image_profile`, `content_hash`, `checkpoint_id`).
- DO checkpoint pointer updates only after checksum and manifest validation succeeds.
- Missing/corrupt/incompatible checkpoints transition state to `error` with actionable recovery guidance.

### Auto-teardown policy

- Auto-teardown preconditions: completion signal detected, durable save verified, no active export lock.
- If any precondition fails, sandbox remains `completed` or `error` and does not transition to `tearing_down`.
- `status` must expose teardown blockers and next operator action.
- Retention window for `completed` state should be configurable to support export-before-destroy workflows.

### Repo and image customization

- `alpine.yaml` remains the per-repo config source.
- Add new config sections for cloud orchestration defaults (`repo`, `sandbox`, `durability`, `github`).
- `--repo` and `--image-profile` override config defaults at launch time.
- Image override in v1 is profile mapping to preconfigured Cloudflare container classes.
- Repo resolution order: `--repo` > `alpine.yaml repo.default` > fail fast with remediation.
- Image profile resolution order: `--image-profile` > `alpine.yaml sandbox.image_profile` > default profile.
- Unknown repo config, inaccessible repo, or unknown profile must fail before provisioning transitions begin.
- Relaunch with changed image profile requires explicit recreate; no silent reuse.

## Technical Considerations

### CLI architecture (in-repo conventions)

- Preserve single-concern command files in `cmd/alpine/`.
- Keep root command and shared flags in `cmd/alpine/main.go`.
- Keep reusable Cloudflare orchestration logic in dedicated files (for example: `orchestrator.go`, `state.go`, `cloudflare.go`, `github_export.go`).
- Keep JSON helpers centralized in `cmd/alpine/output.go`.

### Config evolution

- Replace current local-Docker-focused config shape in `cmd/alpine/config.go` with cloud-first schema.
- Maintain strict validation and clear defaulting.
- Keep missing config fallback behavior deterministic.

### Exit codes and automation contract

- Continue standardized exit code categories:
  - `0`: success
  - `1`: user/config/validation error
  - `2`: system/provider/network/runtime error
- Ensure JSON error payload includes stable fields for automation.

### Security and secrets

- Never print secret values in logs/JSON.
- Require explicit credential checks before launch/export.
- Keep branch export path explicit and auditable.

### GitHub export contract

- Export source of truth defaults to latest verified durable checkpoint.
- Export from live runtime requires explicit `--from-live`.
- Export is allowed from `running`, `completed`, or `destroyed` only when verified checkpoint exists.
- Export requests during `saving` or `tearing_down` fail with retryable `sysErr` and stable reason code.
- Branch naming and collision handling must be idempotent across retries.

### Status freshness contract

- `status` must report both DO metadata state and runtime probe freshness timestamp.
- If runtime probe is stale/unavailable, output must include explicit freshness indicator.

## Dependencies and Risks

### Dependencies

- Cloudflare Sandbox SDK, Durable Objects, R2 bindings, Worker deployment.
- OpenCode sandbox image/profile bindings.
- GitHub auth/token for branch export.

### Key risks and mitigations

- **Risk:** false expectation of exact process resume.
  - **Mitigation:** product language and docs explicitly define pragmatic resume behavior.
- **Risk:** autosave failure before auto-teardown causes data loss.
  - **Mitigation:** enforce save-verification gate before teardown transition.
- **Risk:** image override mismatch with Cloudflare static class config.
  - **Mitigation:** validate requested profile against configured class map and fail fast.
- **Risk:** concurrent commands race on same sandbox.
  - **Mitigation:** DO lock/version checks per lifecycle mutation.

## Alternatives Considered

- **Keep local Docker as primary runtime:** rejected; does not meet cloud-first isolation goal.
- **GitHub-only durability:** rejected for weak runtime/checkpoint fidelity.
- **Exact process continuation in v1:** rejected as high complexity/fragility for hobby scope.

## Non-functional Requirements

- Predictable command idempotency and retry behavior.
- Observable lifecycle with actionable statuses.
- Fast operational feedback for single-user workflow.
- Minimal mental overhead in config and command semantics.

## Implementation Plan

### Phase 1: CLI and state contract foundation

- [x] Replace command registrations in `cmd/alpine/main.go` to new orchestration surface.
- [x] Define canonical lifecycle enums and transition guards.
- [x] Define immutable sandbox identity contract and `--force-recreate` behavior.
- [x] Define stable JSON schemas for `list`, `status`, `launch`, `teardown`, and `export`.
- [x] Update `cmd/alpine/config.go` with cloud-first schema and validation.
- [x] Add/update tests for config and output contract.

### Phase 2: Launch, list, status, and open

- [x] Implement launch orchestration path (create/resume sandbox identity, repo clone, optional task kickoff).
- [x] Implement list/status views backed by DO metadata and runtime checks.
- [x] Implement web UI URL discovery/open flow.
- [x] Enforce pre-provision fail-fast checks for repo/profile validity.
- [x] Add idempotency tests and command retry tests.

### Phase 3: Durable resume and auto teardown

- [x] Implement checkpoint save/restore orchestration with R2 pointers in DO metadata.
- [x] Implement checkpoint manifest/checksum validation and error-state transition behavior.
- [x] Implement completion detection -> durable-save verification -> auto-teardown transition.
- [x] Implement teardown-blocker reporting (`status`) when export/save lock exists.
- [x] Add resume regression tests for sleep/restart scenarios.

### Phase 4: GitHub export and hardening

- [x] Implement branch export workflow with explicit auth checks and structured errors.
- [x] Implement export source selection (`checkpoint` default, `--from-live` explicit).
- [x] Implement export-state gating (`saving`/`tearing_down` retryable failure behavior).
- [x] Add `teardown --force` safeguards and destructive action confirmations.
- [x] Add integration tests for full lifecycle including export.
- [x] Update `README.md` and examples for new command surface and config schema.

## Future Considerations

- Multi-user ownership and access control.
- Policy-based retention and cleanup windows.
- Additional provider profiles and richer image catalogs.
- Optional event streaming/notifications for completed runs.

## References

- Brainstorm: `docs/brainstorms/2026-02-19-cloudflare-opencode-orchestrator-brainstorm.md`
- Existing architecture entrypoint: `cmd/alpine/main.go`
- Existing config pattern: `cmd/alpine/config.go`
- Existing output/error patterns: `cmd/alpine/output.go`, `cmd/alpine/errors.go`
- Existing test harness patterns: `cmd/alpine/testhelpers_test.go`
- Critical patterns: `docs/solutions/patterns/critical-patterns.md`
- Cloudflare Sandbox architecture: `https://developers.cloudflare.com/sandbox/concepts/architecture/`
- Cloudflare sessions: `https://developers.cloudflare.com/sandbox/concepts/sessions/`
- Cloudflare lifecycle: `https://developers.cloudflare.com/sandbox/api/lifecycle/`
- Cloudflare storage: `https://developers.cloudflare.com/sandbox/api/storage/`
- Cloudflare mount buckets: `https://developers.cloudflare.com/sandbox/guides/mount-buckets/`
- Cloudflare git workflows: `https://developers.cloudflare.com/sandbox/guides/git-workflows/`
- Cloudflare wrangler sandbox config: `https://developers.cloudflare.com/sandbox/configuration/wrangler/`
- Additional docs input: `https://developers.cloudflare.com/sandbox/llms-full.txt`
