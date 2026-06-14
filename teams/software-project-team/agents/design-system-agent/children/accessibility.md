---
knowledge-base-summary: "WCAG 2.1 AA requirements applied to design tokens. Contrast ratios, touch targets, focus indicators, font minimums, screen reader considerations, reduced motion, color-blind safe design."
---
# Accessibility

WCAG 2.1 AA is the minimum standard. Accessibility is not a feature — it is a requirement. Every design token, every color combination, every interactive element must meet these standards.

## Color Contrast Requirements

### WCAG 2.1 AA Ratios

```
Content Type                          Minimum Ratio    How to Check
──────────────────────────────────────────────────────────────────────
Normal text (< 18sp regular)          4.5:1            Text color vs background
Large text (>= 18sp regular)          3:1              Text color vs background
Large text (>= 14sp bold)             3:1              Text color vs background
UI components (icons, borders)        3:1              Component color vs background
Active controls (buttons, inputs)     3:1              Border/fill vs background
Non-text contrast (charts, graphs)    3:1              Adjacent colors
Focus indicators                      3:1              Indicator vs background
```

### Contrast Verification Matrix

Every design system must verify these specific combinations:

```
Combination                          Light Mode    Dark Mode
──────────────────────────────────────────────────────────────
on-surface on surface                 VERIFY        VERIFY
on-surface-variant on surface         VERIFY        VERIFY
on-primary on primary                 VERIFY        VERIFY
on-secondary on secondary             VERIFY        VERIFY
on-error on error                     VERIFY        VERIFY
on-primary-container on p-container   VERIFY        VERIFY
placeholder text on input bg          VERIFY        VERIFY
disabled text on surface              EXEMPT*       EXEMPT*
link color on surface                 VERIFY        VERIFY
error text on surface                 VERIFY        VERIFY
success text on surface               VERIFY        VERIFY
warning text on surface               VERIFY        VERIFY
icon on surface                       VERIFY        VERIFY
icon on primary                       VERIFY        VERIFY
outline on surface                    VERIFY        VERIFY
```

*Disabled elements are exempt from contrast requirements per WCAG, but should still be distinguishable.

### Tools for Contrast Checking

- **WebAIM Contrast Checker** — quick web-based check
- **Figma plugins** — Stark, A11y - Color Contrast Checker
- **Chrome DevTools** — built-in contrast ratio in color picker
- **Custom tests** — write unit tests that verify contrast ratios programmatically

## Touch Target Sizes

### Minimum Sizes

```
Platform     Minimum Target    Recommended Target
──────────────────────────────────────────────────
Mobile       44x44 dp          48x48 dp
Tablet       44x44 dp          44x44 dp
Web          24x24 px          32x32 px
```

### Touch Target Rules

1. **The touch target can be larger than the visible element** — a 24px icon can have a 44dp touch area
2. **No overlapping touch targets** — minimum 8dp gap between adjacent targets
3. **Small text links are problematic** — if a link is inline text, ensure the tap area is at least 44dp tall
4. **Bottom navigation icons** — minimum 44dp wide, ideally 48dp with label
5. **List items** — full width, minimum 48dp height for tappable rows

### Implementation

**Flutter:**
```dart
// Icon button with proper touch target
IconButton(
  icon: Icon(Icons.close, size: 24),
  constraints: BoxConstraints(
    minWidth: 44,
    minHeight: 44,
  ),
  onPressed: () {},
)

// Small text with adequate tap target
InkWell(
  onTap: () {},
  child: Padding(
    padding: EdgeInsets.symmetric(vertical: 10), // expands to 44dp
    child: Text('Tap me', style: TextStyle(fontSize: 14)),
  ),
)
```

**React/Tailwind:**
```html
<!-- Icon button with proper touch target -->
<button class="min-w-[44px] min-h-[44px] flex items-center justify-center">
  <Icon size={24} />
</button>

<!-- Small link with adequate tap target -->
<a href="#" class="inline-flex items-center min-h-[44px] py-2">
  Tap me
</a>
```

## Focus Indicators

Every interactive element must have a visible focus indicator for keyboard navigation.

### Focus Style

```
Default focus ring:
  - Color: primary (or a high-contrast color)
  - Width: 2px
  - Offset: 2px from the element edge
  - Style: solid outline (not box-shadow for better visibility)
```

### Focus Rules

1. **Never remove focus outlines** — `outline: none` without a replacement is an accessibility violation
2. **Focus ring must be visible against all backgrounds** — test on light, dark, and colored surfaces
3. **Focus order must be logical** — tab order follows visual reading order (left-to-right, top-to-bottom)
4. **Skip navigation link** — web apps should have a "Skip to content" link as the first focusable element
5. **Focus trapping in modals** — when a modal is open, focus stays within the modal

### Implementation

**Flutter:**
```dart
// Focus is handled automatically by Material widgets
// Custom focus:
Focus(
  child: Container(
    decoration: BoxDecoration(
      border: Border.all(
        color: Theme.of(context).colorScheme.primary,
        width: 2,
      ),
    ),
  ),
)
```

**Tailwind:**
```html
<!-- Standard focus ring -->
<button class="focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2">
  Button
</button>

<!-- Global focus style in CSS -->
<style>
  *:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
  }
</style>
```

## Font Size Minimums

```
Content Type          Minimum Size     Recommended
──────────────────────────────────────────────────────
Body text             14sp             14-16sp
Labels                12sp             12-14sp
Captions              11sp             12sp
Button text           12sp             14sp
Input text            14sp             16sp (mobile)
Headings              18sp             18-32sp
```

### Why 14sp Minimum for Body

- Below 14sp, text becomes hard to read on mobile screens in real-world conditions (outdoor, moving, older users)
- iOS dynamically scales text with Dynamic Type — 14sp is the base that scales well
- Android accessibility settings can increase text size — small base sizes become even more problematic

## Screen Reader Considerations

### Semantic Structure

```
Requirement                          Implementation
──────────────────────────────────────────────────────────────
Headings have proper hierarchy       h1 > h2 > h3 (web), Semantics (Flutter)
Images have alt text                 alt="description" / Semantics.label
Buttons have labels                  aria-label (web) / Semantics.button (Flutter)
Form fields have labels              <label for=""> / InputDecoration.labelText
Lists use list semantics             <ul>/<ol> / Semantics(container: true)
Live regions for dynamic content     aria-live (web) / Semantics.liveRegion (Flutter)
```

### Icon-Only Buttons

Every icon-only button MUST have a text alternative:

**Flutter:**
```dart
IconButton(
  icon: Icon(Icons.delete),
  tooltip: 'Delete item', // provides semantics
  onPressed: () {},
)
```

**React:**
```tsx
<button aria-label="Delete item">
  <TrashIcon size={24} />
</button>
```

## Reduced Motion Support

Users who experience motion sickness or vestigo can enable "Reduce Motion" in their OS settings. The design system must respect this.

### What to Reduce

```
Full animation                     Reduced motion alternative
──────────────────────────────────────────────────────────────
Slide-in transitions               Fade-in (or instant)
Bouncing/elastic effects           Simple ease
Parallax scrolling                 Static
Auto-playing animations            Paused by default
Loading spinners                   Keep (essential feedback)
Progress bars                      Keep (essential feedback)
```

### Implementation

**Flutter:**
```dart
// Check reduced motion
final reduceMotion = MediaQuery.of(context).disableAnimations;

AnimatedContainer(
  duration: reduceMotion
    ? Duration.zero
    : Duration(milliseconds: 300),
  // ...
)
```

**CSS/Tailwind:**
```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

```html
<!-- Tailwind motion-safe / motion-reduce -->
<div class="motion-safe:animate-fadeIn motion-reduce:opacity-100">
  Content
</div>
```

## Color-Blind Safe Design

8% of men and 0.5% of women have some form of color vision deficiency. The design system must not rely on color alone.

### Rules

1. **Never use color as the only indicator** — always pair with icons, text, or patterns
2. **Error states** — red color + error icon + error text (not just red border)
3. **Success states** — green color + check icon + success text
4. **Charts/graphs** — use patterns or labels in addition to colors
5. **Status indicators** — dot color + text label (not just a colored dot)
6. **Form validation** — red border + icon + text message (not just red border)

### Safe Color Combinations

```
Avoid pairing:
  Red + Green (most common confusion)
  Red + Brown
  Green + Brown
  Blue + Purple (less common but exists)

Safe pairings:
  Blue + Orange
  Blue + Red
  Blue + Yellow
  Purple + Yellow
```

### Testing

- Chrome DevTools: Rendering > Emulate vision deficiencies
- Figma: Color Blind plugin
- Manual check: convert UI to grayscale — if information is lost, add non-color indicators

## Accessibility Checklist

### Per Component
- [ ] Color contrast ratios verified (text, icon, border)
- [ ] Touch target minimum 44x44dp (mobile)
- [ ] Focus indicator visible and high-contrast
- [ ] Screen reader label provided
- [ ] Keyboard navigable (if interactive)
- [ ] Reduced motion alternative exists (if animated)

### Per Screen
- [ ] Heading hierarchy is logical (h1 > h2 > h3)
- [ ] Tab order follows visual order
- [ ] No information conveyed by color alone
- [ ] All images have alt text
- [ ] Form fields have associated labels
- [ ] Error messages are announced to screen readers
- [ ] Loading states are announced to screen readers

### Per Theme
- [ ] All light mode combinations pass contrast check
- [ ] All dark mode combinations pass contrast check
- [ ] Semantic colors are distinguishable in grayscale
- [ ] Focus indicators are visible in both modes
- [ ] Text remains readable at 200% zoom (web)
