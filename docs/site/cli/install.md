# `atl install`

Install a team into the current project.

## Usage

```bash
atl install <team>                # idempotent first-time-only no-op (atl ≥ 1.0.0)
atl install <team> --refresh      # force overwrite (discards local modifications with warning)
```

`<team>` can be:

- A **registry short name** — `software-project-team`
- A **registry name with version** — `software-project-team@^1.2.0` (caret, tilde, or exact pin)
- A **GitHub `owner/repo` shorthand** — `agentteamland/starter-extended`
- A **Git URL** — `https://github.com/you/your-team.git`, `git@github.com:you/team.git`, `ssh://...`, `file:///abs/path.git`
- A **local filesystem path** — `./my-team`, `~/projects/my-team`, `/abs/path/to/team` (atl ≥ 0.1.4; the path must be a directory with `team.json` at its root AND a `.git/`)

## Examples

### Registry (public, verified)

```bash
atl install software-project-team
atl install software-project-team@^1.2.0
```

### GitHub shorthand

```bash
atl install agentteamland/starter-extended
```

### Full Git URL (public or private)

```bash
atl install https://github.com/acme/acme-starter.git
atl install git@github.com:you/private-team.git          # SSH, uses your git credentials
atl install https://gitea.example.com/you/team.git       # self-hosted
```

### Local filesystem — no remote required (atl ≥ 0.1.4)

The private-local workflow: build a team on your laptop, install it into your own project without pushing to any git server.

```bash
# One-time: set up the team as a git repo
cd ~/projects/my-team
git init -b main && git add . && git commit -m "init"

# Install into any project:
cd ~/projects/some-app
atl install ~/projects/my-team                   # absolute path
atl install ./my-team                            # relative path
atl install file:///Users/you/projects/my-team   # explicit file:// URL
```

All three forms work identically. The source must be a directory containing `team.json` and must be a git repo (at least one commit). See [Creating a team](/authoring/creating-a-team) for the full walk-through.

## Multi-team installation

Multiple teams can coexist in the same project — `atl` v0.1.2+ supports this natively. Every resource (agents, rules, skills) is **copied** into the project's `.claude/` directory; the global cache at `~/.claude/repos/agentteamland/{team}/` is the source-of-truth, and `atl update` keeps unmodified copies in sync.

```bash
atl install software-project-team
atl install design-system-team

atl list
# ✓ software-project-team@1.2.1
# ✓ design-system-team@0.8.1
```

When two teams declare an item with the same name (e.g., both have a `code-reviewer` agent), the most recently installed one wins. atl prints a one-line warning:

```
⚠ overriding agent "code-reviewer" (was from team-a, now from team-b)
```

This mirrors npm / pip / GNU Stow conventions. Removing a team is safe — `atl remove` deletes its copied resources and replays the remaining teams' copies. Any item the removed team was "winning" by collision falls back to its original owner correctly.

## Project-local copy install (atl v1.0.0+)

Every team resource — agents, rules, skills — installs as a **project-local copy** under `<project>/.claude/`. The global cache at `~/.claude/repos/agentteamland/{team}/` is the source-of-truth shared across all projects on the machine; each project keeps its own self-contained copy.

This is the topology that the [install-mechanism-redesign decision](https://github.com/agentteamland/workspace/blob/main/.atl/docs/install-mechanism-redesign.md) settled on, replacing the pre-v1.0.0 mix of symlinks (for agents + rules) and copies (for skills only). Two reasons drove the change:

1. **Mutations stay local.** `/save-learnings` and the auto-grown `children/` and `learnings/` directories from the [self-updating learning loop](https://github.com/agentteamland/workspace/blob/main/.atl/docs/self-updating-learning-loop.md) write into the project's `.claude/`. With symlinks, those writes would have polluted the global cache and collided on the next `atl update` pull. With copies, mutations stay isolated to the project that produced them.
2. **Claude Code's skill loader.** Independent of the topology decision, Claude Code's project-level skill loader does NOT follow symlinks under `.claude/skills/` (upstream issues [#14836](https://github.com/anthropics/claude-code/issues/14836), [#25367](https://github.com/anthropics/claude-code/issues/25367), [#37590](https://github.com/anthropics/claude-code/issues/37590)). Skills had to be copied anyway; the v1.0.0 redesign generalized that to all resources.

`atl update` is the synchronization mechanism — see below.

## Idempotency (atl v1.0.0+)

Running `atl install <team>` a second time on an already-installed team is a **no-op** with a one-line info message — not the silent reinstall it used to be. Pre-v1.0.0 every install silently overwrote local edits; v1.0.0+ requires `--refresh` to do that.

```bash
atl install software-project-team           # first time → installs
atl install software-project-team           # again → no-op + info line
atl install software-project-team --refresh # force-reinstall (warns if local edits exist)
```

This means re-running `atl install` is safe even in scripts. To pick up upstream changes for a team you have installed, use `atl update` (it auto-refreshes unmodified copies).

## What happens

1. **Resolve.** Registry names are looked up in [`teams.json`](https://github.com/agentteamland/registry/blob/main/teams.json); URLs are used directly.
2. **Clone or pull.** If the team isn't in the shared cache, it's cloned. If it is, the cache is fast-forwarded.
3. **Resolve inheritance.** If `team.json` has an `extends` field, the parent is installed (recursively) before the child.
4. **Validate.** `team.json` is checked against the [schema](/reference/schema). Invalid teams fail here.
5. **Materialize.** Agents, rules, and skills are **copied** into `.claude/agents/`, `.claude/rules/`, `.claude/skills/`. Precedence applies (child wins, excludes drop). Existing project copies are kept (idempotent default); use `--refresh` to overwrite.
6. **Record.** `.claude/.team-installs.json` is updated atomically (tmp + rename) with the installed team, version, chain, and per-resource SHA-256 baselines used by `atl update`'s auto-refresh.

## Offline behavior

If the network is unreachable, `atl install` falls back to the shared cache. You'll get whatever version was last pulled. The CLI logs this explicitly — no silent staleness.

## Troubleshooting

- **"team not found"** — the name isn't in the registry. Try `atl search`.
- **"circular extends chain"** — a team in the chain extends an ancestor. The error prints the full chain.
- **"schema validation failed"** — the team's `team.json` is malformed. Ask the author to fix it, or pin to an earlier version.
- **"invalid team slug"** — the name resolved to a slug that fails the safety regex (e.g., starts with `-`). Pass the team in `owner/repo` form or use a Git URL.

## Related

- [`atl search`](/cli/search) — find a team's name.
- [`atl list`](/cli/list) — see what's installed.
- [`atl update`](/cli/update) — refresh unmodified copies after the cache pulls.
- [`atl remove`](/cli/remove) — uninstall (with `--force` for non-interactive).
