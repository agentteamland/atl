---
knowledge-base-summary: "Sidebar + header + content layout. Collapsible sidebar. Breadcrumb navigation. Responsive: sidebar → drawer on mobile. Sticky header. Content max-width constraint. Grid-based dashboard layout."
---
# Layout Patterns

## Philosophy

Admin panels and dashboards follow a consistent layout: collapsible sidebar on the left, sticky header on top, scrollable content area. Mobile replaces the sidebar with a hamburger menu that opens a drawer. The layout is built once and shared across all authenticated routes via React Router's layout routes. Content has a max-width to remain readable on ultra-wide screens.

## Dependencies

```json
{
  "react-router-dom": "^6.x",
  "zustand": "^4.x",
  "clsx": "^2.x",
  "lucide-react": "latest"
}
```

## Layout State with Zustand

```typescript
// stores/layout-store.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface LayoutState {
  sidebarCollapsed: boolean;
  mobileMenuOpen: boolean;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setMobileMenuOpen: (open: boolean) => void;
}

export const useLayoutStore = create<LayoutState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      mobileMenuOpen: false,
      toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
      setMobileMenuOpen: (open) => set({ mobileMenuOpen: open }),
    }),
    {
      name: 'layout-preferences',
      partialize: (state) => ({ sidebarCollapsed: state.sidebarCollapsed }),
    }
  )
);
```

## Complete Admin Layout

```typescript
// components/layout/admin-layout.tsx
import { Outlet } from 'react-router-dom';
import { useLayoutStore } from '@/stores/layout-store';
import { Sidebar } from './sidebar';
import { Header } from './header';
import { MobileDrawer } from './mobile-drawer';
import { ErrorBoundary } from '@/components/ui/error-boundary';
import clsx from 'clsx';

export function AdminLayout() {
  const { sidebarCollapsed } = useLayoutStore();

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Desktop Sidebar */}
      <div className="hidden lg:block">
        <Sidebar />
      </div>

      {/* Mobile Drawer */}
      <MobileDrawer />

      {/* Main content area */}
      <div
        className={clsx(
          'flex flex-1 flex-col overflow-hidden transition-all duration-300',
          sidebarCollapsed ? 'lg:ml-16' : 'lg:ml-64'
        )}
      >
        <Header />
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
            <ErrorBoundary>
              <Outlet />
            </ErrorBoundary>
          </div>
        </main>
      </div>
    </div>
  );
}
```

## Sidebar Component

```typescript
// components/layout/sidebar.tsx
import { NavLink, useLocation } from 'react-router-dom';
import { useLayoutStore } from '@/stores/layout-store';
import {
  LayoutDashboard,
  Users,
  Settings,
  BarChart3,
  FileText,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import clsx from 'clsx';

interface NavItem {
  label: string;
  path: string;
  icon: React.ElementType;
  children?: { label: string; path: string }[];
}

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
  { label: 'Users', path: '/users', icon: Users },
  { label: 'Reports', path: '/reports', icon: BarChart3 },
  { label: 'Documents', path: '/documents', icon: FileText },
  { label: 'Settings', path: '/settings', icon: Settings },
];

export function Sidebar() {
  const { sidebarCollapsed, toggleSidebar } = useLayoutStore();

  return (
    <aside
      className={clsx(
        'fixed left-0 top-0 z-30 flex h-screen flex-col border-r border-gray-200 bg-white transition-all duration-300',
        sidebarCollapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Logo */}
      <div className="flex h-16 items-center border-b border-gray-200 px-4">
        {sidebarCollapsed ? (
          <div className="mx-auto h-8 w-8 rounded-lg bg-blue-600 flex items-center justify-center text-white text-sm font-bold">
            W
          </div>
        ) : (
          <span className="text-lg font-bold text-gray-900">ExampleApp</span>
        )}
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-1">
        {navItems.map((item) => (
          <SidebarItem
            key={item.path}
            item={item}
            collapsed={sidebarCollapsed}
          />
        ))}
      </nav>

      {/* Collapse toggle */}
      <div className="border-t border-gray-200 p-3">
        <button
          onClick={toggleSidebar}
          className="flex w-full items-center justify-center rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
          aria-label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {sidebarCollapsed ? (
            <ChevronRight className="h-5 w-5" />
          ) : (
            <ChevronLeft className="h-5 w-5" />
          )}
        </button>
      </div>
    </aside>
  );
}

function SidebarItem({
  item,
  collapsed,
}: {
  item: NavItem;
  collapsed: boolean;
}) {
  return (
    <NavLink
      to={item.path}
      className={({ isActive }) =>
        clsx(
          'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors',
          isActive
            ? 'bg-blue-50 text-blue-700 font-medium'
            : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900',
          collapsed && 'justify-center px-2'
        )
      }
    >
      <item.icon className="h-5 w-5 shrink-0" />
      {!collapsed && <span>{item.label}</span>}
    </NavLink>
  );
}
```

## Header Component

```typescript
// components/layout/header.tsx
import { useLocation } from 'react-router-dom';
import { useLayoutStore } from '@/stores/layout-store';
import { Menu, Bell, ChevronRight } from 'lucide-react';
import { Breadcrumbs } from './breadcrumbs';
import { UserMenu } from './user-menu';

export function Header() {
  const { setMobileMenuOpen } = useLayoutStore();

  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-4 border-b border-gray-200 bg-white px-4 sm:px-6 lg:px-8">
      {/* Mobile hamburger */}
      <button
        onClick={() => setMobileMenuOpen(true)}
        className="rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 lg:hidden"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5" />
      </button>

      {/* Breadcrumbs */}
      <div className="flex-1">
        <Breadcrumbs />
      </div>

      {/* Right side: notifications + user menu */}
      <div className="flex items-center gap-3">
        <button className="relative rounded-md p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600">
          <Bell className="h-5 w-5" />
          {/* Notification badge */}
          <span className="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-red-500" />
        </button>
        <UserMenu />
      </div>
    </header>
  );
}
```

## Breadcrumb Navigation

Auto-generated from the current route path.

```typescript
// components/layout/breadcrumbs.tsx
import { Link, useLocation } from 'react-router-dom';
import { ChevronRight, Home } from 'lucide-react';

// Map route segments to human-readable labels
const segmentLabels: Record<string, string> = {
  dashboard: 'Dashboard',
  users: 'Users',
  reports: 'Reports',
  documents: 'Documents',
  settings: 'Settings',
  edit: 'Edit',
  create: 'Create',
};

export function Breadcrumbs() {
  const location = useLocation();
  const segments = location.pathname.split('/').filter(Boolean);

  if (segments.length === 0) return null;

  return (
    <nav className="flex items-center gap-1 text-sm" aria-label="Breadcrumb">
      <Link to="/" className="text-gray-400 hover:text-gray-600">
        <Home className="h-4 w-4" />
      </Link>

      {segments.map((segment, index) => {
        const path = '/' + segments.slice(0, index + 1).join('/');
        const isLast = index === segments.length - 1;
        const label = segmentLabels[segment] ?? segment;

        return (
          <span key={path} className="flex items-center gap-1">
            <ChevronRight className="h-3 w-3 text-gray-300" />
            {isLast ? (
              <span className="font-medium text-gray-900">{label}</span>
            ) : (
              <Link to={path} className="text-gray-500 hover:text-gray-700">
                {label}
              </Link>
            )}
          </span>
        );
      })}
    </nav>
  );
}
```

## Mobile Drawer

Replaces the sidebar on mobile. Slides in from the left.

```typescript
// components/layout/mobile-drawer.tsx
import * as Dialog from '@radix-ui/react-dialog';
import { NavLink } from 'react-router-dom';
import { useLayoutStore } from '@/stores/layout-store';
import { X, LayoutDashboard, Users, Settings, BarChart3, FileText } from 'lucide-react';

const navItems = [
  { label: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
  { label: 'Users', path: '/users', icon: Users },
  { label: 'Reports', path: '/reports', icon: BarChart3 },
  { label: 'Documents', path: '/documents', icon: FileText },
  { label: 'Settings', path: '/settings', icon: Settings },
];

export function MobileDrawer() {
  const { mobileMenuOpen, setMobileMenuOpen } = useLayoutStore();

  return (
    <Dialog.Root open={mobileMenuOpen} onOpenChange={(open) => setMobileMenuOpen(open)}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 lg:hidden data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
        <Dialog.Content className="fixed left-0 top-0 z-50 h-full w-72 bg-white shadow-xl lg:hidden data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left duration-300">
          {/* Header */}
          <div className="flex h-16 items-center justify-between border-b px-4">
            <span className="text-lg font-bold text-gray-900">ExampleApp</span>
            <Dialog.Close asChild>
              <button className="rounded-md p-1 text-gray-400 hover:text-gray-600" aria-label="Close menu">
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          {/* Navigation */}
          <nav className="px-3 py-4 space-y-1">
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                onClick={() => setMobileMenuOpen(false)}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors ${
                    isActive
                      ? 'bg-blue-50 text-blue-700 font-medium'
                      : 'text-gray-600 hover:bg-gray-100'
                  }`
                }
              >
                <item.icon className="h-5 w-5" />
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
```

## User Menu

```typescript
// components/layout/user-menu.tsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { LogOut, Settings, User } from 'lucide-react';
import { useAuthStore } from '@/stores/auth-store';

export function UserMenu() {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 rounded-md p-1 hover:bg-gray-100"
      >
        <div className="h-8 w-8 rounded-full bg-blue-600 flex items-center justify-center text-sm font-medium text-white">
          {user?.name?.charAt(0) ?? 'U'}
        </div>
        <span className="hidden text-sm font-medium text-gray-700 sm:block">
          {user?.name}
        </span>
      </button>

      {open && (
        <>
          <div className="fixed inset-0 z-30" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full z-40 mt-1 w-48 rounded-md border bg-white shadow-lg">
            <div className="border-b px-3 py-2">
              <p className="text-sm font-medium text-gray-900">{user?.name}</p>
              <p className="text-xs text-gray-500">{user?.email}</p>
            </div>
            <button
              onClick={() => { navigate('/settings/profile'); setOpen(false); }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50"
            >
              <Settings className="h-4 w-4" /> Settings
            </button>
            <button
              onClick={() => { logout(); setOpen(false); }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
            >
              <LogOut className="h-4 w-4" /> Sign out
            </button>
          </div>
        </>
      )}
    </div>
  );
}
```

## Dashboard Grid Layout

```typescript
// features/dashboard/dashboard-page.tsx
export function DashboardPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      {/* KPI Cards Row */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <KpiCard title="Total Users" value="12,450" change="+12%" trend="up" />
        <KpiCard title="Active Tasks" value="3,280" change="+5%" trend="up" />
        <KpiCard title="Avg Duration" value="32 min" change="-2%" trend="down" />
        <KpiCard title="Revenue" value="$48,200" change="+18%" trend="up" />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="rounded-lg border bg-white p-4">
          <h2 className="text-sm font-medium text-gray-500 mb-4">Weekly Activity</h2>
          {/* Chart component here */}
        </div>
        <div className="rounded-lg border bg-white p-4">
          <h2 className="text-sm font-medium text-gray-500 mb-4">User Growth</h2>
          {/* Chart component here */}
        </div>
      </div>

      {/* Bottom Row: Table + Side Panel */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="lg:col-span-2 rounded-lg border bg-white p-4">
          <h2 className="text-sm font-medium text-gray-500 mb-4">Recent Activity</h2>
          {/* Table or list component */}
        </div>
        <div className="rounded-lg border bg-white p-4">
          <h2 className="text-sm font-medium text-gray-500 mb-4">Top Users</h2>
          {/* List component */}
        </div>
      </div>
    </div>
  );
}

// KPI Card
interface KpiCardProps {
  title: string;
  value: string;
  change: string;
  trend: 'up' | 'down';
}

function KpiCard({ title, value, change, trend }: KpiCardProps) {
  return (
    <div className="rounded-lg border bg-white p-4">
      <p className="text-sm text-gray-500">{title}</p>
      <p className="mt-1 text-2xl font-bold text-gray-900">{value}</p>
      <p className={`mt-1 text-xs ${trend === 'up' ? 'text-green-600' : 'text-red-600'}`}>
        {change} from last period
      </p>
    </div>
  );
}
```

## Route Registration

```typescript
// routes.tsx
import { createBrowserRouter } from 'react-router-dom';
import { AdminLayout } from '@/components/layout/admin-layout';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AdminLayout />,
    children: [
      { index: true, element: <Navigate to="/dashboard" replace /> },
      { path: 'dashboard', element: <DashboardPage /> },
      { path: 'users', element: <UsersPage /> },
      { path: 'users/:id', element: <UserDetailPage /> },
      { path: 'settings', element: <SettingsPage /> },
    ],
  },
  {
    path: '/login',
    element: <LoginPage />,
  },
]);
```

## Key Rules

1. **Sidebar is fixed position** -- does not scroll with content. Content area has `overflow-y-auto`.
2. **Collapsible sidebar persisted** -- Zustand with `persist` middleware saves collapsed state to localStorage.
3. **Mobile: drawer replaces sidebar** -- `hidden lg:block` for sidebar, `lg:hidden` for mobile drawer. No sidebar on mobile.
4. **Sticky header** -- `sticky top-0 z-20`. Always visible while scrolling content.
5. **Content max-width** -- `max-w-7xl mx-auto`. Prevents content from stretching on ultra-wide monitors.
6. **ErrorBoundary in layout** -- wraps `<Outlet />`. Any route error shows recovery UI without breaking the entire layout.
7. **Breadcrumbs auto-generated** -- derived from `useLocation().pathname`. Map segments to human labels.
8. **Close mobile drawer on navigate** -- call `setMobileMenuOpen(false)` when a NavLink is clicked.
9. **CSS Grid for dashboards** -- `grid-cols-1 sm:grid-cols-2 lg:grid-cols-4` for KPI cards. Responsive by default.
10. **NavLink for active state** -- React Router's `NavLink` provides `isActive` for highlighting the current route.
