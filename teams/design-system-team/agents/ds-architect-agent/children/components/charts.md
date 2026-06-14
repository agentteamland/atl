---
knowledge-base-summary: "chart, sparkline, gauge. Render data as visual shapes. Charts is the contract for code-side chart-library choice — palette, axis typography, tooltip styling are constraints the code agent honors. Empty + loading states mandatory; data-table fallback for accessibility."
---
# Charts group

> Anchor: `#components-charts` in `detail.html`.
> Source-of-truth: `ds.json.components.{chart, sparkline, gauge}`.

**Definition.** Components that **render data as visual shapes** — line / bar / pie / area / radial. Distinct from Display (which presents discrete content units like cards or rows); charts compress quantitative data into a glanceable visual.

Why a dedicated group: handoff. Without a Charts group declaration in the DS, the flutter-agent / react-agent receiving the prototype hand-off has no guidance on which chart library to use — recharts vs nivo vs chart.js becomes a per-screen decision and visual language fragments. With Charts in the DS, the chart palette + axis typography + tooltip styling are constraints the code agent honors.

## Components

### chart

Primary chart primitive. Variant declares the chart type; data shape is consistent across.

```json
"chart": {
  "variants": {
    "line": { "primaryColor": "brand.primary", "axisColor": "neutral.300", "gridColor": "neutral.100", "note": "Trend over a continuous axis (time, ordinal sequence)." },
    "bar":  { "primaryColor": "brand.primary", "axisColor": "neutral.300", "gridColor": "neutral.100", "note": "Comparison across discrete categories." },
    "pie":  { "palette": "brand.scale",         "ringWidth": "0",                                       "note": "Part-of-whole proportion (use sparingly — bar is usually clearer)." },
    "area": { "primaryColor": "brand.primary", "fillOpacity": "0.16", "axisColor": "neutral.300",      "note": "Trend with magnitude emphasis (cumulative-feeling)." }
  },
  "sizes": {
    "sm": { "minHeight": "200px", "axisFontSize": "10px", "marginX": "16px" },
    "md": { "minHeight": "320px", "axisFontSize": "11px", "marginX": "24px", "note": "Default — dashboard-tile-sized." },
    "lg": { "minHeight": "480px", "axisFontSize": "12px", "marginX": "32px" }
  },
  "states": ["idle", "loading", "hover", "empty", "error"],
  "stateRules": {
    "loading": "Replace chart body with skeleton.block matching height; preserve axes / labels.",
    "hover":   "Show tooltip (uses tooltip.default styling) anchored to nearest data point; cursor crosshair on line / area; row highlight on bar / pie segment.",
    "empty":   "Render emptyStateRecipe.empty (icon badge + 'No data for selected range' copy + optional 'Adjust filter' CTA).",
    "error":   "Render emptyStateRecipe.error variant with retry CTA; preserve chart frame so layout doesn't jump."
  },
  "tokens": {
    "tooltip":     "tooltip.default",
    "axisFont":    "typography.scale.caption",
    "legendFont":  "typography.scale.bodySmall",
    "emptyRecipe": "stateRecipes.empty"
  },
  "accessibilityRule": "Every chart must have a data-table fallback announced via aria-live (or accessible via 'View as table' link below). The visual is a complement, not a replacement."
}
```

### sparkline

Inline mini-chart. No axes, no labels — pure shape that fits inside a stat card or table cell.

```json
"sparkline": {
  "variants": {
    "line": { "color": "brand.primary", "lineWidth": "1.5px", "note": "Default — trend hint." },
    "bar":  { "color": "brand.primary", "barWidth": "auto",   "note": "Discrete comparison — small range of categories." },
    "area": { "color": "brand.primary", "fillOpacity": "0.16","note": "Trend with magnitude — adds shaded area under the line." }
  },
  "sizes": {
    "sm": { "height": "24px", "minWidth": "60px"  },
    "md": { "height": "32px", "minWidth": "80px",  "note": "Default — paired with stat.withSparkline." },
    "lg": { "height": "48px", "minWidth": "120px" }
  },
  "states": ["idle", "hover"],
  "stateRules": {
    "hover": "Show small cursor dot at hovered position; tooltip with `{x, y}` value 8px above (uses tooltip.sm)."
  },
  "compositionRule": "Width fills container by default. No padding, no margins — the shape is the entire visual.",
  "behavior": "Renders inline within stat / table cells / list rows · color from brand.primary by default; can override per-instance via `color` prop · keyboard interaction limited to scroll-into-view — sparklines are decorative-with-data, not primary affordances."
}
```

### gauge

Radial progress / KPI dial. Distinct from progress.circular (progress shows progress toward completion; gauge shows a value within a range).

```json
"gauge": {
  "variants": {
    "default": { "shape": "circle", "trackColor": "neutral.200", "fillColor": "brand.primary", "thickness": "8%",  "note": "Full circle (360°) — KPI dial." },
    "arc":     { "shape": "arc",    "trackColor": "neutral.200", "fillColor": "brand.primary", "thickness": "8%",  "sweep": "180°", "note": "Half-circle (180°) — speedometer-style; better for value with threshold zones." },
    "ring":    { "shape": "circle", "trackColor": "neutral.200", "fillColor": "brand.primary", "thickness": "12%", "centerLabel": false, "note": "Thin ring with no center label — used as small KPI accent." }
  },
  "sizes": {
    "sm": { "size": "80px",  "centerLabelSize": "label" },
    "md": { "size": "120px", "centerLabelSize": "title", "note": "Default." },
    "lg": { "size": "160px", "centerLabelSize": "headline" }
  },
  "states": ["idle", "loading", "animating"],
  "stateRules": {
    "loading":   "Replace fill with skeleton.circle matching size.",
    "animating": "On mount, fill animates from 0 to declared value (decelerate, slow). Re-renders animate from previous value to new (decelerate, normal)."
  },
  "colorByValueRule": "Optional `colorByValue` prop maps value ranges to colors (e.g., `[[0, 30, 'feedback.error'], [30, 70, 'feedback.warning'], [70, 100, 'feedback.success']]`). Without it, fill stays brand.primary regardless of value.",
  "structure": "track arc/circle (background) | filled arc/circle (foreground, sized by value/max) | (optional) center label (current value, formatted)"
}
```

## Group-level rules

- **Charts is the contract for code-side chart library choice.** When prototype-agent declares chart usage in a screen, the code agent (flutter-agent / react-agent) MUST honor the variant choice + palette + tooltip token references. The chart library is an implementation detail; the visual contract is in this group.
- **Empty + loading states are mandatory.** Every chart instance declares both — there is no "default" empty state worth showing for charts. Without a declared empty state, an empty-data chart renders as a confusing void.
- **Accessibility — data table fallback.** Every chart in production must offer a data-table view (toggleable via "View as table" link below) for screen readers. The visual chart is a glance-aid, not the only surface.
- **Sparkline ≠ chart.sm.** Sparkline is decorative-with-data: no axes, no labels, no tooltip-by-default. If you need axes / legend, use chart.sm — they're conceptually different primitives.
- **Gauge ≠ progress.circular.** Progress shows movement toward 100% (loading bars). Gauge shows a measured value within a min-max range (CPU usage, satisfaction score). The mental model is different.
