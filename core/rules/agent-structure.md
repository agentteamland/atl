# Agent + skill structure

## Agent: a short identity + a `children/` knowledge base

```
agents/{agent-name}/
├── agent.md      ← identity, area of responsibility, core principles (short), Knowledge Base section
└── children/     ← one topic per file; the agent's detailed, accumulated knowledge
    ├── topic-1.md
    └── topic-2.md
```

Rules:

1. **agent.md stays short** — identity, area of responsibility, core principles, and the auto-aggregated Knowledge Base section. Nothing else.
2. **Detail goes under `children/`** — strategies, patterns, workflows, conventions, each in its own `.md`.
3. **New topic = new file.** Don't hand-edit agent.md; the Knowledge Base section is rebuilt from each child's frontmatter.
4. **Update = one file.** To revise a topic, touch only its child file.
5. **No monolithic agent files** — piling everything into one `.md` is prohibited; it becomes unmanageable.

This single pattern *is* the agent knowledge base. (v1 kept a separate `learnings/` mirror for skills; v2 unifies — there is one knowledge base, the agent's `children/`. Skills are procedures, not knowledge stores.)

## The Knowledge Base section (auto-rebuilt)

In agent.md, every child topic is listed as a heading + its summary + a details link:

```markdown
## Knowledge Base

### Logging Strategy
Log every step in every handler — production has no breakpoints.
→ [Details](children/logging-strategy.md)
```

This section is **derived**: the `/drain` skill rebuilds it wholesale from each child's `knowledge-base-summary` frontmatter (sorted by filename). Hand-edits are overwritten — the source of truth is the frontmatter.

Required frontmatter on every `children/*.md`:

```markdown
---
knowledge-base-summary: "<one-to-three-line summary for the Knowledge Base section>"
---

# <Topic Title>

<the actual knowledge — patterns, examples, the why — as long as needed>
```

## Structural / identity changes — confirmed, never silent

`/drain` writes children files and rebuilds the Knowledge Base automatically, so knowledge accumulates frictionlessly. But changes to an agent's **identity, area of responsibility, or core principles** — and creating a new agent / skill / rule — are NOT automatic: the skill proposes them via `AskUserQuestion` and a human confirms. This is the reactive-creation boundary (decision doc D-1): accumulation is automatic; structural growth is confirmed.

## Blueprint pattern

Every agent has a **primary production unit** — the main thing it creates repeatedly (an API feature, a Flutter screen, a React component). That unit gets a blueprint file in `children/` with: the template (code scaffold), a completion checklist, naming conventions, and the creation → registration → test lifecycle. When the agent makes a new instance, it follows the blueprint — so output is consistent and complete, not accidental.
