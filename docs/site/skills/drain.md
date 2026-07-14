# `/drain`

Fold the pending learning queue into the knowledge base — route each item to the wiki, the journal, or an agent's knowledge base, then ack it so it's deleted.

`/drain` is the **consuming half** of the v2 learning loop. Capture is automatic and deterministic: Claude marks learnings during a conversation (a visible `📝 Learned:` line plus a hidden `<!-- learning -->` marker), and [`atl tick`](/cli/tick) transfers each one into a durable bbolt queue exactly once. This skill is the LLM half the CLI can't do itself — and it now runs **automatically in the background**: when the queue is non-empty the hook signals the agent, which spawns a background drain subagent (per the [learning-capture rule](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md)) rather than waiting for a manual `/drain`. Run manually only to force a pass. It does two judgment jobs:

1. **Mine** the conversation for learnings the agent *forgot* to mark — the user's corrections, reverts, repeated mistakes — and enqueue them like any marker. Marker capture only catches what the agent noticed, and the mistakes worth not-repeating are exactly the ones it didn't.
2. **Quality-gate, then integrate** each queued item — decide whether it's worth keeping (Save / Improve / Absorb / Drop), then route the keepers to the wiki, the journal, or an agent's knowledge base, and ack each.

## When to use it

- When `atl` reports **"N learning(s) pending"** at session start.
- Any time you want to process the learning queue manually.

The CLI half is exposed by [`atl learnings`](#the-cli-half): `status` shows pending counts, `peek` lists the items this skill consumes, and `ack` deletes a processed item. The queue is keyed by the current project directory, so run the skill from the project whose learnings you want to drain.

## Why ack = delete

The queue guarantees exactly-once delivery and dedup by content hash. An acked item is **deleted** from the queue, so it can never be re-reported — the v1 re-report bug class is structurally gone. The mining step does read the recent conversation, but anything it captures is enqueued like a marker and deduped by the same content hash, so re-running `/drain` is safe: a lesson you already saved re-enqueues to a no-op, and an empty queue with nothing new to mine is a no-op too.

## Procedure

### 1. Mine the conversation for unmarked learnings

Before peeking the queue, harvest what the agent forgot to mark. Read the recent conversation flow (prose only — tool calls and results are stripped):

```bash
atl learnings transcript
```

Scan it for **durable** learnings that were never captured as a marker: **user corrections** (the user said the agent was wrong and how to fix it), **reverts** (an approach was tried, rejected, replaced), and **repeated mistakes** (the same class of error recurred). For each, write a one-line learning stating the lesson **with its why**, and enqueue it exactly like a marker:

```bash
atl learnings _enqueue learning "<the lesson, with its reason>"
```

Be **strict** — mine only what's worth never-repeating; a one-off or anything already obvious is noise. The queue dedups by content hash, so re-mining the same lesson is a safe no-op, but the real filter is the quality gate in step 3. If nothing qualifies, enqueue nothing.

### 2. Peek the queue

Run in the project directory:

```bash
atl learnings peek --channel learning --json
```

Each item is `{id, channel, payload, enqueued_at}` — now both agent-dropped markers and anything you just mined. The `payload` is free text. If the list is empty, report "nothing to drain" and stop.

### 3. Quality-gate each item before persisting

A learning store rots when it bloats, so don't save blindly. For each item, **first grep the existing knowledge** (`.atl/wiki/`, `.atl/journal/`, any owning agent's `children/`), then give a holistic verdict:

| Verdict | When | Action |
|---|---|---|
| **Save** | New, durable, worth keeping | Route + write (steps 4–5) |
| **Improve-then-Save** | Worth keeping but vague as written | Sharpen the wording, then route + write |
| **Absorb** | An existing page/entry already covers it | Merge the nuance into that note — no new file — then ack |
| **Drop** | Trivial, one-off, redundant, or obvious | Ack, write nothing |

This is a **holistic** judgment, not a numeric score. Lean toward Absorb over a near-duplicate page and Drop over a marginal one — the bar for a standalone entry is "a future session would be glad this exists." Absorb and Drop both end in an `ack`; only Save / Improve-then-Save proceed to routing.

### 4. Route each kept item by the shape of its payload

The v2 marker carries no `topic`/`kind` metadata — **you** infer the destination from the payload, and derive a kebab-case `topic` from the content (one concept: `auth-refresh`, `redis-ttl`).

| Payload shape | Destination |
|---|---|
| Topic-shaped current truth ("the right way to do auth is …") | **Wiki** — `<proj>/.atl/wiki/<topic>.md` (replace/merge if it exists) **+ journal** |
| Time-stamped narrative ("we tried X, then Y, Y worked") | **Journal only** — `<proj>/.atl/journal/<YYYY-MM-DD>.md` (append) |
| Domain knowledge for a specific installed agent | **Agent KB** — `<scope>/.claude/agents/<agent>/children/<topic>.md` + rebuild that agent's `## Knowledge Base` **+ journal** |
| A repeating workflow, a crystallized convention, a new domain with no owning agent, or an identity expansion of an agent/skill | **Structural** — do NOT write autonomously; collect and propose |

To find the owning agent, look at the installed agents under `<proj>/.claude/agents/` and `~/.claude/agents/` (project shadows global). If no agent clearly owns it, route to the wiki instead. Always include the **WHY** in what you write — a fact without its reason rots.

### 5. Write, then ack — one item at a time

Non-structural writes are silent (no confirmation). After each item is integrated, ack it so it leaves the queue:

```bash
atl learnings ack <id>
```

Ack **only after** the write succeeds. If you can't integrate an item, leave it (don't ack) and note it in the report.

### 6. Structural changes — propose, never auto-apply

For the "structural" row, do not author agents/skills/rules silently. Collect them and, at the end, propose each one through `AskUserQuestion` (new agent / new skill / new rule / identity change). This is the reactive-creation boundary: a human confirms structural growth. Ack a structural item only once its proposal is resolved.

### 7. Report

Summarize what landed where: per item, topic → destination; list any new files created and any structural proposals. Keep it short.

## The CLI half

`/drain` drives deterministic verbs under [`atl learnings`](/cli/learnings):

```bash
atl learnings transcript      # recent conversation flow for the mining step (step 1)
atl learnings status          # pending counts per channel for this project
atl learnings peek            # list pending items (human-readable)
atl learnings peek --json     # the full machine-readable list the skill consumes
atl learnings peek --channel learning   # filter to one channel
atl learnings ack <id>        # delete a processed item from the queue
```

Flags:

- `transcript --limit <n>` — read the most recent N transcripts (default 2); `--json` emits role/text records.
- `peek --json` — emit pending items as JSON (id, channel, payload, enqueued_at).
- `peek --channel <name>` — filter to a single channel (e.g. `learning`).

`status` takes no flags. `ack` takes exactly one argument — the item `id`. The mining step itself enqueues with the hidden `atl learnings _enqueue learning "<lesson>"` helper (the same one capture uses), so dedup lives in the queue.

### Channels

The queue is multi-channel. `/drain` processes the **`learning`** channel only. The `profile-fact` channel is handled by the shipped profile-team's `/profile-drain` (installed with profile-team), not here.

## Agent KB rebuild

An agent's knowledge base is `agent.md` + a `children/` directory. Each child is one topic and carries `knowledge-base-summary` frontmatter:

```markdown
---
knowledge-base-summary: "<one-line summary used in agent.md's Knowledge Base section>"
---

# <Topic Title>

<the actual knowledge — patterns, examples, the why>
```

After writing or updating a child, **fully rebuild** `agent.md`'s `## Knowledge Base` section from the children's frontmatter (sorted by filename). That section is derived, not hand-edited — it's replaced wholesale each run. Skills have no equivalent: they are procedures, not knowledge stores, so `/drain` rebuilds only an agent's `## Knowledge Base` — there is no skill "Accumulated Learnings" section.

## Wiki index rebuild

Whenever a `/drain` run writes or updates a `.atl/wiki/` page, it rebuilds the `<!-- wiki:index -->` block in the project's `CLAUDE.md` so the knowledge map stays in sync — one `- [topic](.atl/wiki/topic.md) — summary` line per page, sorted by filename, derived (not hand-edited). If the project has no `CLAUDE.md`, the rebuild is skipped (`atl init` / `atl install` create the file). See [Claude Code conventions](/guide/claude-code-conventions) for the block's format and placement.

## Examples

### Drain after a session-start prompt

A new session opens and `atl` reports two pending learnings. Process them:

```bash
atl learnings peek --channel learning --json
```

```json
[
  {
    "id": "9f1c2a3b4d5e",
    "channel": "learning",
    "payload": "Redis cache TTL should be 30 minutes, not 15 — 15 caused cold-start thrash under load.",
    "enqueued_at": "2026-06-21T09:14:02Z"
  }
]
```

Write `redis-ttl` to the wiki (current truth) and append a dated bullet to today's journal, then ack:

```bash
atl learnings ack 9f1c2a3b4d5e
```

### Check the queue without draining

```bash
atl learnings status
```

```
learning queue — pending by channel:
  learning       2
```

## Scope

Wiki and journal are project knowledge under `<proj>/.atl/`. Agent KB follows the agent's install scope — a project `.claude/` shadows global `~/.claude/`.

## Source

- Spec: [core/skills/drain/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/drain/SKILL.md)
- CLI: [cli/cmd/atl/commands/learnings.go](https://github.com/agentteamland/atl/blob/main/cli/cmd/atl/commands/learnings.go)
