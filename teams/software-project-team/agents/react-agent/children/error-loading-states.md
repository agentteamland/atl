---
knowledge-base-summary: "React Suspense for loading. Error Boundary for crashes. Skeleton loading (shimmer). Retry on error. Empty state with illustration + message + action. Toast for transient errors. Full-page vs inline error."
---
# Error & Loading States

## Philosophy

Every component that fetches data must handle three states: loading, error, and empty. Users should never see a blank screen, a raw error message, or an unexplained spinner. Loading uses skeletons (not spinners) to preserve layout. Errors are either full-page (the whole route failed) or inline (one section failed but the page still works). Transient errors (mutation failed) use toasts.

## When to Use Which

| Situation | Pattern |
|-----------|---------|
| Route-level data loading | React Suspense with skeleton fallback |
| Component crashed (render error) | Error Boundary with recovery |
| First load of a list/table | Skeleton matching the final layout |
| API call failed, page still functional | Toast notification + retry option |
| Entire page data failed | Full-page error with retry button |
| One section of dashboard failed | Inline error card in that section |
| List query returned 0 items | Empty state with illustration + action |
| Mutation in progress | Button loading state (disabled + spinner) |

## React Suspense for Route-Level Loading

```typescript
// routes.tsx
import { Suspense, lazy } from 'react';
import { PageSkeleton } from '@/components/ui/page-skeleton';

const UsersPage = lazy(() => import('@/features/users/users-page'));
const DashboardPage = lazy(() => import('@/features/dashboard/dashboard-page'));

export const routes = [
  {
    path: '/users',
    element: (
      <Suspense fallback={<PageSkeleton variant="table" />}>
        <UsersPage />
      </Suspense>
    ),
  },
  {
    path: '/dashboard',
    element: (
      <Suspense fallback={<PageSkeleton variant="dashboard" />}>
        <DashboardPage />
      </Suspense>
    ),
  },
];
```

## Error Boundary

Catches render-time errors. Wraps route content. Shows a recovery UI instead of a white screen.

```typescript
// components/ui/error-boundary.tsx
import { Component, ErrorInfo, ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.props.onError?.(error, errorInfo);
    // Log to your error tracking service (Sentry, etc.)
    console.error('ErrorBoundary caught:', error, errorInfo);
  }

  private handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <ErrorFallback
          error={this.state.error}
          onReset={this.handleReset}
        />
      );
    }

    return this.props.children;
  }
}

// Default fallback component
function ErrorFallback({
  error,
  onReset,
}: {
  error: Error | null;
  onReset: () => void;
}) {
  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center p-8 text-center">
      <div className="h-16 w-16 rounded-full bg-red-50 flex items-center justify-center mb-4">
        <svg className="h-8 w-8 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
        </svg>
      </div>
      <h2 className="text-lg font-semibold text-gray-900">Something went wrong</h2>
      <p className="mt-2 max-w-md text-sm text-gray-500">
        An unexpected error occurred. Please try again.
      </p>
      {import.meta.env.DEV && error && (
        <pre className="mt-4 max-w-lg overflow-auto rounded bg-gray-100 p-3 text-xs text-left text-gray-700">
          {error.message}
        </pre>
      )}
      <div className="mt-6 flex gap-3">
        <button
          onClick={onReset}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700"
        >
          Try again
        </button>
        <button
          onClick={() => window.location.reload()}
          className="rounded-md border px-4 py-2 text-sm hover:bg-gray-50"
        >
          Reload page
        </button>
      </div>
    </div>
  );
}
```

## Error Boundary Usage

```typescript
// Wrap at layout level for route protection
import { ErrorBoundary } from '@/components/ui/error-boundary';

export function AdminLayout() {
  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-auto p-6">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </main>
    </div>
  );
}
```

## Skeleton Loading Components

Shimmer animation with Tailwind. Skeletons match the shape of the final content.

```typescript
// components/ui/skeleton.tsx
import clsx from 'clsx';

interface SkeletonProps {
  className?: string;
}

// Base skeleton element with shimmer animation
export function Skeleton({ className }: SkeletonProps) {
  return (
    <div
      className={clsx(
        'animate-pulse rounded bg-gray-200',
        className
      )}
    />
  );
}

// Predefined skeleton shapes
export function SkeletonText({ lines = 3, className }: { lines?: number; className?: string }) {
  return (
    <div className={clsx('space-y-2', className)}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={clsx('h-4', i === lines - 1 ? 'w-3/4' : 'w-full')}
        />
      ))}
    </div>
  );
}

export function SkeletonAvatar({ size = 'md' }: { size?: 'sm' | 'md' | 'lg' }) {
  const sizeMap = { sm: 'h-8 w-8', md: 'h-10 w-10', lg: 'h-16 w-16' };
  return <Skeleton className={clsx('rounded-full', sizeMap[size])} />;
}

export function SkeletonCard() {
  return (
    <div className="rounded-lg border border-gray-200 p-4 space-y-3">
      <Skeleton className="h-4 w-2/3" />
      <Skeleton className="h-3 w-full" />
      <Skeleton className="h-3 w-4/5" />
      <div className="flex gap-2 pt-2">
        <Skeleton className="h-6 w-16 rounded-full" />
        <Skeleton className="h-6 w-20 rounded-full" />
      </div>
    </div>
  );
}
```

## Page-Level Skeleton

```typescript
// components/ui/page-skeleton.tsx
import { Skeleton, SkeletonCard } from './skeleton';

interface PageSkeletonProps {
  variant: 'table' | 'dashboard' | 'detail' | 'list';
}

export function PageSkeleton({ variant }: PageSkeletonProps) {
  if (variant === 'table') {
    return (
      <div className="space-y-4 p-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-10 w-32 rounded-md" />
        </div>
        <div className="rounded-lg border border-gray-200">
          {/* Header row */}
          <div className="flex gap-4 border-b px-4 py-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-3 w-20" />
            ))}
          </div>
          {/* Data rows */}
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="flex gap-4 border-b px-4 py-3">
              {Array.from({ length: 5 }).map((_, j) => (
                <Skeleton key={j} className="h-4 w-24" />
              ))}
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (variant === 'dashboard') {
    return (
      <div className="space-y-6 p-6">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="rounded-lg border p-4 space-y-2">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-8 w-24" />
              <Skeleton className="h-3 w-16" />
            </div>
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Skeleton className="h-64 rounded-lg" />
          <Skeleton className="h-64 rounded-lg" />
        </div>
      </div>
    );
  }

  if (variant === 'detail') {
    return (
      <div className="space-y-6 p-6 max-w-2xl">
        <Skeleton className="h-8 w-64" />
        <div className="space-y-4">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-5/6" />
        </div>
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }

  // list variant
  return (
    <div className="space-y-3 p-6">
      <Skeleton className="h-8 w-48" />
      {Array.from({ length: 6 }).map((_, i) => (
        <SkeletonCard key={i} />
      ))}
    </div>
  );
}
```

## React Query Retry Configuration

```typescript
// lib/query-client.ts
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,                    // retry failed queries twice
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10000), // exponential backoff
      staleTime: 5 * 60 * 1000,   // 5 minutes before refetch
      gcTime: 10 * 60 * 1000,     // 10 minutes garbage collection
      refetchOnWindowFocus: false, // don't refetch on tab switch
    },
    mutations: {
      retry: 0,                    // never retry mutations
    },
  },
});
```

## Empty State Component

```typescript
// components/ui/empty-state.tsx
interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  subtitle?: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({ icon, title, subtitle, actionLabel, onAction }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      {icon ? (
        <div className="mb-4">{icon}</div>
      ) : (
        <div className="mb-4 h-16 w-16 rounded-full bg-gray-100 flex items-center justify-center">
          <svg className="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
        </div>
      )}
      <h3 className="text-base font-semibold text-gray-900">{title}</h3>
      {subtitle && <p className="mt-1 max-w-sm text-sm text-gray-500">{subtitle}</p>}
      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700"
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
```

## Full-Page Error Component

For when the entire route's data fails to load.

```typescript
// components/ui/error-state.tsx
interface ErrorStateProps {
  message?: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="mb-4 h-16 w-16 rounded-full bg-red-50 flex items-center justify-center">
        <svg className="h-8 w-8 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <h3 className="text-base font-semibold text-gray-900">Failed to load data</h3>
      <p className="mt-1 max-w-sm text-sm text-gray-500">
        {message ?? 'An unexpected error occurred. Please try again.'}
      </p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-4 rounded-md border px-4 py-2 text-sm hover:bg-gray-50 inline-flex items-center gap-2"
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Try again
        </button>
      )}
    </div>
  );
}
```

## Inline Error Card

For when one section of a dashboard fails but the rest works.

```typescript
// components/ui/inline-error.tsx
interface InlineErrorProps {
  message?: string;
  onRetry?: () => void;
}

export function InlineError({ message, onRetry }: InlineErrorProps) {
  return (
    <div className="rounded-lg border border-red-200 bg-red-50 p-4">
      <div className="flex items-center gap-3">
        <svg className="h-5 w-5 text-red-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <div className="flex-1">
          <p className="text-sm font-medium text-red-800">
            {message ?? 'Failed to load this section'}
          </p>
        </div>
        {onRetry && (
          <button
            onClick={onRetry}
            className="text-sm font-medium text-red-600 hover:text-red-800 underline"
          >
            Retry
          </button>
        )}
      </div>
    </div>
  );
}
```

## Usage Pattern in a Component

```typescript
// features/dashboard/revenue-card.tsx
import { useRevenueSummary } from './use-revenue-summary';
import { InlineError } from '@/components/ui/inline-error';
import { Skeleton } from '@/components/ui/skeleton';

export function RevenueCard() {
  const { data, isLoading, isError, error, refetch } = useRevenueSummary();

  if (isLoading) {
    return (
      <div className="rounded-lg border p-4 space-y-2">
        <Skeleton className="h-3 w-20" />
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-3 w-16" />
      </div>
    );
  }

  if (isError) {
    return <InlineError message="Failed to load revenue data" onRetry={() => refetch()} />;
  }

  return (
    <div className="rounded-lg border p-4">
      <p className="text-sm text-gray-500">Total Revenue</p>
      <p className="text-2xl font-bold">${data.total.toLocaleString()}</p>
      <p className="text-xs text-green-600">+{data.changePercent}% from last month</p>
    </div>
  );
}
```

## Toast for Transient Errors

```typescript
// Mutation error: show toast, page stays functional
const updateUser = useMutation({
  mutationFn: (data: UpdateUserInput) => api.patch(`/users/${userId}`, data),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['users'] });
    toast.success('User updated');
  },
  onError: (error) => {
    toast.error(
      error instanceof AxiosError && error.response?.status === 422
        ? 'Validation failed. Please check your input.'
        : 'Failed to update user. Please try again.'
    );
  },
});
```

## Key Rules

1. **Skeleton over spinner** -- always use skeleton loading that matches the final layout shape. Spinners are reserved for button loading states and infinite scroll "load more" indicators.
2. **Error Boundary at layout level** -- wraps `<Outlet />` in the layout. Catches any render error in child routes.
3. **Full-page error for route failure** -- when the primary data query for a route fails, show full-page ErrorState with retry.
4. **Inline error for section failure** -- when one widget on a dashboard fails, show InlineError in that widget only.
5. **Toast for mutation failure** -- API call failed but the page still works. Show a toast, do not replace the page content.
6. **React Query retry: 2 for queries, 0 for mutations** -- queries are idempotent (safe to retry). Mutations are not.
7. **Every data component handles 3 states** -- loading, error, empty. No exceptions. Check this in code review.
8. **Empty state has context** -- icon + title + subtitle + optional action button. Never just "No data".
9. **Dev-only error details** -- show stack trace only in development (`import.meta.env.DEV`). Production shows generic message.
10. **Exponential backoff** -- retry delay doubles each attempt. Prevents server hammering on transient failures.
