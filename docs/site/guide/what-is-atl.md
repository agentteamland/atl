# What is atl?

`atl` is a command-line tool that installs **teams of AI agents** into a project — the same way `npm` installs packages of JavaScript code, or `brew` installs Unix binaries.

## The problem

Using Claude Code well requires configuration: agents, skills, and rules that shape how the model reasons about your codebase. You end up copying files between projects, forking someone else's setup, and slowly watching them drift apart. Every new project re-solves problems that a previous project already solved.

## The answer

A **team** is a package of agents, skills, and rules, built around a particular kind of work. One team might be geared for a .NET + Flutter + React stack with a Docker-compose production layout. Another might be for a Next.js + Sanity + Vercel blog stack. A third for data pipelines with Airflow and dbt.

`atl install some-team` resolves the team against a GitHub-backed catalog, fetches its source, and copies its agents, skills, and rules into your current project's `.claude/` directory. Claude Code sees the team the moment you open the editor.

When the team author ships a fix, you run `atl update` and every project that uses that team picks up the change. Your projects stop drifting.

## Not a walled garden

Every team is just a public GitHub repository with a `team.json` file at the root. There's no central registry to submit to: tag a repo with the [`atl-team`](https://github.com/topics/atl-team) topic and it shows up in a generated **catalog**, where anyone can find it by handle and install it by `<handle>/<name>`. `atl search` queries that catalog; `atl install` resolves against it. The CLI is MIT-licensed Go. The team contract is documented here — see [the `team.json` reference](/authoring/team-json).

## Who is this for?

- **Developers** who want a solid Claude Code setup without hand-rolling it for each project.
- **Team leads** who want to standardize how their company uses Claude across repos and onboard new engineers in minutes.
- **Stack authors** who want to publish opinionated agent teams the way framework authors publish CLIs today.

## Where it stands

`atl` is **v2** — a single monorepo ([`agentteamland/atl`](https://github.com/agentteamland/atl)), currently in **alpha**. The install topology is project-local copies fetched from the catalog (no persistent on-disk clone cache — sources are fetched fresh and discarded after each install), the auto-update path runs through Claude Code `SessionStart` + `UserPromptSubmit` hooks, and the learning loop persists session knowledge: inline markers are queued durably, and the `/drain` skill folds each into the journal, wiki, and agent knowledge bases.

Teams are discovered through the open catalog — any public repo tagged with the [`atl-team`](https://github.com/topics/atl-team) topic is listed, and `atl search` finds it by name (the v1-era first-party teams were retired in 2026-07; their v2 rebuild is pending). The whole platform is MIT-licensed and open for contributions.

Next up:
- **[Install `atl`](/guide/install)**
- **[Quickstart — from zero to running team in 60 seconds](/guide/quickstart)**
