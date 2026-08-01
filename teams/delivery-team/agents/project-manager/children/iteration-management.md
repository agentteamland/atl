---
knowledge-base-summary: "Sprint-membership bookkeeping, mode-selected. Under the scrum mode the carrier is the ITERATION FIELD (concept #6): list/create/assign iterations through the active backend's adapter; assignment is an idempotent field update, never a create-membership that could double; concrete iteration names resolved at runtime rather than hardcoded. Under the flow mode there is no schedule at all — the carrier is the sprint:<slug> LABEL (concept #4): the ordinal is resolved from the labels already on the board, the add is idempotent for the same reason, and re-admission SWAPS the label because a label accumulates where a field replaces."
---

# Iteration Management

Iterations are the sprints my planning admits into. This topic is the plumbing under the blueprint:
how I make sure the right sprint exists, how items join it *idempotently*, and how I always talk to
the active backend in the *concrete* iteration names it uses — never a hardcoded label. Craft that
travels: the mechanism is fixed, every concrete iteration name is a runtime value.

> **Which carrier this page is about.** A sprint needs a **carrier** — the durable mark that says
> *this unit belongs to this sprint* — and `methodology.json`'s `mode` selects it
> ([`config-and-methodology.md`](../../../knowledge/config-and-methodology.md) §1.2). Under
> `mode: "scrum"` it is the **iteration field**, and everything from here down to *Creating an
> iteration* is that craft. Under `mode: "flow"` there is **no iteration and no schedule at all** —
> the carrier is the **`sprint:<slug>` label**, which has its own section — *The `flow` carrier* —
> below. I read `mode` first and follow one branch; I never dual-write both.

## Two distinct operations — don't conflate them

The backend separates **the iteration existing on the team's schedule** from **subscribing a team to
that iteration** from **a work-item pointing at that iteration**. I keep them straight:

| Operation (concept #6, the active adapter binds the tool) | Nature |
|---|---|
| List the iterations that exist / the team is subscribed to (its sprint schedule) | read |
| Create an iteration node (if the schedule is missing one) | write |
| Subscribe the team to an iteration | write |
| Point a **work-item** at a sprint (put it *in* the sprint) — an iteration **field** set | write (field) |

> **WHY the distinction is load-bearing.** Subscribing the *team* to an iteration is a schedule
> operation, done once per sprint at setup. Putting a *work-item* into a sprint is a completely
> different thing — an iteration **field** on the item. Confusing the two is the classic idempotency
> trap: a "membership create" repeated on replan would try to add the item twice; a field *update*
> repeated sets the same value — a safe no-op.

## Assigning an item to a sprint = an idempotent field update

When my blueprint admits an item ([sprint-planning-blueprint.md](sprint-planning-blueprint.md) §7),
I set its iteration field to this sprint via the active backend's work-item update (batching the
admitted set into one call where the adapter supports it — batch reads/writes collapse N calls into
one, the resilience policy).

- This is **idempotent by nature** (concept #10): re-running `/sprint-plan` after a crash or a
  re-plan sets the same iteration to the same value. There is nothing to dedup, no "already a
  member" error to catch — the field simply holds the value it should.
- Because assignment is a plain field update, it composes cleanly with the idempotency contract's
  `atl-key` stamping that the `tech-lead` applies at *creation*: I don't create items, so I don't
  stamp keys; I only *update the iteration field* of items that already exist. My idempotency story
  is entirely "field update = convergent".
- Wrap the write in exponential backoff + jitter, honour `Retry-After` (the resilience policy) — a
  batch of assignments under the ~4–6 parallel-worker load will hit rate-limits (429s), which are
  expected, not failures.

> **WHY never model it as create-membership.** A create-membership operation has a "does it already
> exist?" question, and getting that check wrong on a re-run either duplicates or errors. A field
> update has no such question — the value is the value. Modeling assignment as a field update is
> what makes the whole plan *resumable* without a local ledger.

The team-subscription operation (concept #6) is likewise a safe re-run: subscribing an
already-subscribed team is a no-op. But I reach for it only at *setup*, not per work-item.

## Resolving concrete iteration names at runtime — never hardcode a sprint label

The abstract cadence lives in `methodology.json` (`cadence.unit: "sprint"`), but the **concrete
iteration** — the actual node name in the project's iteration schedule — is a live backend fact
I must resolve, never guess.

- Listing the backend's iterations (concept #6) returns the real iteration nodes with their names
  and date ranges. I read them to find **the current/next sprint's actual name** and
  **its number `<n>`** (for the `Sprints/Sprint-<n>-Review` durable-knowledge page,
  [sprint-review-report.md](sprint-review-report.md)).
- A project's sprint might be named `Sprint 7`, `2024-Iteration-12`, `\Project\Sprint 7`, or a
  custom scheme — I resolve the string from the active backend and use it verbatim; I never
  construct `"Sprint 7"` from an assumption.
- This mirrors the runtime-resolution discipline for types/states (resolve the completion/state
  model at runtime, concept #7): the descriptor holds *intent* (a "sprint" cadence), the live
  project holds *concrete names* — I bridge intent→concrete by *reading*, per the config read
  contract ([methodology-as-data.md](methodology-as-data.md)).

> **WHY resolve rather than hardcode.** A hardcoded `"Sprint N"` breaks the moment a project uses a
> different naming scheme, an iteration-name prefix, or dated iterations. Resolving at runtime is
> what makes the same planning craft work on any project on any backend with any iteration-naming
> convention, with zero per-project change to me.

## Creating an iteration — the rare write

Usually the sprints already exist (a team's schedule is set up outside the delivery loop). If a
needed iteration is genuinely missing, I create the iteration node and subscribe the team (both
concept #6) — but I do this only when the schedule truly lacks the sprint I'm planning, and I check
first by listing the existing iterations (found → reuse, per the check-first discipline).
Fabricating iterations the PO didn't intend is scope creep; the default posture is *read the
schedule, plan into what exists*.

## The `flow` carrier — the `sprint:<slug>` label

Under `mode: "flow"` there is no schedule: no iteration node to list, none to create, no team to
subscribe, no date range to resolve. **A flow sprint has no object on the backend at all** — it
exists only as the set of units carrying its label. So every section above collapses into one
operation: **add a label**.

**The shape is `sprint:<slug>`**, and `<slug>` is the sprint's **ordinal** — a positive decimal
integer, unpadded, no leading zeros (`sprint:1`, `sprint:2`, … `sprint:14`), so the whole label
matches `sprint:[0-9]+` (concept #4; full contract in
[`config-and-methodology.md`](../../../knowledge/config-and-methodology.md) §1.2).

**Resolving the ordinal — the flow twin of resolving an iteration name.** I list the `sprint:*`
labels already on the board ("list means all", concept #10 — a result at the query cap is a
truncation to surface, never a complete read) and take the highest ordinal `k`, **compared as an
integer** (`sprint:10` outranks `sprint:9`; a lexical "highest" hands back a stale ordinal and the
sprint I open reuses a number already in use). `sprint:<k>` is the
**current** sprint and stays current until it is **reviewed** — reviewed meaning its
`Sprints/Sprint-<k>-Review` page exists (concept #9), the flow analogue of a *closed* iteration. So
a not-yet-reviewed `sprint:<k>` is the sprint I plan into; a reviewed one means I open
`sprint:<k+1>`. A board with no `sprint:*` label starts at `sprint:1`, except that a project
**migrating from scrum** continues its existing numbering: I take the highest ordinal `m` among the
existing `Sprints/Sprint-<m>-Review` pages and open `sprint:<m+1>` — those pages are the reliable
read, since iteration *names* are arbitrary. Whichever ordinal that resolves to is
also the `<n>` of the review page ([sprint-review-report.md](sprint-review-report.md)): under
`scrum` I read `<n>` off the resolved iteration, under `flow` off the label.

**The idempotency story is the same one, for the same reason.** Adding a label that is already
there is a no-op, so a replan or a crash-resumed run converges exactly as the field update does —
and for the identical reason: there is no "does it already exist?" question to get wrong. I never
model membership as a create-membership operation. Same backoff, same batching, same resilience
policy.

**The one difference a label brings — the swap.** A field *replaces* its value; a label
*accumulates*. So re-admitting a carryover unit into the next sprint is two moves in one step:
**remove the `sprint:` label the unit actually carries, add `sprint:<n>`**. I read that old ordinal
off the unit rather than assuming `sprint:<n-1>`: a unit that stayed blocked through one or more
sprints was never re-admitted, so it still carries the ordinal of the last sprint it *was* in (see
[reject-and-carryover.md](reject-and-carryover.md) — a blocked carryover keeps its old label until
it unblocks). Never leave both.

> **WHY at most one `sprint:` label, and why the swap is mine to do by hand.** Two `sprint:` labels
> on one unit is a corrupt state, not a history: "which sprint is this in?" stops having an answer,
> and the sprint's item read starts returning units that moved on — which silently poisons the DAG
> `/sprint-start` builds from it. The scrum carrier gets this property for free because a field can
> only hold one value; the label carrier has to be *disciplined* into it. Nothing is lost by the
> swap that isn't also lost under scrum, where the field is likewise overwritten: the durable
> membership record is the `Sprints/Sprint-<n>-Review` page, not the label.

And a label is **never** removed to mean "done" — completion is a state (concept #7) and always
was. A unit finishing keeps its sprint label; that is what makes the sprint's membership readable
after the fact.

## Worked example (generic) — `mode: "scrum"`

1. `/sprint-plan` runs. I need "the next open sprint's iteration."
2. Listing the backend's iterations (concept #6) → the schedule's iteration nodes with date ranges;
   I pick the one whose range contains/next-follows today. Say its resolved identifier is `Sprint 8`
   (some backends spell it as a tree path like `\Proj\Release-2\Sprint 8`, others as a plain field
   value). I use that exact value verbatim — I do not build it.
3. `<n> = 8` for the review page (`Sprints/Sprint-8-Review`).
4. My blueprint admits items `A B D G`; I set each item's iteration to that resolved value via one
   batched work-item update (concept #6), wrapped in backoff.
5. A crash mid-batch → re-run: the same batch sets the same iteration values; the already-set items
   are safe no-ops, the unset ones get set. Convergent, no ledger, no dedup logic.

## The same example under `mode: "flow"`

1. `/sprint-plan` runs. I need "the sprint I am planning into" — there is no schedule to consult.
2. Listing the board's `sprint:*` labels (concept #4) → `sprint:1 … sprint:7`. Highest ordinal
   `k = 7`. `Sprints/Sprint-7-Review` **exists**, so sprint 7 is reviewed and I open **`sprint:8`**.
   (Had that page been absent, `sprint:7` would still be current and I would plan into it — that is
   what makes a re-run land on the same sprint instead of opening a new one every time.)
3. `<n> = 8` for the review page (`Sprints/Sprint-8-Review`) — same page name, read off the label
   instead of an iteration.
4. My blueprint admits items `A B D G`; I add `sprint:8` to each in one batched update, wrapped in
   backoff. `D` is a carryover still carrying `sprint:7` → for `D` the step is **remove `sprint:7`,
   add `sprint:8`**.
5. A crash mid-batch → re-run: `sprint:7` is already gone from `D` (nothing to remove) and
   `sprint:8` is already present on the units that got it (nothing to add); the rest get stamped.
   Convergent, no ledger, no dedup logic — and the ordinal re-resolves to `8`, not `9`, because
   `Sprints/Sprint-8-Review` does not exist yet.
