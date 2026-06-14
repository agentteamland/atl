# design-system-team

> Design systems and UI prototypes inside any project — local, file-based, browser-viewable.

**Latest version:** `0.8.1`
**Status:** Verified
**Repository:** [github.com/agentteamland/design-system-team](https://github.com/agentteamland/design-system-team)

## What it does

Brings Claude Design–style design-system + screen-prototype work natively into any project — without requiring an external app, API key, or daemon. All LLM work happens inside your own Claude Code session (using your existing Pro/Max entitlement); outputs land as JSON state + Tailwind-rendered HTML pages in the project's `.dst/` directory, viewable in any browser.

## Install

```bash
cd your-project
atl install design-system-team
```

Requires [`atl`](/cli/overview) ≥ 0.1.2 (per `team.json`'s `requires.atl`). Practical baseline for current behavior is `atl ≥ 1.0.0` — earlier versions predate the project-local copy install model.

## Quickstart

```bash
# 1. Bootstrap .dst/ in current project
/dst-init

# 2. Create a design system (interactive Q&A)
/dst-new-ds primary

# 3. Create a prototype against that design system
/dst-new-prototype --ds primary login-screen

# 4. View everything in your browser
/dst-open
```

## Layout

After `/dst-init`, your project gets:

```
.dst/
  index.html               ← studio landing (open in browser)
  styles.css               ← shared Tailwind helpers
  state.json               ← lightweight manifest

  design-systems/
    primary/
      ds.json              ← palette, typography, spacing, components, brand, voice
      detail.html          ← rich visual page of this DS
      assets/              ← logos, icons

  prototypes/
    login-screen/
      prototype.json       ← screen state (linked to a DS)
      preview.html         ← multi-state visual (idle/loading/error/success)
      handoff.zip          ← optional export bundle
```

`.dst/` is git-friendly — commit it, get free design history.

## Agents (2)

- **ds-architect-agent** — Designs comprehensive design systems (palette theory, typography ramps, spacing, components, brand identity, voice). 7 children files cover schema, palette theory, typography, spacing, components, brand, project-context reading, and template rendering.
- **prototype-agent** — Designs screen prototypes that strictly honor a chosen DS (token fidelity, state coverage, accessibility — non-negotiable). 7 children files.

## Skills (10)

| Skill | Purpose |
|-------|---------|
| `/dst-init` | Bootstrap or refresh `.dst/` in current project (self-healing) |
| `/dst-new-ds <name>` | Create new design system via interactive Q&A |
| `/dst-edit-ds <name> "<change>"` | Apply textual change; cascades to linked prototypes if structural |
| `/dst-delete-ds <name>` | Remove a DS (refuses if prototypes depend on it; `--force` to orphan) |
| `/dst-new-prototype --ds <name> <prototype>` | Create screen prototype linked to a DS |
| `/dst-edit-prototype <name> "<change>"` | Apply textual change, preserving token fidelity |
| `/dst-delete-prototype <name>` | Remove a prototype |
| `/dst-open` | Open `.dst/index.html` in default browser |
| `/dst-handoff <prototype> [--target …]` | Bundle prototype + DS + assets, brief flutter-agent / react-agent to integrate into source code |
| `/dst-questions [init\|sync\|add\|resolve\|list]` | Maintain `.atl/wiki/open-questions.md` (questions blocking screen work); auto-pin active list into `CLAUDE.md` |

## Why it exists

Three forces converged in April 2026:

1. Claude Design's rate limits bite even on Pro/Max plans.
2. Anthropic blocked third-party apps from using Pro/Max OAuth ([April 17 enforcement](https://venturebeat.com/technology/anthropic-cuts-off-the-ability-to-use-claude-subscriptions-with-openclaw-and-third-party-ai-agents)).
3. MCP sampling (the official path for servers to delegate LLM calls) is not yet implemented in Claude Code.

The pivot: **don't build a separate app**. Build a team. Skills do all LLM work in-session. Outputs are static files. App-shaped value via files + browser, not via a separate process.

## Pairs well with

- [`software-project-team`](/teams/software-project-team) — design with `/dst-*` skills, implement with flutter-agent / react-agent / api-agent.

## What shipped

- **v0.4.0** — `dst-init` becomes self-healing (safe upgrades re-render HTML/CSS from preserved JSON state); `dst-handoff` skill (one-command design→code via flutter-agent / react-agent); prototype `target` field
- **v0.5.0** — dark-mode token coverage + first-prototype pipeline upgrades
- **v0.6.0** — partial-based template architecture (one component = one file, no more 600-line monolith); catalog grows to **57 components across 7 groups** (Charts added as 7th group + 12 Tier 1 primitives)
- **v0.7.0** — `/dst-questions` skill + `open-questions-pinning` rule: standardized place for product questions blocking screen work, with auto-pin into `CLAUDE.md` (mirrors `brainstorm@1.1.0` pin pattern)

## Next up (speculative)

- live-reload (file-watcher → SSE for auto-refresh in browser)
- `/dst-export` / `/dst-import` for cross-project DS sharing
