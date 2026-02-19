---
title: "feat: Cloudflare Sandbox Up/Down CLI Cutover"
type: feature
date: 2026-02-19
status: active
---

# feat: Cloudflare Sandbox Up/Down CLI Cutover

## Enhancement Summary

**Deepened on:** 2026-02-19  
**Sections enhanced:** 9  
**Research agents used:** formal-verification-designer, spec-flow-analyzer, architecture-strategist, security-sentinel, performance-oracle, data-integrity-guardian, agent-native-reviewer, framework-docs-researcher, best-practices-researcher, repo-research-analyst, learnings-researcher, plus broad parallel reviewer pass

### Key Improvements

1. Added stronger contract semantics for idempotency, lifecycle transitions, and persistence verification.
2. Added execution-hardening requirements for auth preflight, structured observability, and redaction safety.
3. Added release governance hardening with canary/abort gates, rollback continuity, and compatibility checks.

### New Considerations Discovered

- Cloudflare Sandbox and Containers are beta-grade dependencies; command contracts must be shielded by adapter and schema tests.
- Durable Objects persistence needs bounded payload strategy plus deterministic verification beyond timestamp-only checks.
- Agent automation reliability improves materially with a first-class read primitive (`status`) and versioned JSON envelopes.

## Section Manifest

- Section 1: `Overview` / `Problem Statement` - clarify product boundary, command UX contracts, and migration intent.
- Section 2: `Acceptance Criteria` - strengthen measurable criteria for readiness, persistence, and automation parity.
- Section 3: `Proposed Solution` - harden command contracts, source-of-truth model, idempotency semantics, and lifecycle transitions.
- Section 4: `Technical Considerations` - pin file boundaries, execution conventions, and anti-regression architecture patterns.
- Section 5: `Implementation Plan` - add contract freeze, compatibility audit, resilience testing, and cutover governance steps.
- Section 6: `Success Metrics` - expand operational metrics for p95/p99 behavior and orphan control.

## Overview

Replace Alpine's current core lifecycle with a hard cutover to:

- `alpine up <git-repo> --branch <branch>`
- `alpine down <session-id>`

`alpine up` must create a Cloudflare Sandbox session on tailnet, prepare the repository, install development dependencies, and launch Opencode web in the user's browser so the session is immediately ready for development.

`alpine down` must close the session and persist session data to Cloudflare Durable Objects by default.

### Research Insights

**Best practices:** define CLI as a public API with stable exit codes, stable JSON schema, and deterministic idempotent behavior for `up/down`.

**Performance considerations:** separate cold-start and warm-start `up` timing in metrics and tune timeout budgets by phase.

**Implementation details:** treat `up/down` as asynchronous operations with an `operation_id`, and keep human output readable while making JSON authoritative for automation.

**Edge cases:** network disconnect during provisioning must be recoverable by `operation_id`; browser launch failure must never block a ready session.

**References:** Cloudflare Sandbox lifecycle/sessions docs, Durable Objects limits/consistency docs, CLI lifecycle best-practice research.

## Problem Statement / Motivation

The current workflow is container-first and command-heavy for this new product direction. The team needs a simpler remote lifecycle that:

- reduces setup friction to one start command and one stop command,
- supports both private and public repositories,
- guarantees data persistence on teardown,
- and is resilient to provisioning failures with clear user-facing errors.

Without this cutover, Alpine's UX and architecture stay misaligned with cloud sandbox sessions and internal developer expectations for immediate readiness.

### Research Insights

**Best practices:** hard cutovers succeed when command replacement, migration messaging, and rollback mechanics are explicitly testable.

**Performance considerations:** lack of source-of-truth clarity between provider state and ledger state is a common root cause of stale or contradictory CLI results.

**Edge cases:** duplicate retries, stale locks, and partial teardown outcomes (`stopped_unpersisted`) need deterministic recovery paths.

## Stakeholders

- Internal development team (primary users)
- Platform/infra maintainers for Alpine CLI
- Security and compliance stakeholders (repo credentials, persistence guarantees)
- Agent users and automation workflows relying on stable CLI contracts

## Research Note

### Key local references

- `docs/brainstorms/2026-02-19-cloudflare-sandbox-two-command-cli-brainstorm.md`
- `cmd/alpine/main.go`
- `cmd/alpine/create.go`
- `cmd/alpine/list.go`
- `cmd/alpine/status.go`
- `cmd/alpine/errors.go`
- `cmd/alpine/output.go`
- `cmd/alpine/exec.go`
- `CLAUDE.md`

### Reusable learnings from `docs/solutions/`

- `docs/solutions/developer-experience/oauth-token-support-alpine-cli-20260212.md`: support multiple auth modes and do not assume one token path always works.
- `docs/solutions/developer-experience/claude-auth-onboarding-skip-alpine-cli-20260212.md`: avoid silent auth/setup failures; return explicit actionable errors.
- `docs/solutions/logic-errors/git-worktree-docker-cp-broken-alpine-cli-20260212.md`: avoid host-coupled repo transfer assumptions; bootstrap inside isolated runtime.
- `docs/solutions/developer-experience/container-root-user-claude-setup-alpine-cli-20260212.md`: treat runtime identity and startup checks as first-class quality gates.
- `docs/solutions/patterns/critical-patterns.md`: preserve project-level reliability and command ergonomics patterns.

### External references used

- Cloudflare Sandbox lifecycle/session docs and beta constraints
- Cloudflare Durable Objects storage/lifecycle/limits docs
- Cloudflare secret and env var handling docs
- Cross-industry CLI lifecycle patterns for async remote sessions (`up/down/list/status`)

### Open questions to resolve in implementation

- Product direction is resolved. Remaining implementation details are explicitly captured in phased tasks below.

## Acceptance Criteria

### Functional

- [x] `alpine up <git-repo> --branch <branch>` creates one remote session and returns a generated `session_id`.
- [x] `alpine up` supports private and public repos.
- [ ] `development-ready` means sandbox health probe passes within 10 seconds, repo HEAD matches requested branch tip, dependency install exits 0, workspace write probe succeeds, and Opencode URL returns HTTP 200-399 within 10 seconds.
- [x] `development-ready` also records the pinned `target_commit_sha` resolved at operation start and verifies workspace HEAD equals that SHA before `state=ready`.
- [x] Successful `alpine up` output (human and `--json`) includes `session_id`, `operation_id`, `state=ready`, `opencode_url`, and `ready_at`.
- [x] `up` and `down` are blocking-by-default commands in v1: they return terminal state on success and return retryable error + `operation_id` on timeout/interruption.
- [x] `alpine up` attempts one local browser launch call and does not block more than 3 seconds waiting for browser handoff.
- [x] If browser auto-open fails (headless or no GUI), `alpine up` still succeeds when readiness gates pass and returns `browser_opened=false` with `opencode_url`.
- [x] If setup fails, `alpine up` cleans up failed remote resources and returns error fields `error_code`, `cause`, `next_step`, `retryable`, and `operation_id` in JSON mode.
- [ ] Setup failure cleanup removes or tombstones allocated resources (sandbox instance, operation record, partial session artifacts) and reports cleanup result.
- [x] `alpine down <session-id>` is idempotent and safe to retry.
- [x] `alpine down` persists session data to Durable Objects by default and confirms persistence status.
- [x] Successful `alpine down` output includes `durable_object_id`, `persisted_at`, and persistence verification result.
- [x] Persistence failure on `alpine down` returns explicit terminal state (`stopped_unpersisted`) with retry guidance.
- [x] `alpine list` shows active/recent sessions and includes the generated `session_id` for teardown targeting.
- [x] `alpine list` defines `recent` as last 24 hours, sorts by `updated_at` descending, and is owner-scoped by default.
- [x] Old lifecycle commands (`create`, `teardown`) are removed or hard-failed with migration guidance in this cutover (where command aliases exist).
- [x] `alpine list` includes terminal recovery states (`stopped_unpersisted`, `persist_failed`, `cleanup_failed`) when present.
- [x] `alpine status <session-id> --json` provides a single-session read primitive with lifecycle state, last error, and next-step guidance.

### Non-functional

- [x] Mutating operations (`up`, `down`) are idempotent under network retries and client disconnects.
- [x] Duplicate `up` requests with the same `client_request_id` do not create duplicate side effects and return deterministic operation/session identity.
- [x] Duplicate `down` requests for the same persistence attempt do not create duplicate side effects and return deterministic operation identity.
- [x] `up/down` mutation paths enforce a session operation lock with lease/expiry behavior and deterministic conflict errors.
- [ ] Structured lifecycle events are emitted for `requested`, `provisioning`, `ready`, `failing`, `stopping`, `persisting`, `stopped`, and terminal failure states with `session_id` and `operation_id`.
- [x] Error output is consistent across human and JSON modes (`cause`, `next_step`, `retryable`, `operation_id`).
- [x] Legacy `create/teardown` invocations return a fixed exit code and machine-readable `error_code` with migration `next_step` in JSON mode.
- [x] No credentials or secret values are printed to logs/stdout/stderr.
- [ ] Human and JSON modes are parity-checked in tests for all success and error classes used by automation.

### Quality gates

- [x] Unit tests cover command validation, state transitions, idempotency key behavior, and error classification.
- [x] Unit tests include an explicit transition-table validation suite for allowed and forbidden lifecycle edges.
- [ ] Integration tests cover happy path plus quota, timeout, disconnect, duplicate request, and persistence failure scenarios against sandbox APIs.
- [ ] JSON output contracts are schema-validated in tests.
- [ ] JSON schema compatibility tests enforce required fields for `up`, `down`, `list`, and legacy hard-fail responses.
- [x] Documentation and migration notes are published for the command cutover.

### Research Insights

**Best practices:** acceptance criteria should map 1:1 to contract tests and avoid relying on human-readable logs for automation.

**Implementation details:** require explicit error taxonomy (`error_code`, `retryable`, `next_step`) and parity checks across human and JSON mode.

**Edge cases:** include tests for idempotency boundary collisions, stale operation lock recovery, and branch-tip movement during long provisioning windows.

## Proposed Solution

### Command contracts

- `alpine up <git-repo> --branch <branch> [--json]`
  - validates input and auth prerequisites,
  - provisions a sandbox session,
  - syncs repo and installs dependencies,
  - opens Opencode web,
  - returns `session_id`, lifecycle state, and operation metadata,
  - accepts or generates `client_request_id` used for deterministic idempotency,
  - blocks by default until terminal success/failure (or timeout/interrupt),
  - includes browser fallback contract for headless/CI/no-default-browser environments.

- `alpine down <session-id> [--json]`
  - locates the session,
  - performs graceful shutdown,
  - persists canonical session data to Durable Objects,
  - reconciles close/persist outcomes into deterministic terminal states,
  - blocks by default until terminal success/failure (or timeout/interrupt),
  - returns persistence confirmation and final lifecycle state.

- `alpine list [--json]` remains as discovery for session IDs and health snapshots.

- `alpine status <session-id> [--json]`
  - is a read-only command,
  - returns current lifecycle state, last transition, last error, and recommended next action,
  - is owner-scoped by default with admin override policy.

### Source of truth and reconciliation

- Durable Objects ledger is the canonical state store for session lifecycle and operation history.
- Provider state (Sandbox API) is reconciled to the ledger using deterministic conflict rules, not treated as independent truth.
- `list/status` read from ledger-first views and enrich with provider freshness metadata when available.
- Reconciliation paths for disconnect/timeout/partial teardown are explicit and retry-safe.
- Conflict resolution precedence is explicit:
  - if ledger is terminal and provider is active, reconciliation attempts provider shutdown and records `cleanup_failed` if unable to converge,
  - if provider is terminal and ledger is non-terminal, reconciliation advances ledger to the matching terminal state with reason metadata,
  - user-visible state remains ledger-authoritative during reconciliation.

### Auth and identity contract

- Caller identity for idempotency and ownership is `principal_id` from authenticated Cloudflare access context.
- If `principal_id` cannot be derived, command fails with `ERR_CALLER_IDENTITY_REQUIRED` (no silent fallback identity).
- Public repo flow allows clone without git credentials, but caller identity (`principal_id`) is still required for ownership, audit, and idempotency.
- Private repo flow precedence is: scoped git token from secret binding first, SSH auth second, otherwise fail with `ERR_REPO_AUTH_MISSING`.
- Auth flow is non-interactive; command never enters login prompts during `up`.
- `alpine down` authorization requires owner match (`principal_id == session.owner_principal_id`) or admin role (`alpine.admin`). Unauthorized requests fail with `ERR_SESSION_FORBIDDEN`.
- Admin override actions are always audit-logged with actor, target `session_id`, and reason.

### Persistence contract

- Durable Objects payload schema version is fixed to `v1` for this release.
- Required persisted fields: `schema_version`, `session_id`, `principal_id`, `repo_url`, `branch`, `state_transitions`, `started_at`, `stopped_at`, `persisted_at`, `operation_history`, and `last_error`.
- Payload hard limit is 512 KB; oversize payload returns `persist_failed` with `ERR_PERSIST_PAYLOAD_TOO_LARGE` and retry guidance.
- Persistence verification requires immediate read-after-write of canonical fields: `session_id`, `schema_version`, `principal_id`, `branch`, terminal state, and `persisted_at`.
- Persisted `repo_url` and `last_error` are stored only in redaction-safe normalized form (credential-stripped URL and sanitized error payload).
- If payload approaches limit, deterministic compaction is applied in this order: truncate `operation_history`, then truncate `state_transitions`, retain all identity/timing/error summary fields, and set `history_truncated=true`.
- If required canonical fields still exceed the limit after compaction, `persist_failed` is returned deterministically with `ERR_PERSIST_PAYLOAD_TOO_LARGE`.

### Idempotency, retries, and timeouts

- Primary idempotency identity for `up` uses `client_request_id`; if omitted, CLI generates a UUID request ID and uses it for the full invocation/retry chain.
- Idempotency key for `down` is deterministic from `session_id` + action (`down`) + `persist_attempt`.
- `persist_attempt` increments only when prior terminal state is `persist_failed` or `stopped_unpersisted` and a retry is explicitly invoked.
- Duplicate calls with same idempotency key return the existing `operation_id` and never create a second side effect.
- Default timeout policy:
  - provisioning handshake: 90 seconds,
  - repo sync: 5 minutes,
  - dependency install: 10 minutes,
  - persistence write and verify: 60 seconds,
  - total `up` command timeout: 12 minutes,
  - total `down` command timeout: 3 minutes.
- Retry policy uses exponential backoff with jitter for transient provider/network errors up to 3 attempts per phase.
- A single retry owner is enforced per phase to avoid retry amplification across CLI, SDK transport, and provider layers.

### Lifecycle state model

Use an explicit session state model for both CLI and durable persistence:

- `requested`
- `provisioning`
- `repo_syncing`
- `installing`
- `ready`
- `failing`
- `stopping`
- `persisting`
- `stopped`
- `failed`
- `persist_failed`
- `stopped_unpersisted`
- `cleanup_failed`
- `close_failed`

Each transition must include timestamp, actor, reason, and operation ID.
Allowed/forbidden transitions and recovery transitions are explicit (for example `stopped_unpersisted -> persisting -> stopped`).
Transition definitions are maintained in a single transition matrix artifact and validated in tests.

### Research Insights

**Best practices:** state machines for async lifecycle commands should encode invariants, not just labels, and reject illegal transitions deterministically.

**Performance considerations:** bounded retry budgets and lock leases prevent thundering-herd behavior during transient provider outages.

**Edge cases:** unknown-outcome operations (timeout after side effects) must reconcile before retrying mutating operations.

### Hard cutover behavior

- `up` and `down` are the only lifecycle entry points after release.
- Existing internals are reused where useful, but user-facing command contracts are replaced.
- Existing `list` remains for discoverability and `status` remains as read-only observability primitive.
- Help text, README, and errors direct users to the new model.
- Legacy `create` and `teardown` (if available) return exit code `1` with `error_code=ERR_COMMAND_REPLACED` and migration `next_step`.
- Cutover rollback path: feature-flagged command registry can restore legacy command handlers within a 7-day release window if `up` success rate drops below 90% for 30 minutes.
- Rollback session continuity rule: if rollback triggers, disable new `up` creation, keep `down` and `list` active for existing cloud sessions until active count reaches zero, then re-enable legacy lifecycle commands.

## Technical Considerations

- Preserve repository conventions from `CLAUDE.md`:
  - one concern per file,
  - all external command execution through shared execution helpers,
  - `userErr()` vs `sysErr()` semantics,
  - consistent JSON output via `outputJSON()` and `outputError()`.
- Keep command wiring and global flags centralized in `cmd/alpine/main.go`.
- Add single-concern command files `cmd/alpine/up.go` and `cmd/alpine/down.go`, and isolate cloud/session orchestration in dedicated files.
- Treat Cloudflare Sandbox as asynchronous provisioning; expose progress and recovery via stable status/list outputs.
- Treat Durable Objects as canonical session ledger, not transient in-memory state.
- Model retries/timeouts/cleanup as first-class command behavior, not ad hoc error handling.
- Keep repository bootstrap in-sandbox (remote clone/fetch) and never depend on host worktree copy patterns.
- Preserve one-concern-per-file boundaries with mirrored tests (`foo.go` -> `foo_test.go`) for each new lifecycle component.
- Use an explicit structured event schema for lifecycle boundaries (`requested`, `provisioning`, `ready`, `failing`, `persisting`, terminal states) with correlation IDs.

### Research Insights

**Best practices:** separate command parsing, orchestration, provider adapters, and persistence adapters to prevent contract drift.

**Performance considerations:** prefer websocket transport for high-op sandbox workflows to reduce subrequest pressure in HTTP mode.

**Edge cases:** avoid silent auth/setup failures; all auth bootstrap failures must surface classified errors with redaction-safe detail.

## Dependencies and Risks

### Dependencies

- Cloudflare Sandbox SDK availability and API contract stability (beta platform)
- Cloudflare Durable Objects for persistence and lifecycle metadata
- Auth path(s) for private git repository access
- Existing CLI framework and output/error infrastructure

### Risks

- **Platform beta churn:** Sandbox/containers behavior may change.
  - Mitigation: isolate provider interactions behind internal adapters and keep contract tests.
- **Session duplication on retries:** users can rerun `up` after network failures.
  - Mitigation: idempotency keys and deterministic operation IDs.
- **Data loss perception on `down`:** persistence may fail partially.
  - Mitigation: explicit persistence result states and retry-safe finalization.
- **Credential handling mistakes:** private repo auth can fail or leak details.
  - Mitigation: secrets-only storage, redaction, and explicit auth diagnostics.
- **Cutover disruption:** users accustomed to `create/teardown` may break workflows.
  - Mitigation: migration guide, clear hard-fail messages, and release notes.
- **Contract drift across beta dependencies:** provider behavior changes may silently break readiness or teardown assumptions.
  - Mitigation: adapter boundary, pinned SDK versions, and contract/integration smoke tests.
- **Source-of-truth divergence:** provider and ledger state can diverge under retry/timeout failures.
  - Mitigation: explicit reconciliation rules, transition matrix enforcement, and operation lock semantics.

## Alternatives Considered

- **Phased replacement:** safer rollout but conflicts with requirement to replace core logic now.
- **Dual runtime support:** more flexibility, but significantly higher complexity and long-term maintenance cost.

Hard cutover remains the chosen approach.

## Implementation Plan

### Phase 0: Contract freeze and compatibility audit

- [x] Freeze machine contracts for `up`, `down`, `list`, and `status` JSON payloads and required error taxonomy.
- [x] Publish command migration map (`create -> up`) with concrete examples for scripts and humans.
- [x] Define transition matrix artifact and lock ownership model before implementation starts.
- [ ] Confirm Cloudflare backend assumptions (Sandbox transport mode, Durable Objects backend type, limits) in a preflight checklist.

### Phase 1: Command contract and migration scaffolding

- [x] Define final `up/down/list` command contract, flags, JSON schema, and exit code mapping.
- [x] Define `status` read contract for single-session observability and recovery guidance.
- [x] Update command registration and help surfaces in `cmd/alpine/main.go`.
- [x] Add lifecycle command files (`cmd/alpine/up.go`, `cmd/alpine/down.go`) and remove or hard-fail legacy lifecycle commands with migration guidance.
- [x] Update top-level docs (`README.md`, command help text) to reflect the cutover.

### Phase 2: `up` end-to-end lifecycle

- [x] Implement `up` validation flow (repo URL, branch, auth preflight, prerequisites).
- [ ] Add private/public auth decision matrix (HTTPS token, SSH, missing token, expired token, insufficient scope, SSO-required) with deterministic error classes.
- [x] Implement sandbox provisioning orchestration and lifecycle state updates.
- [x] Add reconnect/reconcile behavior for client disconnect after provisioning starts (resume existing operation/session instead of duplicate creation).
- [x] Implement repo sync and dependency install flow with clear phase transitions.
- [x] Pin `target_commit_sha` at operation start and enforce readiness on that SHA.
- [x] Implement browser launch for Opencode web as best-effort non-fatal behavior.
- [x] Implement failure cleanup path that removes partial resources and returns concise recovery guidance.
- [ ] Add per-step cleanup checklist for failures in provisioning, repo syncing, and installing (browser launch failures are warning-only and do not trigger cleanup).

### Phase 3: `down` persistence-first lifecycle

- [x] Implement `down` orchestration (`stopping -> persisting -> stopped`).
- [x] Implement Durable Objects persistence payload and verification semantics.
- [x] Define persistence payload size limits, versioning strategy, and behavior for oversized or partial writes.
- [x] Define deterministic compaction/truncation strategy for `state_transitions` and `operation_history` under payload pressure.
- [x] Ensure `down` is idempotent and retry-safe on network/API failures.
- [x] Emit explicit results for persistence success, partial failure, and retry requirements.
- [x] Define retry semantics for `stopped_unpersisted` and return exact retry command in output.
- [ ] Implement race behavior for active usage: acquire session lock, apply 30-second graceful stop window, then force close and persist with conflict metadata in output.

### Phase 4: Observability, reliability, and quality hardening

- [ ] Add structured events and correlation IDs for all lifecycle transitions.
- [ ] Add timeout and retry policy with deterministic error classes.
- [ ] Add unit/integration coverage for retry, duplicate request, and disconnected-client scenarios.
- [ ] Add fault-injection coverage for quota/rate-limit/network disconnect/timeout failures and assert deterministic terminal states.
- [ ] Add redaction tests for auth/setup failures to verify no credential leakage in stdout/stderr/events.
- [ ] Validate JSON contract compatibility for automation clients.
- [x] Add transition-matrix conformance tests and illegal-transition rejection tests.

### Phase 5: Cutover release and validation

- [x] Publish migration notes and command mapping examples.
- [ ] Run acceptance and regression validation against internal team workflows.
- [x] Confirm legacy command paths fail with actionable migration instructions.
- [ ] Go/no-go authority is the Alpine CLI maintainer group lead for the release window.
- [ ] Rollback authority is the on-call platform incident commander for the release window.
- [ ] Define release abort gates: rollback if `up` success rate < 90% for 30 minutes or orphan rate > 2% in first hour.
- [ ] Roll out by canary rings (Ring 1: maintainers, Ring 2: internal early adopters, Ring 3: full internal) and promote only when each ring meets success and orphan gates for one full bake window.
- [ ] Execute rollback playbook drill before release.
- [ ] Mark plan status ready for `/workflows:work` execution.

### Research Insights

**Best practices:** keep Phase 0 contract freeze mandatory so implementation does not redefine behavior mid-flight.

**Performance considerations:** canary rollout with ring-by-ring gating catches regressions earlier than single-threshold global rollout.

**Edge cases:** in-flight operations during rollback need deterministic continuity (`down/list` live while new `up` is disabled).

## Success Metrics

- `up` success rate >= 95% and p95 time-to-ready <= 8 minutes for internal users.
- `down` persistence verification success rate >= 99% with manual retry rate <= 2%.
- orphaned active sandbox rate after failed `up` <= 0.5% within 24 hours.
- transient failure retry success rate >= 90%.
- median time-to-first-edit <= 5 minutes.
- `up` p99 time-to-ready <= 12 minutes, and `list` p95 response time <= 1 second at internal scale targets (200 concurrent active sessions, 500 recent-session rows).

## References

- `docs/brainstorms/2026-02-19-cloudflare-sandbox-two-command-cli-brainstorm.md`
- `docs/plans/2026-02-11-feat-ephemeral-dev-environment-cli-plan.md`
- `docs/solutions/developer-experience/oauth-token-support-alpine-cli-20260212.md`
- `docs/solutions/developer-experience/claude-auth-onboarding-skip-alpine-cli-20260212.md`
- `docs/solutions/logic-errors/git-worktree-docker-cp-broken-alpine-cli-20260212.md`
- `docs/solutions/developer-experience/container-root-user-claude-setup-alpine-cli-20260212.md`
- https://developers.cloudflare.com/sandbox/api/lifecycle/
- https://developers.cloudflare.com/sandbox/api/sessions/
- https://developers.cloudflare.com/sandbox/platform/limits/
- https://developers.cloudflare.com/sandbox/platform/beta-info/
- https://developers.cloudflare.com/durable-objects/api/sqlite-storage-api/
- https://developers.cloudflare.com/durable-objects/platform/limits/
- https://developers.cloudflare.com/workers/configuration/secrets/
- https://developers.cloudflare.com/sandbox/configuration/transport/
- https://developers.cloudflare.com/fundamentals/api/reference/limits/
