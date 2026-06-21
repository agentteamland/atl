# `atl update`

Refresh installed teams: pull a fresh catalog index, upgrade any team that has a newer published version, fan the global layer's gains out to your project copies, and reflect the latest platform core into the global layer — always preserving files you've edited locally (pull, never push).

`atl update` is the **manual** surface for the network refresh. The day-to-day work happens automatically through the [in-session cadence](#automatic-updates-the-in-session-cadence); you only reach for this command to force a pass, or after installing the binary by hand.

## Usage

```bash
atl update
```

It takes no arguments and no flags. It always operates on the **current project** (the directory you run it in) plus the **global** layer.

## What it does

`atl update` runs four steps, in order:

1. **Refresh the index cache.** Best-effort network fetch of the catalog (the GitHub-backed team index) into `~/.atl/index.json`. If you're offline the fetch fails quietly and resolution falls back to the cached or embedded index — being offline is fine; nothing else is blocked.
2. **Upgrade teams to newer published versions.** For every team installed at the **project** and **global** layers, `atl update` looks the team up in the resolved index. If the published version is newer than what's installed, it re-fetches the team's source as an ephemeral HTTPS tarball, extracts it, and reflects it onto your installed copy under [fan-out discipline](#fan-out-discipline-how-your-edits-survive): unmodified files refresh to the new version, files you edited are preserved, brand-new files in the release are added. The install manifest is then rewritten at the new version. Teams not present in the index (e.g. a local one) are left alone.
3. **Fan out global gains to the project.** For every team installed at **both** the global and project layers, each project-local file is compared three ways (see below). Unmodified project copies refresh from the global copy; files you edited locally are kept. This is how a gain promoted into the global layer reaches your projects.
4. **Reflect platform core into the global layer.** The core rules and skills ship inside the `atl` binary, so this refreshes them into `~/.claude`, keeping the global layer in lockstep with your binary version.

After upgrading or fanning out, it also surfaces a one-line suggestion for any **globally**-installed team whose global copy has gains not yet pushed upstream (see [Publish suggestions](#publish-suggestions)).

### Output

The summary line reflects what happened:

```text
atl update: upgraded 1 team(s), refreshed 14 file(s) from global
```

```text
atl update: upgraded 1 team(s)
```

```text
atl update: refreshed 14 file(s) from the global layer
```

When nothing was outstanding:

```text
atl update: everything up to date
```

If core files changed, a separate line precedes the summary:

```text
atl update: refreshed 3 core file(s)
```

## Fan-out discipline — how your edits survive

Both the version upgrade (step 2) and the global→project fan-out (step 3) decide each file with the same three-way SHA-256 comparison, against the hash the file had **at install time** (recorded in the [install manifest](#the-install-manifest)):

| Comparison | Meaning | Action |
|---|---|---|
| local **=** upstream | already current | nothing to do |
| local **=** install baseline | you never touched it | **refresh** to the upstream/global version |
| local **≠** baseline | you edited it | **preserve** — your copy is kept |

"Modified" means "diverged from what we installed", not merely "differs from upstream". A file you never changed is refreshed; a file you changed is never silently overwritten. When a copy is refreshed, its baseline advances to the new content, so the next pass starts clean.

There is no force-overwrite flag. To deliberately discard local edits and take the published version, remove and reinstall the team:

```bash
atl remove <handle>/<team>
atl install <handle>/<team>
```

## The install manifest

The baseline that fan-out compares against lives in the team's **install manifest** — one JSON file per team per scope at:

- `~/.atl/installed/<handle>__<name>.json` (global)
- `<project>/.atl/installed/<handle>__<name>.json` (project)

Each manifest records `schemaVersion`, `handle`, `name`, `version`, `scope`, the `source` it was fetched from (`repo`, `subpath`, `ref`), `installedAt`, and a `files` map of each installed path to its SHA-256 at install time. `atl update` reads this map to tell "unmodified" from "edited", and rewrites it (advancing version, source ref, and baseline hashes) whenever it changes a team.

## Automatic updates — the in-session cadence

You rarely run `atl update` by hand because ATL keeps things current automatically. [`atl setup-hooks`](/cli/setup-hooks) (run as a mandatory part of [`atl install`](/cli/install)) wires two Claude Code hooks:

- `SessionStart` → [`atl session-start`](/cli/setup-hooks) — drains the previous session's learnings, runs the doctor self-check, and reflects platform core into the global layer.
- `UserPromptSubmit` → [`atl tick --throttle=10m`](/cli/tick) — a cheap per-prompt **fan-out** (global→project), plus a throttled drain + doctor + promote pass.

The per-prompt [`atl tick`](/cli/tick) handles the local fan-out continuously, so gains promoted into your global layer reach your projects without you doing anything. `atl update` adds the **network** half — re-resolving the index and pulling newer *published* team versions — which is the heavier pass you run manually (or whenever you want to check for new releases now).

## Publish suggestions

After it finishes, `atl update` checks every **globally**-installed team for gains in your global copy that aren't in its published version yet, and prints a nudge per team:

```text
atl update: gains in <handle>/<team> not yet upstream (3 file(s)) — run `atl publish <handle>/<team>` to contribute them
```

This is a suggestion only — nothing is published automatically. Publishing stays an explicit, consent-gated act; see [`atl publish`](/cli/publish). The check is best-effort and silent if a team's published source can't be fetched.

## Offline behavior

`atl update` degrades gracefully offline. The index refresh and any tarball fetch fail quietly, resolution falls back to the cached/embedded index, and teams that can't be re-fetched are simply not upgraded. The local global→project fan-out (step 3) needs no network and still runs.

## Related

- [`atl install`](/cli/install) — first install of a team
- [`atl tick`](/cli/tick) — the per-prompt fan-out + drain pass that keeps projects current automatically
- [`atl setup-hooks`](/cli/setup-hooks) — wire the automation hooks
- [`atl promote`](/cli/promote) — lift a project's gains up to the global layer (the source of what fan-out distributes)
- [`atl publish`](/cli/publish) — contribute global gains back upstream
- [`atl list`](/cli/list) — see what's installed and at which scope
