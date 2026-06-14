---
knowledge-base-summary: "Token fidelity is necessary but **not sufficient** for declaring a prototype done. Typography rhythm, group spacing, hierarchy, and the \"would a designer notice?\" test are independent contracts. Run this checklist in the browser AFTER the prototype.json + preview.html are written, BEFORE reporting done. Never substitute a token-match audit for visual review."
---
# Visual Review — mandatory before "done"

Token fidelity (every color/spacing referenced from `ds.json`) is **necessary but not sufficient** for declaring a prototype done. A screen can pass every token-match check and still look broken — typography rhythm, group spacing, and visual hierarchy are independent contracts.

This file is the checklist I run **after** the prototype.json + preview.html are written, BEFORE I report "done" to the user.

## How to run

1. Open `preview.html` in the browser (the user's Live Server at `127.0.0.1:5500/.dst/prototypes/<name>/preview.html`).
2. Look at every frame as a real designer would — not as a script asserting `getComputedStyle()` values.
3. Walk the checklist below frame-by-frame. If anything fails → fix it BEFORE reporting done.

## Checklist (every frame)

### 1. Typography rhythm
- [ ] **Title + subtitle**: gap 4–8px (NEVER 0; NEVER negative margin to "compensate" for parent flex gap).
- [ ] **Section heading + content**: gap 12–16px.
- [ ] **Inline pair (label + helper / title + meta)**: gap 2–4px.
- [ ] Letter-spacing tightens at large sizes (-0.015em at 20px+, 0 below).
- [ ] Line-height: 1.2–1.3 for headings, 1.4–1.5 for body.

**Anti-pattern:** Slapping `margin-top: -8px` on a subtitle because parent's flex gap is too wide. If the gap is wrong, fix the structure (group title+subtitle in a wrapper with its own gap), not with negative margin compensation.

### 2. Form rhythm (form archetype)
- [ ] **Label → input gap**: 6px (matches `patterns.formComposition.layoutTokens.labelToInput`).
- [ ] **Helper → input gap**: 4px (matches `helperToInput`).
- [ ] **Field → field gap**: 16px (matches `fieldGap`).
- [ ] **Section gap**: 32px (matches `sectionGap`) — only when using section headers.
- [ ] **Submit button** has visibly more breathing room above it than the field-to-field gap (`+ 8px margin-top` is a clean way).
- [ ] Checkbox / switch rows are vertically centered with their labels (`align-items: center`, label `font-size` matches the row).

### 3. Card / panel padding
- [ ] Card padding is in the 20–32px range (use `pageTemplate.layoutTokens` if declared).
- [ ] Title-area top padding ≈ side padding (no "title is glued to the top edge" feel).
- [ ] Button-area bottom padding ≈ side padding.
- [ ] Border-radius is from DS (`radii.xl` or larger for primary surfaces).

### 4. Page-level layout
- [ ] Brand block is centered above the card with `gapBrandToCard` from the page template (typically 24px).
- [ ] Below-card links have `gapCardToBelow` separation (typically 16px).
- [ ] Card max-width matches `pageTemplate.layoutTokens.cardMaxWidth` (typically 360–480px).
- [ ] Page padding reserves space at small viewports — card never touches the viewport edge.

### 5. State-specific visual hierarchy
- [ ] **Idle**: nothing screaming for attention. Primary CTA is the only visually heavy element.
- [ ] **Submitting**: fields visibly disabled (lower contrast), button shows in-progress treatment, NO error/success colors leaking in.
- [ ] **Error**: error recipe at the failing region (top of card for form-level, inline for field-level). Error color is consistent (`feedback.error`); no rogue red shades.
- [ ] **Success**: minimal — confirms without celebrating. Don't double-confirm with both modal AND toast.

### 6. Token-correct-but-broken traps
- [ ] Negative margins on top of flex gap (the trap I fell into). If a gap "feels too wide", restructure the wrapper instead.
- [ ] Color tokens used in the wrong role (e.g., using `feedback.error` for a non-error indicator).
- [ ] Hardcoded fallback values inside `style=""` for "just this one element" — these orphan from token system.
- [ ] Line-height collapsed to `line-height: 1` for "tighter look" — breaks readability.

### 7. The designer-eye test ("would a designer notice?")
After ticking every box above, ask: **would a returning user instinctively trust this screen?** A title glued to its subtitle answers "no" before I can defend the design. Token-fidelity says nothing about this trust.

If anything in the frame causes a 1-second hesitation when scanning it cold → it's a fail, even if every other check passes.

## Wrap-up

If all checks pass: report done. If any fail: fix in `preview.html` (and update `prototype.json` if a token reference needs to change), re-render, re-walk this checklist.

**Never report "done" off the back of a token-match audit alone.** That's the bug this checklist exists to prevent.

## History

This checklist was added 2026-04-26 after the first real login prototype shipped with title-subtitle visual gap = 0px (negative margin compensation). The audit script reported all 6 token-fidelity checks passed and I declared the prototype done. The user (rightfully) flagged the broken visual hierarchy and asked: "you didn't see this?" The honest answer was no — because I'd only run the script, not the visual review. This file ensures that from now on, the visual review is a mandatory step, not an optional supplement.
