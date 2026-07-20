# Workspace — the maintainer hub

The [`agentteamland/workspace`](https://github.com/agentteamland/workspace) repo is the **maintainer hub** for the AgentTeamLand ecosystem. It's a meta-repo: cloning it and running one script gives you every peer repo (atl, docs, .github, etc.) checked out under a single tree at `./repos/`. Every moving part of the platform is one `cd repos/<name>` away.

Use the workspace when you're doing maintenance work that spans multiple repos: cross-repo refactors, multi-PR rollouts, governance audits, or just `git status` across the whole org without many separate `cd` commands.

If you only want to USE atl (install teams in your own projects), you don't need the workspace — the [install script](../guide/install) is enough. The workspace is for ecosystem-side work.

## Bootstrap

```bash
git clone https://github.com/agentteamland/workspace.git
cd workspace
./scripts/sync.sh
```

`sync.sh` clones every peer repo under `agentteamland/` into `./repos/<name>/`. It's idempotent — re-running fast-forward-pulls existing clones and clones any repos in its canonical list that aren't checked out yet. The repo list is hand-maintained inside `sync.sh`; a repo newly added to the org must be added there before sync will pick it up.

After sync, `./repos/` contains the v2 active repos plus the archived v1 repos (kept read-only for history):

```
repos/
├── atl/                       # v2 monorepo — cli + core + teams + docs
├── atl-e2e-team/              # real-GitHub fixture team for the e2e publish blueprints
└── .github/                   # organization profile

# Archived v1 repos (read-only, kept for history):
├── cli/                       # 🗄 ARCHIVED 2026-06-21 — ported into atl monorepo
├── core/                      # 🗄 ARCHIVED 2026-06-21 — ported into atl monorepo
├── brainstorm/                # 🗄 ARCHIVED 2026-06-21 — ported into atl monorepo
├── rule/                      # 🗄 ARCHIVED 2026-06-21 — ported into atl monorepo
├── team-manager/              # 🗄 ARCHIVED 2026-06-21 — bootstrap wrapper; atl is the installer now
├── software-project-team/     # 🗄 ARCHIVED 2026-06-21 — ported into atl monorepo
├── design-system-team/        # 🗄 ARCHIVED 2026-06-21 — ported into atl monorepo
├── starter-extended/          # 🗄 ARCHIVED 2026-06-21 — inheritance dropped from v2
├── create-project/            # 🗄 ARCHIVED 2026-05-03 — v1 project scaffolder, superseded
├── registry/                  # 🗄 ARCHIVED 2026-06-21 — replaced by GitHub topic catalog
├── homebrew-tap/ scoop-bucket/ # 🗄 ARCHIVED 2026-06-21 — distribution via GitHub Releases only
└── docs/                      # 🗄 ARCHIVED 2026-06-22 — docs site ported into the atl monorepo (docs/site/)
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
# Make your changes, follow the repo's conventional-commit + branch-hygiene discipline
git checkout -b <type>/<short-description>
# ... edit files ...
git add <files> && git commit -m "<conventional message>"
git push -u origin <branch-name>
gh pr create
# Wait for review + merge by the maintainer
```

Each peer repo is its own git clone with its own remote. Branch protection on the public production repos enforces the PR flow.

## Using the workspace with Claude Code

Open Claude Code in the workspace root:

```bash
cd ~/projects/my/agentteamland/workspace
claude    # or however you invoke Claude Code
```

When Claude Code starts here, it automatically sees:

- **Every peer repo under `./repos/`** for direct editing — no separate `cd` needed
- **All active brainstorms** (auto-pinned into `CLAUDE.md` per the [brainstorm rule](https://github.com/agentteamland/atl/blob/main/core/rules/brainstorm.md))
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

For a more thorough end-of-session pass, [`/repo-cleanup`](https://github.com/agentteamland/workspace/blob/main/.claude/skills/repo-cleanup/SKILL.md) automates: a learning-capture drain → branch + commit + push + PR + auto-merge → tag + branch prune. Run it from inside Claude Code in the workspace.

## Related

- [Install the `atl` CLI](../guide/install) — if you only want to USE atl, skip the workspace
- [Knowledge system](../guide/knowledge-system) — the journal + wiki layers in the workspace's `.atl/` directory
