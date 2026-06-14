# atl — AgentTeamLand

The AgentTeamLand platform: install agent teams, keep them updated, circulate the gains your agents learn, and let the platform run itself in the background so you can focus on your project.

> **Status:** v2 is **alpha** — the first public pre-release, `v2.0.0-alpha.1`, is out (macOS / Linux / Windows). The CLI spine, the learning loop, gain circulation, and install/update are live; not yet at full v1 feature parity.

## Install

**macOS / Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Then, in a project:

```bash
atl install agentteamland/software-project-team   # install a team
atl list                                          # see what's installed
```

## Commands

| Command | What it does |
|---|---|
| `install <handle>/<team>` | install a team from the index (`--global` / `--project`; project shadows global) |
| `list` / `remove` | list installed teams / remove one |
| `update` | upgrade teams + fan global gains out to project copies (pull, never push) |
| `promote [team]` | lift a project's accumulated gains to your global layer (ring 1→2) |
| `pin` / `unpin` | keep a path project-only, opting it out of promote |
| `publish <team>` | propose your gains upstream / re-publish a team you own (ring 2→3) — *plan ready; apply lands next* |
| `doctor` | automatic self-heal (queue health, hooks, asset integrity) |
| `learnings` | inspect / drain the durable learning queue |
| `tick` / `session-start` | the automatic in-session cadence (run by hooks) |

## How it works

- **Scope axis** — every team lives at a global (`~/.claude`) or project (`<project>/.claude`) layer; the nearest wins (project shadows global). Core rules + skills ship inside the binary and reflect into `~/.claude`.
- **Learning loop** — silent `<!-- learning: … -->` markers you drop in conversation are transferred into a durable bbolt queue **exactly once**, then the `/drain` skill folds each into the knowledge base and deletes it. Processed-then-deleted, so the v1 re-report bug class is structurally gone.
- **Gain circulation** — `promote` lifts a project's gains to your global layer; on `update`, pull-based fan-out distributes global gains to your *other* projects (modified files preserved); `publish` proposes them upstream. Your own world circulates automatically; cross-author sharing stays a consenting handshake.
- **Self-running** — `atl install` binds hooks (automation is mandatory, not opt-in) that run a three-speed in-session cadence: per-prompt fan-out + a throttled drain + doctor self-heal. The platform maintains itself so you focus on the project.

## Monorepo layout

| Path | What |
|---|---|
| `cli/` | the `atl` binary (Go) — the deterministic plumbing layer |
| `core/` | global rules + skills, shipped in the binary (reflected to `~/.claude`) |
| `teams/` | first-party teams (`software-project-team`, `design-system-team`) |
| `docs/` | VitePress docs site (EN + TR) |
| `.atl/` | repo-local architecture reference |

## Why v2

See [`.atl/docs/atl-v2.md`](.atl/docs/atl-v2.md) for the full platform-restructure decision: monorepo consolidation (15 repos → 2), a first-class global/project scope axis, a bbolt durable learning queue that kills the re-report bug class, three-ring gain circulation, GitHub-backed self-serve publish, and an automation-mandatory reliability layer (doctor as an automatic self-heal daemon).

## License

MIT — see [LICENSE](LICENSE).
