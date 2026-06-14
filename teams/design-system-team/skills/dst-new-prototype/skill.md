---
name: dst-new-prototype
description: "/dst-new-prototype --ds <ds-name> <prototype-name> — create a new screen prototype within a chosen design system. Asks an interactive Q&A (archetype, states, copy, breakpoints) and invokes prototype-agent to produce prototype.json + multi-state preview.html. Updates .dst/state.json and re-renders .dst/index.html."
argument-hint: "--ds <ds-name> <prototype-name>"
---

# /dst-new-prototype Skill

## Purpose

Create a new screen prototype linked to an existing design system. Prototype-agent produces both the canonical `prototype.json` and a Tailwind-rendered `preview.html` that shows every state framed and labeled.

## Preconditions

- `.dst/` exists in current project (run `/dst-init` first if not).
- The target DS exists at `.dst/design-systems/<ds-name>/`.
- `<prototype-name>` argument is provided (kebab-case).

## Flow

### Phase 1 — Validate

1. `.dst/state.json` exists? If not → "Run `/dst-init` first."
2. Parse args:
   - `--ds <ds-name>` — required
   - `<prototype-name>` positional — required
3. Validate kebab-case for `<prototype-name>`.
4. `.dst/design-systems/<ds-name>/ds.json` exists? If not → "DS '<ds-name>' not found. Available: …"
5. `.dst/prototypes/<prototype-name>/` doesn't already exist? If exists → "Prototype already exists. Use `/dst-edit-prototype` or pick a different name."

### Phase 2 — Read linked DS context

Load `.dst/design-systems/<ds-name>/ds.json`. The agent will use this for token resolution and constraint checking.

### Phase 3 — Interactive Q&A

Use `AskUserQuestion`:

**Q0 — Target platform**
"Which platform is this screen for?"
- flutter — mobile/tablet Flutter app
- react-admin — back-office web admin (React + TypeScript)
- react-public — marketing / public site (Next.js)
- auto-detect — scan project structure (flutter/ → flutter; web/ + react in package.json → react-admin; otherwise ask)

The chosen value is stored at `prototype.json.target`. Later, `/dst-handoff` defaults to this target without asking again.

If user picks "auto-detect":
1. Check for `flutter/pubspec.yaml` or `./pubspec.yaml` → default `flutter`.
2. Else check for `package.json` with `react` in deps and an `admin/` or `web/` folder → default `react-admin`.
3. Else check for `package.json` with `next` in deps → default `react-public`.
4. If none match → ask the user to pick one (can't auto-detect).

**Q1 — Archetype**
"What kind of screen is this?"
- form (login, signup, settings, contact)
- list / grid (data list with optional filters)
- detail (single entity view)
- dashboard (multi-block overview)
- modal / dialog (inline overlay)
- empty (welcome, no-data state)
- settings (toggles + form sections)
- static (about, terms — no async)
- flow (multi-step wizard)

**Q2 — States** (multiSelect; pre-selected based on archetype defaults from `state-coverage.md`)
"Which states should be rendered?"
- idle ✓ (always)
- loading
- submitting
- error
- empty
- success
- disabled

**Q3 — Actions** (free text, comma-separated)
"What user actions are on this screen?"
e.g., "submit, forgot-password, register"

**Q4 — Primary copy**
- "Primary CTA label?" (e.g., "Sign in")
- "Page title?" (e.g., "Welcome")
- "Empty state message?" (only for list/empty archetypes)
- "Error tone?" (formal / friendly — defaults to DS voice)

**Q5 — Breakpoints**
"Which viewports?"
- mobile (default for mobile apps)
- desktop (default for admin / web)
- mobile + desktop (responsive)
- tablet too (rare)

**Q6 — Description (1-sentence)**
"One-line description of what this screen is/does:"

### Phase 4 — Invoke prototype-agent

Call the agent with a self-contained brief:

```
target prototype_name: {prototype_name}
linked DS: {ds_name}
linked DS path: .dst/design-systems/{ds_name}/ds.json
team repo: ~/.claude/repos/agentteamland/design-system-team/
team templates: templates/prototype-detail.html.tmpl
output directory: .dst/prototypes/{prototype_name}/
user answers: {q0-q6}
target platform: {q0 — flutter | react-admin | react-public, resolved if auto-detect}
```

The agent:
1. Reads the linked DS for token context
2. Synthesizes `prototype.json` per `children/prototype-schema.md` — token-referenced (not hardcoded), state-covered per `children/state-coverage.md`, accessibility-passing per `children/accessibility-coverage.md`
3. Renders `preview.html` from `templates/prototype-detail.html.tmpl` per `children/preview-rendering.md` — every state as a labeled frame
4. Generates assets/ if needed (icons, illustrations — placeholders if user didn't provide)
5. Writes both to `.dst/prototypes/{prototype_name}/`

### Phase 5 — Update state and re-render landing

1. Update `.dst/state.json`:
   - Append `{ name: prototype_name, displayName, linkedDs: ds_name, archetype, version: "1.0.0", createdAt: now, lastModified: now }` to `prototypes`
   - Increment the linked DS's `prototypesCount` in the `designSystems` array
   - Bump `lastUpdated`

2. Re-render `.dst/index.html` with new prototype card.

### Phase 6 — Print summary

```
✓ Created prototype "{prototype_name}" linked to DS "{ds_name}"
  - .dst/prototypes/{prototype_name}/prototype.json
  - .dst/prototypes/{prototype_name}/preview.html ({state_count} states)
  - .dst/state.json updated
  - .dst/index.html re-rendered

Open in browser:
  /dst-open                                                (full studio)
  open .dst/prototypes/{prototype_name}/preview.html       (this prototype only)
```

## Important Rules

1. **Prototype must link to a DS.** No standalone prototypes.
2. **prototype-agent is mandatory** — encodes all design knowledge (state coverage, a11y, token fidelity).
3. **Never hardcode visual values.** prototype-agent enforces this; surface any token-fidelity issues to the user.
4. **State coverage matters.** Don't accept a prototype with only `idle`. Push back during Q&A if user picks too few states.

## Edge Cases

- **User cancels mid-Q&A** — no files written, no state changes.
- **Linked DS missing tokens for screen** — agent flags in `prototype.json.notes`; skill prints warning to user with suggestion to `/dst-edit-ds <ds-name>` to add the missing piece.
- **Prototype name reserved** ("template", "shared", "default") — warn and ask user to confirm.

## What this skill does NOT do

- Doesn't open the browser (use `/dst-open`)
- Doesn't generate handoff.zip by default (separate concern; future skill)
- Doesn't push to git

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
