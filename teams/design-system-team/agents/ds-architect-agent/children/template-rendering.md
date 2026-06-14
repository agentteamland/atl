---
knowledge-base-summary: "How to take a `ds.json` and render it through `templates/ds-detail.html.tmpl` into a polished Tailwind-styled HTML. Section ordering, swatches, ramps, component preview cards."
---
# Template Rendering — `ds.json` → `detail.html`

When you produce or update a design system, you also produce a polished HTML page that visualizes it. The page lives at `.dst/design-systems/{ds-name}/detail.html` and is meant to be opened directly in a browser.

## Source Template

`templates/ds-detail.html.tmpl` in this team's repo. It uses **simple `{{ key }}` placeholders** — no engine required, you (the agent) do string substitution as you write the file. For nested arrays (e.g., palette swatches, type ramp items), you generate the HTML markup yourself for each item by following the patterns shown in the template's example sections.

## What the rendered page must contain

The page has TWO chrome elements + a body of 12 sections.

### Chrome — sticky sidebar TOC (Faz 2)

A `<aside id="ds-sidebar">` is fixed to the left edge on `lg+` viewports (hidden below). It contains four label groups (Foundations / Components / Compositions / Language) each with `.ds-nav-link`s pointing to the section anchors. Components is special — under it, a `.ds-subnav` lists `.ds-subnav-link`s for each non-empty Components group (`#components-actions`, `#components-forms`, `#components-feedback`, `#components-navigation`, `#components-display`, `#components-overlays`).

Skip groups with zero components. The sidebar is hand-rolled per-page based on which sections + groups are populated.

A small `<script>` at the bottom of `<body>` runs scroll-spy: as the user scrolls, the link of the section currently in view gets `.is-active`.

### Chrome — floating theme toggle (Faz 2 — dark mode)

A `<button id="ds-theme-toggle">` is fixed to the bottom-right corner. Clicking it toggles a `body.theme-dark` class and persists the choice via `localStorage`. The button itself swaps between sun (light) and moon (dark) icons via CSS — no per-DS render branching needed.

Per-DS brand-tinted dark overrides (e.g., `pv-btn-primary` background = brand.dark.primary in dark mode) live in the page's inline `<style>` block, scoped under `body.theme-dark`. The shared `styles.css` handles surfaces, text, borders, sidebar, and pill colors generically.

**Mandatory:** the floating toggle is rendered on every page that has dark mode declared (`palette.dark` exists). For DSes without dark mode the toggle is omitted — we do not show a toggle that has no effect.

### Theme-aware visibility — `.theme-only-light` / `.theme-only-dark` (Faz 2.5)

Anywhere a block has a per-theme version, wrap each version in `.theme-only-light` / `.theme-only-dark`. Only the active theme's content shows; the floating toggle controls which.

The earlier "render both themes side-by-side" approach (Brand had a 2×2 grid of light+dark lockups; Iconography had a "On dark surface" strip; Palette / Elevation had separate "Dark surfaces" sections always visible) was retired because the floating toggle already declares the active theme — showing both at once is gratuitous noise, and dark content on a light page (or vice-versa) reads as a layout bug.

**Rule of thumb:** if the same data is meaningful in both themes (e.g., neutral gray ramp, brand color scale 50–900), it has NO theme class — show it always. If the data is theme-specific (dark surface palette, dark elevation shadows, dark logomark), wrap it with the matching `.theme-only-*` class.

### Brand section — single theme-aware stage (Faz 2 — dark mode)

Each lockup card has TWO mutually-exclusive stages — one `.theme-only-light`, one `.theme-only-dark`. The toggle controls visibility.

```
┌─ Brand card ────────────────────────────────────────────┐
│  Lockup (Horizontal)         │  Lockup (Vertical)       │
│  ┌─────────────────────────┐ │  ┌────────────────────┐  │
│  │ active theme stage only │ │  │ active theme only  │  │
│  └─────────────────────────┘ │  └────────────────────┘  │
│  ───────────────────────────────────────────────────────│
│  Personality chips           │  Type & icon meta        │
└─────────────────────────────────────────────────────────┘
```

Per-stage rule:

- **Light stage** — light asset on light surface (`#F5F5F4` neutral or palette.surface).
- **Dark stage** — depends on `brand.logomarkDark` (and `wordmarkDark`):
  - **Declared** → render the dark asset on dark surface (`#0F1612` or palette.dark.background).
  - **`null` / absent** → render the LIGHT asset on the dark surface AND emit a `.ds-lockup-warn` strip telling the user the dark variant is missing. The page never silently swallows the gap.

Wordmark dark text color is hardcoded to a near-white when no `wordmarkDark` is declared — wordmark text inverts trivially. Only the actual logo bitmap/SVG triggers the warning, since text inversion is automatic and lossless.

### Section order, top to bottom

1. **Header** — DS name, version, last-modified date, brand tagline, logomark inline
2. **Brand** — logomark preview (inline SVG if available), wordmark, lockup variants, personality keywords as chips
3. **Iconography** — family name + style + stroke-width + default size; a size ramp (16/20/24/32/48 of the same representative icon) + a product sampler of 20-30 inline SVG tiles grouped by role (navigation / actions / media / social / states / settings). If `palette.dark` exists, include a small "on dark surface" strip showing the same icons on a dark card
4. **Palette** — brand colors first (large swatches with hex), then semantic colors (smaller swatches grouped by role), then **dark mode preview** if `palette.dark` is defined. Dark mode must render both swatches AND a mini typography preview on dark surface. Each swatch shows: hex, role name, contrast ratio against text-on-it
5. **Typography** — type ramp shown as actual rendered text (display through caption), each annotated with size/line-height/weight. Use the actual font (load via Google Fonts CDN if specified)
6. **Spacing & Radii** — visual scale: each spacing value shown as a labeled bar, each radius as a labeled rounded square
7. **Elevation & Motion** — see full spec below (this section is rich)
8. **Components** — see full spec below (this section is rich, with 6 sub-groups)
9. **State Recipes** — empty / loading / error / success treatments rendered as real visual examples. See [state-recipes.md](state-recipes.md) for the full composition spec per recipe
10. **Patterns** — reusable UI compositions (searchFilter / masterDetail / listPagination / formComposition) rendered as real mini-UIs of the pattern, not spec dumps. See [patterns.md](patterns.md)
11. **Page Templates** — whole-page blueprints (login / onboarding / profile / settings) rendered as **abstract wireframes** (dashed-border regions + labels), NOT mini-UIs. The contrast with Patterns is intentional — see [page-templates.md](page-templates.md)
12. **Voice & Tone** — Do/Don't lists with checkmark/X icons, sample copy in styled quote blocks
13. **Accessibility** — WCAG level badge, min tap target, contrast minimums, focus-visible commitment, reduced-motion handling

### Components section — expected structure (Faz 2 + Faz 4 partial system)

Components are organized into **groups** with stable order: `actions → forms → feedback → navigation → display → overlays → charts`. Each populated group is wrapped in `<div id="components-{slug}" class="ds-component-group">` with a header (uppercase brand-colored label + one-line description) and a series of component cards.

#### Partial-based render flow (Faz 4 — primary mechanism)

As of Faz 4, every catalog component has a pre-authored partial at:

```
templates/partials/components/{group-slug}/{component-slug}.html.tmpl
```

For each component in `ds.json.components.*`, the render flow is:

1. **Resolve group** from `children/components/{group}.md` membership (DO NOT guess — schema-level membership, not a render hint).
2. **Read the partial file** at `templates/partials/components/{group-slug}/{component-slug}.html.tmpl`.
3. **Paste the partial contents** into the group wrapper. No string substitution — partials are pre-rendered (variants/sizes/states are canonical per-component, hardcoded inside).

This eliminates ad-hoc per-component rendering. Two implications:

- Editing one component's render = touching ONE partial file. No 600-line monolith to navigate.
- Adding a new catalog component = creating ONE new partial file in the right subdirectory + updating `ds.json` + the `children/components/{group}.md` spec. The agent picks it up automatically.

#### Two card styles (encoded in the partials)

- **Mini-UI partial** (7 components: button, input, select, switch, card, chip, modal). Renders REAL visual examples using `.pv-*` classes defined in the inline `<style>` block: live buttons, real switches, actual card surfaces. Per-DS brand colors come from inline `<style>`.
- **Spec card partial** (35 components: the rest of the catalog). Uses neutral `.dst-*` helpers from shared `styles.css`. Layout:
  - Header strip: `<h3>{Display Name}</h3>` + `<p class="font-mono">{N} variants · {M} sizes · {K} states</p>`
  - One-line `<p class="dst-component-intro">` (the component's intent — pulled from children/components/{group}.md)
  - Three `<div class="dst-spec-row">` blocks: Variants / Sizes / States — each containing pills (`.dst-spec-pill .dst-pill-variant|size|state`)
  - Sizes pills append a key dimension hint when available: `md · 44px`, `lg · 480px`, `default · 40px`
  - Optional bottom `<div class="dst-spec-note">` highlighting 1-2 instructive top-level fields (`placementRule`, `behavior`, `structure`, etc.)

#### Fallback rule (custom / brand-new components)

If a DS declares a custom component without a partial (or you're authoring a brand-new component and the partial hasn't been added yet), fall back to inline spec-card generation following the spec-card layout above. **Adding the partial is preferred** — recurring components should always have a partial so future edits stay surgical.

#### Authoring a new partial — convention

When you add a new component to the catalog (Faz 4 Tier 1 additions, or any future expansion):

1. Create `templates/partials/components/{group-slug}/{component-slug}.html.tmpl` with the spec-card or mini-UI structure (see existing partials in the directory for examples).
2. Variants / sizes / states list = canonical from `children/components/{group}.md` spec. Hardcode them in the partial — they don't change per-DS.
3. The intro paragraph = first sentence of the component's spec block in the children file.
4. The spec-note = pull 1-2 instructive top-level fields (`placementRule`, `behavior`, `structure`, `groupRules`, `accessibility`, etc.) from the children spec.
5. The partial file is its own atomic deliverable — no shared state with other components, no cross-references except documentary header comment pointing to the spec source.

**Group-aware rendering rule:** Look up each component's group from `children/components/{group}.md` (which lists members) — DO NOT guess. The group is the **schema-level membership**, not a render hint. Getting the group wrong puts the component under the wrong sub-anchor AND breaks the partial path lookup.

Component groups + their component lists are documented in [component-specs.md](component-specs.md), which is the index for the per-group children files.

### Elevation & Motion — expected structure

**Elevation** renders twice when `elevation.dark` is defined: once for light surfaces, once on a dark card. For each elevation level (including `none`), emit a preview box (`96 × 72px`, radius 12) with the declared shadow and a use-case hint below (`none → flat surface`, `low → cards, list items`, `medium → dropdowns, menus`, `high → modals, sheets`, `xhigh → floating action`). Use the `.dst-elev-*` helpers from `styles.css` where possible; per-DS overrides go in the page's inline `<style>` block.

**Motion** has two sub-sections:

1. **Durations** — one row per `motion.durations` entry. Each row shows the token name (left), an animated dot moving inside a track (middle), and the value (right). The animation duration on-screen must be **2× the declared token value** so one round-trip equals one cycle at the declared speed. Emit a per-row CSS class (e.g., `.dst-dur-instant`) in the inline `<style>` block setting `animation-name: dst-travel; animation-duration: 2×value`.
2. **Easing curves** — one card per `motion.easings` entry, all at 1.2s so the eye reads the curve shape regardless of timing. Emit a per-curve class (`.dst-ease-{name}`) in the inline `<style>`. Use-case hints:
   - `standard` → "Default — most UI transitions"
   - `decelerate` → "Entering — things coming on screen"
   - `accelerate` → "Leaving — things exiting screen"
   - `emphasized` → "Hero moments — confident, deliberate"

End the Motion card with a reduced-motion notice referencing `prefers-reduced-motion: reduce`.

## Tailwind Setup

Pages use Tailwind via CDN (loaded from `.dst/styles.css` link). You don't need to bundle Tailwind. Use utility classes liberally.

```html
<link rel="stylesheet" href="../../styles.css">
```

The `styles.css` file (created by `/dst-init`) imports Tailwind via CDN and adds team-specific custom styles.

## Quality Bar

Every rendered detail.html should look like a professional design portfolio page. Specifically:

- **Generous whitespace** — `py-12`, `gap-8` minimum between major sections
- **Clear visual hierarchy** — section headings in `text-2xl font-semibold`, subsection in `text-lg`
- **Live previews** — palette swatches are colored boxes with hex labels (not just text). Type ramp uses actual sized + weighted text. Component variants render as actual buttons/inputs (visually).
- **Responsive** — page works at 1024px and above. Mobile-friendly nice-to-have, not required for v0.1.
- **Print-friendly is a bonus** — page should look reasonable when "Print to PDF"'d.

## Rendering Algorithm (your job)

1. Read `templates/ds-detail.html.tmpl` from team repo.
2. Read `ds.json` you just produced/updated.
3. Replace top-level placeholders: `{{ name }}`, `{{ version }}`, `{{ description }}`, etc.
4. For each section that has list/array data (palette swatches, type ramp items, components):
   - The template has an EXAMPLE block showing the markup for ONE item.
   - You generate the markup for each item by following that pattern.
   - You write the assembled HTML into the section's container.
5. Write the final HTML to `.dst/design-systems/{ds-name}/detail.html`.

## Worked Example: Palette Section

Template has:
```html
<section id="palette">
  <h2>Palette</h2>

  <div id="brand-palette" class="grid grid-cols-2 gap-4">
    <!-- AGENT FILLS THIS: one card per brand color -->
    <!-- EXAMPLE PATTERN: -->
    <!-- 
      <div class="rounded-lg overflow-hidden border border-gray-200">
        <div class="h-32" style="background:{{HEX}}"></div>
        <div class="p-4">
          <p class="font-semibold">{{NAME}}</p>
          <p class="text-sm text-gray-500 font-mono">{{HEX}}</p>
          <p class="text-xs mt-2 text-gray-600">Contrast vs white: {{RATIO}}</p>
        </div>
      </div>
    -->
  </div>
  ...
</section>
```

You read the example pattern, then for each entry in `palette.brand`, you produce a `<div>...</div>` block with HEX, NAME, RATIO substituted in, and concatenate them into the `#brand-palette` container.

## After Writing

Don't open the browser yourself — that's `/dst-open`'s job. Just write the file and let the user know it's ready.

## Common Mistakes to Avoid

- ❌ Printing raw JSON in the page — render it as visual elements
- ❌ Forgetting to load Google Fonts when typography uses them
- ❌ Hardcoding values from one DS into the template (template is generic)
- ❌ Skipping sections because data wasn't in `ds.json` — emit the section with a "(not specified)" placeholder, so the page structure stays consistent across DSes
- ❌ Rendering `elevation` and `motion` only as static text (shadow hex values, duration numbers) — both must be **shown**: elevation via real shadowed boxes, motion via actual CSS animations
- ❌ Skipping the Iconography section because "lucide icons are just imports in code" — the DS page MUST show inline SVGs of the declared family so designers can see what the product will look like
- ❌ Dark mode section that only shows color swatches — it must also include a typography preview and a mini component preview (button + input + card) on the dark surface
- ❌ Emitting components ungrouped, alphabetically, or in `ds.json` insertion order — group order is fixed (`actions → forms → feedback → navigation → display → overlays`) and groups are mandatory wrappers, not decoration
- ❌ Sidebar TOC missing the Components sub-anchors — every populated group MUST appear under Components in the sidebar; without sub-anchors the user has to scroll 30+ cards to find one
- ❌ Skipping the scroll-spy script — the sidebar is dead without it; `is-active` highlighting is the only feedback that the user is in the right place
- ❌ Emitting `<div class="ds-component-group-empty">` placeholders for groups with zero components — that's a draft-state aid only; production renders should skip empty groups silently

## Inline `<style>` block — when it's fine (and when it isn't)

Per-DS brand colors drive motion dot color, elevation shadow tints, and other visual details that the generic `styles.css` can't encode. Emit a page-scoped inline `<style>` block at the top of `detail.html` that overrides the neutral `.dst-*` helpers with brand-specific values:

```html
<style>
  /* Brand-tinted motion dot */
  .dst-motion-dot { background: {{ brandPrimaryHex }}; }
  /* Brand-tinted spacing bars */
  .dst-spacing-bar { background: {{ brandPrimaryHex }}; }
  /* Duration + easing class definitions — must be per-DS because tokens differ */
  .dst-dur-fast { animation-name: dst-travel; animation-duration: 320ms; animation-timing-function: cubic-bezier(0.4, 0, 0.2, 1); animation-iteration-count: infinite; animation-direction: alternate; }
  /* ...one per token... */
</style>
```

What does NOT go in the inline block: generic helpers (`.dst-swatch`, `.dst-type-row`, `.dst-elev-box` structural styles) — those belong in the shared `styles.css`.
