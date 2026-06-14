# Worked example: `starter-extended`

[`agentteamland/starter-extended`](https://github.com/agentteamland/starter-extended) is a minimal example team that demonstrates the [inheritance](inheritance) mechanism end-to-end. Read this page if you're about to author your own extension team and want to see what the smallest viable example looks like.

For the inheritance mechanism itself (extends, excludes, override semantics, load order), see the [Inheritance](inheritance) page first.

## What `starter-extended` shows

In one ~50-line repo:

- **`extends`** — declaring a parent team in `team.json` (`extends: software-project-team@^1.0.0`)
- **`excludes`** — dropping an inherited member by name (`excludes: ["ux-agent"]`)
- **Adding** a new agent on top of the parent (`stripe-agent`)
- The canonical agent layout (per [`agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md)) — Identity / Area of Responsibility / Core Principles / Knowledge Base, with topic-per-file under `children/` and `knowledge-base-summary` frontmatter

Total: 1 `team.json` + 1 agent (`stripe-agent` with 1 child file).

## Repo layout

```
starter-extended/
├── README.md                                # stub — points at this docs page
├── LICENSE                                  # MIT
├── team.json                                # declares: extends + excludes + the new agent
└── agents/
    └── stripe-agent/
        ├── agent.md                         # Identity, Responsibility, Core Principles, Knowledge Base
        └── children/
            └── webhook-topology.md          # with knowledge-base-summary frontmatter
```

## The `team.json`

```json
{
  "name": "starter-extended",
  "version": "0.2.0",
  "description": "Minimal example team — demonstrates extending software-project-team, excluding ux-agent, and adding stripe-agent.",
  "author": "agentteamland",
  "license": "MIT",
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"],
  "agents": [
    {
      "name": "stripe-agent",
      "description": "Stripe billing integration: webhook topology, idempotency, refund flows.",
      "path": "agents/stripe-agent/"
    }
  ]
}
```

When someone runs `atl install starter-extended`:

1. `atl` resolves the parent (`software-project-team@^1.0.0`) from the registry → installs all 13 agents + 3 skills
2. The `excludes: ["ux-agent"]` rule removes `ux-agent` from the installed set → 12 agents + 3 skills remain
3. The child team's own agent (`stripe-agent`) is installed on top → 13 agents + 3 skills total

The user ends up with the parent's full stack minus ux-agent, plus a custom Stripe-focused agent. This is the most common "I want most of software-project-team but with my own twist" pattern.

## The agent (`stripe-agent`)

`agents/stripe-agent/agent.md` follows the canonical structure (per [`agent-structure.md`](https://github.com/agentteamland/core/blob/main/rules/agent-structure.md)):

```markdown
# Stripe Agent

## Identity
Stripe billing integration specialist. Handles webhook events,
idempotency, refunds, dispute flows, and Stripe-specific error semantics.

## Area of Responsibility
- Stripe webhook endpoint design + signature verification
- Webhook idempotency (event ID dedup, replay protection)
- Refund + partial-refund flows
- Dispute handling
- Stripe-specific error mapping to internal exceptions

## Core Principles
- Never trust webhook payloads without signature verification
- Always idempotent — Stripe retries aggressively
- Refund logic lives in API; Stripe SDK calls go through a dedicated service

## Knowledge Base

(Auto-rebuilt by /save-learnings from children/*.md frontmatter.)

### Webhook Topology
Stripe's webhook event taxonomy + the exchange/queue topology we
use to fan out events to consumers. Idempotency is enforced at
event-ID level via Redis SETNX with 7-day TTL.
→ [Details](children/webhook-topology.md)
```

The `children/webhook-topology.md` file carries the canonical `knowledge-base-summary` frontmatter:

```markdown
---
knowledge-base-summary: "Stripe's webhook event taxonomy + the exchange/queue topology we use to fan out events to consumers. Idempotency is enforced at event-ID level via Redis SETNX with 7-day TTL."
---

# Webhook Topology

(Detailed content: full schemas, naming conventions, retry behavior, ...)
```

This is the [Children + learnings](../guide/children-and-learnings) pattern in action — the parent `agent.md`'s Knowledge Base section is auto-rebuilt from the frontmatter of every child file.

## Use it as a template

The repo is set up as a GitHub Template. Bootstrap your own extension team:

```bash
gh repo create your-org/your-extension-team --template agentteamland/starter-extended
```

Then edit `team.json`:

- Pick a different parent (`extends: design-system-team@^0.8.0`, etc.)
- Adjust `excludes` to drop members you don't want
- Add your own agents under `agents/`, skills under `skills/`, rules under `rules/`

Validate locally before pushing:

```bash
~/.claude/repos/agentteamland/core/scripts/validate-team-json.sh team.json
```

Or wire the validator into git push (recommended):

```bash
git config core.hooksPath .githooks    # one-time per clone (if you copied .githooks too)
```

After validation passes, push, get a tag, and submit to the [registry](registry-submission) if you want a short-name install.

## When to extend vs author from scratch

Extend when:

- You agree with the parent team's architectural choices (i18n, Mediator, Docker-first, etc.) and just want to add specialty agents on top
- You want updates to flow from the parent — when `software-project-team` ships v1.3.0, your extension picks up the parent updates automatically (per the `^1.0.0` constraint)
- You want a small, focused codebase (your team's repo only contains the deltas)

Author from scratch when:

- Your stack is fundamentally different (different language, different patterns, different infrastructure)
- You want full control without any inherited surface
- You're targeting a different platform (e.g., AgentTeamLand-style team for Rust services)

For the "small specialty addition" case, extension is dramatically less work. `starter-extended` itself is ~50 lines of repo total — that's the entire cost of "the .NET stack but with stripe-agent added."

## Related

- [Inheritance](inheritance) — the mechanism this example demonstrates
- [Creating a team](creating-a-team) — the from-scratch path
- [Children + learnings](../guide/children-and-learnings) — the agent.md / children/ layout used here
- [Registry submission](registry-submission) — getting your team's short name into the catalog

## History

- `v0.1.0` (2026-04-17) — initial example demonstrating inheritance
- `v0.2.0` (2026-05-02) — rescued during the platform-wide review (Phase 2.B/2.C migration applied: agent.md sections renamed to canonical schema + Knowledge Base section added; `knowledge-base-summary` frontmatter added to children files; LICENSE + README added)
