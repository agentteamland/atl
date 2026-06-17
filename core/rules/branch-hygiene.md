# Branch + working-tree hygiene

A short discipline that prevents the recurring "drift" problem: editing on a stale branch, assuming working-tree state without checking, leaving merged branches around, or — worst — abandoning a branch with unmerged work. Four checkpoints.

## 1. Before starting work — verify, don't assume

Before editing in a shared repo:

```bash
git branch --show-current                      # on the branch you expect? (usually main)
git status --porcelain                         # clean?
git rev-list --count HEAD..@{u} 2>/dev/null    # behind upstream?
```

- Branch isn't what you expect → STOP. It may be a stale merged branch or another session's work-in-progress. Surface before editing.
- Working tree is dirty → STOP. Uncommitted changes might be lost work. Surface; ask.
- Behind upstream → `git pull --ff-only` first, then proceed.

When you need to know whether a PR is merged or a branch still exists, trust `gh`/`git` — not a UI's view. An editor or desktop client can show a stale PR/branch state long after GitHub has merged or deleted it. `gh pr view <n> --json state,mergedAt` and `gh api repos/<owner>/<repo>/branches` are authoritative; a tab that says "open" is not.

Uncertainty is a signal to verify, not to assume the previous state still holds.

## 2. After a merge — return to clean main AND delete the merged branch

When a PR merges — you ran the merge, or the user signals it ("merged", "no PRs left", a screenshot, "why is that branch still here") — your next action in each affected repo is:

```bash
git checkout main
git pull --ff-only
git branch -d <merged-branch>          # -d refuses unless truly merged → safe
```

Prefer `gh pr merge --delete-branch` so the **remote** branch is removed in the same step. (`-d` may refuse a squash-merged branch because the commit hash differs; once you've confirmed the PR is MERGED via `gh`, `-D` is safe for that branch specifically.) Detection is semantic, not lexical — equivalent phrasings in any language map to the same intent. Don't defer it; don't switch branches with a dirty tree (loss risk — surface first).

A merged branch is dead weight — leaving it is how orphan clutter accumulates. Delete it as soon as its PR is merged.

## 3. No orphan branches — every branch resolves to merged-or-deleted

A branch ends in exactly one of two states: **merged via a PR**, or **deleted**. There is no legitimate third state — a branch with no PR, a PR opened but never merged, or an abandoned experiment is *drift*, and may be lost work.

At the end of a unit of work — and especially at session end — audit every non-default branch:

```bash
gh api repos/<owner>/<repo>/branches --jq '.[].name' | grep -vE '^(main|master)$'
# for each branch found:
gh pr list --repo <owner>/<repo> --head <branch> --state all --json state,number
```

Then, per branch:

- **Merged** (PR state MERGED) → delete it (remote + local). Free to do; the work is in `main`.
- **No PR, or an unmerged/closed PR, with real commits** → 🚨 the dangerous case. **Never delete it silently.** Open a PR for it, or surface it and ask the user. Deleting unmerged work is the one thing this discipline never does on its own.

That asymmetry is the whole point: **merged branches are deleted freely; unmerged branches are never deleted silently.** "No orphan branches" means *resolve* every branch — not *delete* every branch.

## 4. Before pushing — a last look

Re-read the staged diff before pushing: no debug leftovers, no secrets, no absolute home paths (`/Users/<name>/`, `/home/<name>/`). For any file with machine-checkable constraints (a schema, a length limit), validate locally rather than discovering the failure in CI. Validate-once-trust-never — every push, no "this change is too small to check."

## Anti-patterns

- ❌ "Probably on main" — proceeding without checking
- ❌ Trusting a UI's PR/branch state over `gh`/`git` (UIs cache and go stale)
- ❌ `git checkout main` with a dirty working tree (silent data-loss path)
- ❌ Leaving a merged branch around "just in case" — it's clutter; delete it
- ❌ Deleting an **unmerged** branch (the real data-loss path — surface it, never delete silently)
- ❌ Skipping a local check because "this change is too small"
