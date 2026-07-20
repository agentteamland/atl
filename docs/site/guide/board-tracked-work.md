# Board-tracked work

If your project has a delivery **board backend** — an Azure or GitHub project wired up via the delivery-team — ATL keeps the board honest: **no shippable unit of work happens without a board item.** This page is the user side of that discipline.

## What's happening under the hood

The [`board-tracked-work` rule](https://github.com/agentteamland/atl/blob/main/core/rules/board-tracked-work.md) auto-loads in every session (and into the autonomous `claude -p` workers the delivery-team spawns). It is **conditional**: it activates only when the project has a `.delivery/config.json` at its root. `atl session-start` detects that file and prints a one-line reminder — *"this project is board-backed (github) — record every shippable unit on the board…"* — so the agent knows the rule is live here. In a project with no board backend, the rule is dormant and adds nothing.

It exists to close a specific gap. A board-backed project has a tracker so its state is legible — what's planned, in flight, and shipped. But that only holds if work actually lands on the board. In a fast session it's easy to do a pile of real work — fixes, releases, sweeps — ship it, and record none of it, leaving a board that shows no progress while the code moved a lot. The tracker then lies. This rule makes the board reflect reality.

## What it means in practice

**A board item before a shippable unit starts.** A "shippable unit" is a coherent deliverable — the thing that becomes a commit/PR (a fix, a feature, a doc sweep, a release). It gets one board item, created before the work starts if it doesn't exist yet. The granularity is the deliverable, **not** every sub-edit — the steps inside one unit share its item.

**New work mid-task gets its own item.** When a task spawns more work — a bug found while fixing another, a required follow-up — that gets a board item too, rather than shipping untracked scope under the original.

**The board reflects state.** Items move to *In Progress* on start and *Done* on ship. (On GitHub Projects v2 the start → *In Progress* step is manual — the platform only automates the Done end.)


**On resume, check the board first.** When a session resumes (a "continue" after a break), it first looks at the board for anything already *In Progress* and closes the loop on work left mid-flight — a session that ends mid-task leaves its item In Progress, and the next one must not lose it. This is the read side of the same discipline (`atl session-start` prints the reminder). The board tells you *what is in flight*; it does not by itself dictate *what is next* — a backlog can be mostly deferred design work, so for the next thing to do you still consult the project's own resume convention, not the top card. Board-aware, not board-driven.
## Why it's a core rule

It is what makes a board-backed project's tracker trustworthy — for a human reviewer and for ATL's own **autonomous delivery**, where the board is the single source of truth the whole loop reads. It extends the delivery-team's "the board is truth" discipline from the formal ceremonies (`/kickoff`, `/refine`, `/sprint-plan`, which already create items) to **all** work in the project — interactive edits, maintenance, ad-hoc runs included — so nothing bypasses the board. The one guard against over-correction: track **deliverables, not keystrokes** — a board item per micro-edit drowns the board as badly as an empty one.
