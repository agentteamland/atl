# Claude Code conventions

The marker-block conventions `atl` and its sibling skills use inside `CLAUDE.md` to coordinate state across sessions. These are HTML-comment-delimited blocks that are auto-loaded by Claude Code's project-instruction mechanism — invisible in rendered Markdown, impossible to miss when Claude reads the file.

You can use the same pattern in your own `CLAUDE.md` files. The blocks are just convention — there's no special parser. The comment delimiters make them safe to find/update/remove programmatically.

## The three blocks

| Block | Written by | Purpose |
|---|---|---|
| `<!-- wiki:index -->` | [`/save-learnings`](/skills/drain) | Auto-rebuilt table of contents for `.atl/wiki/` pages. Loads with project context, gives Claude the knowledge map at zero cost. |
| `<!-- brainstorm:active -->` | [`/brainstorm start`](/skills/brainstorm) and [`/brainstorm done`](/skills/brainstorm) | Pins active brainstorm topics into project context so the next session cannot miss them. |
| `<!-- pending-implementation -->` | Brainstorm `done` flow | Reminds the next session that a brainstorm decided X but the implementation hasn't shipped yet. |

All three use the same `<!-- block:start --> ... <!-- block:end -->` delimiter pattern. None of them have a parser in the strict sense — they're convention, not syntax. But the convention is consistent enough to find/update/remove with simple `sed`/regex when needed.

> **Note — why the example block contents below are in English (even on the Turkish mirror of this page):** The three templates (`wiki:index`, `brainstorm:active`, `pending-implementation`) are produced verbatim by `/save-learnings` and `/brainstorm`. Per the `feedback_speak_turkish` rule, these skills always write English (committed files must be English regardless of project language). So in a Turkish project, the `CLAUDE.md` blocks still appear in English — the examples reflect the actual output.

## `<!-- wiki:index -->` — knowledge map

Auto-rebuilt by `/save-learnings` after every change to `.atl/wiki/`. Lives near the top of `CLAUDE.md`, after the H1 + intro:

```markdown
<!-- wiki:index:start -->
## 📚 Knowledge map

Knowledge lives in `.atl/wiki/` (current truth, topic-organized) and `.atl/journal/` (historical record, date-based). Before working on a topic, scan this list — if a page looks relevant, read it before deciding.

**Wiki topics:**
- [docs-audit-false-positive-rate](.atl/wiki/docs-audit-false-positive-rate.md) — ~40% of multi-agent docs-drift audit reports include hallucinated findings
- [pr-merge-discipline](.atl/wiki/pr-merge-discipline.md) — never `gh pr merge` from Claude; surface URL and stop
- [save-learnings-timing](.atl/wiki/save-learnings-timing.md) — run before PR creation as feature branch's last commit
- ...

**Discipline:** Before working on a topic, scan this list. If a topic looks relevant, read the page.
<!-- wiki:index:end -->
```

Each entry is one line: `- [topic](.atl/wiki/topic.md) — one-line summary` (sorted alphabetically by filename). The summary comes from the first non-frontmatter, non-heading line of each wiki page.

**Why it's a marker block, not just a normal section:** the block is rebuilt programmatically. Hand-edits inside the markers are overwritten on the next `/save-learnings` run. To add a topic: don't edit `CLAUDE.md` directly — create the wiki page with the topic title, then `/save-learnings` rebuilds the index.

## `<!-- brainstorm:active -->` — active topics pin

Written by `/brainstorm start`, removed by `/brainstorm done`. Lives near the top of the scope's `CLAUDE.md` (project) or `~/.claude/CLAUDE.md` (global) or team `README.md` (team-scope):

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[docs-sync-automation](.atl/brain-storms/docs-sync-automation.md)** (project, 2026-05-03) — closing the README + docs-site drift gap; save-learnings extension vs. /docs-sync skill vs. doc-agent
<!-- brainstorm:active:end -->
```

Multiple active brainstorms coexist as bullets in the same block. The `done` flow removes only the bullet for the brainstorm being completed; if the bullet list becomes empty, the entire block is removed (no stale "Active brainstorms" heading lingers).

**Why this convention exists:** the brainstorm rule's "scan `.atl/brain-storms/` for `status: active` files" step depended on Claude remembering to do it on every session start. Pinning the active brainstorm into `CLAUDE.md` makes it auto-load — impossible to miss. The directory scan is now a redundancy mechanism, not the primary signal.

Shipped in `brainstorm@1.1.0`.

## `<!-- pending-implementation -->` — decided-but-unshipped reminder

Written when a brainstorm's `done` flow decides on a change that hasn't been implemented yet. Reminds the next session that the decision exists and the work is queued:

```markdown
<!-- pending-implementation:start -->
## 🚧 Pending implementation

Brainstorms have decided these but the work hasn't shipped yet:

- **[install-mechanism-redesign](.atl/docs/install-mechanism-redesign.md)** — symlink → project-local copy migration. Atomic write helper + auto-refresh logic queued for `atl v1.0.0`.
<!-- pending-implementation:end -->
```

Removed when the implementation lands (typically by the PR that ships the change).

**Why this matters:** without the reminder, completed brainstorms can sit in `.atl/docs/` for weeks while the implementation gets queued behind other work. The pin keeps the queue visible.

## Where the blocks live

In a project's root `CLAUDE.md`:

```markdown
# Project Name

Short intro paragraph.

<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms
...
<!-- brainstorm:active:end -->

<!-- pending-implementation:start -->
## 🚧 Pending implementation
...
<!-- pending-implementation:end -->

<!-- wiki:index:start -->
## 📚 Knowledge map
...
<!-- wiki:index:end -->

## What this is
... (rest of normal CLAUDE.md content)
```

Order matters for visual hierarchy (active brainstorms most urgent → pending implementation queue → general knowledge map → free-form content), but the parser doesn't care about order — only the comment delimiters.

In team repos (`~/.claude/repos/agentteamland/{team}/`), the same blocks can live in `README.md` instead of `CLAUDE.md` (the team `README.md` plays the same loaded-by-Claude role for team-scope work).

## Add your own marker block

The pattern is just convention. To add your own automated section:

1. Pick a unique block name (e.g., `<!-- ci-status -->`).
2. Wrap your auto-rebuilt content in `<!-- block:start --> ... <!-- block:end -->`.
3. Have your script find/replace the block on every rebuild.

For example, a simple "current sprint" block:

```markdown
<!-- sprint:start -->
## 🏃 Current sprint

Sprint 5 — Phase 1.D-η — concept pages.
- [ ] knowledge-system page (EN + TR)
- [ ] children-and-learnings page (EN + TR)
- ...
<!-- sprint:end -->
```

Update with a script that takes a sprint name + checklist as input and replaces the block contents in `CLAUDE.md`. Loaded automatically with project context.

## Why HTML comments

Plain markdown headings (`## Active brainstorms`) would work as visual sections, but:

- Hand-editing inside them risks corrupting the auto-rebuild
- A regex find/replace would have to be heading-aware (fragile)
- The user might write a real "Active brainstorms" section with related-but-different content

HTML comments are:

- Invisible in rendered Markdown (no visual clutter when the block is empty / not relevant)
- Easy to find/update/remove with simple regex (no heading-parser needed)
- Distinct from human-written sections (no namespace collision)
- Auto-loaded by Claude Code's project-instruction mechanism (Claude reads them despite the `<!-- -->` framing)

## Related

- [`/brainstorm`](/skills/brainstorm) — writes/removes the `<!-- brainstorm:active -->` block
- [`/save-learnings`](/skills/drain) — writes the `<!-- wiki:index -->` block
- [Knowledge system](/guide/knowledge-system) — what the wiki:index block indexes
- [Concepts: Skill](/guide/concepts#skill) — where these conventions fit in the broader picture
