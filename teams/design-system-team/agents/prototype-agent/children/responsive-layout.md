---
knowledge-base-summary: "Mobile vs tablet vs desktop variations within one prototype. When to render multiple breakpoints, how to declare them in `prototype.json`, how to lay them out in `preview.html`."
---
# Responsive Layout

Prototypes can target one or multiple breakpoints. Each breakpoint = a self-contained variant of every frame.

## Breakpoint definitions

```
mobile   < 768px    (phone in portrait/landscape)
tablet   768-1023px (iPad-ish)
desktop  >= 1024px  (laptop / desktop)
```

These are conventional defaults; the linked DS may override (check `ds.json.breakpoints` if present).

## Single-breakpoint prototypes (default v0.1.0)

Most prototypes are single-breakpoint. The `breakpoints` field in `prototype.json` has one entry:

```json
"breakpoints": ["mobile"]
```

Each frame renders for that one breakpoint only. Preview shows them as standard frame stack.

Use this when:
- Mobile-first app (Flutter mobile prototype) — `["mobile"]`
- Admin panel that's desktop-only — `["desktop"]`
- Simple flow you'll iterate on later

## Multi-breakpoint prototypes (v0.2.0+)

When a screen meaningfully differs across viewport sizes:

```json
"breakpoints": ["mobile", "desktop"]
```

Each frame is rendered TWICE (once per breakpoint), laid out side-by-side in `preview.html`:

```
                IDLE
─────────────────────────────────────
[mobile frame]    [desktop frame]
                 (responsive widths)
```

Use when:
- Mobile + desktop have different layouts (e.g., bottom-nav on mobile, sidebar-nav on desktop)
- Important to validate visual design at both extremes before code

## Per-frame breakpoint variations

If a particular frame needs different content at different breakpoints (rare), declare frame-level breakpoints:

```json
"frames": {
  "idle": {
    "label": "Default",
    "breakpoints": {
      "mobile": { "blocks": [ ... ] },
      "desktop": { "blocks": [ ... ] }
    }
  }
}
```

If a frame's `breakpoints` is omitted, it uses the prototype-level `breakpoints` (same blocks rendered at each viewport width).

## Layout decisions per breakpoint

### Mobile

- Full-width inputs, full-width primary buttons
- Vertical stacking is default
- Bottom-positioned primary actions (thumb reach)
- Single-column lists/grids
- Padding: 16-20px from screen edges

### Tablet

- May allow two-column layouts (master-detail)
- Buttons can be smaller / inline (still ≥ 44px touch target)
- Padding: 24-32px

### Desktop

- Multi-column comfortable
- Sidebars + main content
- Buttons can be inline / smaller
- Hover states matter (cursor-based interaction)
- Padding: 32-48px from edges; max-width container often used (`max-w-6xl`, `mx-auto`)

## Handling responsiveness in `preview.html`

The rendered preview shows each breakpoint at its native intended width:

```html
<div class="frame-container">
  <h3 class="state-label">IDLE</h3>

  <div class="breakpoints-grid">
    <!-- Mobile variant -->
    <div class="bp-mobile" style="width: 375px;">
      <p class="bp-label">mobile (375px)</p>
      [rendered mobile blocks here]
    </div>

    <!-- Desktop variant -->
    <div class="bp-desktop" style="width: 1280px;">
      <p class="bp-label">desktop (1280px)</p>
      [rendered desktop blocks here]
    </div>
  </div>
</div>
```

The page's overall width allows horizontal scroll if both breakpoints don't fit side-by-side.

## When NOT to multi-bp

Don't pad `breakpoints` for the sake of completeness. If mobile and desktop look essentially the same (just different widths), a single `mobile` breakpoint is enough — desktop will inherit the same layout, just rendered wider in the actual app.

The signal for adding a new breakpoint: **the layout meaningfully changes** (sidebar appears, columns shift, navigation moves).

## Default heuristics

If user doesn't explicitly say:
- App archetype suggests target (Flutter app → mobile, admin → desktop)
- DS in `ds.json` may have a hint via `metadata.targetPlatform`
- Default fallback: ask the user during Q&A (simple multiSelect)

## Anti-patterns

- ❌ Three breakpoints for an admin dashboard that nobody opens on a phone
- ❌ Two breakpoints for screens that look identical at both
- ❌ Mobile screen with hover-only interactions (defeats the breakpoint purpose)
- ❌ Desktop screen with full-width buttons (looks awkward; usually narrower buttons inline)
