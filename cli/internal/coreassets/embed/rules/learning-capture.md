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

### Other capture channels

`learning` is the platform's **own** channel: this rule and the `/drain` skill both ship with core, so it exists on every machine. Other channels exist only when an installed team **declares** one — and a declaring team ships everything that channel needs: its marker shape, its drain skill, and the rule that acts on its signal. Core emits the signal and nothing more: it holds no instruction for any channel but its own, only the words a declaration hands it.

Every channel follows the shape described here — the same hidden-comment marker (with that channel's own prefix), the same durable queue, the same auto-drain response. What differs is owned elsewhere: read the owning team's rule for that channel's marker format and its drain. With no such team installed there is no other channel, and no signal for one.

If you mistype a channel prefix the marker is **not** captured. When the typo is close to an active channel, `atl tick` reports it (`… did you mean "<channel>"?`) — but on its next *capture* pass, which the hook throttles to roughly ten minutes, so not seeing that line on the following turn is no evidence the marker was fine. Re-mark it with the correct prefix; nothing was queued.

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

The **capture-watchdog signal** — `atl: capture-watchdog (learning) — no learning markers for N assistant turn(s) …` — is the second trigger for the **same** response, covering the opposite failure: not a full queue, but a marker-less dry stretch (markers you may have forgotten to write — the pipeline's one non-deterministic link, now detected deterministically). On seeing it: quickly review the flagged turns and mark anything durable you missed (visible line + hidden marker, as always), and spawn the same ONE background drain subagent — **an empty queue is valid on this trigger**; the drain's mining step sweeps the stretch and enqueues what it finds. Same single-in-flight discipline. If the stretch was genuinely routine (mechanical edits, status chatter), there is nothing to mark and nothing to mine — that's a correct outcome, not a failure.

The watchdog measures its dry stretch **per channel**, and each signal names the channel that went dry, the drain to spawn for it, and the rule that owns the response. The `(learning)` variant is the one this rule owns; a signal naming any other channel belongs to the team that declared it — follow that channel's own rule, exactly as the auto-drain signals split. Several can appear on the same turn — they are different subagents running different skills, and a stretch that captured none of those kinds genuinely needs each recovery. (Per channel is load-bearing rather than cosmetic: a single shared predicate let a marker on any one channel reset the dry stretch every other channel was accumulating, so a session capturing learnings regularly could never trip the watchdog for a quieter channel.)
- **Don't ask, don't wait for the user, don't run `/drain` inline** in your own context — keep your main turn for the user's request; the background subagent does the integration alongside it.
- **No completion log needed** — the visible `📝 Learned:` line already showed the user what was learned; the routing detail lives in the wiki/journal.
- **Fallback:** if this harness has no background-subagent capability, run the `/drain` procedure yourself, concisely, at the end of your turn.

## Why inline markers, not a tool call?

A tool call per learning would double token cost and slow the conversation. Inline marks ride in text you were already producing. A hook finds the hidden markers at ~0 cost; the AI-heavy drain only runs when the queue is non-empty or the capture-watchdog flags a substantive marker-less stretch — boring sessions stay free (a routine dry stretch never crosses the watchdog's thresholds twice-over: it needs BOTH ≥2 assistant turns AND ≥1000 chars of user input, and fires once per stretch).

## When to skip

- Purely conversational turns (greetings, clarifications, status questions)
- Reading a file and summarizing it (no decision, no discovery)
- Routine edits where nothing surprising happened
- A learning already marked earlier in the same session
