# Branch + working-tree hygiene

A short discipline that prevents the recurring "drift" problem: editing on a stale branch, assuming working-tree state without checking, or pushing without a last look. Three checkpoints.

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

Uncertainty is a signal to verify, not to assume the previous state still holds.

## 2. After a merge — return to a clean main immediately

When the user signals a PR merged — directly ("merged", "approved + merged") or indirectly ("no PRs left", a screenshot of an empty PR inbox, "why is that branch still here") — your next action in each affected repo is:

```bash
git checkout main
git pull --ff-only
```

Detection is semantic, not lexical — equivalent phrasings in any language map to the same intent. Don't defer it to "later"; don't assume a background update caught it (it pulls whatever branch you are on). Don't `git branch -d` the merged branch automatically (harmless to leave; the user owns pruning). Don't switch branches with a dirty tree (loss risk — surface first).

## 3. Before pushing — a last look

Re-read the staged diff before pushing: no debug leftovers, no secrets, no absolute home paths (`/Users/<name>/`, `/home/<name>/`). For any file with machine-checkable constraints (a schema, a length limit), validate locally rather than discovering the failure in CI. Validate-once-trust-never — every push, no "this change is too small to check."

## Anti-patterns

- ❌ "Probably on main" — proceeding without checking
- ❌ `git checkout main` with a dirty working tree (silent data-loss path)
- ❌ Auto-deleting local branches with no user signal
- ❌ Skipping a local check because "this change is too small"
