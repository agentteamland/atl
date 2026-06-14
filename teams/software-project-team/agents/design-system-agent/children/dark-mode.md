---
knowledge-base-summary: "Dark theme design principles — not inverted, redesigned. Surface hierarchy, text opacity, elevation via surface brightness. Brand color adjustment. Testing checklist for dark mode completeness."
---
# Dark Mode

Dark mode is not "invert the colors." It is a carefully designed alternate theme with its own surface hierarchy, text treatments, and color adjustments. Dark mode is designed from day one, not bolted on later.

## Core Principle

In light mode, depth is created by shadows (darker = further).
In dark mode, depth is created by surface brightness (lighter = higher).

This fundamental inversion changes how every visual element works.

## Surface Hierarchy

Dark mode uses a layered surface system where higher surfaces are LIGHTER:

```
Level   Surface Name                 Color        White Overlay
──────────────────────────────────────────────────────────────────
0       surface                      #121212      0%
1       surface-container-lowest     #171717      +2%
2       surface-container-low        #1C1C1C      +4%
3       surface-container            #1E1E1E      +5%
4       surface-container-high       #232323      +8%
5       surface-container-highest    #282828      +11%
```

### Usage

```
Page background:              surface (#121212)
Card on page:                 surface-container (#1E1E1E)
Element inside card:          surface-container-high (#232323)
Dialog on top of everything:  surface-container-highest (#282828)
```

The rule: **each layer is slightly lighter than the one behind it.** This creates visual depth without relying on shadows.

## Text Opacity Hierarchy

Pure white (#FFFFFF) on dark backgrounds causes eye strain (halation effect). Use reduced opacity:

```
Text Level              Color                   Opacity    Usage
──────────────────────────────────────────────────────────────────
High emphasis           #FFFFFF                 87%        Headings, primary text
Medium emphasis         #FFFFFF                 60%        Body text, secondary
Disabled                #FFFFFF                 38%        Disabled text, hints
```

In practice, use pre-mixed colors:

```
on-surface:            #E0E0E0   (white at 87% on #121212)
on-surface-variant:    #9E9E9E   (white at 60% on #121212)
disabled:              #636363   (white at 38% on #121212)
```

## Brand Color Adjustment for Dark

Brand colors designed for light backgrounds often look harsh on dark backgrounds. Adjustments:

### Primary Color

```
Light mode primary:     #1976D2  (deep blue, strong saturation)
Dark mode primary:      #90CAF9  (light blue, reduced saturation)
```

**Rules:**
1. Use a LIGHTER tint of the primary on dark backgrounds — it needs to stand out against the dark
2. DESATURATE slightly — vibrant colors on dark backgrounds feel aggressive
3. The primary should still be recognizable as the same brand color
4. Maintain sufficient contrast against dark surfaces (3:1 minimum for UI, 4.5:1 for text)

### Semantic Colors

```
Light Mode             Dark Mode              Reasoning
──────────────────────────────────────────────────────────────
error: #D32F2F         error: #EF9A9A         Lighter red, less alarming
success: #388E3C       success: #81C784       Lighter green, still clear
warning: #F57C00       warning: #FFB74D       Lighter amber
info: #1976D2          info: #64B5F6          Lighter blue
```

### Container Colors

In dark mode, container colors (primary-container, error-container) should be DARK tints, not the light pastel versions used in light mode:

```
Light mode:  primary-container: #BBDEFB (light blue wash)
Dark mode:   primary-container: #0D47A1 (deep blue)

Light mode:  error-container: #FFCDD2 (light red wash)
Dark mode:   error-container: #7F0000 (deep red)
```

## Elevation in Dark Mode

### Shadow Replacement

Shadows are invisible on dark backgrounds. Replace with:

**Option 1: Surface brightness (recommended)**
Higher elevation = lighter surface. See Surface Hierarchy above.

**Option 2: Subtle borders**
```
Card border (dark):      1px solid rgba(255, 255, 255, 0.08)
Dropdown border (dark):  1px solid rgba(255, 255, 255, 0.12)
Modal border (dark):     1px solid rgba(255, 255, 255, 0.10)
```

**Option 3: Increased shadow opacity**
If keeping shadows in dark mode, increase opacity significantly:
```
Light: rgba(0, 0, 0, 0.12)
Dark:  rgba(0, 0, 0, 0.40)
```

## Component Adjustments for Dark Mode

### Buttons

```
Filled button:
  Light: bg primary, text on-primary
  Dark:  bg primary (lighter tint), text on-primary (darker)

Outlined button:
  Light: border primary, text primary
  Dark:  border primary (lighter), text primary (lighter)

Tonal button:
  Light: bg primary-container (pastel), text on-primary-container
  Dark:  bg primary-container (deep), text on-primary-container (light)
```

### Inputs

```
Outlined input:
  Light: border outline, bg surface
  Dark:  border outline (lighter), bg surface-container

Filled input:
  Light: bg surface-container, no border
  Dark:  bg surface-container-high, no border
```

### Cards

```
Light: bg surface, shadow elevation-1
Dark:  bg surface-container, border 1px outline-variant OR lighter bg
```

### Dividers

```
Light: 1px outline-variant (#E0E0E0)
Dark:  1px outline-variant (#303030 or rgba(255,255,255,0.08))
```

## Images and Illustrations

### Photos
- Generally fine in dark mode, no adjustment needed
- Avoid pure white (#FFFFFF) borders around images — they glow

### Illustrations
- Line illustrations with black outlines look harsh — consider outline color adjustment
- White backgrounds in SVG illustrations should become transparent or match dark surface
- Consider providing dark-mode versions of key illustrations

### Icons
- Icons using `on-surface` color automatically adapt
- Colored icons (e.g., status indicators) should use the dark-mode-adjusted semantic colors

## Implementation Toggle

### Flutter

```dart
class AppTheme {
  static ThemeData light() => ThemeData(
    brightness: Brightness.light,
    colorScheme: AppColors.lightScheme,
    // ...
  );

  static ThemeData dark() => ThemeData(
    brightness: Brightness.dark,
    colorScheme: AppColors.darkScheme,
    scaffoldBackgroundColor: Color(0xFF121212),
    // ...
  );
}

// In MaterialApp
MaterialApp(
  theme: AppTheme.light(),
  darkTheme: AppTheme.dark(),
  themeMode: ThemeMode.system, // follows OS setting
)
```

### React / Tailwind

```typescript
// tailwind.config.ts
export default {
  darkMode: 'class', // or 'media' for OS-based
  theme: {
    extend: {
      colors: {
        surface: {
          DEFAULT: '#FFFFFF',
          dark: '#121212',
        },
        'surface-container': {
          DEFAULT: '#F5F5F5',
          dark: '#1E1E1E',
        },
      },
    },
  },
} satisfies Config;
```

```html
<!-- Usage -->
<div class="bg-white dark:bg-[#121212]">
  <div class="bg-neutral-100 dark:bg-[#1E1E1E] rounded p-4">
    <h3 class="text-neutral-900 dark:text-neutral-100">Title</h3>
    <p class="text-neutral-700 dark:text-neutral-300">Body text</p>
  </div>
</div>
```

### Theme Toggle

```tsx
// React theme toggle hook
function useTheme() {
  const [theme, setTheme] = useState<'light' | 'dark' | 'system'>('system');

  useEffect(() => {
    const root = document.documentElement;
    if (theme === 'dark') {
      root.classList.add('dark');
    } else if (theme === 'light') {
      root.classList.remove('dark');
    } else {
      // system
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      root.classList.toggle('dark', prefersDark);
    }
  }, [theme]);

  return { theme, setTheme };
}
```

## Testing Checklist

### Visual Verification
- [ ] Page background is dark (#121212), not pure black (#000000)
- [ ] Cards/containers are lighter than the page background
- [ ] Text is light gray (#E0E0E0), not pure white (#FFFFFF)
- [ ] Primary color is lighter/desaturated compared to light mode
- [ ] Semantic colors (error, success, warning) are adjusted for dark
- [ ] Buttons are readable and have adequate contrast
- [ ] Input fields are distinguishable from background
- [ ] Dividers are visible but subtle

### Contrast Verification
- [ ] All text passes 4.5:1 contrast on their dark backgrounds
- [ ] Large text passes 3:1 contrast
- [ ] Icons pass 3:1 contrast against their backgrounds
- [ ] Focus indicators are visible in dark mode
- [ ] Active/selected states are clearly distinguishable
- [ ] Error, success, warning states are clearly visible

### Content Verification
- [ ] Images look correct (no white halos, no harsh borders)
- [ ] Charts/graphs are readable in dark mode
- [ ] Illustrations work or have dark variants
- [ ] Maps (if any) have a dark tile set
- [ ] Video thumbnails display correctly

### Edge Cases
- [ ] Loading skeletons are visible in dark mode
- [ ] Empty states have appropriate dark illustrations
- [ ] Toasts/snackbars are readable
- [ ] Bottom sheets have proper dark surfaces
- [ ] Context menus / dropdowns have proper dark surfaces
- [ ] Date/time pickers work in dark mode
- [ ] Third-party embeds (if any) support dark mode or are acceptable

### System Integration
- [ ] Theme follows OS preference when set to "system"
- [ ] Theme toggle works correctly
- [ ] Theme preference is persisted
- [ ] No flash of wrong theme on load (FOUC)
- [ ] Transition between themes is smooth (if animated)
