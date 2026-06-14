---
knowledge-base-summary: "Infinite scroll, virtual list (TanStack Virtual) for large datasets. Search with debounce. Filter chips. Empty/loading/error states. Skeleton loading. Pull-to-refresh equivalent (manual refresh button)."
---
# List Patterns

## Philosophy

React is a BRIDGE. Lists display data from the API with infinite scroll or virtual rendering. The API provides cursor-based pagination; React fetches page after page as the user scrolls. Search and filter parameters are sent to the API -- React never filters client-side. For lists with 10k+ items, TanStack Virtual renders only visible rows.

## Dependencies

```json
{
  "@tanstack/react-query": "^5.x",
  "@tanstack/react-virtual": "^3.x",
  "zustand": "^4.x"
}
```

## Infinite Scroll with useInfiniteQuery

```typescript
// hooks/use-tasks.ts
import { useInfiniteQuery } from '@tanstack/react-query';
import { api } from '@/lib/api-client';

interface TaskListParams {
  search?: string;
  status?: string;
  sortBy?: string;
}

export function useTasks(params: TaskListParams) {
  return useInfiniteQuery({
    queryKey: ['tasks', params],
    queryFn: ({ pageParam }) =>
      api
        .get<PaginatedResponse<Task>>('/tasks', {
          params: {
            cursor: pageParam,
            pageSize: 20,
            q: params.search,
            status: params.status,
            sortBy: params.sortBy,
          },
        })
        .then((res) => res.data),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
  });
}
```

## IntersectionObserver Hook

Trigger next page fetch when a sentinel element enters the viewport.

```typescript
// hooks/use-intersection-observer.ts
import { useEffect, useRef } from 'react';

interface UseIntersectionObserverOptions {
  onIntersect: () => void;
  enabled?: boolean;
  rootMargin?: string;
}

export function useIntersectionObserver({
  onIntersect,
  enabled = true,
  rootMargin = '200px',
}: UseIntersectionObserverOptions) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!enabled) return;

    const el = ref.current;
    if (!el) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          onIntersect();
        }
      },
      { rootMargin }
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, [enabled, onIntersect, rootMargin]);

  return ref;
}
```

## Debounced Search Hook

```typescript
// hooks/use-debounced-value.ts
import { useState, useEffect } from 'react';

export function useDebouncedValue<T>(value: T, delay = 400): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);

  return debounced;
}
```

## Filter State with Zustand

```typescript
// stores/task-filter-store.ts
import { create } from 'zustand';

interface TaskFilterState {
  search: string;
  status: string | null;
  sortBy: string;
  viewMode: 'list' | 'grid';
  setSearch: (search: string) => void;
  setStatus: (status: string | null) => void;
  setSortBy: (sortBy: string) => void;
  setViewMode: (mode: 'list' | 'grid') => void;
  reset: () => void;
}

const initialState = {
  search: '',
  status: null,
  sortBy: 'createdAt',
  viewMode: 'list' as const,
};

export const useTaskFilterStore = create<TaskFilterState>((set) => ({
  ...initialState,
  setSearch: (search) => set({ search }),
  setStatus: (status) => set({ status }),
  setSortBy: (sortBy) => set({ sortBy }),
  setViewMode: (mode) => set({ viewMode: mode }),
  reset: () => set(initialState),
}));
```

## Filter Chips Component

```typescript
// components/ui/filter-chips.tsx
import { X } from 'lucide-react';
import clsx from 'clsx';

interface FilterChip {
  key: string;
  label: string;
  value: string;
}

interface FilterChipsProps {
  chips: FilterChip[];
  onRemove: (key: string) => void;
  onClear: () => void;
}

export function FilterChips({ chips, onRemove, onClear }: FilterChipsProps) {
  if (chips.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-3 py-1 text-sm text-blue-700"
        >
          <span className="text-xs text-blue-400">{chip.label}:</span>
          {chip.value}
          <button
            onClick={() => onRemove(chip.key)}
            className="ml-1 rounded-full p-0.5 hover:bg-blue-100"
          >
            <X className="h-3 w-3" />
          </button>
        </span>
      ))}
      <button
        onClick={onClear}
        className="text-xs text-gray-500 hover:text-gray-700 underline"
      >
        Clear all
      </button>
    </div>
  );
}
```

## Grid/List Toggle

```typescript
// components/ui/view-toggle.tsx
import { LayoutGrid, List } from 'lucide-react';
import clsx from 'clsx';

interface ViewToggleProps {
  mode: 'list' | 'grid';
  onChange: (mode: 'list' | 'grid') => void;
}

export function ViewToggle({ mode, onChange }: ViewToggleProps) {
  return (
    <div className="flex rounded-md border border-gray-200">
      <button
        onClick={() => onChange('list')}
        className={clsx(
          'p-2 rounded-l-md',
          mode === 'list' ? 'bg-gray-100 text-gray-900' : 'text-gray-400 hover:text-gray-600'
        )}
        aria-label="List view"
      >
        <List className="h-4 w-4" />
      </button>
      <button
        onClick={() => onChange('grid')}
        className={clsx(
          'p-2 rounded-r-md',
          mode === 'grid' ? 'bg-gray-100 text-gray-900' : 'text-gray-400 hover:text-gray-600'
        )}
        aria-label="Grid view"
      >
        <LayoutGrid className="h-4 w-4" />
      </button>
    </div>
  );
}
```

## Virtual List for Large Datasets

For lists with 10k+ items, render only visible rows using TanStack Virtual.

```typescript
// components/ui/virtual-list.tsx
import { useRef } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';

interface VirtualListProps<T> {
  items: T[];
  estimateSize: number;
  renderItem: (item: T, index: number) => React.ReactNode;
  onEndReached?: () => void;
  endReachedThreshold?: number;
}

export function VirtualList<T>({
  items,
  estimateSize,
  renderItem,
  onEndReached,
  endReachedThreshold = 5,
}: VirtualListProps<T>) {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => estimateSize,
    overscan: 10,
    onChange: (instance) => {
      const lastItem = instance.getVirtualItems().at(-1);
      if (
        lastItem &&
        lastItem.index >= items.length - endReachedThreshold &&
        onEndReached
      ) {
        onEndReached();
      }
    },
  });

  return (
    <div ref={parentRef} className="h-full overflow-auto">
      <div
        className="relative w-full"
        style={{ height: `${virtualizer.getTotalSize()}px` }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => (
          <div
            key={virtualRow.key}
            className="absolute left-0 w-full"
            style={{
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            {renderItem(items[virtualRow.index], virtualRow.index)}
          </div>
        ))}
      </div>
    </div>
  );
}
```

## Complete Searchable Infinite Scroll List

```typescript
// features/tasks/tasks-list-page.tsx
import { useCallback, useMemo } from 'react';
import { Search, SlidersHorizontal } from 'lucide-react';
import { useTasks } from './use-tasks';
import { useTaskFilterStore } from '@/stores/task-filter-store';
import { useDebouncedValue } from '@/hooks/use-debounced-value';
import { useIntersectionObserver } from '@/hooks/use-intersection-observer';
import { FilterChips } from '@/components/ui/filter-chips';
import { ViewToggle } from '@/components/ui/view-toggle';
import { TaskCard } from './task-card';
import { TaskListItem } from './task-list-item';
import { ListSkeleton } from '@/components/ui/list-skeleton';
import { EmptyState } from '@/components/ui/empty-state';
import { ErrorState } from '@/components/ui/error-state';

export function TasksListPage() {
  const { search, status, sortBy, viewMode, setSearch, setStatus, setViewMode, reset } =
    useTaskFilterStore();

  const debouncedSearch = useDebouncedValue(search, 400);

  const {
    data,
    isLoading,
    isError,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
    refetch,
  } = useTasks({
    search: debouncedSearch,
    status: status ?? undefined,
    sortBy,
  });

  // Flatten all pages into one array
  const items = useMemo(
    () => data?.pages.flatMap((page) => page.items) ?? [],
    [data]
  );

  // Sentinel ref -- triggers fetchNextPage when visible
  const loadMoreRef = useIntersectionObserver({
    onIntersect: useCallback(() => {
      if (hasNextPage && !isFetchingNextPage) {
        fetchNextPage();
      }
    }, [hasNextPage, isFetchingNextPage, fetchNextPage]),
    enabled: hasNextPage && !isFetchingNextPage,
  });

  // Active filter chips
  const activeChips = useMemo(() => {
    const chips = [];
    if (status) chips.push({ key: 'status', label: 'Status', value: status });
    if (debouncedSearch) chips.push({ key: 'search', label: 'Search', value: debouncedSearch });
    return chips;
  }, [status, debouncedSearch]);

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Tasks</h1>
        <ViewToggle mode={viewMode} onChange={setViewMode} />
      </div>

      {/* Search Bar */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search tasks..."
          className="w-full rounded-lg border border-gray-200 py-2 pl-10 pr-4 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>

      {/* Filter Chips */}
      <FilterChips
        chips={activeChips}
        onRemove={(key) => {
          if (key === 'status') setStatus(null);
          if (key === 'search') setSearch('');
        }}
        onClear={reset}
      />

      {/* Status Filter Buttons */}
      <div className="flex gap-2">
        {['all', 'active', 'completed', 'paused'].map((s) => (
          <button
            key={s}
            onClick={() => setStatus(s === 'all' ? null : s)}
            className={`rounded-full px-3 py-1 text-sm ${
              (s === 'all' && !status) || status === s
                ? 'bg-blue-100 text-blue-700'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      {/* Content */}
      {isLoading ? (
        <ListSkeleton count={6} variant={viewMode} />
      ) : isError ? (
        <ErrorState
          message={error instanceof Error ? error.message : 'Failed to load tasks'}
          onRetry={() => refetch()}
        />
      ) : items.length === 0 ? (
        <EmptyState
          title="No tasks found"
          subtitle={search ? 'Try adjusting your search or filters' : 'Start your first task to see it here'}
          actionLabel={!search ? 'Get Started' : undefined}
          onAction={!search ? () => {} : undefined}
        />
      ) : (
        <>
          {/* List or Grid */}
          <div
            className={
              viewMode === 'grid'
                ? 'grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4'
                : 'space-y-2'
            }
          >
            {items.map((task) =>
              viewMode === 'grid' ? (
                <TaskCard key={task.id} task={task} />
              ) : (
                <TaskListItem key={task.id} task={task} />
              )
            )}
          </div>

          {/* Sentinel for infinite scroll */}
          <div ref={loadMoreRef} className="h-1" />

          {/* Loading more indicator */}
          {isFetchingNextPage && (
            <div className="flex justify-center py-4">
              <div className="h-6 w-6 animate-spin rounded-full border-2 border-gray-300 border-t-blue-600" />
            </div>
          )}

          {/* End of list */}
          {!hasNextPage && items.length > 0 && (
            <p className="text-center text-sm text-gray-400 py-4">
              End of list
            </p>
          )}
        </>
      )}
    </div>
  );
}
```

## List Skeleton Component

```typescript
// components/ui/list-skeleton.tsx
interface ListSkeletonProps {
  count: number;
  variant: 'list' | 'grid';
}

export function ListSkeleton({ count, variant }: ListSkeletonProps) {
  if (variant === 'grid') {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: count }).map((_, i) => (
          <div key={i} className="rounded-lg border border-gray-200 p-4 space-y-3 animate-pulse">
            <div className="h-32 bg-gray-100 rounded-md" />
            <div className="h-4 w-3/4 bg-gray-100 rounded" />
            <div className="h-3 w-1/2 bg-gray-100 rounded" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 rounded-lg border border-gray-200 p-4 animate-pulse">
          <div className="h-12 w-12 rounded-full bg-gray-100" />
          <div className="flex-1 space-y-2">
            <div className="h-4 w-1/3 bg-gray-100 rounded" />
            <div className="h-3 w-2/3 bg-gray-100 rounded" />
          </div>
          <div className="h-6 w-16 bg-gray-100 rounded-full" />
        </div>
      ))}
    </div>
  );
}
```

## Key Rules

1. **Cursor-based pagination** -- matches API pattern. `getNextPageParam` returns `nextCursor`. Never offset-based.
2. **useInfiniteQuery** -- React Query handles page accumulation. All pages are flattened with `flatMap`.
3. **IntersectionObserver for scroll detection** -- not `onScroll` event. A hidden sentinel div triggers the next fetch when it enters viewport.
4. **Debounce search 400ms** -- `useDebouncedValue` hook. Prevents API spam while typing.
5. **Filter state in Zustand** -- not URL params, not React state. Zustand persists across navigations.
6. **Skeleton on first load** -- matches the layout (grid skeleton for grid mode, list skeleton for list mode).
7. **Empty + error + loading** -- every list must handle all three states explicitly.
8. **Grid/list toggle** -- controlled by Zustand store. Same data, different layout.
9. **Virtual list for 10k+** -- use TanStack Virtual when the total item count is large. Renders only visible items.
10. **No client-side filtering** -- API filters. React sends query params. The only client-side operation is flattening pages.
