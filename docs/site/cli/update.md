# `atl update`

Pull the latest version of one or all installed / cached agentteamland repos, **and** auto-refresh project-local copies that haven't been locally modified.

## Usage

```bash
atl update                          # update every cached repo + auto-refresh unmodified copies
atl update <team>                   # update just one team's chain (legacy)
atl update --silent-if-clean        # no output if nothing changed (used by hooks)
atl update --check-only             # dry-run: report what WOULD update, pull nothing
atl update --throttle=30m           # skip if last successful run was <30m ago
atl update --skip-self-check        # don't check for a newer atl release
atl update -v                       # verbose (print each git command)
```

## What it updates

With no team name, `atl update` performs three steps:

1. **Cache pull.** Iterates every git repo under `~/.claude/repos/agentteamland/` (`core`, `brainstorm`, `rule`, `team-manager`, every installed team) — `git fetch origin main` → fast-forward `git pull` if behind → no-op if current.
2. **Silent symlink → copy migration (atl ≥ 1.0.0).** Per-project, any pre-v1.0.0 symlinks under `.claude/agents/` and `.claude/rules/` that point into the global cache are replaced with project-local copies. Surfaced as a single info line per project; non-destructive (the symlink target is read first, then the symlink is replaced with a real file containing the same content).
3. **Auto-refresh of unmodified copies (atl ≥ 1.0.0).** For each installed team, every project-local copy of an agent/rule/skill resource is checked via three-way SHA-256 comparison:
   - **install-time baseline** (recorded in `.team-installs.json` at install)
   - **current project copy** (what's in `.claude/...` right now)
   - **current cache content** (what was just pulled)

   When `current project = baseline ≠ cache`, the resource is unmodified locally → silently overwritten with the new cache content. When `current project ≠ baseline`, the user has edited it → skipped, with a per-team hint pointing at `atl install <team> --refresh` for explicit force-overwrite.

It also checks for a newer `atl` binary release (GitHub Releases API, throttled to 24h):

```
⬆  atl 1.1.1 → 1.1.2 available — run: brew upgrade atl
```

The binary is NOT auto-upgraded — the message points at the right package-manager command based on how you installed atl.

## Example — silent-if-clean (used by hooks)

Nothing changed:

```bash
$ atl update --silent-if-clean
$                               # zero output, exit 0
```

Something changed:

```bash
$ atl update --silent-if-clean
🔄 software-project-team 1.2.0 → 1.2.1 (auto-updated)
🔄 core 1.8.0 → 1.9.0 (auto-updated)
   ↪ refreshed 14 unmodified copies in current project
```

## Example — dry-run

```bash
$ atl update --check-only
🔄 software-project-team 1.2.0 → 1.2.1 (auto-updated)
   ↪ would refresh 14 unmodified copies in current project
   ↪ would skip 2 modified copies (run: atl install software-project-team --refresh)
```

Prints what would update; no git pull executed, no copies touched.

## Automatic updates via hooks (recommended)

Set up once, forget forever:

```bash
atl setup-hooks                 # default: UserPromptSubmit throttled to 30m
atl setup-hooks --throttle=5m   # more aggressive
atl setup-hooks --remove        # disable
```

This installs two Claude Code hooks in `~/.claude/settings.json`:

- `SessionStart` → `atl session-start --silent-if-clean` (composite: update + previous-transcript marker scan + atl self-check)
- `UserPromptSubmit` → `atl update --silent-if-clean --throttle=30m` (per-message refresh, throttled)

When Claude Code starts a session or you send a prompt, the hook silently refreshes every cached repo + auto-refreshes unmodified project copies. If something changed, Claude sees a single `🔄` line and acts on the update. If nothing changed, the hook returns instantly (~1ms, just a file-stat check).

On first `atl install`, you're asked whether to enable this. Say yes; you can reverse any time with `atl setup-hooks --remove`.

See [`atl setup-hooks`](/cli/setup-hooks) for details.

## Version constraints (per-team) still honored

If you installed `software-project-team@^1.0.0`, `atl update` pulls up to the latest `1.x.x` — **not** `2.0.0`. Major bumps require explicit `atl install software-project-team@^2.0.0`.

## Throttle internals

Two timestamp files live at:

- `~/.claude/cache/atl-last-repo-check` — repo-fetch throttle
- `~/.claude/cache/atl-last-self-check` — atl-release-API throttle (24h, fixed)

`--throttle=<dur>` checks the repo-fetch timestamp. If the file's modified time is within `<dur>`, the repo scan is skipped entirely (fast path ~1ms). Otherwise, the scan runs, stamps the file on success. On failure (e.g., offline), the stamp is NOT updated, so the next call retries.

## Offline behavior

If the network is unreachable, individual `git fetch` calls fail silently and that repo is reported as `⚠ <name>: fetch: <error>` (not silenced by `--silent-if-clean` — you want to know). The rest of the check continues. Project copies are never touched on offline runs.

## Related

- [`atl install`](/cli/install) — first install (with opt-in auto-update prompt)
- [`atl install <team> --refresh`](/cli/install#idempotency-atl-v100) — explicit force-overwrite for a project (when auto-refresh skipped you due to local mods)
- [`atl setup-hooks`](/cli/setup-hooks) — configure the hooks manually
- [`atl list`](/cli/list) — see what's installed
- [Version constraints](/authoring/team-json#version-constraints) — how `^`, `~`, exact pins resolve
