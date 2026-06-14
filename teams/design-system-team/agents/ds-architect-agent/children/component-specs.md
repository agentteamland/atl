---
knowledge-base-summary: "Tier policy (`core` ~12 vs `full` ~42), 4-section schema (variants/sizes/states/tokens), group-driven file layout, anti-patterns. Anchor for the per-group children."
---
# Component Specs (Index)

Components defined in `ds.json` are **type-level specs** — variant matrices, size tables, state lists. The actual implementation (HTML, JSX, Dart) is the prototype-agent's job. The DS just declares "buttons exist, they have these variants, these sizes, these states."

This file is the **index**. Per-group details live in `components/`:

- [Actions](components/actions.md) — button, icon-button
- [Forms](components/forms.md) — input, textarea, label, select, checkbox, radio, switch, slider, combobox, input-otp, file-upload
- [Feedback](components/feedback.md) — alert, toast, tooltip, progress, skeleton, spinner, banner, callout, notification-bell
- [Navigation](components/navigation.md) — tabs, breadcrumb, pagination, menu, sidebar-nav, stepper, link, back-button, bottom-navigation
- [Display](components/display.md) — avatar, badge, chip, card, divider, table, list, tree, rating, empty-state, accordion, stat, code, kbd
- [Overlays](components/overlays.md) — modal, drawer, sheet, popover, dropdown, command-menu, date-picker, carousel, confirm-dialog
- [Charts](components/charts.md) — chart, sparkline, gauge

## Component scope tiers

Pick one with the user during Q&A:

**core** (default — ~12 components, covers ~80% of UI needs):
- `button`, `input`, `textarea`, `label`, `card`, `badge`, `chip`, `switch`, `checkbox`, `radio`, `modal`, `alert`

**full** (~57 components, comprehensive UI library):
- All of core + the rest of the catalog across all 7 groups. Includes (in addition to core): `icon-button`, `select`, `slider`, `combobox`, `input-otp`, `file-upload`, `toast`, `tooltip`, `progress`, `skeleton`, `spinner`, `banner`, `callout`, `notification-bell`, `tabs`, `breadcrumb`, `pagination`, `menu`, `sidebar-nav`, `stepper`, `link`, `back-button`, `bottom-navigation`, `avatar`, `divider`, `table`, `list`, `tree`, `rating`, `empty-state`, `accordion`, `stat`, `code`, `kbd`, `drawer`, `sheet`, `popover`, `dropdown`, `command-menu`, `date-picker`, `carousel`, `confirm-dialog`, `chart`, `sparkline`, `gauge`.

The earlier `minimal / standard / extensive` triple was retired in Faz 2 — when the full set reached ~40, "extensive" no longer meant "comprehensive" and the middle tier was indistinguishable from the bottom. Faz 4 expanded the full set from 42 → 57 (12 Tier 1 primitives + 3 Charts components) and opened Charts as the 7th group.

## Per-component spec shape (mandatory)

Every component in `ds.json` MUST declare these four sections — even if a section is short:

```jsonc
{
  "<componentName>": {
    "variants": { "<variantName>": { /* style tokens */ } },
    "sizes":    { "<sizeName>":    { /* dimension tokens */ } },
    "states":   ["idle", "hover", "focus", "pressed", "disabled", ...],
    "stateRules": { "<stateName>": "describe the visual change in one sentence" }
    // …plus any component-specific tokens (helper, label, menu, motion, …)
  }
}
```

If a component genuinely doesn't have variants (e.g., divider), declare `"variants": { "default": { ... } }` with one entry — never omit the key. Schema uniformity makes prototype-agent's job tractable: it can iterate every `components.*` and rely on the four keys existing.

## Group structure

Components are organized into seven groups. Each group has its own children file with the JSON specs and group-specific guidance. The groups also drive the `detail.html` rendering: each component sits inside a `<div class="ds-component-group">` block with an anchor `id="components-{slug}"`.

| Group | Slug | Components |
|---|---|---|
| Actions | `actions` | button, icon-button |
| Forms | `forms` | input, textarea, label, select, checkbox, radio, switch, slider, combobox, input-otp, file-upload |
| Feedback | `feedback` | alert, toast, tooltip, progress, skeleton, spinner, banner, callout, notification-bell |
| Navigation | `navigation` | tabs, breadcrumb, pagination, menu, sidebar-nav, stepper, link, back-button, bottom-navigation |
| Display | `display` | avatar, badge, chip, card, divider, table, list, tree, rating, empty-state, accordion, stat, code, kbd |
| Overlays | `overlays` | modal, drawer, sheet, popover, dropdown, command-menu, date-picker, carousel, confirm-dialog |
| Charts | `charts` | chart, sparkline, gauge |

Group ordering in `detail.html` (sidebar TOC + components section): `actions → forms → feedback → navigation → display → overlays → charts`. Charts sits last — it's the most domain-specific and absent from many product DSes (marketing sites, content apps, simple CRUD).

## What goes in `ds.json` vs prototype

`ds.json` defines the **contract**: what variants exist, what sizes, what states. Visual examples are rendered in `detail.html`.

**Prototype-agent** consumes this contract: when it generates a login screen, it uses `button.variants.primary` and `button.sizes.lg`. It doesn't invent a "primary-emphasis" variant — if the screen needs one, the user adds it to `ds.json` first.

This boundary keeps the DS as the source of truth for what's allowed.

## Anti-patterns

- ❌ **Variant explosion.** If you can't articulate when to use each variant in one sentence, you have too many. Cap at 4–5 unless the component genuinely needs more (e.g., button has 4 — primary, secondary, ghost, danger).
- ❌ **Skipping `states` because they're "obvious."** Future agents (and humans) need the full state list to design comprehensively. Always declare hover/focus/disabled at minimum.
- ❌ **Storing actual hex values in component specs.** Reference palette tokens (`"background": "brand.primary"`) so theme changes propagate automatically.
- ❌ **Forgetting `disabled` state.** Most products silently hardcode it later (and inconsistently).
- ❌ **Mixing groups.** Don't put `tabs` in Display — it's Navigation. Don't put `tooltip` in Overlays — it's Feedback (announces system state, not floats above content like a modal does). Group membership drives both children file location AND `detail.html` ordering — getting it wrong causes orphaned anchors and confused readers.
- ❌ **Adding a component without the 4-section shape.** Even if `sizes` only has one entry, declare it. Schema uniformity is a hard rule.
