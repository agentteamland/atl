---
knowledge-base-summary: "modal, drawer, sheet, popover, dropdown, command-menu, date-picker, carousel, confirm-dialog. Float above the page. One-overlay-at-a-time, scrim semantics, mobile pivots."
---
# Overlays group

> Anchor: `#components-overlays` in `detail.html`.
> Source-of-truth: `ds.json.components.{modal, drawer, sheet, popover, dropdown, command-menu, date-picker, carousel, confirm-dialog}`.

**Definition.** Components that **float above the page** — they sit on a higher elevation layer (in z-axis terms), often with a scrim, and demand the user's attention until dismissed. Distinct from Feedback (which informs without blocking) and Display (which lives inline).

## Components

### modal

Centered, blocking dialog. Used for confirmations, focused tasks, content that demands attention.

```json
"modal": {
  "variants": {
    "default": { "background": "surface", "elevation": "high", "radius": "2xl", "scrim": "rgba(21, 25, 21, 0.48)", "note": "Standard." },
    "sheet":   { "background": "surface", "elevation": "high", "radius": "2xl 2xl 0 0", "fullWidth": true, "note": "Mobile bottom sheet." }
  },
  "sizes": {
    "sm":   { "maxWidth": "360px",                "padding": "20px", "note": "Confirmation dialogs." },
    "md":   { "maxWidth": "480px",                "padding": "24px", "note": "Default — single-input forms." },
    "lg":   { "maxWidth": "640px",                "padding": "32px", "note": "Multi-field forms, content-rich dialogs." },
    "xl":   { "maxWidth": "800px",                "padding": "32px", "note": "Multi-step flows, embedded media." },
    "full": { "maxWidth": "calc(100vw - 64px)",   "padding": "32px", "note": "Mobile takeover." }
  },
  "states": ["closed", "opening", "open", "closing"],
  "stateRules": {
    "opening": "Scrim fades in 200ms; dialog scales 0.96 → 1 + fades; decelerate easing, normal duration.",
    "closing": "Scrim fades out 160ms; dialog scales 1 → 0.96 + fades; accelerate easing, fast duration."
  },
  "behavior": {
    "scrimClick":  "Closes — unless `dismissible: false` (e.g., destructive confirm).",
    "escapeKey":   "Closes — same condition.",
    "focusTrap":   "Tab cycles within modal; first interactive element receives focus on open.",
    "scrollLock":  "Body scroll locked while open."
  }
}
```

### drawer

Side-anchored panel. Used for secondary navigation (mobile sidebar), filters, contextual sub-views.

```json
"drawer": {
  "variants": {
    "left":   { "anchor": "left",   "scrim": "rgba(21,25,21,0.48)", "elevation": "high", "note": "Mobile primary nav." },
    "right":  { "anchor": "right",  "scrim": "rgba(21,25,21,0.48)", "elevation": "high", "note": "Filters, contextual details." },
    "bottom": { "anchor": "bottom", "scrim": "rgba(21,25,21,0.48)", "elevation": "high", "radius": "2xl 2xl 0 0", "note": "Mobile alternative to modal — bottom sheet." }
  },
  "sizes": {
    "sm":   { "width": "280px", "padding": "16px" },
    "md":   { "width": "360px", "padding": "20px", "note": "Default." },
    "lg":   { "width": "480px", "padding": "24px" },
    "full": { "width": "100%",  "padding": "24px", "note": "Mobile full-takeover." }
  },
  "states": ["closed", "opening", "open", "closing"],
  "stateRules": {
    "opening": "Slide in from anchor edge (decelerate, normal); scrim fades in.",
    "closing": "Slide out to anchor edge (accelerate, fast); scrim fades out."
  },
  "behavior": "Same scrim/escape/focus rules as modal. Drawer doesn't trap focus when bottom-anchored on tablets where it doesn't fully cover the screen."
}
```

### sheet

Bottom-anchored persistent panel. Distinct from drawer.bottom — sheet is meant to coexist with content (peek state), drawer.bottom is fully blocking.

```json
"sheet": {
  "variants": {
    "default": { "background": "surface", "elevation": "medium", "radius": "2xl 2xl 0 0", "border": "1px solid border", "note": "Persistent — peek + expand." }
  },
  "sizes": {
    "default": { "peekHeight": "120px", "expandedHeight": "60vh", "fullHeight": "calc(100vh - 64px)" }
  },
  "states": ["peek", "expanded", "full", "dragging"],
  "stateRules": {
    "peek":     "Initial state — handle visible, content peeks.",
    "expanded": "Mid state — half-screen, content visible with scroll.",
    "full":     "Top state — covers viewport less top inset.",
    "dragging": "Real-time follows finger; snap to nearest state on release."
  },
  "behavior": "Drag handle (32px wide bar) at top center; tap to expand, drag to set height, swipe down to dismiss to peek/closed."
}
```

### popover

Floating panel anchored to a trigger. Used for inline editing, contextual filters, info popups (richer than tooltip — can contain interactive content).

```json
"popover": {
  "variants": {
    "default": { "background": "surface", "elevation": "medium", "border": "1px solid border", "radius": "lg" },
    "ghost":   { "background": "surface", "elevation": "high",   "border": "none",              "radius": "lg", "note": "Floating on light backgrounds — no border, more shadow." }
  },
  "sizes": {
    "sm": { "padding": "12px", "fontSize": "bodySmall", "minWidth": "200px", "maxWidth": "280px" },
    "md": { "padding": "16px", "fontSize": "body",      "minWidth": "240px", "maxWidth": "360px", "note": "Default." },
    "lg": { "padding": "20px", "fontSize": "body",      "minWidth": "320px", "maxWidth": "480px" }
  },
  "states": ["closed", "opening", "open", "closing"],
  "stateRules": {
    "opening": "Fade + scale (0.95 → 1) from anchor; decelerate, normal.",
    "closing": "Fade + scale (1 → 0.95); accelerate, fast."
  },
  "behavior": {
    "trigger":     "click | hover | focus — declared per use",
    "anchor":      "below trigger by default; flips to above if clipped",
    "arrow":       "Optional 8px triangle pointing at trigger",
    "dismissOn":   "click outside | escape | scroll outside (configurable)",
    "noFocusTrap": "Popovers DO NOT trap focus (unlike modal). User can tab in/out freely."
  }
}
```

### dropdown

Action menu triggered by a button. Architecturally a popover anchored below a trigger, with menu styling — kept as a separate spec because the trigger ↔ menu pairing is the contract, not just the floating panel.

```json
"dropdown": {
  "variants": {
    "default": { "triggerStyle": "button.secondary",   "menuStyle": "menu.default", "note": "Primary — used in toolbars." },
    "icon":    { "triggerStyle": "icon-button.default","menuStyle": "menu.ghost",   "note": "Compact — overflow menus, row-trailing actions." },
    "split":   { "triggerStyle": "button.primary + icon-button.primary chevron", "menuStyle": "menu.default", "note": "Primary action + alternative actions in dropdown." }
  },
  "sizes": {
    "sm": { "minWidth": "160px" },
    "md": { "minWidth": "200px", "note": "Default." }
  },
  "states": ["closed", "open", "focus"],
  "stateRules": {
    "open":  "Trigger gains pressed visual; chevron rotates 180°; menu opens.",
    "focus": "Standard trigger focus outline."
  },
  "behavior": "Tab cycles between trigger and menu items when open. Escape closes. Click outside closes."
}
```

### command-menu

Keyboard-first action launcher (cmd+K / ctrl+K). Pulls together navigation, actions, and search into one fuzzy-matchable surface.

```json
"command-menu": {
  "variants": {
    "default": { "background": "surface", "elevation": "high", "radius": "2xl", "scrim": "rgba(21,25,21,0.48)", "note": "Center-of-viewport overlay." }
  },
  "sizes": {
    "default": { "width": "640px", "maxHeight": "60vh", "padding": "0", "note": "Fixed width regardless of viewport — keyboard users expect consistent location." }
  },
  "states": ["closed", "opening", "open", "closing", "searching"],
  "stateRules": {
    "opening":   "Center fade + scale (0.95 → 1); decelerate, normal.",
    "searching": "Input shows debounce indicator (subtle right-side spinner) when filtering large command sets.",
    "closing":   "Fade + scale (1 → 0.95); accelerate, fast."
  },
  "structure": "search input (top, sticky) | result groups (Recents | Pages | Actions | …) | footer hint strip (↑↓ navigate, ↵ select, esc close)",
  "behavior": {
    "trigger":      "Cmd+K on macOS / Ctrl+K elsewhere — declared by app shell, never per-page.",
    "fuzzyMatch":   "Items match by command label, alias, or path; highlight matched chars in result.",
    "groupHeaders": "Headers (Recents, Pages, Actions) shown only when their group has matches.",
    "emptyState":   "When zero matches: 'No results for `{query}`. Try …' — copy from voice.samples.empty.searchEmpty."
  }
}
```

### date-picker

Calendar-style date selection. Single date, range, or multi-select.

```json
"date-picker": {
  "variants": {
    "single":    { "selectionMode": "single",    "note": "Default — one date." },
    "range":     { "selectionMode": "range",     "note": "Two dates — start + end (e.g., 'check-in / check-out')." },
    "multi":     { "selectionMode": "multi",     "note": "N dates — non-contiguous selection (e.g., availability slots)." }
  },
  "sizes": {
    "sm": { "cellSize": "32px", "fontSize": "bodySmall", "monthLabelSize": "label" },
    "md": { "cellSize": "40px", "fontSize": "body",      "monthLabelSize": "title", "note": "Default." }
  },
  "states": ["closed", "open", "selecting", "selected", "disabled"],
  "stateRules": {
    "open":      "Anchored as popover below trigger input; arrow keys navigate days, enter selects.",
    "selecting": "Range mode — first click sets start, hover previews end-of-range, second click commits.",
    "disabled":  "Days outside `minDate` / `maxDate` range; weekends if `disableWeekends`."
  },
  "structure": "month nav (prev | month-year label | next) | weekday header row | days grid | (optional) preset buttons (Today, Last 7 days, This month, …)",
  "behavior": {
    "anchor":      "Popover anchored to a `select`-styled trigger.",
    "rangePreview":"In range mode, the visual treatment between start and hovered end shows a brand-tinted background.",
    "minMax":      "`minDate` / `maxDate` props grey out unavailable days.",
    "presets":     "Optional left rail with quick presets — 'Today', 'Last 7 days', 'Last month' — for analytics-style use cases."
  }
}
```

### carousel

Horizontally scrollable rail of items with snap behavior. Used for image galleries, featured content, mobile lists where vertical space is precious.

```json
"carousel": {
  "variants": {
    "snap":     { "snap": "mandatory", "showArrows": true,  "showDots": true,  "note": "Default — discrete slides, one focal." },
    "freeflow": { "snap": "proximity", "showArrows": false, "showDots": false, "note": "Continuous scroll — tile rails on mobile (Instagram-like)." },
    "stack":    { "snap": "mandatory", "showArrows": true,  "showDots": true, "stackOffset": "8% peek of next slide", "note": "Tinder-style — current slide centered, neighbors peek." }
  },
  "sizes": {
    "default": { "slideGap": "16px", "arrowSize": "40px", "dotSize": "8px", "dotGap": "8px" }
  },
  "states": ["idle", "dragging", "transitioning"],
  "stateRules": {
    "dragging":      "Slides follow finger/cursor in real-time; momentum on release if free-flow.",
    "transitioning": "Snap target during easing — decelerate, normal duration."
  },
  "controls": {
    "arrows":     "Optional left/right icon-buttons over the rail edges; appear on hover (desktop), always visible (mobile).",
    "dots":       "Optional indicator dots below the rail; click jumps to slide.",
    "keyboard":   "Arrow keys move focus and snap target.",
    "autoPlay":   "Optional — must be pausable (hover or focus pauses)."
  }
}
```

### confirm-dialog

Confirmation primitive — specialized modal for "Are you sure?" prompts.

```json
"confirm-dialog": {
  "variants": {
    "default":     { "ctaVariant": "primary", "iconColor": "feedback.info",    "scrim": "rgba(21, 25, 21, 0.48)", "note": "Standard confirmation — informational." },
    "destructive": { "ctaVariant": "danger",  "iconColor": "feedback.warning", "scrim": "rgba(21, 25, 21, 0.48)", "note": "Destructive action — Delete, Revoke, Reset." },
    "critical":    { "ctaVariant": "danger",  "iconColor": "feedback.error",   "scrim": "rgba(21, 25, 21, 0.64)", "requireTypedConfirmation": true, "note": "High-stakes — requires user to type a phrase to enable Confirm." }
  },
  "sizes": {
    "sm": { "maxWidth": "360px", "padding": "20px", "note": "Quick confirmation." },
    "md": { "maxWidth": "480px", "padding": "24px", "note": "Default — supports a paragraph of context." }
  },
  "states": ["closed", "opening", "open", "closing", "confirming"],
  "stateRules": {
    "opening":    "Same as modal.opening (scrim fade in 200ms; dialog scale 0.96 → 1).",
    "closing":    "Same as modal.closing.",
    "confirming": "Confirm button shows loading state (spinner replacing label); Cancel button disabled."
  },
  "structure": "(optional) icon (top-center, 32px brand-tinted circle) | title (centered, bold) | body (1-2 sentences explaining consequence) | (critical only) input asking user to type confirmation phrase | Cancel + Confirm CTAs (right-aligned)",
  "behavior": {
    "scrimClick":  "Does NOT close — intentional friction. Only Cancel button or Escape closes.",
    "escapeKey":   "Closes (calls onCancel). For critical variant, Escape is also disabled.",
    "focusOnOpen": "Cancel button — safer default. User must explicitly Tab to Confirm.",
    "criticalTypedConfirmation": "Confirm button stays disabled until user types the phrase exactly (e.g., 'DELETE'). Phrase declared via `confirmationPhrase` prop."
  }
}
```

## Group-level rules

- **One overlay at a time.** Stacking modals, drawers on top of modals, popovers within popovers — these all confuse focus management. The DS rule: only one overlay open at any moment.
- **Scrim is meaningful.** Modal/drawer have a scrim because they BLOCK content. Popover/sheet/dropdown DO NOT — they coexist with the page beneath. Don't add a scrim to popover "for emphasis."
- **Mobile pivots.** Many overlays change shape on mobile: modal → drawer.bottom (mobile bottom sheet), dropdown → drawer.bottom (mobile action sheet), popover → bottom sheet (mobile contextual menu). Declare these pivots in the responsive section of the corresponding pattern, not per-page.
- **Focus trap is mandatory for blocking.** Modal, drawer (when full-takeover), command-menu — all trap focus. Popover, sheet (peek state), dropdown — do not.
- **Animation is responsive.** All overlays honor `prefers-reduced-motion` — opening transitions become instant fade only.
