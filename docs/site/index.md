---
layout: home

hero:
  name: AgentTeamLand
  text: AI agent teams, installed like packages.
  tagline: A package manager and registry for curated teams of AI agents — install a stack, inherit another team's work, ship.
  image:
    src: /logo.svg
    alt: AgentTeamLand
  actions:
    - theme: brand
      text: Get started
      link: /guide/quickstart
    - theme: alt
      text: Install atl
      link: /guide/install
    - theme: alt
      text: GitHub
      link: https://github.com/agentteamland

features:
  - icon: 📦
    title: Teams as packages
    details: A team bundles specialized agents, skills, and rules for a kind of work — full-stack apps, data pipelines, mobile products. Install once, copied into your project.
  - icon: 🧬
    title: Inheritance done right
    details: Single-parent extends with override and excludes. Unlimited depth, circular detection, caret version constraints — npm-grade semantics, no diamond problem.
  - icon: ⚡
    title: One static binary
    details: atl is a ~7 MB Go binary. Install via Homebrew, Scoop, winget, or curl — zero runtime dependencies.
  - icon: 🧪
    title: Two verified teams, more on the way
    details: software-project-team (13 agents — .NET, Flutter, React, Docker) and design-system-team (DS + prototype tooling, /dst-* skills). Browse the registry.
  - icon: 🔍
    title: Public registry
    details: Discover teams by name. The registry is a single JSON file, PR-driven, schema-validated in CI.
  - icon: 🛠️
    title: Open and scriptable
    details: Every piece is MIT-licensed. team.json is a public schema. Build your own team and submit it.
---

<div style="text-align:center; margin: 3rem 0 1rem;">

## See it in action

<img src="https://raw.githubusercontent.com/agentteamland/workspace/main/assets/demo.gif" alt="atl demo" width="820" style="max-width:100%; border-radius:8px;"/>

</div>

## In 30 seconds

```bash
# macOS / Linux
brew install agentteamland/tap/atl
```

```powershell
# Windows (PowerShell, no package manager required)
irm https://raw.githubusercontent.com/agentteamland/cli/main/scripts/install.ps1 | iex
```

```bash
# Then, in any project:
atl install software-project-team
atl install design-system-team    # optional: design-system + prototype tooling
atl setup-hooks                   # optional: auto-update on every Claude Code session
```

A full 13-agent stack (API, web, mobile, database, infra, review) wired into your project's `.claude/` directory, ready for Claude Code. Add design-system-team alongside for `/dst-*` design tooling.

## What's next?

- **[What is `atl`?](/guide/what-is-atl)** — the big idea in five minutes.
- **[Quickstart](/guide/quickstart)** — first team installed in under a minute.
- **[Browse teams](/teams/)** — verified teams in the registry.
- **[Team authoring](/authoring/team-json)** — publish your own team.
