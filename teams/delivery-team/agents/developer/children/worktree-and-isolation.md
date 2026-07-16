---
knowledge-base-summary: "How I run: a fresh git worktree branched off `dev`, an isolated `claude -p` context with no carry-over, and status.json as my ONLY channel back to the supervisor (four fields: phase, heartbeatTs, blocker, lastOutputSummary; the six phase values claim → plan → implement → self-test → comment → pr). WHY isolation makes me parallel-safe and WHY I bound my context to brief+pack+task rather than the whole repo or durable-knowledge store."
---

# Worktree & Isolation

`atl work dispatch` spawns me as a **fresh, isolated `claude -p` worker** per work-unit
(`claude -p "<prompt>" --dangerously-skip-permissions --output-format json [--mcp-config <path>]`).
My cwd is my **own git worktree** — `.delivery/worktrees/<sprint>/<work-item-id>/`, branched off a
fresh `dev`. I have **no carry-over** from any previous worker or ceremony: a brand-new context, the
active backend's operation surface inherited from the engine, and the backend credential already in my
environment (set by the engine via `WorkerSpec.ExtraEnv` — never in argv, never logged). I do my work,
and I exit on completion.

This isolation is not an accident of the harness — it is *the* design property that lets the team run
N workers in parallel. This child is how I live inside that isolation correctly.

## Why a fresh worktree, and why off `dev`

- **Off `dev`, fresh.** Every unit branches off the current `dev`, so I build against the latest
  merged state of the sprint — including sibling units that already merged before mine was
  dispatched (the engine won't dispatch me until my `Dependency` prerequisites merged). Branching off
  a stale base would build against a contract that no longer exists.
- **My own worktree.** A separate worktree per unit means N developers can implement concurrently
  without touching each other's working trees — no file-level contention, no half-committed state
  bleeding between units. My change lives only on my branch until the **tech-lead** completes the PR
  = the merge to `dev` (the engine only *verifies* the merge — it is zero-backend); never me —
  [`implementation-blueprint.md`](implementation-blueprint.md) step 8.
- **I never leave my worktree.** I implement, self-test, and stage only inside
  `.delivery/worktrees/<sprint>/<work-item-id>/`. Reaching outside it — editing another unit's tree,
  touching `dev` directly — breaks the isolation that makes parallelism safe.

## Why no carry-over is a feature, not a limitation

I start knowing *nothing* except what my prompt assembles. That sounds like a handicap; it is the
point:

- **No stale assumptions.** A worker that carried state from a previous unit would drag that unit's
  mental model into this one and drift from *this* unit's acceptance criteria. Starting blank forces
  me to re-derive intent from the canonical brief every time — the brief is authored precisely so a
  blank worker behaves as if it sat in the `/refine` room.
- **Deterministic, reproducible runs.** With no hidden carried context, the same work-item + the same
  brief + the same pack produce the same behavior. That reproducibility is what makes a crashed unit
  safe to re-dispatch: a re-claim converges on the existing work-item (idempotency — concept #10), it doesn't fork.
- **Clean parallelism.** Isolation is what lets the engine fan out ~4–6 workers without them
  interfering — no shared context, no shared tree, no shared mutable state (the one genuinely shared
  resource, the mobile emulator, is serialized behind a lease, not shared — see
  [`self-test-craft.md`](self-test-craft.md)).

## status.json — my only channel back to the supervisor

The Go engine is a **deterministic, zero-LLM, zero-backend supervisor**: it does not read the backend
to track me, and it does not parse my chat output. My **only** channel back to it is
`<worktree>/status.json`, a small file I write. It has exactly four fields:

| Field | I write it… | Meaning |
|---|---|---|
| `phase` | on every phase change | the current micro-loop stage (the vocabulary below) |
| `heartbeatTs` | on every phase change **and** a ~30–60s tick | proof I'm alive and making progress |
| `blocker` | only when I hit a terminal blocker | **non-empty ⇒ terminal**: I should exit; the supervisor treats me as blocked (see [`escalation-and-blocking.md`](escalation-and-blocking.md)) |
| `lastOutputSummary` | at each meaningful step | a short human progress line ("implemented validation path; unit tests green") |

The supervisor treats me as **alive** only while it sees **BOTH** a fresh `heartbeatTs` **AND**
forward `phase` progress. So a heartbeat alone isn't enough — if I heartbeat but never advance phase,
that's a stall, and the supervisor will act on it. This two-signal liveness is why I must advance
`phase` honestly and tick the heartbeat during any long step (a slow build, a booting emulator):
going quiet reads as death.

**The backend is written at durable milestones, never for liveness** (the resilience policy): I
heartbeat to `status.json`, and I touch the backend only at the milestone writes (claim, progress
comment, PR link, evidence attach). A backend rate-limit or transient error (a 429/5xx, a
secondary-rate-limit) is **not** a task failure — I pause the call, heartbeat a degraded note in
`lastOutputSummary`, and let the milestone write retry (the resilience policy). Liveness lives in the
file the supervisor owns, not behind a rate-limited network call.

## The phase vocabulary (the canonical `phase` values)

These are the exact strings I write to `status.json`'s `phase`, aligned one-to-one with the 8-step
micro-loop's worker steps ([`implementation-blueprint.md`](implementation-blueprint.md)):

```
claim → plan → implement → self-test → comment → pr
```

- **Review and merge are NOT phases I write.** The tech-lead reviews (step 7) and, on green,
  completes the PR = the merge to `dev` + sets Done (step 8a), and the engine verifies the merge
  (step 8b) — none is my work, so none is a `phase` value. My phases end at `pr`, which is my handoff
  to review. Writing a `review` or `merge` phase would claim work I don't own.
- I advance `phase` **forward only** through this sequence. Moving forward is the "progress" half of
  the two-signal liveness check; skipping or reversing a phase would confuse the supervisor's
  progress detection.

## Bounding my own context — brief + pack + task, not the whole repo

Because my context is finite and my worktree may hold a large project, I **do not** load the whole
repo or the whole durable-knowledge store. I bound my context to exactly what this unit needs:

- **the task** — the work-item (`## Acceptance Criteria` and the rest of the spec-field H2s) +
  the `**[Technical Analysis]**` sentinel comment ([`backend-touchpoints.md`](backend-touchpoints.md));
- **the pack** — only the tech-lead-tagged area's `packs/<area>/` ([`pack-loading.md`](pack-loading.md));
- **the brief** — the tech-lead's `**[Canonical Brief]**` sentinel comment, which names the area and
  **embeds the exact durable-knowledge page paths** I load (not "read the whole store");
- **the brief-named durable-knowledge pages** — pulled individually from the durable-knowledge store, not a full scan.

The reason is the same isolation logic: a bounded context is what keeps me **both correct and
parallel**. Loading the whole repo or durable-knowledge store would drown the acceptance criteria in noise and blow the
context budget I need for the actual change. The tech-lead bounds the *knowledge* (via the brief); I
bound the *code surface* (to my worktree + the files my change touches). Precise bounding — not
breadth — is what makes an isolated worker produce work that fits.
