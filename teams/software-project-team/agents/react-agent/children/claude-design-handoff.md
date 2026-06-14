---
knowledge-base-summary: "Bundle from Claude Design (via `/design-screen done`) is React-native — adaptation, not paradigm translation. Five rules: swap raw web primitives for `@/components/ui/*`, swap useState for project state library (Zustand / React Query), honor project router (Link/href), wire i18next + en/tr namespaces, translate inline styles to Tailwind tokens. Lighter friction than Flutter; same self-contained brief discipline."
---
# Claude Design Handoff → React (admin / public)

When `/design-screen done` calls react-agent with an extracted Claude Design bundle, this file is the integration playbook. Lighter than flutter-agent's equivalent because the bundle's native idiom IS React — we're adapting, not translating across paradigms.

## What's In the Bundle

```
README.md                                  # Claude Design's own brief
components/
  tokens.jsx                               # Self-documenting token reference
  {screen-name}.jsx                        # The screen as a React component
frames/
  desktop-shell.png  (or web-shell.png)    # Visual reference renders
```

The bundle's React is **nearly drop-in for our React projects** — but never *literally* drop-in. There are five things you always change.

## The Five Adaptation Rules

### Rule 1 — Swap web primitives for the project's component library

The bundle uses raw HTML primitives (`<div>`, `<button>`, `<input>`, `<form>`). Your project has wrapped versions in `src/components/ui/` (e.g., `Button`, `Input`, `Form`, `Card`).

| Bundle primitive | Project equivalent (typical) |
|------------------|-------------------------------|
| `<button>` | `<Button variant="..." />` from `@/components/ui/button` |
| `<input>` | `<Input />` from `@/components/ui/input` |
| `<form>` | `<Form>` (react-hook-form wrapper) |
| `<a>` | `<Link>` from `react-router-dom` (admin) or `next/link` (public) |
| Inline icons (svg literals) | `lucide-react` icon component imports |

**Read the project's `components/ui/` once before integrating** so you know what's available. Match props (variants, sizes) to the closest existing component variant.

### Rule 2 — Swap the bundle's local state for the project's state library

The bundle uses `useState`/`useReducer` for everything. The project uses Zustand (admin) and possibly React Query for server state. Pull *server* state into Zustand stores or React Query hooks; keep *purely local UI* state as `useState`.

| Bundle pattern | Project pattern |
|----------------|-----------------|
| `const [user, setUser] = useState(null)` (after fetch) | `useUserStore()` (Zustand) or `useUser()` (React Query) |
| `const [loading, setLoading] = useState(false)` (for fetch) | React Query's `isLoading` |
| `const [error, setError] = useState(null)` (for fetch) | React Query's `error` |
| `const [open, setOpen] = useState(false)` (for modal) | Local `useState` — keep as-is |
| `const [focused, setFocused] = useState('email')` | Form library's focus state OR local `useState` if no form lib |

If the screen calls an API, route the call through the project's `apiClient` (not raw `fetch`).

### Rule 3 — Honor the project's routing

Bundle's links are static `href` strings. Replace with `<Link to="...">` (react-router-dom) or `<Link href="...">` (next/link). If the screen has navigation that doesn't yet have a route in the project's router config, leave a TODO marker AND add the route stub to the router config in the same PR.

### Rule 4 — Wire the project's i18n contract

User-facing strings in the bundle are English literals. Per the i18n contract (`api-agent/children/user-facing-strings.md`), React uses i18next with namespace files in `src/locales/{lang}/{namespace}.json`.

**Workflow:**
1. Identify each user-facing string in the bundle.
2. Choose a namespace (`common`, `auth`, `dashboard`, `users`, etc. — see existing files).
3. Add the key to BOTH `src/locales/en/{ns}.json` AND `src/locales/tr/{ns}.json` in the same change.
4. Reference via `t('key')` or `t('namespace:key')` in the component.

For strings that come from API responses (validation errors, notifications), follow the envelope pattern from the i18n contract — don't hardcode them as keys; resolve via `resolveApiMessage` from `lib/resolve-api-message.ts`.

### Rule 5 — Honor the project's styling conventions

The bundle's CSS-in-JS (or inline styles) is a visual specification. Translate to the project's styling approach:

- **Tailwind project (typical)** → translate to Tailwind utility classes. Match spacing/colors to the project's `tailwind.config.ts` tokens. If the bundle uses `padding: 24px`, that's `p-6` (4×6 = 24); cross-reference the spacing scale.
- **CSS modules / styled-components** → adapt to that flavor.

**Trust the project's tokens.** If the bundle has hardcoded `color: '#1a73e8'`, find the matching token in `tailwind.config.ts` (e.g., `text-primary`) and use it. Pure hex literals in the final code are a smell.

## Workflow When Briefed by `/design-screen done`

1. **Read the intent.md.** Get the Phase A context — verbatim user requirements, original prompt, copy decisions, target (admin or public).
2. **Open the extracted bundle.** Read `README.md`, then `components/tokens.jsx`, then the screen `.jsx`.
3. **Cross-reference tokens.jsx with the project's `tailwind.config.ts`** (or design tokens module). Confirm naming alignment. If the bundle's tokens.jsx self-documents ("derived from web/tailwind.config.ts"), trust the mapping.
4. **List new i18n keys** before writing the component. Add to both `en/` and `tr/` JSON files.
5. **Write the component** following the five adaptation rules.
6. **Wire routing** if needed; add stubs for any nav links that target nonexistent routes.
7. **Update Storybook** (if the project uses it) — add a story for the new screen.
8. **Write the engineer note** at `.claude/design/{slug}/integration-notes.md`. Capture: bundle deviation decisions, new keys added, routes stubbed, files touched.

## Anti-Patterns

### Don't import bundle's React component as-is
Even if it would compile, you'd ship inline styles, raw `<button>` elements, hardcoded English strings, and direct `fetch` calls. Adapt every layer.

### Don't lift the bundle's `useState` for everything
If the bundle has `const [users, setUsers] = useState([])` after a fetch, that's a React Query opportunity, not a state mirror.

### Don't keep the bundle's CSS-in-JS approach
Match the project's styling system. Inline styles in production code are a smell.

### Don't skip the engineer note
Same rule as flutter-agent: even smooth integrations get a note. Future "why is this screen 95% like the design?" questions need an answer.

## Quick Reference Card

```
Bundle has                Project uses
────────────────────────  ─────────────────────────────────────
Raw <button>, <input>     @/components/ui/* wrappers

Inline lucide-react SVG   Already available — same lib

useState for server data  React Query / Zustand store

useState for UI flags     Keep — useState is fine for modal open, etc.

href="/some/path"         <Link> from project's router

Hardcoded English text    i18next namespace + en/tr JSON

Inline styles / CSS-in-JS Tailwind utilities (or project's flavor)

Hex color literals        Tailwind theme token (text-primary, bg-card...)
```

## Difference vs flutter-agent's handoff

The Flutter side translates *across paradigms* (React → Dart, JSX → Widgets). React side adapts *within the paradigm* (raw React → project-conventional React). Roughly:

| Concern | Flutter (translation) | React (adaptation) |
|---------|----------------------|---------------------|
| Language | Dart vs. JSX | Same JSX |
| State primitives | FocusNode, FormFieldState | useState / Zustand / React Query |
| Routing | go_router | react-router-dom / next/link |
| Styling | Theme + Material widgets | Tailwind / CSS modules |
| Friction | High (paradigm gap) | Low (same paradigm) |
| Implementation time after design | ~3 min/screen (pilot) | likely faster |

Both agents end at the same place: a screen that respects the bundle's intent, the project's conventions, and the i18n contract.
