# Alpine

Cloudflare-native orchestrator for OpenCode sandboxes.

Alpine manages sandbox lifecycle metadata, checkpoint pointers, and GitHub export flows from a local CLI. It provides a cloud-first command surface with idempotent lifecycle semantics and structured JSON output for automation.

## Install

```bash
go install github.com/max/alpine/cmd/alpine@latest
```

Or build from source:

```bash
make build
make install
```

## Quick Start

```bash
cd your-project
alpine launch my-feature --repo https://github.com/your-org/your-repo.git --task "implement issue #42"
alpine list
alpine status my-feature
alpine open my-feature --print
```

## Command Surface

### launch

```bash
alpine launch <name> [flags]
```

Launch or resume a sandbox identity.

| Flag | Description |
|---|---|
| `--repo <url>` | Repository override (required unless configured) |
| `--image-profile <name>` | Image profile override |
| `--task <text>` | Optional initial task kickoff |
| `--force-recreate` | Recreate identity when repo/profile differs |
| `--json` | Structured machine-readable output |

### list

```bash
alpine list [--json]
```

List all managed sandboxes with lifecycle state and last activity.

### status

```bash
alpine status <name> [--json]
```

Show lifecycle, durability, operation lock, runtime freshness, and teardown blockers.

### open

```bash
alpine open <name> [--print] [--json]
```

Open or print the OpenCode web URL for a running sandbox.

### export

```bash
alpine export <name> [--branch <name>] [--from-live] [--json]
```

Export sandbox work using checkpoint-first semantics (or explicit live runtime export).

### teardown

```bash
alpine teardown <name> [--force] [--json]
```

Persist checkpoint metadata and destroy runtime state.

## Configuration

Create `alpine.yaml` in your repo root:

```yaml
repo:
  default: https://github.com/your-org/your-repo.git

sandbox:
  image_profile: default
  image_profiles:
    default: opencode-default
    heavy: opencode-heavy
  control_plane_url: http://127.0.0.1:8787
  web_base_url: https://<worker>.workers.dev
  auto_teardown: true
  completed_retention_minutes: 60

durability:
  bucket: alpine-checkpoints
  checkpoint_prefix: sandboxes

github:
  branch_prefix: alpine
  require_auth: true
```

Resolution order:

- repo: `--repo` > `repo.default` > fail
- image profile: `--image-profile` > `sandbox.image_profile` > fail

## Auth Setup

Auth sync is **explicit setup** and must be rerun when credentials rotate:

```bash
make sync-auth-local   # Sync to local Worker dev
make sync-auth-live    # Sync to production Worker
```

This provisions `.dev.vars` for local development and secrets for the deployed Worker.

## Deploy

**Deploy is not part of `alpine launch`.** Worker deployment is a separate step:

```bash
make worker-dev      # Local development (http://127.0.0.1:8787)
make worker-deploy   # Deploy to Cloudflare Workers
```

## Verification

### Local Verification

```bash
make smoke-ui-local
```

Expected artifacts:
- `artifacts/screenshots/` contains UI captures
- Exit code 0 on success

### Live Verification

```bash
make smoke-ui-live
```

Expected artifacts:
- `artifacts/screenshots/` contains production UI captures
- Exit code 0 on success

## Operational Runbook

### Routine Operations

| Task | Command |
|------|---------|
| Deploy Worker | `make worker-deploy` |
| Sync auth (local) | `make sync-auth-local` |
| Sync auth (live) | `make sync-auth-live` |
| Verify local | `make smoke-ui-local` |
| Verify live | `make smoke-ui-live` |

### Auth Rotation

1. Rotate credentials in your secrets manager
2. Run `make sync-auth-live`
3. Run `make smoke-ui-live` to verify

### Troubleshooting

| Symptom | Check |
|---------|-------|
| Worker 401 errors | Re-run `make sync-auth-live` |
| UI smoke test fails | Check `artifacts/screenshots/` for captures |
| Worker dev won't start | Verify `.dev.vars` exists after `make sync-auth-local` |

## Error and Exit Contract

- Exit code `0`: success
- Exit code `1`: user/config/validation error
- Exit code `2`: system/provider/runtime error

With `--json`, errors include stable fields:

```json
{
  "error": "message",
  "exit_code": 2,
  "reason_code": "export_retryable_state",
  "retryable": true
}
```

## Development

```bash
make test
make lint
```

The test suite enforces >=97% statement coverage for `cmd/alpine`.
