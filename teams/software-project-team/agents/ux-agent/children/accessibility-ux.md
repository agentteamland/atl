---
knowledge-base-summary: "UX for everyone. Keyboard navigation (Tab/Enter/Space), logical focus order, skip links, meaningful alt text, color-not-only indicators, scalable font sizes, minimum touch targets (44x44), and visible labels for every input."
---
# Accessibility UX

## Core Principle

Accessibility is not a feature — it is a quality of every feature. Every screen, every component, every interaction must be usable by everyone, including people with visual, motor, auditory, and cognitive disabilities. Good accessibility is good UX for all users.

## Keyboard Navigation

### Every Interactive Element Must Be Keyboard-Accessible

**Tab order:**
- Tab moves focus to the next interactive element
- Shift+Tab moves focus to the previous element
- Focus order matches visual layout (left-to-right, top-to-bottom)
- Skip hidden or invisible elements
- Never trap focus (except in modals — see below)

**Activation:**
- Enter/Return: activates buttons, links, submits forms
- Space: activates buttons, toggles checkboxes, expands accordions
- Arrow keys: navigates within groups (tabs, radio buttons, menus, dropdowns)
- Escape: closes modals, dropdowns, menus, cancels current action

### Focus Management

**Focus indicators:**
- Every focused element has a visible focus ring (outline)
- Default browser focus ring is acceptable — never remove it with `outline: none` without replacement
- Custom focus ring: 2px solid, high-contrast color, offset for clarity
- Focus ring must have sufficient contrast ratio (3:1 against background)

**Focus trapping (modals):**
- When a modal opens, focus moves to the first interactive element inside
- Tab cycles within the modal only (cannot Tab out to the background)
- Escape closes the modal
- When modal closes, focus returns to the element that triggered it

**Focus management after actions:**
- Delete an item from a list: focus moves to the next item (or previous if last)
- Submit a form: focus moves to the success message or the next logical section
- Open a dropdown: focus moves to the first option
- Close a dropdown: focus returns to the trigger element
- Navigate to a new page: focus moves to the main heading or content area

## Skip Links

### "Skip to Content" Link

```html
<!-- Visually hidden, appears on first Tab press -->
<a href="#main-content" class="skip-link">Skip to content</a>
```

**Rules:**
- First focusable element on the page
- Visually hidden until focused (then appears prominently)
- Links to the main content area, skipping navigation
- Essential for screen reader users and keyboard navigators
- Add "Skip to navigation" if navigation is at the bottom

### Additional Skip Links (for complex pages)

- "Skip to search" — on pages with prominent search
- "Skip to filters" — on pages with sidebar filters
- "Skip to results" — on search results pages

## Alt Text

### Images

**Informative images (convey content):**
- Alt text describes what the image shows and why it matters
- "Bar chart showing revenue growth from $10M to $15M over Q1-Q4 2024"
- "User avatar for John Smith"
- "Product photo: Blue running shoes, side view"

**Decorative images (no content):**
- Empty alt attribute: `alt=""`
- Screen readers skip these entirely
- Decorative borders, spacers, background patterns

**Functional images (buttons/links):**
- Alt text describes the action, not the image
- Search icon button: `alt="Search"` (not "magnifying glass")
- Logo link: `alt="AppName home"` (not "logo")
- Close button: `alt="Close"` (not "X icon")

### Icons

- Icons with text labels: icon is decorative (`aria-hidden="true"`)
- Icons without text labels: icon needs `aria-label` describing the action
- Icon button (standalone): `aria-label="Delete item"` on the button
- Status icons: combine icon with text or `aria-label` ("Warning: ...")

## Color

### Never Use Color as the Only Indicator

**BAD:** Red text for errors, green text for success (colorblind users can't distinguish)

**GOOD:** Red text + error icon + "Error:" prefix for errors. Green text + check icon + "Success:" prefix for success.

**Patterns to combine with color:**
- Icons (check, X, warning triangle, info circle)
- Text labels ("Error", "Success", "Warning")
- Patterns or textures (in charts and graphs)
- Underlines (for links, in addition to color)
- Bold or weight changes
- Position or shape differences

### Contrast Ratios (WCAG AA)

| Element | Minimum Contrast Ratio |
|---------|----------------------|
| Normal text (< 18px) | 4.5:1 |
| Large text (>= 18px or >= 14px bold) | 3:1 |
| UI components and graphical objects | 3:1 |
| Focus indicators | 3:1 |

**Testing tools:** Use contrast checker tools during design. Test in grayscale to verify nothing depends solely on color.

## Font Size and Scaling

### User Control

**Rules:**
- Users can scale text up to 200% without loss of content or functionality
- Layout must not break when text is scaled
- Don't use fixed pixel sizes for text — use relative units (rem, em)
- Minimum body text: 16px (1rem)
- Minimum UI text (labels, captions): 12px (0.75rem)
- Line height: 1.5x for body text, 1.2x for headings

**Testing:**
- Zoom browser to 200% — all content still accessible
- Increase system font size — layout doesn't break
- No horizontal scrolling needed at 200% zoom (for content up to 320px wide)
- Text doesn't get clipped or hidden behind other elements

## Touch Targets

### Minimum Sizes

| Platform | Minimum Touch Target | Recommended |
|----------|---------------------|-------------|
| iOS (Apple HIG) | 44x44pt | 48x48pt |
| Android (Material) | 48x48dp | 48x48dp |
| Web (WCAG) | 44x44px | 48x48px |

**Rules:**
- Interactive elements must meet minimum touch target size
- Spacing between touch targets: minimum 8px (to prevent mis-taps)
- Small visual elements (icons) can be smaller IF the tap area is padded to minimum size
- Links in text: ensure the tappable area includes the full text + padding
- Form checkboxes and radio buttons: tappable area includes the label text

## Labels and Instructions

### Every Input Has a Visible Label

**Rules:**
- Label must be visible at all times (not just as a placeholder that disappears)
- Label is programmatically associated with the input (`for`/`id` pair or `aria-labelledby`)
- Placeholder text is NOT a substitute for a label
- Required fields indicated with "(required)" text or asterisk `*` with legend
- Error messages linked to the input (`aria-describedby`)

### Form Instructions

- Instructions appear before the form, not after
- Format hints near the field: "MM/DD/YYYY" for date fields
- Group related fields with `fieldset` and `legend` (address fields, radio groups)
- Progress indicators for multi-step forms: "Step 2 of 4: Address Details"

## Screen Reader Considerations

### Semantic HTML

- Use proper heading hierarchy (h1 > h2 > h3, don't skip levels)
- Use `<nav>` for navigation, `<main>` for main content, `<aside>` for sidebar
- Use `<button>` for actions, `<a>` for navigation
- Use `<table>` for tabular data with `<th>` for headers
- Use lists (`<ul>`, `<ol>`) for list content

### Dynamic Content

- Live regions (`aria-live`) for content that updates without page reload
  - `aria-live="polite"` — announced at next pause (toast messages, status updates)
  - `aria-live="assertive"` — announced immediately (error alerts, urgent messages)
- Loading states: announce "Loading..." when content is fetching
- Content changes: announce "3 new results loaded" when infinite scroll adds items
- Form errors: announce errors when they appear

### ARIA Patterns

| Component | Key ARIA | Notes |
|-----------|----------|-------|
| Modal dialog | `role="dialog"`, `aria-modal="true"`, `aria-labelledby` | Focus trap required |
| Tab panel | `role="tablist"`, `role="tab"`, `role="tabpanel"` | Arrow key navigation |
| Accordion | `aria-expanded`, `aria-controls` | Toggle state announced |
| Dropdown menu | `role="menu"`, `role="menuitem"` | Arrow key navigation |
| Alert/toast | `role="alert"` or `aria-live="assertive"` | Auto-announced |
| Progress bar | `role="progressbar"`, `aria-valuenow`, `aria-valuemin`, `aria-valuemax` | Value announced |
| Toggle/switch | `role="switch"`, `aria-checked` | State announced |

## Motion and Animation

### Respect User Preferences

- Check `prefers-reduced-motion` media query
- When reduced motion preferred: disable animations, transitions, parallax
- Essential motion (loading spinners) can remain but simplify
- Never use motion as the only way to convey information
- Auto-playing videos/animations must have pause/stop controls

### Animation Guidelines
- Duration: 200-300ms for UI transitions (not too fast, not too slow)
- No flashing/strobing: never flash more than 3 times per second (seizure risk)
- Meaningful motion: animations should convey meaning (slide in = enters, fade out = departs)
- User-triggered only: avoid auto-playing animations where possible

## Accessibility Checklist

Before any screen is considered complete:

### Perceivable
- [ ] All images have appropriate alt text
- [ ] Color is not the only way to convey information
- [ ] Text meets contrast ratio requirements (4.5:1 normal, 3:1 large)
- [ ] Content is readable when zoomed to 200%
- [ ] No auto-playing audio or video without controls

### Operable
- [ ] All functionality available via keyboard
- [ ] Focus order is logical and visible
- [ ] No keyboard traps (except modals with proper trapping)
- [ ] Touch targets meet minimum size (44x44)
- [ ] Skip links are present
- [ ] No timing requirements without extension options

### Understandable
- [ ] Labels are clear and visible for all inputs
- [ ] Error messages are specific and helpful
- [ ] Instructions appear before forms/actions
- [ ] Language is simple and jargon-free
- [ ] Consistent navigation across pages

### Robust
- [ ] Semantic HTML used throughout
- [ ] ARIA attributes used correctly where needed
- [ ] Works with screen readers (VoiceOver, TalkBack, NVDA)
- [ ] Works with browser zoom up to 200%
- [ ] Dynamic content changes are announced
