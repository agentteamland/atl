# Concepts

The pieces that make up AgentTeamLand, and how they fit together.

## Team

A **team** is a package. It bundles everything needed to do a specific kind of work with Claude Code:

- **Agents** — specialized personas with their own context and responsibilities.
- **Skills** — user-invocable commands (slash commands) exposed in Claude Code.
- **Rules** — always-loaded behavioral constraints and conventions.

A team lives in a Git repository with a `team.json` at the root. That file describes the team: its name, version, and what it bundles.

Install a team and its contents appear as copies inside `.claude/`. Claude Code sees them immediately.

## Agent

An agent is a Markdown file that defines a role. `backend-agent`, `frontend-agent`, `code-reviewer` — each one is a focused personality with its own area of responsibility and its own knowledge base.

The convention for complex agents is the **children pattern**: the top-level `agent.md` is short (identity, scope, principles) and detailed knowledge lives under `children/` as topic-per-file. This keeps the top-level file tight and makes it cheap to update one topic without touching the rest. Each child file carries a `knowledge-base-summary` frontmatter line that `/drain` lifts into the auto-rebuilt **Knowledge Base** section of `agent.md` — so the index in the parent file is always derived from the children, never hand-edited.

See [Children + learnings](/guide/children-and-learnings) for the full shape.

## Skill

A skill is a user-invocable slash command. `/drain`, `/create-pr`, `/docs-audit`. Skills ship as directories with a `SKILL.md` at their root; the file describes when to use the skill and what it should do.

Skills are **procedures, not knowledge stores** — a skill is the steps to run, so it carries no accumulated-knowledge directory. The knowledge base is unified into the agent's `children/` (v1 mirrored that shape onto skills as a `learnings/` directory; v2 removed it, per [`core/rules/agent-structure.md`](https://github.com/agentteamland/atl/blob/main/core/rules/agent-structure.md)).

Skills can be **global** (shipped with `atl` itself) or **team-scoped** (shipped by a specific team and only visible after the team is installed). Scaffolder skills like `/create-new-project` and `/verify-system` are team-scoped by convention, because the work they do is always stack-specific. `/drain`, `/create-pr`, `/create-code-diagram`, `/brainstorm`, `/rule`, `/rule-wizard`, `/docs-audit`, `/publish`, `/skill-stocktake`, and `/rules-distill` are global because they apply universally.

The split between a skill and the CLI is deliberate: the **CLI is deterministic** (same inputs, same result, no LLM); **skills are LLM-driven** (they reason about your specific code). `/drain` is the judgment half of the learning loop; `atl learnings` is the deterministic half. See [The CLI](#the-cli) below.

## Rule

A rule is a Markdown file that gets loaded into every Claude Code session. Unlike a skill (which waits to be invoked), a rule is always active — it shapes how Claude thinks about the project before you ask it anything.

Global rules live in `~/.claude/rules/`. Team-provided rules are copied into a project's `.claude/rules/` when the team is installed.

## Scope: global and project

Everything `atl` installs exists in one of two **scopes**:

- **Global** (`~/.claude/`) — agents, skills, and rules shared across every project on your machine.
- **Project** (`<project>/.claude/`) — the same kinds of assets, scoped to a single project.

When both scopes carry an asset with the same name, **the project copy wins** — nearest shadows global. This is the axis that every other concept hangs off: you install at a scope, gains circulate _between_ scopes, and Claude Code reads the merged result.

There is no separate ATL-owned asset store. Assets live in Claude Code's own directories; `atl`'s bookkeeping (the learning queue, the cached catalog, pins, install manifests) lives under `~/.atl/` and `<project>/.atl/`.

## The team catalog

Teams are discovered through a **catalog** — a generated index built from public GitHub repositories tagged with the [`atl-team`](https://github.com/topics/atl-team) topic. Running `atl install <handle>/<team>` resolves the reference against that index and installs from the matching repo.

There is no registry repository and no submission PR. To get a team listed, you tag its repo with `atl-team` (or run `atl publish` from the team repo) and the index picks it up. `atl search` queries the index; `atl install` resolves against it. A **`[verified]`** badge marks teams reviewed by AgentTeamLand maintainers — its absence just means a team is self-published, not that it's unsafe.

See [`atl search`](/cli/search) for how the index is queried and refreshed.

## Gain circulation

As your agents work, they accumulate **gains** — new learnings, sharpened skills, project-local rules. AgentTeamLand moves those gains outward through a three-ring ladder so nobody re-solves a solved problem:

1. **Project → global** — `atl promote` lifts a project-local gain to the global layer, so every project benefits. `atl pin` holds a path back when a customization should stay project-only; `atl unpin` releases it.
2. **Global → upstream** — `atl publish` shares your global-layer gains back to the team's source repo: re-publish a team you own, or propose the gains as a GitHub PR for a team you don't. It crosses the author boundary, so it never runs automatically — you invoke it; the owner accepts.
3. **Upstream → everyone** — `atl update` pulls the latest published version of each installed team, fanning shared gains back down to every install.

`promote` and the fan-out in `update` are routine; `publish` is deliberate by design.

## The learning queue

Learning capture is automatic. When a learning marker lands in a session, it is appended to a **durable queue** (`~/.atl/queue.db`, a bbolt store) rather than processed inline — which is why a long session never re-reports the same item. The `/drain` skill processes the queue into agent knowledge bases; `atl learnings` is the deterministic window onto it (`status` for what's pending, `peek` to list items, `ack` to mark one processed).

This replaces v1's `/save-learnings` (now `/drain`) and removes the separate `memory` concept entirely: there is no standalone memory layer. Per-agent learnings live in the agent's knowledge base ([children + learnings](/guide/children-and-learnings)); cross-cutting knowledge lives in the project's [knowledge system](/guide/knowledge-system) (journal + wiki).

## The CLI

`atl` is the deterministic, user-facing tool. Its commands fall into three groups:

**Team commands** (run by hand):

- `atl install [team]` — install a team (by catalog handle) into the current scope.
- `atl list` — show what's installed here.
- `atl remove [team]` — uninstall.
- `atl update [team]` — pull latest for one or all installed teams.
- `atl search [query]` — search the team catalog.

**Gain-circulation commands** (move what your agents learn outward):

- `atl promote` — lift project-local gains to the global layer.
- `atl publish` — re-publish your own team, or propose gains upstream.
- `atl pin` / `atl unpin` — hold a path back from promotion, or release it.
- `atl learnings` — inspect the durable learning queue (`status` / `peek` / `ack`).

**Automation commands** (wired to Claude Code hooks; you rarely type them):

- `atl setup-hooks` — one-time install of the automation hooks: `SessionStart` (session-start maintenance), `UserPromptSubmit` (`atl tick` + per-prompt knowledge retrieval via `atl retrieve`), and `PreToolUse` (`atl guard`).
- `atl session-start` — boot-time maintenance (core refresh + marker scan + doctor self-heal + a daily binary self-update check).
- `atl tick` — the in-session maintenance tick (drains throttled background work every few minutes).
- `atl doctor` — the self-heal daemon: diagnoses drift and repairs the install automatically.

Compared with v1, there is no `config`, `migrate`, or `learning-capture` command. See the [CLI overview](/cli/overview).

## How it plays with Claude Code

Claude Code reads `.claude/` at the start of every session. Whatever a team contributes to that directory shows up immediately — agents available for delegation, skills available as slash commands, rules loaded into every prompt. The project scope shadows the global scope, so the merged view is always "nearest wins."

AgentTeamLand doesn't replace or extend Claude Code. It's a delivery layer: package management — plus a learning loop — for the files Claude Code already reads.
