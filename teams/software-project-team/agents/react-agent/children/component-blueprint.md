---
knowledge-base-summary: "The primary production unit of this agent. Template + checklist for creating new pages and components. Covers: file structure, TypeScript typing, props interface, responsive design, loading/error states, i18n, testing."
---
# Component Blueprint (Primary Production Unit)

This is the React Agent's primary blueprint. Every new page and component follows this template and checklist.

## File Structure Convention

```
src/
  features/
    {feature}/
      pages/
        {Feature}Page.tsx          # Page component (route target)
      components/
        {Component}.tsx            # Feature-specific component
      hooks/
        use{Feature}.ts            # Feature-specific hooks
      types/
        {feature}.types.ts         # Feature-specific types
      api/
        {feature}.api.ts           # React Query hooks for this feature
  components/
    ui/
      {Component}.tsx              # Shared/reusable components (Button, Input, Card)
    layout/
      {Layout}.tsx                 # Layout components (Sidebar, Header)
  hooks/
    use{Hook}.ts                   # Shared hooks
  lib/
    api-client.ts                  # Axios instance
    utils.ts                       # Utility functions
    cn.ts                          # clsx + tailwind-merge helper
  types/
    api.types.ts                   # Shared API response types
```

## Page Template

Every page follows this exact structure:

```tsx
// src/features/users/pages/UsersPage.tsx

import { useTranslation } from 'react-i18next';
import { UserList } from '../components/UserList';
import { useUsers } from '../api/users.api';
import { PageHeader } from '@/components/layout/PageHeader';
import { ErrorState } from '@/components/ui/ErrorState';
import { LoadingState } from '@/components/ui/LoadingState';
import { EmptyState } from '@/components/ui/EmptyState';

export function UsersPage() {
  const { t } = useTranslation('users');
  const { data: users, isLoading, isError, error, refetch } = useUsers();

  if (isLoading) {
    return <LoadingState message={t('loading')} />;
  }

  if (isError) {
    return (
      <ErrorState
        title={t('error.title')}
        message={error.message}
        onRetry={refetch}
      />
    );
  }

  if (!users || users.length === 0) {
    return (
      <EmptyState
        title={t('empty.title')}
        description={t('empty.description')}
        actionLabel={t('empty.action')}
        onAction={() => {/* navigate to create */}}
      />
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
      />
      <UserList users={users} />
    </div>
  );
}
```

## Component Template

Every component follows this exact structure:

```tsx
// src/features/users/components/UserCard.tsx

import { useTranslation } from 'react-i18next';
import { Card } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Avatar } from '@/components/ui/Avatar';
import type { User } from '../types/users.types';

interface UserCardProps {
  user: User;
  onSelect?: (user: User) => void;
  isSelected?: boolean;
}

export function UserCard({ user, onSelect, isSelected = false }: UserCardProps) {
  const { t } = useTranslation('users');

  return (
    <Card
      className={cn(
        'cursor-pointer transition-colors',
        isSelected && 'ring-2 ring-primary'
      )}
      onClick={() => onSelect?.(user)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onSelect?.(user);
        }
      }}
      aria-selected={isSelected}
    >
      <div className="flex items-center gap-3 p-4">
        <Avatar src={user.avatarUrl} alt={user.name} size="md" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 dark:text-white truncate">
            {user.name}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400 truncate">
            {user.email}
          </p>
        </div>
        <Badge variant={user.isActive ? 'success' : 'secondary'}>
          {user.isActive ? t('status.active') : t('status.inactive')}
        </Badge>
      </div>
    </Card>
  );
}
```

## Shared UI Component Template

```tsx
// src/components/ui/Badge.tsx

import { type VariantProps, cva } from 'class-variance-authority';
import { cn } from '@/lib/cn';

const badgeVariants = cva(
  'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        default: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200',
        primary: 'bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-200',
        success: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
        warning: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
        destructive: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
        secondary: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}
```

## TypeScript Props Interface Pattern

```tsx
// Always define props as an interface, not a type alias
interface UserCardProps {
  // Required props first
  user: User;

  // Optional props with defaults documented
  isSelected?: boolean;        // default: false
  showActions?: boolean;       // default: true

  // Event handlers
  onSelect?: (user: User) => void;
  onDelete?: (userId: string) => void;

  // Children (when needed)
  children?: React.ReactNode;
}

// For components extending HTML elements:
interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  isLoading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}
```

## React Query Hook Pattern

```tsx
// src/features/users/api/users.api.ts

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import type { User, CreateUserDto, UpdateUserDto } from '../types/users.types';

// Query key factory
const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (filters: UserFilters) => [...userKeys.lists(), filters] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

// Queries
export function useUsers(filters?: UserFilters) {
  return useQuery({
    queryKey: userKeys.list(filters ?? {}),
    queryFn: () => apiClient.get<User[]>('/users', { params: filters }),
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => apiClient.get<User>(`/users/${id}`),
    enabled: !!id,
  });
}

// Mutations
export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserDto) =>
      apiClient.post<User>('/users', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserDto }) =>
      apiClient.put<User>(`/users/${id}`, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: userKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient.delete(`/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}
```

## Loading / Error / Empty State Pattern

Every data-fetching component handles all three states:

```tsx
// Pattern: always handle loading, error, empty BEFORE rendering data

function FeaturePage() {
  const { data, isLoading, isError, error, refetch } = useFeatureData();

  // 1. Loading state
  if (isLoading) return <LoadingState />;

  // 2. Error state
  if (isError) return <ErrorState message={error.message} onRetry={refetch} />;

  // 3. Empty state
  if (!data || data.length === 0) return <EmptyState />;

  // 4. Data state (the happy path)
  return <FeatureContent data={data} />;
}
```

## Route Registration

```tsx
// src/router.tsx

import { createBrowserRouter } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { AppLayout } from '@/components/layout/AppLayout';
import { authLoader } from '@/lib/auth-loader';
import { LoadingState } from '@/components/ui/LoadingState';

const UsersPage = lazy(() =>
  import('@/features/users/pages/UsersPage').then((m) => ({ default: m.UsersPage }))
);
const UserDetailPage = lazy(() =>
  import('@/features/users/pages/UserDetailPage').then((m) => ({ default: m.UserDetailPage }))
);

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    loader: authLoader,
    children: [
      {
        path: 'users',
        element: (
          <Suspense fallback={<LoadingState />}>
            <UsersPage />
          </Suspense>
        ),
      },
      {
        path: 'users/:id',
        element: (
          <Suspense fallback={<LoadingState />}>
            <UserDetailPage />
          </Suspense>
        ),
      },
    ],
  },
]);
```

## i18n Usage

```tsx
// src/i18n/locales/en/users.json
{
  "title": "Users",
  "description": "Manage your team members",
  "loading": "Loading users...",
  "error": {
    "title": "Failed to load users",
    "retry": "Try again"
  },
  "empty": {
    "title": "No users yet",
    "description": "Get started by adding your first user",
    "action": "Add User"
  },
  "status": {
    "active": "Active",
    "inactive": "Inactive"
  }
}

// In component:
import { useTranslation } from 'react-i18next';

function UsersPage() {
  const { t } = useTranslation('users');

  return <h1>{t('title')}</h1>;
}
```

## Complete Checklist

Before a page or component is considered DONE, verify every item:

### TypeScript
- [ ] TypeScript strict mode — no `any`, no `@ts-ignore`
- [ ] Props defined as an `interface` (not `type`)
- [ ] API responses typed with Zod schemas
- [ ] Event handlers properly typed (not `(e: any) => ...`)

### Component Quality
- [ ] Component has a single responsibility
- [ ] Props are minimal and well-named
- [ ] Default values for optional props
- [ ] No business logic — only display and user interaction
- [ ] No direct API calls — uses React Query hooks

### Data Fetching
- [ ] Uses React Query (useQuery / useMutation)
- [ ] Loading state handled (skeleton or spinner)
- [ ] Error state handled (error message + retry button)
- [ ] Empty state handled (illustration + message + CTA)

### Styling
- [ ] Tailwind CSS only — no custom CSS
- [ ] Responsive: works on mobile (sm), tablet (md), desktop (lg+)
- [ ] Dark mode: `dark:` variants applied
- [ ] Consistent spacing (using Tailwind scale)

### Accessibility
- [ ] Interactive elements are keyboard accessible
- [ ] `aria-label` or `aria-labelledby` on non-text interactive elements
- [ ] Color contrast meets WCAG AA
- [ ] Focus visible styles present
- [ ] Semantic HTML (button, nav, main, section, etc.)

### Internationalization
- [ ] All user-facing strings use `t()` from react-i18next
- [ ] Translation keys added to all language JSON files
- [ ] No hardcoded strings in JSX

### Testing
- [ ] Component renders without crashing
- [ ] Loading state renders correctly
- [ ] Error state renders correctly
- [ ] User interactions work (click, type, submit)
- [ ] Accessibility: no a11y violations (axe-core)

## Anti-Patterns (NEVER DO)

### 1. Business logic in component
```tsx
// BAD: Calculating discount in component
function ProductCard({ product }: Props) {
  const discount = product.price * 0.15; // Business logic!
  const finalPrice = product.price - discount;
  return <span>{finalPrice}</span>;
}

// GOOD: API returns the calculated value
function ProductCard({ product }: Props) {
  return <span>{product.finalPrice}</span>; // Already calculated by API
}
```

### 2. Fetching in useEffect instead of React Query
```tsx
// BAD: Manual data fetching
function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch('/api/users')
      .then((r) => r.json())
      .then(setUsers)
      .finally(() => setLoading(false));
  }, []);
}

// GOOD: React Query handles everything
function UsersPage() {
  const { data: users, isLoading } = useUsers();
}
```

### 3. Using `any` type
```tsx
// BAD
function handleClick(e: any) { ... }
const data: any = response.data;

// GOOD
function handleClick(e: React.MouseEvent<HTMLButtonElement>) { ... }
const data: User[] = response.data;
```

### 4. Hardcoded strings
```tsx
// BAD
return <h1>Users</h1>;

// GOOD
return <h1>{t('title')}</h1>;
```

### 5. Inline styles
```tsx
// BAD
return <div style={{ marginTop: '16px', color: 'red' }}>...</div>;

// GOOD
return <div className="mt-4 text-red-600">...</div>;
```

### 6. Prop drilling more than 2 levels
```tsx
// BAD: Passing props through 3+ components
<GrandParent user={user}>
  <Parent user={user}>
    <Child user={user} />
  </Parent>
</GrandParent>

// GOOD: Use React Context or Zustand store for deep state
const user = useAuthStore((s) => s.user);
```

### 7. Giant monolith component
```tsx
// BAD: 300+ line component doing everything

// GOOD: Break into smaller components
// UsersPage -> UserList -> UserCard -> UserAvatar
```

## Complete Feature Example

When creating a new feature (e.g., "projects"), create these files in order:

1. `src/features/projects/types/projects.types.ts` — Types
2. `src/features/projects/api/projects.api.ts` — React Query hooks
3. `src/features/projects/components/ProjectCard.tsx` — Components
4. `src/features/projects/components/ProjectList.tsx` — List component
5. `src/features/projects/pages/ProjectsPage.tsx` — Page
6. `src/features/projects/pages/ProjectDetailPage.tsx` — Detail page
7. `src/i18n/locales/en/projects.json` — Translations
8. `src/i18n/locales/tr/projects.json` — Translations (TR)
9. Update `src/router.tsx` — Add routes
10. Update `src/i18n/config.ts` — Register namespace
