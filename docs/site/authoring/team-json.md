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
| `license` | SPDX string | — | `"MIT"`, `"Apache-2.0"`, etc. Conventional metadata — the CLI and the catalog do not read it. Ship a LICENSE file in the repo alongside it. |
| `keywords` | string[] | — | For `atl search` matching. `["nextjs", "tailwind", "blog"]`. |
| `repository` | string | — | The team's source URL. Conventional metadata — the catalog derives the source repo from the discovered GitHub repo itself, not from this field. |
| `homepage` | string | — | Docs / landing URL. |
| `agents` | object[] | — | Each: `{ name, description }`. Names must match files/directories under `agents/`. |
| `skills` | object[] | — | Each: `{ name, description }`. Names must match directories under `skills/`. |
| `rules` | object[] | — | Each: `{ name, description }`. Names must match files under `rules/`. |
| `scope` | string | — | Publisher-default install layer: `"project"`, `"global"`, or `"both"`. Defaults to `"project"`. The user can always override at install time with `--global` / `--project`. |
| `dependencies` | object | — | Map of `team-name → version-constraint` for other teams the CLI installs alongside this one. |
| `requires.atl` | string | — | Declared minimum `atl` version, e.g. `">=2.0.0"`. Conventional metadata — the install parser does not currently enforce it. |
| `capabilities` | object | — | Optional contracts, mostly read by the platform's skills rather than the install parser. `capabilities.review: "<agent>"` names the agent [`/create-pr`](/skills/create-pr) spawns as this team's specialist reviewer; `capabilities.profile` declares the profile-layer provider/consumer role (see [profile-team](/teams/profile-team)). The one key the **CLI** reads is `store` — see below. |
| `backends` | string[] | — | For teams shipping per-backend adapter packs under `backends/<name>/` (e.g. the delivery-team's `["azure", "github"]`): declares which backends the team supports. Informational today — the install parser does not read it. |

::: tip Keep the description short
`description` is rendered as a single line in `atl search` output, so a long one wraps awkwardly. Aim for one tight sentence — it's a pitch, not a paragraph.
:::

## Declaring a durable store

Most teams keep everything inside the reflected `.claude` tree, which ATL already tracks. A team that instead keeps its own long-lived data somewhere else — profile-team's `~/.atl/profiles/` is the first — declares that location:

```json
{
  "capabilities": {
    "profile": { "role": "provider", "store": "~/.atl/profiles" }
  }
}
```

`store` may sit under any capability name, holds one path, and may use `~` for the home directory. `atl install` records every declared store into the install manifest, and from then on **session-start and `atl tick` keep that directory under local git** — once per session and once per tick throttle window — committing whatever changed since the last pass.

The point is recoverability. A store whose write policy is last-write-wins loses the previous value on every overwrite: an inferred placeholder replaced by a confirmed answer leaves no trace of what it replaced. With the directory versioned, the old value is one `git show HEAD~1` away.

::: tip Already installed before this shipped?
The declaration is read at install time, so an install that predates the `stores` field carries no record of it. `atl update` backfills that once, by re-fetching the pinned source — and it runs on its own, so this resolves without you doing anything. Until it does, the store simply is not versioned yet.
:::

What this deliberately does **not** do:

- **It never creates the directory.** An absent store means the feature is not in use on this machine; creating it would litter the disk and misreport the feature as active.
- **It never configures a remote and never pushes.** A store tends to hold a user's most sensitive data; carrying a copy off the machine is a separate act that the user has to ask for (profile-team's [`/profile-backup`](/teams/profile-team) is one such path, and it refuses to write into a public repo).
- **It only ever writes to a repo it created itself.** If you already keep the store under your own version control, ATL leaves it strictly alone — you have met the goal yourself, and advancing your branch would move `HEAD` under your in-flight work. ATL marks its own repos with a file at `.git/atl-store`; delete that file and it stops committing to that store.
- **It never touches a store nested inside another repo.** Initialising there would shadow the outer repository.
- **It never disturbs your git state.** The snapshot is written with plumbing against a throwaway index, so your staging area is untouched and no hooks run. A repo in the middle of a merge, rebase or cherry-pick is skipped until it is out of that state.
- **It grants nobody access.** ATL reads the declared *path* for this one mechanical purpose. Who may read or write a store is a separate contract that the platform does not yet enforce.

Declared paths are vetted before any of this runs — a store must be at least two levels below your home directory and must not contain your current working directory, so a malformed or hostile declaration cannot turn `~` or a project checkout into a repository. Set `ATL_NO_STORE_GIT=1` to switch the whole pass off.

The CLI learns nothing about your team from this — no name, no meaning, no knowledge of what the directory holds. It honors a declaration, which is why any future team gets the same treatment for free.

## Version constraints

The `dependencies` values and `requires.atl` are written in standard SemVer range syntax by convention:

| Syntax | Meaning |
|---|---|
| `^1.2.3` | `>=1.2.3 <2.0.0` (caret — default recommended) |
| `~1.2.3` | `>=1.2.3 <1.3.0` (tilde) |
| `1.2.3` | Exact pin |
| `>=1.2.0` | Open-ended minimum |

Caret (`^`) is the conventional recommendation — semantically it gets patch and minor updates and blocks breaking major bumps. Today, though, the CLI does not evaluate these ranges: `atl install` resolves each dependency by name and installs the version currently in the catalog, and `requires.atl` is not enforced. Declare them anyway — they document intent, and range enforcement can arrive without a manifest change.

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

The installable asset directories are `agents/`, `skills/`, `rules/`, `knowledge/`, `backends/`, `scripts/`, and `packs/` (the `teampkg.AssetDirs` set). `agents/`/`skills/`/`rules/` are what Claude Code reads directly; `knowledge/`/`scripts/`/`packs/` carry a team's runtime reference docs, helper scripts, and area packs; `backends/` carries a team's per-backend adapter contracts (e.g. the delivery-team's `backends/{azure,github}/`). Everything else (`team.json`, `README`, `LICENSE`) stays behind.

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
