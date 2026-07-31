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
three conditions — not a git repo, **a repo that isn't demonstrably private**, and an
empty/absent profile store — and reports which of the six outcomes happened:

```bash
set -euo pipefail

# 1. Must be inside a git repo — the snapshot has nowhere to live otherwise.
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || { echo "not-a-git-repo"; exit 1; }

# 2. Must be a PRIVATE repo — checked BEFORE anything is copied.
#    The profile store is the most sensitive data ATL holds: perception-flagged records
#    about other people, state.emotional, and a consent-gated Tier-4 state.financial.
#    Publishing it is irreversible — git history keeps it after a delete, and a public
#    repo may already be cloned or indexed. So this fails CLOSED: proceed only on a
#    definite "private". `gh` resolves the repo from the current directory, so a missing
#    remote, a non-GitHub remote, or an unavailable/unauthenticated `gh` all land in
#    visibility-unknown rather than a guess.
PRIVATE="$(cd "$REPO_ROOT" && gh repo view --json isPrivate -q .isPrivate 2>/dev/null || true)"
case "$PRIVATE" in
  true)  : ;;                                  # private — the only case that proceeds
  false) echo "public-repo"; exit 1 ;;
  *)     echo "visibility-unknown"; exit 1 ;;
esac

# 3. Must have something to back up.
SRC="$HOME/.atl/profiles"
if [ ! -d "$SRC" ] || [ -z "$(ls -A "$SRC" 2>/dev/null)" ]; then
  echo "nothing-to-back-up"; exit 0
fi

# 4. Snapshot global → repo. Clear then copy, so the backup is a true mirror
#    (a profile deleted in global disappears from the snapshot too).
#    The store is itself a git repo (ATL versions it locally so an overwritten value
#    stays recoverable), and its .git MUST NOT be copied: `git add` on a directory
#    that contains a nested .git records a GITLINK instead of the files — it exits 0,
#    the emptiness check below sees a change, and the user is told the backup
#    succeeded while the snapshot holds no profile content at all.
#    Copy everything, then remove the .git — rather than filtering during the copy.
#    A glob-and-skip loop would be shell-dependent: zsh errors on an unmatched glob
#    (its default `nomatch`), which aborts under `set -e` AFTER the rm -rf above, so
#    the snapshot is wiped and no outcome marker is ever printed. This form runs the
#    same under sh, bash and zsh, and it also preserves dangling symlinks.
DEST="$REPO_ROOT/profile-backup"
rm -rf "$DEST"
mkdir -p "$DEST"
cp -R "$SRC/." "$DEST/"
rm -rf "$DEST/.git"

# 5. Version it with a dated commit. -f: profile-backup/ is this skill's own managed
#    artifact, so stage it even if the repo gitignores it (a plain `add` would exit 1
#    under set -e and abort here, after the copy, with no outcome marker printed).
git -C "$REPO_ROOT" add -f profile-backup
# Scope the emptiness check AND the commit to the snapshot path only, so a user's
# unrelated pre-staged changes are neither counted here nor swept into a commit
# labelled as a profile snapshot.
if git -C "$REPO_ROOT" diff --cached --quiet -- profile-backup; then
  echo "already-current"
else
  git -C "$REPO_ROOT" commit -m "chore(profile): snapshot ~/.atl/profiles ($(date +%F))" -- profile-backup
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
- **`public-repo`** (exit 1) — **"This repo is public, so I didn't copy anything.** Your profile
  holds what you've said about the people in your life and your tier-4 facts; committing it here
  would publish all of it, and git history keeps it even after a delete. Run `/profile-backup`
  from a **private** repo instead." (Stop. Nothing was copied — the guard runs before the `cp`.)
  **Do not offer a workaround**, and never suggest making the repo private just to proceed; if
  they have no private repo, say so plainly and leave the choice with them.
- **`visibility-unknown`** (exit 1) — "I couldn't confirm this repo is private (no GitHub remote,
  or `gh` isn't available here), so I stopped rather than guess." Then **ask**: is this repo
  private? Proceed only on an explicit yes from them — never on your own inference from the path,
  the name, or the absence of a remote. (Stop until answered.)

## Boundaries

- **One-way, global-authoritative.** This only ever copies global → repo. It never reads *from*
  the repo, never edits global, and never changes what `/advisor` or `/profile-drain` see —
  those always work off the live global store.
- **The snapshot is a mirror, not an append.** The destination is cleared before each copy, so
  the committed backup always equals the current global state, deletions included. Prior
  snapshots stay recoverable through git history.
- **Deterministic body.** No judgement calls in the copy — the `cp`/`git` block runs as
  written. The LLM's only job is running it and relaying which of the six outcomes occurred.
- **The visibility guard fails closed, and it is not advisory.** This skill is cwd-relative: it
  writes into *whatever* repo you happen to be standing in, with `git add -f`, so a `.gitignore`
  is no protection. One directory too high is enough to publish everything. Guidance ("run this
  from the right repo") is what fails when a skill is invoked from the wrong place, so the check
  is a hard stop, it runs **before** any copy, and only a definite `isPrivate: true` proceeds —
  unknown is treated as unsafe. The asymmetry is the reason: a wrong refusal costs one command,
  a wrong proceed is irreversible.
- **Restore is the guarded inverse.** Bringing a snapshot back into global is `/profile-restore`,
  which diffs and confirms first so newer global memory is never silently lost.
