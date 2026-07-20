# Claude Code conventions

How ATL shapes `CLAUDE.md` — the file Claude Code auto-loads as instructions. Two halves: the **three tiers** (one lean file per layer, with token budgets and an ownership model) and the **marker blocks** the project tier carries to coordinate state across sessions.

[`atl init`](/cli/init) scaffolds a starter for each tier; `atl install` drops the project starter automatically when a project has none. This page is the shape those scaffolds embody.

## The three tiers

One always-loaded file per tier, each a lean starter you grow. Authority layers (identity → rules → guidance) are **ordered sections within one file**, not separate files — no `SOUL`/`RULES`/`AGENTS` multiplication.

| Tier | File | What it holds | Soft budget |
|---|---|---|---|
| **global** | `~/.claude/CLAUDE.md` | Pure user **persona** — how you want Claude to work everywhere (language, style, the engineering defaults you want in every project). **ATL manages nothing here.** | ≤ ~80 lines |
| **project** | `<project>/CLAUDE.md` | **Hybrid:** ATL-managed marker blocks (below) + your own evidence-fillable facts (stack, commands, conventions) + an optional ≤5-row glob→skill routing table. | ≤ ~60 lines |
| **monorepo** | `<repo>/CLAUDE.md` | The project shape **specialized + lean**: a layout table and conventions as **pointers**, not inlined content. | ~30 lines |

The always-loaded chain (global + project) should stay under ~300 lines — every line costs context on every session, so each section earns its place against the budget.

### Ownership — managed vs. user-owned

In the **project** tier, content inside a marker block (the three blocks below) is **ATL-managed**: the `/brainstorm` and `/drain` skills write and rewrite these blocks, so hand-edits inside the markers don't survive. Everything **outside** the markers is **yours** — `atl init` seeds it once and never touches it again (it only ever writes a `CLAUDE.md` that doesn't already exist). The **global** tier has no managed blocks at all: it is entirely yours.

Fill the user-owned facts from **how the code actually behaves** — capture the stack, the real build/test commands, the conventions in use. Don't invent business rules, and align meta-architecture rather than restating syntax.

### Volatility split

Volatile working/sprint state does **not** belong in the always-loaded file — it would cost context every session and churn constantly. Keep it in a separate, read-on-demand tracker and leave only a pointer in `CLAUDE.md` (the project starter's `## State` section is that pointer). The rule of thumb: detailed for the current sprint only; summarize out to docs/journal once it no longer shapes execution.

## The three blocks

| Block | Written by | Purpose |
|---|---|---|
| `<!-- wiki:index -->` | [`/drain`](/skills/drain) | Auto-rebuilt table of contents for `.atl/wiki/` pages. Loads with project context, gives Claude the knowledge map at zero cost. |
| `<!-- brainstorm:active -->` | [`/brainstorm start`](/skills/brainstorm) and [`/brainstorm done`](/skills/brainstorm) | Pins active brainstorm topics into project context so the next session cannot miss them. |
| `<!-- pending-implementation -->` | Brainstorm `done` flow | Reminds the next session that a brainstorm decided X but the implementation hasn't shipped yet. |

All three use the same `<!-- block:start --> ... <!-- block:end -->` delimiter pattern. None of them have a parser in the strict sense — they're convention, not syntax. But the convention is consistent enough to find/update/remove with simple `sed`/regex when needed.

> **Note — why the example block contents below are in English (even on the Turkish mirror of this page):** The three templates (`wiki:index`, `brainstorm:active`, `pending-implementation`) are produced verbatim by `/drain` and `/brainstorm`. Per the platform's `communication-style` rule (committed artifacts are English-only), these skills always write English — committed files must be English regardless of the conversation language. So in a Turkish project, the `CLAUDE.md` blocks still appear in English — the examples reflect the actual output.

## `<!-- wiki:index -->` — knowledge map

Rebuilt by [`/drain`](/skills/drain) whenever it writes or updates a wiki page. A table of contents for `.atl/wiki/` pages, near the top of `CLAUDE.md`, after the H1 + intro:

```markdown
<!-- wiki:index:start -->
## 📚 Knowledge map

Knowledge lives in `.atl/wiki/` (current truth, topic-organized) and `.atl/journal/` (historical record, date-based). Before working on a topic, scan this list — if a page looks relevant, read it before deciding.

- [docs-audit-false-positive-rate](.atl/wiki/docs-audit-false-positive-rate.md) — ~40% of multi-agent docs-drift audit reports include hallucinated findings
- [pr-merge-discipline](.atl/wiki/pr-merge-discipline.md) — never `gh pr merge` from Claude; surface URL and stop
- [complexity-resistance](.atl/wiki/complexity-resistance.md) — when a proposal needs paragraphs to defend, that's a smell
<!-- wiki:index:end -->
```

Each entry is one line: `- [topic](.atl/wiki/topic.md) — one-line summary` (sorted alphabetically by filename). The summary comes from the first non-frontmatter, non-heading line of each wiki page. The block is rebuilt programmatically — hand-edits inside the markers are overwritten on the next `/drain` run, so to add a topic you create the wiki page (or let `/drain` create it), and the index follows.

## `<!-- brainstorm:active -->` — active topics pin

Written by `/brainstorm start`, removed by `/brainstorm done`. Lives near the top of the scope's `CLAUDE.md` (project) or `~/.claude/CLAUDE.md` (global) or team `README.md` (team-scope):

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[profile-team](.atl/brain-storms/profile-team.md)** (project, 2026-05-08) — schema, storage, privacy, and initial scope for the new profile-team package
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

In team repos (assets installed under `.claude/` at the relevant scope), the same blocks can live in `README.md` instead of `CLAUDE.md` (the team `README.md` plays the same loaded-by-Claude role for team-scope work).

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
- [`/drain`](/skills/drain) — rebuilds the `<!-- wiki:index -->` block
- [Knowledge system](/guide/knowledge-system) — what the wiki:index block indexes
- [Concepts: Skill](/guide/concepts#skill) — where these conventions fit in the broader picture
