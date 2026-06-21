# Knowledge system

How knowledge is organized in an `atl`-using project. Two layers: **journal** (date-based historical record) and **wiki** (topic-based current truth). That's it. Two layers. Don't add more.

The canonical rule lives at [`core/rules/knowledge-system.md`](https://github.com/agentteamland/atl/blob/main/core/rules/knowledge-system.md). This page is the user-facing summary.

There is no separate **memory** layer. v1 had three (agent-memory + journal + wiki); the first two were both date-based, append-only, and redundant in practice, so they are one `journal/` now. What "memory" was reaching for is covered by the agent's own knowledge base (its `children/`) plus the user-global layer.

## The two layers at a glance

| Layer | Location | Purpose | Update style |
|---|---|---|---|
| **Journal** | `.atl/journal/{YYYY-MM-DD}.md` | Date-based historical record: what happened, what worked, what didn't, and why. One file per day. | Append-only |
| **Wiki** | `.atl/wiki/{topic}.md` | Topic-based current truth. Reflects what is true NOW; old facts are replaced, not appended. | Replace / update |

Different paradigms, different purposes:

- **Journal answers** "what happened over time?" (chronological narrative)
- **Wiki answers** "what is true now?" (topic-based snapshot)

You can read either; they're not mutually exclusive. But they're written differently.

## Journal — append, never edit

Filename: `{YYYY-MM-DD}.md` — one file per day, shared across whatever ran that day (v1's per-agent `_{agent}` suffix is gone).

What goes here:

- Date-stamped narrative of what happened: discoveries, decisions, bug fixes, what worked, what didn't
- Cross-cutting notes ("for whoever touches X next: …")
- A record of what each drain produced (new wiki pages, new agent knowledge)
- User-approved structural changes (new skill / rule / agent decisions and rejections)

Rules:

- **Append-only.** Existing entries are not edited; new entries go at the end.
- **Never deleted** (historical record).
- **`*.local.md` filename pattern is gitignored** — use it for genuinely private content (uncommon).

The journal layer is what `.atl/agent-memory/` USED to be (per-agent history) merged with the original journal layer (cross-cutting signals). In practice the two had the same format (date + narrative) and frequently cited each other, so they are one layer now.

## Wiki — replace, current truth only

Filename: `{topic}.md` (kebab-case, one concept per page).

The project's living knowledge base. Unlike journal (historical record), wiki reflects **current truth** — when a fact changes, the page is updated, not appended.

Rules:

- **Organized by topic, not by date** (one page per concept)
- **Written by [`/drain`](/skills/drain)** from your inline `<!-- learning -->` markers — topic-shaped current truth lands here, time-stamped narrative goes to the journal
- **Pages reflect what is true NOW** — old info is replaced, not appended
- **Cross-referenced:** related pages link to each other
- **`index.md` is the table of contents**
- **A `<!-- wiki:index -->` marker block** at the top of `CLAUDE.md` auto-aggregates the topic list, so agents discover pages at zero cost

## How knowledge gets written: the learning loop

You don't write the journal or wiki by hand. They're fed by the v2 learning loop, split cleanly across the CLI/Skill boundary:

1. **Capture (automatic, deterministic).** During a conversation Claude drops silent `<!-- learning -->` markers when a learning moment occurs. The canonical capture rule is [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md).
2. **Enqueue (CLI).** [`atl tick`](/cli/tick) — wired to the `UserPromptSubmit` hook by [`atl setup-hooks`](/cli/setup-hooks) — transfers each marker into a durable [bbolt](https://github.com/etcd-io/bbolt) queue at `~/.atl/queue.db`, **exactly once** (marker-hash dedup). [`atl session-start`](/cli/setup-hooks) surfaces the pending count when a session opens.
3. **Drain (Skill, LLM).** [`/drain`](/skills/drain) reads the queued items via [`atl learnings`](/cli/learnings) (`status` / `peek` / `ack`), routes each to the wiki (topic truth), the journal (history), or an agent's knowledge base, then `ack`s it.

The deterministic half (capture + enqueue) is CLI; the judgment half (deciding *where* a learning belongs) is the `/drain` skill — the CLI can't do that part. An acked item is **deleted** from the queue, so it can never be re-reported: that processed-then-deleted design structurally kills v1's long-session re-report bug class.

If the hooks aren't installed, markers are harmless (they're invisible HTML comments) — run [`atl tick`](/cli/tick) and [`/drain`](/skills/drain) manually to process them.

## Agent startup routine

At the start of every conversation, the agent reads (when applicable):

1. **Its own agent file** — from team, via project-local copy. The `agent.md` ships with a Knowledge Base section auto-aggregated from `children/*.md` frontmatter (per [Children + learnings](/guide/children-and-learnings)).
2. **`CLAUDE.md` `<!-- wiki:index -->` block** — auto-loaded; gives the knowledge map at zero cost. Agents discover relevant wiki pages from this list rather than scanning `.atl/wiki/` directly.
3. **Recent journal entries** when the task overlaps with prior work — `.atl/journal/` (last 3–5 by default; extend the scope when the task touches a long-running thread).
4. **Project-specific rules** under `.atl/` if present.

The agent does NOT read all wiki pages. It reads the index (auto-loaded), and only follows links to detail pages when the task touches that domain. This keeps context tight while preserving discoverability.

## Why two layers, not three

v1 defined three layers: **memory** (per-project, per-agent, append-only history), **journal** (per-project, cross-agent signals, append-only), **wiki** (per-project, topic-based, replace/update).

The first two were both date-based, append-only, narrative-shaped. In every workspace they ended up cross-referencing each other or redundantly capturing the same events. The "agent's private memory vs. broadcast to others" distinction was never enforced — anyone could read either layer.

They were merged into a single `journal/` because:

- Same format → no semantic separation
- Same audience (all agents read both)
- Same write pattern (append by date)
- The split added cognitive overhead ("is this for me or for others?") without producing different content

The merged layer is just `journal/`. Wiki stays separate because its paradigm (topic-based current truth) is genuinely different from journal's (date-based history).

## The agent-side mirror: two axes

The same current-truth-vs-history split also exists on the team side, carried *with the agent* rather than scoped to one project. That gives two axes:

- **current-truth vs history** — wiki + an agent's `children/` (current) vs journal (history)
- **project vs agent** — `.atl/` (this project only) vs an agent's `children/` (every project the agent is installed in)

Concretely:

- **Agent children files** (`children/{topic}.md` in the agent's directory) are the agent-side equivalent of wiki — topic-based, replace/update, cross-project domain knowledge for the agent.
- **Skill learnings files** (`learnings/{topic}.md` in a skill's directory) are the per-skill equivalent — same shape, scoped to the skill.

Both carry a `knowledge-base-summary:` frontmatter field that's auto-aggregated into `agent.md` (Knowledge Base section) or `skill.md` (Accumulated Learnings section). See [Children + learnings](/guide/children-and-learnings) for the full pattern.

## Related

- [`/drain`](/skills/drain) — folds the learning queue into journal entries and wiki pages
- [`atl learnings`](/cli/learnings) — the deterministic queue plumbing (`status` / `peek` / `ack`) `/drain` drives
- [`atl tick`](/cli/tick) — transfers captured markers into the queue (the capture half of the loop)
- [Children + learnings](/guide/children-and-learnings) — the agent-side mirror of journal + wiki
- Canonical rule: [`core/rules/knowledge-system.md`](https://github.com/agentteamland/atl/blob/main/core/rules/knowledge-system.md)
