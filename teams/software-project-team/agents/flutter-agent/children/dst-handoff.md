---
knowledge-base-summary: "Different bundle shape than Claude Design — JSON-based (prototype.json + ds.json + resolved-tokens.json + preview.html + README + integration-notes), zipped via the `/dst-handoff` skill. **Mandatory theme-sync step before screen code:** compare `lib/app/theme.dart` against `ds.json` and rewrite the theme first if they drift, otherwise \"trust project theme\" silently yields stale colors. Same 5 translation rules apply. Worked example inside."
---
# DST Handoff (`/dst-handoff` bundle consumption)

When the `/dst-handoff <prototype>` skill from `design-system-team` fires, it spawns you (flutter-agent) with a zipped bundle and an integration brief. This document describes the **bundle shape**, the **mandatory theme-sync step**, and the **5 translation rules** that turn that bundle into working Flutter code.

> **Two bundle shapes exist.** This file covers the **DST bundle** (design-system-team v0.4.0+). The older Claude Design bundle shape (JSX + `tokens.jsx`) is covered by `claude-design-handoff.md`. Translation rules are the same for both; file structure and parsing differ.

## The DST bundle — what's inside

The bundle is always at `.dst/prototypes/<name>/handoff.zip`. Extract it to a temp directory first:

```bash
mkdir -p /tmp/<prototype>-bundle && unzip -o .dst/prototypes/<name>/handoff.zip -d /tmp/<prototype>-bundle
```

Files you'll find, with their purpose:

| File | Shape | Use |
|---|---|---|
| `README.md` | Markdown | Your entry point. Lists states, new i18n keys, routes, warnings. Read first. |
| `integration-notes.md` | Markdown | Target-specific rules (where files go, which patterns to apply). Flutter variant follows this file's guidance. |
| `prototype.json` | JSON | The prototype state. Contains `frames` (one per state), `actions`, `breakpoints`, `tokens` (a summary map), `target: "flutter"`. |
| `ds.json` | JSON | The **linked Design System** — the source of truth for every color, font, spacing value. Read this to answer "what is ds.palette.brand.primary.value?" — it's `#2D5F3F` (or whatever the DS says). |
| `resolved-tokens.json` | JSON | **Pre-computed** flat map: `{ "ds.palette.brand.primary.value": "#2D5F3F", ... }`. The skill walks `prototype.json` for every `{{ ds.<path> }}` reference and resolves it against `ds.json`. Use this for quick lookups instead of re-implementing path resolution. If a value is `null`, the token didn't resolve — flag it, don't guess. |
| `preview.html` | HTML | Visual reference — open in a browser to see the rendered prototype, state by state. Useful for cross-checking that your Flutter output matches intent. |
| `assets/` | Directory | Any prototype-local assets (logomark SVG, icons, images). Copy these into the Flutter project's asset tree if the screen references them. |

The bundle is self-contained. You don't need to reach back into `.dst/` after extracting — everything needed for the integration is in the zip.

## Step zero — Theme sync (non-negotiable)

**Before writing any screen code, check the project's theme against `ds.json`.**

Open `flutter/lib/app/theme.dart` and compare:

| Theme surface | DS source |
|---|---|
| `ColorScheme.fromSeed(seedColor: ...)` | `ds.palette.brand.primary.value` |
| `textTheme.fontFamily` / `fontFamilyFallback` | `ds.typography.fontFamilies.sans.stack` |
| `textTheme.displayLarge` / `displayMedium` / ... sizes + weights | `ds.typography.scale.*` |
| `visualDensity` / spacing | `ds.spacing.scale` |
| `cardTheme.shape`, `elevatedButtonTheme.shape` radii | `ds.radii.{sm,md,lg,full}` |
| (dark mode) `darkTheme` ColorScheme | `ds.palette.darkMode.*` |

If anything is out of sync, **rewrite `theme.dart` from `ds.json` first**, as a separate logical change. Then proceed to the screen work.

If everything is already in sync, skip the rewrite — note "theme already in sync" in your integration report.

### Why this step exists

Downstream, the screen code will reference `Theme.of(context).colorScheme.primary`. That's correct — it's the "trust project theme over precomputed colors" rule (see Rule 1 below). But the rule only yields right results if `theme.dart` was derived from `ds.json`. Without the sync step, the screen compiles, the widgets render, but the brand color is whatever was seeded in `theme.dart` at scaffold time — not what the DS specifies. The screen looks correct structurally and is visually wrong.

Fixing the theme once per project fixes every screen automatically.

### Minimal sync pattern

When you rewrite `theme.dart` from the DS, keep the existing structure of the file (don't rewrite the file architecture — preserve how helpers and getters are organized) and only swap values:

```dart
// flutter/lib/app/theme.dart — minimal DS-synced skeleton

class AppTheme {
  // Pulled from ds.palette.brand.primary.value
  static const _brandPrimary = Color(0xFF2D5F3F);

  // Pulled from ds.typography.fontFamilies.sans.stack[0]
  static const _fontFamily = 'Inter';

  static ThemeData light() {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: _brandPrimary,
      brightness: Brightness.light,
    );
    return ThemeData(
      colorScheme: colorScheme,
      fontFamily: _fontFamily,
      textTheme: _textTheme(colorScheme),
      cardTheme: CardTheme(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),  // ds.radii.lg
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          minimumSize: const Size.fromHeight(52),  // ds.components.button.sizes.lg.height
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
          ),
        ),
      ),
      visualDensity: VisualDensity.comfortable,
    );
  }

  static TextTheme _textTheme(ColorScheme colors) {
    // Values straight from ds.typography.scale
    return TextTheme(
      displayLarge: TextStyle(fontSize: 48, height: 56/48, fontWeight: FontWeight.w600),
      bodyLarge:    TextStyle(fontSize: 16, height: 24/16, fontWeight: FontWeight.w400),
      // ... etc
    );
  }
}
```

Keep comments next to each value tying it back to the DS path (`// ds.radii.lg`) so future readers know where a number came from. That's the documentation that makes the theme file auditable.

## The 5 translation rules (universal — apply to both bundle shapes)

### Rule 1 — Trust the project theme over precomputed colors

After the theme sync step, `Theme.of(context).colorScheme.primary` IS the DS brand primary. Use theme lookups in the widget, not raw hex from `resolved-tokens.json`.

✅ `color: Theme.of(context).colorScheme.primary`
❌ `color: Color(0xFF2D5F3F)` (bypasses the theme; dark mode / future DS changes don't propagate)

`resolved-tokens.json` is a sanity check + a fallback for tokens that aren't in the Material theme (e.g., bespoke semantic colors the DS defines that Material doesn't express).

### Rule 2 — Map prototype state idioms to Flutter

The prototype's `frames` entries represent React-like state snapshots (idle / submitting / error / loading / disabled). Flutter expresses these differently:

| Prototype state | Flutter idiom |
|---|---|
| `idle` | Default widget render, no loading or error conditions triggered |
| `submitting` | `isLoading = true` local state, inputs `enabled: false`, button shows `CircularProgressIndicator` |
| `error` | `errorMessage != null` local state, banner above form, inputs remain editable so user can retry |
| `loading` (lists, detail pages) | `AsyncValue.loading` (Riverpod) or `FutureBuilder` ConnectionState.waiting |
| `empty` (lists) | `AsyncValue.data` with empty list → render empty-state illustration + CTA |
| `success` | Transient — usually a SnackBar + immediate navigation |

A single widget with `Consumer` + local state typically handles all these branches. Don't build three separate widgets for idle/submitting/error; react to one state.

### Rule 3 — Substitute SVG / Lucide icons with Material equivalents

The prototype may reference SVG icons or a web icon set (Lucide, Heroicons). Flutter has Material icons built in:

| Prototype reference | Flutter |
|---|---|
| `mail-outline`, `lucide:mail` | `Icons.mail_outline` |
| `lock-outline` | `Icons.lock_outline` |
| `eye` / `eye-off` | `Icons.visibility` / `Icons.visibility_off` |
| `error-outline` | `Icons.error_outline` |
| `check-circle` | `Icons.check_circle` |
| custom SVG | Copy asset → `SvgPicture.asset(...)` using `flutter_svg` (add dep if not present) |

Never import raw SVG strings from the prototype. Use Material icons or the `flutter_svg` package.

### Rule 4 — Extract every user-facing string to ARB keys

The prototype's frames contain text content directly (`{"text": "Welcome back"}`). Flutter's i18n contract requires every such string goes through `AppLocalizations`.

For each string in the prototype:

1. Choose a key name in the format `{feature}_{element}` (e.g., `login_title`, `login_submit`, `login_error_generic`).
2. Add the key + value to `flutter/lib/l10n/app_en.arb`. Add an `@key` entry with a `description` describing where it's shown.
3. Mirror the key to `flutter/lib/l10n/app_tr.arb` (or whichever locales the project already has) with a translated value.
4. Reference from Dart via `context.l10n.login_title` (the extension lives at `lib/core/localization/app_localizations_extension.dart`).
5. After your changes, note that the user must run `flutter gen-l10n` — don't run it yourself.

**Turkish values:** if the prototype ships Turkish content via `{{ ds.voice.samples.* }}`, resolve that token from `ds.json` and use the voice-sample text. Otherwise translate the English value into neutral formal Turkish (prefer ASCII-safe transliteration if the project ARB files avoid non-ASCII; otherwise use proper Turkish characters).

### Rule 5 — Geometry from bundle, behavior from project

When the bundle says the button is 52px tall with 16px bottom-margin and 12px corner radius, use those numbers (they come from the DS). That's GEOMETRY.

When the button submits, what's the API call? Where does the screen navigate on success? What's the error-handling contract (ProblemDetails resolve via `messageResolver`)? Those come from the PROJECT's existing conventions — `AuthNotifier`, `AuthRepository`, `go_router` routes, `context.go('/home')`, etc.

**Never invent behavior the project doesn't already express.** If the prototype shows a forgot-password link, but `/forgot-password` isn't in the router, leave a `TODO(forgot-password)` comment and a no-op handler. Don't wire up a half-baked flow.

## Worked example — what a good integration looks like

Given: a `splash-screen` bundle targeting flutter. Prototype has 1 `idle` frame with a centered logomark, wordmark, tagline, and full-width "Continue" CTA that advances to `/login`.

```
1. Extract /tmp/atl-fresh-myapp/.dst/prototypes/splash-screen/handoff.zip
   to /tmp/splash-bundle/
2. Read bundle README.md + integration-notes.md
3. Compare lib/app/theme.dart vs ds.json:
   - seed: current theme #4F46E5 vs ds #2D5F3F → OUT OF SYNC
   - fontFamily: 'Roboto' vs 'Inter' → OUT OF SYNC
   → Rewrite theme.dart: seed = #2D5F3F, fontFamily = 'Inter', radii match ds.radii.
   → One Edit to theme.dart, commit-level separation: "theme: sync to ds primary v1.0.0"
4. Write lib/features/splash/splash_screen.dart:
   - StatelessWidget, Scaffold > SafeArea > Center > Column
   - Logomark (placeholder if asset absent — Container with 'M')
   - Text context.l10n.splash_title (wordmark, uses Theme textTheme.displayMedium)
   - Text context.l10n.splash_tagline (uses textTheme.bodyLarge, onSurfaceVariant color)
   - ElevatedButton onPressed: () => context.go('/login'), child: Text(context.l10n.splash_cta_continue)
   - GestureDetector wrapping body for tap-anywhere = advance
5. Update lib/app/router.dart:
   - Import SplashScreen
   - Change initialLocation to '/splash'
   - Add GoRoute '/splash' builder: SplashScreen
   - Extend redirect: authenticated users on '/splash' → '/home'
6. Append keys to lib/l10n/app_en.arb + app_tr.arb:
   - splash_title, splash_tagline, splash_cta_continue
7. Report:
   - "Theme synced: seed #4F46E5 → #2D5F3F, fontFamily Roboto → Inter"
   - Files: lib/features/splash/splash_screen.dart (new), lib/app/router.dart (patch),
     lib/app/theme.dart (patch), lib/l10n/app_en.arb + app_tr.arb (add 3 keys).
   - Next: cd flutter && flutter pub get && flutter gen-l10n && flutter run
```

That's the full shape. No handwaving, no invention.

## Anti-patterns

❌ Hardcoding `Color(0xFF2D5F3F)` anywhere in the widget — goes against Rule 1.

❌ Skipping the theme-sync step because "I'll fix it next time" — silently ships the wrong brand color.

❌ Building `SplashScreenIdle`, `SplashScreenSubmitting`, `SplashScreenError` as three separate widgets — pick one widget, branch on state.

❌ Hardcoding English strings in JSX-like Dart structure (`Text('Continue')`) — goes against Rule 4.

❌ Running `flutter pub get` / `flutter gen-l10n` / `flutter run` yourself — the user does that (you print the commands for them).

❌ Inventing new dependencies not in `pubspec.yaml` — if the prototype needs `flutter_svg` for a logomark, add it conservatively (or suggest it in the report), don't silently pad dependencies.

## When the bundle is ambiguous

The prototype might describe a layout the Flutter platform doesn't express cleanly (web-only affordances, hover states, cursor-specific behaviors). In those cases:

1. Prefer the Flutter-native equivalent (e.g., hover → press-and-hold, cursor-specific → no-op).
2. Note the translation in your integration report under "Deviations from prototype".
3. Don't block on ambiguity — make the best-effort call and document it.

## What this file does NOT cover

- **Claude Design bundle shape** — see `claude-design-handoff.md` (the older tool's bundle format with `tokens.jsx` + `components/*.jsx`).
- **react-agent DST bundles** — react-agent has its own `children/dst-handoff.md` covering React + Tailwind + i18next equivalents.
- **Creating the DST bundle** — that's `design-system-team`'s `/dst-handoff` skill responsibility, not flutter-agent's concern.
