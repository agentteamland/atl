---
knowledge-base-summary: "Utility-first approach. Responsive breakpoints (sm/md/lg/xl). Dark mode with dark: variant. Design tokens in tailwind.config.ts. Component variants with clsx/cva. Consistent spacing, typography, colors. Never hardcode colors."
---
# Styling (Tailwind CSS)

Utility-first. No custom CSS. Design tokens in config. Variants with cva.

## Core Philosophy

1. **Utility-first** — compose styles from Tailwind classes directly in JSX
2. **No custom CSS files** — unless truly impossible with utilities (e.g., complex animations)
3. **Design tokens in config** — colors, spacing, fonts defined once in `tailwind.config.ts`
4. **Conditional classes with clsx** — dynamic styling without string concatenation
5. **Component variants with cva** — type-safe variant system for reusable components
6. **cn() helper** — merge conflicting Tailwind classes safely

## cn() Helper — The Foundation

Every project needs this utility. It combines `clsx` (conditional classes) with `tailwind-merge` (resolves conflicting classes):

```tsx
// src/lib/cn.ts

import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

### Why cn() is Critical

```tsx
// Without cn(): conflicting classes cause bugs
<div className={`p-4 ${className}`} />
// If className="p-8", the element has BOTH p-4 and p-8 → unpredictable

// With cn(): last class wins, conflicts resolved
<div className={cn('p-4', className)} />
// If className="p-8", only p-8 applies → correct
```

## clsx for Conditional Classes

```tsx
import { cn } from '@/lib/cn';

// Boolean toggling
<div className={cn(
  'rounded-lg border p-4',                    // Always applied
  isActive && 'border-primary-500 bg-primary-50',  // When active
  isDisabled && 'opacity-50 cursor-not-allowed',    // When disabled
)} />

// Conditional with ternary
<span className={cn(
  'text-sm font-medium',
  status === 'active' ? 'text-green-600' : 'text-gray-500',
)} />

// Object syntax
<button className={cn(
  'px-4 py-2 rounded-md',
  {
    'bg-primary-600 text-white': variant === 'primary',
    'bg-gray-100 text-gray-900': variant === 'secondary',
    'border border-gray-300': variant === 'outline',
  },
)} />
```

## cva (class-variance-authority) for Component Variants

cva is the standard for building variant-based components. It provides type-safe variant props:

```tsx
// src/components/ui/Badge.tsx

import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/cn';

const badgeVariants = cva(
  // Base classes (always applied)
  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset',
  {
    variants: {
      // Each variant is a group of options
      variant: {
        default:     'bg-gray-50 text-gray-700 ring-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:ring-gray-700',
        primary:     'bg-primary-50 text-primary-700 ring-primary-200 dark:bg-primary-900 dark:text-primary-300',
        success:     'bg-green-50 text-green-700 ring-green-200 dark:bg-green-900 dark:text-green-300',
        warning:     'bg-yellow-50 text-yellow-700 ring-yellow-200 dark:bg-yellow-900 dark:text-yellow-300',
        destructive: 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-900 dark:text-red-300',
      },
      size: {
        sm: 'px-2 py-0.5 text-xs',
        md: 'px-2.5 py-0.5 text-xs',
        lg: 'px-3 py-1 text-sm',
      },
    },
    // Default values
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
);

// Props interface extends cva's variant types
interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, size, ...props }: BadgeProps) {
  return (
    <span
      className={cn(badgeVariants({ variant, size }), className)}
      {...props}
    />
  );
}

// Export for external reuse
export { badgeVariants };
```

### cva with Compound Variants

```tsx
const alertVariants = cva(
  'rounded-lg border p-4',
  {
    variants: {
      variant: {
        info: 'border-blue-200 bg-blue-50 text-blue-800',
        success: 'border-green-200 bg-green-50 text-green-800',
        warning: 'border-yellow-200 bg-yellow-50 text-yellow-800',
        error: 'border-red-200 bg-red-50 text-red-800',
      },
      size: {
        sm: 'p-3 text-sm',
        md: 'p-4 text-sm',
        lg: 'p-6 text-base',
      },
      dismissible: {
        true: 'pr-10', // Extra padding for dismiss button
        false: '',
      },
    },
    // Compound: specific combination of variants
    compoundVariants: [
      {
        variant: 'error',
        size: 'lg',
        className: 'border-2', // Thicker border for large errors
      },
    ],
    defaultVariants: {
      variant: 'info',
      size: 'md',
      dismissible: false,
    },
  }
);
```

## Responsive Design

Tailwind uses mobile-first breakpoints. Write base styles for mobile, then layer up:

```tsx
// Breakpoints: sm(640px), md(768px), lg(1024px), xl(1280px), 2xl(1536px)

// Mobile-first approach
<div className="
  grid grid-cols-1        {/* Mobile: 1 column */}
  sm:grid-cols-2          {/* >=640px: 2 columns */}
  lg:grid-cols-3          {/* >=1024px: 3 columns */}
  xl:grid-cols-4          {/* >=1280px: 4 columns */}
  gap-4
" />

// Responsive text
<h1 className="
  text-xl                 {/* Mobile */}
  md:text-2xl             {/* Tablet */}
  lg:text-3xl             {/* Desktop */}
  font-bold
" />

// Show/hide based on screen size
<div className="hidden md:block">Desktop sidebar</div>
<div className="md:hidden">Mobile menu button</div>

// Responsive spacing
<main className="
  px-4                    {/* Mobile */}
  md:px-6                 {/* Tablet */}
  lg:px-8                 {/* Desktop */}
" />

// Responsive flex direction
<div className="
  flex flex-col            {/* Mobile: stack vertically */}
  md:flex-row              {/* Tablet+: side by side */}
  gap-4
" />
```

## Dark Mode

Use the `dark:` variant. Tailwind detects `prefers-color-scheme` or a `.dark` class on `<html>`:

```tsx
// tailwind.config.ts
export default {
  darkMode: 'class', // Toggle via class on <html>
  // OR
  darkMode: 'media', // Follow OS preference
} satisfies Config;

// Usage in components
<div className="
  bg-white                 {/* Light mode */}
  dark:bg-gray-900          {/* Dark mode */}
  text-gray-900
  dark:text-white
  border-gray-200
  dark:border-gray-700
" />

// Dark mode for interactive states
<button className="
  bg-primary-600
  hover:bg-primary-700
  dark:bg-primary-500
  dark:hover:bg-primary-600
" />

// Toggle dark mode via Zustand
function ThemeToggle() {
  const theme = useUIStore((s) => s.theme);
  const setTheme = useUIStore((s) => s.setTheme);

  useEffect(() => {
    const root = document.documentElement;
    if (theme === 'dark' || (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
  }, [theme]);

  return (
    <Button variant="ghost" size="icon" onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>
      {theme === 'dark' ? <Sun /> : <Moon />}
    </Button>
  );
}
```

## Design Tokens in tailwind.config.ts

```tsx
// tailwind.config.ts

import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      // ─── Colors ───
      colors: {
        primary: {
          50:  '#f0f9ff',
          100: '#e0f2fe',
          200: '#bae6fd',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
          950: '#082f49',
        },
        // Semantic colors
        success: '#22c55e',
        warning: '#f59e0b',
        error:   '#ef4444',
        info:    '#3b82f6',
      },

      // ─── Typography ───
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },

      // ─── Spacing ───
      spacing: {
        '4.5': '1.125rem',
        '18':  '4.5rem',
      },

      // ─── Border Radius ───
      borderRadius: {
        '4xl': '2rem',
      },

      // ─── Shadows ───
      boxShadow: {
        'card': '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
        'card-hover': '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
      },

      // ─── Animations ───
      keyframes: {
        'slide-in-right': {
          '0%':   { transform: 'translateX(100%)' },
          '100%': { transform: 'translateX(0)' },
        },
        'fade-in': {
          '0%':   { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
      animation: {
        'slide-in-right': 'slide-in-right 0.2s ease-out',
        'fade-in': 'fade-in 0.15s ease-out',
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),        // Better form element defaults
    require('@tailwindcss/typography'),    // Prose for rich text
  ],
} satisfies Config;
```

## Common Patterns

### Consistent Card

```tsx
<div className="rounded-lg border border-gray-200 bg-white p-4 shadow-card dark:border-gray-700 dark:bg-gray-800">
  {children}
</div>
```

### Truncated Text

```tsx
<p className="truncate">Very long text that will be cut off...</p>
<p className="line-clamp-2">Text limited to 2 lines with ellipsis...</p>
```

### Skeleton Loading

```tsx
<div className="animate-pulse space-y-3">
  <div className="h-4 w-3/4 rounded bg-gray-200 dark:bg-gray-700" />
  <div className="h-4 w-1/2 rounded bg-gray-200 dark:bg-gray-700" />
  <div className="h-4 w-5/6 rounded bg-gray-200 dark:bg-gray-700" />
</div>
```

### Focus Ring

```tsx
<button className="
  focus:outline-none
  focus-visible:ring-2
  focus-visible:ring-primary-500
  focus-visible:ring-offset-2
" />
```

### Divider

```tsx
<div className="border-t border-gray-200 dark:border-gray-700" />
```

### Screen Reader Only

```tsx
<span className="sr-only">Close menu</span>
```

## Anti-Patterns

```tsx
// BAD: Inline styles
<div style={{ marginTop: '16px' }}>

// GOOD: Tailwind utility
<div className="mt-4">

// BAD: Custom CSS for one-off styling
// styles.module.css: .card { padding: 16px; border-radius: 8px; }

// GOOD: Tailwind directly
<div className="rounded-lg p-4">

// BAD: Hardcoded colors
<div className="bg-[#1a73e8]">

// GOOD: Design tokens from config
<div className="bg-primary-600">

// BAD: String concatenation for classes
<div className={`px-4 ${isActive ? 'bg-blue-500' : ''}`}>

// GOOD: cn() or clsx
<div className={cn('px-4', isActive && 'bg-blue-500')}>

// BAD: Arbitrary values everywhere
<div className="w-[347px] mt-[13px]">

// GOOD: Use Tailwind's scale (or extend config if needed)
<div className="w-80 mt-3">
```

## Rules

1. **Never write custom CSS** — if Tailwind can't do it, extend the config
2. **Never hardcode colors** — use design tokens from `tailwind.config.ts`
3. **Mobile-first** — write base for mobile, add `sm:`, `md:`, `lg:` for larger screens
4. **cn() for every conditional class** — never string concatenation
5. **cva for variant components** — never manual if/else class logic
6. **dark: variant on every color** — every bg, text, border needs a dark mode counterpart
7. **Tailwind's spacing scale** — `p-4` not `p-[16px]` (avoid arbitrary values)
8. **@tailwindcss/forms plugin** — for consistent form element styling
