# `atl publish`

Share the gains your global layer has accumulated for a team back to where it came from — ring 2→3 of gain circulation.

## Usage

```bash
atl publish <handle>/<team>
```

`<handle>/<team>` is the team reference (the handle is the team's GitHub owner). The team must be installed at your **global** layer — `publish` works from your global gains, not a project's.

It diffs your global copy of the team against its **published** version (a fresh fetch of the source repo). Every file that differs — or that you've added — is a *publishable gain*. By default `publish` only **shows the plan**; nothing is forked, committed, or pushed until you pass `--apply`.

## When to use it

Use it after a team has earned improvements through real use — a clearer instruction, a better pattern, a learning worth sharing with anyone who installs that team. `atl update` will also nudge you (`gains in X not yet upstream — run atl publish X …`) when it notices your global copy has drifted ahead of the published version.

`publish` is **deliberate by design**: it crosses the author boundary, so it never runs automatically. You invoke it (your consent to share outward); for a team you don't own, the owner accepts the PR (their consent). Either way, your own local and global gains never depend on acceptance.

## What it does, by ownership

`publish` checks whether the team's repo owner matches your authenticated GitHub login:

- **You own it** → **re-publish**: clone the repo, stage the gains under the team's subpath, patch-bump `team.json`'s version, then commit + tag + push, and ensure the repo carries the `atl-team` topic so the index reindexes.
- **You don't own it** → **propose upstream**: fork the source repo, branch off its default branch, stage the gains, push to your fork, and open a PR against the source repo — a best-effort contribution the owner can accept or decline.

The CLI does only the mechanics. The judgment (which divergences are worth sharing — general improvements, not project- or user-specific ones) and the prose (the PR body or commit message) come from the `/publish` skill, which drives this command and hands it the authored text via `--body-file`.

## Flags

| Flag | Effect |
|---|---|
| `--apply` | Act on the plan (fork + push + open the PR, or commit + tag + push). Without it, `publish` only prints the plan. |
| `--body-file <path>` | File holding the PR body (propose-upstream) or commit message (re-publish). Authored by the `/publish` skill. **Required with `--apply`.** |
| `--dry-run` | With `--apply`, print exactly what would happen (fork/branch/stage/push, or commit/tag/push) without touching GitHub. Works without `--body-file`. |
| `--only <paths>` | Restrict to these `.claude`-relative paths — the subset the skill kept after dropping project/user-specific gains. Repeatable / comma-separated. |

`--apply` needs `--body-file`; to preview an apply without authoring a body, pass `--dry-run`.

## Examples

Show what's publishable (no side effects):

```bash
atl publish mesut/my-team
```

```
atl publish: 2 publishable gain(s) in mesut/my-team (vs published main):
  modified  agents/reviewer.md
  new       rules/commit-style.md

You own github.com/mesut/my-team — these would re-publish to it (commit + version bump + tag).
```

Preview an apply for a team you don't own, then act once the `/publish` skill has written the PR body:

```bash
atl publish acme/example-team --dry-run --apply
atl publish acme/example-team --apply --body-file pr-body.md
```

```
atl publish: opened https://github.com/acme/example-team/pull/42
```

## Notes

- **Whole-file gains** (consistent with `promote` and fan-out) — each file is proposed as it stands in your global layer, not a hand-merge of fragments.
- Propose-upstream uses a deterministic branch (`atl-publish/<handle>-<name>`) and force-pushes, so re-publishing the same team updates its own branch and any open PR rather than colliding with a stale one.
- The mechanics need `gh` (authenticated) on your `PATH`. The default plan view does not.

## Related

- [`atl search`](/cli/search) — discover teams via the same `atl-team`-topic index that `publish` keeps fresh.
- [`atl update`](/cli/update) — surfaces the "gains not yet upstream" suggestion that prompts a `publish`.

---

## Installing `atl`

```bash
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.sh | sh

# Windows (PowerShell)
irm https://raw.githubusercontent.com/agentteamland/atl/main/scripts/install.ps1 | iex
```

Or grab a pre-built binary from [GitHub Releases](https://github.com/agentteamland/atl/releases/latest).
