# Workspace — the maintainer hub

The [`agentteamland/workspace`](https://github.com/agentteamland/workspace) repo is the **maintainer hub** for the AgentTeamLand ecosystem. It's a meta-repo: cloning it and running one script gives you every peer repo (cli, core, registry, software-project-team, etc.) checked out under a single tree at `./repos/`. Every moving part of the platform is one `cd repos/<name>` away.

Use the workspace when you're doing maintenance work that spans multiple repos: cross-repo refactors, multi-PR rollouts, governance audits, or just `git status` across the whole org without 14 separate `cd` commands.

If you only want to USE atl (install teams in your own projects), you don't need the workspace — `brew install agentteamland/tap/atl` is enough. The workspace is for ecosystem-side work.

## Bootstrap

```bash
git clone https://github.com/agentteamland/workspace.git
cd workspace
./scripts/sync.sh
```

`sync.sh` clones every peer repo under `agentteamland/` into `./repos/<name>/`. It's idempotent — re-running fast-forward-pulls existing clones and clones any new ones added to the org since last run.

After sync, `./repos/` contains the full org snapshot (16 repos as of 2026-05-03):

```
repos/
├── cli/                       # atl binary (Go) — the CLI users install
├── core/                      # global skills + rules + JSON schemas
├── brainstorm/                # /brainstorm skill + its rule
├── rule/                      # /rule + /rule-wizard skills
├── team-manager/              # bootstrap install.sh (delegates to atl post-v1.0.0)
├── software-project-team/     # 13 agents + 3 skills (.NET + Flutter + React stack)
├── design-system-team/        # 2 agents + 10 /dst-* skills (native design + prototype)
├── starter-extended/          # inheritance example team
├── create-project/            # 🗄 ARCHIVED 2026-05-04 — scaffolder moved into teams; kept for history
├── registry/                  # teams.json — canonical team catalog
├── docs/                      # this docs site (VitePress, EN + TR)
├── homebrew-tap/              # auto-managed by goreleaser
├── scoop-bucket/              # auto-managed by goreleaser
├── winget-pkgs/               # fork of microsoft/winget-pkgs
└── .github/                   # organization profile
```

## Daily commands

The workspace ships three scripts under `./scripts/`:

```bash
./scripts/sync.sh         # clone missing repos; fast-forward pull existing ones
./scripts/status.sh       # tabular overview — who's dirty, ahead, behind
./scripts/push-all.sh     # dry-run list of unpushed commits (use --force to push)
```

`status.sh` prints a one-line-per-repo table — branch, ahead/behind counts, dirty marker. Run it at the start of any session to see the org's current state at a glance.

`push-all.sh` is dry-run by default — it shows what WOULD push but doesn't actually push. Pass `--force` to actually push. (The "force" name refers to overriding the dry-run, not `git push --force` — the actual push uses normal git semantics.)

## Working in a peer repo

```bash
cd repos/<repo-name>
# Make your changes, follow the team-repo-maintenance discipline
git checkout -b <type>/<short-description>
# ... edit files ...
git add <files> && git commit -m "<conventional message>"
git push -u origin <branch-name>
gh pr create
# Wait for review + merge by the maintainer
```

Each peer repo is its own git clone with its own remote. Branch protection on every public production repo (12 of the 16 — release-pipeline + .github excluded) enforces the PR flow. See [Team-repo maintenance](../authoring/team-repo-maintenance) for the full discipline.

## Using the workspace with Claude Code

Open Claude Code in the workspace root:

```bash
cd ~/projects/my/agentteamland/workspace
claude    # or however you invoke Claude Code
```

When Claude Code starts here, it automatically sees:

- **Every peer repo under `./repos/`** for direct editing — no separate `cd` needed
- **All active brainstorms** (auto-pinned into `CLAUDE.md` per the [brainstorm rule](https://github.com/agentteamland/brainstorm/blob/main/rules/brainstorm.md))
- **Workspace `CLAUDE.md`** — the platform-level orientation document
- **Final decisions** in `.atl/docs/` (settled architecture decisions derived from completed brainstorms)
- **Wiki + journal** in `.atl/wiki/` and `.atl/journal/` (per the [knowledge system](../guide/knowledge-system))

This is the natural setup for cross-repo work: Claude has the full org as its working set.

## Knowledge map

The workspace's `CLAUDE.md` carries a `<!-- wiki:index -->` marker block that auto-loads every wiki page's title + summary into Claude's context. See [Claude Code conventions](../guide/claude-code-conventions) for how the marker block works and why it exists.

The wiki itself (`.atl/wiki/*.md`) is the canonical record of platform-wide patterns, conventions, discoveries, and anti-patterns the maintainer needs handy when working on cross-repo concerns. Pages are kept current — the [knowledge system](../guide/knowledge-system) is replace-style for current truth, append-only journal for history.

## End of session

When wrapping up:

```bash
./scripts/status.sh        # confirm everything is on main + clean
./scripts/push-all.sh      # see what's unpushed
```

For a more thorough end-of-session pass, [`/repo-cleanup`](https://github.com/agentteamland/workspace/blob/main/.claude/skills/repo-cleanup/skill.md) automates: save-learnings → branch + commit + push + PR + auto-merge → tag + registry + branch prune. Run it from inside Claude Code in the workspace.

## Related

- [Install the `atl` CLI](../guide/install) — if you only want to USE atl, skip the workspace
- [Team-repo maintenance](../authoring/team-repo-maintenance) — the discipline every peer-repo PR follows
- [Governance](../guide/governance) — branch protection + the team-repo-maintenance rule pair
- [Knowledge system](../guide/knowledge-system) — the journal + wiki layers the workspace's `.claude/` directory uses
