# `/save-learnings`

Persist what was learned in a conversation (or in marker-tagged regions of multiple transcripts) to journal, wiki, agent `children/`, and skill `learnings/`. Then update the state file so the next session does not re-process the same markers.

This skill is the **processing half** of the auto-trigger learning loop:

```
[atl session-start hook] → reports unprocessed markers
        ↓
[Claude on the next turn] → invokes /save-learnings --from-markers --transcripts ...
        ↓
[skill] → markers land in journal / wiki / agent children / skill learnings
        ↓
[skill closes the loop] → atl learning-capture --commit-from-transcripts
```

Ships as a global skill in [core](https://github.com/agentteamland/core) since `core@1.0.0`. Rewritten in `core@1.8.0` to support the SessionStart-driven hook flow + automatic `children/` + `learnings/` growth.

## Three invocation modes

| Mode | Invocation | When to use |
|---|---|---|
| **Hook mode** (auto-trigger) | `/save-learnings --from-markers --transcripts a.jsonl,b.jsonl` | Most common path. `atl session-start` reports markers; Claude calls this on the next turn. |
| **Single-transcript mode** | `/save-learnings --from-markers` | Legacy: scans only the current session's own transcript. Rarely needed since the hook flow shipped. |
| **Manual mode** | `/save-learnings [agent-name]` | User invokes explicitly; no markers required. Analyzes the live conversation. |

## What it writes

Most writes happen silently — the user reads the final summary, not a prompt per write. Five exceptions go through `AskUserQuestion` (batched into ONE prompt per run):

1. **New skill creation** (a workflow pattern repeated 2+ times)
2. **New rule creation** (an unambiguous "always X" / "never Y" convention crystallized)
3. **New agent creation** (a domain area is unmistakably a separate agent)
4. **Existing agent identity change** (responsibility / principles need to shift)
5. **Existing skill core change** (a skill's steps need to change)

Everything else — journal, wiki, agent `children/`, skill `learnings/`, Knowledge Base + Accumulated Learnings auto-rebuilds — happens without prompting.

### Per-learning destination matrix

Each learning's *shape* determines where it lands. The categorization happens automatically from the marker's `kind` field (or, in manual mode, by inspecting the conversation):

| Learning shape | Destination |
|---|---|
| Time-stamped narrative ("we tried X, then Y, then Z worked") | Journal entry only |
| Topic-shaped current truth ("the right way to do auth is …") | Wiki page (replace if exists) + journal entry |
| Domain knowledge for a specific agent | Agent's `children/{topic}.md` + journal entry |
| Domain knowledge for a specific skill | Skill's `learnings/{topic}.md` + journal entry |
| Repeating workflow (2+ instances) | **AskUserQuestion → new skill** |
| Convention crystallized ("never X", "always Y") | **AskUserQuestion → new rule** via `/rule` |
| Domain area without an owning agent | **AskUserQuestion → new agent** |
| Existing agent's identity expanded | **AskUserQuestion → agent.md core update** |
| Existing skill's core flow needs change | **AskUserQuestion → skill.md core update** |

## What it touches

| Surface | What changes | Format |
|---|---|---|
| `.atl/journal/{YYYY-MM-DD}_{agent}.md` | One entry per agent per date. Append-on-existing, dedup by hash. | Frontmatter (`date`, `agent`, `tags`) + `## Summary` + `## Learnings` + `## Auto-Created` + `## User-Approved Structural Changes` + `## Notes for Other Agents`. |
| `.atl/wiki/{topic}.md` | Replace-style update for current truth. New page from template if topic is fresh. | Standard wiki page format (Last updated / Current state / Sources). |
| `CLAUDE.md` | Rebuilds the `<!-- wiki:index -->` marker block from the current set of wiki pages. | Sorted by filename; one-line summary per page. |
| Agent `children/{topic}.md` | Append-or-create with required `knowledge-base-summary` frontmatter. | Frontmatter + body. |
| Agent `agent.md` Knowledge Base section | Auto-rebuilt from each child file's `knowledge-base-summary` frontmatter. Hand-edits to this section are overwritten. | Same shape as the auto-rebuilt block — see [agent-structure rule](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md). |
| Skill `learnings/{topic}.md` | Mirror of agent children/ pattern — same frontmatter contract. | Frontmatter + body. |
| Skill `skill.md` Accumulated Learnings section | Auto-rebuilt from each learnings file's frontmatter. | Same shape as the agent KB. |

After every successful run the skill calls `atl learning-capture --commit-from-transcripts` to record per-marker hashes in `~/.claude/state/learning-capture-state.json` (FIFO-capped at 5000). This is what closes the loop — the next [`atl session-start`](/cli/setup-hooks) reports zero markers when nothing is new.

## Hook-mode example

A new session opens. `atl session-start` runs as the SessionStart hook, sees 5 unprocessed markers in the previous session's transcripts, and prints (into Claude's `additionalContext`):

```
🧠 learning-capture: 5 unprocessed markers across 4 transcripts
  by kind: 1 convention, 1 decision, 2 discovery, 1 pattern

→ Run: /save-learnings --from-markers --transcripts <path1>,<path2>,<path3>,<path4>
```

Claude reads that, invokes the skill verbatim, and the skill:

1. Extracts each `<!-- learning -->` block from the transcripts
2. Hashes by `(topic, body)` and skips anything already in journal for the same date
3. Categorizes each learning by `kind` + body shape
4. Writes journal + wiki + children + learnings as appropriate
5. (If any of the 5 gated changes are proposed) batches one `AskUserQuestion`
6. Runs `atl learning-capture --commit-from-transcripts` to advance the state file
7. Reports a single summary block

Boring sessions (no markers) cost zero tokens — the hook prints nothing, the skill is never invoked.

## Manual-mode example

```
/save-learnings api-agent
```

Used at the end of a hands-on coding conversation when no inline markers were dropped. The skill scans the live conversation for learnings (patterns, conventions, decisions, discoveries) and applies the same destination matrix as hook mode. The state file is **not** advanced — manual mode is not bound to the marker-state-file contract.

## Marker format

Inline markers are HTML comments embedded in assistant turns. Invisible in rendered output, preserved in the transcript, ~40 tokens each:

```html
<!-- learning
topic: auth-refresh
kind: decision
doc-impact: readme
body: 7-day JWT refresh chosen because we want long sessions; user logs in once a week max.
-->
```

Required fields:

- `topic` — kebab-case, one concept (becomes wiki page name)
- `kind` — one of `bug-fix | decision | pattern | anti-pattern | discovery | convention`
- `doc-impact` — one of `none | readme | docs | both | breaking` (default `none`)
- `body` — 1-3 sentences. **Always include the WHY.**

See the [learning-capture rule](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md) for the full marker spec and the [`atl learning-capture`](/cli/learning-capture) page for the scanner that surfaces them.

## Important rules

1. **No confirmation for non-structural writes.** Memory / journal / wiki / agent children / skill learnings all happen silently.
2. **AskUserQuestion ONLY for new structures or identity changes.** New skill, new rule, new agent, agent identity, skill core.
3. **State file write is the closing bracket.** Until the state file updates, markers remain unprocessed and re-report on next SessionStart.
4. **Sensitive information filter.** Passwords, tokens, API keys are NEVER written to journal / wiki / team repos. Suspected credentials are redacted.
5. **Idempotent everywhere.** Re-running on the same markers produces no incremental change (dedup by hash, replace-with-same is a no-op, KB rebuild is deterministic).
6. **Team repo push is automatic for maintainers, fails gracefully for users.** Users without push permission keep changes locally; the upstream-contribution flow eventually packages them as a PR.
7. **Approval-gate batching.** Multiple structural changes appear in **one** `AskUserQuestion` with multi-select, not N separate prompts.
8. **Skill creation threshold = 2 instances.** Don't auto-propose a new skill on a single workflow occurrence.
9. **Rule creation criteria = unambiguous "always X" / "never Y" wording.** Hedged language goes to wiki, not rule.

## Related

- [`atl learning-capture`](/cli/learning-capture) — the CLI scanner that surfaces markers and commits hashes to state
- [`atl setup-hooks`](/cli/setup-hooks) — wires the SessionStart hook that triggers this skill
- [Concepts: Skill](/guide/concepts#skill) — the `learnings/` pattern this skill maintains
- [`/wiki`](/skills/wiki) — companion knowledge-base skill (this one writes wiki pages; `/wiki` queries / lints them)

## Source

- Spec: [core/skills/save-learnings/skill.md](https://github.com/agentteamland/core/blob/main/skills/save-learnings/skill.md)
- Rule: [core/rules/learning-capture.md](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md)
- Rule: [core/rules/agent-structure.md](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md)
