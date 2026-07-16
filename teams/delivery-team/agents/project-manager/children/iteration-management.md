---
knowledge-base-summary: "Iteration bookkeeping (concept #6): list/create/assign iterations through the active backend's adapter; assigning an item to a sprint is an idempotent iteration FIELD update, never a create-membership that could double; and resolving concrete iteration names at runtime rather than hardcoding a sprint label."
---

# Iteration Management

Iterations are the sprints my planning admits into. This topic is the plumbing under the blueprint:
how I make sure the right sprint exists, how items join it *idempotently*, and how I always talk to
the active backend in the *concrete* iteration names it uses — never a hardcoded label. Craft that
travels: the mechanism is fixed, every concrete iteration name is a runtime value.

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

## Worked example (generic)

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
