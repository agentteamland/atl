# Verified Teams

The official AgentTeamLand registry currently contains **2 verified teams**. Each team is independently versioned, opens to community contribution, and follows the [team.json schema](/authoring/team-json).

> **Verified status** means the team has been reviewed by AgentTeamLand maintainers for quality, scope, and dependency hygiene. Community-contributed teams (status: `community`) appear with a warning during install.

## Browse

| Team | Version | Description |
|------|---------|-------------|
| [`software-project-team`](/teams/software-project-team) | 1.2.1 | 13 specialized agents for full-stack software projects (.NET 9 + Flutter + React + Postgres + RabbitMQ + Redis + Elasticsearch + MinIO). Phase 2.C: agent KB sections auto-rebuilt from children frontmatter. |
| [`design-system-team`](/teams/design-system-team) | 0.8.1 | Design systems and UI prototypes inside any project — local, file-based, browser-viewable. `/dst-*` skills produce JSON state and Tailwind-rendered HTML pages under `.dst/`. |

## Install any team

```bash
atl install <team-name>
```

Both teams above can be installed by short name. atl resolves the name via the public registry, then clones the repo into the shared cache (`~/.claude/repos/agentteamland/`).

## Install multiple teams in one project

Both teams coexist cleanly in the same project — `atl` v0.1.2+ supports multi-team install with collision warnings (when two teams declare an item with the same name, the most recently installed one wins, with a one-line warning).

```bash
cd your-project
atl install software-project-team       # full-stack agents + scaffolder
atl install design-system-team          # add design-system + prototype tooling

atl list
# ✓ software-project-team@1.2.1    13 agents, 3 skills
# ✓ design-system-team@0.8.1        2 agents, 10 skills (dst-*)
```

The two are designed to complement each other: design with `/dst-*` skills, implement with software-project-team agents (flutter-agent, react-agent, etc.).

## Contributing a team

Want to publish your own team? See the [team authoring guide](/authoring/creating-a-team) — write a `team.json`, push to a public git repo, and submit a PR to the [registry](https://github.com/agentteamland/registry).
