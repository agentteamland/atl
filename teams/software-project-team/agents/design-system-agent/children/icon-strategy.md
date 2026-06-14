---
knowledge-base-summary: "Icon set selection and consistency rules. Size scale (16, 20, 24, 32). Color inheritance. Custom icon guidelines (SVG, stroke width). Flutter and React usage patterns."
---
# Icon Strategy

Consistent icon usage across the entire application. One icon set, one style, one set of sizes. Visual consistency in icons is immediately noticeable when broken.

## Icon Set Selection

Pick ONE icon set for the entire project. Do not mix icon sets — mixing Material Icons with Heroicons creates visual dissonance.

### Recommended Icon Sets

**Material Symbols (Recommended for Flutter-first projects)**
- Largest set: 2,500+ icons
- Three styles: Outlined, Rounded, Sharp
- Native Flutter integration via `Icons` class
- Variable icon support (weight, fill, grade, optical size)
- License: Apache 2.0

**Lucide (Recommended for React-first projects)**
- 1,400+ icons, actively maintained fork of Feather Icons
- Clean, minimal 24x24 design with 2px stroke
- Excellent React package (`lucide-react`)
- License: ISC

**Heroicons (Good for Tailwind projects)**
- 300+ icons, designed by Tailwind Labs
- Two styles: Outline (24x24, 1.5px stroke), Solid (20x20)
- Tight Tailwind integration
- License: MIT

### Selection Criteria

1. **Coverage** — Does the set have all the icons the project needs? (navigation, actions, status, content types)
2. **Style match** — Does the icon style match the app's visual tone? (rounded = friendly, sharp = professional)
3. **Platform integration** — How easy is it to use in both Flutter and React?
4. **Maintenance** — Is the icon set actively maintained?
5. **Consistency** — Does every icon in the set share the same visual weight?

### Decision Rule

- Flutter-only or Flutter-primary project: **Material Symbols (Outlined)**
- React-only or React-primary project: **Lucide**
- Tailwind-heavy web project: **Heroicons**
- Cross-platform with equal weight: **Material Symbols** (largest coverage)

## Icon Sizes

Four standard sizes. No other sizes allowed.

```
Size    px    Usage
──────────────────────────────────────────────
16      16    Inline with small text (label, caption), compact UI
20      20    Inline with body text, form field icons, list item icons
24      24    Standard size — navigation, toolbar, action buttons
32      32    Prominent icons — empty states, feature highlights
```

### Size Rules

- **Default size is 24** — if no size is specified, use 24
- **Icons in buttons** use the same size as the button's text level: label-lg text = 20px icon
- **Navigation bar icons**: 24 on mobile, 24 on web sidebar
- **Tab bar icons**: 24
- **Never scale icons arbitrarily** — 18px, 22px, 28px are not in the system

## Icon Color

**Icons follow text color.** This is the fundamental rule.

```
Primary text context   → icon uses on-surface color
Secondary text context → icon uses on-surface-variant color
Primary button         → icon uses on-primary color
Error context          → icon uses error color
Disabled context       → icon uses on-surface with 0.38 opacity
```

### Color Rules

1. **Icons never have their own hardcoded color** — they inherit from context
2. **Active/selected state**: icon uses `primary` color
3. **Inactive/unselected state**: icon uses `on-surface-variant`
4. **Destructive action**: icon uses `error` color
5. **Interactive icons must meet 3:1 contrast ratio** against their background

## Custom Icons

When the chosen icon set does not have a needed icon, create a custom one following these rules:

### SVG Format Requirements

```
Canvas size:    24x24 (match the standard size)
Stroke width:   2px (match Material Outlined / Lucide)
                1.5px (match Heroicons Outline)
Stroke caps:    Round
Stroke joins:   Round
Fill:           None (outline style) or Solid (filled style) — match chosen set
Corner radius:  Match the set's corner treatment
Padding:        2px from canvas edge (20x20 visible area in 24x24 canvas)
Export:          SVG, optimized (no metadata, no editor artifacts)
```

### Naming Convention

```
icon-[category]-[name].svg

Examples:
icon-nav-home.svg
icon-action-scan.svg
icon-status-offline.svg
```

### Custom Icon Checklist

- [ ] Same canvas size as the icon set (24x24)
- [ ] Same stroke width as the icon set
- [ ] Same visual weight as other icons in the set
- [ ] Looks natural alongside standard icons
- [ ] Works at all 4 sizes (16, 20, 24, 32)
- [ ] Works in both light and dark mode
- [ ] SVG optimized (SVGO or equivalent)

## Flutter Implementation

**Using Material Symbols (default):**
```dart
// Standard usage
Icon(
  Icons.home_outlined,
  size: 24,
  color: Theme.of(context).colorScheme.onSurface,
)

// In a navigation bar
NavigationBar(
  destinations: [
    NavigationDestination(
      icon: Icon(Icons.home_outlined),
      selectedIcon: Icon(Icons.home),
      label: 'Home',
    ),
  ],
)

// Icon sizes as constants
class AppIconSize {
  static const double sm = 16;
  static const double md = 20;
  static const double lg = 24;  // default
  static const double xl = 32;
}
```

**Using custom SVG icons:**
```dart
// With flutter_svg package
SvgPicture.asset(
  'assets/icons/icon-action-scan.svg',
  width: 24,
  height: 24,
  colorFilter: ColorFilter.mode(
    Theme.of(context).colorScheme.onSurface,
    BlendMode.srcIn,
  ),
)
```

## React Implementation

**Using Lucide:**
```tsx
import { Home, Search, Settings, AlertCircle } from 'lucide-react';

// Standard usage
<Home size={24} className="text-foreground" />

// With semantic color
<AlertCircle size={20} className="text-error-500" />

// Icon button
<button className="p-2 rounded-md hover:bg-neutral-100">
  <Settings size={24} className="text-neutral-600" />
</button>
```

**Using Heroicons:**
```tsx
import { HomeIcon, MagnifyingGlassIcon } from '@heroicons/react/24/outline';
import { HomeIcon as HomeIconSolid } from '@heroicons/react/24/solid';

// Outline (default)
<HomeIcon className="h-6 w-6 text-neutral-600" />

// Solid (active state)
<HomeIconSolid className="h-6 w-6 text-primary-500" />
```

**Icon size utility:**
```tsx
const iconSize = {
  sm: 'h-4 w-4',   // 16px
  md: 'h-5 w-5',   // 20px
  lg: 'h-6 w-6',   // 24px
  xl: 'h-8 w-8',   // 32px
} as const;

<Home className={iconSize.lg} />
```

## Rules

1. **One icon set per project** — never mix Material and Lucide in the same app
2. **Four sizes only** — 16, 20, 24, 32. No exceptions
3. **Color follows text** — icons do not have independent colors
4. **No decorative icons** — every icon must serve a functional purpose or aid comprehension
5. **Accessible** — icon-only buttons must have an aria-label or tooltip
6. **Consistent style** — if the project uses outlined icons, ALL icons are outlined (no mixing outline and filled except for active/inactive states)
