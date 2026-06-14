---
knowledge-base-summary: "How to design semantic palettes (brand, neutral, semantic — error/warning/success/info), generate tints/shades, ensure WCAG contrast, support dark mode."
---
# Palette Theory

A semantic palette has two layers: **brand** (identity-defining) and **semantic** (functionally named). They serve different audiences — brand for marketing and recognition, semantic for engineers building UI.

## Brand colors

Usually 1-3 colors that define the product's identity. Examples:
- A friendly consumer app: forest green (calm, trustworthy) + warm sand (approachability)
- A fintech: electric blue (trust + tech)
- A wellness app: muted coral + sage

Choose with intent. Personality keywords from `brand.personality` should map to color emotions. Don't pick colors that contradict the personality.

## Semantic palette

Standard semantic roles every UI needs:

| Role | Purpose |
|------|---------|
| `background` | Top-level app background |
| `surface` | Cards, sheets, elevated containers |
| `surfaceContainer` | Subtle inset surface (input fills, list rows) |
| `text.primary` | Primary text on surfaces |
| `text.secondary` | Secondary descriptive text |
| `text.muted` | Hints, placeholders, captions |
| `text.inverse` | Text on filled brand backgrounds |
| `border` | Strong dividers, input borders |
| `divider` | Subtle dividers between sections |
| `feedback.success` | Confirmation messages, success states |
| `feedback.warning` | Cautions, non-fatal warnings |
| `feedback.error` | Errors, destructive actions |
| `feedback.info` | Informational accents |

## Generating tints/shades

If the user provides only a brand seed (e.g., `#2D5F3F`), you can generate a tonal palette using HSL manipulation:

- **Tints** (lighter): increase lightness by 10% steps (50, 100, 200, 300, 400)
- **Brand color** sits at 500
- **Shades** (darker): decrease lightness (600, 700, 800, 900)

For semantic surface ramps, derive from a single neutral hue:
- `surface` = neutral 50 (white-ish)
- `surfaceContainer` = neutral 100 (slight gray)
- `surfaceContainerHigh` = neutral 200

## WCAG contrast

For every text-on-color pairing, compute and store the WCAG contrast ratio. Minimums:
- Body text (≥ 14px regular, < 18px bold): **4.5:1**
- Large text (≥ 18px regular, ≥ 14px bold): **3:1**
- Non-text UI elements (icons, borders): **3:1**

When you produce `ds.json`, include contrast metadata so `detail.html` can show WCAG badges:

```json
"semantic": {
  "text": {
    "primary": "#1A1A1A",
    "primaryOnSurface": { "value": "#1A1A1A", "vs": "#F8F8F6", "ratio": 14.5, "wcag": "AAA" }
  }
}
```

## Dark mode

If user opts in, derive a parallel palette:
- `background`: very dark, hint of brand hue (not pure black — `#0F1410` better than `#000000`)
- Inverted surface ramp (`surface` darker than `background`'s elevation feels wrong; in dark mode surface is LIGHTER than background)
- Text colors inverted but not pure white — `#F0F0F0` is more readable than `#FFFFFF`
- Brand colors may need slight brightness adjustment for dark backgrounds

Always check contrast in dark mode separately — dark mode contrast issues are different from light mode.

## When project tokens already exist

If you read `flutter/lib/app/theme.dart` or `tailwind.config.ts` and find an existing palette:
1. **Use those exact values** as your seed. Don't reinvent.
2. **Document them in ds.json** with the same names the project uses.
3. **Verify they meet WCAG** — if a project's existing `text.secondary` has 3.2:1 against surface, flag it as an issue (in `accessibility.notes`) rather than silently accepting.

The DS reflects what's true in the code, plus what should be true.

## Anti-patterns

- ❌ Naming colors by appearance (`color-blue-1`, `color-blue-2`). Use semantic names.
- ❌ Putting more than ~12 brand colors. If you have 20, you don't have a brand — you have a paint store.
- ❌ Skipping `feedback` colors. Every product has errors. Your DS must too.
- ❌ Using exact same value for two different roles (e.g., `text.primary` and `border` both `#000`). They serve different purposes; flexibility to evolve them independently matters.
