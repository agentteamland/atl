---
name: profile-restore
description: "/profile-restore — bring your profile store back from its private remote onto this machine: create ~/.atl/profiles if missing, attach the remote you supply, and fast-forward from it. Never overwrites memory this machine already has."
---

# /profile-restore — bring your profile back onto this machine

The inverse of `/profile-backup`. Backup pushes the store's own history to a private remote
you supply; restore fetches it back — onto a new machine, or onto one that has fallen behind.

Like backup, it needs exactly one thing from you: **the remote.** Git holds it from then on.

The one non-negotiable rule of this system is unchanged: **it never silently overwrites memory
this machine has and the remote does not.** What changed is who enforces it. This used to be a
file overlay with a hand-written timestamp comparison; now the pull is `--ff-only`, so git
itself refuses the moment the two histories have diverged. A guarantee the tool enforces beats
one a script has to remember to check.

## Procedure

If the user has already given you a remote URL in this conversation, export it first as
`ATL_PROFILE_REMOTE` — otherwise run the block as-is and it will tell you one is needed.

```bash
# `set -eu`, not `-euo pipefail`: `-o pipefail` is non-POSIX and aborts under dash. The
# body is pasted into whatever shell the agent runs — sh, bash or zsh.
set -eu

GLOBAL="$HOME/.atl/profiles"

# 1. The store directory. Restore is the one skill that legitimately runs before anything
#    else exists — a brand-new machine — so it creates rather than complains.
if [ ! -d "$GLOBAL" ]; then
  mkdir -p "$GLOBAL"
fi

# 2. Under git. `atl session-start` normally does this, but it only versions a store that
#    already has content, and on a new machine there is none yet — so restore cannot assume
#    the repo is there.
if ! git -C "$GLOBAL" rev-parse --git-dir >/dev/null 2>&1; then
  git -C "$GLOBAL" init -q
fi

# 3. Where from? The remote is the record; nothing else stores it.
if [ -z "$(git -C "$GLOBAL" remote 2>/dev/null)" ]; then
  if [ -z "${ATL_PROFILE_REMOTE:-}" ]; then echo "no-remote"; exit 0; fi
  git -C "$GLOBAL" remote add origin "$ATL_PROFILE_REMOTE"
fi

if ! git -C "$GLOBAL" fetch -q origin; then echo "fetch-failed"; exit 1; fi

# 4. Which branch is the remote's? Ask the remote rather than guessing: a store created by
#    `git init` is on whatever that git's init.defaultBranch says, which differs by machine
#    and by git version, so "it is master here" proves nothing about the other end.
BRANCH="$(git -C "$GLOBAL" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)"
BRANCH="${BRANCH#origin/}"
if [ -z "$BRANCH" ]; then
  for candidate in main master; do
    if git -C "$GLOBAL" show-ref --verify --quiet "refs/remotes/origin/$candidate"; then
      BRANCH="$candidate"; break
    fi
  done
fi
if [ -z "$BRANCH" ]; then echo "no-branch"; exit 1; fi

# 5. A dirty tree means uncommitted memory. Pulling over it is the one irreversible mistake
#    available here, so stop instead — `atl session-start` commits it on the next session.
if [ -n "$(git -C "$GLOBAL" status --porcelain 2>/dev/null)" ]; then
  echo "dirty-store"; exit 1
fi

# 6. Fast-forward only. If this machine holds commits the remote does not, git refuses and
#    nothing is lost — which is the whole safety property, enforced rather than checked.
if git -C "$GLOBAL" pull -q --ff-only origin "$BRANCH"; then
  echo "restored"
else
  echo "diverged"; exit 1
fi
```

## Report

Relay the outcome plainly, mapped from the marker the script printed:

- **`restored`** — "Your profile store is now up to date with the remote."
- **`no-remote`** (exit 0) — **this is the one that needs the user.** Say: *"I don't know where
  your profile backup lives. Give me the URL of the private repo you push it to and I'll
  attach it and pull."* Then **stop and wait.** Never invent a URL and never guess from a repo
  you happen to see on disk — pulling from the wrong remote writes someone else's data into
  their memory. When they answer, re-run the block with `ATL_PROFILE_REMOTE` set.
- **`fetch-failed`** (exit 1) — the remote could not be reached. Report git's error as-is; it
  is usually authentication or a wrong URL.
- **`no-branch`** (exit 1) — the remote has no `main` or `master` and publishes no default.
  Ask which branch holds the store rather than trying others.
- **`dirty-store`** (exit 1) — "This machine has profile changes that aren't committed yet, so
  I stopped rather than pull over them." `atl session-start` commits them next session; then
  restore is safe.
- **`diverged`** (exit 1) — **do not work around this.** It means this machine holds memory the
  remote does not. Say so plainly: *"Your local profile has entries the backup doesn't — a pull
  would need a merge, and that's your call, not mine."* Never suggest `--force`, never suggest
  `reset --hard`, and never re-run without `--ff-only`. The likely right move is
  `/profile-backup` first, to push what this machine has.

## Safety

- **Fast-forward only.** The one unacceptable failure is losing accumulated memory, and
  `--ff-only` makes it unreachable: git refuses the moment local holds anything the remote does
  not. This replaces a hand-written timestamp comparison, which had to be correct on every
  path to work; a refusal is correct by construction.
- **Never pulls over uncommitted work.** A dirty store stops the skill.
- **One-way, remote-as-source.** Restore reads from the remote and never writes to it. Pushing
  is `/profile-backup`.
- **It never chooses the remote.** No inference from a repo on disk, no reuse of a URL seen
  elsewhere. Where a person's memory lives is theirs to state.

## Boundaries

- **A new machine is the normal case**, which is why this skill creates the store directory and
  its git repo rather than requiring them. Everywhere else in ATL an absent store means the
  feature is not in use; here it means the user has just arrived.
- This skill only moves history between the remote and the store. It does not parse, curate, or
  privacy-gate profile content — that is the `profile-curator`'s job via `/profile-drain`.
- **The remote's visibility is `/profile-backup`'s gate, not this one's.** Restore only reads,
  so it publishes nothing; the check that a remote is private runs on every push, which is
  where the irreversible direction is.
