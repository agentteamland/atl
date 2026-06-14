---
knowledge-base-summary: "Border radius scale, button/input/card sizing tokens. Consistent cross-platform values. Token naming convention. Code examples mapping tokens to Flutter ThemeData and Tailwind config."
---
# Component Tokens

Standardized measurements for UI components. These tokens ensure buttons, inputs, cards, and modals look consistent across every screen and every platform.

## Border Radius Scale

```
Token    Value     Usage
──────────────────────────────────────────────────────────────
none     0px       Sharp corners: tables, code blocks, dividers
sm       4px       Subtle rounding: inputs, small buttons, chips
md       8px       Standard rounding: cards, dialogs, dropdowns
lg       12px      Prominent rounding: large cards, image containers
xl       16px      Heavy rounding: FAB, pills, large modals
full     9999px    Circular: avatars, badges, toggle tracks
```

### Radius Rules

- **One radius per component type** — all cards use `md`, all inputs use `sm`
- **Nested elements use equal or smaller radius** — content inside a card (radius md) uses radius sm or none
- **Never mix sharp and rounded** — if the design language is rounded, keep it consistent
- **Images inside rounded containers** need `overflow: hidden` on the container

## Button Tokens

### Heights

```
Size    Height    Font        Icon     Padding (h)    Usage
─────────────────────────────────────────────────────────────────────
sm      32px      label-md    16px     12px           Compact UI, table row actions,
                                                      inline actions
md      40px      label-lg    20px     16px           Standard buttons, form actions,
                                                      dialog actions
lg      48px      label-lg    20px     24px           Primary CTA, mobile bottom buttons,
                                                      hero actions
```

### Button Variants

```
Variant      Background          Text Color           Border
───────────────────────────────────────────────────────────────────
Filled       primary             on-primary           none
Tonal        primary-container   on-primary-container none
Outlined     transparent         primary              1px primary
Text         transparent         primary              none
Destructive  error               on-error             none
```

### Button States

```
State       Opacity/Overlay
───────────────────────────────
Default     1.0
Hover       +0.08 overlay (web only)
Pressed     +0.12 overlay
Focused     +0.12 overlay + focus ring
Disabled    0.38 opacity (whole button)
Loading     show spinner, disable interaction
```

## Input Tokens

### Heights

```
Size    Height    Font        Padding (h)    Label Size    Usage
─────────────────────────────────────────────────────────────────────
sm      36px      body-sm     12px           label-sm      Compact forms, filters
md      44px      body-md     16px           label-md      Standard forms
lg      52px      body-lg     16px           label-lg      Mobile-first forms
```

### Input Variants

```
Variant      Style                                 Usage
───────────────────────────────────────────────────────────────────
Outlined     1px border, no background fill        Default style, most forms
Filled       Tinted background, no border          Material 3 style
Underlined   Bottom border only                    Minimal, compact forms
```

### Input States

```
State       Border Color          Background        Text Color
────────────────────────────────────────────────────────────────
Default     outline               surface           on-surface
Hover       on-surface            surface           on-surface
Focused     primary (2px)         surface           on-surface
Error       error (2px)           error-container   on-surface
Disabled    outline (0.38)        surface (0.38)    on-surface (0.38)
Read-only   outline               surface-container on-surface
```

## Card Tokens

```
Property              Value
──────────────────────────────────────────
Border radius         md (8px)
Padding               spacing-4 (16px) all sides
Elevation             Level 1 (low)
Background            surface-container (light), surface-container (dark)
Border                none (light), 1px outline-variant (dark, optional)
Header padding        spacing-4 top, spacing-3 bottom
Footer padding        spacing-3 top, spacing-4 bottom
Gap (header-body)     spacing-3 (12px)
Gap (body-footer)     spacing-4 (16px)
```

### Card Variants

```
Variant      Elevation    Border              Usage
──────────────────────────────────────────────────────────────
Elevated     Level 1      none                Default card
Outlined     Level 0      1px outline-variant Flat card with border
Filled       Level 0      none                Tinted background card
```

## Modal / Dialog Tokens

```
Property              Value
──────────────────────────────────────────
Border radius         lg (12px)
Padding               spacing-6 (24px)
Elevation             Level 4 (very high)
Max width             480px (small), 640px (medium), 800px (large)
Min width             280px
Backdrop              black 50% opacity
Title font            title-lg
Body font             body-md
Action gap            spacing-3 (12px) between buttons
```

### Modal Sizing

```
Size      Max Width     Usage
──────────────────────────────────────────
sm        400px         Confirmations, simple inputs
md        560px         Forms, detail views
lg        720px         Complex content, tables
full      90vw          Full-width content (mobile: 100%)
```

## Chip / Badge Tokens

```
Component    Height    Radius     Font          Padding (h)
──────────────────────────────────────────────────────────────
Chip         32px      full       label-md      12px
Badge        20px      full       label-sm      6px
Tag          24px      sm         label-sm      8px
```

## Avatar Tokens

```
Size      Dimension    Font Size    Radius
──────────────────────────────────────────
xs        24x24        label-sm     full
sm        32x32        label-md     full
md        40x40        body-md      full
lg        56x56        title-md     full
xl        72x72        title-lg     full
```

## Divider Tokens

```
Property              Value
──────────────────────────────────────────
Height                1px
Color                 outline-variant
Margin (vertical)     spacing-4 (16px) top and bottom
Full-bleed            edge to edge
Inset                 spacing-4 (16px) left margin
```

## Flutter Implementation

```dart
class AppRadius {
  static const double none = 0;
  static const double sm   = 4;
  static const double md   = 8;
  static const double lg   = 12;
  static const double xl   = 16;
  static const double full = 9999;

  static final BorderRadius noneBR = BorderRadius.zero;
  static final BorderRadius smBR   = BorderRadius.circular(sm);
  static final BorderRadius mdBR   = BorderRadius.circular(md);
  static final BorderRadius lgBR   = BorderRadius.circular(lg);
  static final BorderRadius xlBR   = BorderRadius.circular(xl);
  static final BorderRadius fullBR = BorderRadius.circular(full);
}

class AppButtonSize {
  static const double smHeight = 32;
  static const double mdHeight = 40;
  static const double lgHeight = 48;
}

class AppInputSize {
  static const double smHeight = 36;
  static const double mdHeight = 44;
  static const double lgHeight = 52;
}

class AppAvatarSize {
  static const double xs = 24;
  static const double sm = 32;
  static const double md = 40;
  static const double lg = 56;
  static const double xl = 72;
}
```

**Theme component configuration:**
```dart
// In AppTheme
static ThemeData light() => ThemeData(
  // ...
  filledButtonTheme: FilledButtonThemeData(
    style: FilledButton.styleFrom(
      minimumSize: Size(0, AppButtonSize.mdHeight),
      shape: RoundedRectangleBorder(
        borderRadius: AppRadius.smBR,
      ),
      textStyle: AppTypography.textTheme.labelLarge,
    ),
  ),
  inputDecorationTheme: InputDecorationTheme(
    contentPadding: EdgeInsets.symmetric(
      horizontal: AppSpacing.base,
      vertical: AppSpacing.md,
    ),
    border: OutlineInputBorder(
      borderRadius: AppRadius.smBR,
    ),
  ),
  cardTheme: CardThemeData(
    elevation: 1,
    shape: RoundedRectangleBorder(
      borderRadius: AppRadius.mdBR,
    ),
    margin: EdgeInsets.zero,
  ),
  dialogTheme: DialogThemeData(
    shape: RoundedRectangleBorder(
      borderRadius: AppRadius.lgBR,
    ),
  ),
);
```

## Tailwind Implementation

```typescript
// tailwind.config.ts
export default {
  theme: {
    borderRadius: {
      'none': '0px',
      'sm': '4px',
      DEFAULT: '8px',
      'lg': '12px',
      'xl': '16px',
      'full': '9999px',
    },
    extend: {
      height: {
        'btn-sm': '32px',
        'btn-md': '40px',
        'btn-lg': '48px',
        'input-sm': '36px',
        'input-md': '44px',
        'input-lg': '52px',
      },
      width: {
        'modal-sm': '400px',
        'modal-md': '560px',
        'modal-lg': '720px',
      },
      minWidth: {
        'modal': '280px',
      },
    },
  },
} satisfies Config;
```

**Usage in Tailwind:**
```html
<!-- Button md -->
<button class="h-btn-md px-4 rounded-sm text-label-lg">
  Click me
</button>

<!-- Input md -->
<input class="h-input-md px-4 rounded-sm text-body-md border border-neutral-300
              focus:border-primary-500 focus:ring-2 focus:ring-primary-200" />

<!-- Card -->
<div class="rounded p-4 shadow-elevation-1 bg-white">
  <h3 class="text-title-lg mb-3">Card Title</h3>
  <p class="text-body-md">Card content</p>
</div>

<!-- Modal -->
<div class="w-modal-md min-w-modal rounded-lg p-6 shadow-elevation-4 bg-white">
  <h2 class="text-title-lg mb-4">Modal Title</h2>
  <p class="text-body-md mb-6">Modal content</p>
  <div class="flex justify-end gap-3">
    <button class="h-btn-md px-4 rounded-sm">Cancel</button>
    <button class="h-btn-md px-4 rounded-sm bg-primary-500 text-white">Confirm</button>
  </div>
</div>
```

## Token Naming Convention

All tokens follow the pattern: `{component}-{property}-{variant/size}`

```
button-height-sm, button-height-md, button-height-lg
input-height-sm, input-height-md, input-height-lg
card-radius, card-padding, card-elevation
modal-radius, modal-padding, modal-width-sm
avatar-size-xs, avatar-size-sm, avatar-size-md
```

## Rules

1. **All component sizes come from tokens** — no arbitrary 37px button height
2. **Token names are semantic** — `btn-md` not `h-40`
3. **Same token, same value on every platform** — a `md` button is 40px on Flutter AND Tailwind
4. **Three sizes maximum per component** — sm, md, lg. No xs, no xl (except avatar)
5. **Default size is `md`** — if no size is specified, use `md`
6. **Mobile uses `lg` for primary actions** — bigger touch targets on mobile
