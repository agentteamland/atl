---
knowledge-base-summary: "The canonical shape of `ds.json`. Every field's purpose, allowed values, examples."
---
# Design System Schema (`ds.json`)

This is the canonical structure of every design system this team produces. The JSON file lives at `.dst/design-systems/{ds-name}/ds.json` and is the single source of truth — `detail.html` is rendered from it.

## Top-Level Shape

```json
{
  "schemaVersion": 1,
  "name": "primary",
  "version": "1.0.0",
  "createdAt": "2026-04-22T10:00:00Z",
  "lastModified": "2026-04-22T10:00:00Z",
  "description": "ExampleApp primary brand — outdoor, approachable, trustworthy.",

  "brand": { ... },
  "palette": { ... },
  "typography": { ... },
  "spacing": { ... },
  "radii": { ... },
  "elevation": { ... },
  "motion": { ... },
  "components": { ... },
  "voice": { ... },
  "accessibility": { ... }
}
```

## `brand`

```json
{
  "personality": ["friendly", "approachable", "trustworthy"],
  "tagline": "Designed in your editor.",
  "logomark": {
    "type": "svg-inline",
    "path": "assets/logomark.svg",
    "minSize": "24px",
    "maxSize": "256px",
    "clearSpace": "1× logomark height"
  },
  "logomarkDark": null,
  "wordmark": {
    "text": "ExampleApp",
    "fontFamily": "Inter",
    "fontWeight": 600
  },
  "wordmarkDark": null,
  "lockup": {
    "horizontal": "logomark + wordmark side-by-side",
    "vertical": "logomark above wordmark"
  }
}
```

**Dark brand variants — `logomarkDark` / `wordmarkDark`:**
- Optional. Same shape as their light counterparts (object with `type`/`path` for logomark, object with `text`/`fontFamily`/`fontWeight` for wordmark, OR a string asset path).
- `null` (or absent) is a valid value — means "no dedicated dark asset; render the light one on dark surface."
- detail.html renders **both** light and dark lockups always. When the Dark variant is null, the dark stage shows the light asset on the dark surface AND emits a `.ds-lockup-warn` strip ("⚠ Dark variant not declared"). Designers see the gap explicitly, and the rule is the same across DSes.
- The Q&A flow asks about dark assets only when `palette.dark` is being populated — there's no point asking when dark mode itself is off.

## `palette`

Two layers: `brand` (the seed identity) and `semantic` (functional roles). Optional `dark` variant.

```json
{
  "brand": {
    "primary": { "value": "#2D5F3F", "name": "Forest Green" },
    "secondary": { "value": "#D4A574", "name": "Warm Sand" }
  },
  "semantic": {
    "background": "#FFFFFF",
    "surface": "#F8F8F6",
    "surfaceContainer": "#F0EFEC",
    "text": {
      "primary": "#1A1A1A",
      "secondary": "#4A4A4A",
      "muted": "#7A7A7A",
      "inverse": "#FFFFFF"
    },
    "border": "#E0E0DC",
    "divider": "#EDEDE9",
    "feedback": {
      "success": "#2D7A4F",
      "warning": "#D49A2A",
      "error": "#C04444",
      "info": "#3A6FA8"
    }
  },
  "dark": {
    "background": "#0F1410",
    "surface": "#1A211D",
    "...": "..."
  }
}
```

Always include WCAG-significant pairings (text-on-surface, text-on-primary) so the rendered detail page can show contrast ratios. Compute and store contrast values when you write the file.

## `typography`

```json
{
  "fontFamilies": {
    "sans": {
      "stack": "Inter, -apple-system, system-ui, sans-serif",
      "googleFont": "Inter",
      "weights": [400, 500, 600, 700]
    },
    "serif": null,
    "mono": {
      "stack": "JetBrains Mono, Menlo, monospace",
      "googleFont": "JetBrains Mono",
      "weights": [400, 500]
    }
  },
  "scale": {
    "displayLarge": { "size": "48px", "lineHeight": "56px", "weight": 600, "letterSpacing": "-0.02em" },
    "displayMedium": { "size": "36px", "lineHeight": "44px", "weight": 600, "letterSpacing": "-0.02em" },
    "headline": { "size": "24px", "lineHeight": "32px", "weight": 600, "letterSpacing": "-0.01em" },
    "title": { "size": "20px", "lineHeight": "28px", "weight": 600 },
    "body": { "size": "16px", "lineHeight": "24px", "weight": 400 },
    "bodySmall": { "size": "14px", "lineHeight": "20px", "weight": 400 },
    "label": { "size": "14px", "lineHeight": "20px", "weight": 500 },
    "caption": { "size": "12px", "lineHeight": "16px", "weight": 400 }
  }
}
```

## `spacing`

```json
{
  "unit": 4,
  "scale": [0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128],
  "density": "comfortable"
}
```

## `radii`

```json
{
  "none": "0px",
  "sm": "4px",
  "md": "8px",
  "lg": "12px",
  "xl": "16px",
  "full": "9999px"
}
```

## `elevation`

```json
{
  "none": "none",
  "low": "0 1px 2px rgba(0,0,0,0.05)",
  "medium": "0 4px 8px rgba(0,0,0,0.08)",
  "high": "0 12px 24px rgba(0,0,0,0.12)"
}
```

## `motion`

```json
{
  "durations": {
    "instant": "100ms",
    "fast": "200ms",
    "normal": "300ms",
    "slow": "500ms"
  },
  "easings": {
    "standard": "cubic-bezier(0.4, 0.0, 0.2, 1)",
    "decelerate": "cubic-bezier(0, 0, 0.2, 1)",
    "accelerate": "cubic-bezier(0.4, 0, 1, 1)"
  }
}
```

## `components`

Per-component variant matrix. Stored as definitions; prototype-agent consumes when generating screens.

```json
{
  "button": {
    "variants": {
      "filled": { "background": "brand.primary", "text": "text.inverse", "padding": "12px 24px" },
      "outlined": { "background": "transparent", "border": "1px solid brand.primary", "text": "brand.primary" },
      "text": { "background": "transparent", "text": "brand.primary" }
    },
    "sizes": {
      "sm": { "height": "32px", "fontSize": "bodySmall" },
      "md": { "height": "40px", "fontSize": "body" },
      "lg": { "height": "52px", "fontSize": "body" }
    },
    "states": ["idle", "hover", "focus", "pressed", "disabled", "loading"]
  },
  "input": { ... },
  "card": { ... },
  "chip": { ... }
}
```

## `voice`

```json
{
  "tone": "warm but professional; concise; second-person address",
  "do": [
    "Use active voice",
    "Lead with the benefit, not the feature",
    "Keep button labels under 3 words"
  ],
  "dont": [
    "Don't use exclamation marks except in genuine errors",
    "Don't apologize unnecessarily ('Sorry, we couldn't load...')",
    "Don't use jargon or technical acronyms in user-facing text"
  ],
  "samples": {
    "welcome": "Welcome.",
    "errorGeneric": "Something went wrong. Let's try again.",
    "emptyState": "Nothing here yet. Care to add the first one?"
  }
}
```

## `accessibility`

```json
{
  "wcagLevel": "AA",
  "minimumTapTarget": "44px",
  "minimumContrastBody": 4.5,
  "minimumContrastLarge": 3.0,
  "focusVisible": "always (no `outline: none`)",
  "reducedMotion": "honored (motion durations × 0.5 when prefers-reduced-motion)"
}
```

## Versioning

`schemaVersion: 1` reserved for this initial schema. Bump if breaking changes (new required fields, type changes). Per-DS `version` is independent — bumped when the DS itself is meaningfully edited.

## Notes for Generation

- **All hex values uppercase**, with `#`.
- **All sizes in `px`** (or `%` / `rem` only where explicitly justified — body line-heights stay px for predictability).
- **Compute contrast ratios** when writing palette so the rendered detail.html can display them as WCAG badges.
- **Don't include null fields** — omit if not used (e.g., no serif font → `serif: null` is acceptable but cleaner to omit).
- **All strings UTF-8 / Turkish-safe** — no escaping of Turkish characters.
