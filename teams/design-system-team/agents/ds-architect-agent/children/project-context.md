---
knowledge-base-summary: "Before generating, read project tokens (Flutter `theme.dart`, web `tailwind.config.ts`). How to map them into `ds.json` cleanly. When to seed from project vs. propose fresh."
---
# Project Context Reading

Before generating a new design system, scan the project for existing token sources. If found, use them as the seed instead of inventing fresh values.

## Files to look for

In order of priority:

1. **`flutter/lib/app/theme.dart`** — Flutter project's ThemeData. Pure source of truth for color/typography/spacing if present.
2. **`web/tailwind.config.ts`** or **`apps/admin/tailwind.config.ts`** or **`tailwind.config.js`** — web token source.
3. **`web/src/styles/tokens.ts`** or similar — explicit token modules.
4. **`pubspec.yaml`** — Flutter brand inferences (app name, description).
5. **`package.json`** — Project name, description, dependencies (font usage signals).
6. **`README.md`** — May describe brand explicitly.
7. **`.dst/design-systems/`** — Existing DSes in the project (use as inspiration if user creating a new one).

## What to extract

From `theme.dart`:
- `seedColor` (the brand seed for `ColorScheme.fromSeed`)
- `colorScheme` overrides (any explicit role-mapped colors)
- `textTheme` (typography ramp, fonts)
- `inputDecorationTheme`, `filledButtonTheme`, etc. (component conventions)

From `tailwind.config.ts`:
- `theme.extend.colors` (palette)
- `theme.extend.fontFamily` (fonts)
- `theme.extend.spacing` (custom spacing if any)
- `theme.extend.borderRadius` (radii)

## Mapping to `ds.json`

Project tokens should populate `ds.json` with the project's exact values, not your interpretations. Specifically:

```dart
// flutter/lib/app/theme.dart
ColorScheme.fromSeed(seedColor: Color(0xFF2D5F3F))
```

→ becomes:

```json
"palette": {
  "brand": {
    "primary": { "value": "#2D5F3F", "name": "Brand Primary", "source": "flutter/lib/app/theme.dart" }
  }
}
```

Note the `source` field — provenance metadata so future agents (and users) know where the value came from.

## When tokens conflict

If user has `theme.dart` AND `tailwind.config.ts`, they may not match perfectly. In that case:
- Prefer the **most specific to the new DS's target platform**. If user said "this DS is for the mobile app," use `theme.dart`.
- Surface the conflict in the Q&A: "Your Flutter theme uses #2D5F3F primary, but Tailwind uses #2C5E3E. Which one is canonical?"

## When no tokens exist

Greenfield project — generate from scratch:
- Q&A: ask for personality keywords, brand color preference (or "auto-pick from personality")
- Use defaults from `palette-theory.md`, `typography-ramps.md`, etc.
- Note in `ds.json`: `"source": "generated"` for transparency.

## Project name → DS name relationship

If the project is named `example-app` and the user is creating their first DS, suggest naming it `primary` or `example-primary`. Don't force a naming pattern, but offer one.

If a project has multiple apps (customer + admin), suggest one DS per app:
- `example-primary` (customer)
- `example-admin` (admin panel)

## Reading flow

```
1. Glob for theme.dart, tailwind.config.ts, pubspec.yaml, package.json
2. For each found file: Read + extract relevant fields
3. Synthesize a "project context summary" (mental model — not a file)
4. Use it during Q&A:
   - "I see your Flutter theme has Forest Green (#2D5F3F). Use this as brand?" [Yes/No]
   - "Your tailwind.config has Inter for body. Use Inter in this DS?" [Yes/No]
5. Defaults gracefully fall back to typography-ramps.md / palette-theory.md when no project signal exists.
```

## What NOT to do

- ❌ Silently overwrite project values with your interpretations
- ❌ Generate without checking for project context first (you'll produce a DS that drifts from the code)
- ❌ Pick brand colors without consulting the user when project has none — always ask
- ❌ Ignore `pubspec.yaml`'s app name when naming the DS (use it as a default)
