---
name: profile-backup
description: "/profile-backup — snapshot the global profile store (~/.atl/profiles) into the current git repo and commit it, so your accumulated memory is versioned, portable, and recoverable. Deterministic cp + git; global stays authoritative."
---

# /profile-backup — snapshot your global profile into this repo

Your profiles live **globally** at `~/.atl/profiles/` — that is the single source of truth,
known in every project and every conversation. This skill does **not** move them. It takes a
snapshot of whatever is in global **right now** and copies it into the current git repo (a
`profile-backup/` directory at the repo root), then commits it — so your accumulating memory
becomes git-tracked, versioned, and portable to another machine.

One direction only: **global → repo**. The inverse (repo → global) is `/profile-restore`, and
it is guarded so it never clobbers newer global data. The curation loop is `/profile-drain`.

The procedure below is **exact and deterministic** — a `cp` + `git` sequence, not fuzzy
LLM-copying. This skill is the conversational *home*; its body runs verbatim.

## Procedure

Run this from **inside the git repo you want to version your profile in**. It self-guards on
two conditions — not a git repo, and an empty/absent profile store — and reports which of the
four outcomes happened:

```bash
set -euo pipefail

# 1. Must be inside a git repo — the snapshot has nowhere to live otherwise.
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "not-a-git-repo"; exit 1; }

# 2. Must have something to back up.
SRC="$HOME/.atl/profiles"
if [ ! -d "$SRC" ] || [ -z "$(ls -A "$SRC" 2>/dev/null)" ]; then
  echo "nothing-to-back-up"; exit 0
fi

# 3. Snapshot global → repo. Clear then copy, so the backup is a true mirror
#    (a profile deleted in global disappears from the snapshot too).
DEST="$REPO_ROOT/profile-backup"
rm -rf "$DEST"
mkdir -p "$DEST"
cp -R "$SRC/." "$DEST/"

# 4. Version it with a dated commit. -f: profile-backup/ is this skill's own managed
#    artifact, so stage it even if the repo gitignores it (a plain `add` would exit 1
#    under set -e and abort here, after the copy, with no outcome marker printed).
git -C "$REPO_ROOT" add -f profile-backup
if git -C "$REPO_ROOT" diff --cached --quiet; then
  echo "already-current"
else
  git -C "$REPO_ROOT" commit -m "chore(profile): snapshot ~/.atl/profiles ($(date +%F))"
  echo "committed"
fi
```

## Report

Relay the outcome plainly, mapped from the marker the script printed:

- **`committed`** — "Backed up your profile into `profile-backup/` and committed it." Mention
  they can push the repo to carry the snapshot to another machine.
- **`already-current`** — "Your profile is already snapshotted here — nothing changed, nothing
  to commit."
- **`nothing-to-back-up`** — "There's no profile to back up yet. Talk to `/advisor` first, then
  run this again." (Stop — do not create an empty snapshot.)
- **`not-a-git-repo`** (exit 1) — "This folder isn't a git repo, so there's nowhere to version
  the snapshot. Run `/profile-backup` from inside the git repo you keep your profile in."
  (Stop.)

## Boundaries

- **One-way, global-authoritative.** This only ever copies global → repo. It never reads *from*
  the repo, never edits global, and never changes what `/advisor` or `/profile-drain` see —
  those always work off the live global store.
- **The snapshot is a mirror, not an append.** The destination is cleared before each copy, so
  the committed backup always equals the current global state, deletions included. Prior
  snapshots stay recoverable through git history.
- **Deterministic body.** No judgement calls in the copy — the `cp`/`git` block runs as
  written. The LLM's only job is running it and relaying which of the four outcomes occurred.
- **Restore is the guarded inverse.** Bringing a snapshot back into global is `/profile-restore`,
  which diffs and confirms first so newer global memory is never silently lost.
