# Learning capture (inline marker protocol)

## Who runs this

**You (the agent) drop markers inline as you speak.** A marker is a silent HTML comment — invisible in rendered output, preserved in the transcript. ATL's automation does the rest: `atl tick` (run by a hook every few minutes and at session start) transfers your markers into a durable queue exactly once, and the `/drain` skill folds each into the knowledge base. You never track state or re-scan — capture is fire-and-forget.

Markers are the "save it if you see it" mechanism. Cheap to drop (~20 tokens), free to ignore when nothing interesting happened.

## What counts as a learning moment

- **Bug fix** — a real bug reproduced and fixed
- **Decision** — a choice made between alternatives (and why)
- **Pattern** — an approach that turned out clean and reusable
- **Anti-pattern** — something tried, failed, and the reason it failed
- **Discovery** — a non-obvious fact about the system, a library, an external service
- **Convention** — "from now on we always / never do X"

Routine Q&A, file lookups, and mechanical edits are NOT learning moments. Don't mark every response.

## How to mark

Drop an HTML comment when a learning moment occurs:

```
<!-- learning: 7-day JWT refresh chosen — we want long sessions; the user logs in about once a week. -->
```

That is the whole format: `<!-- learning: <one to three sentences, always including the WHY> -->`. No fields, no schema — just the fact and its reason in plain text. The `/drain` skill reads the payload and decides where it belongs (a wiki topic, a journal entry, or an agent's knowledge base).

Multi-line is fine for a longer thought:

```
<!-- learning:
Redis pool exhausted under load because each request opened its own client.
Fix: one shared pool. Symptom was intermittent timeouts at ~200 rps.
-->
```

**Always include the WHY.** A six-month-old "we chose X" with no reasoning is useless. One marker per learning — don't bundle unrelated learnings; each deserves its own.

### The `profile-fact` channel

A second channel, `profile-fact`, captures durable facts about the user or the people they work with, for the profile layer (a future first-party team). Same comment shape, `profile-fact:` prefix. `/drain` processes only the `learning` channel; `profile-fact` is handled by the profile team's own drain when installed.

## What happens after — the automatic loop

```
[you drop a marker mid-conversation]          ← the only thing you do
        ↓
atl tick (hook-run, every few min + session start)
        → parses markers from the transcript
        → enqueues each into the durable queue (~/.atl/queue.db), exactly once
        ↓
session start surfaces "N learning(s) pending" + a /drain signal
        ↓
/drain skill: peek → route each (wiki / journal / agent KB) → ack (delete)
        ↓
processed items are DELETED from the queue — they can never re-report
```

Capture is automatic and deterministic (markers → queue, exactly once); integration is the LLM half (`/drain`). The old re-scan-and-filter model — re-reading transcripts each session and filtering against a state file — is gone, and with it the re-report bug class: a processed marker physically cannot come back.

## Why inline markers, not a tool call?

A tool call per learning would double token cost and slow the conversation. Inline markers ride in text you were already producing. A hook finds them at ~0 cost; the AI-heavy `/drain` work only runs when markers exist — boring sessions stay free.

## When to skip

- Purely conversational turns (greetings, clarifications, status questions)
- Reading a file and summarizing it (no decision, no discovery)
- Routine edits where nothing surprising happened
- A learning already captured by a marker earlier in the same session
