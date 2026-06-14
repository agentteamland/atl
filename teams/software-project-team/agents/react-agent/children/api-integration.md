---
knowledge-base-summary: "Axios client with interceptors: auth token injection, 401 → refresh token → retry. Typed API response models with Zod validation. React Query hooks for data fetching. Error handling and transformation."
---
# API Integration

Axios for HTTP. React Query for data fetching. Zod for runtime validation. Typed end-to-end.

## Axios Instance

```tsx
// src/lib/api-client.ts

import axios, { type AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios';
import { useAuthStore } from '@/stores/auth.store';

// ─── Create Axios instance ───
const axiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:3000',
  timeout: 30_000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// ─── Request Interceptor: Inject Auth Token ───
axiosInstance.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = useAuthStore.getState().accessToken;

    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// ─── Response Interceptor: Handle 401 + Refresh Token ───
let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (error: unknown) => void;
}> = [];

function processQueue(error: unknown, token: string | null) {
  failedQueue.forEach(({ resolve, reject }) => {
    if (token) {
      resolve(token);
    } else {
      reject(error);
    }
  });
  failedQueue = [];
}

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // Only handle 401 (Unauthorized)
    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    // If already refreshing, queue this request
    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      }).then((token) => {
        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${token}`;
        }
        return axiosInstance(originalRequest);
      });
    }

    originalRequest._retry = true;
    isRefreshing = true;

    try {
      // Attempt to refresh the token
      const response = await axios.post<{ accessToken: string }>(
        `${import.meta.env.VITE_API_URL}/auth/refresh`,
        {},
        { withCredentials: true } // Send refresh token cookie
      );

      const newToken = response.data.accessToken;
      useAuthStore.getState().setAuth(newToken, useAuthStore.getState().user!);

      // Process queued requests with new token
      processQueue(null, newToken);

      // Retry original request
      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${newToken}`;
      }
      return axiosInstance(originalRequest);
    } catch (refreshError) {
      // Refresh failed — clear auth and redirect to login
      processQueue(refreshError, null);
      useAuthStore.getState().clearAuth();
      window.location.href = '/login';
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  }
);

export { axiosInstance };
```

## API Client Class

A typed wrapper over Axios that extracts the response data:

```tsx
// src/lib/api-client.ts (continued, same file)

import { type ZodType, type ZodTypeDef } from 'zod';

class ApiClient {
  /**
   * GET request with optional Zod validation
   */
  async get<T>(url: string, config?: { params?: Record<string, unknown>; schema?: ZodType<T> }): Promise<T> {
    const response: AxiosResponse<T> = await axiosInstance.get(url, {
      params: config?.params,
    });
    return config?.schema
      ? config.schema.parse(response.data)
      : response.data;
  }

  /**
   * POST request
   */
  async post<T>(url: string, data?: unknown, config?: { schema?: ZodType<T> }): Promise<T> {
    const response: AxiosResponse<T> = await axiosInstance.post(url, data);
    return config?.schema
      ? config.schema.parse(response.data)
      : response.data;
  }

  /**
   * PUT request
   */
  async put<T>(url: string, data?: unknown, config?: { schema?: ZodType<T> }): Promise<T> {
    const response: AxiosResponse<T> = await axiosInstance.put(url, data);
    return config?.schema
      ? config.schema.parse(response.data)
      : response.data;
  }

  /**
   * PATCH request
   */
  async patch<T>(url: string, data?: unknown, config?: { schema?: ZodType<T> }): Promise<T> {
    const response: AxiosResponse<T> = await axiosInstance.patch(url, data);
    return config?.schema
      ? config.schema.parse(response.data)
      : response.data;
  }

  /**
   * DELETE request
   */
  async delete<T = void>(url: string): Promise<T> {
    const response: AxiosResponse<T> = await axiosInstance.delete(url);
    return response.data;
  }

  /**
   * POST with FormData (file uploads)
   */
  async upload<T>(url: string, formData: FormData, config?: { schema?: ZodType<T> }): Promise<T> {
    const response: AxiosResponse<T> = await axiosInstance.post(url, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
    return config?.schema
      ? config.schema.parse(response.data)
      : response.data;
  }
}

export const apiClient = new ApiClient();
```

## Zod Runtime Validation

API responses can differ from what TypeScript expects. Zod validates at runtime:

```tsx
// src/features/users/types/users.types.ts

import { z } from 'zod';

// ─── Zod Schemas (runtime validation) ───

export const userSchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  name: z.string(),
  role: z.enum(['admin', 'member', 'viewer']),
  avatarUrl: z.string().url().nullable(),
  isActive: z.boolean(),
  createdAt: z.string().datetime(),
  updatedAt: z.string().datetime(),
});

export const userListSchema = z.array(userSchema);

export const paginatedUsersSchema = z.object({
  items: z.array(userSchema),
  total: z.number(),
  page: z.number(),
  pageSize: z.number(),
  hasNextPage: z.boolean(),
});

// ─── TypeScript Types (derived from Zod) ───

export type User = z.infer<typeof userSchema>;
export type PaginatedUsers = z.infer<typeof paginatedUsersSchema>;

// ─── DTOs (for mutations — no Zod needed, just TS) ───

export interface CreateUserDto {
  email: string;
  name: string;
  role: 'admin' | 'member' | 'viewer';
  password: string;
}

export interface UpdateUserDto {
  name?: string;
  role?: 'admin' | 'member' | 'viewer';
  isActive?: boolean;
}

// ─── Filters ───

export interface UserFilters {
  search?: string;
  role?: string;
  status?: 'active' | 'inactive';
  page?: number;
  pageSize?: number;
}
```

## React Query Custom Hooks

Complete pattern for a feature's API layer:

```tsx
// src/features/users/api/users.api.ts

import { useQuery, useMutation, useQueryClient, useInfiniteQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import {
  type User,
  type PaginatedUsers,
  type CreateUserDto,
  type UpdateUserDto,
  type UserFilters,
  userSchema,
  paginatedUsersSchema,
} from '../types/users.types';

// ─── Query Key Factory ───

export const userKeys = {
  all: ['users'] as const,
  lists: () => [...userKeys.all, 'list'] as const,
  list: (filters: UserFilters) => [...userKeys.lists(), filters] as const,
  details: () => [...userKeys.all, 'detail'] as const,
  detail: (id: string) => [...userKeys.details(), id] as const,
};

// ─── Queries ───

export function useUsers(filters: UserFilters = {}) {
  return useQuery({
    queryKey: userKeys.list(filters),
    queryFn: () =>
      apiClient.get<PaginatedUsers>('/users', {
        params: filters as Record<string, unknown>,
        schema: paginatedUsersSchema,
      }),
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () =>
      apiClient.get<User>(`/users/${id}`, { schema: userSchema }),
    enabled: !!id,
  });
}

// ─── Mutations ───

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
    onSuccess: (updatedUser, { id }) => {
      queryClient.setQueryData(userKeys.detail(id), updatedUser);
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => apiClient.delete(`/users/${id}`),
    onSuccess: (_, deletedId) => {
      queryClient.removeQueries({ queryKey: userKeys.detail(deletedId) });
      queryClient.invalidateQueries({ queryKey: userKeys.lists() });
    },
  });
}
```

## Error Handling

Transform AxiosError into a consistent app error type:

```tsx
// src/lib/api-error.ts

import { AxiosError } from 'axios';

export interface ApiError {
  message: string;
  statusCode: number;
  errors?: Record<string, string[]>; // Field-level validation errors from API
}

/**
 * Transform AxiosError into a structured ApiError.
 * Used in React Query's onError callbacks and error boundaries.
 */
export function parseApiError(error: unknown): ApiError {
  if (error instanceof AxiosError) {
    const data = error.response?.data as {
      message?: string;
      errors?: Record<string, string[]>;
    } | undefined;

    return {
      message: data?.message || error.message || 'An unexpected error occurred',
      statusCode: error.response?.status || 500,
      errors: data?.errors,
    };
  }

  if (error instanceof Error) {
    return {
      message: error.message,
      statusCode: 500,
    };
  }

  return {
    message: 'An unexpected error occurred',
    statusCode: 500,
  };
}
```

### Using in Components

```tsx
import { parseApiError } from '@/lib/api-error';

function CreateUserForm() {
  const { mutate: createUser, isPending } = useCreateUser();

  function handleSubmit(data: CreateUserDto) {
    createUser(data, {
      onSuccess: (user) => {
        toast.success(t('userCreated'));
        navigate(`/users/${user.id}`);
      },
      onError: (error) => {
        const apiError = parseApiError(error);

        if (apiError.statusCode === 409) {
          // Email already exists
          form.setError('email', { message: t('emailTaken') });
          return;
        }

        if (apiError.errors) {
          // Map server validation errors to form fields
          Object.entries(apiError.errors).forEach(([field, messages]) => {
            form.setError(field as keyof CreateUserDto, {
              message: messages[0],
            });
          });
          return;
        }

        // Generic error
        toast.error(apiError.message);
      },
    });
  }
}
```

## Paginated Response Type

```tsx
// src/types/api.types.ts

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
}

// Zod schema factory for paginated responses
import { z, type ZodType } from 'zod';

export function paginatedSchema<T>(itemSchema: ZodType<T>) {
  return z.object({
    items: z.array(itemSchema),
    total: z.number(),
    page: z.number(),
    pageSize: z.number(),
    hasNextPage: z.boolean(),
    hasPreviousPage: z.boolean(),
  });
}
```

## Environment Variables

```tsx
// .env
VITE_API_URL=http://localhost:3000

// .env.production
VITE_API_URL=https://api.myapp.com

// Access in code (Vite injects VITE_ prefixed vars):
const baseURL = import.meta.env.VITE_API_URL;
```

## Rules

1. **One apiClient instance** — shared across the entire app
2. **Interceptors handle auth** — components never manually add tokens
3. **Zod validates responses** — trust nothing from the network
4. **TypeScript types derived from Zod** — `z.infer<typeof schema>`
5. **DTOs are plain interfaces** — no Zod for outgoing data (API validates)
6. **Query key factories** — consistent keys per feature
7. **parseApiError for every onError** — consistent error handling
8. **Never fetch in useEffect** — always React Query
9. **withCredentials for refresh** — refresh token travels as httpOnly cookie
