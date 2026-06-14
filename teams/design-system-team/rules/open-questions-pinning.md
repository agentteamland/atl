# Open-questions pinning (convention auto-loaded by design-system-team)

Every project that uses `design-system-team` for screens / prototypes also accumulates **open product / design questions** that block screen work — revenue model, MVP slicing, persona, vendor choices, legal constraints, etc. These aren't bugs to file; they're **decisions blocking screen design until answered**, and they pile up over a project's lifetime.

This rule defines the **canonical home** for those questions and the **pinning mechanism** that keeps them visible to every Claude session that opens the project.

## The two artifacts

### 1. `.atl/wiki/open-questions.md` — source of truth

A single file under the project's wiki directory. One bullet per open question. Each bullet names the question + what area it blocks. Format:

```markdown
# Open Questions

> Questions that block screen / prototype work until answered. Each entry names the question + what it blocks. Resolved questions are removed (the answered fact lands in the relevant wiki page or `docs/` instead).

## Active

- **<question / topic>** — blocks <area>
- **<question / topic>** — blocks <area>
```

The file may contain other sections (history, conventions, related), but the `## Active` H2 section is what the pin reads.

### 2. `<!-- open-questions:active -->` marker block in `CLAUDE.md` — the pin

Mirrors the active-brainstorm pinning pattern from `brainstorm@1.1.0`:

```markdown
<!-- open-questions:active:start -->
## ❓ Open product questions

These questions block screen / prototype work — address them when their topic comes up. Full list: [.atl/wiki/open-questions.md](.atl/wiki/open-questions.md)

- **<question / topic>** — blocks <area>
- **<question / topic>** — blocks <area>
<!-- open-questions:active:end -->
```

Because `CLAUDE.md` is auto-loaded into every Claude session as project instructions, the open-questions list becomes part of context unconditionally — no scanning, no remembering.

## Insertion order with brainstorm marker

If both `<!-- brainstorm:active:start -->` and `<!-- open-questions:active:start -->` blocks are present, **brainstorm comes first** (active brainstorm is a stronger interrupt — it means a decision is in motion right now). Open questions are static-state-of-things; they sit just below.

## Lifecycle — who maintains the pin

The pin must be **in sync** with `## Active` in `open-questions.md` at all times. Three update paths, in order of preference:

1. **`/dst-questions` skill** (if installed via design-system-team v0.7.0+):
   - `add` / `resolve` — modify the file + auto-sync the pin
   - `sync` — re-render pin from current file (use after manual edits)
   - `init` — bootstrap empty file + add pin
2. **Agents** (`ds-architect-agent`, `prototype-agent`): when editing `open-questions.md` directly, update the marker block in the same response. Do not let the file and the pin drift.
3. **Manual edits by the user**: after a manual edit, run `/dst-questions sync`.

If a session opens with an `open-questions.md` that has bullets but no marker block in `CLAUDE.md` (e.g., the file predates this convention), the agent **restores the marker** as the first action and informs the user.

## Bullet-count cap

If the active list grows beyond ~8 bullets, the marker block lists the first 6 inline and adds a `- … + N more — see open-questions.md` line. Keeps `CLAUDE.md` lean while preserving the visible-at-the-top signal.

## When NOT to use this convention

- Projects without `design-system-team` installed don't need it (the `/dst-questions` skill won't be available anyway).
- Bug tracking, sprint tasks, feature requests — those have other tools (issue tracker, project board). This file is specifically for **decisions blocking screen design**, not work items.
- Personal TODOs — out of scope; use a personal scratch file outside `.claude/`.

## Why this convention lives in design-system-team (not core)

The "open product questions blocking screens" concept is specific to projects building **user-facing UI** with help from `ds-architect-agent` / `prototype-agent`. Backend-only or infra-only projects don't have screen-blocking decisions in the same way. Putting it under design-system-team scope means:

- It activates only when a project actually has design-system / prototype needs
- The two agents that care about screens are aware of it
- Other teams (software-project-team, etc.) don't carry the convention they don't need

## See also

- `skills/dst-questions/skill.md` — the implementation of this convention
- `agentteamland/brainstorm@1.1.0` — the parallel pin mechanism for active brainstorms (same marker style, same insertion logic)
