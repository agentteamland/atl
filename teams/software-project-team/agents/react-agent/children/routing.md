---
knowledge-base-summary: "React Router with layout routes. Auth guard (redirect if not logged in). Lazy loading with React.lazy + Suspense. Nested layouts (sidebar + header + content). Route naming conventions."
---
# Routing

React Router v6 with createBrowserRouter. Layout routes, auth guards, lazy loading.

## Router Setup

```tsx
// src/router.tsx

import { createBrowserRouter, Navigate } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { AppLayout } from '@/components/layout/AppLayout';
import { AuthLayout } from '@/components/layout/AuthLayout';
import { authLoader, guestLoader } from '@/lib/auth-loader';
import { LoadingState } from '@/components/ui/LoadingState';

// Lazy-loaded pages
const DashboardPage = lazy(() =>
  import('@/features/dashboard/pages/DashboardPage').then((m) => ({
    default: m.DashboardPage,
  }))
);
const UsersPage = lazy(() =>
  import('@/features/users/pages/UsersPage').then((m) => ({
    default: m.UsersPage,
  }))
);
const UserDetailPage = lazy(() =>
  import('@/features/users/pages/UserDetailPage').then((m) => ({
    default: m.UserDetailPage,
  }))
);
const CreateUserPage = lazy(() =>
  import('@/features/users/pages/CreateUserPage').then((m) => ({
    default: m.CreateUserPage,
  }))
);
const SettingsPage = lazy(() =>
  import('@/features/settings/pages/SettingsPage').then((m) => ({
    default: m.SettingsPage,
  }))
);
const LoginPage = lazy(() =>
  import('@/features/auth/pages/LoginPage').then((m) => ({
    default: m.LoginPage,
  }))
);
const RegisterPage = lazy(() =>
  import('@/features/auth/pages/RegisterPage').then((m) => ({
    default: m.RegisterPage,
  }))
);
const NotFoundPage = lazy(() =>
  import('@/pages/NotFoundPage').then((m) => ({
    default: m.NotFoundPage,
  }))
);

// Suspense wrapper for lazy routes
function LazyPage({ children }: { children: React.ReactNode }) {
  return <Suspense fallback={<LoadingState />}>{children}</Suspense>;
}

export const router = createBrowserRouter([
  // ─── Auth routes (guest only) ───
  {
    element: <AuthLayout />,
    loader: guestLoader, // Redirect to / if already logged in
    children: [
      {
        path: '/login',
        element: (
          <LazyPage>
            <LoginPage />
          </LazyPage>
        ),
      },
      {
        path: '/register',
        element: (
          <LazyPage>
            <RegisterPage />
          </LazyPage>
        ),
      },
    ],
  },

  // ─── Protected routes (auth required) ───
  {
    path: '/',
    element: <AppLayout />,
    loader: authLoader, // Redirect to /login if not authenticated
    children: [
      {
        index: true,
        element: <Navigate to="/dashboard" replace />,
      },
      {
        path: 'dashboard',
        element: (
          <LazyPage>
            <DashboardPage />
          </LazyPage>
        ),
      },

      // ─── Users feature (nested) ───
      {
        path: 'users',
        children: [
          {
            index: true,
            element: (
              <LazyPage>
                <UsersPage />
              </LazyPage>
            ),
          },
          {
            path: 'new',
            element: (
              <LazyPage>
                <CreateUserPage />
              </LazyPage>
            ),
          },
          {
            path: ':id',
            element: (
              <LazyPage>
                <UserDetailPage />
              </LazyPage>
            ),
          },
        ],
      },

      // ─── Settings ───
      {
        path: 'settings',
        element: (
          <LazyPage>
            <SettingsPage />
          </LazyPage>
        ),
      },
    ],
  },

  // ─── 404 ───
  {
    path: '*',
    element: (
      <LazyPage>
        <NotFoundPage />
      </LazyPage>
    ),
  },
]);
```

## Auth Loader (Route Guard)

```tsx
// src/lib/auth-loader.ts

import { redirect } from 'react-router-dom';
import { useAuthStore } from '@/stores/auth.store';

/**
 * Protects routes that require authentication.
 * If no token exists, redirects to /login with the return URL.
 */
export function authLoader() {
  const { accessToken } = useAuthStore.getState();

  if (!accessToken) {
    const currentPath = window.location.pathname + window.location.search;
    return redirect(`/login?returnTo=${encodeURIComponent(currentPath)}`);
  }

  return null;
}

/**
 * Protects guest-only routes (login, register).
 * If already authenticated, redirects to dashboard.
 */
export function guestLoader() {
  const { accessToken } = useAuthStore.getState();

  if (accessToken) {
    return redirect('/dashboard');
  }

  return null;
}
```

## Layout Routes

Layout routes wrap children with persistent UI (sidebar, header):

```tsx
// src/components/layout/AppLayout.tsx

import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { useUIStore } from '@/stores/ui.store';
import { cn } from '@/lib/cn';

export function AppLayout() {
  const sidebarOpen = useUIStore((s) => s.sidebarOpen);

  return (
    <div className="flex h-screen overflow-hidden bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <Sidebar />

      {/* Main content area */}
      <div
        className={cn(
          'flex flex-1 flex-col overflow-hidden transition-all duration-200',
          sidebarOpen ? 'md:ml-64' : 'md:ml-16'
        )}
      >
        <Header />

        <main className="flex-1 overflow-y-auto p-4 md:p-6 lg:p-8">
          <div className="mx-auto max-w-7xl">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
```

```tsx
// src/components/layout/AuthLayout.tsx

import { Outlet } from 'react-router-dom';

export function AuthLayout() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4 dark:bg-gray-900">
      <div className="w-full max-w-md">
        <Outlet />
      </div>
    </div>
  );
}
```

## Route Params and Search Params

### Route Params (path segments)

```tsx
// Route definition: path: 'users/:id'

import { useParams } from 'react-router-dom';

function UserDetailPage() {
  const { id } = useParams<{ id: string }>();

  // id is string | undefined, always check
  if (!id) return <Navigate to="/users" replace />;

  const { data: user, isLoading } = useUser(id);
  // ...
}
```

### Search Params (query string)

```tsx
import { useSearchParams } from 'react-router-dom';

function UsersPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  // Read params
  const page = Number(searchParams.get('page')) || 1;
  const search = searchParams.get('search') || '';
  const role = searchParams.get('role') || '';

  // Update params (replaces current query string)
  function handlePageChange(newPage: number) {
    setSearchParams((prev) => {
      prev.set('page', String(newPage));
      return prev;
    });
  }

  function handleSearch(query: string) {
    setSearchParams((prev) => {
      if (query) {
        prev.set('search', query);
      } else {
        prev.delete('search');
      }
      prev.set('page', '1'); // Reset to page 1 on new search
      return prev;
    });
  }

  // Pass params to React Query
  const { data } = useUsers({ page, search, role });
}
```

## Programmatic Navigation

```tsx
import { useNavigate } from 'react-router-dom';

function CreateUserForm() {
  const navigate = useNavigate();

  function handleSuccess(userId: string) {
    // Navigate to the new user's detail page
    navigate(`/users/${userId}`);
  }

  function handleCancel() {
    // Go back to previous page
    navigate(-1);
  }

  function handleGoToList() {
    // Replace current history entry (prevent back-button loop)
    navigate('/users', { replace: true });
  }
}
```

## Lazy Loading Best Practices

```tsx
// Always use named exports and remap in the import
const UsersPage = lazy(() =>
  import('@/features/users/pages/UsersPage').then((m) => ({
    default: m.UsersPage,
  }))
);

// Why remap? Because lazy() requires a default export.
// Our pages use named exports for better tree-shaking and explicit imports.
// The .then() adapter bridges the gap without changing the source file.
```

### Route-level code splitting only

```tsx
// GOOD: Split at the route (page) level
const UsersPage = lazy(() => import('@/features/users/pages/UsersPage')...);

// BAD: Don't lazy-load individual components inside a page
// const UserCard = lazy(() => import('../components/UserCard')...);
// This creates too many chunks and network requests
```

## Nested Routes for Feature Areas

For complex features with sub-navigation:

```tsx
// Route config for a settings feature with tabs
{
  path: 'settings',
  element: <SettingsLayout />,  // Has its own sub-navigation
  children: [
    { index: true, element: <Navigate to="general" replace /> },
    { path: 'general', element: <GeneralSettingsPage /> },
    { path: 'profile', element: <ProfileSettingsPage /> },
    { path: 'security', element: <SecuritySettingsPage /> },
    { path: 'notifications', element: <NotificationSettingsPage /> },
    { path: 'billing', element: <BillingSettingsPage /> },
  ],
}
```

```tsx
// src/features/settings/components/SettingsLayout.tsx

import { NavLink, Outlet } from 'react-router-dom';
import { cn } from '@/lib/cn';

const tabs = [
  { label: 'General', path: 'general' },
  { label: 'Profile', path: 'profile' },
  { label: 'Security', path: 'security' },
  { label: 'Notifications', path: 'notifications' },
  { label: 'Billing', path: 'billing' },
];

export function SettingsLayout() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      {/* Tab navigation */}
      <nav className="flex gap-1 border-b border-gray-200 dark:border-gray-700">
        {tabs.map((tab) => (
          <NavLink
            key={tab.path}
            to={tab.path}
            className={({ isActive }) =>
              cn(
                'border-b-2 px-4 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'border-primary-600 text-primary-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              )
            }
          >
            {tab.label}
          </NavLink>
        ))}
      </nav>

      {/* Tab content */}
      <Outlet />
    </div>
  );
}
```

## Entry Point

```tsx
// src/main.tsx

import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { RouterProvider } from 'react-router-dom';
import { QueryClientProvider } from '@tanstack/react-query';
import { queryClient } from '@/lib/query-client';
import { router } from '@/router';
import '@/i18n/config';
import '@/styles/globals.css';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>
);
```

## Rules

1. **All routes defined in one file** (`src/router.tsx`) for visibility
2. **Every page is lazy-loaded** via `React.lazy`
3. **Auth guard via loader** — not a wrapper component
4. **Layout routes for persistent UI** — sidebar/header never unmount on navigation
5. **Named exports** from pages — remap in lazy import
6. **Route params are always `string | undefined`** — validate/parse at the page level
7. **Search params for filters/pagination** — keeps URL shareable and bookmarkable
8. **Navigate with `replace: true`** for redirects to prevent back-button loops
