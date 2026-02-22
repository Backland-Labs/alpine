---
title: Script to Go CLI migration with sprites-go
date: 2026-02-21
module: setup-sprite-opencode
component: cli-migration
problem_type: migration
tags:
  - go
  - cli
  - sprites
  - sdk
severity: medium
---

## Context

`setup-sprite-opencode.sh` had grown into a large shell script with mixed concerns (preflight, remote bootstrap, transfer, git, launch), making safe iteration difficult.

## What worked

- Keep parity-first behavior and map shell phases to explicit Go flow stages.
- Use `sprites-go` primitives for create/list/exec/filesystem to avoid shelling out for core operations.
- Implement deterministic exit code taxonomy (`0`, `1`, `2`, `3`, `4`, `5`) early to stabilize UX.
- Add transfer path allowlists and symlink/traversal rejection before adding copy helpers.

## Pitfalls to avoid

- Relying on implicit org scoping with current SDK behavior can be unsafe; fail closed for `--org` until guarantees are clear.
- Mixing secret transfer and logging without central redaction can leak sensitive values.
- Non-TTY contexts require a dedicated output contract instead of interactive attach attempts.

## Recommended pattern

1. Parse and preflight locally.
2. Resolve name and create sprite.
3. Bootstrap tools remotely.
4. Transfer files with backups and restore-on-failure.
5. Run deterministic git branch setup.
6. Launch interactive session only when TTY is available.
