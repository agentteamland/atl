---
knowledge-base-summary: "Vitest for unit tests. React Testing Library for component tests. MSW (Mock Service Worker) for API mocking. User-event for interaction simulation. What to test: user behavior, not implementation."
---
# Testing

## Philosophy

Test what the user sees and does, not implementation details. Render the component, interact with it (click, type, select), and assert what changed on screen. MSW (Mock Service Worker) intercepts API calls at the network level so components run their real fetch logic. React Query and Zustand stores are tested through the components that use them, not in isolation.

## Dependencies

```json
{
  "vitest": "^1.x",
  "@testing-library/react": "^14.x",
  "@testing-library/jest-dom": "^6.x",
  "@testing-library/user-event": "^14.x",
  "msw": "^2.x",
  "jsdom": "^24.x"
}
```

## Vitest Configuration

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    css: false, // skip CSS parsing in tests
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/test/**', 'src/**/*.d.ts', 'src/main.tsx'],
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
});
```

## Test Setup File

```typescript
// src/test/setup.ts
import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, beforeAll, afterAll } from 'vitest';
import { server } from './mocks/server';

// Start MSW server before all tests
beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }));

// Reset handlers between tests
afterEach(() => {
  cleanup();
  server.resetHandlers();
});

// Clean up after all tests
afterAll(() => server.close());
```

## MSW Setup

```typescript
// src/test/mocks/handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  // GET /users -- paginated list
  http.get('/api/users', ({ request }) => {
    const url = new URL(request.url);
    const search = url.searchParams.get('q') ?? '';
    const pageSize = Number(url.searchParams.get('pageSize') ?? '20');

    const allUsers = [
      { id: '1', name: 'Alice Johnson', email: 'alice@example.com', role: 'admin', status: 'active' },
      { id: '2', name: 'Bob Smith', email: 'bob@example.com', role: 'user', status: 'active' },
      { id: '3', name: 'Charlie Brown', email: 'charlie@example.com', role: 'viewer', status: 'inactive' },
    ];

    const filtered = search
      ? allUsers.filter((u) => u.name.toLowerCase().includes(search.toLowerCase()))
      : allUsers;

    return HttpResponse.json({
      items: filtered.slice(0, pageSize),
      nextCursor: null,
      totalCount: filtered.length,
    });
  }),

  // GET /users/:id
  http.get('/api/users/:id', ({ params }) => {
    return HttpResponse.json({
      id: params.id,
      name: 'Alice Johnson',
      email: 'alice@example.com',
      role: 'admin',
      status: 'active',
      createdAt: '2024-01-15T10:30:00Z',
    });
  }),

  // POST /users
  http.post('/api/users', async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json(
      { id: '4', ...body, createdAt: new Date().toISOString() },
      { status: 201 }
    );
  }),

  // DELETE /users/:id
  http.delete('/api/users/:id', () => {
    return new HttpResponse(null, { status: 204 });
  }),
];
```

```typescript
// src/test/mocks/server.ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
```

## Test Render Helper

Wraps components with all necessary providers (QueryClient, Router, i18n).

```typescript
// src/test/render.tsx
import { ReactNode } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { I18nextProvider } from 'react-i18next';
import i18n from '@/lib/i18n';

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  route?: string;
}

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,           // no retries in tests
        gcTime: 0,              // garbage collect immediately
        staleTime: 0,           // always stale in tests
      },
      mutations: {
        retry: false,
      },
    },
  });
}

export function renderWithProviders(
  ui: ReactNode,
  { route = '/', ...options }: CustomRenderOptions = {}
) {
  const queryClient = createTestQueryClient();

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <MemoryRouter initialEntries={[route]}>
            {children}
          </MemoryRouter>
        </I18nextProvider>
      </QueryClientProvider>
    );
  }

  return {
    ...render(ui, { wrapper: Wrapper, ...options }),
    queryClient,
  };
}
```

## Testing a Component: Full Example

```typescript
// features/users/__tests__/users-page.test.tsx
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { server } from '@/test/mocks/server';
import { renderWithProviders } from '@/test/render';
import { UsersPage } from '../users-page';

describe('UsersPage', () => {
  it('renders the user list after loading', async () => {
    renderWithProviders(<UsersPage />);

    // Should show loading skeleton first
    expect(screen.getByText('Users')).toBeInTheDocument();

    // Wait for data to load
    await waitFor(() => {
      expect(screen.getByText('Alice Johnson')).toBeInTheDocument();
    });

    expect(screen.getByText('Bob Smith')).toBeInTheDocument();
    expect(screen.getByText('Charlie Brown')).toBeInTheDocument();
  });

  it('filters users by search', async () => {
    const user = userEvent.setup();
    renderWithProviders(<UsersPage />);

    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByText('Alice Johnson')).toBeInTheDocument();
    });

    // Type in the search input
    const searchInput = screen.getByPlaceholderText('Search users...');
    await user.type(searchInput, 'Alice');

    // Wait for filtered results
    await waitFor(() => {
      expect(screen.getByText('Alice Johnson')).toBeInTheDocument();
      expect(screen.queryByText('Bob Smith')).not.toBeInTheDocument();
    });
  });

  it('shows empty state when no results match', async () => {
    // Override handler for this test
    server.use(
      http.get('/api/users', () => {
        return HttpResponse.json({
          items: [],
          nextCursor: null,
          totalCount: 0,
        });
      })
    );

    renderWithProviders(<UsersPage />);

    await waitFor(() => {
      expect(screen.getByText('No results found')).toBeInTheDocument();
    });
  });

  it('shows error state and allows retry', async () => {
    // First call fails, second succeeds
    let callCount = 0;
    server.use(
      http.get('/api/users', () => {
        callCount++;
        if (callCount === 1) {
          return HttpResponse.json({ message: 'Server error' }, { status: 500 });
        }
        return HttpResponse.json({
          items: [{ id: '1', name: 'Alice Johnson', email: 'alice@example.com', role: 'admin', status: 'active' }],
          nextCursor: null,
          totalCount: 1,
        });
      })
    );

    const user = userEvent.setup();
    renderWithProviders(<UsersPage />);

    // Wait for error state
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });

    // Click retry
    await user.click(screen.getByRole('button', { name: /try again/i }));

    // Should show data after retry
    await waitFor(() => {
      expect(screen.getByText('Alice Johnson')).toBeInTheDocument();
    });
  });
});
```

## Testing a Form

```typescript
// features/users/__tests__/create-user-modal.test.tsx
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { CreateUserModal } from '../create-user-modal';

describe('CreateUserModal', () => {
  const onClose = vi.fn();

  beforeEach(() => {
    onClose.mockClear();
  });

  it('submits the form with valid data', async () => {
    const user = userEvent.setup();
    renderWithProviders(<CreateUserModal open={true} onClose={onClose} />);

    // Fill the form
    await user.type(screen.getByLabelText(/name/i), 'New User');
    await user.type(screen.getByLabelText(/email/i), 'new@example.com');

    // Submit
    await user.click(screen.getByRole('button', { name: /create/i }));

    // Modal should close on success
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('shows validation errors for empty fields', async () => {
    const user = userEvent.setup();
    renderWithProviders(<CreateUserModal open={true} onClose={onClose} />);

    // Submit without filling
    await user.click(screen.getByRole('button', { name: /create/i }));

    // Should NOT close (form invalid)
    expect(onClose).not.toHaveBeenCalled();
  });

  it('closes on cancel', async () => {
    const user = userEvent.setup();
    renderWithProviders(<CreateUserModal open={true} onClose={onClose} />);

    await user.click(screen.getByRole('button', { name: /cancel/i }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
```

## Testing Mutation Side Effects

```typescript
// features/users/__tests__/delete-user.test.tsx
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test/render';
import { UserRow } from '../user-row';

describe('Delete user flow', () => {
  it('shows confirmation dialog and deletes on confirm', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <UserRow user={{ id: '1', name: 'Alice', email: 'alice@example.com', role: 'admin', status: 'active' }} />
    );

    // Open row actions
    await user.click(screen.getByRole('button', { name: /row actions/i }));

    // Click delete
    await user.click(screen.getByText('Delete'));

    // Confirmation dialog appears
    expect(screen.getByText('Delete User')).toBeInTheDocument();
    expect(screen.getByText(/are you sure/i)).toBeInTheDocument();

    // Confirm deletion
    await user.click(screen.getByRole('button', { name: /delete/i }));

    // Dialog should close after successful deletion
    await waitFor(() => {
      expect(screen.queryByText('Delete User')).not.toBeInTheDocument();
    });
  });
});
```

## Resetting Zustand Stores Between Tests

```typescript
// src/test/setup.ts (add to existing setup)
import { useTaskFilterStore } from '@/stores/task-filter-store';
import { useLayoutStore } from '@/stores/layout-store';

afterEach(() => {
  // Reset all Zustand stores to initial state
  useTaskFilterStore.getState().reset();
  useLayoutStore.setState({
    sidebarCollapsed: false,
    mobileMenuOpen: false,
  });
});
```

## Overriding MSW Handlers Per Test

```typescript
import { http, HttpResponse } from 'msw';
import { server } from '@/test/mocks/server';

it('handles server error', async () => {
  // Override just for this test
  server.use(
    http.get('/api/users', () => {
      return HttpResponse.json(
        { message: 'Internal Server Error' },
        { status: 500 }
      );
    })
  );

  renderWithProviders(<UsersPage />);

  await waitFor(() => {
    expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
  });
});

it('handles 422 validation error on create', async () => {
  server.use(
    http.post('/api/users', () => {
      return HttpResponse.json(
        {
          errors: {
            email: ['Email already exists'],
          },
        },
        { status: 422 }
      );
    })
  );

  // ... test that validation error is displayed
});
```

## File Organization

```
src/
  features/
    users/
      users-page.tsx
      create-user-modal.tsx
      user-row.tsx
      __tests__/
        users-page.test.tsx
        create-user-modal.test.tsx
        delete-user.test.tsx
  hooks/
    use-debounced-value.ts
    __tests__/
      use-debounced-value.test.ts
  components/
    ui/
      data-table.tsx
      __tests__/
        data-table.test.tsx
  test/
    setup.ts
    render.tsx
    mocks/
      handlers.ts
      server.ts
```

Two valid patterns for test file location:
- `__tests__/` directory next to components -- groups tests together, keeps component directory clean
- `.test.tsx` suffix next to component -- test is immediately discoverable

Pick one pattern per project and stick with it.

## What to Test

| Test | Example |
|------|---------|
| User sees content after loading | Data appears in the table after API returns |
| User searches and results filter | Typing in search input triggers filtered API call |
| User submits a form | Fill inputs, click submit, verify success |
| Empty state renders | API returns 0 items, empty state is visible |
| Error state renders + retry works | API returns 500, error shown, retry button works |
| Modal opens and closes | Click trigger, modal appears; click cancel, modal closes |
| Confirmation prevents accidental deletion | Delete requires confirming a dialog |

## What NOT to Test

| Do NOT Test | Why |
|-------------|-----|
| Internal state values (`useState`, Zustand internal state) | Test through UI behavior |
| Whether `useEffect` was called | Implementation detail |
| Component snapshot | Fragile, breaks on every styling change |
| React Query cache internals | Test through visible data changes |
| CSS class names | Use `getByRole`, `getByText`, not class selectors |
| Third-party library internals | Not your code |

## Key Rules

1. **Render, interact, assert** -- every test follows this pattern. Render the component, simulate user actions, check visible output.
2. **`userEvent.setup()`** -- always use `userEvent.setup()`, not `fireEvent`. `userEvent` simulates real browser behavior (focus, input events, etc.).
3. **MSW at network level** -- mock the API at the network layer, not by mocking Axios or React Query hooks. Components exercise their real fetch logic.
4. **`waitFor` for async** -- API calls are async. Use `waitFor` to wait for elements to appear after data loads.
5. **Query by role/text** -- `screen.getByRole('button', { name: /create/i })` over `screen.getByTestId`. Screen readers use roles; tests should too.
6. **No retry in tests** -- `retry: false` in test QueryClient. Tests should fail immediately, not retry silently.
7. **Reset stores after each test** -- call `reset()` or `setState()` on Zustand stores in `afterEach` to avoid test pollution.
8. **Override handlers per test** -- `server.use()` in individual tests overrides default handlers for edge cases (errors, empty responses).
9. **`vi.fn()` for callbacks** -- use Vitest mocks for `onClose`, `onSubmit` props. Assert they were called with expected arguments.
10. **No snapshot tests** -- they break on every UI change and provide no behavioral confidence. Test user behavior instead.
