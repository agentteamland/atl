---
name: work-move
description: /work-move <id> <state|flag> — the drive loop's only state writer. Moves a work item between the backend's real states, writing every surface a state spans in one pass (on GitHub, Done means both the Status field and a closed issue) and reading the result back to confirm it landed. Also sets the annotation flags that are conditions on a state rather than states of their own — blocked, and the verification pair test:pending / test:failed — leaving Status untouched so a unit in verification is still counted in flight. Refuses an illegal transition with the legal ones named, requires a reason for blocking, for reopening and for a failed verification, and refuses Done when the unit's PR has not actually merged or its verification is red. Mutating; run it explicitly.
disable-model-invocation: true
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

- **After a PR merges** — flag the unit `test:pending`. The merge is not the verdict: until someone
  verifies it, the unit is merged-but-unverified, and that is the interval this flag exists to make
  visible. `/work-finish` deliberately leaves the item alone.
- **After verifying a merged unit** — Done if it passed, `test:failed` (with what failed) if not.
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
| **test:pending** | a `test:pending` **label**. **Status does not change** and the issue stays open — merged-but-unverified is a condition on In Progress, not a stage after it. |
| **test:failed** | a `test:failed` **label**, plus a diagnostic comment naming what came back red. **Status does not change** and the issue stays open. |
| **Done** | **both**: `gh issue close <n>` **and** the Status field → its Done option — and clear any `test:` label, because the condition is over. |
| **reopen** | `gh issue reopen <n>` and Status → `In Progress`, plus a reason comment. |

Two consequences to hold onto. **Done needs both writes** — a closed issue whose Status still reads
In Progress, or a Done Status on an open issue, is a half-moved unit that every consumer reads
differently. And **a label is never removed to mean done**: completion is a state, and stripping
the sprint carrier or the area tag to signal it corrupts the very fields the sprint queries run on.

> **That last rule and the `test:` clear on Done are not in conflict** — they are opposite
> directions. The rule forbids *removing a label as the way of signalling* completion: `sprint:3`
> and `area:api` stay on a closed unit, because they record which sprint and which area the work
> belonged to, and the sprint queries still have to find it there. Clearing `test:pending` is the
> reverse — completion is already written on the state axis, and the flag is dropped afterwards
> because the condition it described (awaiting a verdict) has ended. Removing a *fact* that is
> still true is the error; removing a *condition* that has ended is the point. If in doubt: does
> this label still describe something true of the unit after it closed? `sprint:`/`area:` yes,
> `test:pending` no.

The built-in project automation only covers the Done end (`item closed → Done`, `PR merged →
Done`). Nothing sets In Progress for you, which is exactly why a board can read "0 in progress"
while real work is underway.

## Two axes, not one — states and flags

Half the rows above are not states at all, and keeping the two apart is what stops the transition
table from growing until nobody can read it:

- **The state axis** — `Todo` · `In Progress` · `Done` — records **where work is**. It is
  single-valued, and the transition table below governs it.
- **The flag axis** — `blocked` · `test:pending` · `test:failed` — records **what is true about**
  work. Status is untouched, the issue stays open, and flags **compose**: a unit can be blocked
  *while* awaiting verification, and both facts stay readable because a label set holds two values
  where a status field holds one.

The test for which axis something belongs to is a single question: **can a unit be in this and in
something else at the same time?** Yes ⇒ it is a flag. Verification answers yes, which is why
`test:pending` is a label and **not** a `Test` Status column. Making it a column would move the unit
*out* of In Progress, so "how many units are in progress?" and "how many are awaiting verification?"
would become disjoint counts and every unit in verification would vanish from the WIP number — the
board would lose information rather than gain it. It would also oblige the table below to define a
legal move to and from that column for every other state, and leave "blocked while in test" with no
representation at all.

**Flags are set and cleared, never transitioned.** There is no legality table for them and no
"from" state to check — setting a flag on a unit already carrying it is a no-op, and clearing one it
does not carry is also a no-op. They carry exactly one precondition, and it is about the unit rather
than the flag: **the unit must be open** (step 2).

**At most one `test:` flag at a time.** `test:pending` and `test:failed` are mutually exclusive, so
setting one **removes the other in the same write** — on GitHub, a single
`gh issue edit <n> --remove-label test:<prior> --add-label test:<next>`. Drop the `--remove-label`
half when the unit carries no `test:` flag yet; passing an empty or absent label name is a
malformed command, not a no-op. This is the `sprint:` label discipline and it is not optional: a
label set *accumulates* where a field *replaces*, so two `test:` labels on one issue leaves "was
this verified?" with no answer. `blocked` is a different namespace and is never touched by a
`test:` write.

**A `test:failed` is not cleared by resuming work on the unit.** It records that the last
verification of this unit came back red, and that stays true until a *new* verification says
otherwise — which happens when the fix merges (`test:pending`, swapping the failure out) and is
verified (Done, clearing the flag). Clearing it at `/work-start` would assert a result nobody
produced, and would do it at exactly the moment the unit is least verified.

Both `test:` labels must exist in the repo before they can be applied — `--add-label` fails on a
label the repo has never defined. Create them idempotently on first use
(`gh label create test:pending --force`), per the adapter.

## Procedure

### 1. Resolve and read

Read `.delivery/config.json` for `backend` and `branchPair.dev`. Read the item by id, and read its
**current** Status, open/closed state **and labels** — a snapshot from earlier in the session may be
stale. All three are load-bearing downstream and none is optional: the transition check needs where
the unit is now, the Done guard (step 4) needs to know whether a `test:failed` is standing, and the
flag swap (step 5) needs the *prior* `test:` label by name in order to remove it in the same write.
Reading Status alone is how a second `test:` label gets added instead of swapped.

### 2. Branch on the axis, then check the transition

**First decide which axis the argument is on**, because only one of them has a transition table:

- **A flag** (`test:pending`, `test:failed`) — there is no "from" and no legality table. Skip
  straight to the one precondition below, then to step 3. Checking a flag against the state table
  is the category error the two-axis split exists to prevent.
- **A state** (`Todo`, `In Progress`, `Done`, `reopen`) — check it against the table.

**The one precondition, and it is asymmetric: SETTING a flag needs an open unit; CLEARING one is
always allowed.** A flag is a condition on live work, so *setting* one on a Done/closed unit is
refused rather than written — `/work-sync` reports exactly that combination as drift (its shape 6),
and a skill that creates the drift its own sweep then flags is a loop with no fixed point:

```
/work-move: #<id> is Done; a test: flag is a condition on open work.
Reopen it first if the verification result is genuinely new.
```

*Clearing* carries no such check, in either direction. Removing a stale `test:` label from a closed
unit is the documented remedy for shape 6, so refusing it would leave that drift with no fix that
goes through this skill — and clearing can never assert something untrue, where setting can. The
asymmetry is deliberate: the guard belongs on the write that makes a claim.

| From | Legal to |
|---|---|
| Todo | In Progress · Done (a unit closed without work — needs a reason) |
| In Progress | Done · blocked · Todo (un-claim) |
| blocked | In Progress · Todo |
| Done | reopen |

(`blocked` appears in this table for the transitions that *clear* it; **setting** it is a flag write
like the others.)

Anything else is refused **with the legal moves named**, so the refusal is actionable:

```
/work-move: #<id> is Done; "in progress" is not a legal move from there.
Legal: reopen. Reopening asks for a reason and posts it to the item.
```

### 3. Require a reason where a reason is the point

**blocked**, **test:failed**, **reopen**, and **Done-without-a-merged-PR** each take a one-line
reason, asked for before anything is written and posted to the item as a plain comment.

For **test:failed** the reason is the whole point of the flag: `test:failed` alone says a
verification is red, which a human already suspects; what they cannot reconstruct is *which*
criterion failed and on which surface. A `test:failed` with no diagnostic is a flag that starts an
investigation instead of ending one.

Post the reason **before** the state write, not after. If the write fails, a unit carrying an
unexplained state change is worse than one carrying an explanation of a change that did not land.

Comment, never a history/revision field: the item's own change log is the backend's, and prose
written into it is not readable where humans look.

### 4. Guard Done — against an unmerged PR, and against a red verification

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

**And guard Done against a red verification.** If the unit carries `test:failed`, warn and require
confirmation before closing it:

```
/work-move: #<id> carries test:failed — <the diagnostic comment's first line>.
Done would close a unit whose verification came back red. Fix and re-verify,
or confirm you mean to close it with the failure standing.
```

This is the same guarantee as the merge guard and it exists for the same reason: the test gate is
the definition of done, so a Done that fronts a red verification is exactly the false green the
whole loop is built to make impossible. Confirming is legitimate — a criterion can be withdrawn, or
the failure re-scoped to a follow-up unit — but it must be a decision someone made, not a step
nobody noticed.

`test:pending` is **not** a barrier to Done: verifying a unit and then closing it is the ordinary
happy path, and the flag is simply cleared by the close (step 5).

The engine verifies the same property deterministically (`git rev-list --count origin/<dev>..<branch>`
must be zero) and marks a unit blocked rather than done when it is not. A hand-driven unit should
not get a weaker guarantee than an engine-driven one just because a human is at the keyboard.

### 5. Write every surface the move spans — then read it back

Write the surfaces from the table above. For **Done**, that is the close **and** the Status field
**and** clearing any `test:` label; do not stop after one. For a `test:` flag it is the label swap
(remove the prior one, add the new one, in a single call) plus the diagnostic comment on
`test:failed`.

Then **read the item back** and confirm the new values are actually there. Read the **labels** back
too, not just Status — the flag axis is where this skill now carries half its meaning, and a label
write that silently did not land (a label the repo never defined is the common cause) leaves the
board asserting a unit was verified when nothing recorded it. A write that reported
success and did not land is the failure mode this loop is built to make impossible, and a read-back
is two seconds.

If the read-back disagrees with what you wrote, **stop and report the mismatch**. Do not retry in a
loop and do not report success.

### 6. Report

```
#<id> — <title>
  status:  <before> → <after>          (unchanged on a flag write — say so rather than omitting it)
  flags:   <before> → <after>
  issue:   <open|closed>[ → <open|closed>]
  reason:  <the one-liner, when one was required>
  verified: read back, both surfaces agree
```

Print the `status:` line even when it did not change. A flag write that reports only the flag reads
like a state change to someone skimming, which is the exact confusion the two-axis split exists to
remove.

## Idempotent re-run

Moving a unit to the state it is already in is a **no-op, not a rewrite**. Skip the write entirely
rather than writing identical values: a no-op write still bumps the item's revision and changed-date,
which makes the board's history lie about when the transition happened — and those timestamps are
what a sprint review reads.

The same holds on the flag axis, and it is what makes flags safe to re-apply: setting a `test:` flag
the unit already carries writes nothing, and the remove-and-add swap converges on exactly one
`test:` label however many times it runs. So a re-run after a crash, or a second driver repeating a
step, cannot produce the two-labels state that would make the flag unreadable.

An illegal transition self-rejects, so repeated invocation is safe by construction: the table, not
a lock, is what keeps this skill from doing damage twice.
