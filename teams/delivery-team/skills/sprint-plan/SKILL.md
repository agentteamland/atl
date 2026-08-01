---
name: sprint-plan
description: /sprint-plan — sprint selection for the delivery-team, mode-aware. Under the scrum mode it computes the sprint's capacity ceiling from the mean story points of the last N=3 CLOSED sprints (with a cold-start po-seed decay for the first N) and selects up to that ceiling; under the flow mode there is no ceiling and no seed — it admits by priority and keeps the admitted set DAG-closed, pulling each admitted unit's incomplete predecessors in with it so the sprint carries real dependency edges. Either way, acting as the project-manager (with the tech-lead for feasibility) it selects at a single granularity and stamps each unit's sprint carrier — the iteration field under scrum, the sprint:<slug> label under flow. Reads methodology.mode + methodology.capacityModel (scrum only) + prior closed iterations (scrum only) + backlog priority order + the units' dependency links + Architecture/ durable-knowledge pages; writes only the idempotent carrier. A recurring planning ceremony (methodology.cadence.planningCeremonies); velocity is read-only.
---

# /sprint-plan — sprint selection (velocity-driven under scrum, priority + DAG closure under flow)

This is the delivery-team's **planning** ceremony: it decides **which** backlog units go into the
coming sprint (priority order, at one granularity level) and — under `mode: "scrum"` — **how much**
it can hold (a capacity ceiling derived from proven velocity), then commits the choice by stamping
each selected unit's **sprint carrier**. Under `mode: "flow"` there is no ceiling to compute:
nothing derives a budget, and admission is **priority**, with the admitted set kept **DAG-closed** —
a unit's incomplete predecessors are admitted with it, never left outside
([`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §1.1.1). It runs **in-session** and adopts the
`project-manager` (with the `tech-lead` for feasibility) as sequential subagents in its own
shared context. It is the first of the two planning ceremonies; `/sprint-start` follows to hand
the committed sprint to `atl work dispatch`.

| It reads | It writes |
|---|---|
| `.delivery/methodology.json` (`mode`, `artifactHierarchy`, `cadence`, and `capacityModel` — **`scrum` only**); under `scrum`, the last `velocityWindowN` **CLOSED** iterations; the backlog priority order; the candidate units' **dependency links** (concept #8 — required under `flow` to close the admitted set over its predecessors, §1.1.1); `Architecture/` durable-knowledge pages for feasibility | **only** the sprint carrier on the selected units — the iteration field under `scrum`, the `sprint:<slug>` label under `flow`; both are idempotent. **Velocity is read-only** (and not computed at all under `flow`). |

Field semantics live in [`config-and-methodology.md`](../../knowledge/config-and-methodology.md)
(`mode` in its §1.1, **admission vs dispatch readiness in its §1.1.1**, the sprint carrier in its §1.2);
the provider-neutral operation concepts — the tool map, idempotency, pagination, and runtime
type/state resolution — live in [the backend interface](../../knowledge/backend-interface.md), which
the **active backend's adapter** (`backends/<backend>/adapter.md`, selected once at `/delivery-init`
and cached in `.delivery/config.json`, default `azure`) binds to concrete tools. Every backend touch
goes through the active adapter; no ceremony reads or writes a literal credential (auth is a by-name
reference from the environment, never in argv).

## When to run

- **Recurring, once per sprint at planning time** — this is a `methodology.cadence.planningCeremonies`
  slot (paired with `/sprint-start`, which follows it). Run it after the backlog is refined
  (`/refine` has decomposed and dependency-linked the Features) and before the sprint starts.
- **Cold start (the first `velocityWindowN` sprints) — `scrum` only** — there is no empirical
  velocity mean yet; the ceremony blends the PO's `seedVelocity` with accumulating real data
  (Step 2). If `seedVelocity` is `null`, the ceremony **prompts the PO** rather than guessing.
  Under `flow` there is **no cold start at all** — no ceiling is derived, so nothing needs seeding
  and the ceremony never asks.
- **Re-run** to re-plan after a crash, a partial run, or a scope change — idempotent; see
  [Idempotent re-run](#idempotent-re-run). A prior `/sprint-review` that rejected work carried those
  items forward with a recorded reason; a re-run re-admits them **first, as top priority** (ahead of
  new work) — same one admission algorithm, carryover at the front, no separate "rejected" pipeline.

## Procedure

The ceremony adopts two roles **sequentially in one shared session context** (both are
`dispatch: subagent` in `methodology.json`): the `project-manager` does the capacity (under `scrum`)
+ selection arithmetic, and the `tech-lead` is consulted for feasibility on the selected set. This
is not two isolated workers — the PM's selection is handed to the tech-lead *in the same context*,
and the
tech-lead's feasibility notes flow straight back to the PM's assignment step. **No `developer` /
`tester` worker is spawned** — this is a planning ceremony; work-unit execution is `/sprint-start`
handing off to `atl work dispatch`.

Before Step 1: read `.delivery/config.json` (read-only) and `.delivery/methodology.json`, and
**resolve `mode` before anything else** — `"scrum"` or `"flow"`; **absent ⇒ `scrum`**; never
inferred from a missing `capacityModel` or from which board fields exist; an unrecognized value
**halts the ceremony** (surface the two valid values) rather than falling back
([`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §1.1). The mode decides
how this sprint is identified, and which steps run:

- **`scrum`** — resolve the concrete iteration schedule by listing the backend's iterations
  (concept #6); use each iteration's **actual path and name verbatim** (never construct
  `"Sprint N"`), per the
  `project-manager`'s [`iteration-management.md`](../../agents/project-manager/children/iteration-management.md).
  **Steps 1–4 all run.**
- **`flow`** — there is no schedule to resolve, so resolve the sprint's **ordinal** instead: list
  the `sprint:*` labels/tags already on the board (concept #4; "list means all" — a result at the
  query cap is a truncation to surface, not a complete read) and take the highest ordinal `k`,
  **compared as an integer** (`sprint:10` outranks `sprint:9`; a lexical "highest" hands back a stale
  ordinal and the next sprint reuses a number already in use). The
  **current** sprint is `sprint:<k>`, and it stays current until it is **reviewed** — reviewed means
  its `Sprints/Sprint-<k>-Review` page exists (concept #9), the flow analogue of a closed iteration.
  So: `sprint:<k>` **not yet reviewed → this run plans into `sprint:<k>`** (a re-plan, a resumed run,
  or a mid-sprint admission); `sprint:<k>` **already reviewed → open `sprint:<k+1>`**. A board with
  no `sprint:*` label at all starts at `sprint:1`, except that a project **migrating from scrum**
  continues its existing numbering rather than restarting: take the highest ordinal `m` among the
  existing `Sprints/Sprint-<m>-Review` pages and open `sprint:<m+1>` — those pages are the reliable
  read, since iteration *names* are arbitrary. **Call the resolved ordinal `<n>`** —
  it is this sprint's label (`sprint:<n>`) and the `<n>` of its `Sprints/Sprint-<n>-Review` page
  alike. **Skip Steps 1–3 entirely** and go to Step 4.

> **Steps 1–3 compute the capacity ceiling — `mode: "scrum"` only.** Under `flow` the descriptor
> carries no `capacityModel`, nothing derives a ceiling, and there is nothing to seed: **do not
> compute velocity, do not apply an availability factor, and do not prompt the PO for a
> `seedVelocity`.** A cold-start seed is a *scrum* concept — it seeds a **velocity**, and flow
> measures none — so asking for it stalls a headless run on a number the ceremony would then use
> for nothing. Go straight to Step 4, where admission is **priority + DAG closure** — and note
> that `flow` has **no admission ceiling of any kind**: the ~4–6 concurrency cap bounds how many
> units the engine runs at once, never how many this ceremony may admit
> ([`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §1.1.1).

### 1. Compute velocity from the last N=3 CLOSED sprints (`scrum` only)

Acting as the `project-manager` (read
[`../../agents/project-manager/agent.md`](../../agents/project-manager/agent.md) + its
`children/`, chiefly `capacity-and-velocity.md`), compute the empirical velocity — the **mean
completed story points over the last `velocityWindowN` (=3) CLOSED sprints** (read
`velocityWindowN` from `capacityModel`, never hardcode `3`):

- Enumerate the closed iterations from the backend's iteration list (concept #6) — the
  ones whose date range has ended.
- For each closed sprint, read the sprint's items (concept #6, a batch read wrapped in the
  resilience policy), then **keep only the items whose state resolves to the RUNTIME-RESOLVED
  Completed category** — resolve the type's state→category map at runtime (concept #7), never
  hardcode. **Never** compare against the literal `"Done"`; a template may spell completion `Closed`,
  `Completed`, or a custom value.
- Sum the story-points field over those completed items to get the sprint's
  points; average the per-sprint sums: `velocity = mean(sprint_points[])`.
- **Read the whole list.** If a sprint could exceed the iteration read's set, close the gap with an
  exhaustive query (concept #10) filtered to that iteration **and** the Completed category, and
  **treat a result at the query cap as a truncation error to surface**, never as a complete read
  ("list means all"). A half-read Done set silently understates velocity and shrinks every future
  sprint.

Velocity is **read-only** — pure client-side arithmetic over the active backend's work-item queries
(concept #10); no write, inherently idempotent.

### 2. Cold-start seed-decay when fewer than N CLOSED sprints exist (`scrum` only)

Still as the `project-manager`, when `count_closed_sprints < velocityWindowN` there is no honest
empirical mean — apply the `capacityModel.coldStart: "po-seed"` blend so the ceiling isn't a blind
guess and isn't frozen at the guess. For sprint `k` (with `k-1` sprints completed):

```
effectiveVelocity = ( Σ actual_points[1..k-1] + seedVelocity × (N − (k−1)) ) / N
```

- **Sprint 1** — no closed sprints: the ceiling is `seedVelocity` outright (the PO's `/kickoff`
  estimate).
- **Sprints 2 … N−1** — blend the accumulating real closed-sprint points with the seed, the seed's
  weight decaying as real sprints accrue; by sprint `N` the blend is fully actual.
- **Sprint N onward** — `count_closed_sprints ≥ N`: the plain N-sprint mean from Step 1 takes over;
  the seed is gone.
- **If `seedVelocity` is `null`** — do **not** invent a number: **PROMPT the PO** to set the
  cold-start seed, and pause the plan until they do. This is the PO's number, not the ceremony's.

### 3. Apply the availability factor to get the capacity ceiling (`scrum` only)

As the `project-manager`, scale the velocity by the sprint's availability dial:

```
capacity = floor( velocity × availabilityFactor )
```

- `availabilityFactor` defaults to `capacityModel.availabilityFactorDefault` (1.0 = fully staffed);
  the PO owns this 0–1 dial for a short-staffed sprint (holidays, a member on leave) and may
  override it for this sprint. Apply the value given — do not infer who is on leave.
- Optionally read the backend's own team-capacity model (concept #6) as a **secondary corroborating
  signal** for the availability dial (logged days-off), but keep the ceiling **velocity-derived**
  (the backend's capacity is an hours model; this team estimates in story points — the two do not
  convert cleanly).

### 4. Select backlog units by priority, at ONE granularity, and stamp the sprint carrier

As the `project-manager`, read the refined backlog and select — against the capacity ceiling under
`scrum`, and under `flow` by priority with **no ceiling at all**, keeping the admitted set
**DAG-closed**; then, **as the
`tech-lead` building on that selection in the same context** (read
[`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md) + its `children/`), sanity-check
feasibility against the `Architecture/` durable-knowledge store before the assignment is committed.

- **Read the backlog completely** — the ordered-backlog read (concept #10, the priority-ordered
  backlog) and/or an exhaustive query (concept #10) filtered to the ready types and the
  **not-yet-Completed** state (resolve the Completed category at runtime, concept #7). Apply the
  cap-is-truncation rule ("list means all"). **Exclude `/request` candidates** (concept #13 — the
  `candidate` flag / Status): a candidate is a mid-project request the PO has not yet accepted, so it
  is **not** ready-frontier work — admitting it would sweep an unexamined request into a sprint, the
  exact failure `/request` exists to prevent. A candidate enters this query only after `/request`'s
  accept step drops its `candidate` flag.
- **Choose the granularity** — the admitted set is homogeneous at **one** level of
  `artifactHierarchy` (`["Epic","Feature","Pbi","Task"]`): **ALL PBI-level OR ALL task-level, never
  a mix within a sprint** (#15 — no mixed granularity). Mixing a parent and its own child double-counts points
  and confuses the DAG. Which level is a project/ceremony decision read from the hierarchy, not one
  the ceremony invents.
- **Carryover FIRST, then new work by priority up to capacity (`scrum`)** — admit the **workable
  carryover**
  returning from the prior sprint — found by the **`carryover` tag** (concept #4) set at
  `/sprint-review`, still not-Completed and **workable** — nothing *outside* this sprint still
  blocks it: its predecessors are Done, **or** they are admitted here alongside it, in which case it
  waits behind them in-sprint, which is an ordinary edge and not a block
  ([`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §1.1.1 — this reading of
  *blocked* is mode-independent, and
  [`reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md) states
  it the same way). Only a `carryover` unit whose predecessor stays **outside** this sprint and
  incomplete stays blocked and waits; workability is **DAG-derived**, and `blocked` is only a
  surfacing label, not the admission gate, since nothing clears it when the block lifts.
  Admit that workable carryover **ahead of all new candidates, regardless of any new unit's
  priority**: unfinished committed work
  outranks new work, so it takes the front of the admission and is admitted in full even if it alone
  meets or exceeds `capacity` (the team over-committed last sprint — an honest signal, not a reason to
  drop committed work). **Then** take the remaining **new** units in ascending priority order (concept
  #5 — lower value = higher priority; the board orders ascending) until the summed story points (the
  story-points field) would exceed the capacity that *remains* after carryover (possibly zero). A
  *blocked* carryover is surfaced but **not** admitted to the workable set until it unblocks (it can't
  be worked yet). An item with no estimate is a planning gap — surface it, never admit an unestimated
  unit (its point cost is unknown and corrupts the capacity math). Equal/absent priority among the new
  units falls back to the stable backlog order returned by the ordered-backlog read (concept #10).
- **Carryover FIRST, then new work by priority — with the admitted set kept DAG-CLOSED (`flow`)** —
  the same order with the budget removed, and admission gated by **closure**, never by dispatch
  readiness. The rule in full, so this step is decidable without opening another file:

  > **Under `mode: "flow"`, `/sprint-plan` admits by PRIORITY, and the admitted set must be
  > DAG-CLOSED: whenever a unit is admitted, every predecessor it depends on is admitted with it, or
  > is already complete. Never admit a unit whose predecessor stays *outside* the sprint and
  > incomplete — that unit could never start, and that is the only "readiness" admission cares
  > about. A unit blocked by another unit *in the same sprint* is entirely normal: that is precisely
  > the edge `/sprint-start` puts in the DAG and the engine orders.**

  Concretely:
  - **Admit the workable carryover first** — same ordering as under `scrum`, and for the same
    reason (committed work outranks new work), but with workability judged by **closure** rather
    than by the dispatch frontier. A carryover unit whose predecessor is **also admitted into
    this sprint** comes in **with** it (that is the closure rule, and the edge is what the engine
    orders). A carryover unit whose predecessor stays **outside** this sprint and incomplete is
    **blocked**: carried and surfaced with its reason, not admitted
    ([`../../agents/project-manager/children/reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md)).
  - **Then take the new units in ascending priority order** (concept #5) and, before admitting each
    one, **walk its predecessor edges** (concept #8 — the dependency links `/refine` wrote). Every
    **incomplete** predecessor is admitted **with** it, regardless of that predecessor's own
    priority. If a predecessor genuinely cannot be admitted — not yet refined, at a different
    granularity than this sprint's, or itself blocked outside the sprint — then **the unit is not
    admitted either**, and both are surfaced with the reason. Never admit a unit while leaving one
    of its incomplete predecessors outside the sprint.
  - **There is no admission ceiling under `flow`** — no point budget, and the `project-manager`'s
    ~4–6 concurrency cap is **not** one either: it bounds how many units `atl work dispatch` keeps
    **in flight at once**, with refill-on-Done starting the next ready unit as an admitted one
    completes
    ([`../../agents/project-manager/children/sprint-planning-blueprint.md`](../../agents/project-manager/children/sprint-planning-blueprint.md)
    §6). A flow sprint routinely holds more units than that cap; the cap is a runtime bound on the
    engine and the backend, never a limit on membership.
  - **Do not filter the admitted set down to what could start today.** Admitting only units whose
    predecessors are all Done (**`DAG-ready`** — the *dispatch-frontier* sense, which belongs to the
    engine, not to this step) leaves a sprint with **no dependency edge at all**: `/sprint-start`
    then writes a single-node `plan.json`, and the engine's dependency ordering and parallel
    `--cap N` dispatch never run. That is the exact failure this rule exists to prevent
    ([`config-and-methodology.md`](../../knowledge/config-and-methodology.md) §1.1.1 records the
    real occurrence).
  - **An unestimated unit is not a planning gap here**: nothing under `flow` reads an estimate, and
    a flow project need not carry a story-points field at all — so do not surface one and do not
    withhold a unit for missing points. Equal/absent priority falls back to the same stable backlog
    order.
- **Feasibility pass (as the `tech-lead`)** — read the relevant `Architecture/` durable-knowledge
  pages (concept #9; search the store for discovery) and flag any selected unit whose approach
  is infeasible or mis-scoped for this sprint; hand any such flag back to the PM step to drop or
  swap that unit before assignment. The tech-lead does **not** re-decompose here (that is `/refine`).
- **Stamp the sprint carrier** — mode-selected, and the only write this ceremony makes:
  - **`scrum` — assign the iteration.** Set each selected unit's iteration field to this sprint's
    resolved value via the active backend's work-item update (concept #6; batch the admitted set
    into one call; wrap in the resilience policy). This is an **idempotent field update** (concept
    #10, the idempotency contract), **not** a create-membership — a re-run sets the same iteration
    to the same value, a safe no-op.
  - **`flow` — add this sprint's `sprint:<n>` label.** Add the label/tag (concept #4) whose ordinal
    the preamble resolved to each selected unit (batched; same resilience policy). Adding a label
    already present is a no-op, so this is idempotent for exactly the reason the field set is —
    never model it as a create-membership. **A re-admitted carryover unit SWAPS its label**: remove
    the `sprint:` label the unit **actually carries** — read that ordinal off the unit, never assume
    `sprint:<n-1>`; a unit that stayed blocked through one or more sprints was never re-admitted, so
    it still carries the ordinal of the last sprint it *was* in — and add `sprint:<n>` **in the same
    step**, because a label accumulates where a field replaces, and two `sprint:` labels on one unit
    is a corrupt state —
    "which sprint is this in?" stops having an answer and the sprint's item read returns units that
    have moved on. Nothing is lost by the swap: the sprint's membership record is its
    `Sprints/Sprint-<n>-Review` page. A label is **never** removed to mean "done" — completion is a
    state (concept #7).

Nothing is silently dropped. Under `scrum`, **new** units that don't fit this sprint's *remaining*
capacity, or are held back for feasibility, stay on the backlog for the next `/sprint-plan`;
carryover is never bumped by a capacity shortfall — it is committed work, admitted first, and only
new work is subject to the capacity that remains after it. Under `flow` there is no capacity
shortfall to bump anything, and **nothing is held back merely for having an open predecessor**:
what stays on the backlog is what **cannot be made DAG-closed** — a unit some predecessor of which
is not admissible into this sprint — or what the feasibility pass held back. Both are surfaced with
their reason rather than dropped.

## Idempotent re-run

A re-run must **not duplicate items or double-stamp the carrier** (#16 — idempotency), and it converges
on the intended plan:

- **Iteration assignment is idempotent by nature (`scrum`)** — it is an iteration **field update**
  (concept #6), so re-running sets the same iteration to the same value: a safe no-op. Never model
  it as a "create membership" that could double (concept #10).
- **The sprint label is idempotent by nature (`flow`)** — adding a label already on the unit is a
  no-op (concept #4), so the same admitted set re-stamps to the same state. The **swap** converges
  too: on a re-run the unit already carries `sprint:<n>` and no other `sprint:` label, so the removal
  finds nothing left to remove and the add finds the label present — both no-ops. **Resolving the
  ordinal is what makes the re-run land on the same sprint**: the current sprint is the highest ordinal present
  and stays current until its `Sprints/Sprint-<n>-Review` page exists, so a re-run before that review
  re-resolves `sprint:<n>` rather than opening `sprint:<n+1>`. (Advancing on the *highest ordinal
  present* alone would open a fresh sprint on every re-run — the review page is the gate that stops
  it.)
- **Velocity is read-only (`scrum`)** — re-running the Done-item queries sums the same completed
  items to the same number; there is nothing to dedup. Under `flow` no velocity is computed at all,
  so there is nothing to re-run.
- **This ceremony does not create work-items** in the normal path — it *selects existing* backlog
  units and stamps their carrier; the tech-lead's decomposition (and its `atl-key`
  stamping) happened at `/refine`. If a re-run must create any new item (e.g. a feasibility
  swap-in the tech-lead adds), that item carries the two tags (concept #4) the contract requires:
  `atl-run:sprint-plan:<sprint-id>` (provenance) + `atl-key:<hash>` where
  `hash = hash(parent-id + plan-ordinal)` — a **stable plan-ordinal**, never a per-run GUID and
  never `hash(title)` — and is guarded by a **check-first query** (concept #10 for that
  `atl-key`: found → reuse+update, not-found → create-then-stamp; a 409/duplicate is caught and
  resolved to the existing item, never surfaced), per concept #10 and the tech-lead's
  [`decomposition-blueprint.md`](../../agents/tech-lead/children/decomposition-blueprint.md).
- **Branch names**, when referenced, come from `config.branchPair` (config wins over
  `methodology.branches`) — though `/sprint-plan` itself does not touch branches; `/sprint-start`
  and the workers do.
