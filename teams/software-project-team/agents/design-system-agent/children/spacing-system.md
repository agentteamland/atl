---
knowledge-base-summary: "4px grid with all standard values. When to use which size. Internal padding vs external margin. Platform-specific adjustments. Token naming and code examples for Flutter and Tailwind."
---
# Spacing System

All spacing in the design system follows a 4px base grid. No arbitrary values — every margin, padding, and gap uses a token from this scale.

## The 4px Grid

Every spacing value is a multiple of 4. This creates visual rhythm and consistency.

```
Token       Value   px    Usage
──────────────────────────────────────────────────────────
spacing-1     1     4px   Tight gaps: icon-to-text inline, border offsets
spacing-2     2     8px   Small padding: chip padding, compact list item gap
spacing-3     3    12px   Between related items: form field to helper text
spacing-4     4    16px   Standard padding: card content, input padding
spacing-5     5    20px   Comfortable spacing: between form fields
spacing-6     6    24px   Section padding: card body, dialog content
spacing-8     8    32px   Large gaps: between card groups, section spacing
spacing-10   10    40px   Section separators: between major content blocks
spacing-12   12    48px   Page margins (mobile), large section gaps
spacing-16   16    64px   Page margins (tablet/desktop), hero spacing
spacing-20   20    80px   Hero sections, major visual breaks
spacing-24   24    96px   Top-level section spacing, footer gaps
```

## When to Use Which

### Micro Spacing (4-8px)
- Gap between icon and text label: `spacing-1` (4px)
- Padding inside chips, badges: `spacing-2` (8px)
- Gap between avatar and name: `spacing-2` (8px)
- Inline element separation: `spacing-1` (4px)

### Small Spacing (12-16px)
- Form field to its helper/error text: `spacing-3` (12px)
- Card content padding (all sides): `spacing-4` (16px)
- Gap between list items: `spacing-3` (12px)
- Input internal padding (horizontal): `spacing-4` (16px)
- Button internal padding (horizontal): `spacing-4` (16px)

### Medium Spacing (20-24px)
- Between form fields (vertical): `spacing-5` (20px)
- Card body padding: `spacing-6` (24px)
- Dialog content padding: `spacing-6` (24px)
- Between card title and card content: `spacing-4` (16px)

### Large Spacing (32-48px)
- Between card groups: `spacing-8` (32px)
- Page section separation: `spacing-10` (40px)
- Page horizontal margin (mobile): `spacing-12` (48px)
- Bottom navigation safe area: `spacing-8` (32px)

### XL Spacing (64-96px)
- Page horizontal margin (tablet/desktop): `spacing-16` (64px)
- Hero section vertical padding: `spacing-20` (80px)
- Major section separation: `spacing-24` (96px)

## Internal vs External Spacing

### Component Internal Padding
The space INSIDE a component. Defined by the component's design token.

```
Button:
  horizontal: spacing-4 (16px) for md, spacing-3 (12px) for sm
  vertical:   spacing-2 (8px) for md, spacing-1 (4px) for sm

Card:
  padding: spacing-4 (16px) all sides
  header-to-body gap: spacing-3 (12px)

Input:
  horizontal: spacing-4 (16px)
  vertical:   spacing-3 (12px)

Dialog:
  padding: spacing-6 (24px)
  title-to-content: spacing-4 (16px)
  content-to-actions: spacing-6 (24px)
```

### Component External Margin
The space BETWEEN components. Handled by the layout, not the component itself. Components should have zero external margin by default — the parent layout controls spacing.

```
Stack of buttons:   spacing-3 (12px) gap
Form fields:        spacing-5 (20px) gap
List items:         spacing-3 (12px) gap
Card grid:          spacing-4 (16px) gap
Section to section: spacing-8 (32px) or spacing-10 (40px)
```

## Platform Differences

### Mobile
- Page horizontal margin: `spacing-4` (16px) or `spacing-6` (24px)
- Touch-friendly spacing: add extra padding around tappable areas
- Bottom safe area: at least `spacing-8` (32px) above bottom navigation
- Minimum touch target: 44x44dp (may need spacing-3 around small icons)

### Tablet
- Page horizontal margin: `spacing-16` (64px) or more
- Content max width: constrain to ~720px in the center
- Card grids can use smaller gaps due to more space

### Desktop/Web
- Page horizontal margin: `spacing-16` (64px) to `spacing-24` (96px) or percentage-based
- Sidebar width: 240-280px with `spacing-4` (16px) internal padding
- Content area max-width: 1200px centered

## Flutter Implementation

```dart
class AppSpacing {
  static const double xs   =  4.0;  // spacing-1
  static const double sm   =  8.0;  // spacing-2
  static const double md   = 12.0;  // spacing-3
  static const double base = 16.0;  // spacing-4
  static const double lg   = 20.0;  // spacing-5
  static const double xl   = 24.0;  // spacing-6
  static const double xxl  = 32.0;  // spacing-8
  static const double xxxl = 40.0;  // spacing-10

  // Page margins
  static const double pageMobile  = 16.0;
  static const double pageTablet  = 64.0;

  // Convenience EdgeInsets
  static const cardPadding = EdgeInsets.all(base);
  static const dialogPadding = EdgeInsets.all(xl);
  static const inputPadding = EdgeInsets.symmetric(
    horizontal: base,
    vertical: md,
  );

  // Common gaps for Column/Row
  static const verticalXs  = SizedBox(height: xs);
  static const verticalSm  = SizedBox(height: sm);
  static const verticalMd  = SizedBox(height: md);
  static const verticalBase = SizedBox(height: base);
  static const verticalLg  = SizedBox(height: lg);
  static const verticalXl  = SizedBox(height: xl);
}
```

**Usage in Flutter:**
```dart
Padding(
  padding: AppSpacing.cardPadding,
  child: Column(
    children: [
      Text('Title'),
      AppSpacing.verticalMd,
      Text('Content'),
    ],
  ),
);
```

## Tailwind Implementation

```typescript
// tailwind.config.ts
export default {
  theme: {
    spacing: {
      '0': '0px',
      '1': '4px',    // spacing-1: xs
      '2': '8px',    // spacing-2: sm
      '3': '12px',   // spacing-3: md
      '4': '16px',   // spacing-4: base
      '5': '20px',   // spacing-5: lg
      '6': '24px',   // spacing-6: xl
      '8': '32px',   // spacing-8: xxl
      '10': '40px',  // spacing-10: xxxl
      '12': '48px',
      '16': '64px',
      '20': '80px',
      '24': '96px',
    },
  },
} satisfies Config;
```

**Usage in Tailwind:**
```html
<!-- Card with standard padding -->
<div class="p-4">
  <h3 class="mb-3">Title</h3>
  <p>Content</p>
</div>

<!-- Form fields with vertical gap -->
<div class="flex flex-col gap-5">
  <input class="px-4 py-3" />
  <input class="px-4 py-3" />
</div>

<!-- Page margin responsive -->
<main class="px-4 md:px-16">
  <!-- content -->
</main>
```

## Rules

1. **Every spacing value must be from the scale** — no `padding: 7px` or `margin: 15px`
2. **Components own their internal padding** — they do not own their external margin
3. **Parent layouts control gaps** — use `gap`, `SizedBox`, or margin on the parent, not the child
4. **Mobile needs more touch padding** — when in doubt, add more space on mobile, not less
5. **Consistent direction** — if card padding is 16px, it is 16px on all four sides (unless explicitly asymmetric by design)
6. **12px is the exception** — it breaks the pure "multiples of 8" but is essential for the 4px grid between small elements
