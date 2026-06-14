---
knowledge-base-summary: "Logo lockup variants, logomark sizes, iconography family, voice & tone do/don't, sample copy."
---
# Brand Identity

Beyond colors and fonts, a design system carries the product's voice and visual identity at a higher level.

## Logomark

The icon-only mark of the brand. Should:
- Work at 24px minimum (legible app icon size)
- Work at 256px maximum (favicon to splash screen range)
- Have clear space (whitespace) around equal to ~1× its height
- Be SVG-native (vector)

Store in `ds.json`:
```json
"brand": {
  "logomark": {
    "type": "svg-inline" | "svg-file",
    "path": "assets/logomark.svg",
    "minSize": "24px",
    "maxSize": "256px",
    "clearSpace": "1× logomark height"
  }
}
```

If user doesn't have a logomark, generate a placeholder SVG (simple geometric mark using brand color) and note it as "placeholder — replace with brand asset".

## Wordmark

The brand name set in a specific font/weight. Often paired with the logomark in a "lockup."

```json
"brand": {
  "wordmark": {
    "text": "ExampleApp",
    "fontFamily": "Inter",
    "fontWeight": 600,
    "letterSpacing": "-0.01em"
  }
}
```

## Lockup

The combination of logomark + wordmark in fixed arrangements:
- **Horizontal** — logomark left, wordmark right
- **Vertical** — logomark above wordmark
- **Stacked** — variants for different aspect ratios (square, banner)

Store the rules in ds.json so prototypes can reference them:
```json
"brand": {
  "lockup": {
    "horizontal": { "gap": "12px", "alignment": "center" },
    "vertical":   { "gap": "8px",  "alignment": "center" }
  }
}
```

## Personality keywords

3-5 adjectives that capture the brand's essence. Used by:
- Voice tone (e.g., "playful" → exclamation marks ok)
- Color choices (e.g., "trustworthy" → blues, deep greens)
- Typography (e.g., "approachable" → rounded serif/sans)

```json
"brand": {
  "personality": ["outdoor", "approachable", "trustworthy", "calm"]
}
```

## Voice & Tone

Where copy decisions live. Critical for prototype-agent so user-facing strings stay consistent.

```json
"voice": {
  "tone": "warm but professional; concise; second-person address; Turkish-first",
  "do": [
    "Use active voice",
    "Lead with the benefit, not the feature",
    "Keep button labels under 3 words"
  ],
  "dont": [
    "Don't use exclamation marks except in errors",
    "Don't apologize unnecessarily",
    "Don't use technical jargon in user-facing text"
  ],
  "samples": {
    "welcome": "Welcome.",
    "errorGeneric": "Something went wrong. Let's try again.",
    "emptyState": "Nothing here yet. Care to add the first one?",
    "ctaPrimary": "Start",
    "ctaSecondary": "Later"
  }
}
```

The samples are critical — they show the tone in action and prototype-agent reuses them. Generate at least 8-10 samples covering common surfaces: welcome, error, empty, success, button-CTA, link-secondary, label-required, helper-text.

## Iconography family

Decide one icon family for consistency. Options:
- **Material Symbols** (Google) — comprehensive, multi-style, free
- **Lucide** — minimal, modern, popular in React/Tailwind ecosystems
- **Heroicons** — Tailwind's native set, clean
- **Custom** — only for established brands with budget for icon design

```json
"brand": {
  "iconography": {
    "family": "lucide",
    "style": "outline",
    "defaultSize": "24px"
  }
}
```

Prototype-agent uses this when picking icons for screens.

## Anti-patterns

- ❌ Not having voice samples. "Tone is friendly" → useless. Show actual sentences.
- ❌ Multiple icon families on the same product. Pick one; stick with it.
- ❌ Logomark only working at one size. Test at 24px and 128px both.
- ❌ Personality contradictions: "playful, professional, edgy, trustworthy" — pick a coherent set.
