# Brainstorm: Two-Command Cloud Sandbox CLI

**Date:** 2026-02-19  
**Status:** Ready for planning

## What We're Building

Replace Alpine's current core workflow with a strict two-command CLI:

- `alpine up <git-repo> --branch <branch>`
- `alpine down <session-id>`

`alpine up` creates a Cloudflare Sandbox session on the tailnet, prepares the repository on the requested branch, installs development dependencies, and launches Opencode web in the user's browser. The result must be immediately usable for development with no manual setup.

`alpine down` closes the session and persists all session data into Cloudflare Durable Objects by default.

## Why This Matters

- **Lower friction:** One entry command and one exit command reduce onboarding and daily cognitive load.
- **Faster time to coding:** Users move from repo URL to active coding session in a single flow.
- **Consistent environments:** Sandbox-based startup reduces machine-specific drift.
- **Safer lifecycle:** Explicit down flow with guaranteed persistence prevents accidental work loss.

## Approaches Considered

1. **Phased replacement** (safer rollout, temporary overlap)
2. **Hard cutover** (chosen)
3. **Dual runtime support** (flexible but more complex)

### Chosen Direction: Hard Cutover

Adopt `up/down` as the only primary lifecycle interface in one release. This keeps product intent clear, avoids prolonged command duplication, and aligns with the goal of replacing core logic rather than incrementally wrapping it.

## Key Decisions

- **Primary users (v1):** Internal development team.
- **CLI surface:** Hard cutover to `alpine up` and `alpine down` as the core lifecycle commands.
- **Readiness bar for `up`:** Environment is fully prepared, dependencies installed, and Opencode web is launched for immediate use.
- **Repository scope:** Both private and public git repositories are in scope.
- **Session identity:** `up` returns a generated session ID; `down` uses that ID; discovery goes through `alpine list`.
- **Data policy on `down`:** Always persist session data to Durable Objects by default.
- **Failure policy on `up`:** If setup fails, clean up the failed sandbox and return a clear, concise error.

## Resolved Questions

| Question | Decision |
|---|---|
| Who is the primary user in v1? | Internal dev team |
| What is the success bar for v1? | `up` creates a full sandbox with development dependencies and is immediately ready |
| What should happen to data on `down`? | Always persist by default |
| What repo access must be supported? | Private and public repos |
| How should sessions be identified? | Generated session ID plus `alpine list` |
| What if dependency setup fails on `up`? | Fail, clean up, and return a concise error |

## Open Questions

None blocking for planning. Remaining questions are implementation-level and should be resolved in `/workflows:plan`.
