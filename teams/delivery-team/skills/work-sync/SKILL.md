---
name: work-sync
description: /work-sync [dry-run] — sweep the board for drift between what the code says and what the item says: a merged PR under an item still in progress, a Done item whose PR never merged, an orphan delivery branch, an item claimed but never started. Reports every finding with the evidence that produced it, then offers the fixes as a batch. Dry-run by default in spirit — nothing is written without an explicit yes.
---

# /work-sync — the board tells the truth, or it tells you where it doesn't

Drift is not an accident to be prevented; it is the steady state of any board a human drives by
hand. The card that should have moved when the PR merged does not move itself, and the person who
would have moved it was already three units further on.

So drift is handled at three cadences, and this is the widest one. Per unit, `/work-move` guards
each transition. Per session, the drive loop surfaces the unit on the current branch. Per sweep,
this finds everything both of those missed.

Reads its contracts from [`config-and-methodology.md`](../../knowledge/config-and-methodology.md),
[the backend interface](../../knowledge/backend-interface.md), and the active
`backends/<backend>/adapter.md`. **Never invent a tool name.**

## Backend support

**GitHub is bound; Azure halts** — same reason as the rest of the drive loop.

## What counts as drift

| # | Shape | Evidence | Fix |
|---|---|---|---|
| 1 | **Merged, unverified, unflagged** — a PR merged into the integration branch, its item still In Progress and carrying no `test:` flag | the merge commit is an ancestor of `<dev>` | `/work-move <id> test:pending` — **not** straight to Done: a merge is not a verdict, and this sweep has no way to know whether anyone verified it |
| 2 | **Done without a merge** — an item is Done but its PR never merged | `git rev-list --count origin/<dev>..<branch>` is non-zero | reopen, or confirm it was closed deliberately |
| 3 | **Orphan branch** — a `delivery/<slug>/<id>` branch whose item is closed, or which has no PR at all | branch exists; item state / PR absence | see the asymmetry below |
| 4 | **Claimed, never started** — In Progress with no branch and no commits | no matching branch anywhere | un-claim, or leave it and say so |
| 5 | **Two carriers** — a unit wearing more than one `sprint:` label | the labels | stop; a human decides which sprint it is in |
| 6 | **Stale verification flag** — a Done/closed unit still carrying `test:pending` or `test:failed` | the item is closed and the label is present | clear the flag; Done means the condition is over |
| 7 | **Two verification flags** — a unit wearing both `test:pending` and `test:failed` | the labels | stop; they are mutually exclusive and a human decides which is true |

Shape 6 matters more than its size suggests: a stale `test:pending` on a closed unit makes
`/work-list verifying` report work that nobody actually owes, and a queue that lists phantom items
is one the driver stops trusting.

Shapes that look like drift and are **not**, so do not offer to fix them: an open PR that has not
merged yet (that is a PR under review); an item marked `blocked` (a deliberate annotation, and
pulling it back into flight overrides someone's judgement); and an item carrying `test:pending` or
`test:failed` — those are **correctly** annotated units waiting on a human's verdict or fix, which
is the flag doing its job. In particular a merged unit sitting at `test:pending` is not shape 1 all
over again: shape 1 is the *unflagged* case, and once the flag is on, the board is telling the truth
and the sweep has nothing to add.

## Procedure

### 1. Scope, and default to narrow

`mine` by default — the units you can actually speak for. `all` sweeps the whole board, and on a
shared board it will surface other people's units; those get **individual** confirmation, never a
batch yes, because moving someone else's card is a claim about work you did not do.

### 2. Gather the evidence before forming any verdict

For each candidate unit, resolve its branch by the pure `delivery/<slug>/<id>` grammar, then read
git and the board:

- has the branch merged into `config.branchPair.dev`? (`git rev-list --count origin/<dev>..<branch>`
  — zero means merged)
- does a PR exist for it, and in what state?
- what does the item itself say — **its Status and its labels**? The `test:` flags decide between
  shape 1 and no finding at all, so reading Status alone re-reports every correctly-annotated unit
  as drift.

**Read git for the merge, not the PR's own status.** A PR reporting merged and a branch that is
actually an ancestor of the integration branch are two different claims, and only the second one
is the property that matters.

### 3. Present the findings with their evidence

One line per finding, and always the evidence that produced it:

```
#<id>  <title>
  drift:     merged, unverified, carrying no test: flag
  evidence:  PR #<n> merged into <dev> 2 days ago; branch is an ancestor of origin/<dev>;
             labels: area:api sprint:3 (no test: flag)
  fix:       add test:pending — Status stays In Progress
```

A finding without its evidence is an assertion, and a sweep that asserts is one the driver learns
to ignore.

### 4. Ask once, act as a batch — or not at all

Three options: apply all, walk them one at a time, or report only. **Report-only is a legitimate
outcome**, not a failure of the sweep — sometimes the answer is "yes, I know, later".

Every write goes through `/work-move` rather than being re-implemented here. That is the point of
having one state writer: its transition table, its reason prompts, and its read-back verification
apply to a batch fix exactly as they do to a hand-typed one.

### 5. The orphan-branch asymmetry — never resolved by deleting

A branch ends in exactly one of two states: merged via a PR, or deleted. But those two cases are
**not** symmetric, and this is the one place a sweep can destroy work:

- **Merged** ⇒ deleting is free; the work is on the integration branch.
- **Unmerged, with real commits** ⇒ **never delete, and never silently.** Surface it, and offer to
  open a PR for it. Deleting unmerged work is the single irreversible thing in this whole loop.

The engine holds the same line — it *quarantines* a leftover worktree by moving it aside rather
than removing it. A hand-driven sweep must not be more destructive than the machine.

### 6. Report

```
swept <n> units (<scope>)
  drift found:  <n>   [by shape]
  fixed:        <n>
  left alone:   <n>   [with why]
  needs a human: <n>  [unmerged orphan branches, double carriers, double test: flags]
```

Close by naming the last shape explicitly. Those are the ones that cannot be fixed mechanically,
and burying them in a count is how they stay unfixed.

## Idempotent re-run

Naturally convergent: a second sweep finds the units it already fixed out of the drift set, so it
reports nothing to do. That is the correct outcome and should read as one — "no drift" is the
result this skill exists to produce, not an empty run.

`dry-run` is the safe probe and costs nothing but the queries. Reach for it first on `all` scope,
where the finding count is unknown and the confirmation is broadest.
