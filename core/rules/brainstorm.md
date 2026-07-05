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
                       backlog.md (deferred items)
```

- No decision is made without a brainstorm.
- Brainstorm files are never deleted (historical record).
- If decisions change, a new brainstorm is opened and a "superseded by X" note is
  added to the old one.

Both layers of the chain live in the scope's `.atl/` (project) or `~/.atl/`
(global) — the same scope axis the brainstorm itself lives in.

## Backlog discipline

Every item marked "not doing now, later" during a brainstorm must be reflected in
the scope's **`.atl/backlog.md`**. This prevents scope creep — we record what
we're not doing now so we remember when the need resurfaces.

### When to add to the backlog

- A sub-topic deemed "premature, let's defer" during a brainstorm → write to the
  backlog **immediately**; don't say "later".
- A decision item explicitly marked "not included in this step".
- Anything noted as "we'll do this later" during development that needs a
  permanent record.

### Format

- **Prepend** (newest on top) — added to the beginning of `.atl/backlog.md`;
  older items stay below.
- Per item: date + category heading + context link + detailed description + a
  "when does this come up" note + related resources.
- Follow the template at the top of the file.

### Mandatory check during `brainstorm done`

When `/brainstorm done` runs, scan every "deferred" note in the brainstorm file
and **ensure each has a corresponding backlog entry**. If any are missing, ask
the user and add them. This is a checklist step before closing the brainstorm.

### Removing from the backlog

When an item is implemented, it is **deleted** from the backlog (not marked done
and left). It's now part of the active infrastructure, so the relevant
docs/CLAUDE.md is updated instead.
