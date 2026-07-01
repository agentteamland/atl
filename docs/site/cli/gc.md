# `atl gc`

Reclaim orphaned assets — the **reversible inverse of install**. Finds files under `.claude/agents|skills|rules` that no install manifest owns, plus stale promote conflict archives, and removes them without ever destroying anything irreversibly.

`atl install` / `update` / `promote` write assets into `~/.claude` and `<proj>/.claude` and record each one in an install manifest. Nothing prunes what falls out of that contract — a file dropped upstream on an update (left on disk by design), a learning-loop gain left behind after a team is removed, or a directory you made by hand. Over time these accrete. `atl gc` is the missing cleanup half.

**`doctor` heals; `gc` prunes.** [`atl doctor`](/cli/doctor) restores manifest-listed files that went *absent*; `atl gc` removes files that no manifest *owns*. They are deliberate opposites — and gc never deletes irreversibly.

## Usage

```bash
atl gc            # report only — a dry run that touches nothing (default)
atl gc --apply    # soft-delete orphans to ~/.atl/gc-trash (reversible)
atl gc --undo     # restore the most recent soft-delete batch
atl gc --purge    # hard-delete expired trash batches — the only real delete
```

## What counts as an orphan

`atl gc` walks both layers (`~/.claude` global and `<proj>/.claude` project), cross-references every asset file against that layer's install manifests, and flags anything **no manifest owns**. Each is labeled with a guessed origin — a hint, never a certainty:

| Origin | What it usually means |
|---|---|
| *gain or edit beside an installed unit* | A file under an installed agent/skill (e.g. a `children/` learning) that isn't in the manifest — often a learning-loop gain, sometimes a hand edit. |
| *unowned unit (a removed team or a hand-made dir)* | A whole `agents/x` or `skills/x` dir no manifest owns — a team that was removed leaving files behind, or your own non-ATL Claude Code assets. |
| *expired conflict archive* | A promote conflict archive under `~/.atl/history/` older than 30 days (these are content-addressed and never pruned otherwise). |

Because "no manifest owns it" also matches your own non-ATL assets, gc is **dry-run by default** and never deletes irreversibly — you always see the list before anything moves.

## The reversible safety model

Deletion is the one place ATL can't be silently automatic, so gc makes the operation reversible instead of making it manual:

1. **`atl gc`** (default) — reports orphans by scope, origin, and size. Touches nothing.
2. **`atl gc --apply`** — **soft-delete**: moves each orphan into a timestamped batch under `~/.atl/gc-trash/` and writes an undo manifest. Nothing is destroyed.
3. **`atl gc --undo`** — restores the most recent batch to its original paths.
4. **`atl gc --purge`** — the only real delete: hard-removes trash batches older than 30 days.

So there is no irreversible data loss at any step. The action stays manual (you run `atl gc`), but awareness is automatic: a session-start note surfaces high-signal orphans (`atl: N orphaned file(s) beside installed units — run atl gc to review`) so you never have to remember to check.

## Related

- [`atl doctor`](/cli/doctor) — the heal half: restores manifest-listed files that went missing (gc is the prune half)
- [`atl remove`](/cli/remove) — removes a team's manifest-listed files; gc catches what remove leaves behind
- [`atl promote`](/cli/promote) — writes the conflict archives that gc expires after 30 days
