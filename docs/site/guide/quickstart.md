# Quickstart

From zero to a production-ready agent team in under a minute.

## 1. Install `atl`

```bash
# macOS / Linux
brew install agentteamland/tap/atl
```

```powershell
# Windows (PowerShell; no package manager required)
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

Full matrix (scoop, winget, manual ZIP, one-liner fallback): [Install guide](/guide/install).

## 2. Create a project directory

```bash
mkdir my-new-app && cd my-new-app
```

`atl` expects to operate inside a project — it will create a `.claude/` directory here for the team's agents, skills, and rules.

## 3. Discover a team

```bash
atl search dotnet
```

You'll see matching entries from the public [registry](https://github.com/agentteamland/registry), including `software-project-team`.

## 4. Install the team

```bash
atl install software-project-team
```

In a few seconds:

- The team is cloned into the shared cache (first time only)
- 13 agents and 3 skills (`create-new-project`, `verify-system`, `design-screen`) are copied into `.claude/`
- `team.json` is recorded in `.claude/.team-installs.json`

You now have a full .NET + Flutter + React + Docker agent team wired into your project.

## 5. Inspect what you installed

```bash
atl list
```

Shows the teams installed in this project, their effective agent counts, and the inheritance chain.

## 6. Use it in Claude Code

Open Claude Code in this directory. The team's skills are available as slash commands:

- `/create-new-project MyApp` — scaffolds a full stack (Phase 1–5: gather → scaffold → build → verify → commit)
- `/verify-system` — runs an end-to-end health check on containers, ports, apps, and pipelines

And every agent (api-agent, socket-agent, worker-agent, flutter-agent, react-agent, database-agent, redis-agent, rmq-agent, infra-agent, code-reviewer, project-reviewer, design-system-agent, ux-agent) is available for Claude to delegate to.

## 7. Keep up to date

When the team author ships improvements:

```bash
atl update
```

All installed teams pull, dependencies resolve, copies refresh. Nothing in your project code changes.

## What just happened?

You installed a curated, version-pinned, dependency-aware set of agents into a project with a single command. Every other project that installs the same team gets the same configuration — and the same updates when the author ships them.

## Add design tooling (optional)

If you want design-system + screen-prototype tooling alongside, install `design-system-team`:

```bash
atl install design-system-team
```

Then in your Claude Code chat:

```
/dst-init
/dst-new-ds primary
/dst-new-prototype --ds primary login-screen
/dst-open
```

You'll get token-aligned design systems and multi-state HTML prototypes under `.dst/`, viewable in any browser. See [design-system-team](/teams/design-system-team) for the full skill set.

## Next

- **[Browse teams](/teams/)** — verified teams in the registry.
- **[Concepts](/guide/concepts)** — the mental model behind teams, agents, skills, and rules.
- **[CLI reference](/cli/overview)** — every command in detail.
- **[Write your own team](/authoring/creating-a-team)** — publish a team to the registry.
