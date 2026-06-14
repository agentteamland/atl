---
name: drain
description: Fold pending learning-queue items into the knowledge base — route each to the wiki (topic truth), the journal (history), or an agent's knowledge base, then ack it so it's deleted. Run when atl reports "N learning(s) pending", or to process the learning queue manually.
---

# /drain — fold the learning queue into the knowledge base

This is the **consuming half** of the v2 learning loop. Capture (silent markers →
`atl tick` → bbolt queue) is automatic and deterministic; this skill is the LLM
half the CLI can't do itself: read each queued learning, decide where it belongs,
integrate it, and ack it.

The queue already guarantees exactly-once and dedup. You never re-scan
transcripts and never track state — you only `peek`, integrate, and `ack`. An
acked item is **deleted**, so it can never be re-reported (the v1 re-report bug
class is structurally gone).

## Procedure

### 1. Peek the queue
Run in the project directory (the queue is keyed by cwd):

```
atl learnings peek --channel learning --json
```

Each item is `{id, channel, payload, enqueued_at}`. The `payload` is free text —
the marker body the user/assistant captured. If the list is empty, report
"nothing to drain" and stop.

### 2. Route each item by the shape of its payload
The v2 marker carries no `topic`/`kind` metadata — **you** infer the destination
from the payload. Derive a kebab-case `topic` from the content (one concept:
`auth-refresh`, `redis-ttl`).

| Payload shape | Destination |
|---|---|
| Topic-shaped current truth ("the right way to do auth is …") | **Wiki** — `<proj>/.atl/wiki/<topic>.md` (replace/merge if it exists) **+ journal** |
| Time-stamped narrative ("we tried X, then Y, Y worked") | **Journal only** — `<proj>/.atl/journal/<YYYY-MM-DD>.md` (append) |
| Domain knowledge for a specific installed agent ("api-agent: prefer prepared statements") | **Agent KB** — `<scope>/.claude/agents/<agent>/children/<topic>.md` + rebuild that agent's `## Knowledge Base` **+ journal** |
| A repeating workflow, a crystallized convention, a new domain with no owning agent, or an identity expansion of an existing agent/skill | **Structural** — do NOT write autonomously; collect and propose (step 4) |

To find the owning agent, look at the installed agents under
`<proj>/.claude/agents/` and `~/.claude/agents/` (project shadows global). Match
on the agent's area; if no agent clearly owns it, route to the wiki instead.

Always include the **WHY** in what you write — a fact without its reason rots.

### 3. Write, then ack — one item at a time
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

### 4. Structural changes — propose, never auto-apply
For the "structural" row, do not author agents/skills/rules silently. Collect
them and, at the end, use **AskUserQuestion** to propose each one (new agent /
new skill / new rule / identity change). This is the reactive-creation boundary
(decision doc D-1): a human confirms structural growth. Ack a structural item
only once its proposal is resolved (applied or declined).

### 5. Report
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

## Notes

- **Scope**: wiki + journal are project knowledge (`<proj>/.atl/`). Agent KB
  follows the agent's install scope (project `.claude/` shadows global `~/.claude/`).
- **profile-fact channel**: not handled here — that's profile-team's drain
  (a future first-party team). This skill processes the `learning` channel only.
- **Idempotency**: ack deletes the item; there is nothing to dedup against and no
  state file to advance. Re-running `/drain` on an empty queue is a no-op.
