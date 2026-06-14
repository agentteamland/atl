# Team-repo maintenance

Governance for changes to **shared `agentteamland/` public repos** — team repos (cli, core, brainstorm, rule, team-manager, software-project-team, design-system-team, registry, docs, starter-extended, create-project, workspace, .github). The rule covers the discipline gap that existed before branch protection was added to all public repos on 2026-04-24.

Branch protection is the **safety net** — it refuses direct commits to `main` on every public repo in the org. This rule is the **method**: how to produce a clean change that satisfies the safety net AND is useful to the next maintainer reading the git log.

The canonical rule lives at [`core/rules/team-repo-maintenance.md`](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md). This page is the user-facing summary.

## When this applies

Any time you (or Claude on your behalf) modify a file in a cached agentteamland repo under `~/.claude/repos/agentteamland/{team}/`. Includes both team repos (software-project-team, design-system-team, etc.) and global repos (core, brainstorm, rule, team-manager, cli, docs, registry, workspace).

**Does NOT apply to:**

- Your own local project's `.claude/` directory (that's project memory, not shared)
- `homebrew-tap` / `scoop-bucket` / `winget-pkgs` (goreleaser-managed; direct push allowed for the release pipeline)

## Five fixed steps

Every shared-repo change follows the same five steps. They are sequential — each builds on the previous.

### 0. Run `/save-learnings` before opening the PR

Before steps 1–4, run [`/save-learnings`](/skills/save-learnings) as the **last commit on the feature branch**. This captures wisdom from the work that's about to ship and lets it ride along in the same PR.

Why this timing:

- The PR boundary is a natural crystallization moment — decisions are concrete
- `save-learnings` outputs that touch the **current PR's repo** (agent.md updates, README doc-impact drafts, new children files, known-issues entries) automatically commit on the feature branch and ship in the PR
- Review covers both the work AND the extracted wisdom in one atomic unit

The multi-repo caveat: `/save-learnings` often produces outputs that span repos. Only the outputs touching the **current PR's repo** ride along; other-repo outputs still need their own PR flow.

This rule is paired with the inline `<!-- learning -->` marker + `SessionStart` hook (see [learning-capture rule](https://github.com/agentteamland/core/blob/main/rules/learning-capture.md)). The PR-time `/save-learnings` captures crystallized learnings from work that's about to ship; the SessionStart hook on the *next* session catches markers from work that already shipped (review feedback, conflict-resolution insights, post-merge discoveries). Complementary, not redundant — keep both.

### 1. Bump version (`team.json` for teams; `internal/config.Version` for CLI)

Follow strict SemVer:

| Bump | When | Example |
|---|---|---|
| **Patch** (0.4.1 → 0.4.2) | Bug fix, no API change, behavior restored to advertised | `fix(dst-new-ds): Q3 cap` |
| **Minor** (0.4.2 → 0.5.0) | New skill / agent / rule / command, backward-compatible | `feat(core): new rule learning-capture` |
| **Major** (0.4.2 → 1.0.0) | Breaking: removed/renamed command, incompatible config, behavior change users depend on | `feat(cli)!: rename atl install-team → atl install` |

For the CLI, version lives in `internal/config/config.go` (ldflags override at build time via goreleaser tag). For teams, version lives in `team.json`.

**Never** ship a behavior change without a version bump — it silently breaks `atl update`'s `X → Y` notification, defeating the whole update pipeline.

#### When NOT to bump

Genuinely docs-only PRs (translations, typo fixes, README updates, comment changes) carry no behavior delta and don't need to surface in `atl update`'s `X → Y` notification. If `main` carries pre-existing schema violations on `team.json` (most often the 200-char description ceiling), bumping forces a same-PR trim that bloats scope. The pragmatic move:

1. **Skip the bump** on the docs-only PR.
2. **Note the skip explicitly** in the PR body — e.g., "Version bump: skipped — `main` carries pre-existing description-length violations; will be addressed in a follow-up `chore: trim ...` PR."
3. **Open a dedicated** `chore: trim team.json descriptions` PR with the bump.

The "version bump for every change" rule is about ensuring users see meaningful behavior deltas — not about ceremonial cadence on cosmetic edits.

#### `team.json` format conventions

When you edit `team.json`:

1. **Pretty-print, multi-line objects** with 2-space indent. Each agent / skill / keyword on its own line.
2. **`—` Unicode escape for em-dash characters** in description strings (instead of literal `—`).

Reformat:

```bash
python3 -m json.tool team.json > tmp && mv tmp team.json
```

When a feature branch and main both edit `team.json` and conflict over format (e.g., compact vs pretty-print), **adopt main's format and re-apply the feature branch's content on top**. Don't fight the convention — fold into it.

#### Validate `team.json` BEFORE pushing — non-negotiable

Schema constraints (defined in [`core/schemas/team.schema.json`](https://github.com/agentteamland/core/blob/main/schemas/team.schema.json)):

| Field | Rule |
|---|---|
| `description` (top-level) | 10–200 chars |
| `agents[*].description`, `skills[*].description`, `rules[*].description` | max 200 chars |
| `name` (top-level + agents/skills/rules) | kebab-case pattern `^[a-z][a-z0-9-]*$` |
| `keywords[*]` | 1–40 chars, max 20 keywords, unique |
| `version` | strict SemVer `MAJOR.MINOR.PATCH` |

The 200-char description maxLength has bitten three times in production. Each time the fix was a follow-up "trim description" commit. **No more.**

Before every push that touches a `team.json`:

```bash
~/.claude/repos/agentteamland/core/scripts/validate-team-json.sh path/to/team.json
```

Or from inside the core repo:

```bash
./scripts/validate-team-json.sh team.json
```

The script does a quick length check using only Python stdlib AND, if `ajv-cli` is on PATH, runs the full `ajv validate` that CI runs — so you get parity with what GitHub Actions will check. CI failure on a missed length check costs more time than the 2-second local run.

Even without ajv, the script's Python length check catches the constraint that has historically failed. Run it. Always.

### 2. Conventional commit format

```
<type>(<scope>): <one-line summary under 70 chars>

<body — WHY the change, not WHAT (diff shows the what)>
<context — which project / session revealed the need>

<footer — co-author, issue refs, breaking-change notes>
```

Types: `fix`, `feat`, `docs`, `chore`, `style`, `refactor`, `test`, `perf`. Add `!` after type for breaking: `feat(cli)!: …`.

Scope is the sub-module being changed (agent name, skill name, CLI command, repo area).

### 3. "Discovered via" context in the body

When a fix to a shared repo was found while working on a different project, **always** surface that context:

```
Discovered while scaffolding a design system for a downstream project.
The bug is not project-specific; every project running /dst-new-ds
hits the same wall.
```

This audit trail lets future-you (or another maintainer) understand the motivation without having to reconstruct it from memory. The team repo git log becomes self-documenting.

### 4. PR flow (default, enforced by branch protection)

All public `agentteamland/` repos require a pull request to merge to `main`. Direct pushes are refused by branch protection.

```bash
cd ~/.claude/repos/agentteamland/{team}
git checkout -b <fix|feat|chore>/<short-description>
# … make changes, bump version …
git add <files>
git commit -m "<conventional message>"
git push -u origin <branch-name>
gh pr create \
  --title "<type>(<scope>): <summary>" \
  --body  "<see PR body template below>"
```

**Do NOT add `--assignee` or `--reviewer` on your own PRs.** In the current solo-maintainer setup, Claude pushes under the maintainer's GitHub account, which makes the maintainer the PR author automatically:

- Author field already surfaces the PR in the maintainer's "Created by me" / "Involves me" dashboards
- Explicit `--assignee @me` is redundant (author == assignee) and pollutes the "Assigned to me" queue
- GitHub blocks requesting review from the PR author, so `--add-reviewer mkurak` silently fails on own PRs

(When / if a separate bot account is set up for Claude's pushes — so author ≠ maintainer — then `--reviewer mkurak` becomes possible and appropriate.)

#### PR body template

```markdown
## Summary
<What changed and why — 2-4 bullet points>

## Discovered via
<Which project / session / scenario revealed this>

## Version bump
<version: X.Y.Z → X.Y.Z+1> (patch | minor | major — reason)

## Test plan
- [ ] <how to verify the fix works>
- [ ] <regression check>
```

For solo maintainer flow, approvals are not required (count: 0) — the PR exists as **ceremony + audit trail**, not as external gate.

## 🚫 PR merge discipline — absolute, no exceptions

**Claude never merges pull requests.** This is non-negotiable and has no scope limit.

The prohibition covers any action that lands a PR on `main`:

- `gh pr merge` in any form (`--squash`, `--rebase`, `--merge`)
- `gh pr review --approve`
- Clicking "Merge pull request" via any MCP-driven browser
- Any equivalent server-side action through the GitHub REST / GraphQL API

Even when:

- The PR is trivial (one-line typo, formatting fix, broken link)
- `required_approving_review_count` is 0 (solo maintainer)
- The PR was authored by Claude itself
- The user said "push this" or "let's do it" earlier in the conversation
- Branch protection would allow admin bypass
- The maintainer is unreachable and a hotfix feels urgent

→ The answer is **still no**. Merging belongs to the human reviewer. If something is genuinely urgent, surface the PR URL and tell the user plainly that it's blocking — they click merge in 10 seconds.

### What IS allowed on PRs

- `gh pr create` — open PRs
- `gh pr edit` — fix typos in title/body, add/remove labels
- `gh pr list` / `gh pr view` / `gh pr diff` / `gh pr checkout` — read-only inspection
- `gh pr review --comment` — leave a feedback comment (NOT approve, NOT request-changes)

### What is NOT allowed

- Merging (see above)
- Approving (`--approve`)
- Requesting changes on someone else's PR (`--request-changes`)
- Closing (`gh pr close`) — destructive; only the author or user closes PRs
- Reopening a closed PR without explicit user instruction

### Exception: GitHub native auto-merge via tooling

There is **one narrow exception**: when the user explicitly authorizes auto-merge through tooling — most commonly [`/create-pr --auto-merge`](/skills/create-pr) — Claude may run `gh pr merge --auto --merge` to enable GitHub's native auto-merge.

This exception is bounded:

- **The flag must come from the user in the same turn.** "Earlier authorization" or "broad authorization" does NOT count. The user typing `--auto-merge` (or equivalent explicit instruction) is the gate.
- **`--auto` is required.** Anything that merges immediately (`gh pr merge` without `--auto`) is still forbidden.
- **Branch protection's check gate is preserved.** GitHub waits for required checks to pass before merging; if checks fail, the merge does not happen. The "review gate" is delegated to CI — not skipped.
- **No exception for `--approve`, `--squash` without `--auto`, or any other merge variant.** Only `gh pr merge --auto --merge` (or `--auto --squash` / `--auto --rebase` if the repo's settings dictate that strategy) is in scope.

If the user did NOT pass an explicit auto-merge flag through tooling, the original prohibition stands: surface the PR URL and stop.

### Handoff after opening a PR

After `gh pr create` succeeds, surface the URL and stop on that PR:

> PR opened: https://github.com/.../pull/N — let me know once you've reviewed and merged, and I'll continue.

Do not wait on CI to auto-merge on green. Do not self-approve. Do not re-invoke `gh pr merge` "because nothing has happened for 5 minutes." The merge action is the user's signal that they've reviewed; skipping that signal destroys the gate.

## Escape hatches (direct push only, never auto-merge)

### Admin bypass for direct push (emergency only)

Branch protection allows admin to push directly when `enforce_admins` is false (our default). This is the **only** form of PR-flow bypass — and it pushes a commit, not a merge. Use only when:

- Release-pipeline-breaking issue blocks `brew upgrade atl` / `scoop install atl`
- A revert must land within minutes to stop a public regression
- The maintainer explicitly instructs: "push directly, no PR"

When using this, still:

- Bump version
- Use conventional commit
- Follow up with a retrospective: `chore(postmortem): ...` commit or issue

The user-side expectation: admin bypass is **their** tool. Claude does not initiate it.

### Trivial changes still go through PR

Even for the smallest change — a typo, a broken link, a `gofmt` run — the path is:

1. Feature branch → commit → push → `gh pr create`
2. User reviews, clicks merge

There is no "too small for PR" category. The PR ceremony is cheap (30 seconds of work); the review gate catches mistakes that looked trivial but weren't.

## What this rule does NOT cover

- **Private project repos** — your own project's git workflow is up to you. This rule is specifically for `agentteamland/` public repos.
- **Release-pipeline repos** (`homebrew-tap`, `scoop-bucket`, `winget-pkgs`) — goreleaser auto-pushes; branch protection is intentionally not applied.
- **Tag-based releases** — when tagging `cli v0.2.1`, the tag push triggers goreleaser. No PR needed for tag creation itself (it points at an already-merged commit on main).

## Related

- [`/create-pr`](/skills/create-pr) — the skill that automates this discipline
- [`/save-learnings`](/skills/save-learnings) — the Step 0 invocation
- [karpathy-guidelines](/guide/karpathy-guidelines) — the coding-guideline counterpart of this governance rule
- Canonical source: [`core/rules/team-repo-maintenance.md`](https://github.com/agentteamland/core/blob/main/rules/team-repo-maintenance.md)

## History

Before 2026-04-24, team-repo writes via `/save-learnings` could land directly on `main` with an ad-hoc commit message. This let real bug fixes ship quickly but also meant version bumps were frequently forgotten (breaking `atl update`'s diff notifications) and commit-message discipline depended on whoever happened to be at the keyboard.

On 2026-04-24 the maintainer added branch protection to every public repo in the org and requested a principled workflow; this rule is that workflow. Direct push is enforcement-refused; PR ceremony is lightweight (no external approvals needed for solo maintainer) but mandatory — ensuring every team-repo change has a version bump, a conventional message, and a "Discovered via" context.
