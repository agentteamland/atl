---
knowledge-base-summary: "avatar, badge, chip, card, divider, table, list, tree, rating, empty-state, accordion, stat, code, kbd. Present content. Avatar fallback ladder, table-vs-list rule, density tiers, KPI metrics."
---
# Display group

> Anchor: `#components-display` in `detail.html`.
> Source-of-truth: `ds.json.components.{avatar, badge, chip, card, divider, table, list, tree, rating, empty-state, accordion, stat, code, kbd}`.

**Definition.** Components that **present content** — the visual building blocks of any view. Not interactive primitives (those are Actions / Forms); not floating shells (those are Overlays); not feedback signals (those are Feedback). Just structured ways to show what exists.

## Components

### avatar

Person or entity profile picture, fallback initials, or fallback icon.

```json
"avatar": {
  "variants": {
    "image":    { "background": "surfaceContainer", "note": "Standard — image fills the circle." },
    "initials": { "background": "brand.100",        "text": "brand.800", "note": "Fallback when no image — first letter of name." },
    "icon":     { "background": "surfaceContainer", "iconColor": "text.secondary", "note": "Fallback when neither image nor name (e.g., archived or anonymous user)." },
    "group":    { "background": "surface",          "stackOffset": "-12px", "maxVisible": 3, "note": "Cluster of avatars; surplus shows '+N'." }
  },
  "sizes": {
    "xs": { "size": "20px", "fontSize": "10px",       "iconSize": "12px", "borderWidth": "1px" },
    "sm": { "size": "28px", "fontSize": "caption",    "iconSize": "16px", "borderWidth": "1.5px" },
    "md": { "size": "40px", "fontSize": "bodySmall",  "iconSize": "20px", "borderWidth": "2px",   "note": "Default." },
    "lg": { "size": "56px", "fontSize": "label",      "iconSize": "24px", "borderWidth": "2px" },
    "xl": { "size": "80px", "fontSize": "title",      "iconSize": "32px", "borderWidth": "3px" }
  },
  "states": ["idle"],
  "shape": "circle (default) | rounded-square (Discord/Slack-style — declare via `shape: 'rounded'`)",
  "presenceIndicator": "Optional 8px dot at bottom-right corner; tokens: feedback.success (online), neutral.400 (offline), feedback.warning (away)."
}
```

### badge

Small status indicator. Distinct from chip (chip is interactive — filter/select; badge is non-interactive — count, status).

```json
"badge": {
  "variants": {
    "neutral": { "background": "surfaceContainer", "text": "text.primary",   "note": "Default — counts, generic tags." },
    "brand":   { "background": "brand.100",        "text": "brand.800",      "note": "On-brand emphasis — 'New', 'Beta'." },
    "success": { "background": "feedback.success.surface", "text": "feedback.success", "note": "Confirmed status — 'Active', 'Paid'." },
    "warning": { "background": "feedback.warning.surface", "text": "feedback.warning", "note": "Heads-up — 'Pending', 'Overdue'." },
    "error":   { "background": "feedback.error.surface",   "text": "feedback.error",   "note": "Negative status — 'Failed', 'Blocked'." },
    "dot":     { "size": "8px", "background": "feedback.error", "note": "Pure indicator — no text. Notification dots." }
  },
  "sizes": {
    "sm": { "height": "16px", "fontSize": "10px",     "padding": "0 6px",  "radius": "full" },
    "md": { "height": "20px", "fontSize": "caption",  "padding": "0 8px",  "radius": "full", "note": "Default." }
  },
  "states": ["idle"],
  "placementRule": "Often anchored to another element (avatar with notification dot, button with count). Anchor offset: 25% inset from top-right corner of the parent."
}
```

### chip

Interactive tag. Used for filters, selectable categories, removable inputs.

```json
"chip": {
  "variants": {
    "filled":   { "background": "brand.primary",   "text": "text.inverse",  "note": "Selected — primary category." },
    "outlined": { "background": "transparent",     "border": "1px solid border", "text": "text.primary", "note": "Filter chips, unselected state." },
    "ghost":    { "background": "surfaceContainer","text": "text.primary",  "note": "Neutral tag — metadata, status pills." },
    "brand":    { "background": "brand.100",       "text": "brand.800",     "note": "Branded tag — category labels." }
  },
  "sizes": {
    "sm": { "height": "24px", "fontSize": "caption",   "padding": "0 8px",  "radius": "full" },
    "md": { "height": "32px", "fontSize": "bodySmall", "padding": "0 12px", "radius": "full" }
  },
  "states": ["idle", "hover", "selected", "disabled"],
  "stateRules": {
    "hover":    "Outlined → fills with surfaceContainer; ghost → fills with brand.50.",
    "selected": "Switches to `filled` variant treatment.",
    "disabled": "40% opacity; cursor: not-allowed."
  },
  "leadingIcon": "Optional — 16px icon at the start, 6px gap.",
  "trailingClose": "Optional × icon — used when chip is removable (input chips, applied filters)."
}
```

### card

Container for grouped content. The most-used surface in a typical product UI.

```json
"card": {
  "variants": {
    "elevated":    { "background": "surface",          "elevation": "low",  "border": "none",                  "note": "Default — contentful surfaces on list screens." },
    "outlined":    { "background": "surface",          "elevation": "none", "border": "1px solid border",      "note": "Dense layouts — no shadow noise." },
    "filled":      { "background": "surfaceContainer", "elevation": "none", "border": "none",                  "note": "Subtle grouping inside a larger card or sheet." },
    "interactive": { "background": "surface",          "elevation": "low",  "border": "none", "hover": "elevation.medium + translateY(-1px)", "note": "Tappable list items." }
  },
  "sizes": {
    "compact":  { "padding": "12px", "radius": "lg", "gap": "8px",  "note": "Dense list rows, drawer items." },
    "default":  { "padding": "20px", "radius": "xl", "gap": "12px", "note": "Standard card on list and detail screens." },
    "spacious": { "padding": "28px", "radius": "xl", "gap": "16px", "note": "Hero / featured cards on marketing pages." }
  },
  "states": ["idle", "hover", "pressed", "focus"],
  "stateRules": {
    "hover":   "interactive variant only — elevation.medium + translateY(-1px); 200ms standard easing.",
    "pressed": "interactive variant only — elevation.low + translateY(0); 120ms accelerate.",
    "focus":   "2px outline brand.500, 2px offset; never remove."
  },
  "structure": "(optional) header | body | (optional) actions row (right-aligned)"
}
```

### divider

Visible separation between regions.

```json
"divider": {
  "variants": {
    "default":  { "color": "border",              "thickness": "1px",   "note": "Standard horizontal rule." },
    "muted":    { "color": "border",              "thickness": "1px", "opacity": "0.5", "note": "Softer — between sub-groups within a group." },
    "labeled":  { "color": "border",              "thickness": "1px", "labelBackground": "surface", "labelPadding": "0 12px", "note": "Section break with center text label." }
  },
  "sizes": {
    "default": { "marginY": "16px", "note": "Vertical rhythm." },
    "tight":   { "marginY": "8px" },
    "loose":   { "marginY": "32px" }
  },
  "states": ["idle"],
  "orientation": "horizontal (default) | vertical (declare `orientation: 'vertical'`)"
}
```

### table

Tabular dataset with rows and columns. Heavyweight component — only used when the data is genuinely tabular (sortable, filterable, comparable across rows).

```json
"table": {
  "variants": {
    "default": { "borderStyle": "row-only",    "headerBg": "surfaceContainer", "rowHover": "surface alt 1%", "note": "Standard — only horizontal row dividers." },
    "bordered":{ "borderStyle": "all",         "headerBg": "surfaceContainer", "rowHover": "surface alt 1%", "note": "Cell borders — accounting / reports / dense data." },
    "striped": { "borderStyle": "none",        "headerBg": "surfaceContainer", "stripeBg":  "surface alt 1%", "note": "Zebra rows — long lists." }
  },
  "sizes": {
    "sm": { "rowHeight": "36px", "cellPadding": "8px 12px",   "fontSize": "bodySmall" },
    "md": { "rowHeight": "48px", "cellPadding": "12px 16px",  "fontSize": "body", "note": "Default." },
    "lg": { "rowHeight": "60px", "cellPadding": "16px 20px",  "fontSize": "body" }
  },
  "states": ["idle", "hover", "selected", "sorted"],
  "stateRules": {
    "hover":    "Row background: surface alt 2% (slightly darker than zebra stripe).",
    "selected": "Row background: brand.50; leading-edge 3px brand.primary border (when row is the selected one in master-detail).",
    "sorted":   "Header column: arrow icon visible (↑ asc / ↓ desc); column itself: subtle brand.50 background."
  },
  "features": {
    "sortableColumns":    "Click header to sort; toggle asc/desc/unsorted.",
    "rowSelection":       "Optional checkbox column at start; pairs with bulk-action toolbar above table.",
    "stickyHeader":       "Header row sticks to top of scroll container.",
    "stickyFirstColumn":  "Optional — first column sticks to left when scrolling horizontally.",
    "emptyState":         "Always pair with empty-state component when zero rows."
  }
}
```

### list

Vertical sequence of similar items. Lighter than table — used when items aren't comparable across columns.

```json
"list": {
  "variants": {
    "default":     { "rowBackground": "surface",          "divider": "1px solid border", "note": "Standard list with row dividers." },
    "card":        { "rowBackground": "surface",          "divider": "none", "rowGap": "12px", "rowRadius": "lg", "rowElevation": "low", "note": "Each row is a mini-card — looser." },
    "ghost":       { "rowBackground": "transparent",      "divider": "none", "rowGap": "0", "note": "Stripped list inside a card or sheet." }
  },
  "sizes": {
    "sm": { "rowHeight": "44px", "padding": "8px 12px",  "fontSize": "bodySmall" },
    "md": { "rowHeight": "56px", "padding": "12px 16px", "fontSize": "body", "note": "Default." },
    "lg": { "rowHeight": "72px", "padding": "16px 20px", "fontSize": "body" }
  },
  "states": ["idle", "hover", "selected", "focus"],
  "stateRules": {
    "hover":    "Row background: surfaceContainer.",
    "selected": "Row background: brand.50; leading-edge 3px brand.primary border.",
    "focus":    "2px outline brand.500, 2px offset around row."
  },
  "rowStructure": "(optional) leading visual (avatar / icon / chevron) | primary text | (optional) secondary text | (optional) trailing visual (badge / meta / icon-button)"
}
```

### tree

Hierarchical list. Used for file trees, category trees, organizational hierarchies. Heavier than list — only when nesting matters.

```json
"tree": {
  "variants": {
    "default":   { "indent": "20px", "connector": "none",          "note": "Standard — indentation alone shows depth." },
    "connected": { "indent": "20px", "connector": "1px solid border", "note": "Vertical guide lines between siblings." }
  },
  "sizes": {
    "sm": { "rowHeight": "32px", "fontSize": "bodySmall", "iconSize": "14px" },
    "md": { "rowHeight": "40px", "fontSize": "body",      "iconSize": "18px", "note": "Default." }
  },
  "states": ["idle", "hover", "selected", "expanded", "collapsed", "focus"],
  "stateRules": {
    "expanded":  "Chevron rotated 90° (→ ▼); children visible.",
    "collapsed": "Chevron pointing right (→ ▶); children hidden.",
    "selected":  "Row background: brand.50; leading-edge 3px brand.primary border.",
    "hover":     "Row background: surfaceContainer.",
    "focus":     "2px outline brand.500, 2px offset around row."
  },
  "rowStructure": "chevron (or spacer for leaf) | (optional) icon | label | (optional) trailing meta"
}
```

### rating

Star (or other shape) rating display and input.

```json
"rating": {
  "variants": {
    "stars":  { "shape": "star",   "filledColor": "feedback.warning", "emptyColor": "neutral.300", "note": "Default — most familiar." },
    "hearts": { "shape": "heart",  "filledColor": "feedback.error",   "emptyColor": "neutral.300", "note": "Like / favorite-style ratings." },
    "dots":   { "shape": "circle", "filledColor": "brand.primary",    "emptyColor": "neutral.300", "note": "Compact — when 5 stars feels too loud." }
  },
  "sizes": {
    "sm": { "iconSize": "14px", "gap": "2px" },
    "md": { "iconSize": "20px", "gap": "4px", "note": "Default." },
    "lg": { "iconSize": "28px", "gap": "6px" }
  },
  "states": ["readonly", "interactive", "hover", "disabled"],
  "stateRules": {
    "interactive": "Click sets the rating; hover previews the rating with darker filled icons.",
    "readonly":    "Pure display — no pointer cursor, no hover preview.",
    "hover":       "Filled icons up to the hovered position; user feedback before clicking.",
    "disabled":    "40% opacity; cursor: not-allowed."
  },
  "halfStarRule": "If allowing fractional ratings, declare `step: 0.5` and render half-filled icons. Default is integer-only."
}
```

### empty-state

Surface shown when a region has no content. Distinct from skeleton (loading) and error (broken) — empty-state describes a working, but empty, region.

```json
"empty-state": {
  "variants": {
    "default":  { "background": "transparent", "iconBgColor": "brand.50", "iconColor": "brand.primary", "note": "Inline empty — center of a list/grid." },
    "fullPage": { "background": "transparent", "iconBgColor": "brand.50", "iconColor": "brand.primary", "minHeight": "60vh", "note": "Whole-page empty — first-run, blank dashboard." }
  },
  "sizes": {
    "sm": { "iconSize": "48px",  "padding": "24px",      "fontSize": "body" },
    "md": { "iconSize": "72px",  "padding": "40px 24px", "fontSize": "body", "note": "Default." },
    "lg": { "iconSize": "96px",  "padding": "64px 24px", "fontSize": "body" }
  },
  "states": ["idle"],
  "structure": "icon badge | title (heading) | body (1–2 sentences explaining why empty + what to do) | (optional) primary CTA",
  "copyRefs": "Pulls default copy from voice.samples.empty.* — composer's job is to swap to context-specific copy when the empty case has more nuance (e.g., 'no search results' vs 'first-time user')."
}
```

### accordion

Toggleable collapsible sections. Each item has a header (always visible) and a body (revealed on expand).

```json
"accordion": {
  "variants": {
    "single":   { "openMode": "single",   "border": "none",                 "note": "Default — one section open at a time; opening another closes the previous." },
    "multiple": { "openMode": "multiple", "border": "none",                 "note": "Any number of sections can be open simultaneously." },
    "bordered": { "openMode": "single",   "border": "1px solid border",    "note": "Each section wrapped in its own bordered card — used in dense docs / long FAQ pages." }
  },
  "sizes": {
    "sm": { "headerHeight": "40px", "fontSize": "bodySmall", "padding": "10px 12px" },
    "md": { "headerHeight": "48px", "fontSize": "body",      "padding": "12px 16px", "note": "Default." },
    "lg": { "headerHeight": "56px", "fontSize": "body",      "padding": "16px 20px" }
  },
  "states": ["collapsed", "expanded", "hover", "focus", "disabled"],
  "stateRules": {
    "expanded": "Chevron rotates 180° (▼); body slides down (decelerate, normal duration).",
    "hover":    "Header background: surfaceContainer.",
    "focus":    "2px outline brand.500, 2px offset around header.",
    "disabled": "40% opacity; cursor: not-allowed; click does nothing."
  },
  "structure": "header (chevron + label + optional trailing meta) | body (markdown / arbitrary content, padded)",
  "behavior": "Keyboard ↑/↓ moves focus between headers · Enter / Space toggles expanded · `prefers-reduced-motion` disables slide animation."
}
```

### stat

KPI / metric card. Numeric value + label, with optional delta indicator and / or sparkline.

```json
"stat": {
  "variants": {
    "default":       { "showDelta": false, "showSparkline": false, "note": "Label + large value — simplest form." },
    "withDelta":     { "showDelta": true,  "showSparkline": false, "note": "Adds delta arrow + percentage (vs. previous period)." },
    "withSparkline": { "showDelta": true,  "showSparkline": true,  "note": "Adds a 32-tall sparkline below the value (uses charts.sparkline component)." }
  },
  "sizes": {
    "sm": { "minHeight": "96px",  "valueSize": "title",      "labelSize": "caption",   "padding": "16px" },
    "md": { "minHeight": "128px", "valueSize": "headline",   "labelSize": "bodySmall", "padding": "20px", "note": "Default — dashboard staple." },
    "lg": { "minHeight": "160px", "valueSize": "display",    "labelSize": "body",      "padding": "24px" }
  },
  "states": ["idle", "loading"],
  "stateRules": {
    "loading": "Replace value + delta + sparkline with skeleton.line / skeleton.block matching their shapes.",
    "idle":    "Render the actual value, delta direction (↑/↓/→), color (success/error/neutral) per delta sign."
  },
  "deltaRules": {
    "positive": "color: feedback.success; arrow: ↑ ; format: '+N%' or '+N'.",
    "negative": "color: feedback.error;   arrow: ↓ ; format: '-N%' or '-N'.",
    "neutral":  "color: text.muted;        arrow: → ; format: 'N%'."
  }
}
```

### code

Inline code or block of code with optional syntax highlighting.

```json
"code": {
  "variants": {
    "inline":   { "background": "surfaceContainer", "border": "none",          "padding": "1px 6px",  "radius": "sm", "fontSize": "0.9em", "note": "<code> in flowing text — small surface highlight." },
    "block":    { "background": "neutral.900",      "border": "none",          "padding": "16px 20px","radius": "lg", "fontSize": "bodySmall", "color": "neutral.100", "note": "<pre><code> — multi-line, syntax-highlighted." },
    "filename": { "background": "neutral.900",      "border": "none",          "padding": "0",        "radius": "lg", "fontSize": "bodySmall", "color": "neutral.100", "note": "Block + top header bar showing filename + language pill + (optional) copy button." }
  },
  "sizes": {
    "sm": { "fontSize": "12px", "lineHeight": "1.5" },
    "md": { "fontSize": "14px", "lineHeight": "1.6", "note": "Default — block default size." }
  },
  "states": ["idle", "copied"],
  "stateRules": {
    "copied": "Copy button (top-right of block) shows check icon + 'Copied' for 2s, then reverts."
  },
  "syntaxThemeRule": "Block syntax theme follows the DS palette: keywords → brand.300, strings → feedback.warning, comments → neutral.500, types → brand.200. Override via `syntaxTheme` prop.",
  "structure": "filename variant: header bar (filename left + language pill right + copy icon) | code body | (optional) line numbers gutter"
}
```

### kbd

Keyboard shortcut display — renders as elevated key cap.

```json
"kbd": {
  "variants": {
    "default": { "background": "surface", "border": "1px solid border", "shadow": "0 1px 0 rgba(15,23,42,0.08)", "padding": "0 6px", "fontFamily": "mono", "note": "Single key — '⌘', 'Esc', 'A'." },
    "combo":   { "background": "surface", "border": "1px solid border", "shadow": "0 1px 0 rgba(15,23,42,0.08)", "padding": "0 6px", "fontFamily": "mono", "joinSeparator": "+", "note": "Multi-key combo — joins with '+' separator. '⌘ + K'." }
  },
  "sizes": {
    "sm": { "height": "18px", "fontSize": "10px",  "radius": "sm" },
    "md": { "height": "22px", "fontSize": "12px",  "radius": "sm", "note": "Default — paired with body text." },
    "lg": { "height": "26px", "fontSize": "13px",  "radius": "md" }
  },
  "states": ["idle", "pressed"],
  "stateRules": {
    "pressed": "Animation-only — for tutorial overlays. Translates 1px down, removes shadow."
  },
  "platformRule": "Declare `platform: 'auto' | 'mac' | 'win'`. 'auto' (default) reads navigator.userAgent. 'mac' shows ⌘/⌥/⇧/⌃; 'win' shows Ctrl/Alt/Shift.",
  "behavior": "Used inline within tooltips, command-menu items, button labels, callouts. Not interactive — pure display."
}
```

## Group-level rules

- **Display ≠ Interactive shell.** A card with a button inside is still display (the card is content; the button is action). A modal containing the same content is overlay (because the modal IS the chrome that floats above other content).
- **Pair empty/loading/error.** Every list, table, and tree should declare its empty-state and pull from skeleton + state-recipes. Without this, prototypes invent ad-hoc treatments.
- **Avatar fallback ladder.** image → initials → icon. Never blank circle. Always declare which fallback applies.
- **Table vs list.** If the data is comparable across columns (sort, filter, scan a column), use table. If the data is one item per row, use list.
- **Density tiers (sm / md / lg).** Always declare all three for table/list/tree even if md is the only one used in the current product — prototype-agent picks the right density per context.
