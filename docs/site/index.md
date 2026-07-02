---
layout: home

hero:
  name: AgentTeamLand
  text: AI agent teams, installed like packages.
  tagline: A package manager for curated teams of AI agents — install a stack, keep it current, ship.
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
      link: https://github.com/agentteamland/atl

features:
  - icon: 📦
    title: Teams as packages
    details: A team bundles specialized agents, skills, and rules for a kind of work — full-stack apps, design systems, and more. Install once, copied into your project's Claude Code directory.
  - icon: ⚡
    title: One static binary
    details: atl is a ~7 MB Go binary with zero runtime dependencies. Install with a single curl (macOS/Linux) or PowerShell (Windows) command.
  - icon: 🔄
    title: Self-driving updates + learning
    details: Hooks keep your teams current and fold in-session learnings into their knowledge base automatically — promote gains to your global layer, publish them upstream.
  - icon: 🧪
    title: A community catalog
    details: No central gatekeeper — the catalog is generated from public GitHub repos tagged with the atl-team topic. Anything listed installs with one command.
  - icon: 🔍
    title: Open self-publish
    details: Tag your repo with the atl-team GitHub topic and run atl publish — no central gatekeeper. Discover any team by name with atl search.
  - icon: 🛠️
    title: Open and scriptable
    details: Every piece is MIT-licensed. team.json is a public schema. Build your own team and publish it.
---

<div style="text-align:center; margin: 3rem 0 1rem;">

## See it in action

<img src="https://raw.githubusercontent.com/agentteamland/workspace/main/assets/demo.gif" alt="atl demo" width="820" style="max-width:100%; border-radius:8px;"/>

</div>

## In 30 seconds

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

```bash
# Then, in any project:
atl install <handle>/<team>
atl setup-hooks                   # optional: auto-update + learning capture every Claude Code session
```

The team's agents, skills, and rules are wired into your project's `.claude/` directory, ready for Claude Code.

## What's next?

- **[What is `atl`?](/guide/what-is-atl)** — the big idea in five minutes.
- **[Quickstart](/guide/quickstart)** — first team installed in under a minute.
- **[Browse teams](/teams/)** — how the catalog works and how to discover teams.
- **[Team authoring](/authoring/team-json)** — publish your own team.
