---
knowledge-base-summary: "Tenant-based theming for SaaS. CSS custom properties (variables) for runtime color changes. Logo/brand swap per tenant. Tailwind + CSS variables integration. Theme loaded from API at startup."
---
# White Label: Tenant Theming

## Philosophy

Multi-tenant SaaS apps serve different brands from the same codebase. Each tenant gets their own colors, logo, and branding -- but the components, layout, and functionality are identical. Theme customization happens at the CSS variable level, not at the component level. Components never contain tenant-specific logic.

The theme flows: **API (tenant config) --> CSS custom properties --> Tailwind utilities --> Components**.

## Architecture

```
1. App startup: GET /api/tenant/theme
2. Apply CSS variables to :root
3. Tailwind utilities reference CSS variables
4. Components use Tailwind classes (unchanged per tenant)
```

The key insight: components use `bg-primary`, `text-primary`, `border-primary` -- Tailwind resolves these to CSS variables at runtime. Changing the variable changes every component instantly.

## Theme Type Definition

```typescript
// src/types/tenant.ts

export interface TenantTheme {
  // Brand colors
  primary: string;          // main brand color (hex)
  primaryForeground: string; // text on primary background
  secondary: string;
  secondaryForeground: string;
  accent: string;
  accentForeground: string;

  // Surfaces
  background: string;
  foreground: string;
  card: string;
  cardForeground: string;
  muted: string;
  mutedForeground: string;
  border: string;

  // Semantic
  destructive: string;
  destructiveForeground: string;
  success: string;
  warning: string;

  // Typography
  fontFamily?: string;       // custom font family
  borderRadius?: string;     // base border radius (e.g., '0.5rem')

  // Branding
  logoUrl: string;
  faviconUrl?: string;
  appName: string;
}

export interface TenantConfig {
  tenantId: string;
  slug: string;              // subdomain or path prefix
  theme: TenantTheme;
}
```

## Default Theme (Fallback)

If the API fails or returns no theme, the app uses this default. The user should never see a broken, unstyled page.

```typescript
// src/config/default-theme.ts
import type { TenantTheme } from '@/types/tenant';

export const DEFAULT_THEME: TenantTheme = {
  primary: '#2563eb',           // blue-600
  primaryForeground: '#ffffff',
  secondary: '#64748b',         // slate-500
  secondaryForeground: '#ffffff',
  accent: '#f59e0b',            // amber-500
  accentForeground: '#ffffff',

  background: '#ffffff',
  foreground: '#0f172a',        // slate-900
  card: '#ffffff',
  cardForeground: '#0f172a',
  muted: '#f1f5f9',             // slate-100
  mutedForeground: '#64748b',
  border: '#e2e8f0',            // slate-200

  destructive: '#ef4444',
  destructiveForeground: '#ffffff',
  success: '#22c55e',
  warning: '#f59e0b',

  borderRadius: '0.5rem',

  logoUrl: '/logo-default.svg',
  appName: 'MyApp',
};
```

## CSS Variables Integration

Convert hex colors to HSL for Tailwind compatibility (Tailwind's color system uses HSL).

```typescript
// src/lib/theme-utils.ts
import type { TenantTheme } from '@/types/tenant';

/**
 * Convert hex color to HSL string for CSS custom properties.
 * '#2563eb' → '217 91% 60%'
 */
function hexToHsl(hex: string): string {
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const l = (max + min) / 2;

  if (max === min) {
    return `0 0% ${Math.round(l * 100)}%`;
  }

  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);

  let h = 0;
  if (max === r) h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
  else if (max === g) h = ((b - r) / d + 2) / 6;
  else h = ((r - g) / d + 4) / 6;

  return `${Math.round(h * 360)} ${Math.round(s * 100)}% ${Math.round(l * 100)}%`;
}

/**
 * Apply a tenant theme to the document by setting CSS custom properties.
 */
export function applyTheme(theme: TenantTheme) {
  const root = document.documentElement;

  root.style.setProperty('--primary', hexToHsl(theme.primary));
  root.style.setProperty('--primary-foreground', hexToHsl(theme.primaryForeground));
  root.style.setProperty('--secondary', hexToHsl(theme.secondary));
  root.style.setProperty('--secondary-foreground', hexToHsl(theme.secondaryForeground));
  root.style.setProperty('--accent', hexToHsl(theme.accent));
  root.style.setProperty('--accent-foreground', hexToHsl(theme.accentForeground));

  root.style.setProperty('--background', hexToHsl(theme.background));
  root.style.setProperty('--foreground', hexToHsl(theme.foreground));
  root.style.setProperty('--card', hexToHsl(theme.card));
  root.style.setProperty('--card-foreground', hexToHsl(theme.cardForeground));
  root.style.setProperty('--muted', hexToHsl(theme.muted));
  root.style.setProperty('--muted-foreground', hexToHsl(theme.mutedForeground));
  root.style.setProperty('--border', hexToHsl(theme.border));

  root.style.setProperty('--destructive', hexToHsl(theme.destructive));
  root.style.setProperty('--destructive-foreground', hexToHsl(theme.destructiveForeground));

  if (theme.borderRadius) {
    root.style.setProperty('--radius', theme.borderRadius);
  }

  if (theme.fontFamily) {
    root.style.setProperty('--font-family', theme.fontFamily);
  }

  // Update favicon
  if (theme.faviconUrl) {
    const link = document.querySelector<HTMLLinkElement>("link[rel='icon']");
    if (link) link.href = theme.faviconUrl;
  }

  // Update document title
  document.title = theme.appName;
}
```

## Tailwind Configuration

Tailwind references CSS custom properties so that utility classes like `bg-primary`, `text-foreground` resolve to tenant-specific colors at runtime.

```typescript
// tailwind.config.ts
import type { Config } from 'tailwindcss';

const config: Config = {
  darkMode: 'class',
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        sans: ['var(--font-family)', 'Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
};

export default config;
```

## Base CSS (Default Variables)

These are the defaults that apply before any tenant theme is loaded from the API.

```css
/* src/styles/globals.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 0 0% 100%;
    --foreground: 222 47% 11%;

    --card: 0 0% 100%;
    --card-foreground: 222 47% 11%;

    --primary: 221 83% 53%;
    --primary-foreground: 210 40% 98%;

    --secondary: 215 16% 47%;
    --secondary-foreground: 210 40% 98%;

    --accent: 38 92% 50%;
    --accent-foreground: 210 40% 98%;

    --muted: 210 40% 96%;
    --muted-foreground: 215 16% 47%;

    --border: 214 32% 91%;

    --destructive: 0 84% 60%;
    --destructive-foreground: 210 40% 98%;

    --radius: 0.5rem;
    --font-family: 'Inter', system-ui, sans-serif;
  }
}
```

## Theme Provider (Context)

Manages tenant config (logo, app name, etc.) and provides it to components via context. The CSS variables are applied globally via `applyTheme()`, but non-CSS config (logo URL, app name) is provided via context.

```typescript
// src/providers/theme-provider.tsx
import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import type { TenantConfig, TenantTheme } from '@/types/tenant';
import { DEFAULT_THEME } from '@/config/default-theme';
import { applyTheme } from '@/lib/theme-utils';
import { api } from '@/lib/api-client';

interface ThemeContextValue {
  theme: TenantTheme;
  tenantId: string | null;
  appName: string;
  logoUrl: string;
  loading: boolean;
}

const ThemeContext = createContext<ThemeContextValue>({
  theme: DEFAULT_THEME,
  tenantId: null,
  appName: DEFAULT_THEME.appName,
  logoUrl: DEFAULT_THEME.logoUrl,
  loading: true,
});

export function useTheme() {
  return useContext(ThemeContext);
}

const STORAGE_KEY = 'tenant-theme';

interface ThemeProviderProps {
  children: ReactNode;
  tenantSlug?: string; // from subdomain or URL
}

export function ThemeProvider({ children, tenantSlug }: ThemeProviderProps) {
  const [theme, setTheme] = useState<TenantTheme>(() => {
    // Instant paint: load cached theme from localStorage
    const cached = localStorage.getItem(STORAGE_KEY);
    if (cached) {
      try {
        const parsed = JSON.parse(cached) as TenantTheme;
        applyTheme(parsed);
        return parsed;
      } catch {
        // invalid cache, ignore
      }
    }
    applyTheme(DEFAULT_THEME);
    return DEFAULT_THEME;
  });

  const [tenantId, setTenantId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function loadTheme() {
      try {
        const params = tenantSlug ? { slug: tenantSlug } : {};
        const { data } = await api.get<TenantConfig>('/api/tenant/theme', { params });

        if (cancelled) return;

        setTenantId(data.tenantId);
        setTheme(data.theme);
        applyTheme(data.theme);

        // Cache for instant paint on next load
        localStorage.setItem(STORAGE_KEY, JSON.stringify(data.theme));
      } catch {
        // API failed -- keep using cached or default theme
        console.warn('Failed to load tenant theme, using fallback');
        applyTheme(DEFAULT_THEME);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    loadTheme();

    return () => {
      cancelled = true;
    };
  }, [tenantSlug]);

  return (
    <ThemeContext.Provider
      value={{
        theme,
        tenantId,
        appName: theme.appName,
        logoUrl: theme.logoUrl,
        loading,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}
```

## App Entry Point

```typescript
// src/main.tsx
import { ThemeProvider } from '@/providers/theme-provider';

// Extract tenant slug from subdomain
function getTenantSlug(): string | undefined {
  const hostname = window.location.hostname;
  const parts = hostname.split('.');
  // e.g., acme.myapp.com → 'acme'
  if (parts.length >= 3) {
    return parts[0];
  }
  return undefined;
}

function App() {
  const tenantSlug = getTenantSlug();

  return (
    <ThemeProvider tenantSlug={tenantSlug}>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
```

## Dynamic Logo Component

```typescript
// src/components/brand-logo.tsx
import { useTheme } from '@/providers/theme-provider';

interface BrandLogoProps {
  className?: string;
}

export function BrandLogo({ className = 'h-8' }: BrandLogoProps) {
  const { logoUrl, appName } = useTheme();

  return (
    <img
      src={logoUrl}
      alt={appName}
      className={className}
      onError={(e) => {
        // Fallback to text if logo fails to load
        const target = e.currentTarget;
        target.style.display = 'none';
        target.parentElement?.insertAdjacentHTML(
          'beforeend',
          `<span class="text-lg font-bold text-primary">${appName}</span>`
        );
      }}
    />
  );
}
```

### Usage in Sidebar/Header

```typescript
// src/components/sidebar.tsx
import { BrandLogo } from '@/components/brand-logo';

export function Sidebar() {
  return (
    <aside className="flex h-screen w-64 flex-col border-r bg-card">
      <div className="flex items-center gap-2 border-b px-4 py-5">
        <BrandLogo className="h-8 w-auto" />
      </div>
      <nav className="flex-1 p-4">
        {/* navigation items */}
      </nav>
    </aside>
  );
}
```

## Login Page with Tenant Branding

```typescript
// src/pages/login-page.tsx
import { BrandLogo } from '@/components/brand-logo';
import { useTheme } from '@/providers/theme-provider';

export function LoginPage() {
  const { appName } = useTheme();

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm space-y-6 rounded-xl border bg-card p-8 shadow-sm">
        <div className="flex flex-col items-center gap-2">
          <BrandLogo className="h-12 w-auto" />
          <p className="text-sm text-muted-foreground">Sign in to {appName}</p>
        </div>

        <form className="space-y-4">
          {/* email, password fields */}
          <button
            type="submit"
            className="w-full rounded-md bg-primary py-2.5 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            Sign In
          </button>
        </form>
      </div>
    </div>
  );
}
```

## Rules

1. **Components never know about tenants.** A `<Button>` uses `bg-primary`. It never checks `tenantId`. Theme resolution happens at the CSS variable level, completely invisible to components.
2. **CSS variables are the contract.** The API returns hex colors. `applyTheme()` converts to HSL and sets CSS variables. Tailwind references CSS variables. Components use Tailwind classes.
3. **LocalStorage for instant paint.** Cache the theme in localStorage. On next page load, apply the cached theme immediately (before API call). This prevents a flash of default colors.
4. **Default theme fallback.** If the API fails, the app works with the default theme. Never show a broken or unstyled page.
5. **Logo has an error fallback.** If the logo image fails to load, show the app name as text.
6. **Tenant slug from subdomain.** `acme.myapp.com` maps to tenant "acme". The slug is passed to the API to get the correct theme.
7. **Never hardcode brand colors.** Every color goes through a CSS variable. If you see `bg-blue-600` in a white-label app, it is wrong. It should be `bg-primary`.
