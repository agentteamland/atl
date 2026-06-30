# `atl install`

Resolve a team from the GitHub-backed catalog and install it into a scope.

## Usage

```bash
atl install <handle>/<team>              # install at the publisher's default scope
atl install <handle>/<team> --global     # force user-global scope (every project)
atl install <handle>/<team> --project    # force project scope (this project only)
```

`<handle>/<team>` is the catalog reference — the handle is the team's GitHub owner (ownership is authorship) and the team is its name within that handle. There is no `@version` pin, no Git URL, no local filesystem path: install always resolves through the catalog. Find a reference with [`atl search`](/cli/search).

```bash
atl install agentteamland/software-project-team
```

`--global` and `--project` are mutually exclusive. With neither flag, the team installs at the scope its publisher declared (see [Scope](#scope) below).

## What happens

1. **Resolve.** The `<handle>/<team>` reference is looked up in the GitHub-backed catalog — generated from public repos tagged with the [`atl-team`](https://github.com/topics/atl-team) topic. The catalog resolves offline-first from `~/.atl/index.json` (the out-of-band-refreshed cache), falling back to the copy embedded in the binary, so the lookup never blocks on the network. A first-party team resolves to a subpath inside the `agentteamland/atl` monorepo; a standalone team resolves to its own repo root.
2. **Fetch.** The team's source is downloaded as a single ref-pinned HTTPS tarball straight from GitHub — **no `git` binary required** — into a temporary directory that is deleted as soon as the install finishes. There is no persistent clone cache on disk.
3. **Read.** `team.json` is parsed. Validation is minimal: it must be valid JSON and have a `name`. There is no JSON-Schema validator — see [Schema](/reference/schema) for exactly what the CLI checks and [`team.json`](/authoring/team-json) for the full field contract.
4. **Write assets.** The team's `agents/`, `skills/`, and `rules/` subtrees are copied straight into the scope's Claude Code directory — `~/.claude` for a global install, `<project>/.claude` for a project install. Nothing else from the repo (`team.json`, `README`, `LICENSE`) is copied.
5. **Record a manifest.** A per-team manifest is written under the scope's `.atl/` directory at `<layer>/.atl/installed/<handle>__<name>.json`. It records the resolved source ref and a SHA-256 baseline for every copied file, which `atl update`'s refresh and `atl doctor`'s integrity check rely on.
6. **Bind automation hooks.** The automation hooks (`SessionStart → atl session-start`, `UserPromptSubmit → atl tick`, and `PreToolUse (Bash|Edit|Write) → atl guard`) are installed into Claude Code as a mandatory part of install — automation is on by default, not opt-in. A hook-binding failure is surfaced as a warning and does not fail the install.
7. **Reflect platform core.** The platform's own rules and skills (`/drain`, `/create-pr`, `/brainstorm`, and the rest) ship inside the binary and are reflected into the global layer so they're available in every project. Best-effort — a failure is surfaced, not fatal.
8. **Scaffold a project `CLAUDE.md`.** If the project has no `CLAUDE.md`, install drops the project-tier starter (see [`atl init`](/cli/init)) so the `/brainstorm` and `/drain` blocks have a home. Only-if-absent — a file you already have is never touched — and best-effort, so a failure never fails the install.

On success the CLI prints:

```
atl: installed <handle>/<name>@<version> at <scope> scope
```

where `<scope>` is `global`, `project`, or `both`.

## Scope

A team lives at one of two layers:

- **global** — assets under `~/.claude`, ATL state under `~/.atl`. Available in every project on the machine.
- **project** — assets under `<project>/.claude`, ATL state under `<project>/.atl`. Available only in that project.

Each team's publisher declares a default scope in `team.json` — `project` (the default), `global`, or `both`. You override it at install time with `--global` or `--project`; the override always wins. A `both` install writes to **both** layers.

When the same capability exists at both layers, the **project layer shadows global** — nearest wins, the same mental model as Claude Code's own `CLAUDE.md` layering. See [Concepts](/guide/concepts#scope-global-and-project) for the full scope axis.

```bash
atl install agentteamland/software-project-team            # publisher default (project)
atl install agentteamland/software-project-team --global   # every project on this machine
```

## The install manifest

Each install records one JSON file per team per scope at `<layer>/.atl/installed/<handle>__<name>.json` (`<layer>` is `~/.atl` for global, `<project>/.atl` for project). It captures:

- `schemaVersion`, `handle`, `name`, `version`, and the effective `scope`,
- `source` — the `repo`, `subpath`, and `ref` the install resolved from, pinned to the exact bytes fetched,
- `installedAt`,
- `files` — a map of every copied file (path relative to the `.claude` directory) to its SHA-256 at install time.

That `files` map is a dual-purpose record: `atl update` compares the current bytes against it to tell your edits apart from upstream changes (so updates never clobber your local modifications), and `atl doctor` uses it as the integrity set to detect and restore a deleted or corrupted copy.

## Multiple teams in one project

Several teams can coexist in one project — each install copies its assets into the same `.claude/` directory and writes its own manifest. If two installed teams ship an asset with the same name, the most recently written copy is the one on disk. Remove a team with [`atl remove`](/cli/remove); it deletes only that team's manifest-recorded files.

A team can also declare other teams as `dependencies` in its `team.json`; those are installed alongside it. See [`team.json`](/authoring/team-json) for the dependency field.

## Troubleshooting

- **`team … not found in index`** — the reference isn't in the catalog. Check it with [`atl search`](/cli/search). The catalog is generated from public repos tagged [`atl-team`](https://github.com/topics/atl-team), so a brand-new team may not be listed yet.
- **`invalid team reference`** — the argument isn't in `<handle>/<team>` form (both parts required).
- **`fetch … HTTP 404`** — the team's source repo or ref is unreachable. Tarball fetch needs the network; unlike catalog resolution it has no offline fallback.
- **`team ships no installable assets`** — the resolved team has no `agents/`, `skills/`, or `rules/` directory.
- **`team.json has no name`** — the team's `team.json` is malformed. Ask the author to fix it.

## Related

- [`atl search`](/cli/search) — find a team's `<handle>/<team>` reference.
- [`atl list`](/cli/list) — see what's installed, by scope.
- [`atl update`](/cli/update) — refresh installed teams; unmodified copies update in place, local edits are kept.
- [`atl remove`](/cli/remove) — uninstall a team from a scope.
- [`atl setup-hooks`](/cli/setup-hooks) — the automation hooks install binds for you.
- [Concepts](/guide/concepts#scope-global-and-project) — the global/project scope axis.
