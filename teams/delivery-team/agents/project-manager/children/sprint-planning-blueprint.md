---
knowledge-base-summary: "My primary production unit: the /sprint-plan contribution. Build the dependency DAG from dependency links (concept #8), validate acyclicity (refuse + surface the cycle, never plan around it), compute the ready-queue, cap-admit ~4-6 unblocked items — by story points ≤ capacity under the scrum mode, by DAG readiness alone with no point budget under the flow mode — priority tie-break, refill-on-Done, enforce the all-PBI-or-all-task granularity rule, and stamp the sprint carrier idempotently (the iteration field under scrum, the sprint:<slug> label under flow, swapped on re-admission). Full checklist."
---

# Sprint Planning (blueprint)

This is my primary production unit. When the `/sprint-plan` ceremony spawns me as a subagent, I
turn the refined backlog into a committed sprint: a set of unblocked work-items stamped with this
sprint's **carrier**, in an order the dependency DAG (directed acyclic graph
— tasks pointing at what must come first) permits the `atl work dispatch` engine to schedule. I
own **how much fits** and **which items this sprint**. The `tech-lead` owns the *shape* of the
work (decomposition + architecture + the dependency ordering by design); I consume that ordering,
I do not author it.

**The `mode` fork, read once up front.** `.delivery/methodology.json` carries a `mode` — `"scrum"`
or `"flow"`, **absent ⇒ `scrum`**, never inferred
([`config-and-methodology.md`](../../../knowledge/config-and-methodology.md) §1.1). It changes
exactly two things in this blueprint, both of them in §4 and §7:

| | `mode: "scrum"` | `mode: "flow"` |
|---|---|---|
| The ceiling I admit against (§4) | the **story-point capacity** from [capacity-and-velocity.md](capacity-and-velocity.md), *and* the ~4–6 concurrency cap — whichever binds first | the ~4–6 concurrency cap **only**; there is no point budget, so "how much fits" is a runtime question, not a commitment |
| The carrier I stamp (§7) | the **iteration field** (concept #6) | the **`sprint:<slug>` label** (concept #4), swapped on re-admission |

Everything else below — the DAG (§1), the acyclicity refusal (§2), the ready-queue (§3), the
priority tie-break (§5), the reason refill exists (§6), and the granularity rule that shares §7 —
is **mode-independent craft** and reads the same on either kind of project. Where §5 and §6 speak
of capacity, they mean whichever ceiling §4 established.

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
- **The methodology descriptor** — `.delivery/methodology.json`: `mode`, `artifactHierarchy`,
  `capacityModel` (present under `scrum`, absent under `flow`), `cadence`. I read `capacityModel` as
  data and compute; I never hardcode a
  window size or a unit (see [methodology-as-data.md](methodology-as-data.md)).
- **The capacity number — `scrum` only** — the ceiling I admit against, computed per
  [capacity-and-velocity.md](capacity-and-velocity.md) (velocity mean × availability factor). Under
  `flow` there is no such input: no `capacityModel` to read, no velocity to compute, and no seed to
  ask the PO for.
- **The idempotency contract** — every carrier write I make is convergent on re-run (concept #10);
  iteration assignment is an iteration field *update* and a sprint-label add is an add-if-absent,
  both safe no-ops on replan.

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
- do **not** stamp any unit with the sprint's carrier (§7) — no partial commit,
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
**concurrency cap** of ~4–6 items is hit — whichever binds first. **Under `mode: "flow"` only the
second ceiling exists** — see the flow clause at the end of this step.

- **Carryover FIRST — workable carryover ahead of all new work.** Before selecting any new backlog
  unit, admit the prior sprint's **workable carryover**: the `carryover`-tagged, not-yet-Completed
  units whose predecessors are all Done (DAG-ready; a still-un-Done-predecessor carryover stays
  blocked and waits — workability is DAG-derived, not the persistent `blocked` surfacing label),
  regardless of stackRank. Committed work is never dropped — it consumes capacity first, in full even
  if it alone reaches the ceiling; only the capacity that *remains* is offered to the new candidates
  below (see [reject-and-carryover.md](reject-and-carryover.md)). Under `flow`, read *capacity* here
  as the concurrency cap — the only ceiling there is; the carryover-first **ordering** is
  mode-independent.
- The concurrency cap ~4–6 mirrors `atl work dispatch`'s parallel-worker budget (keystone #4). It
  is a **concurrency** ceiling, not a total-work ceiling: it bounds how many work-units are
  in-flight at once, which is what keeps backend rate-limits (429s) and worktree contention
  manageable (the resilience policy). A sprint can *complete* many more than 6 items across its
  length — the cap governs how many are admitted-and-eligible at any moment, and refill-on-Done
  (§6) keeps the pipeline full as items finish.
- The **capacity number** (story points) is the other ceiling **under `mode: "scrum"`**: I sum the
  admitted items' story points (the story-points field) and stop before the sum exceeds capacity.
  An item with no estimate is a planning gap — I surface it, I don't admit an unestimated item
  silently (its point cost is unknown, so it corrupts the capacity math).
- **Under `mode: "flow"` that second ceiling is absent** — there is no capacity number, so **only
  the ~4–6 concurrency cap binds**. Carryover still comes first, for the same reason; the new units
  then follow in priority order, and admission stops at the cap rather than at a budget. Two
  consequences I must not get wrong: an **unestimated unit is not a planning gap** here (nothing
  reads an estimate, and a flow project need not carry a story-points field at all — I neither
  surface it nor withhold the unit), and **nothing "doesn't fit"** — a unit I don't admit is one
  that is not yet DAG-ready or one the cap deferred to the next refill, never one priced out.

> **WHY both ceilings, not one — and why flow keeps only the second.** Capacity (points) answers
> *"how much work fits in the time box?"*; the ~4–6 cap answers *"how much can run at once without
> thrashing the engine and backend?"* A sprint that fits 30 points but tries to start 20 items
> simultaneously would blow the parallel-worker budget. Admitting against the *tighter* of the two
> keeps both the time-box and the runtime healthy. Under `flow` the first question has no honest
> answer — there is no time box to fit work into, and a velocity mean over a team with no stable
> capacity predicts nothing — so it is not asked. The second question is about the runtime, not the
> methodology, so it survives the mode change untouched.

### 5. Priority tie-break

When two ready items compete for the same remaining slot — a capacity slot under `scrum`, a
concurrency slot under `flow` — the **lower priority value
wins** (concept #5 — the board orders ascending, so lower = higher priority). The DAG has already
filtered to the possible; priority chooses *which of the possible* the PO wants first. If priority
is equal or absent, fall back to backlog order as returned by the backend's ordered-backlog read
(concept #10, which is itself priority-ordered) — a stable, PO-owned order, never my invention.

### 6. Refill-on-Done

Sprint planning is not a one-shot admission — the ready-queue is **live**. As an admitted item
reaches the Completed state (concept #7) during the sprint, its dependents may become ready. Refill
means: when a slot frees (an item completes, dropping below the ~4–6 cap) and capacity remains,
admit the next-ready, highest-priority item into this sprint — stamping it with the sprint's
carrier (§7).

- Refill re-runs steps §3–§5 against the *current* Done state — it is the same admission logic,
  re-evaluated. Because the carrier stamp is idempotent (§7), re-running the admission never
  double-assigns an already-in-sprint item.
- **Under `mode: "scrum"`, refill respects the capacity ceiling**: I stop admitting when the
  sprint's committed points would exceed capacity, even if the concurrency cap has room. The
  time-box is the hard limit.
- **Under `mode: "flow"` there is no ceiling for refill to respect** — a freed slot is filled from
  the ready-queue for as long as ready work exists. So nothing *ends* a flow sprint by exhaustion:
  it ends when `/sprint-review` reviews it (the review page's existence is what makes it closed —
  [`config-and-methodology.md`](../../../knowledge/config-and-methodology.md) §1.2), which is the PO's
  call, not a budget's.

> **WHY refill instead of a fixed up-front set.** A predecessor→dependent chain would otherwise
> waste the back half of a sprint: the dependent sits blocked while its predecessor runs, and
> nothing takes the freed slot. Refill keeps the parallel-worker budget saturated with *ready*
> work, which is what turns a dependency chain into throughput. That reasoning is about the
> dependency graph, not the time box, so it holds identically in both modes — under `flow` refill
> is not an optimization on top of a plan, it *is* the plan.

### 7. Stamp the sprint carrier (idempotently) + the granularity rule

Mark each admitted item as belonging to this sprint. Which mark is mode-selected; the *discipline*
is identical, and [iteration-management.md](iteration-management.md) holds the full
field-vs-membership discipline for both.

- **`mode: "scrum"` — assign the iteration.** An **iteration field update** (concept #6, batched
  where the adapter supports it), never a "create membership" — so a replan sets the same value to
  the same value, a safe no-op (concept #10).
- **`mode: "flow"` — add the `sprint:<n>` label.** A tag/label add (concept #4) of this sprint's
  ordinal, batched the same way. Adding a label already present is a no-op, so it is convergent for
  exactly the reason the field set is. **One difference I must handle by hand:** a label
  *accumulates* where a field *replaces*, so re-admitting a carryover unit **swaps** — remove the
  `sprint:` label the unit actually carries (I read that ordinal off the unit; a unit that stayed
  blocked through one or more sprints still carries the ordinal of the last sprint it *was* in, not
  `sprint:<n-1>`), add `sprint:<n>`, in the same step. Two `sprint:` labels on a unit is a corrupt
  state, not a history: the membership record is the `Sprints/Sprint-<n>-Review` page. A label is
  never removed to mean "done" — completion is a state (concept #7).

Wrap either write in the adapter's backoff (the resilience policy) — a batch of assignments under ~4–6
parallel ceremonies will see rate-limits (429s).

**The all-PBI-or-all-task granularity rule (#15):** a sprint's admitted set is homogeneous at one
level of the `artifactHierarchy` — **either all PBIs or all tasks, never a mix**. I read the
hierarchy from `methodology.json` (`artifactHierarchy`: Epic → Feature → PBI → Task) and admit at
a single level. This rule is **mode-independent**.

> **WHY one granularity level.** Capacity math and the concurrency cap only compose if every
> admitted unit is the same *kind* of unit. Mixing a 13-point PBI and its own 3-point child task
> into the same sprint double-counts the work (the task's points are already inside the PBI's) and
> confuses the DAG (a parent→child containment edge is not a dependency edge). Planning at one
> level keeps points additive and the DAG clean. Which level a given sprint plans at is a
> project/ceremony decision I read, not one I invent. Under `flow` the double-counting half of that
> argument goes quiet (there are no points to add up), but the DAG half does not — a containment
> edge is still not a dependency edge — so the rule stands unchanged.

## Worked example (generic) — `mode: "scrum"`

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

## The same example under `mode: "flow"`

Same candidates, same edges, same priority order. **No story points, no capacity** — the sprint
being planned resolves to `sprint:4` (the highest `sprint:*` ordinal on the board is `3`, and
`Sprints/Sprint-3-Review` exists, so a new one opens). Concurrency cap = 5.

Steps 1–3 are **identical** — the DAG, the acyclicity check, and the ready-queue `A, B, D, G` are
mode-independent. From there:

4. **Cap-admit** against the cap alone, by priority order: admit `A`, `B`, `D`, `G` — four items,
   under the cap of 5, and no budget to check them against. (Had there been a fifth ready unit `H`,
   it would have been admitted too, up to the cap.)
5. **Stamp** `A B D G` with `sprint:4` (idempotent label add).
6. **During the sprint**, `A` and `B` complete → `C` becomes ready and a slot is free → **refill
   `C`** and stamp it `sprint:4`. This is where the modes visibly diverge: under `scrum` the same
   `C` was priced out at 18 = capacity and carried to the next sprint; here nothing prices it out,
   so it runs now. Had `C` instead carried over from `sprint:3`, admitting it would **swap** its
   label — remove `sprint:3`, add `sprint:4`, one step — never leave both.

## Completion checklist

- [ ] Backlog read **completely** — the ready-to-pull / idempotency query / backlog read
      (concept #10); a result at the query cap is treated as a truncation error and surfaced, never
      as a complete read ("list means all").
- [ ] Dependency links read for every candidate; DAG built from dependency edges only (parent
      containment is not a dependency edge).
- [ ] Acyclicity validated; **a cycle → refuse, name the loop's ids, assign nothing, surface**.
- [ ] Ready-queue computed (all predecessors satisfied); out-of-sprint-blocked items noted, not
      dropped.
- [ ] `mode` read from the descriptor (absent ⇒ `scrum`, never inferred) **before** §4 and §7.
- [ ] *(`scrum`)* Every admitted item has a story-points estimate; an unestimated candidate is
      surfaced, not silently admitted. *(`flow`: skipped — nothing reads an estimate; do not
      surface a missing one.)*
- [ ] Admission stops at the tighter of capacity (points) and the ~4–6 concurrency cap — *(`flow`:
      at the concurrency cap alone; there is no capacity to compare it against.)*
- [ ] Priority tie-break applied (lower value wins; equal/absent → backlog order).
- [ ] Granularity homogeneous — all PBI or all task, per `artifactHierarchy`; never a mix.
- [ ] Sprint carrier stamped idempotently: *(`scrum`)* the iteration as a **field update** (concept
      #6, not a membership create), or *(`flow`)* the `sprint:<n>` **label add** (concept #4) with
      the unit's existing `sprint:` label — whatever its ordinal, read off the unit — **removed in
      the same step** on a re-admitted carryover — both wrapped in adapter backoff.
- [ ] Refill-on-Done left live for the sprint (re-run §3–§5 as items complete; *(`scrum`)* respect
      the capacity ceiling — *(`flow`)* there is none, so refill runs while ready work remains).
- [ ] Nothing silently dropped — blocked/over-capacity items stay on the backlog for the next
      `/sprint-plan` *(under `flow`, nothing is over-capacity; what stays back is not-yet-ready or
      cap-deferred)*.
