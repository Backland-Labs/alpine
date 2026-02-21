---
title: "feat: Cloudflare Worker setup with OpenAI auth sync and local/live UI verification"
type: feat
date: 2026-02-20
detail_level: comprehensive
status: planned
---

# feat: Cloudflare Worker setup with OpenAI auth sync and local/live UI verification

## Overview

This plan formalizes the migration from Alpine's local simulated orchestrator behavior to a real Cloudflare Worker + Sandbox control plane that serves OpenCode and supports repeatable launch/open workflows.

Primary operator workflow requirement:

1. test locally first,
2. deploy once for live,
3. run `alpine launch/open` repeatedly without redeploying,
4. prove UI availability with saved screenshots.

This plan uses OpenAI auth via the local OpenCode auth file and enforces least-privilege extraction (OpenAI only).

## Problem Statement

Today, `alpine launch` and `alpine open` can report success while still pointing at a URL that is only synthesized from config, not guaranteed to be backed by a running Cloudflare Worker/Sandbox route.

Current gaps:

- no in-repo Cloudflare Worker service wired to Alpine command lifecycle,
- no setup flow that copies auth from local machine into Worker dev/live runtime,
- no deterministic screenshot-based verification for local and live UI accessibility,
- no clear separation between one-time deploy actions and per-launch runtime actions.

## Goals

### Functional goals

- Add an in-repo Worker app under `cloud/opencode-worker`.
- Expose control plane endpoints for Alpine lifecycle commands.
- Support local development via `wrangler dev`.
- Support live deployment via `wrangler deploy`.
- Copy OpenCode auth from local machine and inject into sandbox runtime.
- Extract and propagate only OpenAI auth data from `~/.local/share/opencode/auth.json`.
- Generate local and live screenshots proving OpenCode UI loads.

### Operator experience goals

- Worker deploy is one-time/occasional, not per launch.
- `alpine launch` talks to an existing control plane URL and only orchestrates sandbox state.
- setup commands are explicit, repeatable, and documented.

### Non-goals

- Full replacement of all existing local orchestrator internals in one step.
- Multi-user tenancy/security model redesign.
- Commercial provider multiplexing beyond OpenAI for this phase.

## Lifecycle and Deployment Model (Authoritative)

The system must enforce this execution model:

- **Deploy-time actions** (infrequent):
  - `wrangler deploy`
  - secrets upload (`wrangler secret put`)
  - Worker code/config updates
- **Session-time actions** (per local dev session):
  - `wrangler dev`
  - optional local auth resync
- **Launch-time actions** (frequent):
  - `alpine launch <name>`
  - `alpine open <name>`
  - `alpine status|list|teardown`

`alpine launch` must never trigger Worker deployment.

## Proposed Architecture

### Components

- **Go CLI (Alpine)**
  - command surface remains `launch/list/status/open/teardown`
  - uses HTTP control plane client when configured
  - falls back to current local orchestrator when control plane URL absent
- **Cloudflare Worker (`cloud/opencode-worker`)**
  - routes browser traffic to OpenCode running in Sandbox
  - exposes control plane endpoints for lifecycle operations
  - stores/reads sandbox metadata via Durable Object identity model as needed
- **Sandbox runtime**
  - boots OpenCode server
  - includes `opencode-openai-codex-auth` plugin support
  - receives OpenAI auth payload via Worker env/secret

### Control plane endpoint contract (v1)

- `POST /v1/sandboxes/:name/launch`
- `GET /v1/sandboxes`
- `GET /v1/sandboxes/:name/status`
- `GET /v1/sandboxes/:name/open-url`
- `POST /v1/sandboxes/:name/teardown`
- `GET /healthz`

All endpoints return deterministic JSON payloads with stable reason codes for retryable failure states.

## Auth Strategy (OpenAI-Only Extraction)

### Source of truth

- local machine file: `~/.local/share/opencode/auth.json`

### Extraction policy

- parse source JSON
- keep only OpenAI credential entries
- discard all non-OpenAI providers and unrelated fields

### Transport policy

- use `OPENCODE_AUTH_OPENAI_B64` to carry extracted payload (base64 to avoid multiline escaping issues)
- no raw credential logging
- no credential commit to repository

### Runtime materialization

- decode env/secret in sandbox startup
- write to expected OpenCode auth location(s):
  - `/home/user/.local/share/opencode/auth.json`
  - `/root/.local/share/opencode/auth.json` (compat fallback)
- chmod file to `0600`

## Implementation Workstreams

## Workstream 1: Worker scaffold in-repo

Add:

- `cloud/opencode-worker/package.json`
- `cloud/opencode-worker/wrangler.jsonc`
- `cloud/opencode-worker/src/index.ts`
- `cloud/opencode-worker/Dockerfile`
- `cloud/opencode-worker/.dev.vars.example`

Key requirements:

- include Sandbox class binding and Durable Object migration config
- support browser proxy flow for OpenCode web UI
- provide control plane routes under `/v1/*`
- provide health endpoint for smoke checks

## Workstream 2: Auth sync tooling

Add:

- `scripts/sync-opencode-auth.sh`

Script responsibilities:

- read `~/.local/share/opencode/auth.json`
- validate JSON and presence of OpenAI auth entry
- extract OpenAI-only payload
- encode payload to base64
- write `cloud/opencode-worker/.dev.vars` for local dev
- optionally run `wrangler secret put OPENCODE_AUTH_OPENAI_B64` for live deployment

Behavior modes:

- `--local`: write local `.dev.vars`
- `--live`: upload secret to deployed Worker
- `--both`: perform both

Security constraints:

- never print credential values
- return non-zero on missing/invalid auth input
- avoid shell `set -x` in secret-handling code paths

## Workstream 3: CLI control-plane integration

Modify:

- `cmd/alpine/config.go`
- `cmd/alpine/launch.go`
- `cmd/alpine/list.go`
- `cmd/alpine/status.go`
- `cmd/alpine/open.go`
- `cmd/alpine/teardown.go`

Add:

- `cmd/alpine/control_plane.go` (HTTP client + DTOs)
- tests mirroring above files

Requirements:

- new config: `sandbox.control_plane_url`
- when configured, commands call Worker APIs
- when absent, existing local orchestrator path remains functional
- `open` uses backend-returned URL (no synthesized path assumptions)

## Workstream 4: Plugin/runtime readiness for OpenAI

Worker/sandbox must ensure OpenCode can use the selected provider path:

- install and load `opencode-openai-codex-auth`
- bootstrap OpenCode config compatible with plugin
- ensure auth file is present before OpenCode session starts
- fail with actionable startup error when auth is missing

Note: this plan uses operator-provided local auth artifact and does not introduce new API key handling in repo config.

## Workstream 5: Screenshot smoke verification

Add:

- Playwright smoke tests in `cloud/opencode-worker/tests/`
- artifact output directory `artifacts/screenshots/`

Local smoke flow:

1. run `wrangler dev` for Worker
2. run `alpine launch test`
3. resolve URL from `alpine open test --print`
4. visit page and wait for OpenCode UI marker
5. save `artifacts/screenshots/local-ui.png`

Live smoke flow:

1. run `wrangler deploy`
2. run `alpine launch test`
3. resolve URL from `alpine open test --print`
4. visit deployed URL and wait for UI marker
5. save `artifacts/screenshots/live-ui.png`

Optional:

- run `axe` accessibility checks and save JSON report.

## Workstream 6: Repo setup and DX

Modify:

- `Makefile`
- `README.md`
- `alpine.yaml.example`
- `.gitignore`

Add targets:

- `make worker-dev`
- `make worker-deploy`
- `make sync-auth-local`
- `make sync-auth-live`
- `make smoke-ui-local`
- `make smoke-ui-live`

Docs must clearly call out:

- deploy is not part of `alpine launch`
- auth sync is explicit setup, rerun on auth rotation
- local and live verification steps with expected artifacts

## Configuration Contract Updates

Add to `alpine.yaml` schema usage:

```yaml
sandbox:
  control_plane_url: http://127.0.0.1:8787
  web_base_url: https://<worker>.workers.dev
```

Resolution order for URL usage:

- open URL source: control-plane response first
- fallback: existing configured base URL behavior (local mode only)

## Security and Compliance Requirements

- OpenAI-only credential extraction (least privilege)
- never commit `.dev.vars` or generated secret files
- never log token contents
- keep credential handling scripts idempotent and non-verbose by default
- preserve explicit distinction between personal OAuth plugin usage and production API usage in docs

## Risks and Mitigations

- **Risk:** local auth schema changes in OpenCode
  - **Mitigation:** extraction script validates schema and fails with clear error message
- **Risk:** Worker endpoint mismatch with CLI expectations
  - **Mitigation:** shared DTO tests + integration smoke tests
- **Risk:** false-positive browser-open success
  - **Mitigation:** screenshot verification checks actual page load marker
- **Risk:** secret leakage in logs/artifacts
  - **Mitigation:** redact logs and avoid echoing env payloads
- **Risk:** operator confusion about deploy frequency
  - **Mitigation:** explicit docs and Make target naming (`worker-deploy` vs `launch`)

## Delivery Phases

### Phase 1 (foundation)

- Worker scaffold and control-plane route skeleton
- auth sync script with OpenAI-only extraction
- make targets for local dev and auth sync

### Phase 2 (CLI integration)

- control-plane client in Go
- launch/list/status/open/teardown wired to Worker when configured
- tests for config and endpoint mapping

### Phase 3 (verification)

- Playwright local and live smoke checks
- screenshot artifact generation
- accessibility check integration (optional gate)

### Phase 4 (docs and hardening)

- full README runbook
- troubleshooting section for auth and deploy issues
- error contract and reason-code documentation

## Acceptance Criteria

- [ ] `wrangler dev` serves Worker locally and `/healthz` passes.
- [ ] `make sync-auth-local` succeeds using `~/.local/share/opencode/auth.json` and only extracts OpenAI auth.
- [ ] `alpine launch <name>` uses control plane when configured and does not trigger deploy.
- [ ] `alpine open <name> --print` returns backend-issued URL.
- [ ] Local UI smoke test saves `artifacts/screenshots/local-ui.png`.
- [ ] Live UI smoke test saves `artifacts/screenshots/live-ui.png`.
- [ ] `make sync-auth-live` uploads secret without printing credential values.
- [ ] `README.md` documents one-time deploy vs per-launch operations.
- [ ] `.gitignore` excludes local secret and screenshot-temporary artifacts as intended.

## Operational Runbook (Target)

Local first:

1. `make sync-auth-local`
2. `make worker-dev`
3. `./bin/alpine launch test`
4. `./bin/alpine open test --print`
5. `make smoke-ui-local`

Live:

1. `make sync-auth-live`
2. `make worker-deploy`
3. set `sandbox.control_plane_url` to deployed Worker URL
4. `./bin/alpine launch test`
5. `make smoke-ui-live`

## Open Questions

- Whether Worker startup should materialize auth in both user and root homedirs or only one canonical path after confirming OpenCode runtime user in sandbox image.
- Whether accessibility checks become hard gate for CI or advisory report for operator verification only.

## Exit Criteria

This plan is complete when operators can reliably:

- run one-time setup,
- launch and open real sandboxes repeatedly without redeploying,
- verify local and live UI accessibility through saved screenshots,
- rotate local auth and resync without touching repo secrets directly.
