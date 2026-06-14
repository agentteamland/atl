---
knowledge-base-summary: "The primary production unit of this agent. Step-by-step workflow for setting up a design system in a new project. From brand colors to Flutter theme + Tailwind config. Includes full checklist to verify nothing is missed."
---
# Design Blueprint

The primary production unit of the Design System Agent. This blueprint defines the step-by-step workflow for setting up a complete design system in any new project.

## When to Use

- Starting a new project that needs a design system
- Auditing an existing project's visual consistency
- Migrating from ad-hoc styling to token-based design

## Step-by-Step Workflow

### Step 1: Define Brand Colors

Collect the brand's primary and secondary colors. These are the starting point for the entire color system.

**Inputs needed:**
- Primary brand color (hex)
- Secondary/accent color (hex, optional)
- Brand guidelines or reference materials (if any)

**Output:**
- Primary color + its tonal palette (50-950)
- Secondary color + its tonal palette (50-950)
- Document both in the design token file

### Step 2: Build the Full Color Palette

From the brand colors, generate the complete palette:

1. **Primary scale** — 50 (lightest) to 950 (darkest), 13 stops
2. **Secondary scale** — same structure
3. **Neutral scale** — gray family for backgrounds, borders, text
4. **Semantic colors** — success (green), warning (amber), error (red), info (blue)
5. **Surface colors** — background, surface, surface-container variants
6. **Dark mode variants** — every color redesigned for dark backgrounds

Reference: `children/color-system.md` for full architecture.

### Step 3: Set Typography Scale

Define the type scale with clear hierarchy:

| Level    | Size  | Weight   | Use Case                     |
|----------|-------|----------|------------------------------|
| Display  | 36-57 | 400      | Hero sections, splash        |
| Heading  | 24-32 | 600      | Page titles, section headers |
| Title    | 18-22 | 600      | Card titles, dialog titles   |
| Body     | 14-16 | 400      | Main content, paragraphs     |
| Label    | 12-14 | 500      | Buttons, form labels, tabs   |
| Caption  | 11-12 | 400      | Helper text, timestamps      |

**Decisions to make:**
- Primary font family (body + UI)
- Secondary font family (headings, optional — same font is fine)
- Line heights for each level
- Letter spacing adjustments

Reference: `children/typography.md` for details.

### Step 4: Define Spacing Scale

Use the 4px grid. Standard scale:

```
spacing-1:   4px    → Tight gaps, icon-to-text
spacing-2:   8px    → Small padding, compact elements
spacing-3:  12px    → Between related items
spacing-4:  16px    → Standard padding, card content
spacing-5:  20px    → Comfortable spacing
spacing-6:  24px    → Section padding
spacing-8:  32px    → Large gaps
spacing-10: 40px    → Section separators
spacing-12: 48px    → Page margins (mobile)
spacing-16: 64px    → Page margins (tablet/desktop)
spacing-20: 80px    → Hero spacing
spacing-24: 96px    → Major sections
```

Reference: `children/spacing-system.md` for usage guidelines.

### Step 5: Choose Icon Set

Pick ONE icon set for the entire project. Consistency is more important than any individual icon.

**Recommended options:**
- Material Icons — largest set, best Flutter integration
- Lucide — clean, modern, good for web
- Heroicons — designed for Tailwind ecosystem

**Document:**
- Chosen set and version
- Standard sizes: 16, 20, 24, 32
- Color rule: icons follow text color

Reference: `children/icon-strategy.md` for details.

### Step 6: Set Elevation / Shadows

Define 6 elevation levels (0-5):

| Level | Usage                           | Shadow                              |
|-------|---------------------------------|--------------------------------------|
| 0     | Flat elements, backgrounds      | none                                 |
| 1     | Cards, raised buttons           | 0 1px 3px rgba(0,0,0,0.12)          |
| 2     | Sticky headers, floating cards  | 0 2px 6px rgba(0,0,0,0.16)          |
| 3     | Dropdowns, popovers             | 0 4px 12px rgba(0,0,0,0.15)         |
| 4     | Modals, bottom sheets           | 0 8px 24px rgba(0,0,0,0.18)         |
| 5     | FAB, critical overlays          | 0 12px 32px rgba(0,0,0,0.22)        |

Reference: `children/elevation-shadow.md` for dark mode shadows.

### Step 7: Create Component Tokens

Standard component measurements:

```
Border Radius:
  none:  0px    → Sharp corners (tables, code blocks)
  sm:    4px    → Subtle rounding (inputs, small buttons)
  md:    8px    → Standard rounding (cards, dialogs)
  lg:   12px    → Prominent rounding (large cards)
  xl:   16px    → Heavy rounding (FAB, pills)
  full: 9999px  → Circular (avatars, badges)

Button Heights:
  sm:  32px    → Compact UI, table actions
  md:  40px    → Standard buttons
  lg:  48px    → Primary actions, mobile CTA

Input Heights:
  sm:  36px    → Compact forms
  md:  44px    → Standard inputs
  lg:  52px    → Mobile-first inputs
```

Reference: `children/component-tokens.md` for full list.

### Step 8: Implement in Flutter

Create the theme files in the Flutter project:

```
lib/
  core/
    theme/
      app_theme.dart          → ThemeData.light() and ThemeData.dark()
      app_colors.dart         → All color tokens as static constants
      app_typography.dart     → TextTheme with Google Fonts
      app_spacing.dart        → Spacing constants (double values)
      app_radius.dart         → BorderRadius constants
      app_shadows.dart        → BoxShadow lists per elevation
```

**Flutter ThemeData structure:**

```dart
class AppTheme {
  static ThemeData light() => ThemeData(
    useMaterial3: true,
    colorScheme: AppColors.lightScheme,
    textTheme: AppTypography.textTheme,
    // ... component themes
  );

  static ThemeData dark() => ThemeData(
    useMaterial3: true,
    colorScheme: AppColors.darkScheme,
    textTheme: AppTypography.textTheme,
    // ... component themes
  );
}
```

### Step 9: Implement in Tailwind

Configure the design tokens in `tailwind.config.ts`:

```typescript
import type { Config } from 'tailwindcss';

export default {
  darkMode: 'class',
  theme: {
    colors: {
      primary: { /* 50-950 scale */ },
      secondary: { /* 50-950 scale */ },
      neutral: { /* 50-950 scale */ },
      success: { /* 50-950 scale */ },
      warning: { /* 50-950 scale */ },
      error: { /* 50-950 scale */ },
      info: { /* 50-950 scale */ },
    },
    spacing: {
      '1': '4px',
      '2': '8px',
      '3': '12px',
      // ... full scale
    },
    borderRadius: {
      none: '0px',
      sm: '4px',
      DEFAULT: '8px',
      lg: '12px',
      xl: '16px',
      full: '9999px',
    },
    // ... typography, shadows, etc.
  },
} satisfies Config;
```

### Step 10: Verify with Checklist

Run through the complete checklist below before declaring the design system ready.

## Checklist

### Colors
- [ ] Primary color defined with full tonal palette (50-950)
- [ ] Secondary color defined with full tonal palette
- [ ] Neutral gray scale defined (50-950)
- [ ] Semantic colors defined: success, warning, error, info
- [ ] Surface colors defined: background, surface, surface-container
- [ ] Dark mode variants for ALL colors (not inverted — redesigned)
- [ ] All text-on-background combinations pass WCAG AA contrast (4.5:1)
- [ ] All large-text combinations pass 3:1 contrast ratio
- [ ] All UI component colors pass 3:1 against adjacent colors

### Typography
- [ ] Font family selected and available via Google Fonts
- [ ] Type scale defined: display, heading, title, body, label, caption
- [ ] Line heights set for each level
- [ ] Letter spacing defined where needed
- [ ] Font weights specified per level
- [ ] Body text minimum 14sp
- [ ] Font renders well on both iOS and Android

### Spacing
- [ ] All values follow 4px grid
- [ ] Spacing tokens named consistently
- [ ] Component internal padding documented
- [ ] Page margin values defined per breakpoint
- [ ] Touch target minimum 44x44dp verified

### Icons
- [ ] One icon set selected for the project
- [ ] Standard sizes documented (16, 20, 24, 32)
- [ ] Icon color rule documented (follows text color)
- [ ] Custom icon guidelines (if any)

### Elevation
- [ ] 6 elevation levels defined (0-5)
- [ ] Shadow values specified for each level
- [ ] Dark mode elevation strategy defined
- [ ] Usage guidelines per level documented

### Component Tokens
- [ ] Border radius scale defined
- [ ] Button heights (sm, md, lg) defined
- [ ] Input heights defined
- [ ] Card padding standardized
- [ ] Modal sizing defined

### Dark Mode
- [ ] Light theme complete and tested
- [ ] Dark theme complete (not just inverted)
- [ ] All color combinations retested for dark mode contrast
- [ ] Surface hierarchy works in dark mode
- [ ] Images and illustrations work in dark mode
- [ ] Elevation expressed via surface brightness in dark mode

### Accessibility
- [ ] All contrast ratios verified (text, large text, UI components)
- [ ] Touch targets meet 44x44dp minimum
- [ ] Focus indicators visible and consistent
- [ ] No information conveyed by color alone
- [ ] Reduced motion considered for animations

### Implementation
- [ ] Flutter theme files created and compiling
- [ ] Tailwind config created and building
- [ ] Both platforms produce visually consistent output
- [ ] No hardcoded values in implementation files
- [ ] All tokens referenced by name, not by value
