---
knowledge-base-summary: "Reusable component philosophy. Small, focused components with typed props. Variant pattern (primary/secondary/outline). Composition with children and slots. App-prefix for shared components. Prop extension without duplication."
---
# Component Design

Reusable component philosophy and implementation patterns. Every shared component is small, focused, and typed.

## Core Philosophy

1. **Small and focused** — one component, one job
2. **Typed props** — every prop has a TypeScript type, no `any`
3. **Variants over duplication** — use cva for visual variants
4. **Composition over configuration** — children, render props, slots
5. **Extract on second use** — if you copy-paste, extract immediately

## Naming Conventions

- **PascalCase** for component names: `Button`, `UserCard`, `DataTable`
- **Descriptive** names: `UserAvatarGroup` not `AvatarG`
- **No prefixes** for shared components: `Button` not `AppButton`
- **Feature prefix** only in feature directories: `features/users/components/UserCard.tsx`

## File Organization

```
src/components/
  ui/                     # Atomic UI components
    Button.tsx
    Input.tsx
    Card.tsx
    Badge.tsx
    Avatar.tsx
    Dialog.tsx
    Select.tsx
    Checkbox.tsx
    Tooltip.tsx
    Skeleton.tsx
  layout/                 # Layout components
    AppLayout.tsx
    Sidebar.tsx
    Header.tsx
    PageHeader.tsx
  feedback/               # User feedback components
    LoadingState.tsx
    ErrorState.tsx
    EmptyState.tsx
    Toast.tsx
```

## Variant Pattern with cva (class-variance-authority)

cva is the standard way to create component variants. It provides type-safe variant props and consistent styling.

### Button Component (Full Implementation)

```tsx
// src/components/ui/Button.tsx

import { forwardRef } from 'react';
import { type VariantProps, cva } from 'class-variance-authority';
import { cn } from '@/lib/cn';
import { Loader2 } from 'lucide-react';

const buttonVariants = cva(
  // Base styles (always applied)
  'inline-flex items-center justify-center gap-2 rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        primary:
          'bg-primary-600 text-white hover:bg-primary-700 focus-visible:ring-primary-500',
        secondary:
          'bg-gray-100 text-gray-900 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-100 dark:hover:bg-gray-700',
        outline:
          'border border-gray-300 bg-transparent text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-800',
        ghost:
          'text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800',
        destructive:
          'bg-red-600 text-white hover:bg-red-700 focus-visible:ring-red-500',
        link:
          'text-primary-600 underline-offset-4 hover:underline dark:text-primary-400',
      },
      size: {
        sm: 'h-8 px-3 text-xs',
        md: 'h-10 px-4 text-sm',
        lg: 'h-12 px-6 text-base',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'primary',
      size: 'md',
    },
  }
);

interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  isLoading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant,
      size,
      isLoading = false,
      leftIcon,
      rightIcon,
      disabled,
      children,
      ...props
    },
    ref
  ) => {
    return (
      <button
        ref={ref}
        className={cn(buttonVariants({ variant, size }), className)}
        disabled={disabled || isLoading}
        {...props}
      >
        {isLoading ? (
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        ) : (
          leftIcon
        )}
        {children}
        {!isLoading && rightIcon}
      </button>
    );
  }
);

Button.displayName = 'Button';

// Export variants for external usage (e.g., link styled as button)
export { buttonVariants };
```

### Usage Examples

```tsx
// Primary button (default)
<Button onClick={handleSave}>Save</Button>

// With loading state
<Button isLoading={isPending} onClick={handleSubmit}>
  Submit
</Button>

// Destructive with icon
<Button variant="destructive" leftIcon={<Trash2 className="h-4 w-4" />}>
  Delete
</Button>

// Icon-only button
<Button variant="ghost" size="icon" aria-label="Close">
  <X className="h-4 w-4" />
</Button>

// Outline, small
<Button variant="outline" size="sm">
  Cancel
</Button>
```

### Input Component (Full Implementation)

```tsx
// src/components/ui/Input.tsx

import { forwardRef } from 'react';
import { cn } from '@/lib/cn';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  hint?: string;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, hint, leftIcon, rightIcon, id, ...props }, ref) => {
    const inputId = id || props.name;

    return (
      <div className="space-y-1.5">
        {label && (
          <label
            htmlFor={inputId}
            className="block text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            {label}
            {props.required && (
              <span className="text-red-500 ml-0.5" aria-hidden="true">*</span>
            )}
          </label>
        )}

        <div className="relative">
          {leftIcon && (
            <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400">
              {leftIcon}
            </div>
          )}

          <input
            ref={ref}
            id={inputId}
            className={cn(
              'block w-full rounded-md border bg-white px-3 py-2 text-sm shadow-sm transition-colors',
              'placeholder:text-gray-400',
              'focus:outline-none focus:ring-2 focus:ring-offset-0',
              'disabled:cursor-not-allowed disabled:opacity-50',
              'dark:bg-gray-900 dark:text-white',
              error
                ? 'border-red-500 focus:border-red-500 focus:ring-red-200 dark:focus:ring-red-900'
                : 'border-gray-300 focus:border-primary-500 focus:ring-primary-200 dark:border-gray-600 dark:focus:ring-primary-900',
              leftIcon && 'pl-10',
              rightIcon && 'pr-10',
              className
            )}
            aria-invalid={error ? 'true' : undefined}
            aria-describedby={
              error ? `${inputId}-error` : hint ? `${inputId}-hint` : undefined
            }
            {...props}
          />

          {rightIcon && (
            <div className="absolute inset-y-0 right-0 flex items-center pr-3 text-gray-400">
              {rightIcon}
            </div>
          )}
        </div>

        {error && (
          <p id={`${inputId}-error`} className="text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </p>
        )}

        {!error && hint && (
          <p id={`${inputId}-hint`} className="text-sm text-gray-500 dark:text-gray-400">
            {hint}
          </p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';
```

### Card Component

```tsx
// src/components/ui/Card.tsx

import { forwardRef } from 'react';
import { cn } from '@/lib/cn';

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

const paddingMap = {
  none: '',
  sm: 'p-3',
  md: 'p-4',
  lg: 'p-6',
} as const;

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, padding = 'md', children, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          'rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800',
          paddingMap[padding],
          className
        )}
        {...props}
      >
        {children}
      </div>
    );
  }
);

Card.displayName = 'Card';

// Sub-components for structured cards
export function CardHeader({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('border-b border-gray-200 px-4 py-3 dark:border-gray-700', className)} {...props} />;
}

export function CardContent({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('p-4', className)} {...props} />;
}

export function CardFooter({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('border-t border-gray-200 px-4 py-3 dark:border-gray-700', className)} {...props} />;
}
```

### Avatar Component

```tsx
// src/components/ui/Avatar.tsx

import { useState } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/cn';

const avatarVariants = cva(
  'relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700',
  {
    variants: {
      size: {
        xs: 'h-6 w-6 text-xs',
        sm: 'h-8 w-8 text-sm',
        md: 'h-10 w-10 text-sm',
        lg: 'h-12 w-12 text-base',
        xl: 'h-16 w-16 text-lg',
      },
    },
    defaultVariants: {
      size: 'md',
    },
  }
);

interface AvatarProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof avatarVariants> {
  src?: string | null;
  alt: string;
  fallback?: string;
}

export function Avatar({ src, alt, fallback, size, className, ...props }: AvatarProps) {
  const [hasError, setHasError] = useState(false);

  const initials = fallback || alt
    .split(' ')
    .map((word) => word[0])
    .join('')
    .toUpperCase()
    .slice(0, 2);

  return (
    <div className={cn(avatarVariants({ size }), className)} {...props}>
      {src && !hasError ? (
        <img
          src={src}
          alt={alt}
          className="h-full w-full object-cover"
          onError={() => setHasError(true)}
        />
      ) : (
        <span className="font-medium text-gray-600 dark:text-gray-300" aria-hidden="true">
          {initials}
        </span>
      )}
    </div>
  );
}
```

## Composition Patterns

### Children Pattern
```tsx
interface CardProps {
  children: React.ReactNode;
}

// Usage: content is completely flexible
<Card>
  <h2>Title</h2>
  <p>Any content here</p>
</Card>
```

### Render Props Pattern
```tsx
interface DataListProps<T> {
  items: T[];
  renderItem: (item: T, index: number) => React.ReactNode;
  keyExtractor: (item: T) => string;
}

function DataList<T>({ items, renderItem, keyExtractor }: DataListProps<T>) {
  return (
    <ul className="space-y-2">
      {items.map((item, index) => (
        <li key={keyExtractor(item)}>{renderItem(item, index)}</li>
      ))}
    </ul>
  );
}

// Usage
<DataList
  items={users}
  renderItem={(user) => <UserCard user={user} />}
  keyExtractor={(user) => user.id}
/>
```

### Slot Pattern (Named Children)
```tsx
interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: React.ReactNode;  // Slot for action buttons
  breadcrumb?: React.ReactNode;  // Slot for breadcrumb
}

function PageHeader({ title, description, actions, breadcrumb }: PageHeaderProps) {
  return (
    <div className="space-y-2">
      {breadcrumb}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{title}</h1>
          {description && (
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{description}</p>
          )}
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

// Usage
<PageHeader
  title={t('title')}
  description={t('description')}
  breadcrumb={<Breadcrumb items={[...]} />}
  actions={
    <>
      <Button variant="outline">{t('export')}</Button>
      <Button leftIcon={<Plus className="h-4 w-4" />}>{t('add')}</Button>
    </>
  }
/>
```

## forwardRef Pattern

Use `forwardRef` when a parent component needs DOM access (focus, scroll, measure):

```tsx
import { forwardRef } from 'react';

interface TextAreaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  error?: string;
}

export const TextArea = forwardRef<HTMLTextAreaElement, TextAreaProps>(
  ({ label, error, className, ...props }, ref) => {
    return (
      <div>
        {label && <label className="block text-sm font-medium">{label}</label>}
        <textarea
          ref={ref}
          className={cn('w-full rounded-md border p-2', className)}
          {...props}
        />
        {error && <p className="text-sm text-red-600">{error}</p>}
      </div>
    );
  }
);

TextArea.displayName = 'TextArea';
```

## When to Extract a Component

| Signal | Action |
|--------|--------|
| Copy-pasting JSX | Extract immediately |
| Component exceeds ~100 lines | Split into sub-components |
| 3+ conditional rendering blocks | Extract each branch |
| Reusable in other features | Move to `src/components/ui/` |
| Feature-specific but complex | Keep in `features/{feature}/components/` |

## Rules

1. **Never use `any` in props** — always define an interface
2. **Always set `displayName`** on forwardRef components
3. **Always provide `aria-label`** on icon-only buttons
4. **Default variant values** in cva's `defaultVariants`
5. **Export variants** separately for flexible reuse (e.g., link as button)
6. **One component per file** — no multi-component files (except sub-components like Card + CardHeader)
