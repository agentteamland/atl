---
knowledge-base-summary: "button + icon-button. Triggers — primary CTAs, secondary, ghost, destructive. Disabled-with-reason rule, loading state pairing."
---
# Actions group

> Anchor: `#components-actions` in `detail.html`.
> Source-of-truth: `ds.json.components.{button, icon-button}`.

**Definition.** Components whose primary purpose is to **trigger something** — a navigation, a save, a destructive operation. Actions sit at the start of every interaction loop.

## Components

### button

Main interactive element. Four variants cover the entire CTA spectrum from primary to destructive.

```json
"button": {
  "variants": {
    "primary":     { "background": "brand.primary", "text": "text.inverse",  "border": "none",                            "note": "Main CTA — page-level affordance." },
    "secondary":   { "background": "transparent",   "text": "brand.primary", "border": "1px solid brand.primary",         "note": "Same-context alternative; outlined." },
    "ghost":       { "background": "transparent",   "text": "text.primary",  "border": "none",                            "note": "Tertiary — toolbars, dense lists." },
    "danger":      { "background": "feedback.error","text": "text.inverse",  "border": "none",                            "note": "Destructive — delete, revoke." }
  },
  "sizes": {
    "sm": { "height": "36px", "fontSize": "bodySmall", "padding": "0 12px", "radius": "lg", "iconSize": "16px" },
    "md": { "height": "44px", "fontSize": "label",     "padding": "0 20px", "radius": "lg", "iconSize": "18px", "note": "Default — meets 44px tap target." },
    "lg": { "height": "52px", "fontSize": "body",      "padding": "0 24px", "radius": "lg", "iconSize": "20px" }
  },
  "states": ["idle", "hover", "focus", "pressed", "disabled", "loading"],
  "stateRules": {
    "hover":    "primary darkens one step (700→800); secondary fills with brand.50; ghost fills with surfaceContainer.",
    "focus":    "2px outline brand.500, 2px offset. Never remove — keyboard users rely on it.",
    "pressed":  "primary 900; secondary/ghost brand.100 background.",
    "disabled": "40% opacity; cursor: not-allowed; pointer-events: none.",
    "loading":  "Replace label with spinner; preserve button width to prevent layout shift."
  }
}
```

**Variant choice rule of thumb:** if you find yourself writing a fifth variant, ask whether the new one is really `secondary` with a different color — usually it is. The four names map to four intents, not four colors.

### icon-button

Square button whose only content is an icon. Different shape (square, equal padding) from button — keeps icon-only affordances from being mistaken for plain icons.

```json
"icon-button": {
  "variants": {
    "default": { "background": "transparent",       "text": "text.primary", "border": "none",                "note": "Toolbars, list-row trailing actions." },
    "filled":  { "background": "surfaceContainer",  "text": "text.primary", "border": "none",                "note": "When icon-only is page-level (not toolbar)." },
    "primary": { "background": "brand.primary",     "text": "text.inverse", "border": "none",                "note": "Floating action, mobile primary." },
    "danger":  { "background": "transparent",       "text": "feedback.error","border": "none",               "note": "Destructive icon — delete, dismiss." }
  },
  "sizes": {
    "sm": { "size": "32px", "iconSize": "16px", "radius": "md" },
    "md": { "size": "40px", "iconSize": "20px", "radius": "md", "note": "Default — meets 40px tap target with surrounding hit area." },
    "lg": { "size": "48px", "iconSize": "24px", "radius": "lg" }
  },
  "states": ["idle", "hover", "focus", "pressed", "disabled"],
  "stateRules": {
    "hover":    "default/danger fill with surfaceContainer; filled darkens one step.",
    "focus":    "Same 2px outline as button — never remove.",
    "pressed":  "Background darkens one further step.",
    "disabled": "40% opacity; pointer-events: none."
  }
}
```

**Accessibility note:** icon-buttons MUST have an accessible name (`aria-label`). The icon itself is decorative — screen readers announce the label, not the icon.

## Group-level rules

- **Action ≠ Trigger that toggles state.** A switch is in Forms, not Actions, because its purpose is to capture a binary value, not to trigger an effect.
- **Disabled actions show why.** Button stays in DOM (so layout doesn't jump) but reveals reason on hover (`title` attribute or tooltip).
- **Loading is part of action lifecycle.** Every button used for async work needs a `loading` state — without one, prototypes invent ad-hoc spinners and the visual language fragments.
