# Knowledge system — journal + wiki

The project's knowledge lives in two layers. That's it — two. Don't add more.

| Layer | Location | Purpose | Update style |
|---|---|---|---|
| **Journal** | `.atl/journal/{YYYY-MM-DD}.md` | Date-based historical record: what happened, what worked, what didn't, and why. | Append-only |
| **Wiki** | `.atl/wiki/{topic}.md` | Topic-based current truth: what is true NOW. | Replace / update |

There is no separate "memory" layer. (v1 had three — agent-memory + journal + wiki; the first two were both date-based, append-only, and redundant in practice, so they are one `journal/` now. The user-global layer + the profile team cover what "memory" was reaching for.)

### Journal

A date-stamped narrative: discoveries, decisions, bug fixes, cross-cutting notes ("for whoever touches X next…"), and a record of what each drain produced. Append-only; existing entries are never edited and never deleted. `*.local.md` is gitignored for genuinely private notes.

### Wiki

The living, topic-organized current truth — one page per concept. When a fact changes, the page is updated, not appended. Pages cross-link. The table of contents is the `<!-- wiki:index -->` block at the top of CLAUDE.md, which `/drain` auto-aggregates so agents discover pages at zero cost (that marker block is the ToC — there is no separately-maintained `index.md`).

Both layers are written by the `/drain` skill from your inline markers (see [learning-capture.md](learning-capture.md)) — wiki for topic-shaped current truth, journal for time-stamped narrative.

## Agent knowledge base (the team-side equivalent)

Inside an agent, `children/{topic}.md` files are the agent's own topic-based knowledge — the team-side equivalent of the wiki, carried with the agent into every project it's installed in. Each child carries a `knowledge-base-summary` frontmatter line that is auto-aggregated into the agent's `## Knowledge Base` section (see [agent-structure.md](agent-structure.md)).

So the model is two axes:

- **current-truth vs history** — wiki + agent `children/` (current) vs journal (history)
- **project vs agent** — `.atl/` (this project only) vs an agent's `children/` (every project the agent is in)

## Agent startup routine

At the start of a conversation, an agent reads, when applicable:

1. **Its own `agent.md`** — including the Knowledge Base section auto-aggregated from `children/*.md` frontmatter.
2. **The `<!-- wiki:index -->` block in CLAUDE.md** — the knowledge map, auto-loaded at zero cost. Discover relevant wiki pages from this list rather than scanning `.atl/wiki/` directly.
3. **Recent journal entries** when the task overlaps prior work (last 3–5 by default; extend when the task touches a long-running thread).
4. **Project-specific rules** under `.atl/` if present.

The agent does NOT read every wiki page — it reads the index and follows links only when the task touches that domain. Tight context, full discoverability.
