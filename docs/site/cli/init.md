# `atl init`

Scaffold a lean starter `CLAUDE.md` for the tier you choose — **only if one doesn't already exist**, so your own `CLAUDE.md` is never overwritten.

`CLAUDE.md` is the file Claude Code auto-loads as project (and global) instructions. ATL ships a starter shape for each tier so you don't begin from a blank file; you fill in the parts marked for you, and the `/brainstorm` + `/drain` skills maintain their own marker blocks inside the project file over time.

## Usage

```bash
atl init                 # a project-root CLAUDE.md (default)
atl init --project       # the same, explicit
atl init --global        # your personal ~/.claude/CLAUDE.md persona
atl init --monorepo      # a lean ~30-line orientation file
```

The three flags are mutually exclusive. `atl install` also drops the **project** starter automatically when a project has no `CLAUDE.md` (see [What it does](#what-it-does)), so you usually only run `atl init` by hand for the **global** persona or a **monorepo** orientation file.

## Tiers

| Flag | Target path | Shape |
|---|---|---|
| `--project` (default) | `<project>/CLAUDE.md` | Hybrid: ATL-managed marker blocks (active brainstorms, knowledge index — maintained by `/brainstorm` + `/drain`) plus user-owned, evidence-fillable facts (stack, commands, conventions) and an optional skill-routing table. Soft budget ≤ ~60 lines. |
| `--global` | `~/.claude/CLAUDE.md` | Pure user persona — how you want Claude to work everywhere. **ATL manages nothing here.** Soft budget ≤ ~80 lines. |
| `--monorepo` | `<repo>/CLAUDE.md` | The project shape specialized + lean: a layout table and conventions as **pointers**, not inlined content. Soft budget ~30 lines. |

The tiers, their budgets, and the managed-vs-owned ownership model are explained in full on the [Claude Code conventions](/guide/claude-code-conventions) page.

## What it does

`atl init`:

1. Resolves the target path for the chosen tier (global → `~/.claude/CLAUDE.md`; project / monorepo → the project root's `CLAUDE.md`).
2. **If a `CLAUDE.md` already exists there, it does nothing** — your file is user-owned and never overwritten.
3. Otherwise it writes the tier's starter skeleton (filling the project / repo name) and prints the path it created.

For the **project** and **monorepo** tiers, `atl init` also drops empty `.atl/backlog.md` + `.atl/tasks.md` skeletons alongside the `CLAUDE.md` — the two decision-state files the `/brainstorm` skill keeps current (see [Backlog & tasks](../guide/backlog-and-tasks.md)). Each is written **only if absent**, so your own `backlog.md` / `tasks.md` is never overwritten. The **global** tier is skipped — it has no project `.atl/`.

`atl install` runs the same project-tier scaffold as a best-effort step: when you install a team into a project that has no `CLAUDE.md`, ATL drops the project starter (and the `.atl/backlog.md` + `.atl/tasks.md` skeletons) so the `/brainstorm` and `/drain` blocks have a home. It is only-if-absent and never fails the install.

## Idempotency — safe to re-run

Re-running `atl init` (or `atl install`) when a `CLAUDE.md` is already present is a no-op — it reports that the file exists and leaves it untouched. There is no `--force`; replacing an existing `CLAUDE.md` is a deliberate manual act, not something a scaffold should do.

## Related

- [Claude Code conventions](/guide/claude-code-conventions) — the three tiers, token budgets, ownership model, and the marker blocks the project file carries
- [Backlog & tasks](../guide/backlog-and-tasks.md) — the `.atl/backlog.md` + `.atl/tasks.md` decision-state files this scaffold drops for the project / monorepo tiers
- [`atl install`](/cli/install) — installs a team and drops the project starter if absent
- [`/brainstorm`](/skills/brainstorm) — maintains the `<!-- brainstorm:active -->` block in the project `CLAUDE.md`
- [`/drain`](/skills/drain) — maintains the `<!-- wiki:index -->` knowledge map block
