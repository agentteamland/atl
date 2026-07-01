---
name: create-pr
description: "Ship working-tree changes: auto-named branch, drain pending learnings, an adversarial AI review chain (generic baseline + team specialists + a refute-to-keep pass), commit, push, open a PR. Optional --auto-merge with a polling + auto-fix loop. Returns you to the target branch."
argument-hint: "[--auto-merge] [--no-review] [--no-auto-fix] [--no-drain] [--no-docs] [--timeout N]"
---

# /create-pr

## Purpose

Take the working-tree changes, derive a branch name + commit message + PR title from the diff, fold any pending learnings into the knowledge base, run an AI review chain (generic baseline + any team specialists), commit + push, and open a PR. Optionally enable GitHub auto-merge with a bounded polling + auto-fix loop. Always return to the target branch at the end.

This is the deterministic "ship a piece of work" flow — it applies the `branch-hygiene`, `learning-capture`, and `karpathy-guidelines` rules so you don't re-derive them every PR.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--auto-merge` | off | Enable GitHub auto-merge; poll + auto-fix until merged or terminal failure |
| `--no-review` | review on | Skip the entire review chain |
| `--no-auto-fix` | fix on | During polling, don't try to fix CI/merge failures — surface them instead |
| `--no-drain` | drain on | Skip folding pending learnings into the knowledge base |
| `--no-docs` | docs on | Skip the docs-impact pass (keeping the docs site in sync with the change) |
| `--timeout N` | 10 | Polling timeout in minutes (1-minute interval) |

## Flow

Sequential. Each step has a precondition; if it fails, surface and stop rather than proceeding.

### 1. Pre-checks
- Inside a git repo (`git rev-parse --git-dir`).
- The working tree has changes OR the branch has unpushed commits. If neither: "Nothing to do — working tree clean and branch up-to-date."
- Resolve the default branch: `gh repo view --json defaultBranchRef -q .defaultBranchRef.name` (usually `main`).

### 2. Target branch
The branch the PR merges into AND the branch you return to.
- On the default branch → target = default.
- On a non-default branch → `AskUserQuestion`: upstream branch (auto-detected) / the default branch / other (free text).

### 3. Branch name + commit message
From `git diff --stat HEAD`, `git diff --name-only HEAD`, `git status -s`:
- **Type** — `feat` (new agent/skill/rule or feature), `fix` (bug), `docs` (only docs), `chore`, `refactor`, `test`, `perf`.
- **Scope** — the most specific area the change covers.
- **Slug** — kebab-case, ≤ 50 chars, ASCII.

Branch `{type}/{slug}`; commit subject `{type}({scope}): {summary}` (< 70 chars); body 2–4 bullets. Generate and proceed — don't ask for confirmation.

### 4. Drain pending learnings (unless `--no-drain`)
Invoke `/drain` so any queued markers are folded into wiki / journal / agent knowledge base and ship in the same PR. Empty queue → no-op. If `/drain` isn't installed, skip with a one-line notice — don't fail the skill.

### 4.5. Docs-impact pass (unless `--no-docs`)
Keep the docs site in lockstep with the change — the change-time layer of docs-sync, so drift never forms. **Pre-flight, skip cheaply unless both hold:** the repo has a docs site (`docs/site/.vitepress`), AND the diff touches a doc-affecting surface (`cli/`, `core/`, `docs/`, or a command/skill/rule/concept).

When it applies:
1. **Deterministic first** — run `atl docs check`; fix every FAIL (a missing page, an absent TR mirror, a stale install instruction). Mechanical, zero-false-positive.
2. **Semantic, grep-grounded** — for each page that plausibly describes what the diff changed, read it against the new code; **quote the source verbatim before claiming drift** (the ~40% hallucination guard). Update the affected pages (EN + the TR mirror).
3. Stage the doc edits so they ride **this same PR** — docs and code land atomically.

Boring diffs cost nothing (the pre-flight skips them). If `atl docs check` isn't installed, skip with a one-line notice — don't fail the skill. This is the LLM half the deterministic CI docs gate can't do.

### 5. Review chain (unless `--no-review`)

**5a — Generic reviewer (always).** Spawn a fresh-context agent over the staged diff. Fresh context so the review isn't biased by the model that wrote the diff. Prompt it to apply the four Karpathy guidelines (think-before-coding, simplicity, surgical changes, goal-driven) plus general quality: naming clarity, scope creep, security smells (secrets in logs, injection, hardcoded credentials), dead code, and test coverage. Ask for `🔴 issues` / `🟡 concerns` / `🟢 good`, terse.

**5b — Team specialists.** For each installed team (look under `.claude/agents/` then `~/.claude/agents/` — project shadows global), read its `team.json` `capabilities.review`; if it names an agent, spawn that agent over the same diff for a domain-specific review. A team with no `capabilities.review` is skipped — 5a is the baseline.

**5c — Adversarial verify (always).** The finders are author-adjacent optimists; don't present raw findings. Spawn one fresh-context agent over the *consolidated 5a + 5b findings* (the findings list, not the whole diff again) with two jobs:
- **Evidence gate (drop):** every finding must cite concrete evidence — a `file:line`, a grep pattern, or a failing test/command. A finding that names none is dropped, not shown (the `/docs-audit` "no claim without a verbatim quote" discipline, applied to code review).
- **Refute-to-keep:** for each surviving finding, read the cited lines and try to refute it — keep only the findings that survive, re-weighing severity on what's left. When 5a and a 5b specialist disagree on severity, this pass is the tiebreaker: its evidence-weighed verdict wins.

One extra agent over a small findings list — not a second whole-diff review. Present only the surviving, evidence-backed findings; note how many raw findings were dropped (no evidence) or refuted, so the pass is visible. Ask continue / abort / edit.

### 6. Commit + push
```bash
git checkout -b {branch}
git add -A
git commit -m "{subject}

{body}"
git push -u origin {branch}
```

### 7. Open PR
```bash
gh pr create --base {target} --title "{subject}" --body "<Summary bullets + Test plan checklist>"
```
Capture the PR number + URL. Don't pass `--assignee`/`--reviewer`.

### 8. `--auto-merge` (only if the flag is set)
```bash
gh pr merge {N} --auto --squash
```
This is the **only allowed merge invocation**. It doesn't merge immediately — GitHub waits for required checks, then merges, so the branch-protection gate is preserved. The user opted in by passing the flag.

### 9. Polling + auto-fix (if `--auto-merge`)
Poll at 1-minute intervals up to `{timeout}`.
- `MERGED` → end-of-work (Step 11).
- `CLOSED` → exit cleanly, no end-of-work.
- Healthy/waiting → keep polling.
- CI failure / conflict → `handle_failure` (skip if `--no-auto-fix`):
  - **In-scope (auto-fix, max 3):** merge conflict (fetch + 3-way, accept-both where mechanical), lint/format (run the project formatter), trivial type error / missing import. Fix → commit → push → resume.
  - **Out-of-scope (report + stop):** real test failures, non-trivial build errors, CI-config issues, missing required reviews.

### 10. Manual-merge polling (if `--auto-merge` was NOT set)
Poll for the user to merge within `{timeout}`. `MERGED` → end-of-work; `CLOSED` → exit; timeout → report the URL and stop.

### 11. End-of-work (universal — only after a successful merge)
```bash
git checkout {target}
git pull
```
Return to a clean target branch with the change incorporated. If the pull fails (rare), surface the error and leave the user where they are.

### 12. Final report
One block: branch, PR URL, review summary (surviving issues/concerns + how many findings the adversarial pass dropped/refuted + whether addressed), drain result, auto-merge result, end-of-work. State explicitly anything skipped via a `--no-X` flag.

## Constraints

1. **Never direct-merge.** Use only `gh pr merge --auto` (when `--auto-merge` is passed). An immediate `gh pr merge --squash/--merge/--rebase` is forbidden — auto-merge preserves the required-check gate, and the user opted in with the flag.
2. **No silent partial failures.** Any step fails → stop and report; the user always knows where they are.
3. **Branch hygiene first.** Before branching off, verify the local default branch is current with origin; fast-forward if behind (per `branch-hygiene`). Don't build on a stale base.
