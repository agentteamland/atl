---
knowledge-base-summary: "How to take a `prototype.json` (+ linked DS's `ds.json`) and render it through `templates/prototype-detail.html.tmpl` into a polished, multi-state Tailwind-styled HTML page."
---
# Preview Rendering — `prototype.json` + `ds.json` → `preview.html`

When you produce or update a prototype, you also produce a polished multi-state HTML page. Lives at `.dst/prototypes/{prototype-name}/preview.html`. Opens in any browser.

## Source template

`templates/prototype-detail.html.tmpl` in this team's repo. Same `{{ key }}` placeholder convention as `ds-detail.html.tmpl`.

## What the rendered page contains

Section order, top to bottom:

1. **Header** — prototype `displayName`, version, lastModified, linked DS name with link to its detail.html
2. **Description** — prototype `description` from `prototype.json`
3. **Linked DS quick-reference** — chips showing which palette/typography is in play
4. **Frames** — one section per frame (state):
   - State label + description
   - Rendered frame at the prototype's breakpoint(s)
   - If multi-breakpoint: each breakpoint's frame side-by-side
5. **Notes** — `prototype.json.notes` rendered as bullet list (if any)
6. **Token usage summary** — list of DS tokens this prototype uses (from `prototype.json.tokens` map)

## Rendering algorithm

1. **Read inputs**:
   - `prototype.json` (the prototype to render)
   - Linked `.dst/design-systems/{linkedDs}/ds.json` (token resolution context)
   - `templates/prototype-detail.html.tmpl` (the team-side template)

2. **Resolve all token references** in `prototype.json`:
   - For every `{{ ds.<path> }}` pattern in any block's properties:
     - Look up the value in `ds.json` by path
     - Replace the placeholder with the resolved value
     - If missing → use default if provided; else error with explicit "missing token: <path>"

3. **Generate CSS variables block** — for the `<style>` in `<head>`, emit one `--ds-*` variable per used token:

```html
<style>
  :root {
    --ds-brand-primary: #2D5F3F;
    --ds-text-primary: #1A1A1A;
    --ds-button-height-lg: 52px;
    /* ... only the tokens this prototype actually uses */
  }
</style>
```

4. **Render each frame** — for each state in `prototype.json.frames`:
   - Generate a frame container with state label + description
   - For each block in the frame, generate the corresponding HTML (see "Block-to-HTML mapping" below)
   - Wrap in viewport-sized container per breakpoint

5. **Emit token usage summary** at the bottom — list every token reference from `prototype.json.tokens`, formatted as `<token-name>: <resolved-value>` so reviewers can audit fidelity quickly.

6. **Write to disk** — `.dst/prototypes/{name}/preview.html`.

## Block-to-HTML mapping

Each `prototype.json` block type maps to specific HTML:

| Block type | HTML output |
|------------|-------------|
| `header` | `<header><h1>{title}</h1><p>{subtitle}</p></header>` |
| `text` | `<p class="text-ds-{variant}">{content}</p>` |
| `input` | `<label for="{id}">{label}</label><input id="{id}" type="{inputType}" placeholder="{placeholder}" value="{value}" />` |
| `button` | `<button id="{id}" class="ds-btn-{variant} ds-btn-{size} {fullWidth ? 'w-full' : ''}" {disabled if state==='disabled'}>{text}</button>` |
| `link` | `<a href="{href || '#'}" class="ds-link {alignment}">{text}</a>` |
| `divider` | `<hr class="ds-divider" />` |
| `alert` | `<div role="alert" class="ds-alert ds-alert-{severity}">{message}</div>` |
| `card` | `<div class="ds-card ds-card-{variant}"><h3>{title}</h3>{...nested blocks}</div>` |
| `image` | `<img src="{src}" alt="{alt}" class="..." />` |
| `spacer` | `<div style="height: {size * spacingUnit}px"></div>` |
| `list` | `<ul class="ds-list">{items as <li>}</ul>` |
| `chip` | `<span class="ds-chip ds-chip-{variant}">{text}</span>` |
| `toggle` | `<label class="ds-toggle"><input type="checkbox" {checked if value} /><span>{label}</span></label>` |

The `ds-*` classes are defined in the `.dst/styles.css` and use CSS variables from the resolved DS.

## Quality bar

The rendered preview should look like a designer's screen-flow handoff:

- **Clear state labels** with state name + description above each frame
- **Polished frame borders** — each frame in a card-like container with shadow/border
- **Consistent spacing** between frames (24-32px gap)
- **Live tokens** — colors, fonts, spacing all reflect the linked DS exactly
- **Multiple states visible** — user scrolls through, sees idle → submitting → error all rendered
- **Token-fidelity passes** — no naked hex/px in the rendered HTML

## After writing

Don't open the browser yourself — `/dst-open` does that. Just write the file and let the user know.

## Common mistakes

- ❌ Failing silently when a token is missing — emit a clear error message and refuse to write
- ❌ Hardcoding values that came from the DS into the rendered HTML (use CSS variables)
- ❌ Skipping a state because "it's similar to idle" — render every state explicitly
- ❌ Forgetting to load Google Fonts when the DS uses them (emit `<link>` based on `ds.json.typography.fontFamilies`)
- ❌ Rendering a 600px-wide layout in a desktop breakpoint (use the breakpoint's intended viewport width)
