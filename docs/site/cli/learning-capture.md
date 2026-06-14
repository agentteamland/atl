# `atl learning-capture`

Scan Claude Code transcripts for inline `<!-- learning ... -->` markers dropped by Claude during prior conversations, and print a short report that the next session's first turn can act on.

Driven by the `SessionStart` hook installed by [`atl setup-hooks`](/cli/setup-hooks) — wrapped inside the `atl session-start` composite command. Can also be invoked manually for testing or ad-hoc scans.

Requires `atl` ≥ 1.1.0.

## Why this exists

Without an automated capture step, two kinds of knowledge always slip away:

1. **Learnings go un-saved.** Users forget to run `/save-learnings`, and the agent doesn't always volunteer. Decisions, bug fixes, and discoveries vanish with the conversation.
2. **Docs go stale.** When a feature ships or a behavior changes, the README / doc site often lag by days or weeks — or forever.

`atl learning-capture` + two paired core rules ([learning-capture](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) + [docs-sync](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md)) close both gaps:

- Claude drops an inline `<!-- learning -->` marker each time a real learning moment happens during the conversation. Markers are ~50 tokens, HTML-commented (invisible in rendered output), and free to ignore when a session has nothing interesting in it.
- At the *next* session's start, the harness runs `atl session-start` which calls `atl learning-capture --previous-transcripts`. It scans every project transcript modified since the last successful `/save-learnings` run, finds markers, and prints a short report on stdout — which `SessionStart` correctly delivers to Claude's `additionalContext`.
- Claude's first turn invokes `/save-learnings --from-markers --transcripts <paths>`, which processes the markers into journal + wiki + agent children + skill learnings — AND prepares draft README / doc-site changes for any marker with `doc-impact` set.

Nothing pushes to public repos automatically. You or Claude review the drafts.

## Modes

| Mode | Invocation | When |
|---|---|---|
| **Previous-transcripts** (recommended) | `atl learning-capture --previous-transcripts` | Used by `atl session-start` on `SessionStart` hook fire. Multi-transcript scan, state-file-driven cutoff. |
| **Single-transcript** (legacy) | `atl learning-capture --transcript-path <path>` | Manual scan of one specific transcript file. |
| **Stdin payload** (legacy) | `atl learning-capture < hook-stdin.json` | Reads `transcript_path` from a Claude Code hook's stdin JSON payload. Kept for compatibility with the v0.2.0 SessionEnd / PreCompact registrations (those events never delivered stdout to Claude — see [setup-hooks history](/cli/setup-hooks#history-from-four-hooks-to-two) — but the binary still accepts the call). |

## Previous-transcripts mode

This is what `atl session-start` calls. State is tracked at `~/.claude/state/learning-capture-state.json`:

```json
{
  "projects": {
    "-Users-you-projects-my-app": {
      "lastProcessedAt": "2026-05-02T14:00:31Z"
    }
  }
}
```

The slug is the cwd path with `/` replaced by `-`. On every run, the command:

1. Reads the slug's `lastProcessedAt` from the state file (or defaults to "7 days ago" on first use).
2. Lists every transcript file under `~/.claude/projects/{slug}/*.jsonl` modified after that cutoff.
3. Scans each for `<!-- learning -->` blocks (with the v1.1.1 noise filter — see below).
4. Prints a single consolidated report on stdout when markers are found; silent when not.

The state file is written by `/save-learnings` after a successful run. The CLI never writes it directly — that prevents partial-write corruption from a crashed scan.

## Marker format

Markers are inline HTML comments containing loose YAML fields:

```
<!-- learning
topic: auth-refresh
kind: decision
doc-impact: readme
body: 7-day JWT refresh chosen because we want long sessions; user logs in once a week max.
-->
```

| Field | Required | Allowed values | Purpose |
|---|---|---|---|
| `topic` | yes | kebab-case (lowercase letters / digits, hyphens or dots as separators) | Becomes the wiki / children / learnings page name |
| `kind` | yes | `bug-fix`, `decision`, `pattern`, `anti-pattern`, `discovery`, `convention` | Categorizes the learning |
| `doc-impact` | no | `none`, `readme`, `docs`, `both`, `breaking` | Triggers doc-sync drafts when ≠ `none` |
| `body` | yes | one or more sentences | The actual learning content — ALWAYS include the WHY |

HTML comments are invisible in rendered markdown output, so markers don't pollute the conversation UI; they live in the transcript where only the scanner sees them.

## v1.1.1 noise filter

Earlier scanner versions matched the bare substring `<!-- learning` anywhere in a transcript. Sessions that *discussed* the marker format — rule rewrites, skill rewrites, brainstorms about learning capture — inflated the next session's count by 10-25× because their tool-output text echoed every marker fragment they read. The v1.1.1 filter rejects:

- Non-assistant turns (tool-use input, tool-result output, user messages, summary events). Only `message.role == "assistant"` text content is scanned.
- Markers without a kebab-case topic. The regex is `^[a-z0-9]+([-.][a-z0-9]+)*$` — rejects uppercase, spaces, ellipsis placeholders (`topic: ... doc-impact ...`), and the literal field-spec string `bug-fix | decision | pattern | ...`.

Validation sweep: 149 raw substring hits across 5 workspace transcripts → 16 real markers after filter. The remaining 133 were format documentation, prose mentions, and tool quoting.

## Output

### Empty (no transcripts modified, or no markers found)

```
(silent when --silent-if-empty is passed — used by the SessionStart hook path)
```

Without `--silent-if-empty`:

```
📝 learning-capture: scanned 3 transcripts, no markers found
```

### Markers found

```
🧠 learning-capture: 7 unprocessed markers across 2 transcripts
  by kind: 3 decision, 2 pattern, 1 discovery, 1 bug-fix
  3 markers require doc drafts (README / doc site) — see docs-sync rule

→ Run: /save-learnings --from-markers --transcripts <path1>,<path2>
```

`SessionStart` injects this on stdout into Claude's `additionalContext`. The `learning-capture` core rule instructs Claude to invoke the named `/save-learnings` command on the first turn, which processes the markers into:

- **journal/{date}\_{agent}.md** entries (chronological per-agent record)
- **wiki/{topic}.md** updates (current truth, replace-style)
- **agents/{agent}/children/{topic}.md** auto-grown content (with `knowledge-base-summary` frontmatter; agent.md Knowledge Base section is auto-rebuilt)
- **skills/{skill}/learnings/{topic}.md** auto-grown content (skill.md Accumulated Learnings section is auto-rebuilt)
- **doc drafts** for `doc-impact` markers (presented for review, not auto-pushed)

After successful processing, `/save-learnings` advances the state file's `lastProcessedAt` so the same markers won't re-report on the next `SessionStart`.

## Flags

| Flag | Default | Purpose |
|---|---|---|
| `--previous-transcripts` | (off) | Multi-transcript scan driven by the state file (used by `atl session-start`) |
| `--silent-if-empty` | `false` | Produce no output when no markers are found (for hooks) |
| `--transcript-path <path>` | (from stdin JSON) | Explicit single-file scan; bypasses both the state file and stdin payload |
| `--help` | — | Show command help |

## Cost model

| Scenario | Token cost to Claude | Time cost |
|---|---|---|
| `--previous-transcripts` with no transcripts modified | 0 | <1ms (one stat call) |
| `--previous-transcripts` with N transcripts, no markers | 0 (silent-if-empty) | ~10ms per MB of transcript |
| `--previous-transcripts` with N markers | ~80 tokens for the report | ~10ms per MB of transcript |
| `/save-learnings --from-markers` (processing) | proportional to marker count, not transcript size | seconds |

The design deliberately makes boring sessions free. Only real learnings cost anything, and the cost scales with what was actually learned — never with conversation length.

## Manual testing

```bash
# Create a synthetic transcript with one assistant marker (note the role+content shape)
cat > /tmp/test.jsonl <<'EOF'
{"message":{"role":"assistant","content":[{"type":"text","text":"<!-- learning\ntopic: my-topic\nkind: decision\ndoc-impact: none\nbody: test.\n-->"}]}}
EOF

# Scan it
atl learning-capture --transcript-path /tmp/test.jsonl
```

Expected output:

```
🧠 learning-capture: 1 unprocessed marker across 1 transcript
  by kind: 1 decision

→ Run: /save-learnings --from-markers --transcripts /tmp/test.jsonl
```

## History

`atl v0.2.0` (2026-04-24) introduced the marker protocol with `SessionEnd` and `PreCompact` hooks. Per Claude Code v2.1.x, those hooks do NOT deliver stdout to Claude's `additionalContext`. 324 markers across 9 maintainer-workspace sessions in the month after v0.2.0 produced **zero** auto-processing — every actual `/save-learnings` run came from manual invocation. The capture binary worked; the trigger path was wrong.

`atl v1.1.0` (2026-05-02) introduced the `--previous-transcripts` mode, the `~/.claude/state/learning-capture-state.json` state file, and the `atl session-start` composite wrapper that calls this command from the `SessionStart` hook (the *only* hook event that reliably delivers stdout to Claude). Marker protocol unchanged.

`atl v1.1.1` (2026-05-02) added the noise filter (assistant-role + kebab-case topic).

## Related

- [`atl setup-hooks`](/cli/setup-hooks) — installs the `SessionStart` hook that drives this command via `atl session-start`
- [`atl update`](/cli/update) — the other piece called by `atl session-start`
- [learning-capture rule](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) — behavior spec for when Claude should drop markers
- [docs-sync rule](https://github.com/agentteamland/core/blob/main/rules/docs-sync.md) — paired rule for proactive doc updates (uses `doc-impact` field)
- [`/save-learnings` skill](https://github.com/agentteamland/core/blob/main/skills/save-learnings/skill.md) — processing half of the system
