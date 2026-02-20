# Brainstorm: Cloudflare OpenCode Orchestrator CLI

**Date:** 2026-02-19
**Status:** Ready for planning

## What We Are Building

Replace the current `alpine` CLI with a Cloudflare-native orchestrator for OpenCode sandboxes.

The product lets a single operator:

- launch sandboxed OpenCode instances from CLI,
- optionally kick off work with an initial CLI task,
- interact with each instance in the OpenCode web UI,
- disconnect and return later,
- resume work from durable state when needed,
- export work to GitHub branches,
- and auto-teardown sandboxes after successful durable save.

## Why This Matters

- Keep AI coding workloads off the local laptop entirely.
- Run multiple OpenCode instances in parallel without local environment overhead.
- Support asynchronous workflows where work can continue while the user is offline.
- Preserve useful progress with practical durability, then clean up automatically.

## Key Decisions

1. **Product direction:** Replace the existing `alpine` CLI (no backward compatibility requirement).
2. **Primary interaction model:** CLI for orchestration plus OpenCode web UI for interaction.
3. **Durability source of truth:** Durable Objects + R2 for session/workspace persistence.
4. **Resume model:** Pragmatic resume (restore durable context and continue), not exact in-flight process continuation.
5. **Scale target:** Up to 5 active sandboxes for single-user operation.
6. **Kickoff behavior:** CLI can submit an initial task/prompt when launching.
7. **Completion behavior:** Auto-teardown by default after successful durable save.
8. **GitHub support:** Include basic branch-push export per sandbox in v1.
9. **Repository selection:** Sandbox launch must allow explicit repository selection (config default plus CLI override).
10. **Image selection:** Each sandbox can use a unique container image (repo default plus per-launch CLI override).

## Scope Boundaries (v1)

- Single-user first; no multi-user auth, ownership controls, quotas, or audit requirements.
- Focus on reliable orchestration and practical resume, not deep runtime process resurrection.
- Keep operational model simple and hobby-project friendly.

## Resolved Questions

| Question | Decision |
|---|---|
| Primary interface for v1? | CLI + OpenCode web UI |
| What must survive disconnect/sleep? | Workspace and run state via durable storage |
| Which durability backend is primary? | DO + R2 |
| Include GitHub in v1? | Yes, basic branch export |
| Default completion behavior? | Auto-teardown after successful durable save |
| Resume requirement level? | Pragmatic resume |
| Concurrency target? | Up to 5 active sandboxes |
| Backward compatibility with current Alpine? | No; replace it |
| Can sandbox target a specific repo? | Yes, explicit repo selection at launch |
| Can sandbox use a unique image? | Yes, per-sandbox image override supported |

## Open Questions

None at brainstorm close.
