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

## 2.5. Prune the LOCAL copy — `--delete-branch` does not

`gh pr merge --delete-branch` removes the **remote** branch and leaves your local one behind, still pointing at the pre-squash commits. Nothing surfaces it: `git branch` looks the same as it did, and §3's audit reads the *remote* via `gh`, so a local branch whose remote is gone is invisible to the one check meant to catch orphans.

That is not hypothetical. In `agentteamland/workspace` on 2026-08-07 a single clone had accumulated **27** such branches over several weeks — every one merged, every one's remote already deleted, none of them visible to any audit anyone was running.

Fetch with `--prune` so gone upstreams are marked, then act on the mark:

```bash
git fetch --prune origin
git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads | grep gone
```

For each one, confirm the PR merged before deleting — `-D` is required because a squash-merge gives the branch different commit hashes, which is exactly why `-d` refuses and why the count of "commits not in main" is **not** evidence of unmerged work:

```bash
gh pr list --repo <owner>/<repo> --head <branch> --state all --json state -q '.[0].state'
git branch -D <branch>   # only on MERGED
```

A branch with **no upstream at all** never went through this path — it was never pushed. Treat it as §3's dangerous case, not as clutter: check whether its content reached the base under different hashes before touching it.

Do this on a cadence rather than per-merge, since that is the failure mode: the per-merge step is the one that gets skipped, and nothing degrades until someone tries to read the branch list.

## 3. No orphan branches — every branch resolves to merged-or-deleted

A branch ends in exactly one of two states: **merged via a PR**, or **deleted**. There is no legitimate third state — a branch with no PR, a PR opened but never merged, or an abandoned experiment is *drift*, and may be lost work.

At the end of a unit of work — and especially at session end — audit every non-default branch:

```bash
gh api repos/<owner>/<repo>/branches --jq '.[].name' | grep -vE '^(main|master)$'
# for each branch found:
gh pr list --repo <owner>/<repo> --head <branch> --state all --json state,number
```

Then, per branch:

- **Something points at it by name** → 🛑 **not an orphan at all — leave it, whatever its merge state.** Check this **first**, because it is the only one of the three that a merged branch can fail. See below.
- **Merged** (PR state MERGED) → delete it (remote + local). Free to do; the work is in `main`.
- **No PR, or an unmerged/closed PR, with real commits** → 🚨 the dangerous case. **Never delete it silently.** Open a PR for it, or surface it and ask the user. Deleting unmerged work is the one thing this discipline never does on its own.

That asymmetry is the whole point: **merged branches are deleted freely; unmerged branches are never deleted silently.** "No orphan branches" means *resolve* every branch — not *delete* every branch.

### A branch can be fully merged and still load-bearing

"Merged → delete freely" is a rule about *work*, and some branches are not work — they are **infrastructure that something else names**. Deleting one is not clutter removal, it is breaking a live system, and the merge test says nothing about it: such a branch is often *identical* to `main`, which makes it look maximally safe to remove.

Before deleting any branch, ask what points at it **by name**:

| Pointer | Example |
|---|---|
| A config field | `.delivery/config.json` `branchPair` — `dev` / `release`, required by the two-branch flow |
| A Pages source | GitHub Pages serving from a branch — `gh api repos/<o>/<r>/pages --jq .source.branch` |
| Branch protection, or the repo default | `gh api repos/<o>/<r> --jq .default_branch` |
| A CI trigger, or a test harness | a workflow's `on.push.branches`; an e2e suite that pushes to a fixture branch |

Both halves were measured on 2026-08-07 while cleaning `agentteamland/*`, and both would have been destroyed by a plain "delete every merged branch" sweep. `agentteamland/atl`'s `dev` had just been fast-forwarded to `main` — identical tree, zero commits either way, so *maximally* merged — and it is the integration branch every `atl work dispatch` worktree is cut from. And the archived `agentteamland/docs` carries `pages-redirect`, which is what GitHub Pages actually serves: two files that redirect the old v1 site to the live docs. Deleting it would have 404'd every old bookmark, silently, with nothing in the repo suggesting the branch mattered.

The check is one API call per repo and it is cheap. Run it **before** the merge test, not after: a load-bearing branch that also happens to be merged will pass the merge test and be gone before anyone asks the question.

## 4. Before pushing — a last look

Re-read the staged diff before pushing: no debug leftovers, no secrets, no absolute home paths (`/Users/<name>/`, `/home/<name>/`). For any file with machine-checkable constraints (a schema, a length limit), validate locally rather than discovering the failure in CI. Validate-once-trust-never — every push, no "this change is too small to check."

## Anti-patterns

- ❌ "Probably on main" — proceeding without checking
- ❌ Trusting a UI's PR/branch state over `gh`/`git` (UIs cache and go stale)
- ❌ `git checkout main` with a dirty working tree (silent data-loss path)
- ❌ Leaving a merged branch around "just in case" — it's clutter; delete it
- ❌ Deleting an **unmerged** branch (the real data-loss path — surface it, never delete silently)
- ❌ Deleting a **merged** branch that something names — an integration branch, a Pages source, a CI target. Merged says the work is safe, not that the branch is unused
- ❌ Auditing orphans with `gh` alone — that reads the remote, and a local branch whose upstream is gone is invisible to it (27 accumulated that way)
- ❌ Skipping a local check because "this change is too small"
