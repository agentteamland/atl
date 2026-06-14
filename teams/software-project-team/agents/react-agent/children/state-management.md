---
knowledge-base-summary: "TanStack React Query for server state (API data with automatic caching, refetch, pagination). Zustand for client state (UI state, sidebar, filters). When to use which. No Redux — keep it simple."
---
# State Management

Two tools, two jobs. React Query for server state. Zustand for client state. No Redux.

## Decision Matrix

| Data Source | Tool | Example |
|-------------|------|---------|
| API response | React Query | User list, project details, notifications |
| URL params | React Router | `/users?page=2&sort=name` |
| Form values | React Hook Form | Login form, create user form |
| UI-only state | Zustand | Sidebar open, theme, selected tab |
| Auth state | Zustand (persisted) | Current user, access token |
| Global settings | Zustand (persisted) | Language, timezone, preferences |

**Rule:** If the data comes from the server, use React Query. If it only exists in the browser, use Zustand.

## React Query

### Setup

```tsx
// src/lib/query-client.ts

import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,      // 5 minutes
      gcTime: 10 * 60 * 1000,         // 10 minutes (was cacheTime)
      retry: 1,                        // Retry once on failure
      refetchOnWindowFocus: false,     // Disable auto-refetch on tab focus
    },
    mutations: {
      retry: 0,                        // No retry on mutations
    },
  },
});
```

```tsx
// src/main.tsx

import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { queryClient } from '@/lib/query-client';

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
```

### Query Key Convention

Query keys are structured arrays. Use a factory function per feature:

```tsx
// src/features/users/api/users.api.ts

// Query key factory — keeps keys consistent and type-safe
const userKeys = {
  all:     ['users'] as const,
  lists:   () => [...userKeys.all, 'list'] as const,
  list:    (filters: UserFilters) => [...userKeys.lists(), filters] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail:  (id: string) => [...userKeys.details(), id] as const,
};

// Usage in invalidation:
// Invalidate all user lists (any filter):
queryClient.invalidateQueries({ queryKey: userKeys.lists() });

// Invalidate one specific user:
queryClient.invalidateQueries({ queryKey: userKeys.detail(userId) });

// Nuclear option — invalidate everything user-related:
queryClient.invalidateQueries({ queryKey: userKeys.all });
```

### useQuery — Fetching Data

```tsx
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';

interface UserFilters {
  search?: string;
  role?: string;
  page?: number;
  pageSize?: number;
}

export function useUsers(filters: UserFilters = {}) {
  return useQuery({
    queryKey: userKeys.list(filters),
    queryFn: async () => {
      const response = await apiClient.get<PaginatedResponse<User>>('/users', {
        params: filters,
      });
      return response;
    },
    // Only fetch when we have necessary params (if applicable)
    // enabled: !!filters.organizationId,
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => apiClient.get<User>(`/users/${id}`),
    enabled: !!id, // Don't fetch if id is empty
  });
}
```

### useMutation — Creating/Updating/Deleting

```tsx
import { useMutation, useQueryClient } from '@tanstack/react-query';

export function useCreateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserDto) =>
      apiClient.post<User>('/users', data),

    onSuccess: () => {
      // Invalidate all user lists so they refetch with new data
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },

    onError: (error) => {
      // Error is handled by the component using this hook
      // toast.error() can be called here for global error handling
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserDto }) =>
      apiClient.put<User>(`/users/${id}`, data),

    onSuccess: (updatedUser, { id }) => {
      // Update the cache directly for instant UI feedback
      queryClient.setQueryData(userKeys.detail(id), updatedUser);
      // Also invalidate lists to reflect changes
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient.delete(`/users/${id}`),

    onSuccess: (_, deletedId) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: userKeys.detail(deletedId) });
      // Refetch lists
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}
```

### useMutation in Components

```tsx
function CreateUserForm() {
  const { t } = useTranslation('users');
  const navigate = useNavigate();
  const { mutate: createUser, isPending } = useCreateUser();

  function handleSubmit(data: CreateUserDto) {
    createUser(data, {
      onSuccess: (newUser) => {
        toast.success(t('created'));
        navigate(`/users/${newUser.id}`);
      },
      onError: (error) => {
        toast.error(error.message);
      },
    });
  }

  return (
    <form onSubmit={form.handleSubmit(handleSubmit)}>
      {/* form fields */}
      <Button type="submit" isLoading={isPending}>
        {t('create')}
      </Button>
    </form>
  );
}
```

### Optimistic Updates

For instant UI feedback before the server confirms:

```tsx
export function useToggleFavorite() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (projectId: string) =>
      apiClient.post(`/projects/${projectId}/favorite`),

    // Optimistic update: change UI immediately
    onMutate: async (projectId) => {
      // Cancel in-flight queries to prevent overwrite
      await queryClient.cancelQueries({ queryKey: projectKeys.detail(projectId) });

      // Snapshot previous value for rollback
      const previousProject = queryClient.getQueryData<Project>(
        projectKeys.detail(projectId)
      );

      // Optimistically update the cache
      queryClient.setQueryData<Project>(
        projectKeys.detail(projectId),
        (old) => old ? { ...old, isFavorite: !old.isFavorite } : old
      );

      // Return snapshot for rollback
      return { previousProject };
    },

    // Rollback on error
    onError: (_error, projectId, context) => {
      if (context?.previousProject) {
        queryClient.setQueryData(
          projectKeys.detail(projectId),
          context.previousProject
        );
      }
    },

    // Always refetch to ensure consistency
    onSettled: (_, __, projectId) => {
      queryClient.invalidateQueries({ queryKey: projectKeys.detail(projectId) });
    },
  });
}
```

### useInfiniteQuery — Infinite Scroll

```tsx
export function useInfiniteUsers(filters: UserFilters = {}) {
  return useInfiniteQuery({
    queryKey: [...userKeys.lists(), 'infinite', filters],
    queryFn: ({ pageParam = 1 }) =>
      apiClient.get<PaginatedResponse<User>>('/users', {
        params: { ...filters, page: pageParam },
      }),
    initialPageParam: 1,
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage ? lastPage.page + 1 : undefined,
  });
}

// In component:
function UserInfiniteList() {
  const {
    data,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteUsers();

  // Flatten pages into a single array
  const users = data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <div>
      {users.map((user) => (
        <UserCard key={user.id} user={user} />
      ))}
      {hasNextPage && (
        <Button
          variant="ghost"
          onClick={() => fetchNextPage()}
          isLoading={isFetchingNextPage}
        >
          Load More
        </Button>
      )}
    </div>
  );
}
```

## Zustand

### Creating a Store

```tsx
// src/stores/ui.store.ts

import { create } from 'zustand';

interface UIState {
  // State
  sidebarOpen: boolean;
  theme: 'light' | 'dark' | 'system';

  // Actions
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
}

export const useUIStore = create<UIState>((set) => ({
  // Initial state
  sidebarOpen: true,
  theme: 'system',

  // Actions
  toggleSidebar: () =>
    set((state) => ({ sidebarOpen: !state.sidebarOpen })),

  setSidebarOpen: (open) =>
    set({ sidebarOpen: open }),

  setTheme: (theme) =>
    set({ theme }),
}));
```

### Selectors (Performance)

Always use selectors to prevent unnecessary re-renders:

```tsx
// BAD: Component re-renders whenever ANY store value changes
function Sidebar() {
  const store = useUIStore(); // Subscribes to entire store
  return <div>{store.sidebarOpen ? 'Open' : 'Closed'}</div>;
}

// GOOD: Component only re-renders when sidebarOpen changes
function Sidebar() {
  const sidebarOpen = useUIStore((s) => s.sidebarOpen);
  return <div>{sidebarOpen ? 'Open' : 'Closed'}</div>;
}

// GOOD: Multiple values with shallow comparison
import { useShallow } from 'zustand/react/shallow';

function Header() {
  const { sidebarOpen, theme } = useUIStore(
    useShallow((s) => ({ sidebarOpen: s.sidebarOpen, theme: s.theme }))
  );
}
```

### Persist Middleware

For state that survives page refresh (auth, preferences):

```tsx
// src/stores/auth.store.ts

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AuthState {
  accessToken: string | null;
  user: AuthUser | null;

  setAuth: (token: string, user: AuthUser) => void;
  clearAuth: () => void;
  updateUser: (user: Partial<AuthUser>) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      user: null,

      setAuth: (accessToken, user) =>
        set({ accessToken, user }),

      clearAuth: () =>
        set({ accessToken: null, user: null }),

      updateUser: (updates) =>
        set((state) => ({
          user: state.user ? { ...state.user, ...updates } : null,
        })),
    }),
    {
      name: 'auth-storage',
      // Only persist specific fields (not the entire store)
      partialize: (state) => ({
        accessToken: state.accessToken,
        user: state.user,
      }),
    }
  )
);
```

### Store with Computed Values

```tsx
// src/stores/filter.store.ts

import { create } from 'zustand';

interface FilterState {
  search: string;
  role: string | null;
  status: 'active' | 'inactive' | null;

  // Actions
  setSearch: (search: string) => void;
  setRole: (role: string | null) => void;
  setStatus: (status: 'active' | 'inactive' | null) => void;
  resetFilters: () => void;

  // Computed
  hasActiveFilters: () => boolean;
  getFilterParams: () => Record<string, string>;
}

const initialState = {
  search: '',
  role: null,
  status: null,
};

export const useFilterStore = create<FilterState>((set, get) => ({
  ...initialState,

  setSearch: (search) => set({ search }),
  setRole: (role) => set({ role }),
  setStatus: (status) => set({ status }),
  resetFilters: () => set(initialState),

  hasActiveFilters: () => {
    const { search, role, status } = get();
    return search !== '' || role !== null || status !== null;
  },

  getFilterParams: () => {
    const { search, role, status } = get();
    const params: Record<string, string> = {};
    if (search) params.search = search;
    if (role) params.role = role;
    if (status) params.status = status;
    return params;
  },
}));
```

## Connecting React Query + Zustand

Zustand filters drive React Query:

```tsx
function UsersPage() {
  // Client state: filters (Zustand)
  const filters = useFilterStore((s) => s.getFilterParams());
  const hasFilters = useFilterStore((s) => s.hasActiveFilters());
  const resetFilters = useFilterStore((s) => s.resetFilters);

  // Server state: data (React Query), driven by Zustand filters
  const { data: users, isLoading } = useUsers(filters);

  return (
    <div>
      <FilterBar />
      {hasFilters && (
        <Button variant="ghost" size="sm" onClick={resetFilters}>
          Clear filters
        </Button>
      )}
      {isLoading ? <LoadingState /> : <UserList users={users ?? []} />}
    </div>
  );
}
```

## Rules

1. **No Redux** — React Query + Zustand covers every case
2. **No useState for server data** — always React Query
3. **Always use selectors** with Zustand to prevent unnecessary re-renders
4. **Query keys are arrays** — use factory functions per feature
5. **Invalidate after mutation** — never manually set cache for list data (setQueryData is for detail/optimistic only)
6. **Persist only what must survive refresh** — auth token, user preferences, language
7. **Store actions live in the store** — never modify store state outside of store actions
