---
knowledge-base-summary: "Never silently drop work, and never abandon started work for something new. An unfinished item leaving a sprint (PO-rejected OR carried-over incomplete) is carried to the next sprint as TOP PRIORITY, admitted FIRST ahead of all new work — unfinished committed work outranks new work. Blocked-split: out-of-time / review-not-passed / rejected are workable → top-priority guaranteed; a blocked unit is carried + surfaced but does NOT consume the next sprint's workable capacity or a top slot until it unblocks (so a blocked item can't freeze the sprint). The reason always travels with the item; nothing is lost or bumped by newer work."
---

# Reject & Carryover

The one discipline this topic enforces: **work is never silently dropped, and unfinished work is
always the next sprint's top priority.** An item leaves a sprint in exactly one of three ways — it
completed, it was *rejected* by the PO, or it *carried over* incomplete. The last two carry the item
to the next sprint **as top priority**, its reason recorded — **unfinished committed work outranks
any new work**, always. Nothing falls through a crack, and nothing started is abandoned in favour of
something newer.

> **WHY "never silently drop" is the core rule.** A silently-dropped item is invisible lost work:
> the PO thinks it's still coming, the backlog no longer carries it, and no ceremony will ever pick
> it up again. Recording the reason and carrying it forward is the difference between a deferral
> (visible, re-scheduled) and a loss (invisible, forgotten). This mirrors the platform's
> branch-hygiene rule — resolve every unit of work, never abandon it.

> **WHY unfinished work is top priority.** A half-finished item is committed work-in-progress: the
> team already paid the context-switch + startup cost, and a `developer`/`tester`/`tech-lead` already
> holds the thread. Starting a *new* item while a committed one sits unfinished multiplies WIP and
> defers value already invested. So the next sprint **finishes what it started before it starts
> anything new** — new work, however urgent, fills only the capacity that remains after carryover.
> The one exception is a *blocked* unit, which can't be worked yet (the blocked-split below).

## The PO reject path (#9 resolution)

At `/sprint-review` (or whenever the PO reviews delivered work), the PO may **reject** an item — the
work was done but doesn't meet acceptance. The resolution (#9):

1. The rejected item is **carried to the next sprint** and marked **top priority** — not cleared to
   the backlog to compete with new work. An idempotent field update (concept #6).
2. Its state is set back to a not-Done, ready-to-rework category — resolve the concrete state at
   runtime (concept #7); never write a literal (it might be `New`, `Active`, `Reopened`, or a custom
   value depending on the backend and process template).
3. The **rejection reason is recorded** as a comment (concept #3), so the next developer who picks it
   up knows *why* it came back — the PO's acceptance gap prevents re-delivering the same miss.
4. At the **next `/sprint-plan`** the item is admitted **FIRST, ahead of all new backlog work** — a
   rejected item is unfinished committed work (the acceptance gap isn't closed). There is still no
   special "rejected" *pipeline*: it re-enters the one DAG-and-capacity admission algorithm, but at
   the front of the priority order — it does not compete on stackRank with new candidates.

> **WHY no separate rejected-item pipeline.** A parallel "reprocessing" queue would be a second
> scheduling path to keep consistent with the DAG and capacity math — pure complexity for no gain.
> One admission algorithm handles new, carried-over, and rejected work; carryover simply enters it at
> top priority. Simpler, and impossible to drift.

## Carryover — admitted but not completed

An item I admitted this sprint may not reach the Completed category by sprint end. The three reasons
**split by whether the item is workable** next sprint:

- **Out of time** (workable) — the time-box closed before the item was worked or finished (refill
  never reached it, or it was mid-flight). Next sprint it is **top-priority, guaranteed** — among the
  first work admitted.
- **Review not passed** (workable) — the micro-loop's `green = (tests) ∧ (review passed)` never went
  green (the `tech-lead`'s review found blocking issues). The item stays open, carries with the
  review thread as context, and is **top-priority, guaranteed** next sprint.
- **Blocked** (NOT workable) — a dependency (in- or out-of-sprint) wasn't satisfied in time. A blocked
  unit **carries + is surfaced** in `## Carryover` (never dropped), but it does **NOT** consume the
  next sprint's workable capacity or occupy a top-priority slot — it *can't* be worked until its
  predecessor clears, so privileging it would only freeze the sprint on un-workable work. Its
  dependency edge stays in the DAG; the moment it unblocks (predecessor Done) it becomes
  workable-carryover and takes top priority like the others.

> **WHY the blocked-split.** "Unfinished work is top priority" is a rule about *workable* work — you
> can only make progress on what you can actually do. A naive "always top priority + consumes
> capacity" applied to a blocked unit would hand the sprint's top slot to something no worker can
> touch, and the block could persist for sprints — freezing the team on un-runnable work. Carrying +
> surfacing the blocked unit (so it's never lost) while excluding it from the *workable* set until it
> unblocks keeps both invariants: nothing is dropped, and the sprint always fills with work that can
> actually move.

> **Blocked by a crash/stall, not just a dependency.** A worker that *cleanly* blocks marks the board
> itself; but one that crashes or silently stalls marks nothing, and the dispatch engine (which has
> no backend surface) instead writes a durable `BlockedReport` to `.delivery/blocked/<id>.json`. That
> report is drained at `/sprint-review` (its step 2): the ceremony reflects the `blocked` tag/label +
> diagnostic comment onto the work-item and clears the report, so the unit lands in `## Carryover`
> like any other — never silently dropped.

Carryover handling:

1. **Tag it `carryover` and carry it forward** — at sprint close, add a `carryover` tag/label
   (concept #4) to the incomplete item and record its reason; a *blocked* one additionally carries
   the `blocked` tag. That `carryover` tag is the **durable signal the next `/sprint-plan` reads to
   admit these FIRST**, at top priority, ahead of new candidates: a *workable* carryover (out-of-time
   / review-not-passed / rejected) is admitted and moved to the new sprint's iteration (its
   `carryover` tag cleared as it re-enters); a *blocked* one stays `carryover`-tagged + surfaced but
   is **not** admitted to the workable set until its predecessor clears. Never left silently pinned to
   a closed sprint with no signal, never dropped. Idempotent tag/field update (a re-run re-tags /
   re-admits to the same state).
2. **Record it on the review page** — every carried-over item appears in the `## Carryover` section
   of `Sprints/Sprint-<n>-Review` with its reason **and its workable/blocked status**
   ([sprint-review-report.md](sprint-review-report.md)). This is the visible audit trail: the PO sees
   exactly what didn't finish, why, and what is carrying vs waiting-on-a-block.
3. **It does not count toward the sprint's actual velocity** — only items that reached the Completed
   category contribute points ([capacity-and-velocity.md](capacity-and-velocity.md)). Carryover
   deflates the sprint it *didn't* complete in and inflates the sprint it *does* complete in — the
   honest signal for the velocity mean.

## The unifying shape

Reject and carryover are two entrances to the same exit: **an unfinished item, reason recorded,
carried to the next sprint as top priority (guaranteed, ahead of new work) — or, if blocked, carried
+ surfaced but not yet workable.** New work fills only the capacity that remains after the workable
carryover is admitted. I never invent a side-channel, I never let an item vanish, and I never let
started work lose its place to something newer.

## Worked example (generic)

- Sprint N admits `X Y Z W`. `X` completes. `Y` is PO-rejected at review (acceptance gap). `Z` ran
  out of time. `W` is blocked on an out-of-sprint predecessor.
- `X` → counts toward Sprint N velocity; appears under `## Completed`.
- `Y` → carried to N+1 as **top priority** (reason: review/acceptance), state set to the resolved
  rework category, rejection reason commented; appears under `## Carryover` (workable).
- `Z` → carried to N+1 as **top priority** (reason: out-of-time); appears under `## Carryover`
  (workable).
- `W` → carried + surfaced under `## Carryover` (reason: blocked on `<predecessor>`, **waiting**) but
  **not** admitted to N+1's workable set; becomes top-priority workable-carryover the moment its
  predecessor is Done.
- Sprint N+1's `/sprint-plan` admits `Y` and `Z` **FIRST** (top priority, before any new candidate),
  then fills the remaining capacity with new work by stackRank; `W` waits for its unblock. Same
  admission algorithm, carryover at the front. Nothing was lost; nothing started was abandoned for
  something newer.
