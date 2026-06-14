---
knowledge-base-summary: "Login/register pages, token storage (httpOnly cookie or localStorage). Auto-login on page load. Axios interceptor for token refresh. Protected routes. Logout (clear tokens, redirect). Multi-profile selection for SaaS."
---
# Auth Flow

Login, register, token management, protected routes, logout. Complete authentication lifecycle.

## Architecture Overview

```
Login Page                                       Protected App
    |                                                 |
    |  POST /auth/login                               |
    |  { email, password }                            |
    |        |                                        |
    |        v                                        |
    |  Response:                                      |
    |  { accessToken, user }                          |
    |  + Set-Cookie: refreshToken=... (httpOnly)      |
    |        |                                        |
    |        v                                        |
    |  Store: accessToken in memory (Zustand)         |
    |  Store: user in Zustand (persisted)             |
    |        |                                        |
    |        v                                        |
    |  Navigate to /dashboard  ------>  Axios sends   |
    |                                   Bearer token  |
    |                                   on every req  |
    |                                        |        |
    |                                   401? Refresh   |
    |                                   token flow    |
    |                                        |        |
    |                                   Refresh OK?   |
    |                                   Retry request |
    |                                        |        |
    |                                   Refresh fail? |
    |  <----  Redirect to /login  <----  Clear auth   |
```

## Auth Store (Zustand)

```tsx
// src/stores/auth.store.ts

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: 'admin' | 'member' | 'viewer';
  avatarUrl: string | null;
}

interface Profile {
  id: string;
  name: string;
  organizationId: string;
  organizationName: string;
  role: string;
}

interface AuthState {
  // State
  accessToken: string | null;
  user: AuthUser | null;
  profiles: Profile[];
  activeProfileId: string | null;

  // Actions
  setAuth: (token: string, user: AuthUser) => void;
  setAccessToken: (token: string) => void;
  setProfiles: (profiles: Profile[]) => void;
  setActiveProfile: (profileId: string) => void;
  clearAuth: () => void;

  // Computed
  isAuthenticated: () => boolean;
  activeProfile: () => Profile | null;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      user: null,
      profiles: [],
      activeProfileId: null,

      setAuth: (accessToken, user) =>
        set({ accessToken, user }),

      setAccessToken: (accessToken) =>
        set({ accessToken }),

      setProfiles: (profiles) =>
        set({ profiles }),

      setActiveProfile: (activeProfileId) =>
        set({ activeProfileId }),

      clearAuth: () =>
        set({
          accessToken: null,
          user: null,
          profiles: [],
          activeProfileId: null,
        }),

      isAuthenticated: () => !!get().accessToken,

      activeProfile: () => {
        const { profiles, activeProfileId } = get();
        return profiles.find((p) => p.id === activeProfileId) ?? null;
      },
    }),
    {
      name: 'auth-storage',
      // Persist user info but NOT the access token
      // Access token lives only in memory for security
      partialize: (state) => ({
        user: state.user,
        profiles: state.profiles,
        activeProfileId: state.activeProfileId,
      }),
    }
  )
);
```

## Login Page

```tsx
// src/features/auth/pages/LoginPage.tsx

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { z } from 'zod';
import { useLogin } from '../api/auth.api';
import { useAuthStore } from '@/stores/auth.store';
import { parseApiError } from '@/lib/api-error';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { toast } from 'sonner';

const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Invalid email'),
  password: z.string().min(1, 'Password is required'),
});

type LoginForm = z.infer<typeof loginSchema>;

export function LoginPage() {
  const { t } = useTranslation('auth');
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const setAuth = useAuthStore((s) => s.setAuth);
  const { mutate: login, isPending } = useLogin();

  const form = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  });

  function onSubmit(data: LoginForm) {
    login(data, {
      onSuccess: (response) => {
        // Store auth data
        setAuth(response.accessToken, response.user);

        // Redirect to the page user tried to access, or dashboard
        const returnTo = searchParams.get('returnTo') || '/dashboard';
        navigate(returnTo, { replace: true });
      },
      onError: (error) => {
        const apiError = parseApiError(error);

        if (apiError.statusCode === 401) {
          form.setError('password', {
            message: t('invalidCredentials'),
          });
          return;
        }

        if (apiError.statusCode === 423) {
          toast.error(t('accountLocked'));
          return;
        }

        toast.error(apiError.message);
      },
    });
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t('login.title')}
        </h1>
        <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {t('login.subtitle')}
        </p>
      </div>

      <form onSubmit={form.handleSubmit(onSubmit)} noValidate className="space-y-4">
        <Input
          label={t('fields.email')}
          type="email"
          autoComplete="email"
          error={form.formState.errors.email?.message}
          {...form.register('email')}
          autoFocus
        />

        <Input
          label={t('fields.password')}
          type="password"
          autoComplete="current-password"
          error={form.formState.errors.password?.message}
          {...form.register('password')}
        />

        <Button type="submit" isLoading={isPending} className="w-full">
          {t('login.submit')}
        </Button>
      </form>

      <p className="text-center text-sm text-gray-500">
        {t('login.noAccount')}{' '}
        <a href="/register" className="text-primary-600 hover:underline">
          {t('login.signUp')}
        </a>
      </p>
    </div>
  );
}
```

## Register Page

```tsx
// src/features/auth/pages/RegisterPage.tsx

import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { z } from 'zod';
import { useRegister } from '../api/auth.api';
import { useAuthStore } from '@/stores/auth.store';
import { parseApiError } from '@/lib/api-error';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { toast } from 'sonner';

const registerSchema = z
  .object({
    name: z.string().min(2, 'Name must be at least 2 characters'),
    email: z.string().email('Invalid email address'),
    password: z
      .string()
      .min(8, 'Password must be at least 8 characters')
      .regex(/[A-Z]/, 'Must contain an uppercase letter')
      .regex(/[0-9]/, 'Must contain a number'),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword'],
  });

type RegisterForm = z.infer<typeof registerSchema>;

export function RegisterPage() {
  const { t } = useTranslation('auth');
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);
  const { mutate: registerUser, isPending } = useRegister();

  const form = useForm<RegisterForm>({
    resolver: zodResolver(registerSchema),
    defaultValues: { name: '', email: '', password: '', confirmPassword: '' },
  });

  function onSubmit(data: RegisterForm) {
    registerUser(
      { name: data.name, email: data.email, password: data.password },
      {
        onSuccess: (response) => {
          setAuth(response.accessToken, response.user);
          navigate('/dashboard', { replace: true });
        },
        onError: (error) => {
          const apiError = parseApiError(error);

          if (apiError.statusCode === 409) {
            form.setError('email', {
              message: t('emailAlreadyExists'),
            });
            return;
          }

          if (apiError.errors) {
            Object.entries(apiError.errors).forEach(([field, messages]) => {
              form.setError(field as keyof RegisterForm, {
                message: messages[0],
              });
            });
            return;
          }

          toast.error(apiError.message);
        },
      }
    );
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
          {t('register.title')}
        </h1>
      </div>

      <form onSubmit={form.handleSubmit(onSubmit)} noValidate className="space-y-4">
        <Input
          label={t('fields.name')}
          error={form.formState.errors.name?.message}
          {...form.register('name')}
          autoFocus
        />
        <Input
          label={t('fields.email')}
          type="email"
          autoComplete="email"
          error={form.formState.errors.email?.message}
          {...form.register('email')}
        />
        <Input
          label={t('fields.password')}
          type="password"
          autoComplete="new-password"
          error={form.formState.errors.password?.message}
          {...form.register('password')}
        />
        <Input
          label={t('fields.confirmPassword')}
          type="password"
          autoComplete="new-password"
          error={form.formState.errors.confirmPassword?.message}
          {...form.register('confirmPassword')}
        />
        <Button type="submit" isLoading={isPending} className="w-full">
          {t('register.submit')}
        </Button>
      </form>
    </div>
  );
}
```

## Auth API Hooks

```tsx
// src/features/auth/api/auth.api.ts

import { useMutation } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { axiosInstance } from '@/lib/api-client';

// ─── Types ───

interface LoginRequest {
  email: string;
  password: string;
}

interface RegisterRequest {
  name: string;
  email: string;
  password: string;
}

interface AuthResponse {
  accessToken: string;
  user: {
    id: string;
    email: string;
    name: string;
    role: 'admin' | 'member' | 'viewer';
    avatarUrl: string | null;
  };
}

interface RefreshResponse {
  accessToken: string;
}

// ─── Mutations ───

export function useLogin() {
  return useMutation({
    mutationFn: (data: LoginRequest) =>
      apiClient.post<AuthResponse>('/auth/login', data),
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: (data: RegisterRequest) =>
      apiClient.post<AuthResponse>('/auth/register', data),
  });
}

export function useLogout() {
  return useMutation({
    mutationFn: () =>
      apiClient.post('/auth/logout', {}, ),
  });
}

// ─── Standalone function (not a hook — used in interceptor) ───

export async function refreshAccessToken(): Promise<string> {
  const response = await axiosInstance.post<RefreshResponse>(
    '/auth/refresh',
    {},
    { withCredentials: true } // Send httpOnly refresh token cookie
  );
  return response.data.accessToken;
}
```

## Token Storage Strategy

```
Access Token:
  - Stored in Zustand (in-memory)
  - NOT persisted to localStorage (XSS protection)
  - Short-lived (15 min)
  - Sent as Bearer token in Authorization header

Refresh Token:
  - Stored as httpOnly cookie by the API
  - NOT accessible to JavaScript (XSS protection)
  - Long-lived (30 days)
  - Sent automatically with withCredentials: true

Fallback (if httpOnly cookies not possible):
  - Refresh token in localStorage (less secure but functional)
  - Access token still in memory
```

## Auto-Login on Page Load

When the user refreshes the page, the access token (in memory) is lost. Auto-login attempts to get a new one using the refresh token:

```tsx
// src/features/auth/components/AuthProvider.tsx

import { useEffect, useState } from 'react';
import { useAuthStore } from '@/stores/auth.store';
import { refreshAccessToken } from '../api/auth.api';
import { LoadingState } from '@/components/ui/LoadingState';

interface AuthProviderProps {
  children: React.ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [isInitialized, setIsInitialized] = useState(false);
  const user = useAuthStore((s) => s.user);
  const setAccessToken = useAuthStore((s) => s.setAccessToken);
  const clearAuth = useAuthStore((s) => s.clearAuth);

  useEffect(() => {
    async function initAuth() {
      // If we have user info persisted but no access token (page refresh),
      // try to get a new access token using the refresh token
      if (user && !useAuthStore.getState().accessToken) {
        try {
          const newToken = await refreshAccessToken();
          setAccessToken(newToken);
        } catch {
          // Refresh token expired or invalid — clear everything
          clearAuth();
        }
      }
      setIsInitialized(true);
    }

    initAuth();
  }, []); // Run once on mount

  // Show loading until auth state is resolved
  if (!isInitialized) {
    return <LoadingState message="Loading..." />;
  }

  return <>{children}</>;
}
```

```tsx
// src/main.tsx — wrap the app with AuthProvider

import { AuthProvider } from '@/features/auth/components/AuthProvider';

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>
);
```

## Axios Interceptor (401 -> Refresh -> Retry)

The interceptor handles:
1. Injecting the Bearer token on every request
2. Catching 401 errors
3. Attempting token refresh
4. Queuing concurrent requests during refresh
5. Retrying failed requests with the new token
6. Clearing auth and redirecting if refresh fails

Full implementation is in `api-integration.md` under the Response Interceptor section.

## Protected Route with React Router Loader

```tsx
// src/lib/auth-loader.ts

import { redirect } from 'react-router-dom';
import { useAuthStore } from '@/stores/auth.store';

export function authLoader() {
  const { accessToken } = useAuthStore.getState();

  if (!accessToken) {
    const returnTo = window.location.pathname + window.location.search;
    return redirect(`/login?returnTo=${encodeURIComponent(returnTo)}`);
  }

  return null;
}

export function guestLoader() {
  const { accessToken } = useAuthStore.getState();

  if (accessToken) {
    return redirect('/dashboard');
  }

  return null;
}

// Role-based loader
export function adminLoader() {
  const { accessToken, user } = useAuthStore.getState();

  if (!accessToken) {
    return redirect('/login');
  }

  if (user?.role !== 'admin') {
    return redirect('/dashboard'); // Or show 403 page
  }

  return null;
}
```

## Logout

```tsx
// src/features/auth/hooks/useLogoutAction.ts

import { useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/stores/auth.store';
import { useLogout } from '../api/auth.api';

export function useLogoutAction() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const { mutate: logoutApi } = useLogout();

  function logout() {
    // 1. Tell API to invalidate the refresh token
    logoutApi(undefined, {
      onSettled: () => {
        // 2. Clear local auth state (Zustand)
        clearAuth();

        // 3. Clear ALL React Query cache
        queryClient.clear();

        // 4. Redirect to login
        navigate('/login', { replace: true });
      },
    });
  }

  return logout;
}
```

### Usage in Header

```tsx
function Header() {
  const { t } = useTranslation('common');
  const user = useAuthStore((s) => s.user);
  const logout = useLogoutAction();

  return (
    <header className="flex items-center justify-between border-b px-6 py-3">
      <div>{/* breadcrumb, etc */}</div>

      <div className="flex items-center gap-3">
        <span className="text-sm text-gray-700 dark:text-gray-300">
          {user?.name}
        </span>
        <Button variant="ghost" size="sm" onClick={logout}>
          {t('logout')}
        </Button>
      </div>
    </header>
  );
}
```

## Multi-Profile Selection (SaaS)

For SaaS apps where a user belongs to multiple organizations:

```tsx
// src/features/auth/pages/ProfileSelectPage.tsx

import { useAuthStore } from '@/stores/auth.store';
import { useNavigate } from 'react-router-dom';
import { Card } from '@/components/ui/Card';

export function ProfileSelectPage() {
  const profiles = useAuthStore((s) => s.profiles);
  const setActiveProfile = useAuthStore((s) => s.setActiveProfile);
  const navigate = useNavigate();

  function handleSelect(profileId: string) {
    setActiveProfile(profileId);
    navigate('/dashboard', { replace: true });
  }

  return (
    <div className="mx-auto max-w-md space-y-4">
      <h1 className="text-center text-2xl font-bold">Select Workspace</h1>

      {profiles.map((profile) => (
        <Card
          key={profile.id}
          className="cursor-pointer hover:border-primary-500 transition-colors"
          onClick={() => handleSelect(profile.id)}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleSelect(profile.id);
          }}
        >
          <div className="p-4">
            <p className="font-medium">{profile.organizationName}</p>
            <p className="text-sm text-gray-500">{profile.role}</p>
          </div>
        </Card>
      ))}
    </div>
  );
}
```

### Updated Login Flow with Profiles

```tsx
// In LoginPage onSuccess:
onSuccess: (response) => {
  setAuth(response.accessToken, response.user);

  if (response.profiles && response.profiles.length > 1) {
    // Multiple profiles — user must choose
    setProfiles(response.profiles);
    navigate('/select-profile', { replace: true });
  } else if (response.profiles?.length === 1) {
    // Single profile — auto-select
    setProfiles(response.profiles);
    setActiveProfile(response.profiles[0].id);
    navigate('/dashboard', { replace: true });
  } else {
    navigate('/dashboard', { replace: true });
  }
},
```

## Rules

1. **Access token in memory only** — never localStorage (XSS risk)
2. **Refresh token as httpOnly cookie** — JavaScript cannot access it
3. **AuthProvider wraps the app** — handles auto-login on page load
4. **authLoader on protected routes** — redirect to /login if not authenticated
5. **guestLoader on login/register** — redirect to /dashboard if already logged in
6. **Logout clears everything** — auth store, React Query cache, redirect
7. **returnTo param** — after login, redirect to the page user originally wanted
8. **401 interceptor queues requests** — concurrent requests wait for token refresh
9. **Multi-profile after login** — show profile selection if user has multiple orgs
