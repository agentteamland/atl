---
knowledge-base-summary: "4px / 8px grid systems, vertical rhythm, density tiers (compact/comfortable/spacious), how to surface this in the rendered detail page."
---
# Spacing & Rhythm

Spacing is the invisible structure of UI. A consistent scale prevents arbitrary "12px or 16px?" decisions in every component.

## The grid: 4px

Default base unit: **4px**. Every spacing value is a multiple. This aligns with iOS (4pt) and Material Design.

Standard scale:
```
0, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128
```

14 values is enough. Anything between is a smell — don't introduce 18px or 22px.

## Density tiers

User picks one during Q&A:

- **Compact** (admin, dashboards, dense data UIs) — base padding `8px`, gaps `8-12px`, button height `32px`
- **Comfortable** (default, consumer apps) — base padding `12-16px`, gaps `12-16px`, button height `40px`
- **Spacious** (marketing, onboarding) — base padding `20-24px`, gaps `20-32px`, button height `48px`

Store the chosen density in `ds.json` as a hint:
```json
"spacing": {
  "unit": 4,
  "scale": [0, 4, 8, ...],
  "density": "comfortable"
}
```

Density doesn't change the scale; it changes which values you reach for in components.

## Vertical rhythm

For text-heavy sections, vertical spacing should align with line-heights. If body line-height is 24px:
- Section breaks: 48px (2 × line-height)
- Paragraph spacing: 16px (intuitive but slightly less than 1× line)
- Heading-to-body gap: 12px

This makes content feel "calm" rather than randomly spaced.

## Component-internal spacing recipe

For a typical card:
```
padding: 16px (compact) | 20px (comfortable) | 24px (spacious)
gap between header and body: 12px
gap between body and actions: 16px
gap between actions: 8px
```

For a typical form:
```
gap between fields: 16px (comfortable)
label-to-input gap: 6px
helper-text-to-input gap: 4px
```

## Section-level spacing (page layouts)

```
between major page sections: 64-80px (mobile) to 96-128px (desktop)
within a section: 24-32px
```

## Anti-patterns

- ❌ Using arbitrary values in CSS: `padding: 13px`. Always pick from the scale.
- ❌ Inconsistent density within one screen — compact form on a spacious page feels jarring.
- ❌ Negative margins to "fix" spacing. Means your scale or your component is wrong; don't paper over it.
