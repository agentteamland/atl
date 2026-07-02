# `/create-pr`

Take working-tree changes (uncommitted, or recently committed to the default branch), derive an appropriate branch name + commit message + PR title from the diff, run [`/drain`](/skills/drain) so pending learnings ride along in the same PR, run an adversarial AI review chain (generic baseline + any team-declared specialists + a refute-to-keep verification pass), commit + push, and open a PR. Optionally enable GitHub auto-merge with a bounded polling + auto-fix loop. Always return the user to the target branch at end-of-work.

This skill is the deterministic "ship a piece of work" flow — it applies the [`branch-hygiene`](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md), [`learning-capture`](https://github.com/agentteamland/atl/blob/main/core/rules/learning-capture.md), and [`karpathy-guidelines`](https://github.com/agentteamland/atl/blob/main/core/rules/karpathy-guidelines.md) rules, so you don't re-derive them every PR.

Ships as a global skill in the [atl monorepo](https://github.com/agentteamland/atl).

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--auto-merge` | OFF | Enable GitHub auto-merge (`gh pr merge --auto --squash`); poll + auto-fix until merged or terminal failure |
| `--no-review` | OFF (review on) | Skip the entire review chain (generic + every team reviewer + the adversarial verify pass) |
| `--no-auto-fix` | OFF (fix on) | During the polling loop, do not attempt to fix CI/merge failures; surface to the user instead |
| `--no-drain` | OFF (drain on) | Skip folding pending learnings into the knowledge base |
| `--no-docs` | OFF (docs on) | Skip the docs-impact pass that keeps the docs site in sync with the change |
| `--timeout {min}` | 10 | Polling timeout in minutes; 1-minute interval, applies to both `--auto-merge` and manual-merge wait |

## Flow

The flow runs sequentially. Each step has a clear precondition and postcondition; if a precondition fails, the skill surfaces the issue and stops instead of proceeding.

### Step 1 — Pre-checks

- Current directory is inside a git repo
- Working tree has changes OR the current branch has unpushed commits (if neither: "Nothing to do — working tree clean and branch up-to-date")
- Determine the repo's default branch (`main`/`master`)

### Step 2 — Determine target branch

The "target branch" is what the PR merges into AND what the user returns to at end-of-work.

- **On the default branch** → target = default branch.
- **On a non-default branch** → `AskUserQuestion` with three choices: upstream branch (auto-detected), the default branch, or a free-text Other.

### Step 3 — Generate branch name + commit message

Analyze staged + unstaged + untracked changes:

- **Type** — one of `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf` (heuristic-derived from the diff: new agent/skill/rule or feature → `feat`; bug-fix language → `fix`; only `*.md` → `docs`; etc.)
- **Scope** — most-specific scope covering the change (skill name, rule name, agent name, CLI command, repo area)
- **Slug** — kebab-case, ≤ 50 chars, ASCII

Outputs:

- **Branch name** — `{type}/{slug}` (e.g., `feat/create-pr-skill`, `fix/install-404`, `docs/translate-tr-en`)
- **Commit subject** — `{type}({scope}): {one-line summary}` under 70 chars
- **Commit body** — 2–4 bullets describing the change

The skill does **not** ask the user to confirm names — it generates and proceeds.

### Step 4 — Drain pending learnings (unless `--no-drain`)

Invokes [`/drain`](/skills/drain) so any learnings captured during the session ride along in the same PR:

- `/drain` reads the durable learning queue, routes each pending item to the wiki / journal / agent knowledge base, and acks it.
- An empty queue is a no-op.
- If `/drain` isn't installed, the step is skipped with a one-line notice — it never fails the skill.

v2's marker is plain prose (`<!-- learning: free text -->`); `/drain` infers where each learning belongs, so there's no separate doc-draft step to review. See [`atl learnings`](/cli/learnings) and the [`/drain` skill](/skills/drain).

### Step 4.5 — Docs-impact pass (unless `--no-docs`)

Keeps the docs site in lockstep with the change — the change-time layer of [docs-sync](/cli/docs), so drift never forms. It pre-flights and skips cheaply unless **both** hold: the repo has a docs site (`docs/site/.vitepress`), and the diff touches a doc-affecting surface (`cli/`, `core/`, `docs/`, or a command/skill/rule/concept).

When it applies:

1. **Deterministic first** — runs [`atl docs check`](/cli/docs) and fixes every FAIL (a missing page, an absent TR mirror, a stale install instruction). Mechanical, zero-false-positive.
2. **Semantic, grep-grounded** — for each page that plausibly describes what the diff changed, reads it against the new code and quotes the source verbatim before claiming drift (the ~40% hallucination guard). Updates the affected pages (EN + the TR mirror).
3. Stages the doc edits so they ride the **same PR** — docs and code land atomically.

Boring diffs cost nothing (the pre-flight skips them). This is the LLM half the deterministic [docs CI gate](/cli/docs) can't do. See [`/docs-audit`](/skills/docs-audit) for the full-site backstop.

### Step 5 — Review chain (unless `--no-review`)

Three layers, executed sequentially — two finders, then an adversarial verifier:

**5a — Generic reviewer (always)**

Spawns a fresh-context sub-agent over the staged diff (fresh context so the review isn't biased by the model that wrote the diff), prompted with the four Karpathy guidelines:

- Think Before Coding (assumptions explicit?)
- Simplicity First (over-engineering?)
- Surgical Changes (drive-by edits? orphans?)
- Goal-Driven Execution (verifies against the goal? success criteria?)

Plus general code quality (naming, scope creep, security smells — secrets in logs, injection, hardcoded credentials — dead code, test coverage). Reports as 🔴 issues / 🟡 concerns / 🟢 good.

**5b — Team specialists (per installed team)**

For each installed team (look under `.claude/agents/` then `~/.claude/agents/` — project shadows global), the skill reads `team.json` `capabilities.review`:

- If it names an agent (e.g., `capabilities.review: "code-reviewer"`), that team agent runs against the same diff and produces a domain-specific review.
- If not declared, the team is skipped — 5a is the platform-wide baseline.

**5c — Adversarial verify (always)**

The finders are author-adjacent optimists, so their raw findings aren't presented directly. One fresh-context sub-agent runs over the **consolidated 5a + 5b findings** (the findings list, not the whole diff again) with two jobs:

- **Evidence gate** — every finding must cite concrete evidence (a `file:line`, a grep pattern, or a failing test/command). A finding that names none is dropped, not shown — the [`/docs-audit`](/skills/docs-audit) "no claim without a verbatim quote" discipline, applied to code review.
- **Refute-to-keep** — for each surviving finding, the agent reads the cited lines and tries to refute it; only findings that survive are kept, with severity re-weighed. When 5a and a 5b specialist disagree on severity, this pass is the tiebreaker.

It's one extra agent over a small findings list, not a second whole-diff review. Only the surviving, evidence-backed findings are shown, with a count of how many were dropped or refuted.

The consolidated report is shown to the user: continue / abort / edit.

### Step 6 — Commit + push

```bash
git checkout -b {branch-name}
git add -A
git commit -m "{commit-subject}

{commit-body}"

git push -u origin {branch-name}
```

### Step 7 — Open PR

```bash
gh pr create \
  --base {target-branch} \
  --title "{commit-subject}" \
  --body "<Summary bullets + Test plan checklist>"
```

The skill does **not** pass `--assignee` or `--reviewer`.

### Step 8 — `--auto-merge` (only if the flag is set)

```bash
gh pr merge {N} --auto --squash
```

This is the **only allowed merge invocation in the entire skill set.** It does not merge immediately — GitHub waits for required checks, then merges, so the branch-protection gate is preserved. The user opted in by passing the flag.

### Step 9 — Polling + auto-fix loop (if `--auto-merge`)

Polls PR state at 1-minute intervals, up to `{timeout}` attempts (default 10). State machine:

| State | Action |
|---|---|
| `MERGED` | Success — proceed to end-of-work |
| `CLOSED` | User closed without merge — exit cleanly, no end-of-work |
| `*CLEAN` / `*HAS_HOOKS` | Healthy state, just waiting for checks — continue polling |
| `*BLOCKED` / `*UNSTABLE` / `*DIRTY` / `*BEHIND` | CI failure or merge conflict — `handle_failure` |

#### `handle_failure` classification

**In-scope (auto-fix attempted, max 3):**

- Merge conflicts — fetch latest target, attempt 3-way merge
- Lint / format failures — run the project's formatter (auto-detected: `package.json scripts.lint`, `.prettierrc`, `gofmt`, `cargo fmt`, etc.)
- Trivial type errors / missing imports — apply compiler-suggested fixes

**Out-of-scope (notify and stop):**

- Real test failures (assertions, regressions in existing tests)
- Non-trivial build errors
- Infrastructure / CI config issues
- Missing required reviews (human reviewers blocking)

After 3 in-scope fix attempts, the skill stops and reports.

### Step 10 — Manual-merge polling (only if `--auto-merge` was NOT set)

The skill polls for merge anyway — the user might merge manually within `{timeout}` minutes. Same MERGED / CLOSED / timeout exits.

### Step 11 — End-of-work (universal)

Reached only when the PR was merged successfully:

```bash
git checkout {target-branch}
git pull
```

The user ends the skill on the target branch, with the merged change incorporated, ready for the next task.

### Step 12 — Final report

```
✅ /create-pr complete
   Branch:      feat/create-pr-skill
   PR:          https://github.com/.../pull/N
   Review:      generic + 1 team reviewer (example-team) + adversarial verify
                3 issues, 1 concern surviving (2 dropped: no evidence), all addressed
   Drain:       /drain ran — 2 wiki pages updated, 1 journal entry
   Auto-merge:  enabled, merged after 4 min (1 auto-fix: prettier formatting)
   End-of-work: returned to main, pulled latest
```

## Important constraints

1. **Never merge directly.** The skill uses `gh pr merge --auto --squash` (auto-merge enable) only when `--auto-merge` is passed. An immediate `gh pr merge --squash`/`--merge`/`--rebase` (without `--auto`) is **always forbidden** — auto-merge preserves the required-check gate, and the user opted in by typing the flag.
2. **Idempotent drain.** Running `/drain` here is safe — it processes only unacknowledged queue entries.
3. **team.json validation.** If the staged diff touches a `team.json`, the skill verifies that the file parses, has a `name` field, and all declared assets exist on disk before push.
4. **Branch hygiene before start.** Before deriving the new branch, the skill verifies the local default branch is current with origin; if behind, it fast-forwards first (per [`branch-hygiene`](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md)).
5. **No silent partial failures.** If any step fails, the skill stops and reports — the user always knows where they are.

## Related

- [`/drain`](/skills/drain) — invoked at Step 4 to fold pending learnings into the knowledge base
- [`branch-hygiene` rule](https://github.com/agentteamland/atl/blob/main/core/rules/branch-hygiene.md) — keep the base branch current before branching
- [`karpathy-guidelines` rule](https://github.com/agentteamland/atl/blob/main/core/rules/karpathy-guidelines.md) — the review prompt's foundation

## Source

- Spec: [core/skills/create-pr/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/create-pr/SKILL.md)
