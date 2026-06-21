# First-Party Teams

AgentTeamLand publishes **2 first-party teams**. Each is independently versioned and follows the [`team.json` contract](/authoring/team-json). Both are discoverable through the team catalog and installable by reference.

> The catalog is generated from public GitHub repositories tagged with the [`atl-team`](https://github.com/topics/atl-team) topic. There is no registry to submit to — tagging the repo (or running [`atl publish`](/cli/publish)) is what lists a team.

## Browse

| Team | Version | Description |
|------|---------|-------------|
| [`software-project-team`](/teams/software-project-team) | 1.2.1 | 13 specialized agents for full-stack software projects (.NET 9 + Flutter + React + Postgres + RabbitMQ + Redis + Elasticsearch + MinIO). Phase 2.C: agent KB sections auto-rebuilt from children frontmatter. |
| [`design-system-team`](/teams/design-system-team) | 0.8.1 | Design systems and UI prototypes inside any project — local, file-based, browser-viewable. `/dst-*` skills produce JSON state and Tailwind-rendered HTML pages under `.dst/`. |

Both ship under the `agentteamland/` handle, so they carry the **`[verified]`** badge in [`atl search`](/cli/search). The badge marks teams reviewed by AgentTeamLand maintainers (`agentteamland/*` plus a maintainer allowlist); it is not a status field on the team, and its absence on a self-published team does not mean the team is unsafe.

## Install any team

```bash
atl install agentteamland/software-project-team
```

Teams install by `<handle>/<name>` reference. [`atl install`](/cli/install) resolves the ref against the GitHub-backed catalog, fetches the source as an ephemeral tarball over HTTPS, and copies the team's `agents/`, `skills/`, and `rules/` into the scope's `.claude/` directory:

```
atl: installed agentteamland/software-project-team@1.2.1 at project scope
```

By default a team installs at the scope its publisher declares (project, global, or both). Pass `--global` or `--project` to override. See [scopes](/guide/concepts#scope-global-and-project) for how the two layers interact.

## Install multiple teams in one project

Both teams coexist cleanly in the same project. When two teams declare an asset with the same name, the most recently installed one wins and `atl` prints a one-line collision warning:

```bash
cd your-project
atl install agentteamland/software-project-team    # full-stack agents + scaffolder
atl install agentteamland/design-system-team       # add design-system + prototype tooling

atl list
# project:
#   agentteamland/software-project-team@1.2.1
#   agentteamland/design-system-team@0.8.1
```

The two are designed to complement each other: design with `/dst-*` skills, implement with software-project-team agents (flutter-agent, react-agent, etc.).

## Contributing a team

Want to publish your own team? See the [team authoring guide](/authoring/creating-a-team) — write a `team.json`, push it to a public GitHub repo, then tag that repo with the `atl-team` topic or run [`atl publish`](/cli/publish). The catalog picks it up from there; no PR submission needed.
