# `/brainstorm`

Start and complete brainstorming sessions. Brainstorms are the canonical place to think through a non-trivial decision before writing code — the resulting file becomes the historical record + handoff for any later session that picks up the topic.

Two modes: `start` opens a new brainstorm; `done` completes the active brainstorm and propagates its decisions to the document chain.

Ships as a global skill in the [ATL monorepo](https://github.com/agentteamland/atl).

## Two scopes

A brainstorm lives at one of two levels — pick the scope that matches *who* will care about the decision:

| Flag | Target directory | When |
|---|---|---|
| *(none)* | `.atl/brain-storms/` | Project-specific topics (default) |
| `--global` | `~/.atl/brain-storms/` | Cross-project, personal topics |

## `start` mode

```
/brainstorm start [--global] <initial message describing the topic>
```

Flow:

1. **Infer the topic** from the user's message — derive a kebab-case filename. The user does NOT supply a filename.
2. **Determine scope** from the flag (or default to project).
3. **Create directory** if it doesn't exist (`{scope-base}/brain-storms/`).
4. **Create the brainstorm file** with frontmatter (`status: active`, `scope`, `date`, `participants`) + Context + Discussion + Open Items sections.
5. **Pin the brainstorm into the scope's `CLAUDE.md`** via a `<!-- brainstorm:active -->` marker block. This makes the active brainstorm impossible to miss in the next session — it auto-loads with project context.
6. **Confirm** to the user: filename, scope, pinned location, then dive into the topic.

### The active-brainstorm pin

Every active brainstorm pins itself into the scope's `CLAUDE.md` inside an `<!-- brainstorm:active:start --> ... <!-- brainstorm:active:end -->` block:

```markdown
<!-- brainstorm:active:start -->
## ⚠️ Active brainstorms

These topics have an in-progress brainstorm — read the file before making any decision on them.

- **[profile-team](.atl/brain-storms/profile-team.md)** (project, 2026-05-08) — schema, storage, privacy, and initial scope for the new profile-team package
<!-- brainstorm:active:end -->
```

Multiple active brainstorms coexist as bullets in the same block. Shipped in `brainstorm@1.1.0`.

### Keeping the brainstorm alive (every message cycle)

While a brainstorm is active, on every message:

- **Before responding** — read the brainstorm file (recall context)
- **After responding** — write new ideas, decisions, rejected alternatives + reasons, the user's exact statements at important points, open questions, chronological flow

The file must be **detailed enough** that a Claude reading it in a new context can continue as if it had been present in the original conversation.

## `done` mode

```
/brainstorm done
```

Flow:

1. **Find the active brainstorm.** Searches both scopes (`.atl/brain-storms/` and `~/.atl/brain-storms/`). If multiple, lists them with their scope and asks which to complete.
2. **Complete the brainstorm file.** `status: active` → `status: completed`. Append final notes. Update Open Items (unresolved ones remain). Add a Final Decisions section.
3. **Create or update the docs file.** Settled decisions go to:
   - **Project brainstorm** → `.atl/docs/`
   - **Global brainstorm** → `~/.atl/docs/`
4. **Update CLAUDE.md.** Up to three things happen:
   - Append the completed-brainstorm summary to the appropriate section
   - Remove this brainstorm's bullet from the `<!-- brainstorm:active -->` marker block. If the bullet list becomes empty, remove the entire block (no stale "Active brainstorms" heading lingers).
   - If the decision leaves unshipped implementation, add a bullet to the `<!-- pending-implementation -->` block so the next session sees the queued work (omitted for a pure-decision brainstorm; removed when the implementation ships).

## The document chain

Every discussion and decision flows through three layers:

```
brain-storms/ (process) → docs/ (outcome) → CLAUDE.md (summary)
                     \
                       backlog.md (deferred items)
```

- No decision is made without a brainstorm.
- Brainstorm files are **never deleted** — they're the historical record.
- If decisions change, a NEW brainstorm is opened and a `superseded by X` note is added to the old one.

## Backlog discipline

Every item marked "not doing now, later" during a brainstorm is reflected in `.atl/backlog.md`:

- **Prepend** (newest on top)
- For each item: date + category heading + context link + detailed topic description + "when does this come up" note + related resources
- During `/brainstorm done`, every "deferred" note in the brainstorm is checked against the backlog — missing ones are added before the brainstorm is closed
- When a deferred item is later implemented, it's **deleted** from the backlog (not marked done)

## Important rules

1. **Multiple active brainstorms can exist.** Each lives in its own file. They can be active simultaneously across scopes.
2. **Resilience to context breaks.** The brainstorm file is persistent state. A new session detects active brainstorms via the marker block + directory scan and continues by reading the file.
3. **Filename is not requested from the user.** It's inferred from the message and assigned a kebab-case name.
4. **Brainstorm files are never deleted.** Historical record.
5. **Each brainstorm focuses on a single topic.** Different topics → different files.
6. **Active brainstorm search covers both scopes.** In `done` mode, project + global are scanned.
7. **Scope is in frontmatter.** `scope: project|global` — determines `done`-mode targets.

## Related

- [`/drain`](/skills/drain) — periodic learning capture (parallel to brainstorm; brainstorms are deliberate, learnings are spontaneous)
- [Concepts: Skill](/guide/concepts#skill) — where brainstorms fit in the knowledge model

## Source

- Spec: [core/skills/brainstorm/SKILL.md](https://github.com/agentteamland/atl/blob/main/core/skills/brainstorm/SKILL.md)
- Rule: [core/rules/brainstorm.md](https://github.com/agentteamland/atl/blob/main/core/rules/brainstorm.md)
