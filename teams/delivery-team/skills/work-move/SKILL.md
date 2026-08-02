---
name: work-move
description: /work-move <id> <state> — the drive loop's only state writer. Moves a work item between the backend's real states, writing every surface a state spans in one pass (on GitHub, Done means both the Status field and a closed issue) and reading the result back to confirm it landed. Refuses an illegal transition with the legal ones named, requires a reason for blocking and for reopening, and refuses Done when the unit's PR has not actually merged. Mutating; run it explicitly.
---

# /work-move — the one place state changes

Every other skill in the drive loop deliberately does not touch state. `/work-start` claims a unit
and `/work-finish` opens its PR, but neither marks anything done. This skill is where a unit's
state changes, and concentrating that in one place is what makes the board's history readable:
one writer, one transition table, one verification.

Reads its contracts from [`config-and-methodology.md`](../../knowledge/config-and-methodology.md),
[the backend interface](../../knowledge/backend-interface.md), and the active
`backends/<backend>/adapter.md`. **Never invent a tool name.**

## When to run

- **After a PR merges** — move the unit to Done. `/work-finish` deliberately leaves it alone.
- **When work stalls** — mark it blocked, with the reason.
- **When a closed unit turns out to be unfinished** — reopen it, with the reason.
- `/work-sync` finds the ones that were forgotten and offers this skill's transitions in batch.

## Backend support

**GitHub is bound; Azure halts** — same reason as the rest of the loop. There is an additional,
sharper reason here: on Azure there are no canonical state literals at all. States must be resolved
at run time from the work-item type's state categories, and the adapter is explicit that a literal
`"Done"` or `"Active"` must never be written into a skill. A state-writing skill against an
unverified operation map would be inventing both the call and the value.

## The state model on this backend — two surfaces, not one

This is the part most likely to be got wrong, because "state" is not one field:

| Move | What actually changes |
|---|---|
| **In Progress** | the Projects v2 **Status** field → `In Progress`. The issue stays open. |
| **blocked** | a `blocked` **label**, plus a diagnostic comment. **Status does not change** and the issue stays open — blocked is not a state, it is an annotation on the state the unit is already in. |
| **Done** | **both**: `gh issue close <n>` **and** the Status field → its Done option. |
| **reopen** | `gh issue reopen <n>` and Status → `In Progress`, plus a reason comment. |

Two consequences to hold onto. **Done needs both writes** — a closed issue whose Status still reads
In Progress, or a Done Status on an open issue, is a half-moved unit that every consumer reads
differently. And **a label is never removed to mean done**: completion is a state, and stripping
the sprint carrier or the area tag to signal it corrupts the very fields the sprint queries run on.

The built-in project automation only covers the Done end (`item closed → Done`, `PR merged →
Done`). Nothing sets In Progress for you, which is exactly why a board can read "0 in progress"
while real work is underway.

## Procedure

### 1. Resolve and read

Read `.delivery/config.json` for `backend` and `branchPair.dev`. Read the item by id, and read its
**current** Status and open/closed state — the transition check needs where it is now, and a
snapshot from earlier in the session may be stale.

### 2. Check the transition against the table

| From | Legal to |
|---|---|
| Todo | In Progress · Done (a unit closed without work — needs a reason) |
| In Progress | Done · blocked · Todo (un-claim) |
| blocked | In Progress · Todo |
| Done | reopen |

Anything else is refused **with the legal moves named**, so the refusal is actionable:

```
/work-move: #<id> is Done; "in progress" is not a legal move from there.
Legal: reopen. Reopening asks for a reason and posts it to the item.
```

### 3. Require a reason where a reason is the point

**blocked**, **reopen**, and **Done-without-a-merged-PR** each take a one-line reason, asked for
before anything is written and posted to the item as a plain comment.

Post the reason **before** the state write, not after. If the write fails, a unit carrying an
unexplained state change is worse than one carrying an explanation of a change that did not land.

Comment, never a history/revision field: the item's own change log is the backend's, and prose
written into it is not readable where humans look.

### 4. Guard Done against an unmerged PR

Merge comes first, Done after — so a Done never fronts an unlanded merge.

Before moving to Done, check the unit's PR. If it exists and has **not** merged into
`branchPair.dev`, warn and require confirmation:

```
/work-move: #<id>'s PR (#<n>) has not merged into <dev>.
Done would claim work that is not on the integration branch. Merge it first,
or confirm you mean to close this without the merge.
```

If the unit has no PR at all, that is legitimate — not every unit produces code — but say so in
the report rather than passing over it.

The engine verifies the same property deterministically (`git rev-list --count origin/<dev>..<branch>`
must be zero) and marks a unit blocked rather than done when it is not. A hand-driven unit should
not get a weaker guarantee than an engine-driven one just because a human is at the keyboard.

### 5. Write every surface the move spans — then read it back

Write the surfaces from the table above. For **Done**, that is the close **and** the Status field;
do not stop after one.

Then **read the item back** and confirm the new values are actually there. A write that reported
success and did not land is the failure mode this loop is built to make impossible, and a read-back
is two seconds.

If the read-back disagrees with what you wrote, **stop and report the mismatch**. Do not retry in a
loop and do not report success.

### 6. Report

```
#<id> — <title>
  status:  <before> → <after>
  issue:   <open|closed>[ → <open|closed>]
  reason:  <the one-liner, when one was required>
  verified: read back, both surfaces agree
```

## Idempotent re-run

Moving a unit to the state it is already in is a **no-op, not a rewrite**. Skip the write entirely
rather than writing identical values: a no-op write still bumps the item's revision and changed-date,
which makes the board's history lie about when the transition happened — and those timestamps are
what a sprint review reads.

An illegal transition self-rejects, so repeated invocation is safe by construction: the table, not
a lock, is what keeps this skill from doing damage twice.
