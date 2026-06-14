---
name: dst-init
description: "/dst-init — bootstrap or refresh the .dst/ directory in the current project. BOOTSTRAP: creates state.json + styles.css + tailwind.min.css + empty index.html on fresh projects. REFRESH: safely re-copies templates/CSS and re-renders every HTML page from preserved JSON state, so upgrading the team package always yields up-to-date visuals without touching user content."
---

# /dst-init Skill

## Purpose

Set up (or refresh) the `.dst/` directory in the current project so other `/dst-*` skills have a place to write to, and so template/CSS updates from new team versions can be applied idempotently without losing any design-system or prototype content.

## Two modes

```
cwd/.dst/ missing?                           → BOOTSTRAP MODE
cwd/.dst/ exists + state.json valid?         → REFRESH MODE
cwd/.dst/ exists + state.json missing?       → ERROR (corrupt state)
```

Both modes end with a printed summary and suggest `/dst-open` to view.

## Detect

1. Resolve project root as `cwd`. The `.dst/` directory lives at `<cwd>/.dst/`.
2. Check `.dst/` existence:
   - Missing → BOOTSTRAP MODE.
   - Exists + `.dst/state.json` readable JSON → REFRESH MODE.
   - Exists + no `state.json` (or unparseable) → error: "found .dst/ without a valid state.json — remove or move it aside, then re-run /dst-init."

## Locating the team templates

Templates live in the cached team repo:

```
~/.claude/repos/agentteamland/design-system-team/templates/
```

Check existence; if not found, error with: "design-system-team not properly installed; re-run `atl install design-system-team`."

## BOOTSTRAP MODE (fresh init)

Use this flow when `.dst/` does not exist.

### Step 1 — Create directory tree

```
.dst/
  index.html               ← empty landing
  styles.css               ← copy from team's templates/styles.css.tmpl
  tailwind.min.css         ← copy from team's templates/tailwind.min.css (~2.8MB, one-time, self-contained Tailwind v2)
  state.json               ← starter manifest
  design-systems/          ← empty dir (DSes go here later)
  prototypes/              ← empty dir (prototypes go here later)
```

**Why local Tailwind file?** Browsers block CDN scripts on `file://` URLs (unique-origin security). The team ships a pre-built Tailwind CSS so pages render correctly when opened directly from disk — no server needed, no CDN needed.

### Step 2 — Create `state.json`

```json
{
  "schemaVersion": 1,
  "projectName": "<derived from CWD basename>",
  "createdAt": "<now ISO>",
  "lastUpdated": "<now ISO>",
  "designSystems": [],
  "prototypes": []
}
```

### Step 3 — Copy two CSS files from team repo to `.dst/`

- `templates/tailwind.min.css` → `.dst/tailwind.min.css` (binary copy, ~2.8MB, no template substitution)
- `templates/styles.css.tmpl` → `.dst/styles.css` (static, no template substitution needed)

### Step 4 — Render initial `index.html`

Render from `templates/index.html.tmpl` with:

- `{{ projectName }}` → derived from CWD basename
- `{{ lastUpdated }}` → "just now"
- `{{ designSystemsCount }}` → 0
- `{{ prototypesCount }}` → 0
- `{{ DESIGN_SYSTEMS_CARDS }}` → empty-state markup ("No design systems yet. Run /dst-new-ds <name>")
- `{{ PROTOTYPES_CARDS }}` → empty-state markup

Save to `.dst/index.html`.

### Step 5 — Add `.dst/cache/` to `.gitignore`

Only if `.gitignore` exists in the project and the rule isn't already there. The rest of `.dst/` should be committed.

### Step 6 — Print summary

```
✓ Initialized .dst/ in <project-name>
  - .dst/index.html         (open in browser to view)
  - .dst/state.json         (manifest)
  - .dst/styles.css         (shared styles)
  - .dst/tailwind.min.css   (Tailwind v2 build, local — works on file://)

Next steps:
  /dst-new-ds <name>           Create your first design system
  /dst-open                    Open the studio in your browser
```

## REFRESH MODE (re-runnable upgrade)

Use this flow when `.dst/state.json` already exists. Goal: pull in any template / CSS changes from the current team version **without touching user content** (`ds.json`, `prototype.json`, `assets/`).

### Step 1 — Verify templates

Confirm every template file exists in the team repo:
- `templates/tailwind.min.css`
- `templates/styles.css.tmpl`
- `templates/index.html.tmpl`
- `templates/ds-detail.html.tmpl`
- `templates/prototype-detail.html.tmpl`

If any are missing → error with the missing path; don't touch `.dst/`.

### Step 2 — Re-copy shared CSS (always overwrite)

- `templates/tailwind.min.css` → `.dst/tailwind.min.css`
- `templates/styles.css.tmpl` → `.dst/styles.css`

These are purely derivable, never user-edited. Safe to overwrite.

### Step 3 — Re-render `.dst/index.html`

Re-render from `templates/index.html.tmpl` using the current (preserved) `.dst/state.json`:

- `{{ projectName }}` → `state.json.projectName`
- `{{ lastUpdated }}` → now ISO (bumped in step 6)
- `{{ designSystemsCount }}` → `state.json.designSystems.length`
- `{{ prototypesCount }}` → `state.json.prototypes.length`
- `{{ DESIGN_SYSTEMS_CARDS }}` → one card per DS (empty-state markup if 0)
- `{{ PROTOTYPES_CARDS }}` → one card per prototype (empty-state markup if 0)

The skill itself can do this substitution (pure template-fill; no semantic reasoning needed).

### Step 4 — Regenerate every DS's `detail.html`

For each `ds` in `state.json.designSystems`:

1. Read `.dst/design-systems/<ds.name>/ds.json` (do NOT modify).
2. Invoke `ds-architect-agent` with a self-contained brief:

   ```
   Re-render the detail page for "<ds.name>" from its current ds.json.
   DO NOT change ds.json. Only overwrite detail.html.
   Team template: ~/.claude/repos/agentteamland/design-system-team/templates/ds-detail.html.tmpl
   Source state: .dst/design-systems/<ds.name>/ds.json
   Output: .dst/design-systems/<ds.name>/detail.html
   ```

3. If the agent fails for this DS, capture the failure and continue with the next DS; include in the final summary.

### Step 5 — Regenerate every prototype's `preview.html`

For each `prototype` in `state.json.prototypes`:

1. Read `.dst/prototypes/<prototype.name>/prototype.json` (do NOT modify).
2. Read linked DS `.dst/design-systems/<prototype.linkedDs>/ds.json` (do NOT modify).
3. Invoke `prototype-agent` with a self-contained brief:

   ```
   Re-render the preview page for "<prototype.name>" from its current prototype.json.
   DO NOT change prototype.json. Only overwrite preview.html.
   Team template: ~/.claude/repos/agentteamland/design-system-team/templates/prototype-detail.html.tmpl
   Source state: .dst/prototypes/<prototype.name>/prototype.json
   Linked DS: .dst/design-systems/<prototype.linkedDs>/ds.json
   Output: .dst/prototypes/<prototype.name>/preview.html
   ```

4. If the agent fails for this prototype, capture the failure and continue; include in the final summary.

### Step 6 — Bump `state.json.lastUpdated`

Write the same `state.json` back with only `lastUpdated` updated to now ISO. All other fields preserved.

### Step 7 — Print summary

```
✓ Refreshed .dst/ from current team templates
  - tailwind.min.css + styles.css copied fresh
  - index.html re-rendered
  - <N> design system(s) regenerated: <names>
  - <M> prototype(s) regenerated: <names>
  <if any failures:>
  ⚠ Skipped due to error:
    - <name>: <error summary>

Open .dst/index.html in your browser to view the refreshed studio.
```

## Important rules for REFRESH MODE

1. **Never touch `ds.json` or `prototype.json`.** They are user content. Read-only from this skill's perspective.
2. **Never touch `assets/` directories.** User-uploaded artifacts stay exactly as-is.
3. **Always overwrite rendered HTML + CSS.** They are fully derivable from state + templates; a fresh render is always safe.
4. **Atomic-ish:** per-DS and per-prototype regen runs independently. One failure does not abort the whole refresh.
5. **Skill does `index.html` directly.** It's pure substitution (no semantic fill). Per-DS and per-prototype HTML need their owning agent because templates contain semantic-fill regions the skill can't author.
6. **Verify after:** A successful run should leave JSON files byte-identical (except `state.json.lastUpdated`) — only HTML + CSS files show up in `git diff`.

## Edge cases

- **`.dst/` exists but `state.json` is unreadable JSON** — treat as corrupt, error out, don't touch anything. Instructs user to remove or move aside.
- **Project has no `.gitignore`** (bootstrap only) — fine, just skip the gitignore step.
- **Template files missing in team repo** — error with explicit path; don't silently degrade or leave `.dst/` half-refreshed.
- **A referenced DS directory is missing during refresh** (e.g., user deleted it manually) — skip that entry and note in summary; don't crash.
- **A prototype references a `linkedDs` whose directory is missing** — skip that prototype and note in summary (the prototype is effectively orphaned; user should run `/dst-delete-prototype` or restore the DS).

## What this skill does NOT do

- Doesn't create design systems or prototypes (those are `/dst-new-ds`, `/dst-new-prototype`).
- Doesn't read project tokens (that's `ds-architect-agent`'s job during `/dst-new-ds`).
- Doesn't open the browser (that's `/dst-open`).
- Doesn't install team dependencies (atl handles that during install).
- Doesn't migrate `state.json` schema versions — if `schemaVersion` ever bumps, a dedicated migration skill is introduced.

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
