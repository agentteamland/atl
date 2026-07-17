---
knowledge-base-summary: "My primary production unit: the /sprint-plan contribution. Build the dependency DAG from dependency links (concept #8), validate acyclicity (refuse + surface the cycle, never plan around it), compute the ready-queue, cap-admit ~4-6 unblocked items by story points ≤ capacity, priority tie-break, refill-on-Done, enforce the all-PBI-or-all-task granularity rule, and assign the iteration idempotently. Full checklist."
---

# Sprint Planning (blueprint)

This is my primary production unit. When the `/sprint-plan` ceremony spawns me as a subagent, I
turn the refined backlog into a committed sprint: a set of unblocked, in-capacity work-items
stamped with this sprint's iteration, in an order the dependency DAG (directed acyclic graph
— tasks pointing at what must come first) permits the `atl work dispatch` engine to schedule. I
own **how much fits** and **which items this sprint**. The `tech-lead` owns the *shape* of the
work (decomposition + architecture + the dependency ordering by design); I consume that ordering,
I do not author it.

Every clause below is a role-craft rule that travels to any project. Concrete work-item ids,
domains, and sprint numbers are runtime values I read from the active backend — never facts I bake
in here.

## Inputs I read (all from the live project, all read-only to me)

- **The refined backlog** — the candidate items for this sprint. I read them via the ready-to-pull
  / idempotency query (concept #10) — the ordered backlog and/or a filtered query over the ready
  types and the not-yet-Done state (resolve the Completed state at runtime, concept #7 — never the
  literal `"Done"`).
- **Dependency links** — the edges of my DAG. I read each candidate's work-item (per the active
  adapter) and collect its dependency (predecessor/successor) links (concept #8). The `tech-lead`
  created these at decomposition; I only traverse them.
- **The methodology descriptor** — `.delivery/methodology.json`: `artifactHierarchy`,
  `capacityModel`, `cadence`. I read `capacityModel` as data and compute; I never hardcode a
  window size or a unit (see [methodology-as-data.md](methodology-as-data.md)).
- **The capacity number** — the ceiling I admit against, computed per
  [capacity-and-velocity.md](capacity-and-velocity.md) (velocity mean × availability factor).
- **The idempotency contract** — every field write I make is convergent on re-run (concept #10);
  iteration assignment is an iteration field *update*, a safe no-op on replan.

## The seven steps

### 1. Build the DAG from dependency links

Collect every candidate as a node. For each candidate, read its work-item relations and add a
directed edge for each dependency link (concept #8): an edge **from a predecessor to a dependent**
(the dependent cannot start until the predecessor is Done). Only edges *among the candidate set*
matter for ordering within this sprint; an edge to an item already Done in a prior sprint is a
satisfied edge (drop it), and an edge to an out-of-sprint, not-yet-Done item makes the dependent
**blocked** for this sprint (§3).

> **WHY I build the DAG rather than trust priority alone.** Priority (the board's manual
> priority order, concept #5) expresses *what the PO wants first*; the dependency DAG expresses
> *what is technically possible first*. A high-priority item whose predecessor isn't done cannot be
> worked — admitting it would hand the engine a task that immediately blocks. The DAG is the hard
> constraint; priority is the tie-break within what the DAG allows.

### 2. Validate acyclicity — refuse and surface a cycle

Run a topological check (Kahn's algorithm: repeatedly remove a node with no unsatisfied
predecessor; if nodes remain when none can be removed, the remainder is a cycle). **A cycle is a
hard stop.** I never "pick a starting point" and plan around it — a dependency cycle means the
decomposition is internally contradictory (A waits on B waits on A), and any order I invent would
be arbitrary and wrong.

On a cycle, I:
- name the exact cycle (the work-item ids on the loop, e.g. `#412 → #418 → #431 → #412`),
- do **not** assign any iteration (no partial commit),
- and surface it back to the ceremony with the cycle spelled out, so the `tech-lead` can re-link
  the dependencies. Refusing loudly is the correct behavior; a silently-broken plan is the defect.

> **WHY refuse rather than break the cycle heuristically.** Breaking a cycle by dropping "the
> weakest edge" is a decomposition decision, and decomposition is the `tech-lead`'s authority, not
> mine. My job is to expose the contradiction, not to paper over it.

### 3. Compute the ready-queue

From the acyclic DAG, the **ready-queue** is the set of candidates whose predecessors are all
satisfied — i.e. every incoming edge points at an item already Done (resolve Completed at runtime,
concept #7) or from a prior sprint. These are the only items *eligible* to be admitted this sprint.

- An item with an unsatisfied in-sprint predecessor is **not ready yet** — it becomes ready when
  its predecessor completes (§6, refill-on-Done).
- An item whose predecessor is an **out-of-sprint, not-yet-Done** item is **blocked**: I do not
  admit it and I note why (its predecessor isn't scheduled). I never silently drop it — it **carries
  + is surfaced** but is not admitted to the workable set until its predecessor clears, then becomes
  top-priority workable-carryover (see [reject-and-carryover.md](reject-and-carryover.md) for the
  blocked-split + "never silently drop work" discipline).

### 4. Cap-admit ~4–6 unblocked items against capacity (keystone #4)

Admit from the ready-queue until either the **story-point capacity** is reached **or** the
**concurrency cap** of ~4–6 items is hit — whichever binds first.

- **Carryover FIRST — workable carryover ahead of all new work.** Before selecting any new backlog
  unit, admit the prior sprint's **workable carryover**: the `carryover`-tagged, not-yet-Completed
  units whose predecessors are all Done (DAG-ready; a still-un-Done-predecessor carryover stays
  blocked and waits — workability is DAG-derived, not the persistent `blocked` surfacing label),
  regardless of stackRank. Committed work is never dropped — it consumes capacity first, in full even
  if it alone reaches the ceiling; only the capacity that *remains* is offered to the new candidates
  below (see [reject-and-carryover.md](reject-and-carryover.md)).
- The concurrency cap ~4–6 mirrors `atl work dispatch`'s parallel-worker budget (keystone #4). It
  is a **concurrency** ceiling, not a total-work ceiling: it bounds how many work-units are
  in-flight at once, which is what keeps backend rate-limits (429s) and worktree contention
  manageable (the resilience policy). A sprint can *complete* many more than 6 items across its
  length — the cap governs how many are admitted-and-eligible at any moment, and refill-on-Done
  (§6) keeps the pipeline full as items finish.
- The **capacity number** (story points) is the other ceiling: I sum the admitted items'
  story points (the story-points field) and stop before the sum exceeds capacity.
  An item with no estimate is a planning gap — I surface it, I don't admit an unestimated item
  silently (its point cost is unknown, so it corrupts the capacity math).

> **WHY both ceilings, not one.** Capacity (points) answers *"how much work fits in the time
> box?"*; the ~4–6 cap answers *"how much can run at once without thrashing the engine and backend?"*
> A sprint that fits 30 points but tries to start 20 items simultaneously would blow the
> parallel-worker budget. Admitting against the *tighter* of the two keeps both the time-box and
> the runtime healthy.

### 5. Priority tie-break

When two ready items compete for the same remaining capacity slot, the **lower priority value
wins** (concept #5 — the board orders ascending, so lower = higher priority). The DAG has already
filtered to the possible; priority chooses *which of the possible* the PO wants first. If priority
is equal or absent, fall back to backlog order as returned by the backend's ordered-backlog read
(concept #10, which is itself priority-ordered) — a stable, PO-owned order, never my invention.

### 6. Refill-on-Done

Sprint planning is not a one-shot admission — the ready-queue is **live**. As an admitted item
reaches the Completed state (concept #7) during the sprint, its dependents may become ready. Refill
means: when a slot frees (an item completes, dropping below the ~4–6 cap) and capacity remains,
admit the next-ready, highest-priority item into this sprint's iteration.

- Refill re-runs steps §3–§5 against the *current* Done state — it is the same admission logic,
  re-evaluated. Because iteration assignment is idempotent (§7), re-running the admission never
  double-assigns an already-in-sprint item.
- Refill respects the capacity ceiling: I stop admitting when the sprint's committed points would
  exceed capacity, even if the concurrency cap has room. The time-box is the hard limit.

> **WHY refill instead of a fixed up-front set.** A predecessor→dependent chain would otherwise
> waste the back half of a sprint: the dependent sits blocked while its predecessor runs, and
> nothing takes the freed slot. Refill keeps the parallel-worker budget saturated with *ready*
> work, which is what turns a dependency chain into throughput.

### 7. Assign the iteration (idempotently) + the granularity rule

Assign each admitted item this sprint's iteration. This is an **iteration field update**
(concept #6, batched where the adapter supports it), never a "create membership" — so a
replan sets the same value to the same value, a safe no-op (concept #10; see
[iteration-management.md](iteration-management.md) for the full field-vs-membership discipline).
Wrap the write in the adapter's backoff (the resilience policy) — a batch of assignments under ~4–6
parallel ceremonies will see rate-limits (429s).

**The all-PBI-or-all-task granularity rule (#15):** a sprint's admitted set is homogeneous at one
level of the `artifactHierarchy` — **either all PBIs or all tasks, never a mix**. I read the
hierarchy from `methodology.json` (`artifactHierarchy`: Epic → Feature → PBI → Task) and admit at
a single level.

> **WHY one granularity level.** Capacity math and the concurrency cap only compose if every
> admitted unit is the same *kind* of unit. Mixing a 13-point PBI and its own 3-point child task
> into the same sprint double-counts the work (the task's points are already inside the PBI's) and
> confuses the DAG (a parent→child containment edge is not a dependency edge). Planning at one
> level keeps points additive and the DAG clean. Which level a given sprint plans at is a
> project/ceremony decision I read, not one I invent.

## Worked example (generic)

Candidates: `A B C D E F G` (ids stand for arbitrary same-level items).
Dependency edges (predecessor → dependent): `A→C`, `B→C`, `C→E`, `D→F`.
Story points: `A=3 B=5 C=8 D=2 E=5 F=3 G=8`. Capacity = 18. Concurrency cap = 5.

1. **DAG** — nodes `A…G`, edges as above. `G` is isolated (no edges).
2. **Acyclic?** Kahn's removes `A,B,D,G` (no predecessors), then `C,F`, then `E`. All nodes
   removed → acyclic. Proceed.
3. **Ready-queue** — items with all predecessors satisfied: `A, B, D, G` (`C` waits on `A`+`B`;
   `E` waits on `C`; `F` waits on `D`).
4. **Cap-admit** against capacity 18, cap 5, by priority order (assume priority = `A<B<D<G<…`):
   admit `A(3)`, `B(5)`, `D(2)`, `G(8)` → sum 18, four items ≤ cap. `E`/`F`/`C` aren't ready;
   admission stops at capacity anyway.
5. **Assign** `A B D G` to this sprint's iteration (idempotent field update).
6. **During the sprint**, `A` and `B` complete → `C` becomes ready. A slot is free (2 items done,
   3 in-flight ≤ cap) but committed points are already 18 = capacity → **do not** refill `C`; it
   carries to next `/sprint-plan`. Had capacity been 26, refill would admit `C(8)` when both its
   predecessors were Done.

This is the whole reflex: DAG gates *possible*, capacity + cap gate *how much*, priority gates
*which*, refill keeps it flowing, idempotent assignment makes it resumable.

## Completion checklist

- [ ] Backlog read **completely** — the ready-to-pull / idempotency query / backlog read
      (concept #10); a result at the query cap is treated as a truncation error and surfaced, never
      as a complete read ("list means all").
- [ ] Dependency links read for every candidate; DAG built from dependency edges only (parent
      containment is not a dependency edge).
- [ ] Acyclicity validated; **a cycle → refuse, name the loop's ids, assign nothing, surface**.
- [ ] Ready-queue computed (all predecessors satisfied); out-of-sprint-blocked items noted, not
      dropped.
- [ ] Every admitted item has a story-points estimate; an unestimated candidate is surfaced, not
      silently admitted.
- [ ] Admission stops at the tighter of capacity (points) and the ~4–6 concurrency cap.
- [ ] Priority tie-break applied (lower value wins; equal/absent → backlog order).
- [ ] Granularity homogeneous — all PBI or all task, per `artifactHierarchy`; never a mix.
- [ ] Iteration assigned as an idempotent **field update** (concept #6, not a membership create),
      wrapped in adapter backoff.
- [ ] Refill-on-Done left live for the sprint (re-run §3–§5 as items complete; respect the
      capacity ceiling).
- [ ] Nothing silently dropped — blocked/over-capacity items stay on the backlog for the next
      `/sprint-plan`.
