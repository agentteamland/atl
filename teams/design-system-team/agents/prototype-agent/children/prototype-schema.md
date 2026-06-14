---
knowledge-base-summary: "The canonical shape of `prototype.json`. Linked-DS reference, frames per state, content blocks, interaction notes, asset paths."
---
# Prototype Schema (`prototype.json`)

The canonical structure of every prototype this agent produces. Lives at `.dst/prototypes/{prototype-name}/prototype.json`. The `preview.html` is rendered from it.

## Top-level shape

```json
{
  "schemaVersion": 1,
  "name": "login-screen",
  "displayName": "Login Screen",
  "linkedDs": "primary",
  "version": "1.0.0",
  "createdAt": "2026-04-22T10:00:00Z",
  "lastModified": "2026-04-22T10:00:00Z",
  "archetype": "form",
  "target": "flutter",
  "description": "Email + password sign-in with inline error and forgot/register links.",

  "breakpoints": ["mobile"],
  "actions": ["submit", "forgot-password", "register"],

  "pageTemplate":      "login",
  "patternsUsed":      ["formComposition"],
  "stateRecipesUsed":  ["error", "success"],
  "componentsUsed":    ["input", "button", "checkbox", "alert"],

  "frames": {
    "idle":       { ... },
    "submitting": { ... },
    "error":      { ... }
  },

  "tokens": { ... },
  "notes": [ "..." ]
}
```

### Composition reference fields (Faz 3)

These four fields make the prototype's DS dependency graph explicit. They are populated from the **TOP-DOWN composition walk** described in [screen-blueprint.md](screen-blueprint.md), and consumed by `/dst-handoff` to bundle the right subset of the DS.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pageTemplate` | string \| null | optional | A key from `linkedDs.ds.json.pageTemplates.*` if a matching template exists. `null` if archetype doesn't fit any declared template. Example: `"login"`. |
| `patternsUsed` | string[] | required | Array of keys from `linkedDs.ds.json.patterns.*`. Empty array if the screen doesn't compose any patterns (rare — usually at least one applies). Example: `["formComposition"]`. |
| `stateRecipesUsed` | string[] | required | Array of keys from `linkedDs.ds.json.stateRecipes.*` for non-happy frames. Example: `["error", "success"]`. The `idle` recipe is implicit and not listed. |
| `componentsUsed` | string[] | required | Array of `linkedDs.ds.json.components.*` keys. Lists every primitive component the screen instantiates. Example: `["input", "button", "checkbox", "alert"]`. Used by `/dst-handoff` to bundle only relevant component specs (not all 42). |

**Validation rules:**
- Every `componentsUsed` entry must exist in the linked DS's `ds.json.components`.
- Every `patternsUsed` entry must exist in `ds.json.patterns`.
- If `pageTemplate` is set, every entry in that template's `patternsUsed` must also appear in this prototype's `patternsUsed` (unless the prototype intentionally diverges — note in `notes`).
- `stateRecipesUsed` must include at minimum the recipes for archetype-required states (e.g., form archetype requires `error` + `success` recipes referenced).

**Anti-patterns:**
- ❌ Leaving these fields empty/missing because "the frames already describe everything" — handoff bundle won't know what to include.
- ❌ Listing components used inside referenced patterns but NOT listing the pattern itself in `patternsUsed` — flattens the composition tree, loses the higher-level rationale.

## Field reference

### Top-level metadata

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schemaVersion` | int | ✅ | Always `1` for now. |
| `name` | string | ✅ | Kebab-case, matches directory name. |
| `displayName` | string | ✅ | Title Case, shown in studio UI. |
| `linkedDs` | string | ✅ | Name of the design system this prototype uses. Must exist in `.dst/design-systems/`. |
| `version` | string | ✅ | SemVer. Bumped when prototype is meaningfully edited. |
| `createdAt` / `lastModified` | ISO timestamp | ✅ | Audit metadata. |
| `archetype` | enum | ✅ | One of: `form`, `list`, `detail`, `dashboard`, `modal`, `empty`, `settings`, `static`, `flow` |
| `target` | enum | ✅ | One of: `flutter`, `react-admin`, `react-public`. Determines which code-agent `/dst-handoff` will brief. Chosen during `/dst-new-prototype` Q&A (Q0). |
| `description` | string | ✅ | One-sentence summary of what the screen is/does. |

### `breakpoints`

```json
"breakpoints": ["mobile"]
```

Array of: `mobile`, `tablet`, `desktop`. Each breakpoint = one variant of every frame in the rendered preview. Default v0.1.0: single breakpoint per prototype (multi-bp v0.2.0+).

### `actions`

```json
"actions": ["submit", "forgot-password", "register"]
```

User actions available on this screen. Used to:
- Drive Q&A about labels/destinations during creation
- Generate ARIA labels for buttons
- List in the rendered preview as "actions on this screen" callout

### `frames`

The heart of the prototype. One entry per state. Frame objects share a structure:

```json
"frames": {
  "idle": {
    "label": "Default",
    "description": "Form ready for input. No values entered, no errors.",
    "blocks": [
      {
        "type": "header",
        "content": {
          "title": "{{ ds.voice.samples.welcome }}",
          "subtitle": "Sign in to your account"
        }
      },
      {
        "type": "input",
        "id": "email",
        "label": "Email",
        "placeholder": "name@example.com",
        "inputType": "email",
        "value": ""
      },
      {
        "type": "input",
        "id": "password",
        "label": "Password",
        "placeholder": "",
        "inputType": "password",
        "value": ""
      },
      {
        "type": "button",
        "id": "submit",
        "variant": "filled",
        "size": "lg",
        "text": "Sign in",
        "fullWidth": true,
        "state": "idle"
      },
      {
        "type": "link",
        "id": "forgot-password",
        "text": "Forgot password",
        "alignment": "center"
      }
    ]
  },
  "submitting": {
    "label": "Submitting",
    "description": "Form being submitted. Inputs disabled, button shows spinner.",
    "blocks": [ /* same shape, with state changes */ ]
  },
  "error": {
    "label": "API error",
    "description": "Submit returned 401 (invalid credentials). Banner above form.",
    "blocks": [
      {
        "type": "alert",
        "severity": "error",
        "message": "{{ ds.voice.samples.errorInvalidCredentials | default: 'Invalid email or password.' }}"
      },
      /* + same form blocks from idle */
    ]
  }
}
```

### Block types (composable building units)

- `header` — `{ title, subtitle? }`
- `text` — `{ content, variant? }` (variant = body / caption / etc., default body)
- `input` — `{ id, label, placeholder?, inputType, value, helperText?, errorText? }`
- `button` — `{ id, variant, size, text, fullWidth?, state, icon? }` (state = idle/loading/disabled)
- `link` — `{ id, text, href?, alignment? }`
- `divider` — `{}`
- `alert` — `{ severity (error|warning|info|success), message, dismissable? }`
- `card` — `{ title?, content (nested blocks), variant? }`
- `image` — `{ src, alt, width?, height?, fit? (cover/contain) }`
- `spacer` — `{ size (number; multiple of DS spacing unit) }`
- `list` — `{ items: [...], divider? }`
- `chip` — `{ text, variant? }`
- `toggle` — `{ id, label, value (boolean), state }`

Add more types as needed in future versions.

### `tokens` map

Convenience hash of token references this prototype uses. Helps the renderer and human reviewers see at a glance which DS bits are touched:

```json
"tokens": {
  "brandPrimary": "{{ ds.palette.brand.primary.value }}",
  "buttonHeightLg": "{{ ds.components.button.sizes.lg.height }}",
  "spacing4": "{{ ds.spacing.scale[4] }}"
}
```

If `linkedDs` doesn't have a referenced token, the render fails with a clear error message — surface the missing token to the user instead of silently degrading.

### `notes`

Free-form array of strings for things that don't fit the schema:

```json
"notes": [
  "submitting state's button spinner uses Material loader animation (default)",
  "register link goes to /auth/register (route to be wired in app)"
]
```

## Token reference syntax

Anywhere you'd put a literal value in `frames`, you can reference a DS token using `{{ ds.<path> }}` placeholder syntax. The renderer resolves these at render time.

Examples:
- `{{ ds.palette.brand.primary.value }}` → `"#2D5F3F"`
- `{{ ds.typography.scale.body.size }}` → `"16px"`
- `{{ ds.voice.samples.welcome }}` → `"Welcome."`
- `{{ ds.components.button.sizes.lg.height }}` → `"52px"`

Optional default fallback:
- `{{ ds.voice.samples.errorInvalidCredentials | default: 'Invalid sign-in.' }}`

## Versioning

`schemaVersion: 1` for now. Bump if breaking changes (new required fields, type changes).

`version` per prototype is independent — bumped by edits via `/dst-edit-prototype`.

## Notes for generation

- **Always reference DS tokens** for visual values; raw hex/px in blocks is a smell.
- **Use voice samples from DS** for default copy when applicable (welcome, error, empty messages).
- **Cover all applicable states** for the archetype — see `state-coverage.md`.
- **Don't omit `description`** at top level or per-frame — they make the preview self-explanatory.
