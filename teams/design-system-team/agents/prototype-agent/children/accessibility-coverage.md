---
knowledge-base-summary: "WCAG checklist for prototypes: contrast verification (using DS metadata), keyboard navigation, focus visibility, ARIA, touch targets, reduced motion. What to validate before declaring \"done.\""
---
# Accessibility Coverage

A prototype that fails accessibility is incomplete. Use the linked DS's accessibility commitments (`ds.json.accessibility`) as the contract; verify every prototype meets them before declaring "done."

## The non-negotiable checklist

Every prototype must pass before being written:

- [ ] **Color contrast** — body text ≥ 4.5:1, large text ≥ 3:1, UI elements ≥ 3:1. Use the DS's stored contrast metadata (every text-on-color pairing has a precomputed ratio) — don't re-compute manually.
- [ ] **Touch targets** — every interactive element ≥ 44×44px (iOS HIG min) — desktop can go smaller (32×32) but mobile must respect 44.
- [ ] **Focus visible** — every interactive element has a visible focus ring. NEVER `outline: none` without a replacement.
- [ ] **Keyboard navigation** — Tab order is logical; Enter/Space activate buttons; Esc closes modals.
- [ ] **Semantic HTML** — `<button>` for buttons, `<a>` for links, `<input>` with `<label>`, `<form>` for forms. NOT `<div onclick="...">`.
- [ ] **ARIA labels** for icon-only buttons (`aria-label="Close dialog"`).
- [ ] **Form labels** — every input has a `<label>` or `aria-labelledby`.
- [ ] **Error announcements** — error messages use `role="alert"` or `aria-live="polite"`.
- [ ] **Reduced motion respected** — animations honor `prefers-reduced-motion`.

## How to check contrast using DS metadata

The DS's `palette` includes precomputed contrast info per pairing:

```json
"semantic": {
  "text": {
    "primary": "#1A1A1A",
    "primaryOnSurface": { "value": "#1A1A1A", "vs": "#F8F8F6", "ratio": 14.5, "wcag": "AAA" }
  }
}
```

When generating a frame, for each text-on-background pairing you use:
1. Look up the color values in the DS
2. Find the pairing's `ratio` and `wcag` field
3. If `wcag` is "fail" or below the target → flag as accessibility issue, don't proceed without resolving

If the DS doesn't include contrast metadata for a pairing you need, compute it (formula: WCAG 2.1 contrast ratio) — but better: file an issue with ds-architect-agent that the DS is incomplete.

## Touch target sizing

Reference `ds.json.components.<name>.sizes` for canonical heights. Buttons usually have:

```json
"button": {
  "sizes": {
    "sm": { "height": "32px" },
    "md": { "height": "40px" },
    "lg": { "height": "52px" }
  }
}
```

For mobile, prefer `md` or `lg` — `sm` (32px) doesn't meet the 44px touch target unless on desktop.

For non-button interactive elements (links, icons used as buttons), wrap in a tappable area ≥ 44px even if visual is smaller (use padding to extend hit zone).

## Focus styles

Tailwind way to ensure focus ring (use in every interactive element):

```html
<button class="... focus:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary focus-visible:ring-offset-2">
```

Note: `focus-visible:` (not `focus:`) — only shows ring when keyboard-focused, not on mouse click. Better UX.

## Semantic HTML examples

```html
<!-- ✅ Correct -->
<button type="button">Close</button>
<a href="/forgot-password">Forgot password?</a>
<label for="email">E-posta</label>
<input id="email" type="email" />

<!-- ❌ Incorrect -->
<div class="button" onclick="...">Close</div>
<span onclick="navigate(...)">Forgot password?</span>
<input type="email" placeholder="E-posta" />  <!-- placeholder is not a label -->
```

The renderer's job is to emit the correct semantic element per `prototype.json` block type. `block.type === "button"` → `<button>`, `block.type === "link"` → `<a>`, etc.

## Error message announcements

For inline form validation errors:

```html
<input
  id="email"
  aria-invalid="true"
  aria-describedby="email-error"
/>
<p id="email-error" role="alert" class="text-error">
  Enter a valid email address.
</p>
```

For top-of-form alerts:

```html
<div role="alert" class="bg-error-light text-error p-4 rounded-md">
  {{ ds.voice.samples.errorInvalidCredentials }}
</div>
```

## Reduced motion

If you include any animation in the prototype (transitions, micro-interactions), respect `prefers-reduced-motion`:

```css
@media (prefers-reduced-motion: reduce) {
  * {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

The `.dst/styles.css` should include this fallback by default. Don't add custom motion in prototypes that ignores it.

## When you're stuck

If the DS doesn't provide what you need to be accessible (e.g., no error color in palette, no focus-visible token), raise it as a "blocker" in `prototype.json.notes`:

```json
"notes": [
  "Accessibility blocker: DS palette has no semantic.feedback.error color. Used #C04444 as placeholder; please add to DS via /dst-edit-ds primary."
]
```

Then proceed (don't stall the prototype) but make the issue visible.

## Anti-patterns

- ❌ `outline: none` to "clean up" focus styles
- ❌ Color alone for status (use icons + text too)
- ❌ Small text + low contrast for "aesthetic"
- ❌ Skipping focus state because "it's just a prototype"
- ❌ Inventing accessibility values — ALWAYS reference the linked DS's accessibility section
