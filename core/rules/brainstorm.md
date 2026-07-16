# Brainstorm rules

A brainstorm is persistent decision state. These rules keep an active brainstorm
impossible to miss across context breaks, and keep its file alive while it runs.

## Active-brainstorm check

There are TWO redundant signals for active brainstorms — honor both.

### Signal 1 — CLAUDE.md marker (auto-loaded, hard to miss)

Every active brainstorm pins itself into the scope's `CLAUDE.md` inside a
`<!-- brainstorm:active:start --> ... <!-- brainstorm:active:end -->` block at
start time, and removes itself at done time. Because `CLAUDE.md` auto-loads into
every Claude session as project instructions, the active brainstorm appears at
the top of context unconditionally — no scanning required.

If you see this block at the start of a session: **read the linked brainstorm
file before doing any work** that touches its topic.

### Signal 2 — Directory scan (source of truth)

The marker block is a redundancy mechanism — the directory is authoritative. At
the start of every conversation, also check for files with `status: active`
frontmatter in both scope layers:

- `<project>/.atl/brain-storms/` (current project)
- `~/.atl/brain-storms/` (global, cross-project)

**Parse the frontmatter, don't raw-grep the body.** Extract the frontmatter
block (between the leading `---` fences) and check its `status:` there — a
`grep -rl "status: active"` over the file body is wrong, because a
correctly-closed brainstorm legitimately quotes the literal string `status:
active` in its prose (historical notes, config tables, "when promoted to
`status: active`…"). Matching the body flags closed brainstorms as active.

If a `status: active` file exists but no marker block does (e.g., the brainstorm
predates this rule, or someone hand-edited files), restore the marker via the
same format the `start` skill writes — and tell the user the marker was
recovered.

### What to do when an active brainstorm is detected

1. Read the file and understand the full context.
2. Tell the user there is an active brainstorm and which one.
3. Update the file on every message cycle (see "Keeping it alive" below).

## Keeping it alive (when an active brainstorm exists)

On every message cycle:
- **When a message arrives:** read the brainstorm file (recall context).
- **After responding:** write new decisions, rejected ideas, and their reasons
  to the file.

### What belongs in the file

- NOT just "Decision X was made" → "X was proposed, rejected due to Y, a modified
  version of X was accepted due to Z".
- The user's exact statements at important points — the spirit lives there.
- Open questions and next steps.
- Chronological flow — which idea came after which.

### Purpose

The brainstorm file should be detailed and spirited enough that a Claude reading
it in a new context can continue as if it had been present in that conversation.

## Document chain

Every discussion and decision flows through a three-layer chain:

```
brain-storms/ (process) -> docs/ (outcome) -> CLAUDE.md (summary)
                     \
                       backlog.md (deferred superset) -> tasks.md (active-intent subset)
```

- No decision is made without a brainstorm.
- Brainstorm files are never deleted (historical record).
- If decisions change, a new brainstorm is opened and a "superseded by X" note is
  added to the old one.

Every layer of the chain lives in the scope's `.atl/` (project) or `~/.atl/`
(global) — the same scope axis the brainstorm itself lives in.

## Backlog + tasks discipline

Deferred decisions live in a two-tier surface under the scope's `.atl/`:

- **`.atl/backlog.md`** — the passive, **trigger-gated superset** of everything
  deferred, punted, or left uncertain. It prevents scope creep: we record what
  we're *not* doing now so the need isn't lost when it resurfaces. Most entries
  are gated on a trigger (a condition under which they come back) — the backlog
  is a scannable index of deferred work, not a to-do list.
- **`.atl/tasks.md`** — the **active-intent subset**: the short, prioritized list
  of what we actually mean to do next. An item moves **backlog → tasks** when we
  decide to pull it forward (a trigger fired, or we simply chose to prioritize it).

**Board-backend projects — the board holds the deferrals.** When a project runs a
delivery board backend (a `.delivery/config.json` with a `backend` field), the
**project board is the authoritative deferral surface**, not these two files —
one surface only, so the two can't drift. `/brainstorm done` then syncs deferrals
and active intent to the board (via the delivery-team's backend adapter) and
stops writing `.atl/backlog.md` / `tasks.md`; retiring any content those files
already hold (a superseding pointer + a one-time migration onto the board) is a
separate, per-project step, not something `/brainstorm done` does on its own. The
two-tier `.atl/` surface described here is the default for every project *without*
a board backend.

### When to add to the backlog

- A sub-topic deemed "premature, let's defer" during a brainstorm → write to the
  backlog **immediately**; don't say "later".
- A decision item explicitly marked "not included in this step".
- Anything noted as "we'll do this later" during development that needs a
  permanent record.

### Backlog format (lean, grouped by area)

- Group items under `## Area` headings by theme, not by date — a scannable index.
- One line per item: `- **Title** — one sentence. _Trigger:_ when it resurfaces. ↳ [source](...)`.
- The rich "why deferred / full context" lives in the linked brainstorm — don't
  duplicate it here. The backlog is the index; the brainstorm is the detail.

### Tasks format (active intent)

- `- [ ] **Title** — one sentence. ↳ [source](...)`, grouped under `## Now` / `## Next`.
- Keep it short and honest: if nothing is actively planned, `tasks.md` is nearly
  empty — that is the correct state, not a gap to fill. Don't manufacture tasks;
  unplanned deferred work belongs in `backlog.md`.

### Mandatory check during `brainstorm done`

When `/brainstorm done` runs, before the docs file is written (for a board-backend
project this check runs against the board instead of the two files — see the
board-backend note above):
1. **Backlog:** scan the brainstorm for every deferred / "later" / "not now" /
   left-uncertain item and ensure each has a `backlog.md` entry under the right
   area group. If any is missing, add it — or ask the user when it's ambiguous.
2. **Tasks:** if the brainstorm decided to actively pursue something *now* (rather
   than defer it), promote that intent into `tasks.md`; and remove from `tasks.md`
   anything this brainstorm actually shipped.

Closing a brainstorm without the backlog check is how deferred scope silently
disappears.

### Removing from either file

An item **leaves `backlog.md`** when it ships (the `docs/` + CLAUDE.md become the
source of truth) or when it's promoted into `tasks.md`. A task **leaves `tasks.md`**
when it ships — deleted, never left behind as a checked-off ✅. Don't leave
completed items lingering in either file.
