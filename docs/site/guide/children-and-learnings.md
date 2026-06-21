# Children + learnings

The shape that complex agents and skills use for accumulated domain knowledge: a short top-level file plus a directory of topic-per-file detail pages, each carrying a one-line `knowledge-base-summary` frontmatter that auto-rebuilds the parent file's index section.

Same pattern, two names — **`children/`** for agents, **`learnings/`** for skills. Single mental model across both.

The canonical rule lives at [`core/rules/agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md). This page is the user-facing summary.

## Why this pattern exists

Without it, complex agents and skills end up as one of two anti-shapes:

1. **Monolithic files** — everything piled into one `agent.md` or `skill.md`. Hard to update one piece without touching the rest. Diffs are noisy. Re-reads burn tokens.
2. **Hand-curated index sections** — a separate `agent.md` table of contents that a human maintains alongside the topic files. Drifts the moment someone forgets to update it.

The children + learnings pattern resolves both:

- **Topic-per-file** — update one piece without touching others
- **Auto-rebuilt index** — the top-level file's "Knowledge Base" / "Accumulated Learnings" section is regenerated from frontmatter on every [`/save-learnings`](/skills/drain) run. Hand edits are overwritten — the source of truth is each child's frontmatter.

Result: knowledge accumulates frictionlessly, the top-level file stays tight, and the index never goes stale.

## Children — for agents

Every complex agent is organized as:

```
~/.claude/repos/agentteamland/{team}/agents/{agent-name}/
├── agent.md              ← Identity, area of responsibility, core principles (short, embedded)
└── children/             ← Detailed information, patterns, strategies (each topic in a separate file)
    ├── topic-1.md
    ├── topic-2.md
    └── ...
```

After [`atl install`](/cli/install), the same structure is copied into your project at `.claude/agents/{agent-name}/`.

### Rules

1. **`agent.md` stays short.** Only: identity, area of responsibility (positive list), core principles (unchanging, short bullets), Knowledge Base section (auto-aggregated), "read children/" instruction.
2. **Everything detailed goes under `children/`.** Strategies, patterns, workflows, conventions — each in a separate `.md` file.
3. **New topic = new file.** Without touching `agent.md` by hand, add a `.md` file under `children/`. The Knowledge Base section is rebuilt automatically by `/save-learnings` from each child file's frontmatter.
4. **Update = single file.** To update a topic, only the relevant `children/` file is touched.
5. **Monolithic agent files are prohibited.**
6. **This pattern applies to all agents.** API, Socket, Worker, Flutter, React, Mail, Log, Infra — all follow the same structure.

## Learnings — for skills

Every complex skill mirrors the agent shape. Two locations matter:

```
# Source-of-truth (the team / core repo, served by atl):
~/.claude/repos/agentteamland/{team-or-core}/skills/{skill-name}/
├── skill.md              ← The skill's procedure (steps, identity, flow). Stays short.
└── learnings/            ← Accumulated edge cases, successful patterns, anti-patterns
    ├── topic-1.md
    ├── topic-2.md
    └── ...

# Project-local copy (post-atl-v1.0.0 install topology):
{project}/.claude/skills/{skill-name}/
├── skill.md              ← Identical copy, refreshed by `atl update` if unmodified
└── learnings/            ← Same pattern; auto-grown copies
```

[`atl install`](/cli/install) copies skills (and agents + rules) into the project. [`atl update`](/cli/update) refreshes unmodified copies via three-way SHA-256 comparison. `/save-learnings` writes to the project-local copy first; the auto-update flow propagates changes upstream when the user is the team-repo maintainer.

Same shape, same rules, same `knowledge-base-summary` frontmatter convention. The skill's `skill.md` ships with an "Accumulated Learnings" section auto-aggregated from `learnings/*.md` frontmatter — same mechanism as `agent.md`'s Knowledge Base.

**Why mirror agents on skills?** The "self-improving skill" framing benefits from a structured place for accumulated wisdom that agents (Claude) can see when invoking the skill. Without `learnings/`, every skill use starts from zero on edge cases that came up in prior runs.

## The frontmatter contract

Every `children/*.md` and `learnings/*.md` file MUST carry a `knowledge-base-summary` frontmatter field:

```markdown
---
knowledge-base-summary: "<one-to-three-line summary used in the auto-rebuilt index section>"
---

# <Topic Title>

<the actual content — patterns, strategies, examples — as long as needed>
```

This summary is what feeds the parent file's Knowledge Base / Accumulated Learnings section. Without it, `/save-learnings` either skips the topic in the rebuild OR (for new files it created itself) writes the field with a generated summary; in both cases the file should have one.

## Auto-rebuilt index sections

When `/save-learnings` runs, it rebuilds the parent file's index section from the frontmatter of every `children/*.md` (for agents) or `learnings/*.md` (for skills). The shape is identical for both:

```markdown
## Knowledge Base                     ← (or "Accumulated Learnings" for skills)

### <Topic 1 (heading-cased from filename)>
<knowledge-base-summary>
→ [Details](children/topic-1.md)     ← (or learnings/topic-1.md for skills)

### <Topic 2>
<knowledge-base-summary>
→ [Details](children/topic-2.md)

...
```

Hand edits to this section are **overwritten** on the next `/save-learnings` run — the source of truth is each child file's frontmatter. The rest of `agent.md` / `skill.md` (identity, responsibility, principles, flow) is **not touched** by the rebuild.

## Three layers of update

The split lets "knowledge accumulates" be automatic and frictionless, while protecting the top-level file's identity from drift:

| Layer | What changes | How |
|---|---|---|
| **A — auto** | A `children/{topic}.md` or `learnings/{topic}.md` file is created or updated. | `/save-learnings` writes it directly. No prompt. |
| **B — auto** | The parent's Knowledge Base / Accumulated Learnings section is rebuilt from the new frontmatter set. | `/save-learnings` rebuilds it. No prompt. |
| **C — gated** | The parent's identity / responsibility / principles / skill flow needs to change. | `/save-learnings` raises an `AskUserQuestion` gate. The user approves; the file is updated. The user rejects; the proposal is logged to journal as "rejected." |

The C-layer protects the top-level identity from automatic drift. Once the user approves a change, the file is updated.

## Blueprint pattern (agents only)

Every agent has a **primary production unit** — the main thing it creates repeatedly. This unit MUST have a blueprint file in `children/` that contains:

1. **Template** — the structural skeleton of the production unit (code scaffold)
2. **Checklist** — everything that must be verified before the unit is complete
3. **Naming conventions** — how files, classes, methods are named
4. **Lifecycle** — creation → registration → testing flow

When the agent needs to create a new instance of its production unit, it reads the blueprint and follows it step by step.

| Agent | Primary production unit | Blueprint file |
|---|---|---|
| API Agent | Feature (Command/Query/Handler/Validator) | `children/workflows.md` |
| Socket Agent | Hub method + Event | `children/hub-method-blueprint.md` |
| Worker Agent | Scheduled Job | `children/job-blueprint.md` |
| Flutter Agent | Screen / Widget | `children/screen-blueprint.md` |
| React Agent | Component / Page | `children/component-blueprint.md` |

Without a blueprint, the agent guesses how to create new units. With a blueprint:

- Every unit follows the same structure
- Nothing is forgotten (checklist guarantees completeness)
- New team members (or new Claude sessions) produce consistent output
- Quality is repeatable, not accidental

(Skills don't have a blueprint pattern — a skill IS the procedure, not a template-driven unit. The Accumulated Learnings section is the skill's equivalent of a blueprint's checklist: things to remember, edge cases to watch for.)

## Related

- [Knowledge system](/guide/knowledge-system) — the project-side mirror (journal + wiki) of this team-side pattern
- [`/save-learnings`](/skills/drain) — writes children/ and learnings/ files; rebuilds parent index sections
- [Concepts: Skill](/guide/concepts#skill) — where the learnings/ pattern fits
- Canonical rule: [`core/rules/agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md)

## History

- `core@1.0.0`: agent children pattern introduced. Knowledge Base section was hand-maintained.
- `core@1.8.0`: Q3 of [self-updating-learning-loop](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) extends children pattern to skills (`learnings/` mirror of `children/`). Knowledge Base + Accumulated Learnings sections become auto-rebuild from frontmatter. C-layer onay gate for identity / core changes formalized as part of the rule. Renamed from "Agent Configuration Rules" to "Agent + skill structure rules" to reflect the broader scope.
