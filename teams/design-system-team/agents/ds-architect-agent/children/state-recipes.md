---
knowledge-base-summary: "Recipes for the non-happy-path UI states — empty / loading / error / success. Live at `ds.json.stateRecipes`, reference voice / motion / iconography tokens so tone propagates."
---
# State Recipes

Recipes for the **non-happy-path** UI states — empty, loading, error, success. State recipes live at the top level of `ds.json` as `stateRecipes`, alongside `components`. They reference tokens from `voice.samples`, `motion`, and `brand.iconography` so a single tone/style shift propagates everywhere.

## Why they exist in the DS

Empty/loading/error/success are not one-off UI decisions — every screen needs them, and without a shared recipe they drift. A DS that declares them:

- Guarantees tone consistency — every empty state reads like the same product
- Avoids bespoke illustrations per screen (early product stages, at least)
- Gives prototype-agent + flutter-agent + react-agent a single source to compose from
- Makes the "what should this screen look like when X" answerable by reading the DS, not guessing

## The four canonical states

Every DS should declare all four. Skipping one means screens will invent their own.

### 1. empty

User has no data yet — first-time, cleared list, filtered-out result. NOT for errors.

Composition: **circular icon badge (brand-tinted) → title → short body → primary CTA**, centered, max-width 280–320px.

Required fields:
- `iconHint` — icon name or role (e.g., "map-pin-off", "search-x")
- `layout.iconSize`, `layout.containerPadding`, `layout.maxWidth`, `layout.align`
- `copyRefs.title` — must reference a `voice.samples.*` key
- `copyRefs.body` — optional, often null (title does the work)
- `copyRefs.action` — must reference a `voice.samples.*` key (or literal label)
- `useWhen` — one-sentence guidance
- `antiPatterns` — array of what to avoid

### 2. loading

Content is on the way. Prefer skeletons when the final shape is known (they feel faster); use spinners for sub-1-second ops inside buttons or small sub-areas.

Two sub-variants:

- **skeleton** — `rowHeight`, `rowGap`, `rowCount` (usually 3–5), `motion` (e.g., "shimmer 1200ms decelerate"), `surface`
- **spinner** — `size`, `strokeWidth`, `color` (brand.primary), `useInside` (where it's appropriate)

Required field `useWhen` guides which sub-variant to pick.

### 3. error

Something failed. Always give a retry path. Never blame the user. Never show raw error codes.

Composition: **circular icon badge (error-tinted) → title → body → secondary retry CTA**, centered, max-width 320–360px.

Required fields:
- `iconHint` — context-aware (triangle-alert / cloud-off / server-crash)
- `variants` object covering `inline`, `fullScreen`, `toast` (each with its own usage note)
- `copyRefs.titleGeneric`, `copyRefs.titleOffline`, `copyRefs.actionLabel`
- `motion` — standard is "160ms fast decelerate in"
- `useWhen` + `antiPatterns`

### 4. success

Action completed. **If the result is obvious on screen (the new item appears in the list), skip the toast** — don't double-confirm. Otherwise a brief toast suffices; only use a blocking modal if the user must acknowledge something.

Composition: **icon + brief message**, no CTA unless acknowledgment needed.

Required fields:
- `iconHint` — usually "check-circle"
- `variants` object covering `toast` (most common) and `inline`
- `copyRefs.title` — must reference a `voice.samples.*` key
- `motion` — standard is "240ms decelerate in · auto-dismiss 3s · 160ms accelerate out"
- `useWhen` + `antiPatterns`

## Schema pattern

```json
"stateRecipes": {
  "empty":   { "composition": "...", "iconHint": "...", "layout": { ... }, "copyRefs": { ... }, "useWhen": "...", "antiPatterns": [ ... ] },
  "loading": { "composition": "...", "skeleton": { ... }, "spinner": { ... }, "useWhen": "...", "antiPatterns": [ ... ] },
  "error":   { "composition": "...", "iconHint": "...", "variants": { "inline": {...}, "fullScreen": {...}, "toast": {...} }, "copyRefs": { ... }, "motion": "...", "useWhen": "...", "antiPatterns": [ ... ] },
  "success": { "composition": "...", "iconHint": "...", "variants": { "toast": {...}, "inline": {...} }, "copyRefs": { ... }, "motion": "...", "useWhen": "...", "antiPatterns": [ ... ] }
}
```

## Rendering in `detail.html`

See [template-rendering.md](template-rendering.md) for section placement. Summary:

- Section sits **between Components and Voice & Tone**
- Each recipe rendered as a full visual example (not just a spec dump): real icons, real copy, real buttons
- Variants of one recipe shown as separate cards (e.g., Loading-skeleton + Loading-spinner = 2 cards)
- Footer block explains the three cross-token relationships (voice / motion / iconography)
- Use the `.dst-state-card`, `.dst-state-card-label`, `.dst-state-card-preview`, `.dst-state-card-usewhen` structural helpers from `styles.css`
- Use `.dst-skeleton-row`, `.dst-spinner`, `.dst-toast`, `.dst-toast-success/-error/-info` recipe-specific helpers
- Brand color overrides (tinted icon badges, primary-colored toasts) go in the page's inline `<style>` block

## What NOT to put here

- **Full-page layouts** (login, dashboard) — those are `pageTemplates`, not state recipes
- **Multi-step flows** (onboarding screen 1 → 2 → 3) — also `pageTemplates` or `patterns`
- **Long illustrations** — v1 uses icon badges, not illustrations. Illustrations come in a later DS expansion tier

## Anti-patterns (aggregate)

- ❌ Empty state with no CTA — dead end
- ❌ Spinner on a full page (skeleton says "here is what's coming")
- ❌ Skeleton that doesn't match the real content shape (feels dishonest)
- ❌ Raw error codes (500, 404) shown to users
- ❌ Alarmist copy — "CRITICAL ERROR", "FATAL"
- ❌ Success toast that duplicates what the UI already shows
- ❌ Blocking success modal for routine operations
- ❌ Recipes defined but unused — prototype-agent should reference `stateRecipes.empty` when designing an empty screen, not reinvent
