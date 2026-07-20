---
name: sprint-plan
description: /sprint-plan — velocity-driven sprint selection for the delivery-team. Computes the sprint's capacity ceiling from the mean story points of the last N=3 CLOSED sprints (with a cold-start po-seed decay for the first N), then acting as the project-manager (with the tech-lead for feasibility) selects backlog units by priority up to capacity at a single granularity and assigns each the sprint's iteration. Reads methodology.capacityModel + prior closed iterations + backlog priority order + Architecture/ durable-knowledge pages; writes only the idempotent iteration field. A recurring planning ceremony (methodology.cadence.planningCeremonies); velocity is read-only.
---

# /sprint-plan — velocity-driven sprint selection

This is the delivery-team's **planning** ceremony: it decides **how much** the coming sprint
can hold (a capacity ceiling derived from proven velocity) and **which** backlog units go into
it (priority order, up to that ceiling, at one granularity level), then commits the choice by
setting each selected unit's iteration field. It runs **in-session** and adopts the
`project-manager` (with the `tech-lead` for feasibility) as sequential subagents in its own
shared context. It is the first of the two planning ceremonies; `/sprint-start` follows to hand
the committed sprint to `atl work dispatch`.

| It reads | It writes |
|---|---|
| `.delivery/methodology.json` (`capacityModel`, `artifactHierarchy`, `cadence`); the last `velocityWindowN` **CLOSED** iterations; the backlog priority order; `Architecture/` durable-knowledge pages for feasibility | **only** the iteration field on the selected units (an idempotent field update). **Velocity is read-only.** |

Field semantics live in [`config-and-methodology.md`](../../knowledge/config-and-methodology.md);
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
- **Cold start (the first `velocityWindowN` sprints)** — there is no empirical velocity mean yet;
  the ceremony blends the PO's `seedVelocity` with accumulating real data (Step 2). If
  `seedVelocity` is `null`, the ceremony **prompts the PO** rather than guessing.
- **Re-run** to re-plan after a crash, a partial run, or a scope change — idempotent; see
  [Idempotent re-run](#idempotent-re-run). A prior `/sprint-review` that rejected work carried those
  items forward with a recorded reason; a re-run re-admits them **first, as top priority** (ahead of
  new work) — same one admission algorithm, carryover at the front, no separate "rejected" pipeline.

## Procedure

The ceremony adopts two roles **sequentially in one shared session context** (both are
`dispatch: subagent` in `methodology.json`): the `project-manager` does the capacity + selection
arithmetic, and the `tech-lead` is consulted for feasibility on the selected set. This is not two
isolated workers — the PM's selection is handed to the tech-lead *in the same context*, and the
tech-lead's feasibility notes flow straight back to the PM's assignment step. **No `developer` /
`tester` worker is spawned** — this is a planning ceremony; work-unit execution is `/sprint-start`
handing off to `atl work dispatch`.

Before Step 1: read `.delivery/config.json` (read-only) and `.delivery/methodology.json`, and
resolve the concrete iteration schedule by listing the backend's iterations (concept #6)
— use each iteration's **actual path and name verbatim** (never construct `"Sprint N"`), per the
`project-manager`'s [`iteration-management.md`](../../agents/project-manager/children/iteration-management.md).

### 1. Compute velocity from the last N=3 CLOSED sprints

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

### 2. Cold-start seed-decay when fewer than N CLOSED sprints exist

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

### 3. Apply the availability factor to get the capacity ceiling

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

### 4. Select backlog units by priority up to capacity, at ONE granularity, and assign the iteration

As the `project-manager`, read the refined backlog and select against the ceiling; then, **as the
`tech-lead` building on that selection in the same context** (read
[`../../agents/tech-lead/agent.md`](../../agents/tech-lead/agent.md) + its `children/`), sanity-check
feasibility against the `Architecture/` durable-knowledge store before the assignment is committed.

- **Read the backlog completely** — the ordered-backlog read (concept #10, the priority-ordered
  backlog) and/or an exhaustive query (concept #10) filtered to the ready types and the
  **not-yet-Completed** state (resolve the Completed category at runtime, concept #7). Apply the
  cap-is-truncation rule ("list means all").
- **Choose the granularity** — the admitted set is homogeneous at **one** level of
  `artifactHierarchy` (`["Epic","Feature","Pbi","Task"]`): **ALL PBI-level OR ALL task-level, never
  a mix within a sprint** (#15 — no mixed granularity). Mixing a parent and its own child double-counts points
  and confuses the DAG. Which level is a project/ceremony decision read from the hierarchy, not one
  the ceremony invents.
- **Carryover FIRST, then new work by priority up to capacity** — admit the **workable carryover**
  returning from the prior sprint — found by the **`carryover` tag** (concept #4) set at
  `/sprint-review`, still not-Completed and **DAG-ready** (all predecessors Done — a `carryover` unit
  whose predecessor is still not-Done stays blocked and waits; workability is **DAG-derived**, and
  `blocked` is only a surfacing label, not the admission gate, since nothing clears it when the block
  lifts)
  ([`../../agents/project-manager/children/reject-and-carryover.md`](../../agents/project-manager/children/reject-and-carryover.md))
  — **ahead of all new candidates, regardless of any new unit's priority**: unfinished committed work
  outranks new work, so it takes the front of the admission and is admitted in full even if it alone
  meets or exceeds `capacity` (the team over-committed last sprint — an honest signal, not a reason to
  drop committed work). **Then** take the remaining **new** units in ascending priority order (concept
  #5 — lower value = higher priority; the board orders ascending) until the summed story points (the
  story-points field) would exceed the capacity that *remains* after carryover (possibly zero). A
  *blocked* carryover is surfaced but **not** admitted to the workable set until it unblocks (it can't
  be worked yet). An item with no estimate is a planning gap — surface it, never admit an unestimated
  unit (its point cost is unknown and corrupts the capacity math). Equal/absent priority among the new
  units falls back to the stable backlog order returned by the ordered-backlog read (concept #10).
- **Feasibility pass (as the `tech-lead`)** — read the relevant `Architecture/` durable-knowledge
  pages (concept #9; search the store for discovery) and flag any selected unit whose approach
  is infeasible or mis-scoped for this sprint; hand any such flag back to the PM step to drop or
  swap that unit before assignment. The tech-lead does **not** re-decompose here (that is `/refine`).
- **Assign the iteration** — set each selected unit's iteration field to this sprint's resolved
  value via the active backend's work-item update (concept #6; batch the admitted set into one call;
  wrap in the resilience policy). This is an **idempotent field update** (concept #10, the
  idempotency contract), **not** a create-membership — a re-run sets the same iteration to the same
  value, a safe no-op.

Nothing is silently dropped: **new** units that don't fit this sprint's *remaining* capacity, or are
held back for feasibility, stay on the backlog for the next `/sprint-plan`. Carryover is never bumped
by a capacity shortfall — it is committed work, admitted first; only new work is subject to the
capacity that remains after it.

## Idempotent re-run

A re-run must **not duplicate items or double-assign iterations** (#16 — idempotency), and it converges
on the intended plan:

- **Iteration assignment is idempotent by nature** — it is an iteration **field update**
  (concept #6), so re-running sets the same iteration to the same value: a safe no-op. Never model
  it as a "create membership" that could double (concept #10).
- **Velocity is read-only** — re-running the Done-item queries sums the same completed items to the
  same number; there is nothing to dedup.
- **This ceremony does not create work-items** in the normal path — it *selects existing* backlog
  units and updates their iteration field; the tech-lead's decomposition (and its `atl-key`
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
