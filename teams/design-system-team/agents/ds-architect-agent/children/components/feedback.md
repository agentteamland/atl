---
knowledge-base-summary: "alert, toast, tooltip, progress, skeleton, spinner, banner, callout, notification-bell. Surface system response. Urgency ladder (tooltip → callout → alert → banner → toast/modal). Color + icon pairing for accessibility."
---
# Feedback group

> Anchor: `#components-feedback` in `detail.html`.
> Source-of-truth: `ds.json.components.{alert, toast, tooltip, progress, skeleton, spinner, banner, callout, notification-bell}`.

**Definition.** Components that **surface system response** — what just happened, what's happening, what's about to happen. Distinct from Overlays (modals/drawers float over content to ask the user something); feedback components inform without demanding action.

## Components

### alert

Inline, dismissible message attached to a region. Used at the top of forms, inside cards, above lists.

```json
"alert": {
  "variants": {
    "info":    { "background": "feedback.info.surface",    "border": "1px solid feedback.info.border",    "iconColor": "feedback.info",    "title": "text.primary", "note": "Neutral notification." },
    "success": { "background": "feedback.success.surface", "border": "1px solid feedback.success.border", "iconColor": "feedback.success", "title": "text.primary", "note": "Confirmation that something worked." },
    "warning": { "background": "feedback.warning.surface", "border": "1px solid feedback.warning.border", "iconColor": "feedback.warning", "title": "text.primary", "note": "Heads-up — non-blocking issue." },
    "error":   { "background": "feedback.error.surface",   "border": "1px solid feedback.error.border",   "iconColor": "feedback.error",   "title": "text.primary", "note": "Something is broken — block submit until resolved." }
  },
  "sizes": {
    "sm": { "padding": "10px 12px", "fontSize": "bodySmall", "iconSize": "16px", "radius": "md" },
    "md": { "padding": "14px 16px", "fontSize": "body",      "iconSize": "20px", "radius": "lg", "note": "Default." }
  },
  "states": ["idle", "dismissed"],
  "stateRules": {
    "dismissed": "Component animates out (accelerate, fast); next page load it doesn't reappear (persisted dismissal optional)."
  },
  "structure": "icon | title + body | (optional) action button | (optional) close icon-button"
}
```

### toast

Transient notification that auto-dismisses. Stacks at a corner of the viewport.

```json
"toast": {
  "variants": {
    "info":    { "background": "surface", "border": "1px solid feedback.info.border",    "iconBg": "feedback.info.surface",    "iconColor": "feedback.info" },
    "success": { "background": "surface", "border": "1px solid feedback.success.border", "iconBg": "feedback.success.surface", "iconColor": "feedback.success" },
    "warning": { "background": "surface", "border": "1px solid feedback.warning.border", "iconBg": "feedback.warning.surface", "iconColor": "feedback.warning" },
    "error":   { "background": "surface", "border": "1px solid feedback.error.border",   "iconBg": "feedback.error.surface",   "iconColor": "feedback.error" }
  },
  "sizes": {
    "md": { "padding": "12px 16px", "fontSize": "body", "iconSize": "20px", "radius": "lg", "minWidth": "280px", "maxWidth": "420px" }
  },
  "states": ["entering", "visible", "exiting"],
  "stateRules": {
    "entering": "Slides in from corner (decelerate, normal).",
    "exiting":  "Fades out (accelerate, fast)."
  },
  "behavior": {
    "duration": "5000ms (success/info), 8000ms (warning/error). Hover pauses the timer.",
    "stacking": "Maximum 3 visible; older toasts auto-dismiss to make room.",
    "position": "Bottom-right on desktop; top-center on mobile."
  }
}
```

### tooltip

Tiny label revealed on hover or focus. For terse hints — "Save (⌘S)", "Last updated 3m ago", icon-button labels.

```json
"tooltip": {
  "variants": {
    "default": { "background": "neutral.900", "text": "text.inverse", "elevation": "medium" }
  },
  "sizes": {
    "sm": { "padding": "4px 8px",  "fontSize": "caption",   "radius": "md" },
    "md": { "padding": "6px 10px", "fontSize": "bodySmall", "radius": "md", "note": "Default." }
  },
  "states": ["hidden", "visible"],
  "behavior": {
    "trigger":   "hover (300ms delay) | focus (immediate)",
    "anchor":    "above by default; flips to below if clipped at top of viewport",
    "maxWidth":  "240px — wraps if longer",
    "arrow":     "8px triangle on the side facing the trigger"
  },
  "antiPatterns": [
    "Don't put critical info in a tooltip — touch users can't hover.",
    "Don't put interactive elements (buttons, links) inside a tooltip — use popover instead."
  ]
}
```

### progress

Progress indicator. Two variants: linear bar (file uploads, multi-step forms) and circular (page-level loading, sparser visual).

```json
"progress": {
  "variants": {
    "linear":   { "trackBg": "surfaceContainer", "fillBg": "brand.primary", "note": "File uploads, multi-step forms — when total work is known." },
    "circular": { "trackBg": "surfaceContainer", "fillBg": "brand.primary", "note": "Page-level loading — when total work is known but visual budget is tight." }
  },
  "sizes": {
    "sm": { "linearHeight": "4px",  "circularSize": "24px", "circularStroke": "3px" },
    "md": { "linearHeight": "6px",  "circularSize": "40px", "circularStroke": "4px", "note": "Default." },
    "lg": { "linearHeight": "8px",  "circularSize": "64px", "circularStroke": "5px" }
  },
  "states": ["determinate", "indeterminate"],
  "stateRules": {
    "determinate":   "Fill width/arc reflects `value/max` ratio. Use when progress is measurable.",
    "indeterminate": "Animated stripe (linear) or sweep (circular). Use ONLY when total is unknown — for measurable progress, an indeterminate bar feels deceptive."
  },
  "labelPairing": "Always pair with a numeric label nearby (e.g., '32 of 100' for determinate, 'Loading…' for indeterminate). A bar without context is just decoration."
}
```

### skeleton

Placeholder shape that fills space while data loads. Reduces perceived latency by setting layout expectations.

```json
"skeleton": {
  "variants": {
    "line":   { "background": "shimmer.gradient", "radius": "sm",  "note": "Single line of text — heading, label, paragraph row." },
    "block":  { "background": "shimmer.gradient", "radius": "lg",  "note": "Card, image, list row — anything with a contained shape." },
    "circle": { "background": "shimmer.gradient", "radius": "full","note": "Avatar, icon-button, profile image." }
  },
  "sizes": {
    "default": { "shimmerDuration": "1.2s", "shimmerEasing": "decelerate", "lineHeight": "16px", "blockHeight": "80px", "circleSize": "40px" }
  },
  "states": ["shimmering", "static"],
  "stateRules": {
    "shimmering": "Background: linear-gradient with 200% size; animation: pv-shimmer 1.2s decelerate infinite.",
    "static":     "Used when prefers-reduced-motion is set — animation is disabled, gradient frozen."
  },
  "compositionRule": "Mirror the actual component layout. A list of cards loading should show a list of card skeletons — not a single 'Loading…' label. The point is layout fidelity."
}
```

### spinner

Indeterminate loader for short waits (< 3s). For longer waits, prefer skeleton (sets expectations) or progress (sets measurable expectations).

```json
"spinner": {
  "variants": {
    "default": { "trackColor": "brand.100", "fillColor": "brand.primary" },
    "inverse": { "trackColor": "rgba(255,255,255,0.24)", "fillColor": "text.inverse", "note": "On colored surfaces (inside a primary button)." }
  },
  "sizes": {
    "xs": { "size": "12px", "stroke": "1.5px" },
    "sm": { "size": "16px", "stroke": "2px" },
    "md": { "size": "20px", "stroke": "2px", "note": "Default." },
    "lg": { "size": "32px", "stroke": "3px" }
  },
  "states": ["spinning", "static"],
  "stateRules": {
    "spinning": "Animation: pv-spin 0.8s linear infinite.",
    "static":   "When prefers-reduced-motion is set, spinner is replaced with a static indeterminate progress dots fallback."
  }
}
```

### banner

Page-level message that occupies a full row. Used for system status (maintenance windows, upgrade prompts, cookie consent).

```json
"banner": {
  "variants": {
    "info":      { "background": "feedback.info.surface",    "text": "text.primary", "note": "Status update — non-actionable." },
    "promo":     { "background": "brand.50",                 "text": "brand.900",     "note": "Marketing or upgrade nudge." },
    "warning":   { "background": "feedback.warning.surface", "text": "text.primary", "note": "Heads-up that requires attention but not action." },
    "critical":  { "background": "feedback.error.surface",   "text": "text.primary", "note": "System-wide outage or pre-deletion warning." }
  },
  "sizes": {
    "md": { "padding": "12px 24px", "fontSize": "body", "radius": "0", "note": "Default — full-width edge-to-edge." }
  },
  "states": ["idle", "dismissed"],
  "structure": "icon | message | (optional) action link | (optional) close icon-button",
  "placementRule": "Always pinned to the top of the viewport (above the top nav) or the top of the affected region. Never inline within content."
}
```

### callout

In-content highlight box. Used inside docs/articles to call out a tip, note, or warning. Distinct from alert (alert is dismissible system feedback; callout is editorial emphasis).

```json
"callout": {
  "variants": {
    "tip":     { "background": "feedback.success.surface", "border": "1px solid feedback.success.border", "iconColor": "feedback.success", "label": "Tip" },
    "note":    { "background": "feedback.info.surface",    "border": "1px solid feedback.info.border",    "iconColor": "feedback.info",    "label": "Note" },
    "warning": { "background": "feedback.warning.surface", "border": "1px solid feedback.warning.border", "iconColor": "feedback.warning", "label": "Warning" }
  },
  "sizes": {
    "md": { "padding": "16px 20px", "fontSize": "body", "iconSize": "20px", "radius": "lg" }
  },
  "states": ["idle"],
  "structure": "icon + label (e.g., 'Tip') | body (markdown supported) | (optional) link",
  "placementRule": "Inline within long-form content. Never use as system feedback — that's `alert`."
}
```

### notification-bell

Notification indicator anchored at the top-right of the app shell. Pairs with menu / popover for the notification list.

```json
"notification-bell": {
  "variants": {
    "default":   { "showIndicator": false,                                                                       "note": "Pure bell icon — no unread state." },
    "withDot":   { "showIndicator": true,  "indicatorType": "dot",   "indicatorColor": "feedback.error",         "note": "Unread indicator — small dot at top-right corner of bell." },
    "withCount": { "showIndicator": true,  "indicatorType": "badge", "indicatorColor": "feedback.error", "format": "{count}", "maxDisplay": 99, "overflowSuffix": "+", "note": "Unread count — badge with number. Shows '99+' if count > 99." }
  },
  "sizes": {
    "sm": { "size": "32px", "iconSize": "18px", "indicatorSize": "8px",  "badgeHeight": "16px" },
    "md": { "size": "40px", "iconSize": "20px", "indicatorSize": "8px",  "badgeHeight": "16px", "note": "Default — paired with sidebar-nav / top nav." },
    "lg": { "size": "48px", "iconSize": "24px", "indicatorSize": "10px", "badgeHeight": "18px" }
  },
  "states": ["idle", "hover", "focus", "active"],
  "stateRules": {
    "hover":  "Background: surfaceContainer.",
    "focus":  "2px outline brand.500, 2px offset.",
    "active": "When the notification panel (menu/popover) is open: background brand.50; bell tilt animation on open."
  },
  "structure": "icon-button (bell icon) | (optional) indicator anchored top-right (badge.dot OR badge.error)",
  "behavior": "Trigger pattern — pairs with menu (action list) or popover (rich panel). Click toggles. Indicator clears on panel open OR on individual notification dismissal (declared by app)."
}
```

## Group-level rules

- **Choose the right surface for the right urgency.** From least → most: tooltip → callout → alert → banner → toast (transient) / modal (Overlays group, blocking).
- **Pair color with icon.** Color-blindness defense — every variant uses an icon as well as a color.
- **Auto-dismiss only for transient.** Toasts dismiss; alerts and banners do not. Anything actionable must persist until acknowledged.
- **Reduced motion respect.** All shimmer / spin / slide-in animations honor `prefers-reduced-motion`.
- **Tooltip is feedback, not navigation.** Don't use tooltips as a substitute for proper labels. If users need to discover an action, label it directly.
