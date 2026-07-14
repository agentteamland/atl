# Learning marker lifecycle

End-to-end picture of how knowledge flows from a conversation into the project's knowledge base. The v2 pattern is **inline marks → durable queue → auto-drain → ack** — cheap to write, captured automatically, drained automatically in the background, processed exactly once, and impossible to re-report.

The canonical rule lives at [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md). This page is the user-facing summary.

## The flow at a glance

```
[mid-conversation]          Claude marks a learning: a visible "📝 Learned: …"
                            line the user sees, plus a hidden <!-- learning: … -->
                            marker the pipeline captures. No tool call, no extra cost.
        ↓
atl tick                    A hook runs `atl tick` on every prompt. It parses the
(UserPromptSubmit hook,     hidden markers from this project's transcripts and
 every turn + session start)  enqueues each into the durable queue — exactly once,
                            deduped by content hash. It also reads the queue count.
        ↓
~/.atl/queue.db             One embedded bbolt file, per-project buckets keyed
                            by the working directory. No server, no daemon.
        ↓
[queue non-empty]           tick prints an AUTO-DRAIN signal into Claude's context:
                            "N learning(s) pending — auto-drain them now …".
        ↓
[same turn, background]     Claude spawns ONE background drain subagent (using the
                            running session's auth). It routes each item to the
                            wiki / journal / agent knowledge base, and acks it.
                            No one runs /drain by hand.
        ↓
atl learnings ack <id>      An acked item is DELETED from the queue.
        ↓
[loop closed]              A processed item is gone — it can never re-report.
                            There is no state file to advance.
```

The split is deliberate: **capture is automatic and deterministic** (markers → queue, done by the CLI), and **integration is automatic too** — the hook signals every turn the queue is non-empty, and the agent drains it in the background (the [`/drain`](/skills/drain) routing is the LLM half). The only remaining human touch point:

- **The user** answers an `AskUserQuestion` gate only when a drain proposes a *structural* change (a new agent / skill / rule, or an identity expansion). Routine writes to wiki / journal / agent KB happen silently in the background.

Nothing requires the user (or the agent) to remember to run `/drain` — that manual step is gone.

## What counts as a learning moment

Any of these, when it happens during a conversation, is a learning moment:

- **Bug fix** — a real bug was reproduced and fixed
- **Decision** — a choice was made between alternatives (JWT vs session, Redis vs memcached, 7d vs 15d refresh)
- **Pattern** — an approach turned out to be clean and reusable
- **Anti-pattern** — something was tried, failed, and we know why
- **Discovery** — a non-obvious fact about the system, library, or external service
- **Convention** — "from now on, we always / never do X"

Routine Q&A, file lookups, and mechanical edits are NOT learning moments. Don't mark every response.

## The mark format — a visible line + a hidden marker

When a learning moment occurs, Claude writes **both**: a visible line so the user sees what was learned in the moment, and a hidden HTML comment the hook captures.

```
📝 Learned: 7-day JWT refresh chosen — we want long sessions; the user logs in ~weekly.
<!-- learning: 7-day JWT refresh chosen — we want long sessions; the user logs in about once a week. -->
```

- The **visible line** (`📝 Learned: …`) renders in the chat — it's how the user sees, at capture time, what was picked up. This replaces the old separate "processed" log: the mark itself is the visibility.
- The **hidden marker** (`<!-- learning: … -->`) is invisible in rendered output but preserved in the transcript the hook scans. It carries the same fact, always **including the WHY**.

The marker is the **whole** capture format:

```
<!-- learning: <one to three sentences, always including the WHY> -->
```

No fields, no schema — just the fact and its reason in plain text. The [`/drain`](/skills/drain) routing reads the payload and infers where it belongs (a wiki topic, a journal entry, or an agent's knowledge base) and derives a kebab-case topic from the content. Multi-line is fine for a longer thought:

```html
<!-- learning:
Redis pool exhausted under load because each request opened its own client.
Fix: one shared pool. Symptom was intermittent timeouts at ~200 rps.
-->
```

**Always include the WHY.** A six-month-old "we chose X" with no reasoning is useless. One learning per mark — don't bundle unrelated learnings; each deserves its own line + marker.

> **Changed from v1.** The old marker carried structured YAML fields (`topic`, `kind`, `doc-impact`, `body`). v2 drops all of them: the payload is plain prose, and the drain does the routing the fields used to encode.

### The `profile-fact` channel

The queue is multi-channel. A second channel, `profile-fact`, captures durable facts about the user or the people they work with — same hidden-comment shape, `profile-fact:` prefix:

```html
<!-- profile-fact: Prefers TypeScript over JavaScript for all new services. -->
```

The learning auto-drain processes only the `learning` channel; `profile-fact` is handled by the profile team's own `/profile-drain` (installed with profile-team), not here.

## Why inline marks, not a tool call

A tool call per learning would double token cost and slow the conversation. Inline marks are embedded in text the agent was going to produce anyway. A grep-level pass inside [`atl tick`](/cli/tick) finds the hidden markers at ~0 cost; the AI-heavy drain only runs when items are queued — boring sessions stay free.

## When to skip marking

- Purely conversational turns (greetings, clarifications, status questions)
- Reading a file and summarizing its contents (no decision, no discovery)
- Routine edits where nothing surprising happened
- A learning already marked earlier in the same session (don't duplicate)

## Step-by-step under the hood

### 1. `atl tick` captures the markers and signals

[`atl setup-hooks`](/cli/setup-hooks) wires [`atl tick`](/cli/tick) to the `UserPromptSubmit` hook, and `atl session-start` runs a pass at session start. On each run, `tick`:

- discovers this project's Claude Code transcripts modified since the last tick,
- extracts the assistant text and parses `<!-- learning: ... -->` (and `<!-- profile-fact: ... -->`) hidden markers,
- **enqueues each into the durable queue exactly once** — idempotency comes from the queue's content-hash dedup, so re-draining the same text enqueues nothing new,
- reads the queue count and, when it's non-empty, prints the **auto-drain signal** into Claude's context (unthrottled, so it fires every turn there's pending work — the heavier capture pass is what the `--throttle` gates).

`tick` only enqueues and signals. It never integrates — folding a learning into the knowledge base is LLM work, so it stays on the skill side of the CLI/Skill boundary.

### 2. The durable queue

The queue is one embedded [bbolt](https://github.com/etcd-io/bbolt) file at `~/.atl/queue.db` — no server, no daemon. Every project's queue lives in that one file, isolated into per-project buckets keyed by the working directory. [`atl learnings`](/cli/learnings) is the deterministic read/ack surface:

```bash
atl learnings status                    # pending counts per channel (this project)
atl learnings peek                      # list pending items (human-readable)
atl learnings peek --channel learning --json   # the machine-readable list the drain consumes
atl learnings ack <id>                  # mark an item processed (delete it)
```

### 3. The auto-drain signal

Whenever the queue is non-empty — at session start and on every subsequent prompt — the hook reports the pending count as a short signal in Claude's `additionalContext`:

```
atl: 2 learning(s) pending — auto-drain them now in a background subagent (per the learning-capture rule)
```

When nothing is queued, output is empty (zero token cost).

### 4. The agent auto-drains in the background

On seeing that signal, the agent (per the [learning-capture rule](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md)) **spawns one background drain subagent** — no `/drain` command from the user, no waiting. The subagent inherits the running session's auth (so there's no separate headless `claude -p` and no auth problem), and runs the drain:

1. Reads the pending items with `atl learnings peek --channel learning --json` (`{id, channel, payload, enqueued_at}`).
2. Routes each item by the shape of its payload, deriving a kebab-case topic from the content:
   - **Topic-shaped current truth** → wiki page (`<proj>/.atl/wiki/<topic>.md`, replace/merge) + journal
   - **Time-stamped narrative** → journal only (`<proj>/.atl/journal/<YYYY-MM-DD>.md`, append)
   - **Domain knowledge for an installed agent** → that agent's `children/<topic>.md` + rebuild its `## Knowledge Base` section + journal
   - **Structural** (repeating workflow, crystallized convention, a new domain with no owning agent, an identity expansion) → propose via `AskUserQuestion`; never author autonomously
3. Writes each non-structural item silently, then **acks only after the write succeeds**.
4. Reports a short summary of what landed where.

**Single-in-flight:** the signal keeps appearing until the queue drains, so the agent spawns only one drain subagent at a time — a running one clears the queue; the agent doesn't stack a second. If a drain fails or a turn is missed, the items survive in the queue and the next turn's signal retries them, so **nothing is ever lost** — the worst case is a learning integrated a turn later.

### 5. ack = delete; the loop closes structurally

`atl learnings ack <id>` **deletes** the item from the queue. There is no state file to advance and nothing to dedup against later — a processed marker physically cannot come back.

This is what structurally kills v1's long-session re-report bug class: in v1, reports came from re-scanning an ever-growing transcript filtered against a JSON state file, and the filter could mis-fire. In v2, reports come from the queue, and processing removes the item. Re-running the drain on an empty queue is a no-op.

## When the hook isn't installed

The hidden markers are harmless without the hook — they're HTML comments, invisible in rendered output, inert as text (the visible `📝 Learned:` line still shows the user what was learned). The capture habit stays valuable.

For automatic capture + auto-drain, run [`atl setup-hooks`](/cli/setup-hooks). Without it, nothing enqueues or signals automatically; you can still force a capture pass yourself with [`atl tick`](/cli/tick) and then run [`/drain`](/skills/drain) manually. The markers accumulate in transcripts and remain available for whenever a `tick` pass runs.

## History

This flow has gone through four shapes:

1. **Original (pre-`atl`):** "Claude should proactively save learnings at the end of every session." Depended on Claude remembering a prose instruction. Unreliable.
2. **v1 (transcript scan + `/save-learnings`):** Inline markers carried structured YAML fields; a `SessionStart` hook re-scanned the previous session's transcripts, filtered against a JSON state file, and reported unprocessed markers. The re-scan-against-a-filter model was the source of a long-session re-report bug class, and the marker schema coupled capture to a docs-sync step.
3. **v2 (marker → bbolt queue → manual `/drain` → ack):** The marker became plain prose; [`atl tick`](/cli/tick) enqueues each into a durable queue exactly once; the re-report bug class is gone by construction. But integration still needed a human to run `/drain` after a session-start signal.
4. **Current (auto-drain + visible marks):** The mark is now a visible line + a hidden marker, and the hook signals every turn the queue is non-empty so the agent drains it **in the background automatically** — the manual `/drain` step is gone. The queue's durability makes a missed drain self-healing.

## Related

- [`atl tick`](/cli/tick) — the in-session pass that parses markers, enqueues them, and emits the auto-drain signal
- [`atl learnings`](/cli/learnings) — inspect and drain the durable queue (`status` / `peek` / `ack`)
- [`/drain`](/skills/drain) — the LLM half: routes each queued learning into the knowledge base, then acks it
- [`atl setup-hooks`](/cli/setup-hooks) — wires the `UserPromptSubmit` + `SessionStart` hooks that run `tick`
- [`atl doctor`](/cli/doctor) — surfaces the same pending count on demand
- [Knowledge system](/guide/knowledge-system) — where journal and wiki live
- [Children + learnings](/guide/children-and-learnings) — where agent / skill domain knowledge lands
- Canonical rule: [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md)
