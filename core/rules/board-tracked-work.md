# Board-tracked work

When the project has a delivery **board backend** configured, **no shippable unit of work is done without a board item.** Before you start it, the work exists on the board; when new work surfaces mid-task, it gets its own item too. The board is the source of truth for what is happening — work that isn't on it is invisible, and invisible work can't be tracked, reviewed, or trusted.

## Why this rule exists

A board-backed project has a tracker precisely so its state is legible: what's planned, what's in flight, what shipped. But that only holds if work actually lands on the board. It is easy — especially in a fast interactive or autonomous session — to do a large amount of real work (fixes, releases, sweeps) directly, ship it, and never record any of it. The result is a board that shows no progress while the codebase moved substantially: the tracker lies, and no one can see what happened from it. This rule closes that gap. It extends the delivery-team's "the board is the single source of truth" discipline from the *autonomous* delivery loop (which already creates items) to **all** work in a board-backed project — interactive, ad-hoc, and maintenance work included.

## When this rule is active

Only when the project has a board backend — concretely, a **`.delivery/config.json`** exists at the project root (Azure or GitHub). `atl session-start` detects it and surfaces a reminder. In a project with no board backend, this rule is dormant and imposes nothing.

## What the agent must do

1. **A board item exists before a shippable unit starts.** A "shippable unit" is a coherent deliverable — the thing that becomes a commit/PR (a fix, a feature, a doc sweep, a release). Before starting one, ensure it has a board item; if it doesn't, create it first. Granularity is the shippable unit, **not** every sub-edit — the steps inside one deliverable share its one item.
2. **New work discovered mid-task gets its own item.** When work spawns more work — a bug found while fixing another, a follow-up a change requires — add a board item for it rather than folding untracked scope in silently. Deferred work follows the existing backlog/deferral path; work you actually do gets a tracked item.
3. **Reflect state on the board.** Move an item to *In Progress* when you start it and to *Done* when it ships. On GitHub Projects v2 the *start → In Progress* transition is manual (the platform only automates the Done end — see the delivery-team's board conventions), so set it explicitly.
4. **The board reflects reality.** At the end of a stretch of work, the board should show what actually happened. If it doesn't, the work was done wrong — surface the gap and backfill it.

## Failure modes to watch

- **Emergent-work blindness** — doing a pile of real, shipped work (a release, a sweep, a fix cluster) without ever creating items, so the board shows no movement while the code moved a lot. This is the exact failure the rule exists to prevent.
- **Scope-creep off-board** — a task grows new sub-work mid-flight and the extra scope ships untracked under the original item.
- **Ticket-spam over-correction** — creating a board item per micro-edit instead of per shippable unit, drowning the board so it's as useless as an empty one. Track deliverables, not keystrokes.

## When this rule does NOT apply

- **No board backend** — a project with no `.delivery/config.json` has no board to track against; the rule is dormant.
- **Non-shippable actions** — reading, exploring, answering a question, running a check. The rule governs *work that ships* (becomes a commit/PR), not every action in a session.

## How this rule interacts with other rules

- **The delivery-team ceremonies** — `/kickoff`, `/refine`, `/sprint-plan` already create board items for the autonomous loop. This rule extends the same board-is-truth discipline to work done *outside* those ceremonies (interactive edits, maintenance, ad-hoc `claude -p` runs) so nothing bypasses the board.
- **`brainstorm` (backlog/deferrals)** — deferred work goes to the backlog/board via `/brainstorm done`; this rule covers the complement — work you *actually do now* gets a tracked item, not just deferrals.
- **`branch-hygiene`** — its "every branch resolves to a merged-or-deleted PR" pairs with this rule's "every shippable unit resolves to a board item": together, the branch, the PR, and the board item are one traceable thread.
