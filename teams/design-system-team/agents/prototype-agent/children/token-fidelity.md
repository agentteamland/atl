---
knowledge-base-summary: "The non-negotiable rule: NEVER hardcode visual values. Always reference DS tokens. How to express token references in `prototype.json` and resolve them at render time. What to do when DS lacks a needed token."
---
# Token Fidelity

The single most important rule: **NEVER hardcode visual values in a prototype.** Every color, font, size, spacing, radius, shadow, animation duration MUST reference the linked DS's tokens.

## Why

The whole point of a design system is that one place defines the visual language and many places consume it. If a prototype hardcodes `#2D5F3F` instead of referencing `palette.brand.primary`, then:
- Changing the palette means hunting through every prototype to update
- The prototype drifts from the system silently
- The prototype's preview looks "right" but doesn't actually use the system

When prototype agents and human reviewers can trust that "this prototype uses the DS exactly," reviews focus on layout/interaction (the actual design choices), not on "did you remember the right hex."

## How — token references in prototype.json

Anywhere you'd put a visual value, use a `{{ ds.<path> }}` reference:

```json
{
  "type": "button",
  "variant": "filled",
  "text": "{{ ds.voice.samples.ctaSubmit | default: 'Submit' }}",
  "tokens": {
    "background": "{{ ds.palette.brand.primary.value }}",
    "color": "{{ ds.palette.semantic.text.inverse }}",
    "height": "{{ ds.components.button.sizes.lg.height }}",
    "padding": "{{ ds.spacing.scale[3] }} {{ ds.spacing.scale[6] }}",
    "borderRadius": "{{ ds.radii.lg }}"
  }
}
```

The renderer resolves these placeholders at render time using the linked `ds.json`.

## How — token references in HTML

Two patterns:

### Pattern A — Tailwind utility classes mapped to DS via styles.css

The team's `styles.css` (in .dst/) extends Tailwind to expose DS tokens as utilities:

```css
/* In .dst/styles.css after Tailwind import */
.bg-ds-brand-primary { background-color: var(--ds-brand-primary); }
.text-ds-text-primary { color: var(--ds-text-primary); }
.h-ds-button-lg { height: var(--ds-button-height-lg); }
```

And the rendered preview.html sets those CSS variables from the linked DS:

```html
<style>
  :root {
    --ds-brand-primary: #2D5F3F;
    --ds-text-primary: #1A1A1A;
    --ds-text-inverse: #FFFFFF;
    --ds-button-height-lg: 52px;
    /* ... resolved from ds.json at render time */
  }
</style>
```

Then the prototype markup uses utility classes:

```html
<button class="bg-ds-brand-primary text-ds-text-inverse h-ds-button-lg px-6 rounded-lg">
  {{ ds.voice.samples.ctaSubmit }}
</button>
```

This is the **preferred pattern** because it's pure CSS — no inline styles, easy to read, and DS changes propagate via the variables on re-render.

### Pattern B — Inline styles (acceptable for one-off cases)

```html
<button style="background:#2D5F3F; color:#FFFFFF; height:52px; padding: 0 24px; border-radius: 12px;">
  Sign in
</button>
```

This is acceptable ONLY for values that are clearly resolved-from-tokens (the renderer wrote them after looking up `ds.palette.brand.primary.value`). Even so, the previous pattern is cleaner. Prefer A.

## The forbidden style

```html
<!-- ❌ NEVER -->
<button style="background:#2D5F3F">  <!-- naked literal -->
<input style="border:1px solid #E0E0DC">  <!-- unconnected to system -->
<div style="margin-top: 17px">  <!-- not on the spacing scale -->
```

If you find yourself writing values that don't trace back to a DS token, stop and reconsider.

## What to do when DS lacks a needed token

Sometimes the prototype needs something the DS doesn't provide:
- A color tone (e.g., a third gray between `surface` and `surfaceContainer`)
- A specific spacing not on the scale (rare; usually means scale is wrong)
- A new component variant

**Do not invent silently.** Choices:

### Option 1 — Use the closest DS token + flag in notes

```json
"blocks": [
  {
    "type": "card",
    "tokens": {
      "background": "{{ ds.palette.semantic.surfaceContainer }}"  /* close enough */
    }
  }
],
"notes": [
  "Used surfaceContainer for the inset card. Ideal would be a tone between surface and surfaceContainer; consider adding 'surfaceContainerLow' to the DS via /dst-edit-ds primary."
]
```

### Option 2 — Block prototype generation; ask the user to update DS first

If the gap is fundamental (e.g., entire palette section missing for a feature), tell the user:
> "Your DS doesn't have a `feedback.warning` color, but this screen needs warning states. Update the DS first: `/dst-edit-ds primary 'add warning color'`. Then re-run /dst-new-prototype."

Use judgment — small gaps → workaround + flag (Option 1); structural gaps → block (Option 2).

## Verifying token fidelity in preview.html

Before declaring a prototype done, scan the rendered HTML for:
- Naked `#xxxxxx` values in inline styles → should be CSS variables or utility classes
- Pixel values not in the DS spacing scale (`17px`, `13px`, etc.) → should be from scale
- Font sizes/weights not in the DS typography ramp → should be from ramp
- `rgba()` values that aren't part of palette → should be from semantic palette

A simple regex scan finds violations:

```bash
grep -oE 'style="[^"]*#[0-9A-Fa-f]{3,6}' preview.html
```

Should be empty (or only contain `var(--ds-*)` patterns).

## Token reference syntax (full grammar)

```
{{ ds.<path> }}                  → required, errors if missing
{{ ds.<path> | default: 'X' }}   → optional, falls back to literal if missing
```

Path syntax:
- `palette.brand.primary.value` — dot-separated
- `spacing.scale[4]` — array indexing
- `voice.samples.welcome` — nested object access

Examples:
- `{{ ds.palette.brand.primary.value }}` → `#2D5F3F`
- `{{ ds.typography.scale.body.size }}` → `16px`
- `{{ ds.spacing.scale[4] }}` → `16` (note: just the number; px appended at render time)
- `{{ ds.voice.samples.welcome }}` → `Welcome.`

## Anti-patterns

- ❌ `style="background:#2D5F3F"` (naked hex)
- ❌ `padding: 17px` (off-scale)
- ❌ Hardcoded font-family in a prototype (should reference DS typography.fontFamilies)
- ❌ Inline `<svg fill="#2D5F3F">` icons (use CSS `currentColor` and color via class)
- ❌ Multiple prototypes with the SAME hex repeated → centralize in DS palette
- ❌ "It's just a prototype, hex is fine" (no — habits matter)
