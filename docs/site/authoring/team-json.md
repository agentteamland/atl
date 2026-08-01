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
| `capabilities` | object | — | Optional contracts, mostly read by the platform's skills rather than the install parser. `capabilities.review: "<agent>"` names the agent [`/create-pr`](/skills/create-pr) spawns as this team's specialist reviewer; `capabilities.profile` declares the profile-layer provider/consumer role (see [profile-team](/teams/profile-team)). Two keys are read by the **CLI** itself: `store` and `channel` — see below. |
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

Declared paths are vetted before any of this runs: symlinks are resolved first, and the destination must be at least two levels below your home directory and must not contain your current working directory. So a declaration of `~`, of a bare top-level directory, of anywhere outside your home, or of the project you are working in is refused rather than turned into a repository — silently, with no diagnostic. It is not a sandbox: a directory that satisfies those rules is accepted whatever it holds, because a team legitimately keeping its store there looks exactly the same. Set `ATL_NO_STORE_GIT=1` to switch the whole pass off.

::: warning A store outside your home is not versioned
If you keep the store on an external volume or a sync folder — declared directly, or reached through a symlink — it falls outside the vetted region and simply never gets versioned. There is no warning today.
:::

One known limit: if the store itself contains a nested git repository, that subtree is recorded as a gitlink and its contents are not captured in the snapshot. A store is machine-written data, so this does not arise in practice — but it is silent if it does.

The CLI learns nothing about your team from this — no name, no meaning, no knowledge of what the directory holds. It honors a declaration, which is why any future team gets the same treatment for free.

## Declaring a capture channel

ATL's capture loop — an inline marker dropped mid-conversation, transferred into the durable queue, drained in the background by a skill — is not core's alone. Core owns one channel, `learning`. A team can own another by declaring it:

```json
{
  "capabilities": {
    "profile": {
      "role": "provider",
      "channel": {
        "name": "profile-fact",
        "drain": "/profile-drain",
        "rule": "profile-capture",
        "describes": "durable entity facts"
      }
    }
  }
}
```

Like `store`, `channel` may sit under any capability name. Each of its four fields is consumed somewhere:

| Field | What it feeds |
|---|---|
| `name` | the queue channel, and the marker prefix the capture pass looks for — `<!-- profile-fact: … -->`. Also what `atl learnings peek --channel <name>` accepts. |
| `drain` | the skill the agent is told to spawn as a background subagent. |
| `rule` | the rule both signals name as the thing that says what to do. |
| `describes` | the human label in the watchdog's "review the recent turns for missed **…**". |

`atl install` records the declaration into the install manifest, and from then on [`atl tick`](/cli/tick) and session start emit that channel's signals beside core's own:

```
atl: 2 profile-fact(s) pending — auto-drain them now in a background subagent (per the profile-capture rule)
atl: capture-watchdog (profile-fact) — no profile-fact markers for 4 assistant turn(s) / ~2100 chars of user input; review the recent turns for missed durable entity facts and mark them, and spawn ONE background /profile-drain subagent to mine the stretch (per the profile-capture rule, valid even with an empty queue)
```

The division of labour is the whole point. **The platform emits a generic signal; the team's rule acts on it.** ATL knows a channel's four words and nothing more — not which team shipped it, not what a `profile-fact` is, not what `/profile-drain` does with one. The instruction that turns a signal into behaviour ships in the rule the declaration names, installed with the team that owns the channel. A machine without that team therefore never sees the signal at all: no declaration, no channel, nothing to act on.

That direction also settles what happens to a marker on an **undeclared** channel — nothing. It is never queued, so a typo (`profile-fct` for `profile-fact`) cannot open a phantom channel where a fact *looks* captured but no drain will ever claim it. A near-miss of an active channel is reported so the typo is visible instead of silently swallowed; `atl learnings peek --channel` rejects an unknown channel that matches nothing, rather than answering a typo with "no pending items". (It still reads a channel that *has* queued items but is no longer active — a team was uninstalled with items pending, say. Those items are real, and `atl learnings status` lists them for the same reason.)

A declaration missing `drain`, `rule` or `describes` is refused rather than emitted with a hole in it, and a channel whose name is already taken — by core's `learning`, or by another installed team — is ignored. Both surface as a warning from [`atl doctor`](/cli/doctor), because either one is otherwise a silent loss: markers on that channel are never captured and nothing says why.

What `atl doctor` *cannot* check is whether `rule` and `drain` point at assets that exist: it reads an installed manifest, long after the team's source tree is gone. A declaration naming a rule the team never ships is the worse failure of the two — the channel goes active and its markers *are* captured, but nothing ever tells an agent to drain them. For first-party teams in this monorepo, [`atl skills check`](/cli/skills) resolves both names against `rules/` and `skills/` and fails CI. A team published from its own repo has no such gate, so check it yourself: the rule you name must be a file you ship.

::: tip Already installed before this shipped?
Same story as `store`: the declaration is read at install time, so an install that predates the `channels` field carries no record of it and behaves exactly like a team that declares none. `atl update` backfills it once by re-fetching the pinned source, and it runs on its own — until it does, that channel's markers are not captured.
:::

**It grants nobody access.** ATL reads the declared *words* for this one mechanical purpose — wording a signal, and deciding which channels exist at all. Who may read or write what a capability names is a separate contract the platform does not yet enforce.

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
