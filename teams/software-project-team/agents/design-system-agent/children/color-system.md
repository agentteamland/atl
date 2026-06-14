---
knowledge-base-summary: "Complete color architecture: brand, neutral scale (50-950), semantic colors. Color roles (surface, on-surface, container). Material 3 mapping. Dark mode color design. Contrast ratio requirements. Code examples for Flutter ColorScheme and Tailwind config."
---
# Color System

Complete color architecture for the design system. Every color in the application must come from this system — no hardcoded hex values anywhere in UI code.

## Color Categories

### 1. Brand Colors

The starting point. Every project has at least a primary brand color. Secondary is optional but recommended.

```
Primary    → Main brand color. Used for: buttons, links, active states, FAB
Secondary  → Accent color. Used for: secondary buttons, highlights, badges
```

### 2. Neutral Scale (Gray)

The backbone of any UI. Used for text, backgrounds, borders, dividers.

```
neutral-50:   #FAFAFA   → Page background (light mode)
neutral-100:  #F5F5F5   → Card background, surface
neutral-200:  #EEEEEE   → Dividers, borders (subtle)
neutral-300:  #E0E0E0   → Disabled backgrounds
neutral-400:  #BDBDBD   → Placeholder text, disabled icons
neutral-500:  #9E9E9E   → Secondary text (light mode)
neutral-600:  #757575   → Icons, helper text
neutral-700:  #616161   → Body text (light mode)
neutral-800:  #424242   → Headings (light mode)
neutral-900:  #212121   → Primary text (light mode)
neutral-950:  #121212   → Surface background (dark mode)
```

### 3. Semantic Colors

Convey meaning. Each has a full tonal palette for versatility.

```
Success (Green):
  success-50:  #E8F5E9   → Success background
  success-100: #C8E6C9   → Success background (hover)
  success-500: #4CAF50   → Success text, icons
  success-700: #388E3C   → Success text (dark bg)
  success-900: #1B5E20   → Success dark

Warning (Amber):
  warning-50:  #FFF8E1   → Warning background
  warning-100: #FFECB3   → Warning background (hover)
  warning-500: #FF9800   → Warning text, icons
  warning-700: #F57C00   → Warning text (dark bg)
  warning-900: #E65100   → Warning dark

Error (Red):
  error-50:  #FFEBEE     → Error background
  error-100: #FFCDD2     → Error background (hover)
  error-500: #F44336     → Error text, icons
  error-700: #D32F2F     → Error text (dark bg)
  error-900: #B71C1C     → Error dark

Info (Blue):
  info-50:  #E3F2FD      → Info background
  info-100: #BBDEFB      → Info background (hover)
  info-500: #2196F3      → Info text, icons
  info-700: #1976D2      → Info text (dark bg)
  info-900: #0D47A1      → Info dark
```

## Color Roles

Material 3 introduces role-based color assignment. Every color has a purpose:

```
Surface Roles:
  surface              → Main background color
  surface-container    → Card, dialog backgrounds
  surface-container-high → Elevated card backgrounds
  surface-container-highest → Top-level elevated surfaces
  on-surface           → Text/icons on surface
  on-surface-variant   → Secondary text on surface

Primary Roles:
  primary              → Primary action color
  on-primary           → Text/icon on primary (usually white)
  primary-container    → Primary tinted background
  on-primary-container → Text/icon on primary-container

Secondary Roles:
  secondary            → Secondary action color
  on-secondary         → Text/icon on secondary
  secondary-container  → Secondary tinted background
  on-secondary-container → Text/icon on secondary-container

Error Roles:
  error                → Error indication color
  on-error             → Text/icon on error
  error-container      → Error tinted background
  on-error-container   → Text/icon on error-container

Outline:
  outline              → Borders, dividers
  outline-variant      → Subtle borders
```

## Dark Mode Color Mapping

Dark mode is NOT just inverting colors. It requires redesign:

```
Light Mode                  →  Dark Mode
─────────────────────────────────────────────────
surface: white (#FFFFFF)    →  surface: #121212
surface-container: #F5F5F5  →  surface-container: #1E1E1E
on-surface: #212121         →  on-surface: #E0E0E0
primary: #1976D2            →  primary: #90CAF9 (lighter!)
on-primary: #FFFFFF         →  on-primary: #003C71
outline: #BDBDBD            →  outline: #424242
```

**Key rules for dark mode colors:**
1. Primary color becomes LIGHTER (not darker) — it needs to stand out against dark backgrounds
2. Surfaces use near-black, not pure black (#000000)
3. Text becomes light gray, not pure white (#FFFFFF) — reduces eye strain
4. Semantic colors shift lighter (error red becomes salmon, success green becomes mint)
5. Desaturate brand colors slightly for dark mode — vibrant colors on dark backgrounds are harsh

Reference: `children/dark-mode.md` for full dark mode strategy.

## Contrast Ratio Requirements (WCAG 2.1 AA)

```
Normal text (< 18sp or < 14sp bold):  4.5:1 minimum
Large text (>= 18sp or >= 14sp bold): 3:1 minimum
UI components (icons, borders):        3:1 minimum
```

**Always verify these combinations:**
- on-surface against surface
- on-primary against primary
- on-secondary against secondary
- on-error against error
- Body text against all surface variants
- Placeholder text against input backgrounds
- Icon colors against their backgrounds

**Tools for checking:**
- WebAIM Contrast Checker
- Figma contrast plugins
- `flutter_test` — custom contrast ratio tests

## Flutter Implementation

```dart
class AppColors {
  // --- Light Scheme ---
  static const lightScheme = ColorScheme(
    brightness: Brightness.light,
    primary: Color(0xFF1976D2),
    onPrimary: Color(0xFFFFFFFF),
    primaryContainer: Color(0xFFBBDEFB),
    onPrimaryContainer: Color(0xFF0D47A1),
    secondary: Color(0xFF455A64),
    onSecondary: Color(0xFFFFFFFF),
    secondaryContainer: Color(0xFFCFD8DC),
    onSecondaryContainer: Color(0xFF263238),
    surface: Color(0xFFFFFFFF),
    onSurface: Color(0xFF212121),
    onSurfaceVariant: Color(0xFF757575),
    error: Color(0xFFD32F2F),
    onError: Color(0xFFFFFFFF),
    errorContainer: Color(0xFFFFCDD2),
    onErrorContainer: Color(0xFFB71C1C),
    outline: Color(0xFFBDBDBD),
    outlineVariant: Color(0xFFE0E0E0),
    shadow: Color(0xFF000000),
    surfaceContainerHighest: Color(0xFFF5F5F5),
  );

  // --- Dark Scheme ---
  static const darkScheme = ColorScheme(
    brightness: Brightness.dark,
    primary: Color(0xFF90CAF9),
    onPrimary: Color(0xFF003C71),
    primaryContainer: Color(0xFF0D47A1),
    onPrimaryContainer: Color(0xFFBBDEFB),
    secondary: Color(0xFFB0BEC5),
    onSecondary: Color(0xFF263238),
    secondaryContainer: Color(0xFF37474F),
    onSecondaryContainer: Color(0xFFCFD8DC),
    surface: Color(0xFF121212),
    onSurface: Color(0xFFE0E0E0),
    onSurfaceVariant: Color(0xFF9E9E9E),
    error: Color(0xFFEF9A9A),
    onError: Color(0xFF7F0000),
    errorContainer: Color(0xFFB71C1C),
    onErrorContainer: Color(0xFFFFCDD2),
    outline: Color(0xFF424242),
    outlineVariant: Color(0xFF303030),
    shadow: Color(0xFF000000),
    surfaceContainerHighest: Color(0xFF1E1E1E),
  );

  // --- Semantic Colors ---
  static const success = Color(0xFF4CAF50);
  static const successLight = Color(0xFFE8F5E9);
  static const warning = Color(0xFFFF9800);
  static const warningLight = Color(0xFFFFF8E1);
  static const info = Color(0xFF2196F3);
  static const infoLight = Color(0xFFE3F2FD);
}
```

## Tailwind Implementation

```typescript
// tailwind.config.ts
export default {
  theme: {
    colors: {
      primary: {
        50: '#E3F2FD',
        100: '#BBDEFB',
        200: '#90CAF9',
        300: '#64B5F6',
        400: '#42A5F5',
        500: '#1976D2',  // default
        600: '#1565C0',
        700: '#0D47A1',
        800: '#0A3A82',
        900: '#072D63',
        950: '#041E44',
      },
      neutral: {
        50: '#FAFAFA',
        100: '#F5F5F5',
        200: '#EEEEEE',
        300: '#E0E0E0',
        400: '#BDBDBD',
        500: '#9E9E9E',
        600: '#757575',
        700: '#616161',
        800: '#424242',
        900: '#212121',
        950: '#121212',
      },
      success: {
        50: '#E8F5E9',
        500: '#4CAF50',
        700: '#388E3C',
      },
      warning: {
        50: '#FFF8E1',
        500: '#FF9800',
        700: '#F57C00',
      },
      error: {
        50: '#FFEBEE',
        500: '#F44336',
        700: '#D32F2F',
      },
      info: {
        50: '#E3F2FD',
        500: '#2196F3',
        700: '#1976D2',
      },
    },
  },
} satisfies Config;
```

## Usage Rules

1. **Never use hex values in component code** — always reference tokens
2. **Never create new colors without adding them to the system** — if you need a new shade, add it to the palette
3. **Always check contrast** before pairing any two colors
4. **Use semantic colors for meaning** — green for success, red for error. Never use red for a non-error purpose
5. **Dark mode is not optional** — every color addition must include a dark variant
