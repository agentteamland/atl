---
knowledge-base-summary: "Different bundle shape than Claude Design — JSON-based (prototype.json + ds.json + resolved-tokens.json + preview.html + README + integration-notes), zipped via the `/dst-handoff` skill. **Mandatory theme-sync step before component code:** compare `tailwind.config.ts` / `tokens.css` against `ds.json` and rewrite the theme first if they drift, otherwise \"trust project theme\" silently yields stale colors. Target variants: `react-admin` (Vite + shadcn + Zustand) and `react-public` (Next.js + next/image + SEO metadata). Same 5 translation rules apply. Worked example inside."
---
# DST Handoff (`/dst-handoff` bundle consumption)

When the `/dst-handoff <prototype>` skill from `design-system-team` fires, it spawns you (react-agent) with a zipped bundle and an integration brief. This document describes the **bundle shape**, the **mandatory theme-sync step**, and the **5 translation rules** that turn that bundle into working React code.

> **Two bundle shapes exist.** This file covers the **DST bundle** (design-system-team v0.4.0+). The older Claude Design bundle shape (JSX + `tokens.jsx`) is covered by `claude-design-handoff.md`. Translation rules are the same for both; file structure and parsing differ.

## The DST bundle — what's inside

The bundle is always at `.dst/prototypes/<name>/handoff.zip`. Extract to a temp dir:

```bash
mkdir -p /tmp/<prototype>-bundle && unzip -o .dst/prototypes/<name>/handoff.zip -d /tmp/<prototype>-bundle
```

| File | Shape | Use |
|---|---|---|
| `README.md` | Markdown | Entry point. Lists states, new i18n keys, routes, warnings. |
| `integration-notes.md` | Markdown | Target-specific rules (react-admin vs react-public variants differ slightly). |
| `prototype.json` | JSON | Prototype state. `frames` per state, `actions`, `breakpoints`, `tokens` map, `target`. |
| `ds.json` | JSON | Linked Design System — source of truth for every visual value. |
| `resolved-tokens.json` | JSON | Pre-computed flat map `{ "ds.palette.brand.primary.value": "#2D5F3F", ... }`. `null` = unresolved token. |
| `preview.html` | HTML | Visual reference — open in a browser to cross-check. |
| `assets/` | Directory | Any prototype-local assets (copy to `public/` or `src/assets/`). |

## Step zero — Theme sync (non-negotiable)

**Before writing any component code, check the project's theme layer against `ds.json`.**

React projects express their theme in one of several places — identify which the project uses:

| Theme surface | DS source |
|---|---|
| `tailwind.config.ts` `theme.extend.colors.*` | `ds.palette.*` |
| `tailwind.config.ts` `theme.extend.fontFamily` | `ds.typography.fontFamilies.*` |
| `tailwind.config.ts` `theme.extend.spacing` / `borderRadius` | `ds.spacing.scale`, `ds.radii.*` |
| `src/styles/tokens.css` CSS variables | All of above mapped to `--color-brand-primary` etc. |
| shadcn/ui — `src/app/globals.css` `:root` vars | Same CSS variable layer |

If ANY of those drift from `ds.json`, rewrite the theme file first. Then proceed to component work.

If the theme is already in sync, note "theme already in sync" in your report.

### Why this step exists

Downstream, component code will reference `className="bg-brand-primary"` (Tailwind) or `color: var(--brand-primary)` (CSS vars). Those resolve via the theme config — so the config must reflect the DS. Without sync, the component looks correct but renders the scaffold's placeholder color.

### Minimal sync pattern (Tailwind + CSS vars)

```ts
// web/admin/tailwind.config.ts
export default {
  // ...
  theme: {
    extend: {
      colors: {
        // All values from ds.json
        brand: {
          primary: '#2D5F3F',       // ds.palette.brand.primary.value
          secondary: '#D4A574',     // ds.palette.brand.secondary.value
          accent: '#E07A5F',        // ds.palette.brand.accent.value
        },
        neutral: {
          50:  '#F9F9F8',           // ds.palette.neutral.50
          100: '#F1F1EF',
          // ...
          900: '#1A1A1A',
        },
        semantic: {
          error: '#B94040',         // ds.palette.semantic.error.value
          warning: '#8F6100',       // etc.
          success: '#2D6B3F',
          info: '#2A5C8F',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],  // ds.typography.fontFamilies.sans.stack
      },
      borderRadius: {
        sm: '4px',                  // ds.radii.sm
        md: '8px',
        lg: '12px',
        xl: '16px',
      },
      spacing: {
        // ds.spacing.scale[0..24] × 4px — usually align with Tailwind's default
      },
    },
  },
};
```

Always leave `// ds.<path>` comments pinning each value back to the DS source so the file is auditable.

## The 5 translation rules

### Rule 1 — Trust the project theme over precomputed colors

After the theme sync step, `className="bg-brand-primary"` IS the DS brand primary. Use Tailwind / theme class lookups, not inline hex from `resolved-tokens.json`.

✅ `<Button className="bg-brand-primary text-white">`
❌ `<Button style={{ background: '#2D5F3F' }}>` — bypasses theme; dark-mode / future DS changes don't propagate.

`resolved-tokens.json` is a sanity check + fallback for tokens Tailwind's theme doesn't express.

### Rule 2 — Map prototype state idioms to React patterns

The prototype's `frames` entries are React-like already, but patterns in the project's state library matter:

| Prototype state | React idiom (TanStack Query + Zustand projects) |
|---|---|
| `idle` | Default render, `queryStatus === 'idle'` or initial `useState()` |
| `submitting` | `useMutation.isPending === true`, inputs `disabled`, button renders spinner |
| `error` | `mutation.error !== null`, inline alert above form, inputs editable |
| `loading` (lists) | `useQuery.isLoading === true` → skeleton UI |
| `empty` | `useQuery.data` array is empty → empty-state component |
| `success` | Transient toast + navigation (`useNavigate()(-1)` or `to: '/home'`) |

Single component with hooks, branches on state. Don't build `LoginIdle`, `LoginSubmitting`, `LoginError` as three components.

### Rule 3 — Use the project's component library

Check `package.json`:

- **shadcn/ui present** (`@/components/ui/*` or `components/ui/`) → use its `<Button>`, `<Input>`, `<Alert>`, `<Dialog>` etc.
- **MUI / Chakra / Mantine** → use their primitives.
- **No library** → plain elements with Tailwind utilities.

Never import from unrelated libraries (e.g., `@headlessui/react` in a shadcn project). If a library provides a primitive, use it.

### Rule 4 — Extract every user-facing string to i18n keys

`i18next` / `react-intl` / whatever the project uses. Pattern:

1. Add key to `src/i18n/en.json` (or equivalent): `{ "login": { "title": "Welcome back" } }`.
2. Mirror to `src/i18n/tr.json` with translated value.
3. In the component: `const { t } = useTranslation(); <h1>{t('login.title')}</h1>`.
4. Use ICU syntax for pluralization / interpolation: `t('tasks.count', { count: items.length })`.

Never hardcode English or Turkish text in JSX.

### Rule 5 — Geometry from bundle, behavior from project

Button is `h-[52px]` because the DS says so (`ds.components.button.sizes.lg.height`). That's GEOMETRY.

On submit, which API client? (`apiClient` from `src/core/api/client.ts`.) Which error handler? (`messageResolver` for ProblemDetails envelope.) Where to navigate on success? (`useNavigate()` + project's route definitions.) That's BEHAVIOR, comes from the project.

**Don't invent routes or API endpoints the project doesn't already express.** If prototype shows a `/forgot-password` link but the route isn't in `router.tsx`, leave `// TODO(forgot-password)` on a no-op handler. Don't ship a half-baked flow.

## Target variants — react-admin vs react-public

Target values from `prototype.json.target`:

### `react-admin`

- Output path: `web/admin/src/features/<feature>/pages/<ScreenName>Page.tsx` (or `web/admin/src/pages/<feature>/` depending on project layout)
- State: Zustand for UI state, TanStack Query for server state (per project conventions)
- Style: Tailwind if present; otherwise the project's styling convention
- Component library: shadcn/ui when present; raw elements otherwise

### `react-public`

- Output path: `app/<path>/page.tsx` (Next.js App Router) or `pages/<path>.tsx` (Pages Router)
- State: Server components by default — client components only for interactivity
- SEO: Every page exports `metadata` (Next.js App Router) derived from prototype.json.description / title
- i18n: next-intl or next-i18next — check which is present
- Images: `next/image` with explicit sizing; copy assets from bundle to `public/`
- Performance: Marketing pages prioritize SEO + bundle size. Don't import heavy client libraries unless the prototype needs them.

## Worked example — react-admin login handoff

```
1. Extract /tmp/.../.dst/prototypes/login-screen/handoff.zip
2. Read README + integration-notes
3. Compare web/admin/tailwind.config.ts vs ds.json:
   - colors.brand.primary: '#4F46E5' vs '#2D5F3F' → OUT OF SYNC
   → Rewrite tailwind.config.ts colors.*, fontFamily.*, borderRadius.* from ds.json.
4. Write web/admin/src/features/auth/pages/LoginPage.tsx:
   - Uses shadcn/ui <Card>, <Input>, <Button>, <Alert>
   - useMutation for submit
   - authStore (Zustand) for user state after success
   - t('login.title'), t('login.submit'), etc.
   - 3 states: idle / submitting / error — branches in one component
5. Update web/admin/src/i18n/en.json + tr.json with login.* keys
6. Report:
   - "Theme synced: brand.primary #4F46E5 → #2D5F3F, fontFamily → Inter"
   - Files: LoginPage.tsx, tailwind.config.ts, i18n/en.json, i18n/tr.json
   - Next: cd web/admin && npm install && npm run dev
```

## Anti-patterns

❌ Hardcoding `style={{ background: '#2D5F3F' }}` — Rule 1 violation.
❌ Skipping theme sync "because I'll fix it next time" — silently wrong brand color.
❌ Three separate state-specific components (`LoginIdle`, `LoginSubmitting`) — one component branches on state.
❌ Inline English in JSX (`<h1>Welcome back</h1>`) — Rule 4 violation.
❌ Running `npm install` / `npm run dev` yourself — user does that; print commands.
❌ Adding dependencies silently — if bundle needs a lib the project doesn't have, flag in report.

## What this file does NOT cover

- **Claude Design bundle shape** — see `claude-design-handoff.md`.
- **flutter-agent DST bundles** — flutter-agent has its own `children/dst-handoff.md`.
- **Creating the DST bundle** — that's `design-system-team`'s `/dst-handoff` skill responsibility.
