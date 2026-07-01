# CLI Overview

`atl` installs agent teams, keeps them updated, circulates the gains your agents learn, and runs itself in the background so you can focus on your project.

Commands fall into three groups: **team commands** you run by hand, the **gain-circulation** ring that promotes and shares what your agents learn, and **automation** that Claude Code triggers for you. Everything operates on the **current project** (the directory you ran `atl` in) unless noted, with a second **user-global** layer that the project shadows (nearest wins).

## Team commands

| Command | What it does |
|---|---|
| [`atl install`](/cli/install) | Install a team by handle (resolved against the GitHub-backed index) into the current scope. |
| [`atl list`](/cli/list) | Show teams installed in this project. |
| [`atl remove`](/cli/remove) | Uninstall a team. |
| [`atl update`](/cli/update) | Pull latest for one or all installed teams. |
| [`atl search`](/cli/search) | Search the team catalog (the GitHub-backed index). |
| [`atl gc`](/cli/gc) | Reclaim orphaned assets no manifest owns — the reversible inverse of install (dry-run by default; soft-delete + undo). |

## Gain-circulation commands

As your agents work, they accumulate **gains** — new learnings, sharpened skills, project-local rules. These commands move those gains outward through the three-ring ladder.

| Command | What it does |
|---|---|
| `atl promote` | Lift project-local gains to the user-global layer (so every project benefits). |
| `atl publish` | Publish your global-layer gains — re-publish your own team, or propose them upstream as a GitHub PR. |
| `atl pin` | Keep a project-local path from being promoted to global. |
| `atl unpin` | Allow a previously pinned path to be promoted again. |
| `atl learnings` | Inspect the durable learning queue: `status` (pending per channel/project), `peek` (list items, used by the `/drain` skill), `ack <id>` (mark an item processed). |

## Automation commands

These are wired to Claude Code hooks by [`atl setup-hooks`](/cli/setup-hooks) and run **unattended** — you normally never type them. They are listed here only so you recognize them in hook output or reach for them when troubleshooting.

| Command | What it does |
|---|---|
| [`atl setup-hooks`](/cli/setup-hooks) | One-time install/remove of the Claude Code hooks (`SessionStart`, `UserPromptSubmit`) that drive the automation below. |
| `atl session-start` | Boot-time maintenance run by the `SessionStart` hook (cache refresh + auto-update + previous-transcript marker scan + self-version check). |
| `atl tick` | The in-session maintenance tick (every 5–10 minutes via prompt-piggyback): drains throttled background work. |
| `atl doctor` | The self-heal daemon — diagnoses drift and repairs the install automatically. |

> Compared with v1, there is **no `config`, `migrate`, or `learning-capture` command.** Learning capture is now automatic (markers land in a durable queue that `atl learnings` inspects and the `/drain` skill processes); configuration and state-file migration are not part of the v2 surface.

## Companion skills

`atl` is the deterministic half of the platform; the judgment-heavy half lives in Claude Code skills the teams install:

- **`/drain`** — process the learning queue into agent knowledge bases (the v1 `/save-learnings`).
- **`/create-pr`** — branch → review → commit → PR.
- **`/create-code-diagram`** — generate an architecture/class diagram of the codebase.
- **`/brainstorm`**, **`/rule`**, **`/rule-wizard`** — design, author, and scaffold rules.

The split is deliberate: the **CLI is deterministic** (same inputs, same result, no LLM), **skills are LLM-driven** (they reason about your specific code).

## Global flags

| Flag | Effect |
|---|---|
| `--help`, `-h` | Print usage and exit. |
| `--version`, `-v` | Print the installed `atl` version. |

Each command has its own `--help` page: `atl install --help`, `atl publish --help`, and so on.

## State `atl` keeps

Assets live in **Claude Code's own directories**, in one of two scopes — there is no separate ATL-owned asset store:

```
~/.claude/                 ← user-global layer (agents/skills/rules shared across all projects)
<project>/.claude/         ← project layer (shadows global; nearest wins)
```

`atl`'s own bookkeeping lives under `~/.atl/` (user-global) and `<project>/.atl/` (per-project):

```
~/.atl/
├── queue.db               ← the durable learning queue (bbolt)
├── index.json             ← cached team catalog (refreshed by atl update)
├── generation             ← global-layer change counter (drives every-prompt fan-out)
├── pins.json              ← paths held back from promotion
├── cache/                 ← cache stamps
└── installed/             ← per-team install manifests + integrity baselines
```

## Philosophy

- **Deterministic.** Same inputs, same result. No hidden state.
- **Observable.** Every action prints what it did. Use the output, not a spinner.
- **Hands-off.** The automation commands keep things current without you thinking about them.

## Next

- **[`atl install`](/cli/install)** — the command you'll run most.
- **[`atl search`](/cli/search)** — discover what's in the catalog.
