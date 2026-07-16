---
knowledge-base-summary: "Never silently drop work. The PO reject path (#9 resolution): a PO-rejected item returns to the backlog with its iteration cleared and is naturally re-picked at the next /sprint-plan. Carryover handling for admitted-but-incomplete items (blocked / out-of-time / review-not-passed). Both funnel back into the DAG-and-capacity admission; the reason is always recorded, never lost."
---

# Reject & Carryover

The one discipline this topic enforces: **work is never silently dropped.** An item leaves a sprint
in exactly one of three ways — it completed, it was *rejected* by the PO, or it *carried over*
incomplete. The last two both return the item to the backlog with its reason recorded, so the next
`/sprint-plan` re-considers it through the normal DAG-and-capacity admission. Nothing falls through
a crack.

> **WHY "never silently drop" is the core rule.** A silently-dropped item is invisible lost work:
> the PO thinks it's still coming, the backlog no longer carries it, and no ceremony will ever pick
> it up again. Recording the reason and returning it to the backlog is the difference between a
> deferral (visible, re-scheduled) and a loss (invisible, forgotten). This mirrors the platform's
> branch-hygiene rule — resolve every unit of work, never abandon it.

## The PO reject path (#9 resolution)

At `/sprint-review` (or whenever the PO reviews delivered work), the PO may **reject** an item —
the work was done but doesn't meet acceptance. The resolution (#9) is deliberately simple:

1. The rejected item's **iteration is cleared** (removed from the sprint) via a work-item update
   (concept #6) — an idempotent field update, the same field-not-membership mechanism as
   assignment ([iteration-management.md](iteration-management.md)). With no iteration, it is
   back on the backlog.
2. Its state is set back to a not-Done, ready-to-rework category — resolve the concrete state at
   runtime (concept #7); never write a literal (the rework state might be `New`, `Active`,
   `Reopened`, or a custom value depending on the backend and process template).
3. The **rejection reason is recorded** as a comment (concept #3), so the next developer who picks
   it up knows *why* it came back — the PO's acceptance gap is the context that prevents
   re-delivering the same miss.
4. At the **next `/sprint-plan`**, the item is simply a backlog candidate again: my blueprint reads
   it with everything else, places it in the DAG, and admits it against capacity by priority —
   **no special "rejected" queue.** A rejected item re-enters the exact same admission logic as any
   other ready item. This is the whole of #9: rejection is "clear the iteration, record why, let
   the standard plan re-pick it."

> **WHY no separate rejected-item pipeline.** A parallel "reprocessing" queue would be a second
> scheduling path to keep consistent with the DAG and capacity math — pure complexity for no gain.
> Returning the item to the backlog means one admission algorithm handles new, carried-over, and
> rejected work identically. Simpler, and impossible to drift.

## Carryover — admitted but not completed

An item I admitted this sprint may not reach the Completed category by sprint end. Three common
reasons, all handled the same way — return to backlog, record the reason:

- **Blocked** — a dependency (in- or out-of-sprint) wasn't satisfied in time. Its dependency edge
  is still in the DAG; next plan, it's ready only once its predecessor is Done.
- **Out of time** — the sprint's time-box closed before the item was worked (it was admitted but
  refill/capacity never reached it, or it was mid-flight). It's a plain backlog candidate next
  sprint.
- **Review not passed** — the micro-loop's `green = (tests) ∧ (review passed)` never went green
  (the `tech-lead`'s review found blocking issues). The item stays open; next sprint it's a
  candidate again, with the review thread as context.

> **Blocked by a crash/stall, not just a dependency.** A worker that *cleanly* blocks marks the board
> itself; but one that crashes or silently stalls marks nothing, and the dispatch engine (which has
> no backend surface) instead writes a durable `BlockedReport` to `.delivery/blocked/<id>.json`. That
> report is drained at `/sprint-review` (its step 2): the ceremony reflects the `blocked` tag/label +
> diagnostic comment onto the work-item and clears the report, so the unit lands in `## Carryover`
> like any other — never silently dropped.

Carryover handling:

1. **Clear or leave the iteration** — for a clean re-plan, I clear the incomplete item's
   iteration at sprint close so the next `/sprint-plan` picks it up as a fresh backlog
   candidate (rather than leaving it pinned to a closed sprint). Idempotent field update, as
   always.
2. **Record it on the review page** — every carried-over item appears in the `## Carryover` section
   of `Sprints/Sprint-<n>-Review` with its reason ([sprint-review-report.md](sprint-review-report.md)).
   This is the visible audit trail: the PO sees exactly what didn't finish and why.
3. **It does not count toward the sprint's actual velocity** — only items that reached the
   Completed category contribute points ([capacity-and-velocity.md](capacity-and-velocity.md)).
   Carryover deflates the sprint it *didn't* complete in and inflates the sprint it *does* complete
   in — which is the honest signal for the velocity mean.

## The unifying shape

Reject and carryover are two entrances to the same exit: **an unfinished item, with a recorded
reason, back on the backlog, re-admitted by the standard DAG-and-capacity plan.** I never invent a
side-channel and I never let an item vanish. The reason travels with the item (a comment and/or the
review page); the item travels back to the backlog (iteration cleared); the next plan treats it
as an ordinary candidate.

## Worked example (generic)

- Sprint N admits `X Y Z`. `X` completes. `Y` is PO-rejected at review (acceptance gap). `Z` is
  blocked on an out-of-sprint predecessor.
- `X` → counts toward Sprint N velocity; appears under `## Completed`.
- `Y` → iteration cleared, state set to the resolved rework category, rejection reason
  commented; appears under `## Carryover` (reason: review/acceptance); re-enters the backlog.
- `Z` → iteration cleared; appears under `## Carryover` (reason: blocked on `<predecessor>`);
  re-enters the backlog, becomes ready once its predecessor is Done.
- Sprint N+1's `/sprint-plan` reads `Y` and `Z` as ordinary candidates — `Y` ready immediately,
  `Z` ready only if its predecessor completed. Same admission logic, no special cases. Nothing was
  lost.
