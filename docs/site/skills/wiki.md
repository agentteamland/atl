# `/wiki`

Project knowledge base — living, cross-referenced, always current. Initialize, ingest new knowledge, query existing, lint for staleness.

The wiki answers: **"What is the current truth about X in this project?"** Unlike journal entries (which are append-only historical narrative) or settled-decision docs (which are static records), wiki pages are **actively maintained** — when knowledge changes, the old fact is replaced, not stacked alongside.

Ships as a global skill in [core](https://github.com/agentteamland/core).

## Where the wiki lives

```
.atl/wiki/
├── index.md                ← Auto-maintained table of contents
├── {topic-1}.md            ← Knowledge pages (kebab-case, one concept per page)
├── {topic-2}.md
└── ...
```

Pages are also indexed inside the project's root `CLAUDE.md` via a `<!-- wiki:index -->` marker block — auto-rebuilt by [`/save-learnings`](/skills/save-learnings) so every Claude session loads the wiki map at start.

## Four modes

### `init` — scaffold an empty wiki

```
/wiki init
```

Creates `.atl/wiki/` and an `index.md` template. Idempotent — re-running on an already-initialized wiki is a no-op (`wiki: already initialized (N pages)`).

`init` is also fired automatically on a project's first session when `atl setup-hooks` is configured but the project has no `.atl/wiki/` yet.

### `ingest` — pull knowledge from project sources into wiki pages

```
/wiki ingest
```

Scans every project knowledge source and updates wiki pages:

| Source | Why it matters |
|---|---|
| `<!-- learning -->` markers in the current session transcript | Primary source — live conversation knowledge |
| `.atl/journal/*.md` | Per-agent, per-date learning record (the post-Q4 single layer) |
| `.atl/docs/*.md` | Settled-decision documents from completed brainstorms |
| `.atl/brain-storms/*.md` (status: completed only) | Decision context |
| Recent conversation context | What was just discussed or built |

For each piece of knowledge: determine the topic, find or create the wiki page, **merge new info while resolving contradictions**, update cross-references and `index.md`.

> Wiki pages reflect **current truth.** If old memory says "we use pattern X" but later memory says "X caused problems, switched to Y," the wiki page says "we use Y" — not both.

### `query` — search the wiki for a question

```
/wiki query how does caching work in this project?
```

Reads `index.md` to find relevant pages, reads those pages, synthesizes an answer with citations to the source pages.

Useful when:

- A new developer (or a new Claude session) needs to understand a topic without re-reading the whole repo
- You forgot how something was decided / implemented
- You want a quick refresher before making a change

### `lint` — health check the entire wiki

```
/wiki lint
```

Checks performed:

| Check | Action |
|---|---|
| Stale pages (>30 days, source files have changed since) | Flag for review |
| Contradictions between pages | Flag for resolution |
| Orphan pages (no incoming links) | Suggest connections |
| Missing pages (referenced but no file) | Suggest creation; create stub |
| Duplicate topics | Suggest merge |
| Index sync (`index.md` ↔ actual files) | Auto-fix if out of sync |

Auto-fixable issues are fixed silently. Contradictions and stale content are reported for human review.

Sample output:

```
Wiki Health Report:
──────────────────────────
📊 Total pages: 12
✅ Healthy: 9
⚠️  Stale (>30 days): 2 (caching-patterns.md, email-setup.md)
❌ Contradiction found: auth.md says "15min token" but jwt-config.md says "30min token"
🔗 Orphan: database-indexes.md (no incoming links)
📝 Missing: "rate-limiting" referenced in api-endpoints.md but no page exists
──────────────────────────

Fixing automatically...
✅ index.md synced
✅ Created rate-limiting.md (stub)
⚠️  Review needed: auth.md vs jwt-config.md contradiction
⚠️  Review needed: 2 stale pages
```

## Page format

Wiki pages follow a consistent structure so both humans and agents can read them quickly:

```markdown
# {Topic Title}

> Last updated: {date}
> Sources: [journal](../journal/...), [brainstorm](../brain-storms/...)

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

## How it interacts with other skills

The wiki is mostly **auto-maintained** — humans rarely edit pages by hand. The normal flow when `atl setup-hooks` is configured:

1. Claude drops `<!-- learning topic=... -->` markers inline during a conversation (per [learning-capture rule](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md))
2. **SessionStart of the next session** → `atl session-start` runs `atl learning-capture --previous-transcripts`, scans for markers
3. If markers found → Claude invokes [`/save-learnings --from-markers`](/skills/save-learnings)
4. `/save-learnings` updates `journal/`, agent `children/`, skill `learnings/`, and **wiki pages** (replace / update)

Example marker propagation:

```
<!-- learning topic: redis-cache; body: TTL should be 30 min, not 15 -->
  → journal/{date}_{agent}.md: append historical note with date
  → wiki/redis-cache.md: UPDATE "TTL is 30 minutes" (replace old "15 minutes")
  → (if domain-specific) agents/{agent}/children/redis-cache.md
```

[`/brainstorm done`](/skills/brainstorm) similarly ingests its decisions into the wiki when a brainstorm completes.

Without markers (or without hooks), `/wiki ingest` and `/save-learnings` (manual mode) still work — both can be invoked directly.

## Important rules

1. **Wiki = current truth.** Not history, not plans. What is true RIGHT NOW.
2. **Update, don't append.** When a fact changes, the old version is replaced.
3. **Cross-reference always.** Every page links to related pages. Orphans are flagged by `lint`.
4. **Auto-maintained.** Humans rarely edit wiki directly — `/save-learnings`, `/brainstorm done`, and `/wiki ingest` keep it current.
5. **Agent-readable.** Pages are structured for both human and AI consumption — clear sections, no ambiguity.
6. **Topic-based, not date-based.** Unlike journal (date-based), wiki is organized by topic. One page per concept.
7. **Lint regularly.** Run `/wiki lint` periodically (monthly, or when something feels off).

## Related

- [`/save-learnings`](/skills/save-learnings) — writes the wiki pages this skill queries / lints
- [`/brainstorm`](/skills/brainstorm) — `done` mode ingests decisions into wiki
- [Concepts: Skill](/guide/concepts#skill) — wiki vs journal vs settled docs

## Source

- Spec: [core/skills/wiki/skill.md](https://github.com/agentteamland/core/blob/main/skills/wiki/skill.md)
