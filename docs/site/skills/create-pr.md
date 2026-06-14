# `/create-pr`

Take working-tree changes (uncommitted, or recently committed to the default branch), derive an appropriate branch name + commit message + PR title from the diff, run [`/save-learnings`](/skills/save-learnings) so wisdom rides along in the same PR, run an AI review chain (generic baseline + any team-declared specialists), commit + push, open a PR. Optionally enable GitHub auto-merge with a bounded polling + auto-fix loop. Always return the user to the target branch at end-of-work.

This skill is the deterministic "ship a piece of work" flow — it consumes the disciplines defined by `team-repo-maintenance`, `branch-hygiene`, `learning-capture`, `docs-sync`, and `karpathy-guidelines`, so the user does not have to re-derive them every PR.

Ships as a global skill in [core](https://github.com/agentteamland/core) since `core@1.4.0`.

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--auto-merge` | OFF | Enable GitHub auto-merge (`gh pr merge --auto --merge`); poll + auto-fix until merged or terminal failure |
| `--no-review` | OFF (review on) | Skip the entire review chain (generic + every team reviewer) |
| `--no-auto-fix` | OFF (fix on) | During the polling loop, do not attempt to fix CI/merge failures; surface to the user instead |
| `--no-learning` | OFF (learning on) | Skip `/save-learnings` + doc-impact pipeline |
| `--timeout {min}` | 10 | Polling timeout in minutes; 1-minute interval, applies to both `--auto-merge` and manual-merge wait |

## Flow

The flow runs sequentially. Each step has a clear precondition and postcondition; if a precondition fails, the skill surfaces the issue and stops instead of proceeding.

### Step 1 — Pre-checks

- Current directory is inside a git repo
- Working tree has changes OR the current branch has unpushed commits
- Determine the repo's default branch (`main`/`master`)

### Step 2 — Determine target branch

The "target branch" is what the PR merges into AND what the user returns to at end-of-work.

- **On the default branch** → target = default branch.
- **On a non-default branch** → `AskUserQuestion` with three choices: upper branch (auto-detected), default branch, or a free-text Other.

### Step 3 — Generate branch name + commit message

Analyze staged + unstaged + untracked changes:

- **Type** — one of `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `style` (heuristic-derived from the diff: new file under `skills/agents/rules/` → `feat`; bug-fix language → `fix`; only `*.md` → `docs`; etc.)
- **Scope** — most-specific scope covering the change (skill name, rule name, agent name, CLI command, repo area)
- **Slug** — kebab-case, ≤ 50 chars, ASCII

Outputs:

- **Branch name** — `{type}/{slug}` (e.g., `feat/create-pr-skill`, `fix/winget-403`, `docs/translate-trk-en`)
- **Commit subject** — `{type}({scope}): {one-line summary}` under 70 chars
- **Commit body** — 2–4 bullets describing the change. Last line is a "Discovered via" context if invoking from a team-repo context (per [team-repo-maintenance §3](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md))

The skill does **not** ask the user to confirm names — it generates and proceeds.

### Step 4 — Save learnings (unless `--no-learning`)

Invokes [`/save-learnings`](/skills/save-learnings) in manual mode (analyzes the live conversation):

- Writes wiki / journal / agent children / skill learnings updates (project-local)
- For each `<!-- learning doc-impact: readme/docs/breaking -->` marker, prepares a doc draft
- Each draft is presented inline to the user for accept / reject / edit:

```
📝 Doc draft for README.md:
<diff>
Accept? (y/n/edit)
```

Accepted drafts get staged; rejected drafts are discarded.

### Step 5 — Review chain (unless `--no-review`)

Two layers, executed sequentially:

**5a — Generic reviewer (always)**

Invokes a fresh-context sub-agent (`subagent_type: general-purpose`) with a Karpathy-grounded review prompt:

- Think Before Coding (assumptions explicit?)
- Simplicity First (over-engineering?)
- Surgical Changes (drive-by edits? orphans?)
- Goal-Driven Execution (verifies against goal? success criteria?)

Plus general code quality (naming, scope creep, security smells, dead code, test coverage). Reports as 🔴 Issues / 🟡 Concerns / 🟢 Looks good.

**5b — Team reviewers (per installed team)**

For each installed team, the skill reads `team.json` and looks for `capabilities.review`:

- If declared (e.g., `capabilities.review: "code-reviewer"`), the named team agent runs against the same diff and produces a domain-specific review
- If not declared, skipped silently — there is no fallback per team. The generic reviewer is the platform-wide baseline.

The consolidated report is shown to the user. Continue / abort / edit.

### Step 6 — Commit + push

```bash
git checkout -b {branch-name}
git add -A
git commit -m "{commit-subject}

{commit-body}

{discovered-via-line if applicable}"

git push -u origin {branch-name}
```

### Step 7 — Open PR

```bash
gh pr create \
  --base {target-branch} \
  --title "{commit-subject}" \
  --body "..."
```

Body has Summary, Discovered via, Version bump (if applicable), Test plan. Per [team-repo-maintenance §4](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md), the skill does **not** pass `--assignee` or `--reviewer`.

### Step 8 — `--auto-merge` polling (only if flag set)

```bash
gh pr merge {N} --auto --merge
```

This is the **only allowed merge invocation in the entire skill set.** It does not merge immediately — GitHub waits for required checks then merges automatically. Branch protection's check gate is preserved.

### Step 9 — Polling + auto-fix loop (if `--auto-merge`)

Polls PR state at 1-minute intervals, up to `{timeout}` attempts (default 10). State machine:

| State | Action |
|---|---|
| `MERGED` | Success — proceed to end-of-work |
| `CLOSED` | User closed without merge — exit cleanly, no end-of-work |
| `*CLEAN` / `*HAS_HOOKS` | Healthy state, just waiting for checks — continue polling |
| `*BLOCKED` / `*UNSTABLE` / `*DIRTY` / `*BEHIND` | CI failure or merge conflict — `handle_failure` |

#### `handle_failure` classification

**In-scope (auto-fix attempted):**

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
git pull origin {target-branch}
```

The user ends the skill on the target branch, with the merged change incorporated, ready for the next task.

### Step 12 — Final report

```
✅ /create-pr complete
   Branch:      feat/create-pr-skill
   PR:          https://github.com/.../pull/N
   Review:      generic + 1 team reviewer (software-project-team)
                3 issues, 1 concern, all addressed
   Learnings:   /save-learnings ran — 2 wiki pages updated, 1 README draft accepted
   Auto-merge:  enabled, merged after 4 min (1 auto-fix: prettier formatting)
   End-of-work: returned to main, pulled latest
```

## Important constraints

1. **Never merge directly.** This skill uses `gh pr merge --auto --merge` (auto-merge enable) only when `--auto-merge` flag is passed. Direct merge (`--merge`/`--squash`/`--rebase` without `--auto`) is **always forbidden** — see [team-repo-maintenance "PR merge discipline"](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md). The user has explicitly authorized auto-merge by typing the flag; this is the documented exception.
2. **Discovered-via context.** When invoked from a shared / team repo, the skill follows team-repo-maintenance discipline: include "Discovered via" in PR body, bump version, conventional commit. Detection: cwd under `~/.claude/repos/agentteamland/` or matches a known shared-repo pattern.
3. **Idempotent save-learnings.** Running `/save-learnings` again here is safe — it appends, deduplicates, and processes only fresh content.
4. **Schema validation.** If the staged diff touches a `team.json`, the validator runs before push (`~/.claude/repos/agentteamland/core/scripts/validate-team-json.sh`).
5. **Branch hygiene before start.** Before deriving the new branch, the skill verifies the local default branch is current with origin. If not, fast-forward first.
6. **No silent partial failures.** If any step fails, the skill stops and reports.

## Related

- [`/save-learnings`](/skills/save-learnings) — invoked at Step 4
- [team-repo-maintenance rule](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md) — governance for shared repos
- [karpathy-guidelines rule](https://github.com/agentteamland/core/blob/main/rules/karpathy-guidelines.md) — review prompt's foundation

## Future evolution (v2)

- **Domain-aware review routing** — each team agent declares `domains: ["*.tsx", ...]` glob; skill matches the diff's file types and invokes only relevant agents
- **Parallel team review** — run team reviewers concurrently
- **Auto-fix scope expansion** — extend in-scope to test failures where the test was added in the same diff

## Source

- Spec: [core/skills/create-pr/skill.md](https://github.com/agentteamland/core/blob/main/skills/create-pr/skill.md)
