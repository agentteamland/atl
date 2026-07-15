# Learning capture (inline marker protocol)

## Who runs this

**You (the agent) mark learnings inline as you speak.** ATL's automation does the rest — `atl tick` (a hook, every turn) transfers your markers into a durable queue exactly once, and when the queue is non-empty it signals you to **drain it in the background automatically**. You never run `/drain` by hand and you never track state — capture and integration are both automatic. You do exactly two things: mark a learning when one happens, and spawn a background drain when the queue signals.

Markers are the "save it if you see it" mechanism. Cheap to drop, free to ignore when nothing interesting happened.

## What counts as a learning moment

- **Bug fix** — a real bug reproduced and fixed
- **Decision** — a choice made between alternatives (and why)
- **Pattern** — an approach that turned out clean and reusable
- **Anti-pattern** — something tried, failed, and the reason it failed
- **Discovery** — a non-obvious fact about the system, a library, an external service
- **Convention** — "from now on we always / never do X"

Routine Q&A, file lookups, and mechanical edits are NOT learning moments. Don't mark every response.

## How to mark

When a learning moment occurs, do **both** — a visible line so the user sees what was learned, and a hidden marker that the pipeline captures:

```
📝 Learned: 7-day JWT refresh chosen — we want long sessions; the user logs in ~weekly.
<!-- learning: 7-day JWT refresh chosen — we want long sessions; the user logs in about once a week. -->
```

- The **visible line** (`📝 Learned: …`) renders in the chat — it's how the user sees, in the moment, what you picked up. Keep it to one short sentence.
- The **hidden marker** (`<!-- learning: … -->`, an HTML comment invisible in rendered output) is what `atl tick` parses into the queue. Same fact as the visible line, always **including the WHY**.

Marker format: `<!-- learning: <one to three sentences, always including the WHY> -->` — no fields, no schema, plain text. Multi-line is fine for a longer thought:

```
<!-- learning:
Redis pool exhausted under load because each request opened its own client.
Fix: one shared pool. Symptom was intermittent timeouts at ~200 rps.
-->
```

**Always include the WHY.** A six-month-old "we chose X" with no reasoning is useless. One learning per mark — don't bundle unrelated learnings; each deserves its own visible line + marker.

### The `profile-fact` channel

A second channel, `profile-fact`, captures durable facts about the entities in the user's world (people, orgs, and more), for the profile layer — a shipped first-party team (`profile-team`). Same hidden-comment shape, `profile-fact:` prefix; the exact marker format is owned by profile-team's own `profile-capture` rule, which installs with the team. That channel auto-drains the same way this one does — a background `/profile-drain` subagent — but the action instruction lives in the `profile-capture` rule: `atl tick` emits the auto-drain signal for both channels, while the rule that acts on each ships with its owner, so a session without profile-team simply never acts on the `profile-fact` signal.

## What happens after — the automatic loop

```
[you mark a learning: a visible 📝 line + a hidden <!-- learning --> marker]   ← the only thing you do by hand
        ↓
atl tick (UserPromptSubmit hook, every turn) + session-start
        → parses hidden markers from the transcript → enqueues into the durable queue, exactly once
        → reads the queue count; if non-empty, prints an AUTO-DRAIN signal into your context
        ↓
you, on seeing that signal → spawn ONE background drain subagent (session auth — no separate claude -p)
        ↓
the subagent runs /drain: peek → route each (wiki / journal / agent KB) → ack (delete)
        ↓
processed items are DELETED from the queue — they can never re-report
```

Capture is automatic and deterministic (markers → queue, exactly once). **Integration is now automatic too:** the hook signals every turn the queue is non-empty, and you drain it in the background — no one runs `/drain` by hand. The queue is the durable safety net — if a background drain fails or you miss a turn, the items survive and the next turn's signal retries them, so **nothing is ever lost**; the worst case is a learning integrated a turn later.

## Auto-drain — what you do when the queue signals

When a hook prints `atl: N learning(s) pending — auto-drain …` into your context:

- **Spawn ONE background drain subagent** (the Agent tool, run in the background) that runs the `/drain` procedure. It inherits this session's auth, so there is no separate `claude -p` and no auth problem.
- **Single-in-flight:** if a drain subagent you already spawned is still running, do **not** spawn another — the signal keeps appearing until the queue drains, and the running one will clear it. Wait for its completion, then it's done.
- **Don't ask, don't wait for the user, don't run `/drain` inline** in your own context — keep your main turn for the user's request; the background subagent does the integration alongside it.
- **No completion log needed** — the visible `📝 Learned:` line already showed the user what was learned; the routing detail lives in the wiki/journal.
- **Fallback:** if this harness has no background-subagent capability, run the `/drain` procedure yourself, concisely, at the end of your turn.

## Why inline markers, not a tool call?

A tool call per learning would double token cost and slow the conversation. Inline marks ride in text you were already producing. A hook finds the hidden markers at ~0 cost; the AI-heavy drain only runs when the queue is non-empty — boring sessions stay free.

## When to skip

- Purely conversational turns (greetings, clarifications, status questions)
- Reading a file and summarizing it (no decision, no discovery)
- Routine edits where nothing surprising happened
- A learning already marked earlier in the same session
