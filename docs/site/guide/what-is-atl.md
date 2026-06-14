# What is atl?

`atl` is a command-line tool that installs **teams of AI agents** into a project — the same way `npm` installs packages of JavaScript code, or `brew` installs Unix binaries.

## The problem

Using Claude Code well requires configuration: agents, skills, and rules that shape how the model reasons about your codebase. You end up copying files between projects, forking someone else's setup, and slowly watching them drift apart. Every new project re-solves problems that a previous project already solved.

## The answer

A **team** is a package of agents, skills, and rules, built around a particular kind of work. One team might be geared for a .NET + Flutter + React stack with a Docker-compose production layout. Another might be for a Next.js + Sanity + Vercel blog stack. A third for data pipelines with Airflow and dbt.

`atl install some-team` fetches the team, caches it, and copies its agents, skills, and rules into your current project's `.claude/` directory. Claude Code sees the team the moment you open the editor.

When the team author ships a fix, you run `atl update` and every project that uses that team picks up the change. Your projects stop drifting.

## Not a fork

Teams can **extend** other teams. If you want 95 % of `software-project-team` but one different agent and one extra skill, you write a tiny team that `extends` it, overrides the one agent, and adds the skill. The parent keeps evolving in its own repo; your child inherits updates for free.

## Not a walled garden

Every team is just a Git repository with a `team.json` file at the root. There's a public [registry](https://github.com/agentteamland/registry) so teams get short names like `software-project-team`, but you can install from any Git URL. The schema is public. The CLI is MIT-licensed Go. The spec is documented here.

## Who is this for?

- **Developers** who want a solid Claude Code setup without hand-rolling it for each project.
- **Team leads** who want to standardize how their company uses Claude across repos and onboard new engineers in minutes.
- **Stack authors** who want to publish opinionated agent teams the way framework authors publish CLIs today.

## Where it stands

`atl` is at **v1.1.x** — production. The install topology is project-local copies (with a global cache as source-of-truth), the auto-update path runs through Claude Code `SessionStart` + `UserPromptSubmit` hooks, and a self-updating learning loop persists session knowledge into journal/wiki/agent-children/skill-learnings layers.

Two verified teams ship in the registry today: `software-project-team` (13 agents — .NET API + Flutter + React + Docker stack) and `design-system-team` (2 agents + 10 `/dst-*` skills for design-system + prototype tooling). The full ecosystem is MIT-licensed and open for PRs.

Next up:
- **[Install `atl`](/guide/install)**
- **[Quickstart — from zero to running team in 60 seconds](/guide/quickstart)**
