---
name: dst-new-ds
description: "/dst-new-ds <name> — create a new design system in the current project's .dst/ directory. Asks an interactive set of Q&A to gather brand intent + scope, then invokes ds-architect-agent to produce ds.json + Tailwind-rendered detail.html. Updates .dst/state.json and re-renders .dst/index.html."
argument-hint: "<ds-name>"
---

# /dst-new-ds Skill

## Purpose

Create a new design system inside the project, fully populated (palette, typography, spacing, components, brand, voice, accessibility), and render its visual detail page so the user can open it in their browser.

## Preconditions

- `.dst/` exists in current project (run `/dst-init` first if not).
- `<ds-name>` argument is provided (kebab-case, e.g., `primary`, `wfm-admin`).

## Flow

### Phase 1 — Validate

1. Check `.dst/state.json` exists. If not → error: "Run `/dst-init` first."
2. Read `<ds-name>` from arg. Validate kebab-case (lowercase letters, digits, hyphens). Reject otherwise.
3. Check `.dst/design-systems/<ds-name>/` doesn't already exist. If it does → error: "Design system `<ds-name>` already exists. Use `/dst-edit-ds <ds-name>` to modify, or pick a different name."

### Phase 2 — Read project context

Before asking the user anything, scan the project. The findings inform Q&A defaults.

Look for (using `Glob` + `Read`):
- `flutter/lib/app/theme.dart` (or any `**/theme.dart`)
- `tailwind.config.ts`, `tailwind.config.js` (any path)
- `pubspec.yaml`
- `package.json`
- Existing `.dst/design-systems/*/ds.json` (if any — for naming/style consistency)

Build a mental "project context" summary you'll reference during Q&A.

### Phase 3 — Interactive Q&A

Use `AskUserQuestion` for each. Present project-context-derived defaults when available.

**Q1 — Brand origin**
"Where should the brand come from?"
- (a) Use existing project tokens *(only if found in Phase 2)*
- (b) Generate from a seed color
- (c) Generate from personality keywords (auto-pick palette)

**Q2 — Brand color** *(only if Q1 = b)*
"What's the brand seed color? (hex like `#2D5F3F`, or describe — 'forest green', 'electric blue')"
Free text. Convert description to hex if needed.

**Q3 — Personality**
"Pick a personality flavor for this brand:"
single-select (NOT multiSelect — `AskUserQuestion` caps at 4 options per question, which rules out a free-form keyword list). Curate 4 flavor bundles from project context; each option bundles 3-4 keywords from the vocabulary below.

Keyword vocabulary (compose bundles from these):
warm, professional, playful, minimal, bold, calm, outdoor, urban, friendly, technical, elegant, energetic, trustworthy, approachable, refined, raw

Example bundles (adapt to the project's tone):
- Outdoor & human — outdoor, approachable, trustworthy, warm
- Minimal & refined — minimal, refined, calm, trustworthy
- Energetic & playful — energetic, playful, warm, approachable
- Professional & technical — professional, technical, minimal, trustworthy

**Q4 — Typography**
"Typography choice:"
- (a) System fonts only (zero load cost, native feel)
- (b) Inter for body + display (default; Google Fonts; Turkish-safe)
- (c) Custom: specify family

**Q5 — Density**
"UI density:"
- (a) Compact (admin / dashboard)
- (b) Comfortable (default; consumer apps)
- (c) Spacious (marketing / onboarding)

**Q6 — Dark mode**
"Include dark mode from day 1?"
- Yes
- No (can add later via /dst-edit-ds)

**Q7 — Component scope**
"Component scope:"
- Minimal (button, input, card)
- Standard (+ chip, toggle, modal, select)
- Extensive (+ tabs, alert, tooltip, dropdown, progress, etc.)

### Phase 4 — Invoke ds-architect-agent

Call the agent with a self-contained brief:

```
target ds_name: {ds_name}
project context: {summary from phase 2}
user answers: {q1-q7}
team repo path: ~/.claude/repos/agentteamland/design-system-team/
output directory: .dst/design-systems/{ds_name}/
team templates: templates/ds-detail.html.tmpl
```

The agent:
1. Synthesizes a complete `ds.json` per the schema (see agent's `children/ds-schema.md`)
2. Computes WCAG contrast ratios for palette pairings
3. Renders `detail.html` from `templates/ds-detail.html.tmpl`
4. Writes both to `.dst/design-systems/{ds_name}/`
5. If logomark is needed and not provided → generates a placeholder SVG to `assets/logomark.svg` and notes it as placeholder in ds.json

### Phase 5 — Update state and re-render landing

1. Update `.dst/state.json`:
   - Append `{ name: ds_name, version: "1.0.0", createdAt: now, lastModified: now, prototypesCount: 0 }` to `designSystems`
   - Bump top-level `lastUpdated`

2. Re-render `.dst/index.html` from `templates/index.html.tmpl` with the updated state. Replace `{{ DESIGN_SYSTEMS_CARDS }}` placeholder with one card per DS in state.json (use the EXAMPLE PATTERN comment in the template).

### Phase 6 — Print summary

```
✓ Created design system "{ds_name}"
  - .dst/design-systems/{ds_name}/ds.json
  - .dst/design-systems/{ds_name}/detail.html
  - .dst/state.json updated
  - .dst/index.html re-rendered

Open in browser:
  /dst-open                    (full studio)
  open .dst/design-systems/{ds_name}/detail.html   (this DS only)
```

## Important Rules

1. **Never overwrite existing `ds.json`** without explicit user confirmation. The validation in Phase 1 catches this; if somehow bypassed, double-check before write.

2. **`ds-architect-agent` is mandatory.** Don't try to write `ds.json` directly from this skill — the agent encodes the full design knowledge (palette theory, typography ramps, etc.). Skill is the orchestrator; agent is the authority.

3. **Render quality matters.** `detail.html` should look polished in a browser. Verify after writing by checking file exists and is non-trivial in size (> 5KB).

4. **State.json is the project-wide manifest.** Always update both `state.json` AND re-render `index.html` after creating a DS. Otherwise the landing page goes stale.

5. **Honor user time.** Q&A should be ~6 questions, not 20. Defaults from project context reduce question count when possible (e.g., skip Q2 if Q1 picked existing tokens).

## Edge Cases

- **User cancels mid-Q&A** — don't write any files. Don't update state. Skill is a no-op.
- **Project has multiple `theme.dart` candidates** — ask user which one to use (or default to most recently modified).
- **`<ds-name>` collides with reserved name** ("template", "shared", "default") — warn but allow if user confirms.
- **Network unavailable** (Google Fonts can't load in browser) — DS still works; rendered detail.html will fall back to system fonts. Don't fail.

## What this skill does NOT do

- Doesn't create prototypes (that's `/dst-new-prototype`).
- Doesn't open the browser (that's `/dst-open`).
- Doesn't push to git or anything outside `.dst/`.
- Doesn't read or modify project source files (only their config to inform DS).

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
