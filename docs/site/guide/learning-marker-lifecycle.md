# Learning marker lifecycle

End-to-end picture of how knowledge flows from a conversation into the project's knowledge base. The pattern is **inline markers + scan on next session start** — cheap to write, automatic to process.

The canonical rule lives at [`core/rules/learning-capture.md`](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md). This page is the user-facing summary.

## The flow at a glance

```
[Session N]                                    Claude drops <!-- learning -->
                                               markers inline as it speaks.
                                               No tool call, no extra cost.
        ↓
[Session N ends]                               Markers sit in the transcript.
                                               No hook fires at session end.
        ↓
[Session N+1 starts]                           SessionStart hook fires:
                                               atl session-start --silent-if-clean
        ↓
   step 3: atl learning-capture                Reads ~/.claude/state/
           --previous-transcripts              learning-capture-state.json,
                                               enumerates project transcripts
                                               modified after the cutoff,
                                               grep-scans for marker blocks.
        ↓
[Output reaches Claude]                        🧠 learning-capture: N markers
                                               → Run: /save-learnings ...
        ↓
[Claude's first turn]                          Invokes /save-learnings
                                               --from-markers --transcripts ...
        ↓
[/save-learnings persists]                     Journal entry, wiki page(s),
                                               agent children, skill learnings.
                                               Calls atl learning-capture
                                               --commit-from-transcripts
                                               to advance state file.
        ↓
[Loop closed]                                  Next SessionStart sees
                                               zero unprocessed markers.
```

End-to-end automatic except for **two human touch points**:

1. **You (the agent)** invoke `/save-learnings --from-markers --transcripts ...` after seeing the additionalContext recommendation. Per the maintainer's design, that's a single command call — no manual marker-by-marker review.
2. **The user** answers the AskUserQuestion gate when new structures (skill / rule / agent / identity / skill core change) are proposed. One multi-select prompt per run.

Everything else (journal, wiki, children, learnings, KB rebuilds, state advance) happens silently.

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

Drop an HTML comment in the response text when a learning moment occurs. Invisible in rendered output, preserved in the transcript the hook scans, ~40 tokens:

```html
<!-- learning
topic: auth-refresh
kind: decision
doc-impact: readme
body: 7-day JWT refresh chosen because we want long sessions; user logs in once a week max.
-->
```

### Fields

| Field | Required | Description |
|---|---|---|
| `topic` | ✅ | kebab-case, one concept (becomes the wiki page name). Example: `auth-refresh`, `redis-ttl`, `build-pipeline`. |
| `kind` | ✅ | One of `bug-fix \| decision \| pattern \| anti-pattern \| discovery \| convention`. |
| `doc-impact` | ✅ | One of `none \| readme \| docs \| both \| breaking`. Default `none` when unsure. Drives the [docs-sync rule](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md). |
| `body` | ✅ | One to three sentences. **Always include the WHY.** A 6-month-old "we chose X" without reasoning is useless. |

### Multiple markers per response

Fine when multiple learnings happen. Do **not** bundle unrelated learnings into one marker — each topic deserves its own.

## Why inline markers, not a tool call

A tool call per learning would double token cost and slow the conversation. Inline markers are embedded in text the agent was going to produce anyway. A grep-level hook finds them at ~0 cost; the AI-heavy `/save-learnings` work only runs when markers exist — boring sessions stay free.

## When to skip marking

- Purely conversational turns (greetings, clarifications, status questions)
- Reading a file and summarizing its contents (no decision, no discovery)
- Routine edits where nothing surprising happened
- A learning already captured by a recent marker in the same session (don't duplicate)

## Step-by-step under the hood

### 1. SessionStart hook

The hook ([`atl setup-hooks`](/cli/setup-hooks) installs it) wires:

```
SessionStart → atl session-start --silent-if-clean
```

`atl session-start` runs three boot-time tasks in order:

1. **Auto-update**: pull every cached agentteamland repo
2. **Previous-transcript marker scan**: the step that surfaces unprocessed learnings
3. **`atl` self-version check**: GitHub Releases API, throttled to 24h

### 2. Marker scan

The marker scan inside `atl session-start` is exactly:

```
atl learning-capture --previous-transcripts
```

What it does:

- Reads `~/.claude/state/learning-capture-state.json` for the per-project `lastProcessedAt` cutoff (or last 7 days on first run)
- Enumerates transcripts modified after that cutoff
- Grep-scans for `<!-- learning -->` blocks emitted by **assistant** turns only (the v1.1.1 noise filter rejects prose mentions, tool inputs/outputs, summary events, and topics that fail the kebab-case regex)
- Per-marker hash dedup against the state file's `processedMarkers` set (FIFO-capped at 5000 entries) — this closes the long-session re-report bug fixed in `atl v1.1.3` + `core@1.10.0`
- Prints a compact report

### 3. Output reaches Claude

`SessionStart` and `UserPromptSubmit` are the only Claude Code hooks whose stdout reaches Claude's `additionalContext` (per [Claude Code v2.1.x docs](https://docs.claude.com/en/docs/claude-code/hooks)). The previous v0.2.0 design used `SessionEnd` + `PreCompact` for marker scanning — and silently lost output for ~7 weeks because those events don't deliver to additionalContext.

The current SessionStart-only design fixes that gap. See the [`atl setup-hooks` History note](/cli/setup-hooks#history-from-four-hooks-to-two) for the full migration story.

### 4. `/save-learnings` processes

The agent (you) reads the additionalContext report and invokes:

```
/save-learnings --from-markers --transcripts <path1>,<path2>,...
```

The skill:

1. Extracts each `<!-- learning -->` block from the listed transcripts
2. Hashes by `(kind + topic + body)` and skips anything already in journal for the same date
3. Categorizes each learning by `kind` + body shape
4. Writes journal + wiki + agent children + skill learnings as appropriate
5. (If any of 5 gated changes are proposed) batches one `AskUserQuestion`
6. Calls `atl learning-capture --commit-from-transcripts` to record per-marker hashes in the state file
7. Reports a single summary block

### 5. State advances; loop closes

The state-file write at step 6 is what closes the loop. The next `atl session-start` reads the same state and sees zero unprocessed markers when nothing new has happened. Boring sessions stay free.

If `/save-learnings` fails partway, the state file is **not** updated — markers re-report on the next session and processing retries. Failure modes don't lose data.

## When the hook isn't installed

Markers are harmless without a processing hook — they're HTML comments, invisible in rendered output, inert as text. The capture habit is still valuable (markers are legible even to a human reader of the transcript).

For automatic processing, the user runs `atl setup-hooks`. Without those hooks, the user must invoke `/save-learnings` manually at session boundaries; markers still accumulate in transcripts and remain available for whenever processing happens.

## History

This rule has gone through three shapes:

1. **Original (pre-`atl` versions):** "Claude should proactively save learnings at the end of every session." Worked sometimes; depended on Claude remembering a prose instruction. Unreliable.
2. **First `atl` version (v0.2.0 — `core@1.3.0`):** Inline markers + `atl learning-capture` registered on `SessionEnd` and `PreCompact` hooks. **Silently broken** — those events don't deliver hook stdout to Claude's additionalContext. 324 markers across 9 sessions in the maintainer workspace produced **zero** auto-processing during the month it was in production. All actual `/save-learnings` work in that period was triggered by manual user invocation, not hook output.
3. **Current (v1.1.0+ — `core@1.8.0`):** Hook moved to `SessionStart` via the new `atl session-start` wrapper, scanning the previous session's transcripts via the new `--previous-transcripts` mode. Output reaches additionalContext. State file tracks per-project `lastProcessedAt` and per-marker hashes (the latter shipped in `atl v1.1.3` + `core@1.10.0` to fix a long-session re-report bug). The loop closes deterministically.

## Related

- [`atl learning-capture`](/cli/learning-capture) — the CLI scanner
- [`atl setup-hooks`](/cli/setup-hooks) — wires the SessionStart hook
- [`/save-learnings`](/skills/save-learnings) — processes the markers
- [Knowledge system](/guide/knowledge-system) — where journal and wiki live
- [Children + learnings](/guide/children-and-learnings) — where agent / skill domain knowledge lands
- [Claude Code conventions](/guide/claude-code-conventions) — the marker block conventions used throughout
- Canonical rule: [`core/rules/learning-capture.md`](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md)
