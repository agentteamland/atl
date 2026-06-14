---
knowledge-base-summary: "Reusable UI compositions — search+filter / master-detail / list+pagination / form composition. Live at `ds.json.patterns`. Encode responsive behavior + component variants + state refs in one place so prototype-agent composes screens from a vocabulary, not from primitives."
---
# Patterns

Patterns are **reusable UI compositions** that sit between individual components and full pages. They live at `ds.json.patterns`, alongside `components` and `stateRecipes`. A pattern is "how these components fit together to solve a recurring interaction problem" — search+filter, master-detail, list+pagination, form composition, etc.

## Why they exist in the DS

Patterns encode **proven solutions** so they aren't re-invented per screen. A DS with patterns:

- Gives prototype-agent a compositional vocabulary (not just primitives)
- Documents responsive behavior in one place (mobile/tablet/desktop per pattern)
- Links components together so changes propagate cleanly (a chip redesign affects every searchFilter pattern instance)
- Ties into state recipes by reference (`patterns.searchFilter.stateRefs.empty = "stateRecipes.empty"`)

## The four canonical patterns (v1)

### 1. searchFilter

Intent: let the user narrow a large set through search + category-style filters.

Composition: **search input (full-width) → chip row (horizontally scrollable) → results (grid or list)**.

Key fields:
- `components` — which component variants compose the pattern (e.g., `input.variants.filled`, `chip.variants.outlined + filled`, `card.variants.interactive`)
- `layoutTokens` — `gap`, `chipRowPaddingY`, `gridGap`
- `responsive` — mobile / desktop behavior
- `stateRefs` — `empty`, `loading`, `error` keys pointing to `stateRecipes.*`
- `useWhen` — one-line guidance on when to pick this pattern
- `antiPatterns` — what to avoid

### 2. masterDetail

Intent: keep a browsable list always visible while the user dives into one item's detail.

Composition: **left column (list, ~320px fixed) + right column (detail, flex-1)**. Mobile stacks and navigates between them.

Key fields:
- `layoutTokens.listWidth`, `layoutTokens.listMaxWidth` (for very wide screens)
- `responsive` — mobile stacks with "back to list" affordance; tablet uses slide-over; desktop side-by-side always
- `stateRefs` — typically `empty` and `loading`

### 3. listPagination

Intent: present a ranked or chronological list with controlled page-by-page navigation.

Composition: **list/grid of cards → pagination control**.

Key fields:
- `pagingMode` — `numbered | cursor | infinite`
- `pagingDefault` — `numbered` for countable sets, `cursor` for infinite feeds, NEVER both
- `responsive` — mobile shows prev/next + total count; desktop shows numbered pages with ellipsis

### 4. formComposition

Intent: group related fields under section headers with consistent label/helper alignment and a clear submit path.

Composition: **section header → field pairs (label above input, helper below) → section gap → final actions row (primary submit + secondary cancel, right-aligned)**.

Key fields:
- `layoutTokens.fieldGap`, `labelToInput`, `helperToInput`, `sectionGap`
- `validation.strategy` — inline under field; summary only when form scrolls
- `validation.trigger` — on blur (not keystroke); on submit for required-field checks
- `responsive` — mobile stacks actions (primary on top, full-width); desktop right-aligns

## Schema pattern

```json
"patterns": {
  "searchFilter":    { "intent": "...", "composition": "...", "components": [ ... ], "layoutTokens": { ... }, "responsive": { ... }, "stateRefs": { ... }, "useWhen": "...", "antiPatterns": [ ... ] },
  "masterDetail":    { "intent": "...", ... },
  "listPagination":  { "intent": "...", ... },
  "formComposition": { "intent": "...", ... }
}
```

## Rendering in `detail.html`

Section sits **between State Recipes and Voice & Tone**. Each pattern is rendered as a **real mini-UI**, not a spec dump:

- **searchFilter preview** → actual `<input>` with search icon, chip row with one `is-selected` chip, 2×2 grid of mini-cards below
- **masterDetail preview** → 40% / 60% split, list on left with one highlighted row, detail panel on right with title + skeleton lines + small CTA
- **listPagination preview** → 3 list-item cards + a numbered pager bar (1 *2* 3 4 5 … 24)
- **formComposition preview** → 2 section headers with labeled inputs, one field showing error state, right-aligned actions row

Shell reuses `.dst-state-card` structure (state recipes + patterns share the preview-card visual language). Pattern previews use the `.dst-pattern-preview` body (full-bleed instead of centered). Helper classes: `.dst-mini-card`, `.dst-mini-list-row`, `.dst-mini-chip`, `.dst-mini-pager` — all neutral; brand-tinted overrides (selected chip color, active page button) go in the page's inline `<style>` block.

## What NOT to put here

- **Full-page layouts** (login, dashboard) — those are `pageTemplates`
- **Individual components** — that's `components`
- **State-only recipes** (empty/loading/error) — those are `stateRecipes`. Patterns reference them, don't redefine them.
- **Deep faceted search sidebars** — too specialized for v1. If product needs this, add as a 5th pattern (`facetedSearch`) when the use case is concrete.

## Anti-patterns (aggregate)

- ❌ Filter chips shown alongside a deep faceted sidebar (pick one approach)
- ❌ Search bar without visible submit affordance on mobile
- ❌ Master-detail where the detail replaces the list on desktop (defeats the pattern)
- ❌ Pagination + infinite scroll mixed on the same list
- ❌ Page numbers when total count is unknowable
- ❌ Validation that fires on every keystroke (noisy)
- ❌ Cancel button more prominent than submit
- ❌ Placeholder used as label replacement (fails when the user starts typing)
- ❌ Pattern defined but unused — prototype-agent should reference `patterns.*` when composing, not reinvent
