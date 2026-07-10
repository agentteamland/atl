---
name: profile-restore
description: Restore a profile snapshot from this repo's profile-backup/ back into the global store at ~/.atl/profiles — a diffed, dry-run-by-default overlay that requires an explicit --apply and never deletes memory accumulated since the snapshot.
---

# /profile-restore — bring a profile snapshot back into the global store

The inverse of `/profile-backup`. Backup snapshots the authoritative global store
(`~/.atl/profiles/`) into this repo's `profile-backup/` so it's git-trackable, versioned,
and portable; restore copies that snapshot **back into global**. The store stays
global-authoritative — the repo is a versioned mirror, not the source of truth.

Restore is **deterministic** (`diff` + `git` + `cp`), not fuzzy copying. And it obeys the
one non-negotiable rule of this system: **it never silently clobbers global data that is
newer than the snapshot.** Losing accumulated memory is the single unacceptable failure, so
restore is an **overlay** (it adds and overwrites, it never deletes global-only data),
**dry-run by default**, and it **only writes when you pass `--apply` after seeing the diff.**

## Procedure

### 1. Preview — always first (dry run, writes nothing)

Run this from the repo that holds the snapshot. It prints exactly what a restore would
change and stops. Read the output *with the user* before doing anything else.

```bash
set -euo pipefail

MODE="${1:-preview}"                  # preview (default) | --apply
SNAP="./profile-backup"               # the snapshot committed in THIS repo
GLOBAL="$HOME/.atl/profiles"          # the authoritative global store (restore target)

# No snapshot here → nothing to restore.
[ -d "$SNAP" ] || { echo "No profile-backup/ in $(pwd) — nothing to restore (take one with /profile-backup)."; exit 0; }

# Provenance + the snapshot's reference time. CRITICAL: derive the snapshot's age
# from its git COMMIT time, never from file mtimes — git does not preserve mtimes
# across clone/checkout, so a checked-out snapshot file always looks freshly modified
# and an mtime comparison would silently fail to flag genuinely-newer global memory on
# another machine (the one unacceptable failure this skill exists to prevent).
SNAP_EPOCH=""
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Snapshot: $(git log -1 --format='%h  %ci  %s' -- "$SNAP" 2>/dev/null || echo 'present but untracked in git')"
  SNAP_EPOCH=$(git log -1 --format=%ct -- "$SNAP" 2>/dev/null || echo "")
else
  echo "Note: not inside a git repo — provenance (commit/date) is unknown; restoring profile-backup/ as-is."
  echo "      Newer-than-snapshot detection is UNAVAILABLE without git; absence of a !! flag below does NOT mean 'safe to overwrite'."
fi

# Diff the snapshot (source) against global (target).
if [ ! -d "$GLOBAL" ]; then
  echo "No global store at $GLOBAL yet — restore would be a pure create, no overwrite."
else
  DIFF=$(diff -rq "$SNAP" "$GLOBAL" 2>/dev/null || true)
  echo
  echo "ADD  (in snapshot, missing from global — restore brings these back):"
  echo "$DIFF" | grep "^Only in $SNAP" || echo "  (none)"
  echo "KEEP (global-only — memory accumulated SINCE the snapshot; restore PRESERVES these, never deletes):"
  echo "$DIFF" | grep "^Only in $GLOBAL" || echo "  (none)"
  echo "OVERWRITE (differ — the snapshot's version would replace global's):"
  # Anchor to the real 'Files X and Y differ' lines (they end in ' differ'); a bare
  # grep 'differ' would also catch an 'Only in ...' line for a slug containing 'differ'.
  D=$(echo "$DIFF" | grep " differ$" || true)
  if [ -z "$D" ]; then
    echo "  (none — shared files already match the snapshot)"
  else
    echo "$D" | while read -r _ f _ g _; do
      GEP=$(stat -f %m "$g" 2>/dev/null || stat -c %Y "$g" 2>/dev/null || echo "")
      if [ -z "$SNAP_EPOCH" ]; then
        echo "  ?? $g — snapshot age unknown (not in git); cannot tell if global is newer — do NOT assume safe"
      elif [ -n "$GEP" ] && [ "$GEP" -gt "$SNAP_EPOCH" ]; then
        echo "  !! $g is NEWER than the snapshot (modified after the snapshot commit) — applying would overwrite newer memory with older data"
      else
        echo "     $g"
      fi
    done
  fi
fi

# Preview stops here; --apply performs the overlay.
if [ "$MODE" != "--apply" ]; then
  echo
  echo "DRY RUN — nothing written. If this is what you want, confirm, then re-run with --apply."
  exit 0
fi

# APPLY — reversible seatbelt first, then overlay (add + overwrite, never a delete).
SAFETY=""
if [ -d "$GLOBAL" ]; then
  SAFETY="$HOME/.atl/profiles.pre-restore-$(date +%Y%m%d-%H%M%S)"
  cp -R "$GLOBAL" "$SAFETY"
  echo "Safety copy of current global → $SAFETY"
fi
mkdir -p "$GLOBAL"
cp -R "$SNAP"/. "$GLOBAL"/
if [ -n "$SAFETY" ]; then
  echo "Restored (overlay: snapshot applied, global-only files preserved). Undo: rm -rf \"$GLOBAL\" && mv \"$SAFETY\" \"$GLOBAL\""
else
  echo "Restored — created $GLOBAL from the snapshot."
fi
```

### 2. Confirm, then apply — only on explicit yes

Never pass `--apply` on your own. After the user has seen the preview and **explicitly
confirms** (a clear "yes" / "apply"), re-run the same script with `--apply`. If the preview
showed any `!!` line — a global file newer than the snapshot — call it out plainly and make
sure the user means to overwrite it before you proceed.

```bash
# only after the user has seen the diff and said yes:
bash <the script above> --apply
```

If the preview showed nothing under OVERWRITE/ADD, there is nothing to restore — say so and
stop.

## Safety

- **Dry-run by default, write only on `--apply`.** No path writes to global without an
  explicit flag *and* an explicit user confirmation.
- **Overlay, never mirror.** Restore adds snapshot files and overwrites matching ones; files
  present in global but absent from the snapshot (memory accumulated since it was taken) are
  **preserved**. Restore contains no delete. If the user truly wants an exact mirror, that is
  a separate, deliberate act — not this skill.
- **Newer-than-snapshot is flagged loudly.** Any shared file where global was modified *after
  the snapshot's git commit time* is marked `!!` in the preview — the exact case where
  restoring would trade newer memory for older data. The check keys off the commit time, not
  file mtimes: git does not preserve mtimes across a clone, so an mtime check would miss this
  on another machine. Outside a git repo the snapshot's age is unknown, so shared files are
  marked `??` instead — and an absent flag must never be read as "safe".
- **`--apply` is itself reversible.** Before overwriting, restore copies the current global
  to `~/.atl/profiles.pre-restore-<timestamp>/` and prints the one-line undo.

## Boundaries

- **No snapshot present** (`profile-backup/` missing) → report it and stop; nothing to do.
- **Not in a git repo** → provenance (which commit/when) can't be shown, but the folder can
  still be restored; the skill notes the missing provenance and proceeds under the same gate.
- This skill only moves files between the repo snapshot and the global store. It does not
  parse, curate, or privacy-gate profile content — that is the `profile-curator`'s job via
  `/profile-drain`.
