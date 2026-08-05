---
name: profile-backup
description: "/profile-backup — push the global profile store (~/.atl/profiles) to a private remote you supply, so your accumulated memory survives losing the machine. Deterministic git; the store stays authoritative and never moves."
---

# /profile-backup — get your profile off this machine

Your profiles live **globally** at `~/.atl/profiles/`, and `atl session-start` already keeps
that directory under local git — so an overwritten value stays recoverable. That protects you
from a bad write. It does **not** protect you from losing the disk: local history and the
files it describes die together.

This skill closes that gap, and it needs exactly one thing from you: **a private remote.**
Git then holds the destination itself, so nothing else has to record it — and because the
store's own repo is what gets pushed, nothing is ever copied anywhere, which is what keeps
this simple.

One direction only: **the store's history → your remote.** The store is never moved, never
edited, and never read *from* the remote. The inverse is `/profile-restore`; the curation
loop is `/profile-drain`.

The procedure below is **exact and deterministic** — a `git` sequence, not fuzzy LLM-copying.
This skill is the conversational *home*; its body runs verbatim.

## Procedure

If the user has already given you a remote URL in this conversation, export it first as
`ATL_PROFILE_REMOTE` — otherwise run the block as-is and it will tell you one is needed.

```bash
# `set -eu`, not `-euo pipefail`: `-o pipefail` is the one non-POSIX construct here and
# aborts under dash with "Illegal option -o pipefail". The body is pasted into whatever
# shell the agent runs — sh, bash or zsh — so it has to be portable.
set -eu

SRC="$HOME/.atl/profiles"

# 1. Must have something to back up.
if [ ! -d "$SRC" ] || [ -z "$(ls -A "$SRC" 2>/dev/null)" ]; then
  echo "nothing-to-back-up"; exit 0
fi

# 2. The store is versioned by `atl session-start`, not here. If it is not a repo, that
#    path has not run or is disabled — report it rather than quietly creating a second
#    mechanism that versions the same directory on different terms.
git -C "$SRC" rev-parse --git-dir >/dev/null 2>&1 || { echo "not-versioned"; exit 1; }

# 3. Where would it go? A remote IS the recorded destination — git already holds it, so no
#    config file, no path to vet, and no way for the record to drift from reality.
REMOTE="$(git -C "$SRC" remote get-url origin 2>/dev/null || true)"
if [ -z "$REMOTE" ]; then
  if [ -z "${ATL_PROFILE_REMOTE:-}" ]; then echo "no-remote"; exit 0; fi
  REMOTE="$ATL_PROFILE_REMOTE"
fi

# 4. The remote must be PRIVATE — checked BEFORE it is attached and BEFORE anything is
#    pushed, and re-checked on every run, because the repo you recorded in June can be
#    public in August. Fails closed: only a definite "private" proceeds, so a non-GitHub
#    remote or an unavailable `gh` stops and asks rather than guessing.
SLUG="$REMOTE"
case "$SLUG" in
  git@github.com:*)       SLUG="${SLUG#git@github.com:}" ;;
  ssh://git@github.com/*) SLUG="${SLUG#ssh://git@github.com/}" ;;
  https://github.com/*)   SLUG="${SLUG#https://github.com/}" ;;
esac
SLUG="${SLUG%.git}"
PRIVATE="$(gh repo view "$SLUG" --json isPrivate -q .isPrivate 2>/dev/null || true)"
case "$PRIVATE" in
  true)  : ;;
  # A definite "public" is never overridable. This branch has no escape hatch on purpose.
  false) echo "public-remote"; exit 1 ;;
  # Unknown is a different thing from public: a self-hosted git, a GitLab, or an
  # unauthenticated `gh` all land here, and refusing them outright would mean the
  # feature simply does not exist for those users. So unknown stops and asks, and the
  # user's spoken "yes, it is private" is what sets the variable below — the same shape
  # as the old --apply gate. The agent must never set it on its own; see Report.
  #    The answer is recorded against the exact URL, in the store's own git config — the same
  #    reasoning that put the destination in `git remote`: the record lives with the thing it
  #    describes and cannot drift from it. Re-asking every run would buy nothing here, because
  #    an unverifiable host is unverifiable every time; it would just make the feature unusable
  #    for anyone not on GitHub. Change the URL and the confirmation does not come with it.
  *)     CONFIRMED="$(git -C "$SRC" config --get atl.confirmedPrivateRemote 2>/dev/null || true)"
         if [ "${ATL_PROFILE_REMOTE_CONFIRMED_PRIVATE:-}" = "1" ] || [ "$CONFIRMED" = "$REMOTE" ]; then
           git -C "$SRC" config atl.confirmedPrivateRemote "$REMOTE"
         else
           echo "visibility-unknown"; exit 1
         fi ;;
esac

# 5. Only now is it safe to attach.
if [ -z "$(git -C "$SRC" remote 2>/dev/null)" ]; then
  git -C "$SRC" remote add origin "$REMOTE"
fi

# 6. Commit anything written since the last session-start snapshot, so a backup taken
#    mid-session carries this session's work rather than yesterday's.
git -C "$SRC" add -A
if git -C "$SRC" diff --cached --quiet; then
  :
else
  git -C "$SRC" -c user.name=atl -c user.email=atl@localhost -c commit.gpgsign=false \
    commit -q -m "chore(profile): snapshot $(date +%F)"
fi

# 7. Push. The count is over commits no remote-tracking branch holds, which is the same
#    question `atl session-start` reports on — so "already-current" here and silence there
#    can never disagree.
if [ "$(git -C "$SRC" rev-list --count HEAD --not --remotes)" = "0" ]; then
  echo "already-current"
else
  BRANCH="$(git -C "$SRC" rev-parse --abbrev-ref HEAD)"
  if git -C "$SRC" push -q -u origin "$BRANCH"; then
    echo "pushed"
  else
    echo "push-failed"; exit 1
  fi
fi
```

## Report

Relay the outcome plainly, mapped from the marker the script printed:

- **`pushed`** — "Your profile is now on the remote — it survives losing this machine."
- **`already-current`** — "Already pushed; nothing has changed since."
- **`no-remote`** (exit 0) — **this is the one that needs you.** Say: *"Your profile store has
  no remote, so it exists only on this disk. Give me the URL of a **private** repo and I'll
  attach it and push."* Then **stop and wait.** Never invent a URL, never pick a repo for
  them, and never create one on their behalf — where their memory is stored is theirs to
  decide. When they answer, re-run the block with `ATL_PROFILE_REMOTE` set to what they gave.
- **`nothing-to-back-up`** — "There's no profile to back up yet. Talk to `/advisor` first."
- **`not-versioned`** (exit 1) — "The store isn't under git, which `atl session-start`
  normally handles — it may be disabled (`ATL_NO_STORE_GIT`) or it has not run here."
- **`public-remote`** (exit 1) — **"That repo is public, so I pushed nothing.** Your profile
  holds what you've said about the people in your life and your tier-4 facts; pushing it
  there would publish all of it, and git history keeps it after a delete. Give me a
  **private** repo instead." (Stop. Nothing was attached — the guard runs first.) **Do not
  offer a workaround**, and never suggest making the repo private just to proceed.
- **`visibility-unknown`** (exit 1) — "I couldn't confirm that remote is private (not a
  GitHub URL, or `gh` isn't available here), so I stopped rather than guess." Then **ask**:
  is it private? Proceed only on an explicit yes from them — never on your own inference
  from the URL, the name, or the account it sits under. (Stop until answered.) Only after
  they say yes, re-run with `ATL_PROFILE_REMOTE_CONFIRMED_PRIVATE=1` set. **Setting that
  variable without having asked is the one unrecoverable mistake available in this skill** —
  it is the user's statement, not your assessment, and a wrong one publishes their profile
  permanently. A `public-remote` result can never be overridden this way. Their answer is
  recorded against that exact URL, so they are asked once and not again; point the store at a
  different remote and it asks afresh.
- **`push-failed`** (exit 1) — the remote was confirmed private but the push was rejected.
  Report the git error as-is; do not retry with force, ever.

## Boundaries

- **The remote is the destination, and the user supplies it.** Nothing else records where
  the backup goes, which means the record cannot go stale — and it means this skill has no
  opinion about *which* repo. It never creates one and never guesses.
- **The visibility guard fails closed, and it is not advisory.** It runs before the remote is
  attached and before any push, and only a definite `isPrivate: true` proceeds — unknown is
  treated as unsafe. The asymmetry is the reason: a wrong refusal costs one command, a wrong
  push is irreversible. It re-runs every time, because a repo's visibility can change after
  you recorded it.
- **Nothing is copied.** Earlier versions of this skill mirrored the store into another
  repo's `profile-backup/` directory, which required deleting the nested `.git` first — a
  copy of a repo into a repo records a *gitlink* and reports success over zero content.
  Pushing the store's own history has no such edge, because there is no second copy.
- **This does not version the store.** `atl session-start` does, on every session. Step 6
  only catches writes made since that snapshot, so a backup taken mid-session is current.
- **One-way, store-authoritative.** The remote is a destination, never a source. Bringing a
  snapshot back is `/profile-restore`, which diffs and confirms first so newer memory in the
  store is never silently lost.
- **`atl session-start` tells you when this is needed.** It reports a store with no remote,
  and a store whose local history is ahead of it — so the gap surfaces on its own rather than
  waiting to be remembered.
