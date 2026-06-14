---
knowledge-base-summary: "Role-based UI rendering. Show/hide components based on user role. Route-level guard (admin-only pages). Component-level guard (<CanAccess role=\"admin\">). Permission checking from JWT claims."
---
# Permission Guard: Role-Based Access Control

## Philosophy

The API is the authority on permissions. React only hides or disables UI elements based on roles from the JWT token. This is UX convenience, not security. Every API endpoint independently checks permissions -- if someone bypasses the UI guard, the API rejects the request.

Two levels of guarding:
1. **Route level** -- entire pages are inaccessible (redirect to /unauthorized or home)
2. **Component level** -- specific buttons, menus, sections are hidden or disabled

## Role Types

Roles come from JWT claims, decoded and stored in the auth store at login:

```typescript
// src/types/auth.ts
export type UserRole = 'admin' | 'manager' | 'dealer' | 'user';

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  tenantId?: string;
  permissions?: string[]; // granular permissions (optional)
}
```

## Auth Store (Role Source)

```typescript
// src/stores/auth-store.ts
import { create } from 'zustand';
import { jwtDecode } from 'jwt-decode';
import type { AuthUser, UserRole } from '@/types/auth';

interface JwtPayload {
  sub: string;
  email: string;
  name: string;
  role: UserRole;
  tenantId?: string;
  permissions?: string[];
  exp: number;
}

interface AuthState {
  user: AuthUser | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  setToken: (token: string) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,

  setToken: (token) => {
    const decoded = jwtDecode<JwtPayload>(token);
    set({
      accessToken: token,
      isAuthenticated: true,
      user: {
        id: decoded.sub,
        email: decoded.email,
        name: decoded.name,
        role: decoded.role,
        tenantId: decoded.tenantId,
        permissions: decoded.permissions,
      },
    });
  },

  logout: () => {
    set({ user: null, accessToken: null, isAuthenticated: false });
  },
}));
```

## Permission Matrix

A centralized map of routes to required roles. Single source of truth for both route guards and sidebar rendering.

```typescript
// src/config/permissions.ts
import type { UserRole } from '@/types/auth';

interface RoutePermission {
  path: string;
  roles: UserRole[];
  label: string;          // for sidebar/menu display
  icon?: string;
}

export const ROUTE_PERMISSIONS: RoutePermission[] = [
  { path: '/dashboard',       roles: ['admin', 'manager', 'dealer', 'user'], label: 'Dashboard' },
  { path: '/orders',          roles: ['admin', 'manager', 'dealer'],         label: 'Orders' },
  { path: '/products',        roles: ['admin', 'manager'],                   label: 'Products' },
  { path: '/users',           roles: ['admin'],                              label: 'Users' },
  { path: '/settings',        roles: ['admin'],                              label: 'Settings' },
  { path: '/reports',         roles: ['admin', 'manager'],                   label: 'Reports' },
  { path: '/dealer-panel',    roles: ['dealer'],                             label: 'Dealer Panel' },
];

export function canAccessRoute(path: string, role: UserRole): boolean {
  const permission = ROUTE_PERMISSIONS.find((p) => path.startsWith(p.path));
  if (!permission) return true; // no restriction defined = public
  return permission.roles.includes(role);
}

export function getAccessibleRoutes(role: UserRole): RoutePermission[] {
  return ROUTE_PERMISSIONS.filter((p) => p.roles.includes(role));
}
```

## useHasPermission Hook

Simple boolean check. Used in components to conditionally render or disable elements.

```typescript
// src/hooks/use-has-permission.ts
import { useAuthStore } from '@/stores/auth-store';
import type { UserRole } from '@/types/auth';

/**
 * Check if the current user has one of the specified roles.
 *
 * Usage:
 *   const canEdit = useHasPermission('admin', 'manager');
 *   const isAdmin = useHasPermission('admin');
 */
export function useHasPermission(...roles: UserRole[]): boolean {
  const userRole = useAuthStore((s) => s.user?.role);
  if (!userRole) return false;
  return roles.includes(userRole);
}

/**
 * Check a specific granular permission string (optional system).
 *
 * Usage:
 *   const canDeleteOrders = useHasGranularPermission('orders:delete');
 */
export function useHasGranularPermission(permission: string): boolean {
  const permissions = useAuthStore((s) => s.user?.permissions ?? []);
  return permissions.includes(permission);
}
```

## Route-Level Guard

Using React Router loaders. The guard runs before the page renders. If the user lacks the role, redirect to `/unauthorized`.

```typescript
// src/lib/route-guard.ts
import { redirect } from 'react-router-dom';
import { useAuthStore } from '@/stores/auth-store';
import { canAccessRoute } from '@/config/permissions';
import type { UserRole } from '@/types/auth';

/**
 * Route loader that checks role-based access.
 * Use in route definition: loader: requireRole('admin', 'manager')
 */
export function requireRole(...roles: UserRole[]) {
  return () => {
    const user = useAuthStore.getState().user;

    if (!user) {
      return redirect('/login');
    }

    if (!roles.includes(user.role)) {
      return redirect('/unauthorized');
    }

    return null; // access granted
  };
}

/**
 * Route loader that checks access using the permission matrix.
 * Automatically looks up allowed roles for the current path.
 */
export function requireAccess() {
  return ({ request }: { request: Request }) => {
    const user = useAuthStore.getState().user;
    const url = new URL(request.url);

    if (!user) {
      return redirect('/login');
    }

    if (!canAccessRoute(url.pathname, user.role)) {
      return redirect('/unauthorized');
    }

    return null;
  };
}
```

### Route Definition

```typescript
// src/router.tsx
import { createBrowserRouter } from 'react-router-dom';
import { requireRole, requireAccess } from '@/lib/route-guard';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppLayout />,
    children: [
      {
        path: 'dashboard',
        lazy: () => import('@/features/dashboard/pages/dashboard-page'),
        loader: requireAccess(),
      },
      {
        path: 'users',
        lazy: () => import('@/features/users/pages/users-page'),
        loader: requireRole('admin'),
      },
      {
        path: 'orders',
        lazy: () => import('@/features/orders/pages/orders-page'),
        loader: requireRole('admin', 'manager', 'dealer'),
      },
      {
        path: 'unauthorized',
        lazy: () => import('@/pages/unauthorized-page'),
      },
    ],
  },
]);
```

## Component-Level Guard: CanAccess

A wrapper component that conditionally renders children based on role. Two behaviors: **hide** (default) or **disable** (show but grayed out with tooltip).

```typescript
// src/components/can-access.tsx
import type { ReactNode } from 'react';
import { useHasPermission } from '@/hooks/use-has-permission';
import type { UserRole } from '@/types/auth';

interface CanAccessProps {
  roles: UserRole[];
  children: ReactNode;
  fallback?: ReactNode;       // shown when access denied (default: nothing)
  mode?: 'hide' | 'disable';  // hide removes from DOM, disable grays out
  disabledTooltip?: string;   // tooltip when disabled
}

export function CanAccess({
  roles,
  children,
  fallback = null,
  mode = 'hide',
  disabledTooltip = 'You do not have permission for this action',
}: CanAccessProps) {
  const hasAccess = useHasPermission(...roles);

  if (hasAccess) {
    return <>{children}</>;
  }

  if (mode === 'disable') {
    return (
      <div
        className="pointer-events-none opacity-50 cursor-not-allowed"
        title={disabledTooltip}
        aria-disabled="true"
      >
        {children}
      </div>
    );
  }

  // mode === 'hide'
  return <>{fallback}</>;
}
```

### Usage Examples

```typescript
// Hide the delete button entirely for non-admins
<CanAccess roles={['admin']}>
  <Button variant="destructive" onClick={handleDelete}>
    Delete User
  </Button>
</CanAccess>

// Show the button but disabled with tooltip for non-admins
<CanAccess roles={['admin']} mode="disable" disabledTooltip="Only admins can export">
  <Button onClick={handleExport}>
    Export Data
  </Button>
</CanAccess>

// Show a different component for unauthorized users
<CanAccess
  roles={['admin', 'manager']}
  fallback={<p className="text-sm text-muted-foreground">Contact your admin for access.</p>}
>
  <SettingsPanel />
</CanAccess>
```

## Sidebar: Filter Menu Items by Role

The sidebar renders only routes the user can access, using the permission matrix:

```typescript
// src/components/sidebar.tsx
import { useAuthStore } from '@/stores/auth-store';
import { getAccessibleRoutes } from '@/config/permissions';
import { NavLink } from 'react-router-dom';

export function Sidebar() {
  const role = useAuthStore((s) => s.user?.role);

  if (!role) return null;

  const routes = getAccessibleRoutes(role);

  return (
    <nav className="flex flex-col gap-1 p-4">
      {routes.map((route) => (
        <NavLink
          key={route.path}
          to={route.path}
          className={({ isActive }) =>
            `flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors
             ${isActive ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'}`
          }
        >
          {route.label}
        </NavLink>
      ))}
    </nav>
  );
}
```

## Unauthorized Page

```typescript
// src/pages/unauthorized-page.tsx
import { useNavigate } from 'react-router-dom';
import { ShieldX } from 'lucide-react';

export function Component() {
  const navigate = useNavigate();

  return (
    <div className="flex flex-col items-center justify-center gap-4 py-20">
      <ShieldX className="h-16 w-16 text-destructive" />
      <h1 className="text-2xl font-semibold">Access Denied</h1>
      <p className="text-muted-foreground">
        You do not have permission to view this page.
      </p>
      <button
        onClick={() => navigate('/dashboard')}
        className="mt-4 rounded-md bg-primary px-4 py-2 text-sm text-primary-foreground"
      >
        Go to Dashboard
      </button>
    </div>
  );
}
```

## Rules

1. **Never rely on UI guards for security.** The API must independently verify permissions on every request. UI guards are UX convenience only.
2. **Hide, don't show-and-block.** If a user should never see a feature, hide it entirely. Only use "disable" mode when the user should know the feature exists but needs a higher role.
3. **Single source of truth for permissions.** The `ROUTE_PERMISSIONS` config is the one place where route-role mappings are defined. Sidebar, route guards, and component guards all read from it.
4. **Roles come from JWT, not from a local config.** The auth store decodes the JWT and extracts the role. If roles change, the user re-authenticates.
5. **No `any` in role types.** Use the `UserRole` union type everywhere. Adding a new role means updating the type -- TypeScript catches all missing cases.
