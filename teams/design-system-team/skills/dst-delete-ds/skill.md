---
name: dst-delete-ds
description: "/dst-delete-ds <name> — remove a design system from the project's .dst/. Warns if any prototype depends on this DS (and refuses without --force). Updates .dst/state.json and re-renders .dst/index.html."
argument-hint: "<ds-name> [--force]"
---

# /dst-delete-ds Skill

## Purpose

Permanently remove a design system. Designed to be safe-by-default (refuses if dependent prototypes exist) but escapable with `--force`.

## Flow

### Phase 1 — Validate

1. `.dst/state.json` exists? If not → "No design systems found. Run /dst-init."
2. `<name>` arg provided? If not → list current DSes, ask which to delete.
3. `.dst/design-systems/<name>/` exists? If not → "Design system `<name>` not found. Available: …"

### Phase 2 — Dependency check

Read `.dst/state.json`. For every prototype with `linkedDs == name`, collect them into a list.

If list non-empty AND `--force` was NOT passed:
```
Cannot delete design system "primary" — these prototypes depend on it:
  - login-screen
  - dashboard-empty
  - profile

Options:
  - Delete the prototypes first: /dst-delete-prototype <name>
  - Re-link them to a different DS (manual ds.json edit, /dst-edit-prototype to come)
  - Force-delete this DS anyway (orphans the prototypes): /dst-delete-ds primary --force
```
Stop.

If `--force` was passed → proceed. Print: "Force-deleting; the following prototypes will be orphaned: …"

### Phase 3 — Delete

1. `rm -rf .dst/design-systems/<name>/` (the entire DS directory).
2. Remove the DS entry from `.dst/state.json`'s `designSystems` array.
3. If `--force` AND there were dependent prototypes: in each prototype's `prototype.json`, set `linkedDs: null` and add a note in `state.json` flagging the prototype as orphaned.
4. Update `state.json`'s `lastUpdated`.

### Phase 4 — Re-render

Re-render `.dst/index.html` from `templates/index.html.tmpl` reflecting the updated state.

### Phase 5 — Print summary

```
✗ Deleted design system "primary"
  - .dst/design-systems/primary/ removed
  - .dst/state.json updated
  - .dst/index.html re-rendered
  [if force]: 3 prototypes orphaned: login-screen, dashboard-empty, profile
```

## Important Rules

1. **Never delete without confirmation when prototypes depend** — `--force` is the explicit override.
2. **Preserve cached repo** — only `.dst/` is touched. The cached team source under `~/.claude/repos/agentteamland/design-system-team/` is untouched.
3. **Manifest update is atomic with deletion** — write state.json AFTER successful directory removal, not before.

## Edge Cases

- **DS exists in filesystem but not in state.json** — anomaly. Print warning, delete the directory, no state update needed (it wasn't there).
- **DS in state.json but directory missing** — anomaly. Just remove the state entry.
- **Last DS in project** — works fine. State.json has empty `designSystems: []` array.

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
