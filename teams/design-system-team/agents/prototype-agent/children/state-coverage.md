---
knowledge-base-summary: "Which states apply to which screen archetypes (forms, lists, dashboards, modals, empty surfaces). What each state must show. Anti-patterns (\"just idle\" is never enough)."
---
# State Coverage

A prototype is incomplete if it shows only the "happy path." Real users hit loading, errors, empty data, network failures, disabled actions. Every prototype must render every applicable state.

## States by archetype

| Archetype | Required | Recommended | Optional |
|-----------|----------|-------------|----------|
| **form** (login, signup, settings, contact) | idle, submitting, error | success | disabled, validation-error |
| **list** / **grid** | idle (with data), loading, empty, error | filtered-empty | refreshing, paginating |
| **detail** (entity view) | loading, idle (with data), error | not-found, deleted | editing |
| **dashboard** | loading, idle, error (per data block) | partial-load | refreshing |
| **modal** | open | closing, opening | confirmation-needed |
| **empty** (welcome, no-data) | empty | with-content (visual diff) | — |
| **settings** | idle | submitting, success, error | unsaved-changes |
| **static** (about, terms) | idle | — | — |
| **flow** (multi-step wizard) | step 1 idle, intermediate steps, success | error per step | back-navigation |

## What each state must show

### idle
- Default rendered state
- Empty input fields, default values, no errors
- All actions enabled

### submitting / loading
- Form: button shows spinner or "Sending..." text, fields disabled
- List: skeleton placeholders OR overlay spinner
- Detail: skeleton OR spinner centered
- Critical: indicate that action is in flight; user knows to wait

### error
- Specific error message (use DS voice.samples or contextual)
- Recovery action visible (retry button, "go back" link)
- For forms: error banner above OR inline near affected field
- For lists: empty area with error icon + retry CTA

### empty (lists)
- Illustrative empty state (icon or illustration)
- Helpful message: "No items yet" + "Add your first one" CTA
- Don't show a blank screen

### success
- Confirmation message OR redirect indication ("Logged in. Redirecting...")
- Often transient — user moves on
- Visual: green check, success-toned colors from DS

### disabled
- Button gray, cursor `not-allowed`, no hover effects
- Reason indicated when possible (tooltip or helper text)

### validation-error (form-level)
- Per-field error messages below inputs
- Submit button may be disabled or labeled with count of errors

## How to render multi-state in preview.html

The `templates/prototype-detail.html.tmpl` lays out each frame as a labeled block:

```
─────────────────────────────────────
   IDLE
   The default state of the screen
─────────────────────────────────────
[ rendered idle frame ]


─────────────────────────────────────
   SUBMITTING
   Form being submitted; inputs disabled
─────────────────────────────────────
[ rendered submitting frame ]


─────────────────────────────────────
   ERROR (Invalid credentials)
   401 from API; banner above form
─────────────────────────────────────
[ rendered error frame ]
```

User scrolls down, sees all states without re-running anything. Each frame is a complete render — not just a "diff" or annotation.

## When a state truly doesn't apply

Sometimes a state genuinely doesn't apply. Example: a static "About" page has no loading state because there's no async data.

In `prototype.json`, document the omission:

```json
{
  "frames": {
    "idle": { ... }
  },
  "notes": [
    "No loading/error states: this is a static page with no async data."
  ]
}
```

This makes the omission explicit (not an oversight).

## Anti-patterns

- ❌ Only `idle` for a form. Real users will hit submitting, error, validation. Show them all.
- ❌ Generic "Error" messages with no recovery. Always show what to do next.
- ❌ Skeleton loaders that look identical to the real UI. Make skeletons clearly distinguishable (gray placeholders, no buttons).
- ❌ Empty states with just text. Add an icon or illustration. Empty states are an opportunity to set tone.
- ❌ Error states that hide the form. Keep the form visible so user can fix and retry.

## Quick checklist before declaring "done"

- [ ] Every applicable state from the table is in `prototype.json.frames`
- [ ] Each frame has a `description` explaining when it shows
- [ ] Error frame has a recovery action visible
- [ ] Empty frame has a CTA to fill it
- [ ] Loading frame indicates progress (spinner or skeleton)
- [ ] Disabled actions show a reason or hint when possible
