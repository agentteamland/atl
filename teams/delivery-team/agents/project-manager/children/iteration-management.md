---
knowledge-base-summary: "Iteration bookkeeping: list/create/assign iterations (work_list_iterations / work_create_iterations / work_assign_iterations); assigning an item to a sprint is an idempotent IterationPath FIELD update (wit_update_work_item), never a create-membership that could double; and resolving concrete iteration names/paths at runtime rather than hardcoding a sprint label."
---

# Iteration Management

Iterations are the sprints my planning admits into. This topic is the plumbing under the blueprint:
how I make sure the right sprint exists, how items join it *idempotently*, and how I always talk to
Azure in the *concrete* iteration names it uses — never a hardcoded label. Craft that travels: the
mechanism is fixed, every concrete path is a runtime value.

## Two distinct operations — don't conflate them

Azure separates **the iteration existing on the team's schedule** from **subscribing a team to that
iteration** from **a work-item pointing at that iteration**. I keep them straight:

| Concept | Tool | Nature |
|---|---|---|
| The iterations that exist / the team is subscribed to (its sprint schedule) | `work_list_iterations` | read |
| Create an iteration node (if the schedule is missing one) | `work_create_iterations` | write |
| Subscribe the team to an iteration | `work_assign_iterations` | write |
| Point a **work-item** at a sprint (put it *in* the sprint) | `wit_update_work_item` (`IterationPath`) | write (field) |

> **WHY the distinction is load-bearing.** `work_assign_iterations` subscribes the *team* to an
> iteration (a schedule operation, done once per sprint at setup). Putting a *work-item* into a
> sprint is a completely different thing — an `IterationPath` field on the item. Confusing the two
> is the classic idempotency trap: a "membership create" repeated on replan would try to add the
> item twice; a field *update* repeated sets the same value — a safe no-op.

## Assigning an item to a sprint = an idempotent field update

When my blueprint admits an item ([sprint-planning-blueprint.md](sprint-planning-blueprint.md) §7),
I set its `System.IterationPath` field to this sprint's path via `wit_update_work_item` (or
`wit_update_work_items_batch` for the admitted set in one call — batch reads/writes collapse N
calls into one, adapter §3).

- This is **idempotent by nature** (adapter §5): re-running `/sprint-plan` after a crash or a
  re-plan sets the same `IterationPath` to the same value. There is nothing to dedup, no "already a
  member" error to catch — the field simply holds the value it should.
- Because assignment is a plain field update, it composes cleanly with the idempotency contract's
  `atl-key` stamping that the `tech-lead` applies at *creation*: I don't create items, so I don't
  stamp keys; I only *update the iteration field* of items that already exist. My idempotency story
  is entirely "field update = convergent".
- Wrap the write in exponential backoff + jitter, honour `Retry-After` (adapter §3) — a batch of
  assignments under the ~4–6 parallel-worker load will hit 429s, which are expected, not failures.

> **WHY never model it as create-membership.** A create-membership operation has a "does it already
> exist?" question, and getting that check wrong on a re-run either duplicates or errors. A field
> update has no such question — the value is the value. Modeling assignment as a field update is
> what makes the whole plan *resumable* without a local ledger.

`work_assign_iterations` (team subscription) is likewise a safe re-run: subscribing an
already-subscribed team is a no-op. But I reach for it only at *setup*, not per work-item.

## Resolving concrete iteration names at runtime — never hardcode a sprint label

The abstract cadence lives in `methodology.json` (`cadence.unit: "sprint"`), but the **concrete
IterationPath** — the actual node name in the project's classification tree — is a live Azure fact
I must resolve, never guess.

- `work_list_iterations` returns the real iteration nodes with their paths, names, and date
  ranges. I read them to find **the current/next sprint's actual path** and
  **its number `<n>`** (for the `Sprints/Sprint-<n>-Review` page,
  [sprint-review-report.md](sprint-review-report.md)).
- A project's sprint might be named `Sprint 7`, `2024-Iteration-12`, `\Project\Sprint 7`, or a
  custom scheme — I resolve the string from Azure and use it verbatim; I never construct
  `"Sprint 7"` from an assumption.
- This mirrors the runtime-resolution discipline for types/states (resolve via
  `wit_get_work_item_type`, adapter §6): the descriptor holds *intent* (a "sprint" cadence), the
  live project holds *concrete names* — I bridge intent→concrete by *reading*, per the config read
  contract ([methodology-as-data.md](methodology-as-data.md)).

> **WHY resolve rather than hardcode.** A hardcoded `"Sprint N"` breaks the moment a project uses a
> different naming scheme, an iteration path prefix, or dated iterations. Resolving at runtime is
> what makes the same planning craft work on any Azure project with any iteration-naming
> convention, with zero per-project change to me.

## Creating an iteration — the rare write

Usually the sprints already exist (a team's schedule is set up outside the delivery loop). If a
needed iteration is genuinely missing, `work_create_iterations` adds the node and
`work_assign_iterations` subscribes the team — but I do this only when the schedule truly lacks the
sprint I'm planning, and I check first with `work_list_iterations` (found → reuse, per the
check-first discipline). Fabricating iterations the PO didn't intend is scope creep; the default
posture is *read the schedule, plan into what exists*.

## Worked example (generic)

1. `/sprint-plan` runs. I need "the next open sprint's path."
2. `work_list_iterations` → the schedule's iteration nodes with date ranges; I pick the one
   whose range contains/next-follows today. Say its `path` is `\Proj\Release-2\Sprint 8` and its
   `name` is `Sprint 8`. I use that exact `path` — I do not build it.
3. `<n> = 8` for the review page path (`Sprints/Sprint-8-Review`).
4. My blueprint admits items `A B D G`; I set each item's `IterationPath = \Proj\Release-2\Sprint 8`
   via one `wit_update_work_items_batch` call, wrapped in backoff.
5. A crash mid-batch → re-run: the same batch sets the same paths; the already-set items are safe
   no-ops, the unset ones get set. Convergent, no ledger, no dedup logic.
