---
name: dst-handoff
description: "/dst-handoff <prototype-name> [--target flutter|react-admin|react-public] — bundle a prototype + its linked design system into a handoff package and brief the appropriate code agent (flutter-agent / react-agent) to integrate into the project's actual source code. One command from design to code."
argument-hint: "<prototype-name> [--target flutter|react-admin|react-public]"
---

# /dst-handoff Skill

## Purpose

Take a finished prototype from `.dst/` and integrate it into the project's actual source code (Flutter app, React admin, or React public site) via the appropriate code agent. Produces a self-contained handoff bundle and briefs the implementation agent with everything it needs — no additional context required from `.dst/`.

One command from design to code. No more manual copy/paste of prototype artifacts into agent briefs.

## Preconditions

- `.dst/` exists in the current project (run `/dst-init` first if not).
- The target prototype exists at `.dst/prototypes/<prototype-name>/`.
- The prototype's `linkedDs` directory exists at `.dst/design-systems/<linkedDs>/`.

## Flow

### Phase 1 — Validate

1. `.dst/state.json` exists? If not → error: "Run `/dst-init` first."
2. `<prototype-name>` arg provided? If not → error: "Usage: /dst-handoff <prototype-name> [--target flutter|react-admin|react-public]."
3. `.dst/prototypes/<prototype-name>/` exists? If not → error: "Prototype '<name>' not found. Available: …" (list from `state.json.prototypes`).
4. Read `.dst/prototypes/<name>/prototype.json`. Check `linkedDs`.
5. `.dst/design-systems/<linkedDs>/ds.json` exists? If not → error: "Prototype is linked to DS '<linkedDs>' but that DS is missing. Restore it or re-link with /dst-edit-prototype first."

### Phase 2 — Determine target

Resolve the target platform in this order:

1. **`--target` flag** — if passed on the command line, use it directly.
2. **`prototype.json.target`** — set during `/dst-new-prototype` Q0.
3. **Project scan** — if neither above:
   - `flutter/pubspec.yaml` or `./pubspec.yaml` exists → default `flutter`.
   - Else `package.json` with `react` in deps and an `admin/`/`web/` folder → default `react-admin`.
   - Else `package.json` with `next` in deps → default `react-public`.
4. **Ask** — if still ambiguous (multiple plausible targets, or none match), use `AskUserQuestion`:

   > "Which code target should we integrate into?"
   > - flutter — Flutter mobile app
   > - react-admin — React + TypeScript admin (back-office)
   > - react-public — Next.js marketing / public site

Record the chosen target for this run.

### Phase 3 — Build handoff bundle

Assemble the bundle at `.dst/prototypes/<name>/handoff.zip`:

```
handoff.zip/
  README.md             ← agent brief: what to integrate, where, how
  prototype.json        ← copy of source prototype state
  ds.json               ← copy of linked DS (full, for context)
  preview.html          ← copy of rendered preview (visual reference)
  assets/               ← any prototype-local assets (copied from .dst/prototypes/<name>/assets/ if present)
  resolved-tokens.json  ← flat key→value map of every DS token this prototype references
  integration-notes.md  ← target-specific guidance
```

#### Build steps

1. **Stage a temp directory** (e.g., `.dst/prototypes/<name>/.handoff-build/`). Clean first if it exists.
2. **Copy files:**
   - `.dst/prototypes/<name>/prototype.json` → `prototype.json`
   - `.dst/design-systems/<linkedDs>/ds.json` → `ds.json`
   - `.dst/prototypes/<name>/preview.html` → `preview.html`
   - `.dst/prototypes/<name>/assets/` → `assets/` (recursive, if directory exists)
3. **Compute `resolved-tokens.json`:**
   - Scan `prototype.json` recursively for every `{{ ds.<path> }}` token reference (including `| default: '...'` fallback forms).
   - For each, resolve against `ds.json` by walking the dot path.
   - Produce `{ "ds.palette.brand.primary.value": "#2D5F3F", "ds.typography.scale.body.size": "16px", ... }`.
   - If a token reference doesn't resolve → record it with value `null` and add a warning line to `README.md` (see below). Do not abort the handoff — the agent can still integrate; it just has to fall back to the literal default or ask.
4. **Render `README.md`** (see template below).
5. **Render `integration-notes.md`** (see per-target templates below).
6. **Zip:** create `.dst/prototypes/<name>/handoff.zip` from the temp directory. Overwrite if it exists (handoffs are always fresh).
7. **Clean:** remove the temp directory after zipping.

#### `README.md` template (inside the bundle)

```markdown
# Handoff: <prototype.displayName>

**Target**: <target>
**Linked DS**: <linkedDs> (v<ds.version>)
**Prototype version**: v<prototype.version>
**Source**: .dst/prototypes/<name>/
**Integration date**: <ISO timestamp>

## What to do

Implement this prototype in the project's <target> codebase, following the
project's existing conventions.

- The full prototype state is in `prototype.json`.
- The linked design system is in `ds.json` — every visual value MUST trace to
  a token in `ds.json` (no hardcoded hex, no arbitrary px).
- `resolved-tokens.json` is a pre-computed key→value map for every token
  reference the prototype uses, so you don't have to re-implement resolution.
  (If a value is `null` the token didn't resolve against the current ds.json —
  flag it in your integration report.)
- `preview.html` is the visual reference: open it in a browser to see what the
  final result should look like, state by state.
- `integration-notes.md` contains target-specific guidance (conventions to
  follow, rules to apply).

### For Flutter targets

See flutter-agent's children/claude-design-handoff.md for the React→Flutter
translation rules (trust project theme over precomputed colors, map React state
idioms, substitute icons, extract ARB keys, take geometry from bundle / behavior
from project conventions). Apply them.

### For React targets

See react-agent's children/claude-design-handoff.md for the bundle adaptation
rules (Zustand state mapping, Tailwind utilities, react-router integration,
i18next extraction).

## States to render

<for each state in prototype.json.frames:>
- **<state name>** — <frame.label>: <frame.description>

## New i18n keys to add

<scan prototype.json for string literals that should become localized keys
per the project's i18n contract (see api-agent/children/user-facing-strings.md
if software-project-team is installed). List them here, one per line, with
the source text:>
- `<key>` — "<English source>"

<if no strings detected:>
No new localized strings detected in this prototype.

## Routes to wire

<if prototype.actions includes navigation-related entries, list them:>
- `<action>` → route expected by the prototype (destination to be defined in the app)

<if none:>
No new routes required by this prototype.

## Warnings

<if any unresolved tokens were recorded:>
- Token `<path>` did not resolve against ds.json — fall back to the literal
  default (if any) or flag in the integration report.

<if none:>
No warnings.
```

#### `integration-notes.md` — per-target content

**flutter**

```markdown
# Integration notes — Flutter

## Where things should go

- Screen widget → `flutter/lib/features/<feature>/<screen>_screen.dart`
  (or match the project's existing feature folder layout).
- Route declaration → the project's router file (usually `flutter/lib/app/router.dart`
  using `go_router`).
- New ARB keys → `flutter/lib/l10n/app_en.arb` and `flutter/lib/l10n/app_tr.arb`
  (or whichever locales the project already ships). Run `flutter gen-l10n` after.
- Tokens → the project's theme (usually `flutter/lib/app/theme.dart`). If the
  project theme already covers a token, prefer `Theme.of(context)` lookups over
  reading `ds.json` literal values at runtime — the theme is the source of truth
  inside the app; `ds.json` is the design contract.

## Rules

1. Trust project theme over precomputed colors when Flutter types exist (e.g.,
   `colorScheme.primary`). Use `resolved-tokens.json` values only as a sanity
   check / for cases where the theme doesn't cover the token.
2. Map React state idioms to Flutter: `FocusNode`, `FormFieldState`, `AsyncValue`
   (if the project uses Riverpod) or `BlocBuilder` (if it uses bloc).
3. Substitute SVG/Lucide icons with Material 3 equivalents (`mail_outline`,
   `lock_outline`, `error_outline`, etc.) — never import raw SVGs unless the
   project already does.
4. Extract every user-facing string to ARB keys. Never hardcode localized text.
5. Take geometry (spacing, sizes, corner radius, typography scale) from the
   bundle; take behavior (navigation, state management, error handling) from
   the project's existing conventions.

## Don't do

- Don't auto-run `flutter pub get`, `flutter gen-l10n`, or `flutter run`. Print
  them as next-step commands for the user.
- Don't introduce new dependencies unless the prototype genuinely requires them
  — prefer what's already in `pubspec.yaml`.
```

**react-admin**

```markdown
# Integration notes — React admin

## Where things should go

- Page component → per the project's conventions (often `src/pages/<feature>/...`
  or `admin/app/<feature>/page.tsx`).
- Route declaration → the project's router (react-router v6 routes file, or
  Next.js file-based routing if applicable).
- i18n keys → the project's i18next resource bundle (e.g., `src/i18n/en.json`,
  `src/i18n/tr.json`).
- Tokens → Tailwind config (`tailwind.config.ts`) if the project uses Tailwind;
  CSS variables otherwise. Prefer the project's existing token layer over
  reading `ds.json` at runtime.

## Rules

1. Map React state idioms to the project's conventions (Zustand store, Redux
   slice, or React Query — whichever the codebase uses).
2. Use the project's component library (if any — shadcn/ui, MUI, Chakra, etc.)
   before introducing raw elements. Prefer existing `<Button />` etc. over
   re-styling HTML buttons.
3. Use `className` with Tailwind utility classes when the project uses Tailwind;
   otherwise match the project's styling convention (CSS modules, styled-components,
   etc.).
4. Extract every user-facing string to i18n keys.
5. Take geometry from the bundle; take routing + state from the project.

## Don't do

- Don't auto-run `npm install`, `npm run dev`, or tests. Print commands for the user.
- Don't introduce new deps unless required.
```

**react-public**

```markdown
# Integration notes — React public / Next.js

## Where things should go

- Page/route → Next.js App Router (`app/<path>/page.tsx`) or Pages Router
  (`pages/<path>.tsx`), matching the project layout.
- SEO metadata (title, description, og-tags) → Next.js `metadata` export or
  `<Head>` component; derive from prototype.json.description when applicable.
- i18n → next-intl / next-i18next / whatever the project uses.
- Tokens → Tailwind config or CSS variables; prefer existing theme layer.

## Rules

1. Server components by default for static content; client components only when
   interactivity requires it (forms, animations, state).
2. Image assets → Next.js `<Image>` with proper sizing; use `assets/` from the
   bundle.
3. Marketing pages usually prioritize performance + SEO — keep bundle size lean,
   avoid unnecessary client-side JS.
4. Extract user-facing strings per the project's i18n convention.

## Don't do

- Don't auto-run `npm run build`, `npm run dev`. Print commands for the user.
- Don't introduce analytics, A/B testing, or tracking unless the prototype specifies it.
```

### Phase 4 — Invoke target agent

Determine the agent name from the target:

- `flutter` → `flutter-agent`
- `react-admin` → `react-agent` (or `react-admin-agent` if the project uses the per-variant naming)
- `react-public` → `react-agent` (or `react-public-agent`)

Pick whichever agent name exists in the current project's team install. If neither variant exists, error out:

> "No code agent found for target '<target>'. Install `software-project-team` (or a team that provides flutter-agent / react-agent) to use /dst-handoff with this target."

Spawn the agent using the Agent tool with this self-contained brief:

```
You are integrating a design-system-team prototype into this project's
<target> codebase.

The handoff bundle is at: .dst/prototypes/<name>/handoff.zip

Extract it and follow handoff/README.md — it describes what to build,
where to put it, and which rules apply. integration-notes.md inside the
bundle gives target-specific conventions.

## Step A — Sync the project theme to the DS (before touching any screen)

**This is step zero for every integration.** Before writing any widget /
component code for the screen, make sure the project's global theme layer
is derived from ds.json. Otherwise the "trust project theme over
precomputed colors" rule downstream silently yields stale colors.

Check the project's theme file for the target:

- Flutter: `flutter/lib/app/theme.dart`
  (`ColorScheme.fromSeed`, `TextTheme`, `ThemeData.visualDensity`, etc.)
- React admin: `web/admin/tailwind.config.ts` OR `web/admin/src/styles/tokens.css`
  (CSS variables / Tailwind theme.extend)
- React public: same as react-admin — project-specific config file

Compare the theme's seed / token values against `ds.json`:

- `brand.primary.value` vs theme's primary seed
- `typography.fontFamilies.sans.stack` vs theme fontFamily
- `spacing.scale` vs theme spacing scale
- `radii` vs theme cornerRadius / borderRadius
- (dark mode) `palette.darkMode.*` vs theme darkTheme

If ANY of those drift, the theme file is out of sync with the DS.
Rewrite the theme file from ds.json FIRST, as a separate logical change
(separate commit, if the project is git-tracked). Then proceed to the
screen work. Explain this in your integration report.

If the theme is already in sync (DS was created and theme was derived from
it at scaffold-time), skip rewrite — note "theme already in sync" in the
report and move on.

**Why this matters:** without this step, the integration looks "done" but
renders in the wrong brand colors. The screen code references
`Theme.of(context).colorScheme.primary` correctly — but the theme itself
lies about what that value is. Fixing the theme once per project fixes all
screens automatically.

## Step B — Integrate the screen

After the theme is in sync, implement the screen per the bundle's README +
integration-notes. Follow:

- The project's existing conventions (read CLAUDE.md,
  .atl/docs/coding-standards/ if present).
- The linked DS's tokens (every visual value references ds.json — never
  hardcode hex or px).
- The project's i18n contract (extract strings to ARB / i18next per the
  project's user-facing-strings contract).
- Token-fidelity guarantee: never replace a token reference with a
  hardcoded value. If a token doesn't exist in the DS, flag it back —
  don't invent.

## Reporting

When done, report:
- **Theme sync status** — did you rewrite the theme file (and which fields), or was it already in sync?
- Files changed (full paths relative to project root)
- New i18n keys added (and in which resource files)
- Routes added or expected
- Any deviation from the prototype (with reason)
- Next-step commands the user should run (e.g., `flutter pub get &&
  flutter gen-l10n`, `npm install`, etc.)
```

### Phase 5 — Capture the agent report

Parse the agent's reply for:

- **files_changed** — list of paths
- **i18n_keys_added** — list of keys
- **routes_added** — list of route names
- **deviations** — short text
- **next_commands** — shell lines the user should run

If the agent didn't provide a structured section for one of these, treat it as empty.

### Phase 6 — Append integration record to `prototype.json.notes`

Open `.dst/prototypes/<name>/prototype.json`, append one entry to `notes[]`:

```
"<ISO timestamp> — integrated to <target>. Files: <comma-separated short paths>. i18n: <count> keys. Routes: <list>. Deviations: <one-line>."
```

Bump `prototype.json.lastModified`. Save. This keeps an integration history on the prototype itself.

Also update `state.json`'s matching `prototypes[]` entry's `lastModified` to the same ISO timestamp.

### Phase 7 — Print summary

```
✓ Integrated <prototype-name> into <target>
  Bundle: .dst/prototypes/<prototype-name>/handoff.zip (<size> KB)

  Files changed:
    <one per line>

  <if i18n keys added:>
  i18n keys added:
    <one per line, per resource file>

  <if routes added:>
  Routes:
    <one per line>

  <if deviations:>
  Deviations from prototype:
    <one-line summary>

Next:
  <next-step commands, as shell lines>

  Edit the design via: /dst-edit-prototype <name> "<change>"
  Re-integrate after design changes: /dst-handoff <name>
```

## Important rules

1. **The handoff bundle is the contract.** flutter-agent / react-agent should be able to integrate from JUST the bundle — no reading `.dst/` directly. This keeps the handoff portable across project boundaries (and legible to humans reviewing a PR).
2. **`resolved-tokens.json` is helpful, not load-bearing.** It pre-resolves tokens for convenience. If it's somehow missing or stale, the agent falls back to resolving against `ds.json` directly. The canonical source is always `ds.json` + `prototype.json`.
3. **Never auto-run build commands.** No `flutter gen-l10n`, `npm install`, `npm run build`, etc. Print them for the user — build invocations are project-specific (monorepos vary), and we don't want to assume.
4. **Fresh bundle every run.** If `handoff.zip` already exists, overwrite it without asking. It's always regenerated from current state.
5. **Don't push to git.** This skill doesn't commit or push anything. The user reviews the agent's changes and commits themselves.
6. **One prototype per run.** Handing off multiple prototypes = one command per prototype (keeps the agent briefs focused; avoids interleaved diffs).

## Edge cases

- **Prototype has no `linkedDs` value** (legacy / pre-v0.4.0 state) — error: "Prototype has no linked DS; use /dst-edit-prototype to set one, then re-run."
- **Target agent not installed** — error with install instructions, as above.
- **Handoff bundle build fails mid-way** — clean up the temp dir; don't leave partial `handoff.zip` files.
- **Agent run cancelled mid-integration** — bundle still exists in `.dst/` (it's a reusable artifact); no code changes on disk; `prototype.json.notes` untouched (only written on success).
- **Agent reports no files changed** — still update `prototype.json.notes` with the run record (deviations: "no integration occurred — <reason>") so the history is complete.

## What this skill does NOT do

- Doesn't edit the prototype or DS (use `/dst-edit-prototype`, `/dst-edit-ds`).
- Doesn't open the browser.
- Doesn't install target agents (install them via `atl install software-project-team` or equivalent).
- Doesn't commit or push code changes (user reviews + commits themselves).
- Doesn't remove old integration artifacts from the project code if the prototype shrank (drift cleanup is the agent's call during integration, not ours).

## Accumulated Learnings

(Auto-rebuilt by /save-learnings from `learnings/*.md` frontmatter. Do not edit by hand. Initially empty — entries appear as the skill encounters reusable edge cases.)
