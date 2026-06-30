# `/drain`

Fold the pending learning queue into the knowledge base — route each item to the wiki, the journal, or an agent's knowledge base, then ack it so it's deleted.

`/drain` is the **consuming half** of the v2 learning loop. Capture is automatic and deterministic: Claude drops silent `<!-- learning -->` markers during a conversation, and [`atl tick`](/cli/tick) transfers each one into a durable bbolt queue exactly once. This skill is the LLM half the CLI can't do itself — read each queued learning, decide where it belongs, integrate it, and ack it.

## When to use it

- When `atl` reports **"N learning(s) pending"** at session start.
- Any time you want to process the learning queue manually.

The CLI half is exposed by [`atl learnings`](#the-cli-half): `status` shows pending counts, `peek` lists the items this skill consumes, and `ack` deletes a processed item. The queue is keyed by the current project directory, so run the skill from the project whose learnings you want to drain.

## Why ack = delete

The queue guarantees exactly-once delivery and dedup. You never re-scan transcripts and never track state — you only `peek`, integrate, and `ack`. An acked item is **deleted** from the queue, so it can never be re-reported. The v1 re-report bug class is structurally gone: there is no state file to advance and nothing to dedup against, so re-running `/drain` on an empty queue is a no-op.

## Procedure

### 1. Peek the queue

Run in the project directory:

```bash
atl learnings peek --channel learning --json
```

Each item is `{id, channel, payload, enqueued_at}`. The `payload` is free text — the marker body that was captured. If the list is empty, report "nothing to drain" and stop.

### 2. Route each item by the shape of its payload

The v2 marker carries no `topic`/`kind` metadata — **you** infer the destination from the payload, and derive a kebab-case `topic` from the content (one concept: `auth-refresh`, `redis-ttl`).

| Payload shape | Destination |
|---|---|
| Topic-shaped current truth ("the right way to do auth is …") | **Wiki** — `<proj>/.atl/wiki/<topic>.md` (replace/merge if it exists) **+ journal** |
| Time-stamped narrative ("we tried X, then Y, Y worked") | **Journal only** — `<proj>/.atl/journal/<YYYY-MM-DD>.md` (append) |
| Domain knowledge for a specific installed agent | **Agent KB** — `<scope>/.claude/agents/<agent>/children/<topic>.md` + rebuild that agent's `## Knowledge Base` **+ journal** |
| A repeating workflow, a crystallized convention, a new domain with no owning agent, or an identity expansion of an agent/skill | **Structural** — do NOT write autonomously; collect and propose |

To find the owning agent, look at the installed agents under `<proj>/.claude/agents/` and `~/.claude/agents/` (project shadows global). If no agent clearly owns it, route to the wiki instead. Always include the **WHY** in what you write — a fact without its reason rots.

### 3. Write, then ack — one item at a time

Non-structural writes are silent (no confirmation). After each item is integrated, ack it so it leaves the queue:

```bash
atl learnings ack <id>
```

Ack **only after** the write succeeds. If you can't integrate an item, leave it (don't ack) and note it in the report.

### 4. Structural changes — propose, never auto-apply

For the "structural" row, do not author agents/skills/rules silently. Collect them and, at the end, propose each one through `AskUserQuestion` (new agent / new skill / new rule / identity change). This is the reactive-creation boundary: a human confirms structural growth. Ack a structural item only once its proposal is resolved.

### 5. Report

Summarize what landed where: per item, topic → destination; list any new files created and any structural proposals. Keep it short.

## The CLI half

`/drain` drives three deterministic verbs under [`atl learnings`](/cli/learnings):

```bash
atl learnings status          # pending counts per channel for this project
atl learnings peek            # list pending items (human-readable)
atl learnings peek --json     # the full machine-readable list the skill consumes
atl learnings peek --channel learning   # filter to one channel
atl learnings ack <id>        # delete a processed item from the queue
```

Flags:

- `peek --json` — emit pending items as JSON (id, channel, payload, enqueued_at).
- `peek --channel <name>` — filter to a single channel (e.g. `learning`).

`status` and `ack` take no flags. `ack` takes exactly one argument — the item `id`.

### Channels

The queue is multi-channel. `/drain` processes the **`learning`** channel only. The `profile-fact` channel is reserved for a future first-party profile team and is not handled here.

## Agent KB rebuild

An agent's knowledge base is `agent.md` + a `children/` directory. Each child is one topic and carries `knowledge-base-summary` frontmatter:

```markdown
---
knowledge-base-summary: "<one-line summary used in agent.md's Knowledge Base section>"
---

# <Topic Title>

<the actual knowledge — patterns, examples, the why>
```

After writing or updating a child, **fully rebuild** `agent.md`'s `## Knowledge Base` section from the children's frontmatter (sorted by filename). That section is derived, not hand-edited — it's replaced wholesale each run. The same pattern applies to a skill's `learnings/` directory and its `## Accumulated Learnings` section, if a learning targets a skill.

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
