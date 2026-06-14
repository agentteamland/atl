---
knowledge-base-summary: "How to write `theme.dart` / `tailwind.config.ts` so Claude Design (via `/design-screen`) reads tokens reliably. MVP set is sufficient — palette via `ColorScheme.fromSeed`, typography ramp, 4px grid in comments, key component themes, dark mode. Single canonical entry-point file per platform; top-of-file comments document intent; no magic numbers; non-obvious computations get explanatory comments. Pilot evidence: well-written `theme.dart` led to zero hex literals in resulting Dart and self-documenting bundle output."
---
# Writing Tokens That Claude Design Can Read

When the project uses Claude Design (per `/design-screen` skill), the bundle's quality is bounded by how readable your tokens are. The reference-project pilot (2026-04-19) demonstrated that **a minimal, well-documented `theme.dart` is enough** — Claude Design's `tokens.jsx` self-documented as "derived from flutter/lib/app/theme.dart" and the resulting Dart had zero hex literals. That outcome is reproducible only if your tokens are written intentionally.

## The MVP Set (sufficient for Claude Design)

You don't need the full design system before running `/design-screen start`. The minimum viable set:

| Layer | What | Why |
|-------|------|-----|
| **Palette** | `seedColor` + `ColorScheme.fromSeed(...)`. Define light + dark. | Claude Design will use these as the entire color vocabulary. |
| **Typography** | `TextTheme` with at least `headlineMedium`, `titleLarge`, `bodyLarge`, `bodyMedium`, `labelLarge`, `labelMedium`. | Bundle needs to know your hierarchy. |
| **Spacing** | A 4px or 8px grid documented in a comment near the theme definition. | Bundle infers vertical rhythm from this. |
| **Radii** | `cardTheme.shape`, `filledButtonTheme.shape`, `inputDecorationTheme.border`. | Bundle reads these to set component corner radii. |
| **Component themes** | At minimum: `filledButtonTheme`, `inputDecorationTheme`, `appBarTheme`. | Bundle picks button heights, input fills, app bar tones from here. |
| **Dark mode** | Both `light` and `dark` ThemeData defined in the same file. | Bundle can emit dark variants if your project supports them. |

That's it. The pilot ran with exactly this set (no motion tokens, no custom typography ramps, no extended color palette) and got 4/5 fidelity.

## What Makes Tokens "Readable" by Claude Design

### 1. Single canonical file per platform

- **Flutter:** `flutter/lib/app/theme.dart` exports `lightTheme` and `darkTheme`. Don't scatter token definitions across `colors.dart`, `typography.dart`, `spacing.dart` files unless you ALSO have the consolidated `theme.dart` that imports them. Claude Design reads the consolidated file.
- **React:** `web/tailwind.config.ts` (or `apps/admin/tailwind.config.ts` etc.). The `theme.extend` block is the readable surface.

If your project has multiple files, ensure the consolidated entry point exists and is named obviously (`theme.dart`, `tailwind.config.ts`).

### 2. Comments at the top of the file describe intent

The pilot's tokens.jsx self-documented. That's because the project's `theme.dart` had explanatory comments at the top. Example:

```dart
// flutter/lib/app/theme.dart
//
// Source of truth for ALL visual tokens.
// - Palette: ColorScheme.fromSeed(seedColor: AppColors.brandPrimary)
//   — light and dark generated from the same seed for consistency
// - Spacing scale: 4px grid (4, 8, 12, 16, 20, 24, 28, 32, 40, 48, 64)
//   — used directly in Padding/SizedBox; do NOT introduce arbitrary values
// - Vertical rhythm reference: 28-6-28-20-8-20 for compact form stacks
// - Radii: 12 for inputs/buttons, 16 for cards, 24 for sheets
// - Motion: Material 3 defaults (don't override unless brand requires)
//
// When Claude Design reads this file (via /design-screen), the comments
// here become its constraints. Keep them current.
```

These comments serve double duty: they document for human developers AND they ground Claude Design's interpretation when it reads the file.

### 3. Token names are explicit and consistent

```dart
// ❌ HARMFUL — Claude Design can't infer intent
const c1 = Color(0xFF1A73E8);
const c2 = Color(0xFFE8E2D9);

// ✅ GOOD — semantic naming, scheme-derived
class AppColors {
  static const brandPrimary = Color(0xFF1A73E8);  // seed for ColorScheme
  // (everything else derived from ColorScheme.fromSeed)
}

// ✅ EVEN BETTER — let ColorScheme do the work, never expose raw hex outside the seed
final lightTheme = ThemeData.from(
  colorScheme: ColorScheme.fromSeed(seedColor: AppColors.brandPrimary),
);
```

Claude Design can read `ColorScheme.fromSeed(...)` and emit a tokens.jsx that uses Material 3's full surface ramp (`surface`, `surfaceContainer`, `surfaceContainerHighest`, etc.) automatically. If you bypass `ColorScheme` and define raw colors, the bundle's color vocabulary becomes whatever you happened to name — much narrower.

### 4. Document non-obvious computations

Pilot insight: when the bundle had `surfaceContainerHighest × 0.3 over surface` for input fill, the project's theme.dart had a comment explaining that decision. The bundle preserved the math instead of opaquely hardcoding. Lesson:

```dart
inputDecorationTheme: InputDecorationTheme(
  // Input fill: surfaceContainerHighest at 0.3 alpha over surface base.
  // This produces a ~6% lift that reads "filled" without competing with cards.
  // Runtime alpha (not precomputed blend) so it adapts when scheme changes.
  filled: true,
  fillColor: scheme.surfaceContainerHighest.withValues(alpha: 0.3),
  // ...
),
```

When Claude Design reads this with the comment, it knows to mark the bundle's fill calculation as an intentional alpha (which then signals flutter-agent to keep runtime resolution per `claude-design-handoff.md` Rule 1).

### 5. No magic numbers in component themes

```dart
// ❌ HARMFUL
filledButtonTheme: FilledButtonThemeData(
  style: FilledButton.styleFrom(
    minimumSize: const Size.fromHeight(52),  // why 52?
  ),
),

// ✅ GOOD
class _Sizes {
  /// Minimum button height. 52dp = 48dp (Material accessibility minimum) +
  /// 4dp lift for visual prominence on dense forms.
  static const buttonHeight = 52.0;
}

filledButtonTheme: FilledButtonThemeData(
  style: FilledButton.styleFrom(
    minimumSize: const Size.fromHeight(_Sizes.buttonHeight),
  ),
),
```

Claude Design reads the constant + comment and emits the bundle's button geometry as a documented decision. The Dart implementation later cites `from FilledButtonTheme in .dart` (pilot showed this) — round-trip closure.

## What Claude Design Does NOT Need (don't over-engineer)

- Motion tokens (durations, curves) — Material defaults work fine; bundle picks defaults too.
- Extended color palettes (success/warning/info beyond `error`) — add them when the project has actual UI for them, not preemptively.
- Custom typography with bespoke font ramps — use Material's TextTheme defaults plus `Theme.of(context).textTheme` overrides only when needed.
- A complete component library before any screen exists — minimum theme + the screens themselves are enough.

The MVP set above is genuinely enough. Don't gate Claude Design on a "complete design system" — that's how teams stall.

## Updating Tokens After a Claude Design Run

If a screen's bundle reveals that your tokens are missing a primitive (e.g., the bundle had to invent a "subtle divider" tone because your scheme only had `outline` and `outlineVariant`), that's signal:

1. Decide if the new primitive belongs in the system (often: yes, the screen needs a third divider tone).
2. Add it to `theme.dart` with a comment explaining when to use it.
3. Update the relevant screen to consume the project token instead of the bundle's invented value.
4. Future Claude Design runs will pick up the new token from your file.

This is the feedback loop: design tokens evolve in response to real screens, not in a vacuum.

## Project Layout

```
flutter/lib/app/
  theme.dart                 # Canonical entry point; lightTheme + darkTheme exports
  ─ (optionally) colors.dart, typography.dart, spacing.dart imported by theme.dart

web/{admin,public}/
  tailwind.config.ts         # Canonical entry point; theme.extend block
  ─ (optionally) tokens.ts imported by tailwind.config.ts
```

Whatever the file layout, the rule is: **one obvious canonical entry point per platform**. Claude Design reads that one file and infers everything from there.

## Checklist Before Running `/design-screen` for the First Time

- [ ] `flutter/lib/app/theme.dart` (or `tailwind.config.ts`) exists and is the canonical entry.
- [ ] Top-of-file comment block describes spacing scale, radii rules, motion approach.
- [ ] `ColorScheme.fromSeed` (Flutter) or matching Tailwind palette config defines the full palette from a seed.
- [ ] Both light and dark themes defined.
- [ ] Component themes defined for at least `FilledButtonTheme` and `InputDecorationTheme`.
- [ ] No hex literals outside the seed color definition (or, for Tailwind, outside the theme.extend block).
- [ ] Non-obvious computations (e.g., alpha blends) have comments.

If you can tick all of these, Claude Design will produce token-aligned output. If not, expect bundle drift and more friction during integration.
