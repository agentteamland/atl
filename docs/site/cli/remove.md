# `atl remove`

Uninstall a team, removing only the files it installed — reversibly.

## Usage

```bash
atl remove <handle>/<team>            # remove from the project scope (default)
atl remove <handle>/<team> --global   # remove from the user-global layer
```

`<handle>/<team>` is the team's reference — the GitHub owner plus the team name, the same form you pass to [`atl install`](/cli/install). Run [`atl list`](/cli/list) to see what's installed and at which scope.

## Example

```bash
$ atl remove acme/example-team
atl remove: removed acme/example-team (17 files) from project scope — reversible with `atl gc --undo`
```

If the team isn't installed at that scope:

```bash
$ atl remove acme/example-team
acme/example-team is not installed at project scope
```

## What happens

1. The install manifest for the team at the chosen scope is read from `<layer>/.atl/installed/<handle>__<name>.json` — `<layer>` is `~/.atl` for `--global`, `<project>/.atl` for the project scope.
2. Every file the manifest recorded (under the installed asset dirs — `.claude/agents/`, `skills/`, `rules/`, `knowledge/`, `scripts/`, `packs/`) is **soft-deleted** into `~/.atl/gc-trash`, not hard-deleted — so a promoted gain that landed in the manifest can always be recovered.
3. The directories that held those files are pruned, deepest first — but only the ones that are now empty. A directory still holding another team's files or your own content is left in place.
4. The manifest itself is removed.

When files were soft-deleted, the removal is reversible: [`atl gc --undo`](/cli/gc) restores the most recent batch, and [`atl gc --purge`](/cli/gc) clears the trash for good. The output reports how many files were removed and from which scope:

```
atl remove: removed <handle>/<name> (N files) from <scope> scope — reversible with `atl gc --undo`
```

If the manifest's files were already gone from disk, nothing is moved to `~/.atl/gc-trash` — so the output omits the reversibility promise and reports that the files were already absent (only the manifest is dropped):

```
atl remove: dropped <handle>/<name> manifest from <scope> scope — no files were soft-deleted (they were already absent)
```

::: tip Only manifest-recorded files are removed
`atl remove` deletes exactly the files the team registered when it was installed — nothing more. Anything else under `.claude/` is left untouched: auto-grown agent `children/` and `learnings/`, your own skills, wiki pages, journal entries, and any other content you authored. None of it was recorded in the manifest, so it survives the uninstall.
:::

## Scope

`atl remove` operates on the **project scope** by default — the `.claude/` and `.atl/` of the directory you run it in. Pass `--global` to remove a team from the user-global layer (`~/.claude` assets, `~/.atl` manifest) instead.

A team can be installed at both scopes independently; removing one leaves the other in place. Run [`atl list`](/cli/list) to see which scope a team lives at before removing it.

## Flags

| Flag | Effect |
|---|---|
| `--global` | Remove from the user-global layer instead of the project. |

## Related

- [`atl list`](/cli/list) — see what's installed and at which scope.
- [`atl install`](/cli/install) — reinstall if you change your mind.
- [`atl update`](/cli/update) — refresh installed teams to the latest catalog version.
