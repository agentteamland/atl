---
knowledge-base-summary: "Elevation levels 0-5 with shadow tokens. When to use which level. Material 3 elevation mapping. Dark mode shadows (outline approach). Code examples for both platforms."
---
# Elevation & Shadow

Elevation creates visual hierarchy by simulating depth. Higher elevation = more prominent = more attention. The system defines 6 levels (0-5) with consistent shadow tokens.

## Elevation Levels

```
Level   Name        Shadow                                    Usage
──────────────────────────────────────────────────────────────────────────────
0       Flat        none                                      Backgrounds, inline content
1       Low         0 1px 3px 0 rgba(0,0,0,0.12),            Cards, list items, raised buttons
                    0 1px 2px -1px rgba(0,0,0,0.08)
2       Medium      0 2px 6px 0 rgba(0,0,0,0.16),            Sticky headers, floating cards,
                    0 1px 3px -1px rgba(0,0,0,0.08)           navigation bars
3       High        0 4px 12px 0 rgba(0,0,0,0.15),           Dropdowns, popovers, menus,
                    0 2px 4px -2px rgba(0,0,0,0.08)           search results overlay
4       Very High   0 8px 24px 0 rgba(0,0,0,0.18),           Modals, bottom sheets,
                    0 4px 8px -4px rgba(0,0,0,0.10)           dialogs
5       Highest     0 12px 32px 0 rgba(0,0,0,0.22),          FAB, snackbar, toast,
                    0 6px 12px -4px rgba(0,0,0,0.12)          critical overlays
```

## When to Use Which Level

### Level 0 — Flat
- Page background
- Content areas that sit at the base level
- Table rows, inline forms
- Elements that are "part of the surface"

### Level 1 — Low
- Standard cards (product card, info card, stat card)
- Raised buttons (non-primary)
- List items that need separation
- Toolbar in its default position

### Level 2 — Medium
- Sticky/fixed headers when scrolled
- App bar / navigation bar
- Floating action elements that stay in place
- Cards that need more prominence than level 1

### Level 3 — High
- Dropdown menus
- Autocomplete suggestion lists
- Popovers and tooltips with content
- Date pickers, color pickers

### Level 4 — Very High
- Modal dialogs
- Bottom sheets
- Side drawers (overlay type)
- Full-screen overlays with content

### Level 5 — Highest
- Floating Action Button (FAB)
- Snackbar / Toast notifications
- Elements that must always be on top
- Drag-and-drop items while being dragged

## Material 3 Elevation System

Material 3 uses a tonal elevation approach in addition to shadows:

```
Level   Shadow Elevation   Tonal Surface
──────────────────────────────────────────
0       0dp                surface
1       1dp                surface-container-lowest
2       3dp                surface-container-low
3       6dp                surface-container
4       8dp                surface-container-high
5       12dp               surface-container-highest
```

In Material 3, elevation is expressed BOTH by shadow AND by surface tint (a slight primary color overlay). This is especially important for dark mode where shadows are less visible.

## Dark Mode Shadows

Shadows are nearly invisible on dark backgrounds. Dark mode uses different strategies:

### Strategy 1: Outline (Recommended)
Replace shadows with subtle borders on dark mode:

```
Light mode:  box-shadow: 0 1px 3px rgba(0,0,0,0.12)
Dark mode:   border: 1px solid rgba(255,255,255,0.08)
```

### Strategy 2: Surface Brightness
Higher elevation = lighter surface color (Material 3 approach):

```
Level 0:  surface       → #121212
Level 1:  +3% white     → #1E1E1E
Level 2:  +5% white     → #232323
Level 3:  +8% white     → #2C2C2C
Level 4:  +11% white    → #333333
Level 5:  +14% white    → #383838
```

### Strategy 3: Combine Both
Use lighter surface + subtle shadow for maximum clarity:

```
Dark Level 1: background: #1E1E1E; box-shadow: 0 1px 3px rgba(0,0,0,0.4)
Dark Level 3: background: #2C2C2C; box-shadow: 0 4px 12px rgba(0,0,0,0.5)
```

**Recommended approach:** Strategy 2 (surface brightness) for most elements, with Strategy 1 (outlines) for elements that need crisp boundaries (cards, dropdowns).

## Flutter Implementation

```dart
class AppShadows {
  // --- Light Mode Shadows ---
  static const List<BoxShadow> elevation0 = [];

  static const List<BoxShadow> elevation1 = [
    BoxShadow(
      offset: Offset(0, 1),
      blurRadius: 3,
      color: Color.fromRGBO(0, 0, 0, 0.12),
    ),
    BoxShadow(
      offset: Offset(0, 1),
      blurRadius: 2,
      spreadRadius: -1,
      color: Color.fromRGBO(0, 0, 0, 0.08),
    ),
  ];

  static const List<BoxShadow> elevation2 = [
    BoxShadow(
      offset: Offset(0, 2),
      blurRadius: 6,
      color: Color.fromRGBO(0, 0, 0, 0.16),
    ),
    BoxShadow(
      offset: Offset(0, 1),
      blurRadius: 3,
      spreadRadius: -1,
      color: Color.fromRGBO(0, 0, 0, 0.08),
    ),
  ];

  static const List<BoxShadow> elevation3 = [
    BoxShadow(
      offset: Offset(0, 4),
      blurRadius: 12,
      color: Color.fromRGBO(0, 0, 0, 0.15),
    ),
    BoxShadow(
      offset: Offset(0, 2),
      blurRadius: 4,
      spreadRadius: -2,
      color: Color.fromRGBO(0, 0, 0, 0.08),
    ),
  ];

  static const List<BoxShadow> elevation4 = [
    BoxShadow(
      offset: Offset(0, 8),
      blurRadius: 24,
      color: Color.fromRGBO(0, 0, 0, 0.18),
    ),
    BoxShadow(
      offset: Offset(0, 4),
      blurRadius: 8,
      spreadRadius: -4,
      color: Color.fromRGBO(0, 0, 0, 0.10),
    ),
  ];

  static const List<BoxShadow> elevation5 = [
    BoxShadow(
      offset: Offset(0, 12),
      blurRadius: 32,
      color: Color.fromRGBO(0, 0, 0, 0.22),
    ),
    BoxShadow(
      offset: Offset(0, 6),
      blurRadius: 12,
      spreadRadius: -4,
      color: Color.fromRGBO(0, 0, 0, 0.12),
    ),
  ];

  /// Get shadow by elevation level
  static List<BoxShadow> of(int level) {
    switch (level) {
      case 0: return elevation0;
      case 1: return elevation1;
      case 2: return elevation2;
      case 3: return elevation3;
      case 4: return elevation4;
      case 5: return elevation5;
      default: return elevation0;
    }
  }
}
```

**Usage in Flutter:**
```dart
Container(
  decoration: BoxDecoration(
    color: Theme.of(context).colorScheme.surface,
    borderRadius: BorderRadius.circular(8),
    boxShadow: AppShadows.elevation1,
  ),
  child: ...,
)
```

## Tailwind Implementation

```typescript
// tailwind.config.ts
export default {
  theme: {
    boxShadow: {
      'none': 'none',
      'elevation-1': '0 1px 3px 0 rgba(0,0,0,0.12), 0 1px 2px -1px rgba(0,0,0,0.08)',
      'elevation-2': '0 2px 6px 0 rgba(0,0,0,0.16), 0 1px 3px -1px rgba(0,0,0,0.08)',
      'elevation-3': '0 4px 12px 0 rgba(0,0,0,0.15), 0 2px 4px -2px rgba(0,0,0,0.08)',
      'elevation-4': '0 8px 24px 0 rgba(0,0,0,0.18), 0 4px 8px -4px rgba(0,0,0,0.10)',
      'elevation-5': '0 12px 32px 0 rgba(0,0,0,0.22), 0 6px 12px -4px rgba(0,0,0,0.12)',
    },
  },
} satisfies Config;
```

**Usage in Tailwind:**
```html
<!-- Card with low elevation -->
<div class="shadow-elevation-1 rounded-md bg-white dark:bg-neutral-800 dark:border dark:border-neutral-700">
  Card content
</div>

<!-- Modal with very high elevation -->
<div class="shadow-elevation-4 rounded-lg bg-white dark:bg-neutral-800">
  Modal content
</div>

<!-- Dark mode: outline instead of shadow -->
<div class="shadow-elevation-1 dark:shadow-none dark:border dark:border-white/10">
  Card in dark mode uses border
</div>
```

## Rules

1. **Only 6 levels (0-5)** — no in-between values like "elevation 1.5"
2. **Higher elevation = more important** — do not use level 4 for a regular card
3. **Shadows are subtle** — the goal is depth perception, not decoration
4. **Dark mode needs different treatment** — shadows are mostly invisible on dark backgrounds
5. **Consistent per component type** — all cards use the same elevation level, all modals use the same level
6. **Elevation changes on interaction** — card can go from level 1 to level 2 on hover (web only)
7. **Never combine custom shadows with elevation tokens** — use the system, not ad-hoc shadow values
