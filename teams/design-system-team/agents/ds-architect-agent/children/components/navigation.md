---
knowledge-base-summary: "tabs, breadcrumb, pagination, menu, sidebar-nav, stepper, link, back-button, bottom-navigation. Move between contexts. Active-state mandatory, keyboard non-negotiable."
---
# Navigation group

> Anchor: `#components-navigation` in `detail.html`.
> Source-of-truth: `ds.json.components.{tabs, breadcrumb, pagination, menu, sidebar-nav, stepper, link, back-button, bottom-navigation}`.

**Definition.** Components that **move the user between contexts** — between sibling sections, between hierarchical levels, between sequential steps, between paginated chunks of the same dataset.

## Components

### tabs

Sibling sections of the same parent. The tab strip is the navigation; the content below is what changes.

```json
"tabs": {
  "variants": {
    "underline": { "indicator": "underline 2px brand.primary", "padding": "10px 16px", "note": "Default — desktop default look." },
    "pill":      { "indicator": "filled brand.primary background", "padding": "6px 14px", "radius": "full", "note": "Compact toolbars; mobile." },
    "segmented": { "background": "surfaceContainer", "activeBackground": "surface", "padding": "6px 14px", "radius": "lg", "note": "iOS-style — equal-width." }
  },
  "sizes": {
    "sm": { "fontSize": "bodySmall", "tabHeight": "32px" },
    "md": { "fontSize": "label",     "tabHeight": "40px", "note": "Default." }
  },
  "states": ["idle", "hover", "active", "disabled", "focus"],
  "stateRules": {
    "active":   "underline: indicator visible, text becomes brand.primary; pill: background filled; segmented: background surface, others inactive.",
    "hover":    "Inactive tabs lighten background; active tab unchanged.",
    "focus":    "2px outline brand.500, 2px offset around the tab box.",
    "disabled": "40% opacity; cursor: not-allowed."
  },
  "overflowBehavior": "Horizontal scroll on mobile (no wrap); on desktop, overflow tabs collapse into a 'More ▼' menu."
}
```

### breadcrumb

Path back to ancestor pages. Used when hierarchy is more than 2 levels deep — for shallow trees, a back button suffices.

```json
"breadcrumb": {
  "variants": {
    "default": { "separator": "›", "separatorColor": "text.muted", "linkColor": "text.secondary", "currentColor": "text.primary" },
    "slash":   { "separator": "/", "separatorColor": "text.muted", "linkColor": "text.secondary", "currentColor": "text.primary" }
  },
  "sizes": {
    "sm": { "fontSize": "caption",   "gap": "6px" },
    "md": { "fontSize": "bodySmall", "gap": "8px", "note": "Default." }
  },
  "states": ["idle", "hover", "focus"],
  "stateRules": {
    "hover": "Link items underline; current (last) item never underlines — it's not navigable.",
    "focus": "2px outline brand.500, 2px offset on the link."
  },
  "overflowBehavior": "If 4+ levels: collapse middle to '…' that expands on click. Always show first and last."
}
```

### pagination

Numbered or cursor-based navigation through paginated content.

```json
"pagination": {
  "variants": {
    "numbered": { "showFirstLast": true, "showPrevNext": true, "ellipsis": "…", "note": "Default for finite, countable sets (search results, product grids)." },
    "cursor":   { "showFirstLast": false, "showPrevNext": true, "note": "Streaming / infinite-feed-like data — no total page count, just prev/next." },
    "compact":  { "showFirstLast": false, "showPrevNext": true, "showCount": "{current} of {total}", "note": "Mobile, dense toolbars." }
  },
  "sizes": {
    "sm": { "buttonSize": "32px", "fontSize": "bodySmall", "gap": "4px" },
    "md": { "buttonSize": "40px", "fontSize": "body",      "gap": "6px", "note": "Default." }
  },
  "states": ["idle", "hover", "active", "disabled", "focus"],
  "stateRules": {
    "active":   "Background: brand.primary; text: text.inverse.",
    "hover":    "Background: surfaceContainer.",
    "disabled": "Prev when on page 1; Next when on last page. 40% opacity, cursor: not-allowed.",
    "focus":    "2px outline brand.500, 2px offset around button."
  },
  "rule": "Numbered pagination ALWAYS pairs with a 'X–Y of Z' summary nearby. Cursor never shows numbered — they're mutually exclusive."
}
```

### menu

Floating list of actions, anchored to a trigger (icon-button, button, or text link). Distinct from select (which captures a value); menu invokes actions.

```json
"menu": {
  "variants": {
    "default": { "background": "surface",          "elevation": "medium", "border": "1px solid border" },
    "ghost":   { "background": "surface",          "elevation": "high",   "border": "none", "note": "Floating menus on light backgrounds — no border, more shadow." }
  },
  "sizes": {
    "sm": { "itemHeight": "32px", "fontSize": "bodySmall", "padding": "4px",  "minWidth": "160px", "radius": "md" },
    "md": { "itemHeight": "40px", "fontSize": "body",      "padding": "6px",  "minWidth": "200px", "radius": "lg", "note": "Default." }
  },
  "states": ["closed", "opening", "open", "closing"],
  "stateRules": {
    "opening": "decelerate normal — fade + scale-up from anchor (0.95 → 1).",
    "closing": "accelerate fast — fade + scale-down."
  },
  "itemTypes": {
    "action":    "Default — clickable. May have leading icon, trailing shortcut hint.",
    "separator": "Horizontal divider between groups.",
    "header":    "Group label — non-interactive uppercase caption.",
    "destructive": "Action variant tinted feedback.error — used for delete/revoke."
  },
  "anchorRule": "Menu opens flush with anchor edge (left-aligned by default, right-aligned if it would clip the viewport)."
}
```

### sidebar-nav

Vertical list of top-level destinations. Used in app shells (settings, admin dashboards, this very detail page).

```json
"sidebar-nav": {
  "variants": {
    "default":   { "background": "surface",          "border": "1px solid border", "width": "240px" },
    "compact":   { "background": "surface",          "border": "1px solid border", "width": "64px",  "note": "Icon-only — labels appear in tooltip on hover." },
    "ghost":     { "background": "transparent",      "border": "none",              "width": "240px", "note": "Embedded in a wider chrome — no own background." }
  },
  "sizes": {
    "default": { "itemHeight": "40px", "itemPadding": "0 12px", "iconSize": "20px", "fontSize": "label", "groupGap": "16px" }
  },
  "states": ["idle", "hover", "active", "focus", "disabled"],
  "stateRules": {
    "active":   "Background: brand.50; text: brand.700; (optional) 3px brand.primary border on the leading edge.",
    "hover":    "Background: surfaceContainer.",
    "focus":    "2px outline brand.500, 2px offset around item.",
    "disabled": "40% opacity; cursor: not-allowed."
  },
  "structure": "(optional) header (logo + product name) | nav items grouped by `ds-sidebar-label` | (optional) footer (settings, user, sign-out)"
}
```

### stepper

Sequential progress through a multi-step flow. Shows where the user is, where they came from, where they're going.

```json
"stepper": {
  "variants": {
    "horizontal": { "orientation": "horizontal", "note": "Multi-step forms on desktop." },
    "vertical":   { "orientation": "vertical",   "note": "Mobile flows, dense flows with long step labels." }
  },
  "sizes": {
    "sm": { "circleSize": "24px", "fontSize": "caption",   "gap": "32px" },
    "md": { "circleSize": "32px", "fontSize": "bodySmall", "gap": "48px", "note": "Default." }
  },
  "states": ["pending", "current", "completed", "error"],
  "stateRules": {
    "pending":   "Circle: 1px border, surfaceContainer fill; number visible inside.",
    "current":   "Circle: brand.primary fill, white number; label below: text.primary, weight 600.",
    "completed": "Circle: brand.primary fill, check icon (no number); connector to next: brand.primary.",
    "error":     "Circle: feedback.error fill, alert icon; label: feedback.error."
  },
  "connectorRule": "Connectors between steps reflect the state BEFORE the next circle. Connector from step 2 to step 3 is filled if step 2 is completed."
}
```

### link

Generic link primitive. Distinct from button (button triggers an action; link navigates).

```json
"link": {
  "variants": {
    "default":  { "color": "brand.primary",   "textDecoration": "none",      "hoverDecoration": "underline", "note": "Standard inline link — underline on hover." },
    "inline":   { "color": "brand.primary",   "textDecoration": "underline",                                  "note": "Long-form prose — always underlined for discoverability." },
    "nav":      { "color": "text.primary",    "textDecoration": "none",                                       "note": "App nav / menu links — no underline; rely on context for affordance." },
    "external": { "color": "brand.primary",   "textDecoration": "none",      "hoverDecoration": "underline", "trailingIcon": "arrow-up-right", "iconSize": "12px", "note": "External link — adds ↗ icon and target=\"_blank\" rel=\"noopener\"." }
  },
  "sizes": {
    "sm": { "fontSize": "caption" },
    "md": { "fontSize": "body",       "note": "Default." },
    "lg": { "fontSize": "bodyLarge" }
  },
  "states": ["idle", "hover", "focus", "active", "visited"],
  "stateRules": {
    "hover":   "Per-variant: see hoverDecoration above. Color stays the same.",
    "focus":   "2px outline brand.500, 2px offset; underline appears for default/nav variants.",
    "active":  "Color: brand.700 (slightly darker).",
    "visited": "Color: text.secondary; reset on click for SPA routes (declare `resetVisited: true`)."
  },
  "semanticsRule": "Always uses semantic <a> element with href. Never use <button> styled as link or <span onClick> — keyboard + screen-reader users need real link semantics."
}
```

### back-button

Back navigation primitive — chevron-left + optional label.

```json
"back-button": {
  "variants": {
    "icon":    { "showLabel": false, "note": "Chevron-only — toolbar / mobile header." },
    "labeled": { "showLabel": true,  "label": "Back", "note": "Chevron + 'Back' label — desktop / web app." },
    "context": { "showLabel": true,  "labelSource": "router.previousRoute.title", "note": "Chevron + previous page name (e.g., '← Profile')." }
  },
  "sizes": {
    "sm": { "size": "32px", "iconSize": "16px", "fontSize": "bodySmall", "padding": "0 8px",  "radius": "md" },
    "md": { "size": "40px", "iconSize": "20px", "fontSize": "body",      "padding": "0 12px", "radius": "md", "note": "Default — meets 40px tap target." },
    "lg": { "size": "48px", "iconSize": "24px", "fontSize": "body",      "padding": "0 16px", "radius": "lg" }
  },
  "states": ["idle", "hover", "focus", "pressed"],
  "stateRules": {
    "hover":   "Background: surfaceContainer.",
    "focus":   "2px outline brand.500, 2px offset.",
    "pressed": "Background: surfaceContainerHigh."
  },
  "structure": "icon (chevron-left, leading edge, fixed) + (optional) label",
  "behavior": "Triggers router.back() or equivalent · context variant requires router state with previous route info; falls back to 'Back' if unavailable."
}
```

### bottom-navigation

Mobile bottom nav bar. Always pinned to viewport bottom.

```json
"bottom-navigation": {
  "variants": {
    "icon-only": { "showLabel": false, "itemCount": "3-5", "note": "Icon-only — tighter; tooltips on long-press." },
    "labeled":   { "showLabel": true,  "itemCount": "3-5", "note": "Default — icon + label below; clearer for first-time users." },
    "floating":  { "showLabel": true,  "itemCount": "3-5", "withFAB": true,    "note": "Adds a centered floating action button cutting through the bar (Material-style)." }
  },
  "sizes": {
    "default": { "height": "64px", "iconSize": "24px", "fontSize": "10px", "labelMarginTop": "4px", "note": "Plus safe-area-inset-bottom for iOS." }
  },
  "states": ["idle", "active", "focus", "disabled"],
  "stateRules": {
    "active":   "Icon + label: brand.primary. Optional 4px brand.primary indicator above icon.",
    "focus":    "2px outline brand.500, 2px offset around the item box.",
    "disabled": "40% opacity; pointer-events: none."
  },
  "structure": "Top border (1px border) | flex row of N items (icon + optional label) | safe-area-inset-bottom padding",
  "behavior": "Always 3-5 items max — overflow → drawer (More menu). Hide on scroll-down, show on scroll-up (optional behavior). Floating variant: FAB sits 16px above bar's top edge."
}
```

## Group-level rules

- **Pick one navigation primitive per level.** Don't mix tabs and pagination at the same level — that's two competing notions of "where am I."
- **Active state is mandatory.** Every navigation component must visibly mark the current selection. Without it, navigation is just a list.
- **Keyboard navigation is non-negotiable.** Tabs respond to ←/→; menus to ↑/↓ + Enter; sidebar nav to ↑/↓ + Tab. The DS contract: every component listed here is keyboard-operable.
- **Mobile vs desktop.** Tabs collapse to overflow menu; sidebar-nav becomes drawer (Overlays group); breadcrumb collapses middle with `…`. These responsive behaviors live on each component, not invented per page.
