# Governance

How shared `agentteamland/` repos are governed: branch protection at the GitHub level + the [team-repo-maintenance rule](/authoring/team-repo-maintenance) at the agent level. Together they ensure every change to a shared repo carries a version bump, a conventional commit, a "Discovered via" context, and goes through a PR — without depending on memory or goodwill.

This is the high-level "policy" view. For the procedural detail (how to actually open the PR, what the commit message should look like), see [`team-repo-maintenance`](/authoring/team-repo-maintenance).

## Two layers

| Layer | What | Why |
|---|---|---|
| **Branch protection** | Refuses bad shape at the server | Safety net — doesn't rely on remembering the rule |
| **`team-repo-maintenance` rule** | Produces good shape locally | Method — tells Claude how to write a PR-ready change |

Neither alone is sufficient:

- **Protection without a rule** → Claude tries to direct-push, git refuses, the user has to debug why and teach Claude PR flow every time
- **Rule without protection** → Claude follows the rule until it doesn't; first missed step ships a bad commit to main

Together: the rule sets up the commit correctly, protection ensures it goes through PR regardless.

## Branch protection — applied to 12 production repos

GitHub branch-protection rules are applied to every public production repo in the `agentteamland/` org:

**Protected** (PR required, direct push refused):

- Code repos: `core`, `cli`, `docs`, `brainstorm`, `rule`, `team-manager`, `registry`, `create-project`, `workspace`, `.github`
- Team repos: `software-project-team`, `design-system-team`, `starter-extended`

**Unprotected** (goreleaser auto-pushes; protection would break the release pipeline):

- Release-pipeline repos: `homebrew-tap`, `scoop-bucket`, `winget-pkgs`

### Settings

| Setting | Value | Why |
|---|---|---|
| `required_approving_review_count` | `0` | Solo maintainer — PR is required as ceremony + audit trail, not as external gate. (Bump to `1` if a second maintainer joins.) |
| `enforce_admins` | `false` | Admin (the maintainer) can bypass in genuine emergencies (release-pipeline break, urgent revert). |
| `allow_force_pushes` | `false` | No history rewrites. |
| `allow_deletions` | `false` | No branch deletion. |

Applied via the GitHub branch-protection API. Once applied, every direct push to `main` is refused at the server level — not at Claude's discretion.

## The rule — five fixed steps

The [`team-repo-maintenance`](/authoring/team-repo-maintenance) rule mandates five steps when Claude (or anyone) edits a cached shared repo:

0. **Run `/save-learnings` before opening the PR** — capture wisdom from the work that's about to ship; let it ride along in the same PR.
1. **Bump the version** (`team.json` for teams; `internal/config.Version` for CLI) following strict SemVer.
2. **Conventional commit** format: `type(scope): summary` with a body explaining *why*.
3. **"Discovered via" context** in the body — which project/session revealed the need, so the team-repo log is self-documenting.
4. **PR flow** (default, enforced by branch protection): feature branch → push → `gh pr create` → user reviews + clicks merge.

See the [team-repo-maintenance page](/authoring/team-repo-maintenance) for each step in detail, including the `team.json` validation contract and the absolute NEVER-MERGE discipline.

## Two absolute constraints

Beyond the five fixed steps, two constraints are non-negotiable:

### 1. Claude never merges PRs

No exceptions. Not for trivial changes, not for "0 required approvals" ceremony, not for self-authored PRs, not for hotfixes. **Merging belongs to the human reviewer.**

The only narrow exception: when the user explicitly authorizes auto-merge through tooling — most commonly [`/create-pr --auto-merge`](/skills/create-pr) — Claude may run `gh pr merge --auto --merge` to enable GitHub's native auto-merge. Branch protection's check gate is preserved (GitHub waits for required checks before merging). See the [team-repo-maintenance "PR merge discipline" section](/authoring/team-repo-maintenance#pr-merge-discipline-absolute-no-exceptions) for the full prohibition + exception spec.

Direct-push bypass for genuine emergencies remains (admin can commit directly), because that's a different operation — but Claude does not initiate it; the user does.

### 2. No `--assignee` or `--reviewer` on Claude's own PRs

In the current solo-maintainer setup, Claude pushes under the maintainer's own GitHub account, which makes the maintainer the PR author automatically:

- Author field already surfaces the PR in the maintainer's "Created by me" / "Involves me" dashboards
- Explicit `--assignee @me` is redundant (author == assignee) and pollutes the "Assigned to me" queue
- GitHub blocks requesting review from the PR author, so `--add-reviewer mkurak` silently fails on own PRs

(When a separate bot account is provisioned for Claude's pushes — so author ≠ maintainer — `--reviewer mkurak` becomes possible and appropriate. That is future work, not current policy.)

## Solo-maintainer considerations

**Why `required_approving_review_count = 0`?** Because with one active maintainer, requiring an external approver makes every PR a 24-hour block (wait for another account to approve). Zero approvals = PR exists as documentation, but the maintainer can self-merge. If a second maintainer joins, bump to 1.

**Why allow admin bypass?** Release-pipeline breakage, urgent revert, or a situation where waiting for the PR ceremony costs real user-facing time. When used, the commit still follows conventional format and should be paired with a retrospective.

**Solo vs team tradeoff:** Some governance purists would require approvals even from a second machine/identity. For this ecosystem at this stage, that's over-engineering. The branch protection + rule combination catches the 95% case (commit hygiene, version bumps, audit trail) and the 5% (self-review quality) is something a solo maintainer has to own through discipline anyway.

## What this governance does NOT cover

- **Private project repos** — your own project's git workflow is up to you. This governance is specifically for `agentteamland/` public repos.
- **Release-pipeline repos** (`homebrew-tap`, `scoop-bucket`, `winget-pkgs`) — goreleaser auto-pushes; branch protection is intentionally not applied.
- **Tag-based releases** — when tagging `cli v0.2.1`, the tag push triggers goreleaser. No PR needed for tag creation itself (it points at an already-merged commit on main).

## Rollout

Applied 2026-04-24:

1. **GitHub API branch protection** to 13 repos (12 production + workspace) — immediate effect
2. **`core@1.3.1` PR**: [feat(core): team-repo-maintenance rule](https://github.com/agentteamland/core/pull/1) — first change submitted through the new flow
3. **`design-system-team@0.4.2` PR**: [fix(dst-new-ds): Q3 single-select](https://github.com/agentteamland/design-system-team/pull/1) — concurrent first-customer of the new flow
4. **Workspace `CLAUDE.md` state snapshot** + the [settled-decision doc](https://github.com/agentteamland/workspace/blob/main/.atl/docs/branch-protection-and-team-repo-governance.md) — updated to reflect the new normal

## Related

- [Team-repo maintenance](/authoring/team-repo-maintenance) — the procedural detail (how to actually open the PR, what the commit message looks like)
- [`/create-pr`](/skills/create-pr) — the skill that automates this discipline
- [Karpathy guidelines](/guide/karpathy-guidelines) — the coding-guideline counterpart of this governance
- Settled-decision doc: [branch-protection-and-team-repo-governance.md](https://github.com/agentteamland/workspace/blob/main/.atl/docs/branch-protection-and-team-repo-governance.md)
- Branch-protection API reference: [docs.github.com/en/rest/branches/branch-protection](https://docs.github.com/en/rest/branches/branch-protection)
