---
knowledge-base-summary: "Type scale from display to caption. Font selection criteria, line heights, letter spacing. Responsive typography for tablet. Google Fonts integration. Code examples for Flutter TextTheme and Tailwind typography config."
---
# Typography

Type system for the design system. Typography creates hierarchy, guides the eye, and establishes the visual tone of the application.

## Type Scale

Six levels, each with a clear purpose. No more, no less.

| Level     | Size (sp) | Weight | Line Height | Letter Spacing | Use Case                              |
|-----------|-----------|--------|-------------|----------------|---------------------------------------|
| Display L | 57        | 400    | 1.12 (64)   | -0.25          | Splash, hero, onboarding              |
| Display M | 45        | 400    | 1.16 (52)   | 0              | Large headings                        |
| Display S | 36        | 400    | 1.22 (44)   | 0              | Section heroes                        |
| Heading L | 32        | 600    | 1.25 (40)   | 0              | Page titles                           |
| Heading M | 28        | 600    | 1.29 (36)   | 0              | Major section headers                 |
| Heading S | 24        | 600    | 1.33 (32)   | 0              | Sub-section headers                   |
| Title L   | 22        | 600    | 1.27 (28)   | 0              | Card titles, dialog headers           |
| Title M   | 18        | 600    | 1.33 (24)   | 0.15           | List item titles, toolbar titles      |
| Title S   | 16        | 600    | 1.38 (22)   | 0.1            | Small titles, emphasized text         |
| Body L    | 16        | 400    | 1.50 (24)   | 0.5            | Main body text, paragraphs            |
| Body M    | 14        | 400    | 1.43 (20)   | 0.25           | Standard content, descriptions        |
| Body S    | 12        | 400    | 1.33 (16)   | 0.4            | Compact content, secondary info       |
| Label L   | 14        | 500    | 1.43 (20)   | 0.1            | Button text, form labels, tabs        |
| Label M   | 12        | 500    | 1.33 (16)   | 0.5            | Small buttons, badges, chips          |
| Label S   | 11        | 500    | 1.45 (16)   | 0.5            | Overline, uppercase labels            |
| Caption   | 12        | 400    | 1.33 (16)   | 0.4            | Helper text, timestamps, footnotes    |

## Font Selection Criteria

When choosing a font for the project:

1. **Legibility** — Must be highly readable at 14sp on mobile screens
2. **Language support** — Must cover required character sets (Latin, Cyrillic, Turkish, etc.)
3. **Weight range** — Minimum 400 (regular) and 600 (semibold). Ideal: 300-700
4. **Free and open** — Google Fonts preferred for easy integration
5. **Variable font** — Preferred for smaller bundle size and flexibility
6. **Performance** — Limit to 2-3 weights to minimize load time

### Recommended Font Pairings

**Clean and Modern:**
- Primary: Inter — excellent legibility, wide language support, variable font
- Alternative: Roboto — Google's default, safe choice

**Friendly and Warm:**
- Primary: Nunito Sans — slightly rounded, approachable
- Alternative: Source Sans 3 — clean, professional

**Technical and Precise:**
- Primary: JetBrains Mono (code only) — for monospaced/code content
- Body: Inter or IBM Plex Sans

### Single Font Strategy (Recommended)

For most projects, ONE font family is sufficient. Use weight and size to create hierarchy. Two fonts should only be used when there is a clear design reason (display headings vs body content).

## Line Height Rules

- **Body text**: 1.4 - 1.5x font size (for readability)
- **Headings**: 1.2 - 1.3x font size (tighter, more compact)
- **Display**: 1.1 - 1.2x font size (very tight)
- **Labels/Buttons**: 1.0 - 1.2x font size (single line, compact)

**Rule of thumb:** Larger text = tighter line height. Smaller text = looser line height.

## Letter Spacing

- **Display/Heading**: 0 or slightly negative (-0.25) — large text can be tighter
- **Body**: 0.25 - 0.5 — slightly loose for readability
- **Labels**: 0.1 - 0.5 — slightly tracked
- **Uppercase labels**: 0.5 - 1.0 — uppercase needs more tracking to remain legible

## Responsive Typography

Text should scale appropriately for larger screens:

```
Mobile (< 600px):    Base sizes as defined in the scale
Tablet (600-905px):  +2sp for display and heading levels
Desktop (> 905px):   +4sp for display, +2sp for heading levels
```

**Body text does NOT scale up** — 14-16sp is optimal across all screen sizes. Only display and heading levels grow on larger screens.

## Flutter Implementation

```dart
import 'package:google_fonts/google_fonts.dart';

class AppTypography {
  static TextTheme get textTheme => TextTheme(
    // Display
    displayLarge: GoogleFonts.inter(
      fontSize: 57,
      fontWeight: FontWeight.w400,
      height: 1.12,
      letterSpacing: -0.25,
    ),
    displayMedium: GoogleFonts.inter(
      fontSize: 45,
      fontWeight: FontWeight.w400,
      height: 1.16,
    ),
    displaySmall: GoogleFonts.inter(
      fontSize: 36,
      fontWeight: FontWeight.w400,
      height: 1.22,
    ),
    // Heading
    headlineLarge: GoogleFonts.inter(
      fontSize: 32,
      fontWeight: FontWeight.w600,
      height: 1.25,
    ),
    headlineMedium: GoogleFonts.inter(
      fontSize: 28,
      fontWeight: FontWeight.w600,
      height: 1.29,
    ),
    headlineSmall: GoogleFonts.inter(
      fontSize: 24,
      fontWeight: FontWeight.w600,
      height: 1.33,
    ),
    // Title
    titleLarge: GoogleFonts.inter(
      fontSize: 22,
      fontWeight: FontWeight.w600,
      height: 1.27,
    ),
    titleMedium: GoogleFonts.inter(
      fontSize: 18,
      fontWeight: FontWeight.w600,
      height: 1.33,
      letterSpacing: 0.15,
    ),
    titleSmall: GoogleFonts.inter(
      fontSize: 16,
      fontWeight: FontWeight.w600,
      height: 1.38,
      letterSpacing: 0.1,
    ),
    // Body
    bodyLarge: GoogleFonts.inter(
      fontSize: 16,
      fontWeight: FontWeight.w400,
      height: 1.50,
      letterSpacing: 0.5,
    ),
    bodyMedium: GoogleFonts.inter(
      fontSize: 14,
      fontWeight: FontWeight.w400,
      height: 1.43,
      letterSpacing: 0.25,
    ),
    bodySmall: GoogleFonts.inter(
      fontSize: 12,
      fontWeight: FontWeight.w400,
      height: 1.33,
      letterSpacing: 0.4,
    ),
    // Label
    labelLarge: GoogleFonts.inter(
      fontSize: 14,
      fontWeight: FontWeight.w500,
      height: 1.43,
      letterSpacing: 0.1,
    ),
    labelMedium: GoogleFonts.inter(
      fontSize: 12,
      fontWeight: FontWeight.w500,
      height: 1.33,
      letterSpacing: 0.5,
    ),
    labelSmall: GoogleFonts.inter(
      fontSize: 11,
      fontWeight: FontWeight.w500,
      height: 1.45,
      letterSpacing: 0.5,
    ),
  );
}
```

**Usage in Flutter:**
```dart
Text(
  'Page Title',
  style: Theme.of(context).textTheme.headlineLarge,
);

Text(
  'Body content goes here',
  style: Theme.of(context).textTheme.bodyMedium,
);
```

## Tailwind Implementation

```typescript
// tailwind.config.ts
export default {
  theme: {
    fontSize: {
      'display-lg': ['3.5625rem', { lineHeight: '1.12', letterSpacing: '-0.015em', fontWeight: '400' }],
      'display-md': ['2.8125rem', { lineHeight: '1.16', letterSpacing: '0', fontWeight: '400' }],
      'display-sm': ['2.25rem', { lineHeight: '1.22', letterSpacing: '0', fontWeight: '400' }],
      'heading-lg': ['2rem', { lineHeight: '1.25', letterSpacing: '0', fontWeight: '600' }],
      'heading-md': ['1.75rem', { lineHeight: '1.29', letterSpacing: '0', fontWeight: '600' }],
      'heading-sm': ['1.5rem', { lineHeight: '1.33', letterSpacing: '0', fontWeight: '600' }],
      'title-lg': ['1.375rem', { lineHeight: '1.27', letterSpacing: '0', fontWeight: '600' }],
      'title-md': ['1.125rem', { lineHeight: '1.33', letterSpacing: '0.009em', fontWeight: '600' }],
      'title-sm': ['1rem', { lineHeight: '1.38', letterSpacing: '0.006em', fontWeight: '600' }],
      'body-lg': ['1rem', { lineHeight: '1.5', letterSpacing: '0.03em', fontWeight: '400' }],
      'body-md': ['0.875rem', { lineHeight: '1.43', letterSpacing: '0.015em', fontWeight: '400' }],
      'body-sm': ['0.75rem', { lineHeight: '1.33', letterSpacing: '0.025em', fontWeight: '400' }],
      'label-lg': ['0.875rem', { lineHeight: '1.43', letterSpacing: '0.006em', fontWeight: '500' }],
      'label-md': ['0.75rem', { lineHeight: '1.33', letterSpacing: '0.03em', fontWeight: '500' }],
      'label-sm': ['0.6875rem', { lineHeight: '1.45', letterSpacing: '0.03em', fontWeight: '500' }],
    },
    fontFamily: {
      sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
    },
  },
} satisfies Config;
```

**Usage in Tailwind:**
```html
<h1 class="text-heading-lg">Page Title</h1>
<p class="text-body-md">Body content goes here</p>
<span class="text-label-lg">Button Text</span>
```

## Rules

1. **Never hardcode font sizes** — always use the type scale tokens
2. **Minimum body text is 14sp** — anything smaller is only for captions and labels
3. **One font family per role** — do not mix fonts arbitrarily
4. **Weight creates hierarchy, not just size** — 600 for headings, 400 for body, 500 for labels
5. **Test on real devices** — font rendering varies between iOS, Android, and web browsers
6. **Include fallback fonts** — system-ui stack for when custom fonts fail to load
