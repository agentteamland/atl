# Glossary

**Agent** — a Markdown file defining a specialized role for Claude Code. Shipped as part of a team. Lives in `agents/` in a team repo; copied into `.claude/agents/` at the install scope.

**atl** — the CLI (`atl install`, `atl list`, …). Resolves and installs teams from the catalog. Go binary.

**Catalog / Index** — the way teams are discovered. A generated index built from public GitHub repositories tagged with the [`atl-team`](https://github.com/topics/atl-team) topic. `atl search` queries it; `atl install` resolves a handle against it. There is no registry repository, no `teams.json`, and no submission PR — tag a repo `atl-team` (or run `atl publish` from it) and the index picks it up. The cached copy lives at `~/.atl/index.json`. See [`atl search`](/cli/search).

**Children pattern** — a convention for complex agents: top-level `agent.md` stays short (identity, scope, principles, Knowledge Base); detailed knowledge lives as topic-per-file under `children/`. Each child file carries a `knowledge-base-summary` frontmatter field that [`/drain`](/skills/drain) uses to auto-rebuild the parent `agent.md`'s Knowledge Base section.

**Dependencies** — additional teams a team requires, specified via the `dependencies` field in `team.json` (a map of team name → version constraint). Resolved and installed at the same time as the team itself.

**Handle** — the catalog identifier you install by, in the form `<handle>/<team>` (e.g. `acme/example-team`). The handle is the publisher namespace; `atl install <handle>/<team>` looks it up in the index and installs from the matching repo.

**Manifest** — the per-team, per-scope install record at `<layer>/.atl/installed/<handle>__<name>.json` (`<layer>` is `~/.atl` for global, `<project>/.atl` for project). Records `schemaVersion`, handle, name, version, scope, source (`repo`, `subpath`, `ref`), `installedAt`, and a `files` map of installed path → SHA-256. Used by `atl remove`, `atl update`, and `atl doctor`.

**Project** — a directory you run `atl` in. Project-scope installs populate its `.claude/` with the team's assets; ATL's project-scope bookkeeping lives under `<project>/.atl/`.

**Rule** — a Markdown file that's always loaded by Claude Code (unlike skills, which must be invoked). Shipped as part of a team in `rules/`; copied into `.claude/rules/` at the install scope.

**Scaffolder** — team-scoped skill named `/create-new-project` that bootstraps a new project on the team's stack. Must follow the [Scaffolder spec](/authoring/scaffolder-spec).

**Scope** — the layer an asset is installed into. Two scopes exist: **global** (assets in `~/.claude`, ATL state in `~/.atl`) and **project** (assets in `<project>/.claude`, ATL state in `<project>/.atl`). A team's publisher declares a default scope (`project`, `global`, or `both`; default `project`) in `team.json`; the user overrides with `--global` / `--project`. When a capability exists at both layers, the project layer **shadows** global — nearest wins.

**SemVer constraint** — version range syntax used in `dependencies` and `requires.atl`. `^1.0.0` (caret), `~1.2.0` (tilde), `1.2.3` (exact), `>=1.2.0` (open-ended).

**Skill** — a user-invocable slash command (e.g. `/drain`). Shipped as a directory with `SKILL.md` at its root. Global skills live in `~/.claude/skills/`; team-scoped skills ship with a team and appear in `.claude/skills/` after install.

**Team** — a Git repository with `team.json` at its root, bundling agents, skills, and rules for a specific kind of work.

**team.json** — the manifest file at the root of every team repo. Declares the team's name, version, description, author, license, what it bundles (agents/skills/rules), its `dependencies`, the minimum `requires.atl` version, and an optional default `scope`. See the [team.json contract](/authoring/team-json).

**Workspace** — `agentteamland/workspace`, the maintainer hub repo where the platform is developed. Not needed to use AgentTeamLand; only relevant if you're contributing to the platform itself.

**Journal** — chronological historical record under `.atl/journal/{YYYY-MM-DD}.md` (one dated file per day, shared across agents — not per-agent). Written by [`/drain`](/skills/drain) as it folds the learning queue into the knowledge base; read by Claude during agent startup per the [knowledge-system rule](https://github.com/agentteamland/atl/blob/main/core/rules/knowledge-system.md).

**knowledge-base-summary** — required YAML frontmatter field on every `children/{topic}.md` file. One- to three-line summary that [`/drain`](/skills/drain) extracts to rebuild the parent `agent.md`'s Knowledge Base section. Source-of-truth — hand edits to the rebuilt section are overwritten on the next `/drain` run.

**knowledge-system** — the core rule that defines the two-layer knowledge model (`journal/` + `wiki/`). Renamed from `memory-system` after the agent-memory layer was merged into journal.

**Learning marker** — inline HTML comment dropped by Claude during a conversation when a learning moment occurs. Format: `<!-- learning: free text -->`. Enqueued into the durable queue (`~/.atl/queue.db`) by `atl tick` (exactly once, deduped by content hash), then processed by [`/drain`](/skills/drain) and acked (deleted).

**Wiki** — topic-organized current-truth knowledge under `.atl/wiki/{topic}.md`. Replaced (not appended) when truth changes; the `<!-- wiki:index -->` marker block in CLAUDE.md keeps the live index visible to Claude on every session start.
