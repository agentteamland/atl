# Learning marker lifecycle

End-to-end picture of how knowledge flows from a conversation into the project's knowledge base. The v2 pattern is **inline markers → durable queue → drain → ack** — cheap to write, captured automatically, processed exactly once, and impossible to re-report.

The canonical rule lives at [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md). This page is the user-facing summary.

## The flow at a glance

```
[mid-conversation]          Claude drops <!-- learning: ... --> markers inline
                            as it speaks. No tool call, no extra cost.
        ↓
atl tick                    A hook runs `atl tick` on every prompt (throttled)
(hook-run, every few min    and at session start. It parses markers from this
 + at session start)        project's transcripts and enqueues each into the
                            durable queue — exactly once, deduped by content hash.
        ↓
~/.atl/queue.db             One embedded bbolt file, per-project buckets keyed
                            by the working directory. No server, no daemon.
        ↓
[session start]             The SessionStart hook surfaces a count:
                            "N learning(s) pending" + a /drain signal.
        ↓
[your first turn]           You invoke /drain. It reads the pending items
                            (atl learnings peek --json), routes each one to the
                            wiki / journal / agent knowledge base, and acks it.
        ↓
atl learnings ack <id>      An acked item is DELETED from the queue.
        ↓
[loop closed]              A processed item is gone — it can never re-report.
                            There is no state file to advance.
```

The split is deliberate: **capture is automatic and deterministic** (markers → queue, exactly once, done by the CLI), and **integration is the LLM half** ([`/drain`](/skills/drain) — deciding where each learning belongs). The only human touch points are:

1. **You (the agent)** invoke [`/drain`](/skills/drain) after seeing the session-start "N pending" signal — one command.
2. **The user** answers an `AskUserQuestion` gate only when `/drain` proposes a *structural* change (a new agent / skill / rule, or an identity expansion). Routine writes to wiki / journal / agent KB happen silently.

## What counts as a learning moment

Any of these, when it happens during a conversation, is a learning moment:

- **Bug fix** — a real bug was reproduced and fixed
- **Decision** — a choice was made between alternatives (JWT vs session, Redis vs memcached, 7d vs 15d refresh)
- **Pattern** — an approach turned out to be clean and reusable
- **Anti-pattern** — something was tried, failed, and we know why
- **Discovery** — a non-obvious fact about the system, library, or external service
- **Convention** — "from now on, we always / never do X"

Routine Q&A, file lookups, and mechanical edits are NOT learning moments. Don't mark every response.

## The marker format

Drop an HTML comment in the response text when a learning moment occurs. Invisible in rendered output, preserved in the transcript the hook scans, ~20 tokens:

```html
<!-- learning: 7-day JWT refresh chosen — we want long sessions; the user logs in about once a week. -->
```

That is the **whole** format:

```
<!-- learning: <one to three sentences, always including the WHY> -->
```

No fields, no schema — just the fact and its reason in plain text. The [`/drain`](/skills/drain) skill reads the payload and infers where it belongs (a wiki topic, a journal entry, or an agent's knowledge base) and derives a kebab-case topic from the content. Multi-line is fine for a longer thought:

```html
<!-- learning:
Redis pool exhausted under load because each request opened its own client.
Fix: one shared pool. Symptom was intermittent timeouts at ~200 rps.
-->
```

**Always include the WHY.** A six-month-old "we chose X" with no reasoning is useless. One marker per learning — don't bundle unrelated learnings; each deserves its own.

> **Changed from v1.** The old marker carried structured YAML fields (`topic`, `kind`, `doc-impact`, `body`). v2 drops all of them: the payload is plain prose, and `/drain` does the routing the fields used to encode. The `doc-impact` field is gone because v2 has no docs-sync step.

### The `profile-fact` channel

The queue is multi-channel. A second channel, `profile-fact`, captures durable facts about the user or the people they work with — same comment shape, `profile-fact:` prefix:

```html
<!-- profile-fact: Prefers TypeScript over JavaScript for all new services. -->
```

[`/drain`](/skills/drain) processes only the `learning` channel; `profile-fact` is reserved for a future first-party profile team's own drain and is not handled here.

## Why inline markers, not a tool call

A tool call per learning would double token cost and slow the conversation. Inline markers are embedded in text the agent was going to produce anyway. A grep-level pass inside [`atl tick`](/cli/tick) finds them at ~0 cost; the AI-heavy [`/drain`](/skills/drain) work only runs when items are queued — boring sessions stay free.

## When to skip marking

- Purely conversational turns (greetings, clarifications, status questions)
- Reading a file and summarizing its contents (no decision, no discovery)
- Routine edits where nothing surprising happened
- A learning already captured by a marker earlier in the same session (don't duplicate)

## Step-by-step under the hood

### 1. `atl tick` captures the markers

[`atl setup-hooks`](/cli/setup-hooks) wires [`atl tick`](/cli/tick) to the `UserPromptSubmit` hook (throttled, e.g. `--throttle=10m`), and `atl session-start` runs a pass at session start. On each run, `tick`:

- discovers this project's Claude Code transcripts modified since the last tick,
- extracts the assistant text and parses `<!-- learning: ... -->` (and `<!-- profile-fact: ... -->`) markers,
- **enqueues each into the durable queue exactly once** — idempotency comes from the queue's content-hash dedup, so re-draining the same text enqueues nothing new.

`tick` only **enqueues**. It never integrates — folding a learning into the knowledge base is LLM work, so it stays on the skill side of the CLI/Skill boundary.

### 2. The durable queue

The queue is one embedded [bbolt](https://github.com/etcd-io/bbolt) file at `~/.atl/queue.db` — no server, no daemon. Every project's queue lives in that one file, isolated into per-project buckets keyed by the working directory. [`atl learnings`](/cli/learnings) is the deterministic read/ack surface:

```bash
atl learnings status                    # pending counts per channel (this project)
atl learnings peek                      # list pending items (human-readable)
atl learnings peek --channel learning --json   # the machine-readable list /drain consumes
atl learnings ack <id>                  # mark an item processed (delete it)
```

### 3. Session start surfaces the count

When you open a new session, the `SessionStart` hook ([`atl session-start`](/cli/setup-hooks)) runs a `tick` pass and reports the pending count — the same number [`atl doctor`](/cli/doctor) reports — as a short signal in Claude's `additionalContext`:

```
🧠 2 learning(s) pending → run /drain
```

When nothing is queued, output is empty (zero token cost).

### 4. `/drain` processes the queue

The agent (you) reads the signal and invokes:

```
/drain
```

The skill:

1. Runs `atl learnings peek --channel learning --json` to read the pending items (`{id, channel, payload, enqueued_at}`).
2. Routes each item by the shape of its payload, deriving a kebab-case topic from the content:
   - **Topic-shaped current truth** → wiki page (`<proj>/.atl/wiki/<topic>.md`, replace/merge) + journal
   - **Time-stamped narrative** → journal only (`<proj>/.atl/journal/<YYYY-MM-DD>.md`, append)
   - **Domain knowledge for an installed agent** → that agent's `children/<topic>.md` + rebuild its `## Knowledge Base` section + journal
   - **Structural** (repeating workflow, crystallized convention, a new domain with no owning agent, an identity expansion) → propose via `AskUserQuestion`; never author autonomously
3. Writes each non-structural item silently, then **acks only after the write succeeds**.
4. For structural items, collects them and proposes each through one `AskUserQuestion` (the reactive-creation boundary — a human confirms structural growth).
5. Reports a short summary of what landed where.

### 5. ack = delete; the loop closes structurally

`atl learnings ack <id>` **deletes** the item from the queue. There is no state file to advance and nothing to dedup against later — a processed marker physically cannot come back.

This is what structurally kills v1's long-session re-report bug class: in v1, reports came from re-scanning an ever-growing transcript filtered against `~/.claude/state/learning-capture-state.json`, and the filter could mis-fire. In v2, reports come from the queue, and processing removes the item. Re-running `/drain` on an empty queue is a no-op.

If `/drain` can't integrate an item, it leaves it un-acked and notes it in the report — failure modes don't lose data.

## When the hook isn't installed

Markers are harmless without the hook — they're HTML comments, invisible in rendered output, inert as text. The capture habit is still valuable (markers are legible even to a human reading the transcript).

For automatic capture, run [`atl setup-hooks`](/cli/setup-hooks). Without it, nothing enqueues automatically; you can still force a capture pass yourself with [`atl tick`](/cli/tick) (no `--throttle`) and then run [`/drain`](/skills/drain). The markers accumulate in transcripts and remain available for whenever a `tick` pass runs.

## History

This flow has gone through three shapes:

1. **Original (pre-`atl`):** "Claude should proactively save learnings at the end of every session." Depended on Claude remembering a prose instruction. Unreliable.
2. **v1 (transcript scan + `/save-learnings`):** Inline markers carried structured YAML fields; a `SessionStart` hook re-scanned the previous session's transcripts, filtered against a JSON state file, and reported unprocessed markers for the `/save-learnings` skill to process. The state file was advanced on success. The model worked, but re-scanning an ever-growing transcript against a filter was the source of a long-session re-report bug class, and the marker schema (`topic`/`kind`/`doc-impact`/`body`) coupled capture to a docs-sync step.
3. **Current (v2 — marker → bbolt queue → `/drain` → ack):** The marker is plain prose. [`atl tick`](/cli/tick) enqueues each one into a durable [bbolt](https://github.com/etcd-io/bbolt) queue exactly once; [`/drain`](/skills/drain) folds each into the knowledge base and acks it (deletes it). No transcript re-scan, no state file, no docs-sync coupling — and the re-report bug class is gone by construction.

## Related

- [`atl tick`](/cli/tick) — the in-session pass that parses markers and enqueues them
- [`atl learnings`](/cli/learnings) — inspect and drain the durable queue (`status` / `peek` / `ack`)
- [`/drain`](/skills/drain) — the LLM half: routes each queued learning into the knowledge base, then acks it
- [`atl setup-hooks`](/cli/setup-hooks) — wires the `UserPromptSubmit` + `SessionStart` hooks that run `tick`
- [`atl doctor`](/cli/doctor) — surfaces the same pending count on demand
- [Knowledge system](/guide/knowledge-system) — where journal and wiki live
- [Children + learnings](/guide/children-and-learnings) — where agent / skill domain knowledge lands
- [Claude Code conventions](/guide/claude-code-conventions) — the marker block conventions used throughout
- Canonical rule: [`core/rules/learning-capture.md`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md)
