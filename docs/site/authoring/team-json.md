# `team.json`

Every team is a Git repository with a `team.json` at its root. That file is the entire contract: what the team is called, what it ships, what it depends on, and where it installs by default.

## Minimal example

```json
{
  "schemaVersion": 1,
  "name": "my-team",
  "version": "0.1.0",
  "description": "A starter team for small Next.js projects.",
  "author": { "name": "Your Name", "url": "https://github.com/you" },
  "agents": [
    { "name": "web-agent", "description": "Next.js + Tailwind reviewer and builder." }
  ]
}
```

That's enough to install. The CLI parses the manifest, copies `agents/web-agent.md` (or `agents/web-agent/agent.md`) into the scope's `.claude/agents/`, and records the install in a per-scope manifest.

## Full field reference

| Field | Type | Required | Description |
|---|---|---|---|
| `schemaVersion` | integer | ✅ | Currently `1`. Bumped only on a breaking change to the manifest shape. |
| `name` | string | ✅ | The team's catalog name. Lowercase kebab-case. Combined with your GitHub handle it forms the install ref `<handle>/<name>`. |
| `version` | semver string | ✅ | SemVer 2.0.0 (`1.2.3`, `1.2.3-beta.1`). `atl update` compares this to decide whether to pull. |
| `description` | string | ✅ | One-sentence pitch shown in `atl search`. Keep it tight — it's a single line in catalog output. |
| `author` | object | — | Optional metadata the install parser does not currently read. If provided, an object `{ "name": "...", "url": "...", "email": "..." }` is the conventional shape; a plain string is accepted (silently ignored), not rejected. |
| `license` | SPDX string | — | `"MIT"`, `"Apache-2.0"`, etc. Defaults to `"MIT"` if omitted. |
| `keywords` | string[] | — | For `atl search` matching. `["nextjs", "tailwind", "blog"]`. |
| `repository` | string | — | The team's source URL, surfaced in the catalog. |
| `homepage` | string | — | Docs / landing URL. |
| `agents` | object[] | — | Each: `{ name, description }`. Names must match files/directories under `agents/`. |
| `skills` | object[] | — | Each: `{ name, description }`. Names must match directories under `skills/`. |
| `rules` | object[] | — | Each: `{ name, description }`. Names must match files under `rules/`. |
| `scope` | string | — | Publisher-default install layer: `"project"`, `"global"`, or `"both"`. Defaults to `"project"`. The user can always override at install time with `--global` / `--project`. |
| `dependencies` | object | — | Map of `team-name → version-constraint` for other teams the CLI installs alongside this one. |
| `requires.atl` | string | — | Minimum `atl` version, e.g. `">=2.0.0"`. |

::: tip Keep the description short
`description` is rendered as a single line in `atl search` output, so a long one wraps awkwardly. Aim for one tight sentence — it's a pitch, not a paragraph.
:::

## Version constraints

The `dependencies` map and `requires.atl` accept standard SemVer range syntax:

| Syntax | Meaning |
|---|---|
| `^1.2.3` | `>=1.2.3 <2.0.0` (caret — default recommended) |
| `~1.2.3` | `>=1.2.3 <1.3.0` (tilde) |
| `1.2.3` | Exact pin |
| `>=1.2.0` | Open-ended minimum |

Caret (`^`) is the default recommendation — it gets patch and minor updates, blocks breaking major bumps.

## Directory conventions

`atl` discovers your bundled files by reading `team.json` and looking for matching paths under `agents/`, `skills/`, and `rules/`:

```
my-team/
├── team.json
├── agents/
│   ├── web-agent.md             ← simple agent (single file)
│   └── db-agent/
│       ├── agent.md             ← complex agent (children pattern)
│       └── children/
│           ├── migrations.md
│           └── rls.md
├── skills/
│   └── create-new-project/
│       └── SKILL.md
└── rules/
    └── commit-style.md
```

The installable asset directories are `agents/`, `skills/`, `rules/`, `knowledge/`, `scripts/`, and `packs/` (the `teampkg.AssetDirs` set). `agents/`/`skills/`/`rules/` are what Claude Code reads directly; `knowledge/`/`scripts/`/`packs/` carry a team's runtime reference docs, helper scripts, and area packs. Everything else (`team.json`, `README`, `LICENSE`) stays behind.

A team must ship at least one file under an asset directory or `atl install` fails (`team ships no installable assets`). Individual declared `agents[]`/`skills[]`/`rules[]` entries are catalog metadata and are not validated against disk at install time — the `atl skills check` dev command cross-checks the declared `agents[]` and `skills[]` for first-party teams.

## Validation

There is no separate JSON Schema file and no schema-validation CI step in v2. Validation is minimal and lives in the CLI itself:

- `team.json` must parse as JSON.
- It must have a `name`.
- The team must ship at least one file under an asset directory — `atl install` errors if a team ships no installable assets.

That's the whole contract. If `atl install` accepts your team, it's valid; there's nothing else to run locally or in CI.

## Next

- **[Creating a team](./creating-a-team)** — step by step.
- **[`atl install`](/cli/install)** — how a team is resolved and installed.
