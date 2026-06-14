---
knowledge-base-summary: "React+HTML bundle from Claude Design (via `/design-screen done`) gets translated to Flutter. Five rules: trust project theme over precomputed colors, map React state to FocusNode/AsyncValue/etc., substitute SVG icons with M3 equivalents, extract new ARB keys + gen-l10n, take geometry from bundle but behavior from project. Write engineer notes; document deltas honestly (4/5 fidelity is realistic)."
---
# Claude Design Handoff → Flutter

When `/design-screen done` calls flutter-agent with an extracted Claude Design bundle, this file is the translation playbook. It captures every pattern the reference-project login pilot (2026-04-19) surfaced as friction, plus the workflow rules.

## What's In the Bundle

A typical bundle (`.claude/design/{slug}/{date}-v{N}-extracted/`) contains:

```
README.md                                  # Claude Design's own brief
components/
  tokens.jsx                               # Self-documenting token reference
  {screen-name}.jsx                        # The screen as a React component
frames/
  iOS-shell.png  (or similar)              # Visual reference renders
  Android-shell.png
```

Plus a Claude Code curl command in the README — that's the Anthropic-hosted fetch URL for the same bundle. We use it to pull the bundle, not to drive implementation.

**The bundle is React+HTML.** There is NO Dart in it. Translation is your job. The bundle is a *high-fidelity visual + token specification*, not a code drop-in.

## The Five Translation Rules

### Rule 1 — Trust the project theme, NOT the bundle's precomputed colors

The bundle pre-computes blended colors (e.g., `surfaceContainerHighest × 0.3 over surface` produces a fixed hex). Flutter does this at runtime via `withValues(alpha:)`. **Use Flutter's runtime resolution.** A ~1% subpixel difference is acceptable and unavoidable.

```dart
// ❌ BAD — porting the bundle's precomputed hex
fillColor: Color(0xFFE8E2D9),

// ✅ GOOD — runtime alpha on the project's token
fillColor: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
```

**Why:** the project's `theme.dart` is the source of truth. If theme tokens change, the runtime expression updates automatically; the precomputed hex would silently drift.

### Rule 2 — Map React state shapes to idiomatic Flutter, do NOT mimic

Claude Design ships React state like `focused: 'email' | 'password' | null`, `showPassword: bool`, `apiError: string | null`. Resist the urge to mirror these one-to-one.

| React in bundle | Flutter idiom |
|-----------------|---------------|
| `focused: string` | `FocusNode` per field; check `_emailFocus.hasFocus` |
| `showPassword: bool` | `obscureText: !_showPassword` on `TextFormField` |
| `apiError: string \| null` | `FormFieldState`'s `errorText` OR a dedicated banner widget driven by `AsyncValue.error` |
| `loading: bool` | `AsyncValue<T>` from a Riverpod provider; UI reads `value.isLoading` |
| Form submit handler | `Form.of(context).validate()` + Riverpod notifier method |

The bundle's state shape is a visual specification — what gets shown, when. Translate the *intent*, not the *plumbing*.

### Rule 3 — Icons are visual references, not literal assets

The bundle ships custom SVGs (mail, lock, eye, eye-off, alert-triangle, etc.). These are visual hints, not assets to import.

**Default approach:** use the nearest Material 3 icon from the project's icon family.

| Bundle SVG | M3 equivalent |
|------------|---------------|
| Mail | `Icons.mail_outline` |
| Lock | `Icons.lock_outline` |
| Eye | `Icons.visibility_outlined` |
| Eye-off | `Icons.visibility_off_outlined` |
| Alert / warning | `Icons.error_outline` |
| Check | `Icons.check_circle_outline` |
| Search | `Icons.search` |
| Close / X | `Icons.close` |
| Arrow back | `Icons.arrow_back` |

**Exception:** if the SVG carries brand identity (a custom logo/glyph that no M3 icon approximates), import it as `SvgPicture.asset` (requires `flutter_svg` in pubspec) — copy the SVG into `flutter/assets/icons/` and register the asset path.

When in doubt, ask the user. Don't invent custom assets silently.

### Rule 4 — ARB key extraction is part of the integration

Bundles often introduce new user-facing strings the project doesn't have ARB entries for yet (the pilot needed `login_subtitle`, `login_register`, `login_register_cta`, `login_forgot`, `lang_tooltip`).

**Workflow:**
1. As you write the screen, collect every new user-facing string as you go.
2. Add each key to BOTH `lib/l10n/app_en.arb` AND `lib/l10n/app_tr.arb` per the i18n contract (`api-agent/children/user-facing-strings.md`).
3. Use the `{feature}_{element}_{variant}` naming convention from the i18n doc.
4. Run `flutter gen-l10n` (or trigger a build that does) to regenerate `AppLocalizations`.
5. Reference the new keys in the screen via `context.l10n.{key}`.

**CI gate:** if the project doesn't already have `flutter gen-l10n` in its pre-build steps, mention it in the integration summary. (Pilot caught this as a "ship-prep" item.)

### Rule 5 — Geometry from bundle, behavior from project

Take from the bundle:
- Vertical rhythm (e.g., 28-6-28-20-8-20 spacing stack from the pilot)
- Component dimensions (button height, input height, icon sizes)
- Border radii
- Layout structure (column/row ordering, alignment)

Take from the project:
- Animation curves (use the project's `Theme` motion specs, not the bundle's hardcoded React animations)
- Routing (`go_router` per `flutter-agent/children/routing.md`)
- State management patterns (Riverpod per `state-management.md`)
- Error handling patterns (`AsyncValue` per `error-loading-states.md`)
- Form validation (per `form-handling.md`)

The bundle says "what it looks like." The project's existing children files say "how it behaves."

## Workflow When Briefed by `/design-screen done`

The skill calls you with a self-contained brief. Expected shape:

```
intent: <intent.md path or summary>
extracted bundle: <path>
project conventions to honor: <pointer to this file + relevant existing children>
specific frictions known in advance: <e.g., new ARB keys expected>
```

Your steps:

1. **Read the intent.md** (path provided in brief). Get the full Phase A context — verbatim user requirements, original prompt, copy decisions, target platform.
2. **Open the extracted bundle**. Read `README.md`, then `components/tokens.jsx`, then the screen `.jsx` file.
3. **Cross-reference tokens.jsx with `flutter/lib/app/theme.dart`** — confirm the bundle's token names map to project tokens. If `tokens.jsx` says "derived from flutter/lib/app/theme.dart" (Claude Design typically self-documents this), trust it.
4. **Identify new ARB keys** the screen will need. Add them to both `app_en.arb` and `app_tr.arb` BEFORE writing the screen — this prevents "key not found" loops during hot reload.
5. **Write the screen** following the five translation rules above. Use existing children files (`screen-blueprint.md`, `routing.md`, `form-handling.md`, etc.) for the structural patterns.
6. **Wire routing.** If the screen needs new routes (e.g., bundle had Forgot/Register links pointing nowhere), add them to `go_router` config OR mark them with TODOs in the integration summary so they're explicit.
7. **Write a brief engineer note** at `.claude/design/{slug}/pilot-notes.md` (or `integration-notes.md` for non-pilot work). Capture: friction encountered, decisions taken (e.g., "swapped custom mail SVG for `mail_outline`"), open TODOs (e.g., "Forgot password route empty"), files touched.

## Anti-Patterns

### Don't port the bundle's React inputs verbatim
The bundle often re-implements input chrome (focus rings, floating labels, fill states) because React doesn't have `InputDecorationTheme`. Flutter does — use it via `Theme.of(context).inputDecorationTheme`. Don't write a custom `TextField` clone to match the bundle's React structure.

### Don't import the bundle's CSS-in-JS layout values as raw numbers
If the bundle has `padding: 24px`, look up your project's spacing scale. It's probably `16` or `20` from a 4px grid. Match the spirit, not the digit. Exception: tight numeric specs from the bundle ARE worth following when they're carrying intentional vertical rhythm (the pilot's 28-6-28-20-8-20 stack was load-bearing).

### Don't promise the user "1:1 visual match"
Realistic fidelity is 4/5. Document the deltas (brand glyph differences, animation curves, ~1% color blends) in your engineer note so the user isn't surprised on visual review.

### Don't skip the engineer note
Even if integration is smooth, write the note. Six months from now, "why does this screen look 95% like the bundle but not 100%?" is answered by your note.

## Quick Reference Card

```
Bundle says               You write
────────────────────────  ─────────────────────────────────────
Custom SVG icon           Material 3 nearest equivalent
                          (or SvgPicture.asset for brand glyphs)

React state slots         FocusNode / obscureText / FormFieldState /
                          AsyncValue / Riverpod notifier

Precomputed color blend   Project token + .withValues(alpha:)

Component-level CSS       InputDecorationTheme / FilledButtonTheme /
                          existing project theme primitives

Hardcoded layout values   Project spacing scale (4px grid)
                          (UNLESS load-bearing rhythm — keep then)

New string in mockup      ARB key in both en + tr; gen-l10n; context.l10n
```
