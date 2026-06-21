# `/wiki`

The wiki is the project's knowledge base — living, cross-referenced, always current. It answers one question: **"What is the current truth about X in this project?"** Unlike journal entries (append-only historical narrative) or settled-decision docs (static records), wiki pages are **actively maintained** — when knowledge changes, the old fact is replaced, not stacked alongside.

::: warning No standalone `/wiki` skill in v2
v2 has **no `/wiki` skill**. In v1, `/wiki` was an explicit four-mode skill (`init` / `ingest` / `query` / `lint`) you invoked by hand. In v2 the wiki is a **destination** of the learning loop, not a command: [`/drain`](/skills/drain) writes and updates wiki pages as it folds the learning queue into the knowledge base, and you read the pages directly (they live as plain Markdown under `.atl/wiki/`). This page documents what the wiki **is** and how its pages are shaped; for the write/update mechanics, see [`/drain`](/skills/drain).
:::

## Where the wiki lives

```
.atl/wiki/
├── index.md                ← Table of contents
├── {topic-1}.md            ← Knowledge pages (kebab-case, one concept per page)
├── {topic-2}.md
└── ...
```

Pages are also indexed inside the project's root `CLAUDE.md` via a `<!-- wiki:index -->` marker block, so every Claude session loads the wiki map at start.

## How pages get written: `/drain`

The wiki is **auto-maintained** — humans rarely edit pages by hand. The v2 learning loop keeps it current:

1. Claude drops silent `<!-- learning -->` markers inline during a conversation (per the learning-capture rule).
2. [`atl tick`](/cli/tick) transfers each marker into a durable queue exactly once. `atl` then reports **"N learning(s) pending"** at the next session start.
3. You run [`/drain`](/skills/drain). For each queued item, `/drain` infers a kebab-case topic and routes it: topic-shaped *current truth* lands in `<proj>/.atl/wiki/<topic>.md` (replacing or merging the stale part if the page exists), with a dated bullet also appended to the journal.

So a learning like *"Redis cache TTL should be 30 minutes, not 15"* updates `wiki/redis-ttl.md` to say "TTL is 30 minutes" — replacing the old "15 minutes", not adding a second line.

Because `/drain` writes the wiki, the standalone `ingest` / `lint` / `init` verbs from v1 are gone — ingestion happens through the queue, and there is no separate lint command. To read the wiki, open `index.md` and the relevant pages directly, or ask Claude in-session.

## Page format

Wiki pages follow a consistent structure so both humans and agents can read them quickly:

```markdown
# {Topic Title}

> Last updated: {date}
> Sources: [journal](../journal/...), [docs](../docs/...)

## Summary
{2-3 sentence overview}

## Current State
{What is true RIGHT NOW — not history, not plans, just current reality}

## Key Decisions
{Important decisions about this topic, with brief reasoning}

## Patterns & Rules
{Established conventions for this topic}

## Known Issues
{Current problems or limitations}

## Related
- [{related-topic-1}]({related-topic-1}.md)
- [{related-topic-2}]({related-topic-2}.md)
```

## Important rules

1. **Wiki = current truth.** Not history, not plans. What is true RIGHT NOW.
2. **Update, don't append.** When a fact changes, the old version is replaced. (History lives in the journal.)
3. **Cross-reference always.** Every page links to related pages.
4. **Auto-maintained.** Humans rarely edit the wiki directly — [`/drain`](/skills/drain) keeps it current from the learning queue.
5. **Agent-readable.** Pages are structured for both human and AI consumption — clear sections, no ambiguity.
6. **Topic-based, not date-based.** Unlike the journal (date-based), the wiki is organized by topic. One page per concept.
7. **Always include the WHY.** A fact without its reason rots — record the reasoning, not just the conclusion.

## Wiki vs. journal vs. settled docs

| Layer | Shape | Edited how |
|---|---|---|
| **Wiki** (`.atl/wiki/`) | Current truth, topic-organized | Replaced/merged in place by `/drain` |
| **Journal** (`.atl/journal/`) | Historical narrative, date-organized | Append-only by `/drain` |
| **Settled docs** (`.atl/docs/`) | Static records of completed decisions | Written once when a [`/brainstorm`](/skills/brainstorm) completes |

## Related

- [`/drain`](/skills/drain) — writes and updates the wiki pages from the learning queue.
- [`atl tick`](/cli/tick) — moves learning markers into the queue that `/drain` consumes.
- [`/brainstorm`](/skills/brainstorm) — `done` mode produces the settled docs the wiki cross-references.

## Source

The wiki has no dedicated skill in v2; its pages are produced by `/drain`.

- Spec: [core/skills/drain/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/drain/SKILL.md)
