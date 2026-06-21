# Quickstart

From zero to a production-ready agent team in under a minute — install the CLI, install your first team, open a session.

## 1. Install `atl`

`atl` is a single static Go binary with zero runtime dependencies. Install it with the one-line script for your platform:

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Prefer to install by hand? Grab a pre-built binary from [GitHub Releases](https://github.com/agentteamland/atl/releases/latest) and drop it on your `PATH`. There is no Homebrew, Scoop, or winget channel — the script (or the release ZIP) is the supported path. Full detail: [Install guide](/guide/install).

Verify:

```bash
atl --version
```

## 2. Create a project directory

```bash
mkdir my-new-app && cd my-new-app
```

`atl` operates inside a project. When you install a team it writes the team's agents, skills, and rules into this project's `.claude/` directory — exactly where Claude Code reads them.

## 3. Find a team

```bash
atl search dotnet
```

[`atl search`](/cli/search) queries the GitHub-backed catalog — generated from public repos tagged with the [`atl-team`](https://github.com/topics/atl-team) topic. Each result prints the `<handle>/<team>@<version>` reference (the handle is the team's GitHub owner — ownership is authorship) and the exact `atl install` command to copy. There is no central registry repo to PR against any more; tagging a repo `atl-team` (or running [`atl publish`](/cli/publish) from it) is how a team gets listed.

## 4. Install the team

```bash
atl install agentteamland/software-project-team
```

In a few seconds:

- The team is resolved from the index and fetched (cached for reuse).
- 13 agents and 3 skills (`create-new-project`, `verify-system`, `design-screen`) are written into `.claude/`.
- A manifest of baseline file hashes is recorded under `.atl/` so future updates can tell your edits from upstream changes.
- The automation hooks are bound into Claude Code as part of the install — automation is on by default, not opt-in.

You now have a full .NET + Flutter + React + Docker agent team wired into the project.

::: tip Global vs. project scope
A team is installed wherever its publisher's default points. Override with `--global` (every project) or `--project` (this one only); when both layers carry a team, the project copy shadows the global one. See [Concepts](/guide/concepts) for the scope axis.
:::

## 5. See what you installed

```bash
atl list
```

[`atl list`](/cli/list) shows the teams installed at each scope — global (`~/.claude`) and project (`<cwd>/.claude`). A team present at both is listed under each.

## 6. Use it in Claude Code

Open Claude Code in this directory. The team's skills are available as slash commands:

- `/create-new-project MyApp` — scaffolds a full stack (gather → scaffold → build → verify → commit).
- `/verify-system` — runs an end-to-end health check on containers, ports, apps, and pipelines.

Every agent the team ships (api-agent, socket-agent, worker-agent, flutter-agent, react-agent, infra-agent, database-agent, redis-agent, rmq-agent, code-reviewer, project-reviewer, design-system-agent, ux-agent) is available for Claude to delegate to.

The platform's own global skills are there too — `/drain`, `/create-pr`, `/create-code-diagram`, `/brainstorm`, `/rule`, `/rule-wizard` — usable in any project regardless of which team you installed.

## 7. Let the learning loop run itself

This is the part that keeps your setup getting better instead of drifting. While you work, agents capture what they learn into a durable queue. The automation hooks from step 4 mean you don't manage any of it by hand:

- A maintenance **tick** runs in-session (and via `atl tick`), folding queued learnings into the knowledge base.
- `atl doctor` self-heals the install — it's the always-on health daemon, not a command you have to remember.
- When something is waiting on you, `atl` reports `N learning(s) pending`; the `/drain` skill (run it in your session) routes each item to the right home — a wiki page, the journal, or an agent's knowledge base — then deletes it from the queue.

Peek at the queue any time:

```bash
atl learnings status
```

`atl learnings peek` lists the pending items and `atl learnings ack <id>` marks one processed.

## 8. Keep current

When a team author ships improvements:

```bash
atl update
```

All installed teams refresh; copies you haven't modified are updated in place, and your local edits are preserved. Nothing in your project's own code changes.

## What just happened?

You installed a curated, version-pinned set of agents into a project with one command, and turned on a self-running maintenance loop. Every other project that installs the same team gets the same configuration — and the same updates when the author ships them — while the gains your agents learn circulate back through `atl promote` and `atl publish`.

## Add design tooling (optional)

For design-system + screen-prototype tooling, install `design-system-team`:

```bash
atl install agentteamland/design-system-team
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

- **[Browse teams](/teams/)** — catalogued teams you can install.
- **[Concepts](/guide/concepts)** — teams, agents, skills, rules, and the global/project scope axis.
- **[CLI reference](/cli/overview)** — every command in detail.
- **[Creating a team](/authoring/creating-a-team)** — author and publish your own.
