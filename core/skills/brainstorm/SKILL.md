---
name: brainstorm
description: "Start and complete brainstorming sessions. start = begin a new brainstorm, done = complete the active brainstorm and propagate to the document chain. 2 scopes: project (default), --global."
argument-hint: "<start|done> [--global] [initial message]"
---

# /brainstorm — think a decision through before writing code

A brainstorm is the canonical place to reason through a non-trivial decision
before any code is written. The resulting file is persistent state: it becomes
the historical record + handoff so any later session can pick up the topic as if
it had been in the room.

## Parameter parsing

The first word selects the mode: `start` or `done`. The remaining text (if any)
is the user's initial message.

## Two scopes (the v2 scope axis)

ATL has one scope axis — **project** (the default) vs **global** (`~/.atl`),
isomorphic with Claude Code's own layering (project shadows global). A brainstorm
lives at the layer that matches *who* will care about the decision.

| Flag | Target directory | When |
|---|---|---|
| *(none)* | `<project>/.atl/brain-storms/` | Project-specific topics (default) |
| `--global` | `~/.atl/brain-storms/` | Cross-project, personal topics |

---

## `start` mode

When the user says `/brainstorm start ...`:

### 1. Understand the topic
Extract the topic from the user's message. The user does NOT supply a title —
you infer it and derive an appropriate `kebab-case` filename.

### 2. Determine scope
- `--global` present → `base_dir = ~/.atl/`
- otherwise → `base_dir = <project>/.atl/` (project root)

### 3. Create the directory (if missing)
Create `{base_dir}/brain-storms/` if it doesn't exist.

### 4. Create the file
Create `{base_dir}/brain-storms/{name}.md`:

```markdown
---
status: active
scope: {project|global}
date: {today's date}
participants: <user-name>, Claude
---

# {Topic Title}

## Context

{Context and motivation understood from the user's initial message}

## Discussion

### {Date} — Start

{Summary of the initial message and any first ideas}

## Open Items

- {Questions or points to discuss extracted from the initial message}
```

### 5. Pin to CLAUDE.md (active-brainstorm marker)

Inject a marker block into the scope's `CLAUDE.md` so future Claude sessions
cannot miss the active brainstorm. The block auto-loads with project context,
making the active brainstorm impossible to overlook even when the rule's "scan
the directory" step is skipped.

**Target file by scope:**
- **project** → `CLAUDE.md` at the project root
- **global** → `~/.claude/CLAUDE.md`

**Marker block format** (HTML comments are delimiters — used to find/update/remove the block):

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[{brainstorm-name}]({relative-path-to-brainstorm-file})** ({scope}, {date}) — {one-line topic summary}
<!-- brainstorm:active:end -->
```

**Insertion rules:**
1. **If the marker block does NOT exist:** insert it near the top of the file,
   right after the H1 + opening description (before the first H2). For project
   `CLAUDE.md` this is after the intro paragraph; for `~/.claude/CLAUDE.md` it
   goes right after the title.
2. **If the marker block EXISTS:** add a new bullet to the list (preserve
   existing bullets — multiple active brainstorms coexist). Do not duplicate a
   bullet for the same brainstorm.
3. **Relative path:** path relative to the file you're editing (e.g.,
   `.atl/brain-storms/foo.md` from project `CLAUDE.md`; `brain-storms/foo.md`
   from `~/.claude/CLAUDE.md`).
4. **One-line summary:** distill from the brainstorm's H1 title or context —
   under ~80 chars.

### 6. Respond
Tell the user the brainstorm has started — filename, scope, and that it was
pinned to the appropriate `CLAUDE.md`. Then dive into the topic.

### 7. On subsequent messages
Update the brainstorm file on every message cycle:
- Add new ideas, decisions, rejected alternatives and their reasons
- Preserve the user's important statements verbatim (in quotes)
- Maintain chronological order — add new subheadings under Discussion
- Update Open Items (resolved ones removed, new ones added)

**IMPORTANT:** The file must be detailed enough that a Claude reading it in a new
context can continue as if it had been present in the conversation.

---

## `done` mode

When the user says `/brainstorm done`:

### 1. Find the active brainstorm
Search **both locations**:
- `<project>/.atl/brain-storms/` (project)
- `~/.atl/brain-storms/` (global)

Find files with `status: active`. If multiple, list them (showing each one's
scope) and ask which to complete.

### 2. Complete the brainstorm file
- `status: active` → `status: completed`
- Add final notes at the end of the Discussion section
- Update Open Items (unresolved ones remain)
- Add a "Final Decisions" section — a summary of every definitive decision from
  the whole discussion

### 3. Create / update the docs file
Determine the docs location from the brainstorm's scope:
- **Project brainstorm** → write under `<project>/.atl/docs/`
- **Global brainstorm** → write under `~/.atl/docs/`

Reference the brainstorm at the top of the file.

### 4. Update CLAUDE.md
- **Project brainstorm** → update `CLAUDE.md` at the project root
- **Global brainstorm** → update `~/.claude/CLAUDE.md`

Up to three updates happen:
1. **Add the completed-brainstorm summary** to the appropriate section (existing
   behavior).
2. **Remove the active-brainstorm marker** for THIS brainstorm:
   - Find the `<!-- brainstorm:active:start -->` … `<!-- brainstorm:active:end -->` block.
   - Remove the bullet whose link points to this brainstorm's file.
   - **If the bullet list becomes empty after removal**, remove the entire marker
     block (heading + intro line included) so no stale "Active brainstorms"
     section lingers.
   - **If other bullets remain**, keep the block intact — other brainstorms are
     still active.
3. **Pin a pending-implementation reminder — only if the decision leaves unshipped
   work.** If this brainstorm decided on a change that hasn't been implemented yet,
   add a bullet to the `<!-- pending-implementation:start -->` …
   `<!-- pending-implementation:end -->` block so the next session sees the queue.
   **Omit this entirely for a pure-decision brainstorm** (a rejection, or a doc-only
   choice with nothing to build).
   - Block format (insert near the top if absent — after any `<!-- brainstorm:active -->`
     block, above the `<!-- wiki:index -->` block):

     ```markdown
     <!-- pending-implementation:start -->
     ## 🚧 Pending implementation

     Brainstorms have decided these but the work hasn't shipped yet:

     - **[{name}]({relative-path-to-docs-or-brainstorm})** — {decided X; impl pending}
     <!-- pending-implementation:end -->
     ```
   - Add a bullet (preserve existing; don't duplicate). The bullet is removed when
     the implementation ships — by hand, or by the PR that ships the change.

### 5. Respond
Tell the user the brainstorm is complete and list the created/updated files.

---

## Important rules

1. **Multiple active brainstorms can exist.** Each lives independently in its own
   file. They can be active simultaneously across both scopes.
2. **Resilience to context breaks.** The brainstorm file is persistent state. In a
   new context, the rule detects active brainstorms and continues by reading it.
3. **Filename is not requested from the user.** You infer it and assign an
   appropriate kebab-case name.
4. **Brainstorm files are never deleted.** They remain as historical records.
5. **Each brainstorm focuses on a single topic.** Different topics → different files.
6. **Active-brainstorm search covers both locations.** In `done` mode, project +
   global directories are scanned.
7. **Scope is in frontmatter.** `scope: project|global` — determines the correct
   target in `done` mode.

## Accumulated Learnings

(Auto-rebuilt by /drain from this skill's knowledge files. Do not edit by hand.
Currently empty — populates as the skill is used and edge-case learnings
accumulate.)
