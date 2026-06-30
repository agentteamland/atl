---
name: drain
description: Fold pending learning-queue items into the knowledge base — route each to the wiki (topic truth), the journal (history), or an agent's knowledge base, then ack it so it's deleted. Run when atl reports "N learning(s) pending", or to process the learning queue manually.
---

# /drain — fold the learning queue into the knowledge base

This is the **consuming half** of the v2 learning loop. Capture (silent markers →
`atl tick` → bbolt queue) is automatic and deterministic; this skill is the LLM
half the CLI can't do itself. It does two judgment jobs:

1. **Mine** the conversation for learnings the agent *forgot* to mark — the user's
   corrections, reverts, repeated mistakes — and enqueue them like any marker. This
   is the missing input side: marker capture only catches what the agent noticed,
   and the mistakes worth not-repeating are exactly the ones it didn't.
2. **Quality-gate, then integrate** each queued item: decide whether it's worth
   keeping (Save / Improve / Absorb / Drop), then route the keepers to the wiki,
   the journal, or an agent's knowledge base, and ack each.

The queue guarantees exactly-once and dedup by content hash, so mining is safe to
re-run (a learning you've seen before re-enqueues to a no-op) and an acked item is
**deleted**, so it can never be re-reported (the v1 re-report bug class is
structurally gone).

## Procedure

### 1. Mine the conversation for unmarked learnings
Before peeking the queue, harvest what the agent forgot to mark. Read the recent
conversation flow (prose only — tool calls/results are stripped):

```
atl learnings transcript
```

Scan it for **durable** learnings the agent never captured as a marker:

- **User corrections** — the user told the agent it was wrong and how to do it right
  ("no, use refresh tokens not sessions", "stop editing the config, fix the code").
- **Reverts / do-overs** — an approach was tried, rejected, and replaced.
- **Repeated mistakes** — the same class of error recurred across the session.

For each one, write a one-line learning **stating the lesson, with the why**, and
enqueue it exactly like a marker:

```
atl learnings _enqueue learning "<the lesson, with its reason>"
```

Be **strict** — mine only what's worth never-repeating. A preference voiced once,
a one-off, or anything already obvious is noise; skip it. The queue dedups by
content hash, so re-mining the same lesson is a safe no-op — but don't lean on
that to lower the bar. (The quality gate in step 3 is the real filter; this step
just feeds it candidates.) If nothing qualifies, enqueue nothing and move on.

### 2. Peek the queue
Run in the project directory (the queue is keyed by cwd):

```
atl learnings peek --channel learning --json
```

Each item is `{id, channel, payload, enqueued_at}` — now both agent-dropped markers
and anything you just mined. The `payload` is free text — the captured marker body.
If the list is empty, report "nothing to drain" and stop.

### 3. Quality-gate each item before persisting
Don't blindly save every item — a learning store rots when it bloats. For each
item, **first grep the existing knowledge** (the project `.atl/wiki/`,
`.atl/journal/`, and any owning agent's `children/`) for the same topic, then give
a holistic verdict:

| Verdict | When | Action |
|---|---|---|
| **Save** | New, durable, worth keeping | Route + write (steps 4–5) |
| **Improve-then-Save** | Worth keeping but vague/unclear as written | Sharpen the wording (keep the why), then route + write |
| **Absorb** | A page/entry already covers this | Merge the new nuance into the existing note — **no new file** — then ack |
| **Drop** | Trivial, one-off, redundant, or already obvious | Ack and write nothing |

This is a **holistic** judgment, not a score — no confidence numbers. Lean toward
**Absorb** over a near-duplicate new page and **Drop** over a marginal one; the
bar for a standalone entry is "a future session would be glad this exists." Absorb
and Drop both end in an `ack` (the item is handled — it just didn't become a new
file). Only Save / Improve-then-Save proceed to routing.

### 4. Route each kept item by the shape of its payload
The v2 marker carries no `topic`/`kind` metadata — **you** infer the destination
from the payload. Derive a kebab-case `topic` from the content (one concept:
`auth-refresh`, `redis-ttl`).

| Payload shape | Destination |
|---|---|
| Topic-shaped current truth ("the right way to do auth is …") | **Wiki** — `<proj>/.atl/wiki/<topic>.md` (replace/merge if it exists) **+ journal** |
| Time-stamped narrative ("we tried X, then Y, Y worked") | **Journal only** — `<proj>/.atl/journal/<YYYY-MM-DD>.md` (append) |
| Domain knowledge for a specific installed agent ("api-agent: prefer prepared statements") | **Agent KB** — `<scope>/.claude/agents/<agent>/children/<topic>.md` + rebuild that agent's `## Knowledge Base` **+ journal** |
| A repeating workflow, a crystallized convention, a new domain with no owning agent, or an identity expansion of an existing agent/skill | **Structural** — do NOT write autonomously; collect and propose (step 6) |

To find the owning agent, look at the installed agents under
`<proj>/.claude/agents/` and `~/.claude/agents/` (project shadows global). Match
on the agent's area; if no agent clearly owns it, route to the wiki instead.

Always include the **WHY** in what you write — a fact without its reason rots.

### 5. Write, then ack — one item at a time
Non-structural writes are **silent** (no confirmation needed):

- **Wiki** (`<proj>/.atl/wiki/<topic>.md`): current truth. If the page exists,
  merge/replace the stale part — don't blindly append. Topic = filename.
- **Journal** (`<proj>/.atl/journal/<YYYY-MM-DD>.md`): append a dated bullet
  (`- <topic>: <body, with the why>`). Create with a `# <date>` heading if new.
- **Agent KB**: write `children/<topic>.md` with the required frontmatter, then
  rebuild the agent's summary section (see below).

After each item is integrated, ack it so it leaves the queue:

```
atl learnings ack <id>
```

Ack **only after** the write succeeds. If you can't integrate an item, leave it
(don't ack) and note it in the report.

### 6. Structural changes — propose, never auto-apply
For the "structural" row, do not author agents/skills/rules silently. Collect
them and, at the end, use **AskUserQuestion** to propose each one (new agent /
new skill / new rule / identity change). This is the reactive-creation boundary
(decision doc D-1): a human confirms structural growth. Ack a structural item
only once its proposal is resolved (applied or declined).

### 7. Report
Summarize what landed where: per item, topic → destination; list any new files
created and any structural proposals. Keep it short.

## Agent KB rebuild (the `## Knowledge Base` contract)

An agent's knowledge base is `agent.md` + a `children/` directory. Each child is
one topic and MUST carry frontmatter:

```markdown
---
knowledge-base-summary: "<one-line summary used in agent.md's Knowledge Base section>"
---

# <Topic Title>

<the actual knowledge — patterns, examples, the why — as long as needed>
```

After writing/updating a child, **fully rebuild** `agent.md`'s `## Knowledge
Base` section from the children's frontmatter (sort by filename):

```markdown
## Knowledge Base

### <Topic 1 (title-cased from filename)>
<knowledge-base-summary>
→ [Details](children/topic-1.md)

### <Topic 2>
...
```

This section is derived, not hand-edited — replace it wholesale each run; the
source of truth is each child's frontmatter. (Same pattern applies to a skill's
`learnings/` + `## Accumulated Learnings`, if a learning targets a skill.)

## Wiki index rebuild (the `<!-- wiki:index -->` contract)

Whenever this run writes or updates any `<proj>/.atl/wiki/<topic>.md` page, rebuild
the project's knowledge-map block so the index stays in sync with the pages. (No
wiki write this run → skip; nothing to update.)

Target: the `<!-- wiki:index:start --> … <!-- wiki:index:end -->` block in the
**project root** `CLAUDE.md`. If the project has no `CLAUDE.md`, skip the rebuild
(don't create the file from here — `atl init` / `atl install` own that).

Rebuild the block contents from every `<proj>/.atl/wiki/*.md`, sorted
alphabetically by filename:

```markdown
<!-- wiki:index:start -->
## 📚 Knowledge map

Knowledge lives in `.atl/wiki/` (current truth, topic-organized) and `.atl/journal/` (historical record, date-based). Before working on a topic, scan this list — if a page looks relevant, read it before deciding.

- [<topic>](.atl/wiki/<topic>.md) — <summary>
<!-- wiki:index:end -->
```

Each entry is one line: `- [<topic>](.atl/wiki/<topic>.md) — <summary>`, where
`<topic>` is the filename without `.md` and `<summary>` is the page's first
non-frontmatter, non-heading line. The block is **derived, not hand-edited** —
replace its contents wholesale each run (same discipline as the agent KB).

**Placement:** if the block exists, replace its contents in place. If it's absent,
insert it near the top of `CLAUDE.md` — after the H1 + intro, before the first
plain `##` section, and **below** any `<!-- brainstorm:active -->` /
`<!-- pending-implementation -->` blocks (the knowledge map is the least urgent of
the three managed blocks, so it sits last).

## Notes

- **Scope**: wiki + journal are project knowledge (`<proj>/.atl/`). Agent KB
  follows the agent's install scope (project `.claude/` shadows global `~/.claude/`).
- **profile-fact channel**: not handled here — that's profile-team's drain
  (a future first-party team). This skill processes the `learning` channel only.
- **Idempotency**: ack deletes the item, and both capture and mining dedup by
  content hash — so re-running `/drain` is safe. Mining a lesson you already saved
  re-enqueues to a no-op; an empty queue with nothing new to mine is a no-op.
