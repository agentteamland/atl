---
knowledge-base-summary: "Type scale construction (modular scales, line-height pairing), distinguishing display/body/label, system-font safety, web-font budgets, multi-script support (Turkish + Latin Extended)."
---
# Typography Ramps

A type ramp is the predefined scale of sizes/weights/line-heights used across the product. Define it once, reference everywhere.

## Default ramp (use this if user has no preference)

| Slot | Size | Line Height | Weight | Use |
|------|------|-------------|--------|-----|
| `displayLarge` | 48px | 56px | 600 | Marketing hero, splash screen titles |
| `displayMedium` | 36px | 44px | 600 | Section headlines on landing pages |
| `headline` | 24px | 32px | 600 | Screen titles |
| `title` | 20px | 28px | 600 | Card titles, modal headers |
| `body` | 16px | 24px | 400 | Default body text |
| `bodySmall` | 14px | 20px | 400 | Secondary body |
| `label` | 14px | 20px | 500 | Form labels, button text |
| `caption` | 12px | 16px | 400 | Hints, timestamps |

This is a 1.25× modular scale (16 → 20 → 24 → 36 → 48), proven across web/mobile.

## Font family choices

**Sans-serif** (default for UI):
- **Inter** — best general-purpose, supports Turkish, OFL licensed, Google Fonts. Default unless user specifies otherwise.
- **System UI** (`-apple-system, system-ui, ...`) — zero load cost, native feel.
- **Manrope** — more rounded, playful. Good for consumer apps.
- **DM Sans** — slightly compressed, minimal.

**Serif** (rarely used in UI; mostly editorial):
- **Source Serif Pro** — clean, web-optimized, good multi-script.

**Monospace** (for code, numbers, data):
- **JetBrains Mono** — excellent multi-script, ligatures.
- **Fira Code** — popular in dev tools.

## Multi-script considerations

For extended Latin (covers Turkish, Polish, Vietnamese, etc.):
- All fonts above ship full extended-Latin coverage without fallback (verify against your target languages before final selection).
- Flag in `ds.json` if a chosen font lacks coverage for any required diacritics (e.g., g-with-breve, s-with-cedilla).
- Avoid Roboto for body text when extended-Latin diacritics matter — the breve and cedilla glyphs render subpar compared to Inter.

## Web font budget

If using Google Fonts:
- 1 font family with up to 4 weights = ~50-80kb. Acceptable.
- 2 families = budget consideration. Accept only if there's a real distinction (e.g., display + body).
- 3+ families = rarely justified. Push back during Q&A.

## Letter-spacing

Display sizes (32px+) benefit from negative tracking: `letter-spacing: -0.02em`. Without it, large headers feel loose.
Body sizes: leave at default (`0`).
Caption/all-caps: positive tracking `0.04em` for legibility.

## Line-height pairing

Rule of thumb: line-height ≈ 1.4× for body, 1.2× for headlines.
For 16px body → 24px line-height (1.5× — slightly more for readability).
For 48px display → 56px line-height (1.17× — tight, intentional).

Always store both `size` and `lineHeight` explicitly in the ramp; don't compute at render time.

## Anti-patterns

- ❌ Including 12 type slots. If you have `headlineLarge`, `headlineMedium`, `headlineSmall`, you have too many. Default ramp is 8 slots — that's enough.
- ❌ Weight `300` for body text. Looks elegant in mockups, illegible at scale.
- ❌ Weight `900` for headers. Most fonts don't support it well; falls back to faux-bold.
- ❌ Mixing font sizes within a ramp role (e.g., `body` is sometimes 14, sometimes 16). Pick one; consistency is the system.
