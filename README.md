# Alpine

Cloud sandbox lifecycle CLI for internal development.

Alpine now uses an `up/down` lifecycle contract for session management:

- `alpine up <git-repo> --branch <branch>`
- `alpine down <session-id>`
- `alpine list`
- `alpine status <session-id>`

Legacy lifecycle commands (`create`, `teardown`) now hard-fail with migration guidance.

## Install

```bash
make build
make install
```

## Quick Start

```bash
export ALPINE_PRINCIPAL_ID="you@example.com"
export GITHUB_TOKEN="..." # or SSH_AUTH_SOCK

alpine up https://github.com/org/repo.git --branch main
alpine list
alpine status <session-id>
alpine down <session-id>
```

## Command Contracts

### `alpine up <git-repo> --branch <branch>`

- Creates a session with deterministic `session_id` + `operation_id` output
- Applies lifecycle transitions: `requested -> provisioning -> repo_syncing -> installing -> ready`
- Resolves and stores `target_commit_sha` from remote branch tip
- Attempts browser launch (best effort, non-fatal)
- Supports idempotent retries with `--client-request-id`

Key flags:

- `--branch <branch>` (required)
- `--client-request-id <id>`
- `--public` (skip credential preflight)
- `--no-browser`
- `--json`

### `alpine down <session-id>`

- Stops session lifecycle and persists canonical payload
- Applies lifecycle transitions: `stopping -> persisting -> stopped`
- Returns `durable_object_id`, `persisted_at`, and persistence verification
- Uses deterministic idempotency key by `session_id + persist_attempt`
- Uses `--retry-persist` for `stopped_unpersisted` recovery

Key flags:

- `--retry-persist`
- `--json`

### `alpine list`

- Lists owner-scoped active/recent sessions
- `recent` window is last 24 hours for terminal sessions
- Sort order is `updated_at` descending
- `--all-owners` requires admin override (`ALPINE_ADMIN=1` or `ALPINE_ROLES=...,alpine.admin,...`)

### `alpine status <session-id>`

- Read primitive for one session
- Returns lifecycle state, last error, and recommended next step

## Identity and Access

All command ownership/idempotency is scoped to principal identity:

- `ALPINE_PRINCIPAL_ID` (preferred)
- `CF_ACCESS_SUB` (fallback)

Admin override for cross-owner reads/stops:

- `ALPINE_ADMIN=1`, or
- `ALPINE_ROLES` containing `alpine.admin`

## Persistence and Local Ledger

Session metadata and durable payloads are stored in a local ledger file:

- Default path: `~/.alpine/sessions.json`
- Override path: `ALPINE_LEDGER_PATH=/path/to/sessions.json`

Persistence payload behavior:

- Schema version: `v1`
- Max payload size: `512 KB`
- Deterministic compaction order: `operation_history` then `state_transitions`
- `history_truncated=true` when compaction is applied

## Error Contract

JSON error responses include:

- `error_code`
- `cause`
- `next_step`
- `retryable`
- `operation_id` (when available)
- `exit_code`

## Development

```bash
go test ./cmd/alpine/...
make lint
```
