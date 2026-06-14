---
name: dst-edit-prototype
description: "/dst-edit-prototype <name> \"<change>\" — apply a textual change to an existing prototype. Reads current prototype.json + linked DS, invokes prototype-agent to interpret the change, updates files, re-renders preview.html and .dst/index.html."
argument-hint: "<prototype-name> \"<change description>\""
---

# /dst-edit-prototype Skill

## Purpose

Apply a free-text change to an existing prototype. Examples:
- "make the submit button blue instead of green"
- "add a 'forgot password' link below the form"
- "shorten the title to 'Sign in'"
- "add a loading state"
- "switch breakpoint to desktop"

## Preconditions

- `.dst/` exists.
- Target prototype exists at `.dst/prototypes/<name>/`.

## Flow

### Phase 1 — Validate

1. `.dst/state.json` exists?
2. `<name>` provided?
3. `.dst/prototypes/<name>/` exists?
4. `<change>` (quoted string) provided?

### Phase 2 — Read context

1. Read `.dst/prototypes/<name>/prototype.json`.
2. Read linked DS at `.dst/design-systems/<linkedDs>/ds.json`.
3. Read `templates/prototype-detail.html.tmpl`.

### Phase 3 — Invoke prototype-agent

Brief the agent with:

```
target prototype: {prototype.json contents}
linked DS: {ds.json contents}
template: {prototype-detail.html.tmpl path}
user change request: "{change}"
```

The agent:
1. Interprets the change request
2. Updates `prototype.json` accordingly:
   - Adding new blocks → blocks array gets new entries
   - Color changes → token references updated (not hardcoded values)
   - State additions → new frame entries
   - Renames → name/displayName fields
3. Preserves token fidelity throughout (never converts a token reference to a hardcoded value)
4. Bumps `prototype.json.version` (patch bump for content edits, minor for state additions)
5. Updates `lastModified`
6. Re-renders `preview.html`

### Phase 4 — Update state.json

Update the prototype's `lastModified` and `version` in `state.json`. Re-render `index.html` if displayName changed.

### Phase 5 — Print summary

```
✓ Updated prototype "{prototype_name}"
  - prototype.json patched: {brief diff summary, e.g., "added 'forgot' link, bumped to v1.0.1"}
  - preview.html re-rendered
  - .dst/state.json updated
```

## Important Rules

1. **Token fidelity preserved.** If user says "make it blue", the agent looks up which DS token is "blue" (e.g., `palette.brand.secondary`) and references that, NOT `#0000FF`.

2. **If the change requires a DS update**, the agent flags it:
   > "Your change requested 'add a third gray tone' — that's a DS-level change. Run `/dst-edit-ds <ds-name> 'add neutral.subtle'` first, then re-request this change."

3. **State additions update breakpoints if needed.** If user says "add a tablet view" but breakpoints didn't include tablet, agent updates breakpoints AND adds the tablet frames.

4. **Don't lose data on edits.** Existing frames not touched by the change must be preserved exactly.

## Edge Cases

- **Change is unclear** — agent asks 1 clarifying question via AskUserQuestion before proceeding.
- **Change requests a feature out of scope** (e.g., "add real-time collaboration") — agent declines: "That's runtime behavior, not a prototype change."
- **Change conflicts with linked DS** (e.g., "use a font we don't have") — surface the conflict, suggest DS update.

## What this skill does NOT do

- Doesn't change `linkedDs` (separate skill if needed in future)
- Doesn't push to git

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
