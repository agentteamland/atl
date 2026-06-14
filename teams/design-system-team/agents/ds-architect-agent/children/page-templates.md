---
knowledge-base-summary: "Whole-page composition blueprints — login / onboarding / profile / settings. Live at `ds.json.pageTemplates`. Each names its regions, references the patterns each region composes, and declares responsive behavior. Rendered as abstract wireframes (NOT mockups — that contrast is intentional)."
---
# Page Templates

Page templates are **whole-page composition blueprints**. They live at `ds.json.pageTemplates`, alongside `components`, `stateRecipes`, and `patterns`. A page template names the regions of a page, declares how those regions compose, and references the patterns each region uses.

## Why they exist in the DS

Without page templates, every new screen is designed from scratch even when it's structurally identical to an existing one. A DS with templates:

- Gives prototype-agent a scaffolding vocabulary at the page level (not just component or pattern level)
- Encodes responsive behavior per region, consistent across pages
- Lets changes to a pattern (e.g., `formComposition`) automatically flow into every template that references it
- Prevents reinvention of canonical flows (login, onboarding, settings)

## The four canonical templates (v1)

### 1. login

Intent: sign-in screen for existing users.

Layout: **single-column centered card**. Brand above card, card in middle (using `patterns.formComposition`), belowCard links (forgot password / signup).

Key fields:
- `regions` — `[brand, card, belowCard]`
- `layoutTokens.cardMaxWidth`, `verticalCenter`, `pagePadding`
- `stateRefs.error` → `stateRecipes.error.inline`, `stateRefs.loading` → spinner inside submit button
- `useWhen` — existing users. For new accounts, a separate `signup` template (not in v1)

### 2. onboarding

Intent: first-run multi-step introduction — one concept per step.

Layout: **full-screen per step with top progress indicator**. Regions: `progress`, `visual`, `copy`, `actions`.

Key fields:
- `stepCount` — recommend 3–5. Past 4 steps drop-off compounds.
- `regions` — each gets its own blueprint region
- `responsive` — mobile pins primary CTA to bottom with safe-area; desktop centers a card
- `useWhen` — first run only, guarded by a persisted flag

### 3. profile

Intent: view of a user (self or other) with their identity and activity.

Layout: **hero → tabs → content**. Content uses `patterns.listPagination`.

Key fields:
- `regions` — `[hero, tabs, content]`
- `patternsUsed` — `["listPagination"]`
- `responsive` — mobile stacks hero vertically (avatar above name); desktop side-by-side
- `stateRefs.empty` → active tab's empty state

### 4. settings

Intent: user preferences and account management.

Layout: **sidebar nav + main content**. Main uses `patterns.formComposition`.

Key fields:
- `regions` — `[sidebar, main]`
- `layoutTokens.sidebarWidth`, `mainMaxWidth`, `gap`
- `patternsUsed` — `["formComposition"]`
- `responsive` — mobile collapses sidebar into top tabs
- `stateRefs.success` → `stateRecipes.success.toast` after save

## Schema pattern

```json
"pageTemplates": {
  "login": {
    "intent": "...",
    "layout": "...",
    "regions": [
      { "name": "brand",     "composition": "..." },
      { "name": "card",      "composition": "...", "patternRef": "patterns.formComposition" },
      { "name": "belowCard", "composition": "..." }
    ],
    "layoutTokens": { ... },
    "responsive": { ... },
    "stateRefs": { ... },
    "useWhen": "...",
    "antiPatterns": [ ... ]
  },
  "onboarding": { ... },
  "profile":    { "patternsUsed": ["listPagination"], ... },
  "settings":   { "patternsUsed": ["formComposition"], ... }
}
```

## Rendering in `detail.html`

Section sits **between Patterns and Voice & Tone**.

**Critical rendering rule:** page templates are rendered as **abstract wireframes**, NOT as real mini-UIs. This is the key visual distinction from Patterns (which ARE rendered as real mini-UIs).

Why: a pattern is about the **instance** (what it looks like with real data); a template is about the **structure** (what regions exist and how they compose). Blueprints communicate structure more honestly than mockups — they don't bias the viewer with specific content decisions that haven't been made yet.

Shell reuses `.dst-state-card` structure. Blueprint body is `.dst-blueprint` (dashed-border regions with labels). Helper classes: `.dst-blueprint-region` (with `data-label="region-name"`), `.dst-blueprint-sketch` (horizontal bars for text), `.dst-blueprint-sketch.is-title` (thicker, shorter), `.dst-blueprint-sketch.is-muted` (lighter), `.dst-blueprint-avatar`, `.dst-blueprint-button` + `.is-ghost`, `.dst-blueprint-dots` (progress), `.dst-blueprint-icon-hero`, `.dst-blueprint-tab` + `.is-active`, `.dst-blueprint-navitem` + `.is-active`.

Brand-tinted overrides (active tab color, active nav-item background, primary button color) go in the page's inline `<style>` block.

## What NOT to put here

- **Specific mockups with real copy/imagery** — that's prototype-agent's output
- **Component implementations** — that's `components` (and ultimately flutter-agent / react-agent)
- **Small reusable compositions** — those are `patterns`
- **One-off screens unique to the product** — templates are DS-level, only include truly canonical flows (login is canonical; "a downstream project's deeply-nested wizard step" is not)

## Anti-patterns (aggregate)

- ❌ Login CTA labeled "Submit" — use action-specific language ("Sign in", "Log in")
- ❌ Skip link in onboarding hidden or styled as disabled
- ❌ Progress indicator that doesn't update on step change
- ❌ 8+ onboarding steps — users drop off past step 4
- ❌ Profile where hero is the same for self and others (missing edit affordance on own profile)
- ❌ Settings with 8+ sidebar items — split into primary + advanced
- ❌ Save button that appears only after a change AND is invisible until then — use enabled/disabled state instead
- ❌ Templates redefining patterns (every login form reinvents validation) — reference `patterns.formComposition`
- ❌ Rendering templates as mockups with specific content — keep them as structural blueprints in the DS; mockups are prototype-agent's job
