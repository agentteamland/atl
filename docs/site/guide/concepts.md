# Concepts

The pieces that make up AgentTeamLand, and how they fit together.

## Team

A **team** is a package. It bundles everything needed to do a specific kind of work with Claude Code:

- **Agents** — specialized personas with their own context and responsibilities.
- **Skills** — user-invocable commands (slash commands) exposed in Claude Code.
- **Rules** — always-loaded behavioral constraints and conventions.

A team lives in a Git repository with a `team.json` at the root. That file describes the team: its name, version, what it bundles, what it depends on, what it extends.

Install a team into a project and its contents appear as copies inside `.claude/`. Claude Code sees them immediately.

## Agent

An agent is a Markdown file that defines a role. `api-agent`, `flutter-agent`, `code-reviewer` — each one is a focused personality with its own area of responsibility and its own knowledge base.

The convention for complex agents is the **children pattern**: the top-level `agent.md` is short (identity, scope, principles) and detailed knowledge lives under `children/` as topic-per-file. This keeps the top-level file tight and makes it cheap to update one topic without touching the rest. Each child file carries a `knowledge-base-summary` frontmatter line that `/save-learnings` lifts into the auto-rebuilt **Knowledge Base** section of `agent.md` — so the index in the parent file is always derived from the children, never hand-edited.

## Skill

A skill is a user-invocable slash command. `/create-new-project`, `/verify-system`, `/save-learnings`. Skills ship as directories with a `skill.md` at their root; the file describes when to use the skill and what it should do.

Skills mirror the agent shape: complex skills carry a `learnings/` subdirectory of topic-per-file edge cases and accumulated wisdom, with the same `knowledge-base-summary` frontmatter contract. `/save-learnings` rebuilds the **Accumulated Learnings** section of `skill.md` from those frontmatter lines — `learnings/` is to skills what `children/` is to agents.

Skills can be **global** (shipped via bootstrap) or **team-scoped** (shipped by a specific team and only visible after the team is installed). `/create-new-project`, `/verify-system`, and `/design-screen` are team-scoped because the work they do is always stack-specific. `/brainstorm`, `/rule`, `/save-learnings`, `/wiki`, `/create-pr`, `/create-code-diagram` are global because they apply universally.

## Rule

A rule is a Markdown file that gets loaded into every Claude Code session. Unlike a skill (which waits to be invoked), a rule is always active — it shapes how Claude thinks about the project before you ask it anything.

Global rules live in `~/.claude/rules/`. Team-provided rules are copied into a project's `.claude/rules/` when the team is installed.

## Registry

The **registry** is a single JSON file at [`agentteamland/registry`](https://github.com/agentteamland/registry) that maps short team names to Git URLs. Running `atl install software-project-team` looks up the name, finds the URL, and installs from there.

Registry submissions are PRs. CI validates each entry against the JSON schema, checks URL reachability, and flags duplicates.

## Inheritance

A team can **extend** another team. The child inherits the parent's agents, skills, and rules; can **override** any of them by name; and can **exclude** anything it doesn't want.

```json
{
  "name": "my-team",
  "extends": "software-project-team@^1.0.0",
  "excludes": ["ux-agent"],
  "agents": [
    { "name": "api-agent", "description": "Custom overrides for our API patterns" }
  ]
}
```

Constraints:

- **Single parent** — no multiple inheritance.
- **Depth unlimited** — chains can go as deep as they like.
- **Circular detection** — `A extends B extends A` fails fast with the full chain in the error.
- **Load order** — deepest ancestor first, current team last. Closer-to-you wins.

Full details: [Inheritance](/authoring/inheritance).

## The CLI

`atl` is the user-facing tool. It does five things:

- `atl install [team]` — install a team (by registry name or Git URL) into the current project.
- `atl list` — show what's installed here.
- `atl remove [team]` — uninstall.
- `atl update [team]` — pull latest for one or all installed teams.
- `atl search [query]` — search the registry.

See [CLI overview](/cli/overview).

## Workspaces

A **project** is a directory where you run `atl`. It gets a `.claude/` subdirectory with team content copied in.

The **cache** (`~/.claude/repos/agentteamland/`) holds the actual team repos — cloned once, reused across every project that installs the same team. Deleting the cache is safe; `atl update` re-populates it.

## How it plays with Claude Code

Claude Code reads `.claude/` at the start of every session. Whatever a team contributes to that directory shows up immediately — agents available for delegation, skills available as slash commands, rules loaded into every prompt.

AgentTeamLand doesn't replace or extend Claude Code. It's a delivery layer: package management for the files Claude Code already reads.
