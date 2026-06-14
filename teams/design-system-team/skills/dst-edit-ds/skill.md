---
name: dst-edit-ds
description: "/dst-edit-ds <name> \"<change>\" — apply a textual change to an existing design system. Reads current ds.json, invokes ds-architect-agent to interpret the change, updates files, re-renders detail.html and .dst/index.html. Optionally re-renders linked prototypes if structural changes affect them."
argument-hint: "<ds-name> \"<change description>\""
---

# /dst-edit-ds Skill

## Purpose

Apply a free-text change to an existing design system. Examples:
- "add a warning color to the palette"
- "make the body font Manrope instead of Inter"
- "tighten spacing scale to 8px grid"
- "add an 'extensive' button variant set"
- "update voice tone to be more playful"

## Preconditions

- `.dst/` exists.
- Target DS exists at `.dst/design-systems/<name>/`.

## Flow

### Phase 1 — Validate

1. `.dst/state.json` exists?
2. `<name>` provided?
3. `.dst/design-systems/<name>/ds.json` exists?
4. `<change>` (quoted) provided?

### Phase 2 — Read context

1. Read `.dst/design-systems/<name>/ds.json`.
2. Read `templates/ds-detail.html.tmpl`.
3. Scan `.dst/state.json` for prototypes linked to this DS — they may need re-rendering after changes.

### Phase 3 — Invoke ds-architect-agent

Brief the agent with:

```
target ds: {ds.json contents}
template: {ds-detail.html.tmpl path}
linked prototypes (will be affected if structural change): {prototype names}
user change request: "{change}"
```

The agent:
1. Interprets the change
2. Updates `ds.json`:
   - Palette additions → palette section grows
   - Font changes → typography.fontFamilies updated
   - Spacing changes → spacing.scale array updated
   - Component additions → components section grows
   - Voice changes → voice samples updated
   - Accessibility commitments → accessibility section updated
3. Re-computes WCAG contrast ratios if palette changed
4. Bumps `ds.json.version` (patch for content, minor for additions, major for breaking removals)
5. Updates `lastModified`
6. Re-renders `detail.html`

### Phase 4 — Cascade to linked prototypes (if structural change)

A "structural change" is one that affects token references prototypes might use:
- Palette changes (rename, removal)
- Spacing scale changes
- Component variant removal
- Typography family/scale changes

For each linked prototype:
1. Read its `prototype.json`
2. Check if any token references it uses are affected by the change
3. If yes:
   - For removals: print warning, set the affected token reference to `{{ ds.<old-path> | default: '<placeholder>' }}` and add a `prototype.json.notes` entry
   - For renames: update the path
4. Re-render the prototype's `preview.html` using the updated DS

For non-structural changes (additions, voice updates, accessibility tightening) — no prototype updates needed; existing references still resolve.

### Phase 5 — Update state.json

1. Update DS's `version` and `lastModified` in `state.json`.
2. If linked prototypes were updated, bump their `lastModified` too.
3. Re-render `index.html`.

### Phase 6 — Print summary

```
✓ Updated design system "{ds_name}" (v{old_version} → v{new_version})
  - ds.json patched: {brief diff summary, e.g., "added warning color, recomputed contrast"}
  - detail.html re-rendered
  [if cascade]: 3 linked prototypes re-rendered: login-screen, dashboard-empty, profile
  - .dst/state.json updated
```

## Important Rules

1. **Re-compute contrast when palette changes.** Don't ship a DS where contrast metadata is stale.

2. **Cascade structural changes to prototypes.** Don't leave dangling references unhandled.

3. **Don't silently break prototypes.** If a structural change WOULD break a prototype's resolution, surface that to the user and add a fallback (don't just remove the value).

4. **Voice updates are content-only.** Updating `voice.samples.welcome` doesn't break prototypes; just re-renders.

5. **Major version bump for breaking changes.** If user removes the brand primary color (or renames it), bump major and warn.

## Edge Cases

- **Change is ambiguous** — agent asks 1 clarifying question.
- **Change would orphan tokens used by prototypes** — surface, suggest alternative (deprecate vs. rename).
- **Change introduces accessibility regression** (e.g., new color fails WCAG) — flag in `accessibility.notes` AND in summary; don't auto-block.

## What this skill does NOT do

- Doesn't change `name` (DS name is its identity; rename via delete + recreate)
- Doesn't push to git
- Doesn't auto-fix accessibility issues — surfaces them, leaves resolution to user

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
