---
name: dst-delete-prototype
description: "/dst-delete-prototype <name> — remove a prototype from the project's .dst/. Updates .dst/state.json and re-renders .dst/index.html."
argument-hint: "<prototype-name>"
---

# /dst-delete-prototype Skill

## Purpose

Remove a prototype permanently. Simpler than delete-ds because prototypes have no dependents.

## Flow

### Phase 1 — Validate

1. `.dst/state.json` exists?
2. `<name>` provided? If not → list prototypes, ask.
3. `.dst/prototypes/<name>/` exists?

### Phase 2 — Delete

1. `rm -rf .dst/prototypes/<name>/`
2. Remove the prototype entry from `.dst/state.json`'s `prototypes` array.
3. Decrement the `prototypesCount` of the linked DS (in state.json).
4. Update `lastUpdated`.

### Phase 3 — Re-render

Re-render `.dst/index.html`.

### Phase 4 — Print summary

```
✗ Deleted prototype "login-screen"
  - .dst/prototypes/login-screen/ removed
  - .dst/state.json updated
  - .dst/index.html re-rendered
```

## Edge Cases

- **Prototype dir exists but not in state.json** — print warning, delete the directory.
- **State entry exists but directory missing** — just remove the state entry.
- **Linked DS already deleted** — no decrement needed; just clean up.

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
