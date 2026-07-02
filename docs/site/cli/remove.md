# `atl remove`

Uninstall a team, deleting only the files it installed.

## Usage

```bash
atl remove <handle>/<team>            # remove from the project scope (default)
atl remove <handle>/<team> --global   # remove from the user-global layer
```

`<handle>/<team>` is the team's reference — the GitHub owner plus the team name, the same form you pass to [`atl install`](/cli/install). Run [`atl list`](/cli/list) to see what's installed and at which scope.

## Example

```bash
$ atl remove acme/example-team
atl remove: removed acme/example-team (17 files) from project scope
```

If the team isn't installed at that scope:

```bash
$ atl remove acme/example-team
acme/example-team is not installed at project scope
```

## What happens

1. The install manifest for the team at the chosen scope is read from `<layer>/.atl/installed/<handle>__<name>.json` — `<layer>` is `~/.atl` for `--global`, `<project>/.atl` for the project scope.
2. Every file the manifest recorded (under `.claude/agents/`, `.claude/skills/`, `.claude/rules/`) is deleted.
3. The directories that held those files are pruned, deepest first — but only the ones that are now empty. A directory still holding another team's files or your own content is left in place.
4. The manifest itself is removed.

The output reports how many files were deleted and from which scope:

```
atl remove: removed <handle>/<name> (N files) from <scope> scope
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
