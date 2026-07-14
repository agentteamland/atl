# Children + learnings

The shape a complex agent uses for accumulated domain knowledge: a short top-level `agent.md` plus a `children/` directory of topic-per-file detail pages, each carrying a one-line `knowledge-base-summary` frontmatter that auto-rebuilds the parent file's Knowledge Base section.

This is the **agent knowledge base** — an agent's `children/` directory. (v1 mirrored the same shape onto skills as a `learnings/` directory; **v2 removed that** — skills are procedures, not knowledge stores. See [History](#history).)

The canonical rule lives at [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md). This page is the user-facing summary.

## Why this pattern exists

Without it, a complex agent ends up as one of two anti-shapes:

1. **Monolithic files** — everything piled into one `agent.md`. Hard to update one piece without touching the rest. Diffs are noisy. Re-reads burn tokens.
2. **Hand-curated index sections** — a separate `agent.md` table of contents that a human maintains alongside the topic files. Drifts the moment someone forgets to update it.

The children pattern resolves both:

- **Topic-per-file** — update one piece without touching others
- **Auto-rebuilt index** — the top-level file's **Knowledge Base** section is regenerated from frontmatter on every [`/drain`](/skills/drain) run. Hand edits are overwritten — the source of truth is each child's frontmatter.

Result: knowledge accumulates frictionlessly, the top-level file stays tight, and the index never goes stale.

## Children — for agents

Every complex agent is organized as:

```
.claude/agents/{agent-name}/
├── agent.md              ← Identity, area of responsibility, core principles (short, embedded)
└── children/             ← Detailed information, patterns, strategies (each topic in a separate file)
    ├── topic-1.md
    ├── topic-2.md
    └── ...
```

[`atl install`](/cli/install) fetches the team from the catalog and copies agents, skills, and rules into your project's `.claude/` directory at this path.

### Rules

1. **`agent.md` stays short.** Only: identity, area of responsibility (positive list), core principles (unchanging, short bullets), Knowledge Base section (auto-aggregated), "read children/" instruction.
2. **Everything detailed goes under `children/`.** Strategies, patterns, workflows, conventions — each in a separate `.md` file.
3. **New topic = new file.** Without touching `agent.md` by hand, add a `.md` file under `children/`. The Knowledge Base section is rebuilt automatically by `/drain` from each child file's frontmatter.
4. **Update = single file.** To update a topic, only the relevant `children/` file is touched.
5. **Monolithic agent files are prohibited.**
6. **This pattern applies to all agents.** API, Socket, Worker, Flutter, React, Mail, Log, Infra — all follow the same structure.

## Skills have no `learnings/` mirror

In v1 this same shape was mirrored onto skills: a `learnings/` directory beside `SKILL.md`, auto-rebuilt into an "Accumulated Learnings" section. **v2 removed that.** The knowledge base is unified into the agent's `children/` — **skills are procedures, not knowledge stores**, so a skill directory carries no `learnings/` mirror.

This is the canonical rule in [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md): "v1 kept a separate `learnings/` mirror for skills; v2 unifies — there is one knowledge base, the agent's `children/`. Skills are procedures, not knowledge stores." The rest of this page is about that one knowledge base: the agent `children/` directory.

## The frontmatter contract

Every `children/*.md` file MUST carry a `knowledge-base-summary` frontmatter field:

```markdown
---
knowledge-base-summary: "<one-to-three-line summary used in the auto-rebuilt index section>"
---

# <Topic Title>

<the actual content — patterns, strategies, examples — as long as needed>
```

This summary is what feeds the parent file's Knowledge Base section. Without it, `/drain` either skips the topic in the rebuild OR (for new files it created itself) writes the field with a generated summary; in both cases the file should have one.

## Auto-rebuilt index sections

When `/drain` runs, it rebuilds the `agent.md` **Knowledge Base** section from the frontmatter of every `children/*.md`:

```markdown
## Knowledge Base

### <Topic 1 (heading-cased from filename)>
<knowledge-base-summary>
→ [Details](children/topic-1.md)

### <Topic 2>
<knowledge-base-summary>
→ [Details](children/topic-2.md)

...
```

Hand edits to this section are **overwritten** on the next `/drain` run — the source of truth is each child file's frontmatter. The rest of `agent.md` (identity, responsibility, principles) is **not touched** by the rebuild.

## Three layers of update

The split lets "knowledge accumulates" be automatic and frictionless, while protecting the top-level file's identity from drift:

| Layer | What changes | How |
|---|---|---|
| **A — auto** | A `children/{topic}.md` file is created or updated. | `/drain` writes it directly. No prompt. |
| **B — auto** | The parent's Knowledge Base section is rebuilt from the new frontmatter set. | `/drain` rebuilds it. No prompt. |
| **C — gated** | The parent's identity / responsibility / principles need to change. | `/drain` raises an `AskUserQuestion` gate. The user approves; the file is updated. The user rejects; the proposal is logged to journal as "rejected." |

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

(Skills don't have a blueprint pattern — a skill IS the procedure, not a template-driven unit.)

## Related

- [Knowledge system](/guide/knowledge-system) — the project-side mirror (journal + wiki) of this team-side pattern
- [`/drain`](/skills/drain) — writes `children/` files; rebuilds the agent Knowledge Base section
- Canonical rule: [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md)

## History

- `core@1.0.0`: agent children pattern introduced. Knowledge Base section was hand-maintained.
- `core@1.8.0`: Q3 of [self-updating-learning-loop](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) extended the children pattern to skills (a `learnings/` mirror of `children/`), with both the Knowledge Base and an "Accumulated Learnings" section auto-rebuilt from frontmatter. C-layer approval gate for identity / core changes formalized as part of the rule. The rule was renamed from "Agent Configuration Rules" to "Agent + skill structure rules" to reflect the broader scope.
- **atl v2**: the skill `learnings/` mirror was **removed** — the knowledge base unified back to the agent's `children/`; skills are procedures, not knowledge stores (per [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md)).
