# CLI Overview

`atl` has eight commands. The five **user commands** operate on the **current project** (the directory you ran `atl` in) unless otherwise noted. The three **automation commands** are typically wired to Claude Code hooks and run unattended — you'll usually only invoke them manually for setup or troubleshooting.

## User commands

| Command | What it does |
|---|---|
| [`atl install`](/cli/install) | Install a team by registry name or Git URL. |
| [`atl list`](/cli/list) | Show teams installed in this project. |
| [`atl remove`](/cli/remove) | Uninstall a team. |
| [`atl update`](/cli/update) | Pull latest for one or all installed teams. |
| [`atl search`](/cli/search) | Search the public registry. |

## Automation commands

| Command | What it does |
|---|---|
| [`atl setup-hooks`](/cli/setup-hooks) | One-time install/remove of Claude Code hooks (`SessionStart`, `UserPromptSubmit`) that wire up auto-update + learning capture. |
| `atl session-start` | Composite boot-time wrapper invoked by the `SessionStart` hook (cache pull + symlink→copy migration + auto-refresh + previous-transcript marker scan + atl self-version check). Not normally run by hand. |
| [`atl learning-capture`](/cli/learning-capture) | Scan Claude Code transcripts for `<!-- learning -->` markers. Invoked by the `SessionStart` wrapper; can also be run manually for testing or backfill. |

## Global flags

| Flag | Effect |
|---|---|
| `--help`, `-h` | Print usage and exit. |
| `--version`, `-v` | Print the installed `atl` version. |

Each command has its own `--help` page: `atl install --help`, `atl search --help`, and so on.

## State `atl` keeps

**Shared cache** (one per machine):

```
~/.claude/repos/agentteamland/
└── <team-name>/          ← cloned Git repo, reused across all projects
```

**Per-project state** (per directory you run `atl` in):

```
<project>/
└── .claude/
    ├── .team-installs.json       ← which teams are installed, at what versions
    ├── agents/                   ← copies to team-provided agents
    ├── skills/                   ← copies to team-provided skills
    ├── rules/                    ← copies to team-provided rules
    └── ...
```

The copies point into the shared cache. That's why `atl update` takes effect in every project at once: you update the cache, not the project.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success. |
| `1` | General error (invalid args, team not found, network). |
| `2` | Validation failure (`team.json` didn't match the schema). |
| `3` | Inheritance error (circular chain, missing parent). |

## Philosophy

- **Deterministic.** Same inputs, same copies. No hidden state.
- **Idempotent.** Re-running `atl install` on an already-installed team is a no-op (or a pull).
- **Observable.** Every action prints what it did. Use the output, not a spinner.

## Next

- **[`atl install`](/cli/install)** — the command you'll run most.
- **[`atl search`](/cli/search)** — discover what's in the registry.
