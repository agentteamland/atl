# Backlog & tasks

Two small files under `.atl/` hold a project's deferred decisions and its near-term intentions. They are decision *state* — a sibling to the [knowledge system](./knowledge-system.md)'s journal and wiki, not a third knowledge layer — and they are written and kept current by the [`/brainstorm`](../skills/brainstorm.md) skill.

## The two tiers

| File | What it is | How it's kept |
|---|---|---|
| **`.atl/backlog.md`** | The passive, **trigger-gated superset** of everything deferred, punted, or left uncertain. A scannable index, not a to-do list. | Grouped by area; an item is deleted when it ships or is promoted |
| **`.atl/tasks.md`** | The **active-intent subset** — the short, prioritized list of what you actually mean to do next. | `Now` / `Next`; an item is deleted when it ships |

An item moves **backlog → tasks** when you decide to pull it forward — a trigger fired, or you simply chose to prioritize it. That is the whole relationship: the backlog is everything you've consciously chosen *not* to do now; tasks is the slice you've chosen *to* do.

## backlog.md

The backlog exists so nothing is lost when scope moves on. Every time a brainstorm defers a sub-topic, marks something "not in this step", or leaves a question open, it lands here immediately — so a decision made months ago is still discoverable when its moment comes.

- **Grouped by area, not by date.** Headings are themes (`## Learning loop`, `## Distribution`, …); the file is an index you scan, not a chronology.
- **One line per item:** `- **Title** — one sentence. _Trigger:_ when it resurfaces. ↳ [source](...)`.
- **Trigger-gated.** Most items carry a `_Trigger:_` — the condition under which they come back. The backlog is not a to-do list; it's the memory of what you deferred and why it would return.
- **The detail lives in the source.** The rich "why deferred / full context" stays in the linked brainstorm — the backlog is the index, the brainstorm is the record. Don't duplicate it.

## tasks.md

Tasks is the honest short list of active intent.

- **Format:** `- [ ] **Title** — one sentence. ↳ [source](...)`, grouped under `## Now` / `## Next`.
- **Short and honest.** If nothing is actively planned, `tasks.md` is nearly empty — that is the correct state, not a gap to fill. Don't manufacture tasks; unplanned deferred work belongs in `backlog.md`.
- **Deleted when shipped.** A finished task is removed (the `docs/` and CLAUDE.md become the source of truth), never left behind as a checked-off item.

## Lifecycle

```
brainstorm defers  →  backlog.md  →(pull forward)→  tasks.md  →(ship)→  deleted
                          └──────────────(ship directly)──────────────→  deleted
```

- **Into the backlog:** [`/brainstorm done`](../skills/brainstorm.md) scans the closed brainstorm and files every deferred or uncertain item under the right area group. Skipping this is how deferred scope silently disappears.
- **Backlog → tasks:** promote an item when you decide to act on it.
- **Out:** an item leaves either file the moment it ships — the docs become the truth, and nothing lingers as done.

## Scaffolding

[`atl init`](../cli/init.md) (and `atl install`) drop empty `backlog.md` and `tasks.md` skeletons under `.atl/`, only if they don't already exist — your own files are never overwritten. The global tier has no project `.atl/`, so it is skipped.
