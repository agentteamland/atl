---
name: sprint-start
description: /sprint-start — build the sprint's dependency DAG from the active backend, fail-fast on a cycle or a missing prerequisite, run the mobile-emulator preflight, then materialize .delivery/plan.json and hand off to the deterministic `atl work dispatch` engine. Reads the sprint's admitted work-units (their dependency links, area tags, and priority) over the active backend; writes exactly one derived artifact (plan.json); recurring, the second planning ceremony after /sprint-plan.
---

# /sprint-start — DAG build, emulator preflight, dispatch launch

This is the delivery-team's **bridge ceremony**: it sits after `/sprint-plan` has admitted a
sprint's work-units (assigned their iteration, tagged their areas, linked their
dependencies) and turns that settled backend state into the one file the deterministic Go engine
reads, then launches the engine. It is the single point where an LLM ceremony hands off to
`atl work dispatch` — no other ceremony spawns a `developer`/`tester` worker; the engine does.

The ceremony itself neither codes nor schedules workers. It **reads** the sprint from the active
backend, **validates** the plan is startable (acyclic + every mobile unit's emulator
is bootable), **materializes** the plan, and **launches** the engine. It writes exactly one
artifact:

| Artifact | What it holds | Consumed by |
|---|---|---|
| `.delivery/plan.json` | the resolved work-unit DAG for this sprint — `sprintSlug`, `granularity` (`pbi`\|`task`), and `units[]` (each `id`, `title`, `predecessors`, `stackRank`) | `atl work dispatch` (the Go scheduler) |

Everything the ceremony reads about *how this project works* is data:
[`config-and-methodology.md`](../../knowledge/config-and-methodology.md) (the `.delivery/`
config + methodology descriptor); the concepts every backend operation depends on are defined
provider-neutrally in [the backend interface](../../knowledge/backend-interface.md), and each
operation follows the one contract in the active backend's adapter
(`backends/<backend>/adapter.md`). Building and validating the DAG is
judgment-heavy — which links are real predecessors, is the graph acyclic, is the emulator
actually up — which is Skill territory under the CLI/Skill boundary; the **deterministic**
scheduling that follows (admit up to a cap, refill on completion, run the recovery ladder) is the
Go engine's, and this ceremony deliberately does not re-implement it.

## When to run

- **Recurring, once per sprint**, immediately after `/sprint-plan` — it is the second slot in the
  methodology's `cadence.planningCeremonies` (`["sprint-plan", "sprint-start"]`, read from
  [`methodology.json`](../../knowledge/config-and-methodology.md)). `/sprint-plan` decides *which
  items and how much*; `/sprint-start` turns that into the runnable DAG and starts the engine.
- **Re-run** to resume a sprint after a crash, a partial dispatch, or a re-plan. The DAG build and
  the preflight are read-only/environmental, and `plan.json` is **derived from the active backend** — so a
  re-run converges (see [Idempotent re-run](#idempotent-re-run)).

## Procedure

Read `.delivery/config.json` + `.delivery/methodology.json` first (both are read-only to
ceremonies — only `/delivery-init` writes them). The concurrency cap, the `area:*` conventions,
and the branch pair all flow from this data; the sprint's actual `dev` branch name comes from
`config.branchPair` (config wins over `methodology.branches`), which the engine bases every
worktree off.

### 1. Build the sprint's dependency DAG (roles: tech-lead + project-manager, in shared context)

Acting as the `tech-lead` (read [`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md)
+ its `children/`, chiefly `decomposition-blueprint.md` and `canonical-brief.md`) and then, in the
**same session context**, as the `project-manager` (read
[`../../agents/project-manager/agent.md`](../../agents/project-manager/agent.md) + its `children/`,
chiefly `sprint-planning-blueprint.md`), assemble the DAG from the sprint's committed work-units:

- Read the sprint's admitted units — read the sprint's items (concept #6) for this sprint's
  iteration. **"List means all"** (concept #10, the query/idempotency substrate): if the set
  could exceed the tool's return, close the gap with a high-limit query and treat a
  result **at** the query's cap as a truncation error to surface, never as a complete read — a
  half-read sprint yields a broken DAG.
- Batch-read each unit's fields + relations — read the work-items (concept #1) in a batch (a single
  work-item read per unit only when a relation detail is missing from the batch). From each
  unit collect: its **dependency links** (concept #8) — a forward edge names its successor, a
  reverse edge its predecessor — its `area:*` tags (concept #4), and its priority (concept #5).
- The DAG edge set is the **predecessor → dependent** direction, restricted to edges *among this
  sprint's admitted units*: an edge to a unit already in the runtime-resolved completed state
  (resolve the completion/state model at runtime — concept #7; **never** a literal `"Done"`) is
  satisfied and dropped; an edge to an out-of-sprint, not-yet-completed unit makes the dependent
  un-startable and is surfaced (a `/sprint-plan` gap), not silently included.

As `project-manager`, this reuses the DAG your `sprint-planning-blueprint.md` already built at
`/sprint-plan`; as `tech-lead`, **confirm each admitted unit has its canonical brief** (the
per-unit brief a `developer` worker reads — `canonical-brief.md`) recorded, since the engine's
workers depend on it. This is a re-read + confirm, not a re-decomposition.

**Degenerate sprint — refuse before validating, materialize nothing (fail-fast).** With the admitted
units read, if there is **no workable unit to dispatch**, do not fabricate a plan — refuse and
surface *why*, exactly as a cycle or a missing device does. **Never** silently write an empty
`plan.json` and no-op the engine: a degenerate sprint is a planning state to surface, not a silent
pass. Two cases, two **distinct** messages, so the PO knows which and can act:

- **Empty** — zero units admitted to this sprint (an empty backlog, or `/refine` produced nothing):
  *"Sprint is empty — no admitted work-units. Run `/sprint-plan` to admit from the backlog, or
  `/refine` first if the backlog itself is empty."*
- **Complete** — every admitted unit is already in the runtime-resolved completed state (concept #7 —
  a re-run, or all work finished): *"Sprint is complete — all N admitted units are already Done.
  Nothing to dispatch."*

(A *blocked* carryover unit that is not yet workable does not, by itself, count toward the workable
set — see [`reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md);
a sprint holding only blocked units likewise has nothing workable, so it refuses too, naming the
blocked units awaiting their unblock.)

### 2. Validate acyclicity FIRST — refuse and surface a cycle (fail-fast)

Before anything else, topologically sort the DAG (Kahn's algorithm: repeatedly remove a node with
no unsatisfied predecessor; if nodes remain when none can be removed, the remainder is a cycle).

- **Acyclic → proceed.**
- **A cycle → REFUSE to start.** Name the exact loop by work-item id (e.g. `#412 → #418 → #431
  → #412`), materialize nothing, and surface it back to the ceremony. A dependency cycle is a
  **planning error the `tech-lead`/`project-manager` must fix** — a decomposition contradiction (A
  waits on B waits on A) — never something to break heuristically by dropping "the weakest edge"
  (breaking a cycle is a decomposition decision, the tech-lead's authority, not this ceremony's).
  This is `atl work dispatch`'s own start-gate mirrored at ceremony time so the cycle is caught
  before the engine is even invoked; the engine independently re-validates (duplicate id, self-loop,
  dangling predecessor, cycle) and refuses to start on any of them.

### 3. Emulator preflight — fail-fast on a missing mobile prerequisite

If **any** admitted unit carries `area:mobile` / `area:ios` / `area:android` in its tags (concept #4),
run the mobile-emulator preflight before dispatch (the discipline is the tester's
[`../../agents/tester/agent.md`](../../agents/tester/agent.md) `children/mobile-and-web-surfaces.md`
— single-slot lease, preflight bootability, block-never-silent-pass; the runtime wiring is
[`../../knowledge/testing-surfaces.md`](../../knowledge/testing-surfaces.md) §3 + its scripts, which
this ceremony runs):

- **Probe bootability** — run [`scripts/emulator-preflight.sh`](../../scripts/emulator-preflight.sh)
  (`ios`/`android`) for each mobile platform the sprint touches. It lists the available devices
  (`xcrun simctl list` / `emulator -list-avds`), attempts a **trial boot** of the shared device, and
  gates the wait on the device's **readiness signal** (iOS `simctl bootstatus`; Android
  `adb wait-for-device` + `sys.boot_completed`) with a bounded poll — **never a fixed `sleep`** (iOS
  boot is 30–90s+ and needs a GUI, so a fixed sleep flakes or wastes the whole team's time).
- **No bootable device → REFUSE to start.** Surface the **exact** missing prerequisite — no GUI
  session, no AVD configured, an unaccepted Xcode license — so the human can fix it. A mobile unit
  dispatched with no emulator would be forced to either block or (the cardinal sin) silently pass an
  un-run mobile gate; refusing up front is the honest failure.
- **Bootable → boot the shared device once and keep it warm** for the sprint. The single-slot lease
  means mobile verification serializes across workers; non-mobile units are unaffected and run at
  full concurrency. Note this lease is a **second constraint, orthogonal to the DAG + cap
  admission**: it bites at the emulator gate inside a worker, not at the scheduler's admission
  decision — the engine still admits by ready-set + cap, and mobile units simply queue on the lease
  when they reach their mobile check.
- No mobile-tagged unit → skip this step entirely.

### 4. Materialize `.delivery/plan.json`, then hand off to `atl work dispatch`

Write the plan the Go engine reads to `.delivery/plan.json` — **exactly** this schema (it is
deserialized into `dispatch.Plan`; extra or renamed fields are ignored/dropped, so match it
verbatim):

```json
{
  "sprintSlug": "<fs-safe-sprint-id>",
  "granularity": "pbi",
  "units": [
    { "id": 4821, "title": "<title>", "predecessors": [], "stackRank": 1 },
    { "id": 4822, "title": "<title>", "predecessors": [4821], "stackRank": 2 }
  ]
}
```

- `sprintSlug` — a filesystem-safe id for this sprint; it becomes the `{slug}` in the engine's
  `delivery/{slug}/{id}` branch + worktree naming, so every branch traces to one unit and one PR.
- `granularity` — `"pbi"` or `"task"`, matching the single level `/sprint-plan` admitted at (the
  all-PBI-or-all-task rule; the DAG never spans levels). Do not mix.
- `units[]` — one entry per admitted unit: `id` (the backend work-item id, concept #1 — the stable
  identity, so there is no separate slug field), `title` (for logging + PR/branch context),
  `predecessors` (the work-item ids that must complete first — the predecessor edges from step 1,
  concept #8, **among this sprint's units only**), and `stackRank` (the priority, concept #5 — the
  admission tie-break: lower value = higher priority). The engine accepts **either** JSON key for
  this field — `priority` (the concept-#5 name) **or** the legacy `stackRank` — so a plan emitting
  either lands a real value (`priority` wins if both are present); emit neither and the frontier
  falls back to id-order.

Then launch the engine: run **`atl work dispatch`** (optionally `--cap N`; the default cap is `4`,
matching the ~4–6 concurrency budget). The deterministic Go scheduler reads `plan.json`,
re-validates the DAG, admits ready units up to the cap (`stackRank` ascending as the tie-break),
spawns one isolated `claude -p` worker per unit in a git worktree off `config.branchPair.dev`,
refills as units complete, and runs the recovery ladder. **The ceremony does not itself spawn any
`developer`/`tester` worker** — that is the engine's job (the `worker`-dispatch roles; the ceremony
only ever adopts the `subagent` roles above).

**Strict milestone ordering (the idempotency policy, concept #10, + the engine's durable-state
discipline):** on a green unit the **tech-lead completes the PR (= the merge to `dev`, non-squash,
concept #11) BEFORE** setting the runtime-resolved completed transition (concept #7); the **engine**
(zero-backend) then *verifies* the merge landed and gates refill on it — it never merges itself. The
Done transition never precedes the merge, so a crash between the two never loses a merge and never
refills against an un-merged unit. This ordering is the contract
([pr-and-review.md](../../knowledge/pr-and-review.md) §4–§5); the ceremony's only obligation is to
hand the engine a well-formed, acyclic plan.

## Idempotent re-run

A re-run — after a crash, a partial dispatch, or a re-plan — **converges**; it never duplicates
work:

- **The DAG build (step 1) is read-only** — reading the sprint's units, their dependency links
  (concept #8), and their priority (concept #5) over the active backend mutates nothing in the backend.
- **The emulator preflight (step 3) is environmental** — probing bootability and warming the shared
  device is a runtime check, not a persisted change.
- **`plan.json` is DERIVED from the active backend, not authored** — re-running against the same
  admitted sprint (same units, same dependency links, same priority) re-materializes the **same**
  plan. Overwriting it with an equal plan is a safe no-op; a re-plan that changed the sprint in the
  backend re-derives the new plan from that new state, staying convergent with the source of truth.
  There is no local ledger to drift (concept #10, the idempotency policy).
- **`atl work dispatch` is itself resumable** — it observes durable state (the git merge, the
  runtime-resolved completed transition, concept #7) rather than trusting a worker's exit code, so
  re-launching it after a
  partial run picks up the un-merged, not-yet-completed units and skips the already-done ones.

No secret is read or written by this ceremony: the credential is referenced by name
(`config.pat.ref` on Azure, `config.credential.ref` on GitHub) and lives in the active backend's
environment — this skill never reads a literal token and never writes one to `plan.json` or
anywhere else.
