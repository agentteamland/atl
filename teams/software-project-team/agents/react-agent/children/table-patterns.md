---
knowledge-base-summary: "TanStack Table for data tables. Server-side sorting, filtering, pagination. Column definitions with TypeScript. Row actions (edit, delete). Selectable rows. Responsive: table on desktop, card list on mobile."
---
# Table Patterns

## Philosophy

React is a BRIDGE. Tables display paginated data from the API. The API handles sorting, filtering, and pagination -- React mirrors those parameters as query strings. TanStack Table provides the headless table engine; we supply the UI with Tailwind. On mobile, tables collapse into card lists because horizontal scrolling is bad UX.

## Dependencies

```json
{
  "@tanstack/react-table": "^8.x",
  "@tanstack/react-query": "^5.x",
  "clsx": "^2.x"
}
```

## Types

```typescript
// types/api.ts -- shared across all paginated endpoints
interface PaginatedResponse<T> {
  items: T[];
  nextCursor: string | null;
  totalCount: number;
}

interface PaginationParams {
  cursor?: string;
  pageSize: number;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  search?: string;
  filters?: Record<string, string>;
}

// types/user.ts -- example entity
interface User {
  id: string;
  name: string;
  email: string;
  role: 'admin' | 'user' | 'viewer';
  status: 'active' | 'inactive';
  createdAt: string;
}
```

## API Hook with Server-Side Pagination

```typescript
// hooks/use-users.ts
import { useQuery, keepPreviousData } from '@tanstack/react-query';
import { api } from '@/lib/api-client';

interface UseUsersParams {
  page: number;
  pageSize: number;
  sortBy?: string;
  sortDirection?: 'asc' | 'desc';
  search?: string;
  filters?: Record<string, string>;
}

export function useUsers(params: UseUsersParams) {
  return useQuery({
    queryKey: ['users', params],
    queryFn: () =>
      api.get<PaginatedResponse<User>>('/users', {
        params: {
          cursor: params.page > 0 ? params.page.toString() : undefined,
          pageSize: params.pageSize,
          sortBy: params.sortBy,
          sortDirection: params.sortDirection,
          q: params.search,
          ...params.filters,
        },
      }).then((res) => res.data),
    placeholderData: keepPreviousData,
  });
}
```

## Column Definitions

```typescript
// features/users/columns.tsx
import { ColumnDef } from '@tanstack/react-table';
import { RowActions } from './row-actions';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { formatDate } from '@/lib/utils';

export const userColumns: ColumnDef<User>[] = [
  // Checkbox selection column
  {
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        checked={table.getIsAllPageRowsSelected()}
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
      />
    ),
    enableSorting: false,
    size: 40,
  },

  // Name column -- sortable
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => (
      <div className="flex items-center gap-3">
        <div className="h-8 w-8 rounded-full bg-gray-200 flex items-center justify-center text-sm font-medium text-gray-600">
          {row.original.name.charAt(0)}
        </div>
        <div>
          <div className="font-medium text-gray-900">{row.original.name}</div>
          <div className="text-sm text-gray-500">{row.original.email}</div>
        </div>
      </div>
    ),
  },

  // Role column
  {
    accessorKey: 'role',
    header: 'Role',
    cell: ({ row }) => (
      <Badge variant={row.original.role === 'admin' ? 'default' : 'secondary'}>
        {row.original.role}
      </Badge>
    ),
  },

  // Status column
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => (
      <Badge variant={row.original.status === 'active' ? 'success' : 'muted'}>
        {row.original.status}
      </Badge>
    ),
  },

  // Date column
  {
    accessorKey: 'createdAt',
    header: 'Created',
    cell: ({ row }) => (
      <span className="text-sm text-gray-500">
        {formatDate(row.original.createdAt)}
      </span>
    ),
  },

  // Actions column
  {
    id: 'actions',
    header: '',
    cell: ({ row }) => <RowActions user={row.original} />,
    size: 50,
  },
];
```

## Row Actions Dropdown

```typescript
// features/users/row-actions.tsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { MoreHorizontal, Eye, Pencil, Trash2 } from 'lucide-react';
import { useDeleteUser } from './use-delete-user';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';

interface RowActionsProps {
  user: User;
}

export function RowActions({ user }: RowActionsProps) {
  const navigate = useNavigate();
  const deleteUser = useDeleteUser();
  const [showConfirm, setShowConfirm] = useState(false);
  const [open, setOpen] = useState(false);

  return (
    <>
      <div className="relative">
        <button
          onClick={() => setOpen(!open)}
          className="p-1 rounded hover:bg-gray-100"
          aria-label="Row actions"
        >
          <MoreHorizontal className="h-4 w-4" />
        </button>

        {open && (
          <div className="absolute right-0 top-8 z-10 w-48 rounded-md border bg-white shadow-lg">
            <button
              onClick={() => {
                navigate(`/users/${user.id}`);
                setOpen(false);
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50"
            >
              <Eye className="h-4 w-4" /> View
            </button>
            <button
              onClick={() => {
                navigate(`/users/${user.id}/edit`);
                setOpen(false);
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50"
            >
              <Pencil className="h-4 w-4" /> Edit
            </button>
            <button
              onClick={() => {
                setShowConfirm(true);
                setOpen(false);
              }}
              className="flex w-full items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
            >
              <Trash2 className="h-4 w-4" /> Delete
            </button>
          </div>
        )}
      </div>

      <ConfirmDialog
        open={showConfirm}
        onClose={() => setShowConfirm(false)}
        onConfirm={() => deleteUser.mutate(user.id)}
        title="Delete User"
        description={`Are you sure you want to delete ${user.name}? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        loading={deleteUser.isPending}
      />
    </>
  );
}
```

## Complete DataTable Component

```typescript
// components/ui/data-table.tsx
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  ColumnDef,
  SortingState,
  RowSelectionState,
} from '@tanstack/react-table';
import { ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react';
import clsx from 'clsx';

interface DataTableProps<T> {
  columns: ColumnDef<T, unknown>[];
  data: T[];
  totalCount: number;
  pageSize: number;
  page: number;
  onPageChange: (page: number) => void;
  sorting?: SortingState;
  onSortingChange?: (sorting: SortingState) => void;
  rowSelection?: RowSelectionState;
  onRowSelectionChange?: (selection: RowSelectionState) => void;
  isLoading?: boolean;
  emptyState?: React.ReactNode;
}

export function DataTable<T>({
  columns,
  data,
  totalCount,
  pageSize,
  page,
  onPageChange,
  sorting = [],
  onSortingChange,
  rowSelection = {},
  onRowSelectionChange,
  isLoading,
  emptyState,
}: DataTableProps<T>) {
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    manualSorting: true,       // server handles sorting
    manualPagination: true,    // server handles pagination
    state: { sorting, rowSelection },
    onSortingChange: (updater) => {
      const next = typeof updater === 'function' ? updater(sorting) : updater;
      onSortingChange?.(next);
    },
    onRowSelectionChange: (updater) => {
      const next = typeof updater === 'function' ? updater(rowSelection) : updater;
      onRowSelectionChange?.(next);
    },
    pageCount: Math.ceil(totalCount / pageSize),
    enableRowSelection: true,
  });

  const totalPages = Math.ceil(totalCount / pageSize);

  // --- LOADING: Skeleton ---
  if (isLoading && data.length === 0) {
    return <TableSkeleton columns={columns.length} rows={pageSize} />;
  }

  // --- EMPTY STATE ---
  if (!isLoading && data.length === 0) {
    return emptyState ?? <DefaultEmptyState />;
  }

  return (
    <div>
      {/* Desktop: Table */}
      <div className="hidden md:block overflow-x-auto rounded-lg border border-gray-200">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b border-gray-200">
            {table.getHeaderGroups().map((headerGroup) => (
              <tr key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <th
                    key={header.id}
                    className={clsx(
                      'px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider',
                      header.column.getCanSort() && 'cursor-pointer select-none hover:text-gray-700'
                    )}
                    style={{ width: header.getSize() !== 150 ? header.getSize() : undefined }}
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    <div className="flex items-center gap-1">
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {header.column.getCanSort() && (
                        <SortIcon direction={header.column.getIsSorted()} />
                      )}
                    </div>
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-gray-200 bg-white">
            {table.getRowModel().rows.map((row) => (
              <tr
                key={row.id}
                className={clsx(
                  'hover:bg-gray-50 transition-colors',
                  row.getIsSelected() && 'bg-blue-50'
                )}
              >
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-4 py-3 whitespace-nowrap">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Mobile: Card List */}
      <div className="md:hidden space-y-3">
        {table.getRowModel().rows.map((row) => (
          <MobileCard key={row.id} row={row} columns={columns} />
        ))}
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200">
        <div className="text-sm text-gray-500">
          {Object.keys(rowSelection).length > 0 && (
            <span>{Object.keys(rowSelection).length} selected / </span>
          )}
          Showing {page * pageSize + 1}-{Math.min((page + 1) * pageSize, totalCount)} of {totalCount}
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={page === 0}
            className="px-3 py-1 text-sm border rounded-md disabled:opacity-50 hover:bg-gray-50"
          >
            Previous
          </button>
          <button
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages - 1}
            className="px-3 py-1 text-sm border rounded-md disabled:opacity-50 hover:bg-gray-50"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Sort Icon ---
function SortIcon({ direction }: { direction: false | 'asc' | 'desc' }) {
  if (direction === 'asc') return <ArrowUp className="h-3 w-3" />;
  if (direction === 'desc') return <ArrowDown className="h-3 w-3" />;
  return <ArrowUpDown className="h-3 w-3 text-gray-300" />;
}

// --- Mobile Card ---
function MobileCard<T>({
  row,
  columns,
}: {
  row: import('@tanstack/react-table').Row<T>;
  columns: ColumnDef<T, unknown>[];
}) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 space-y-2">
      {row.getVisibleCells().map((cell) => {
        if (cell.column.id === 'select') return null;
        return (
          <div key={cell.id} className="flex items-center justify-between">
            <span className="text-xs text-gray-500 uppercase">
              {typeof cell.column.columnDef.header === 'string'
                ? cell.column.columnDef.header
                : cell.column.id}
            </span>
            <div className="text-sm">
              {flexRender(cell.column.columnDef.cell, cell.getContext())}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// --- Loading Skeleton ---
function TableSkeleton({ columns, rows }: { columns: number; rows: number }) {
  return (
    <div className="rounded-lg border border-gray-200 overflow-hidden">
      <div className="bg-gray-50 border-b border-gray-200 px-4 py-3">
        <div className="flex gap-8">
          {Array.from({ length: columns }).map((_, i) => (
            <div key={i} className="h-3 w-20 bg-gray-200 rounded animate-pulse" />
          ))}
        </div>
      </div>
      <div className="divide-y divide-gray-200">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="px-4 py-3 flex gap-8">
            {Array.from({ length: columns }).map((_, j) => (
              <div
                key={j}
                className="h-4 bg-gray-100 rounded animate-pulse"
                style={{ width: `${60 + Math.random() * 80}px` }}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

// --- Default Empty State ---
function DefaultEmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="h-16 w-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <svg className="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
      </div>
      <h3 className="text-sm font-medium text-gray-900">No results found</h3>
      <p className="mt-1 text-sm text-gray-500">Try adjusting your search or filter criteria.</p>
    </div>
  );
}
```

## Page-Level Usage

```typescript
// features/users/users-page.tsx
import { useState } from 'react';
import { SortingState, RowSelectionState } from '@tanstack/react-table';
import { DataTable } from '@/components/ui/data-table';
import { userColumns } from './columns';
import { useUsers } from './use-users';

export function UsersPage() {
  const [page, setPage] = useState(0);
  const [sorting, setSorting] = useState<SortingState>([]);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [search, setSearch] = useState('');
  const pageSize = 20;

  const { data, isLoading } = useUsers({
    page,
    pageSize,
    sortBy: sorting[0]?.id,
    sortDirection: sorting[0]?.desc ? 'desc' : 'asc',
    search,
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Users</h1>
        <input
          type="text"
          placeholder="Search users..."
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(0); // reset to first page on search
          }}
          className="rounded-md border px-3 py-2 text-sm"
        />
      </div>

      <DataTable
        columns={userColumns}
        data={data?.items ?? []}
        totalCount={data?.totalCount ?? 0}
        pageSize={pageSize}
        page={page}
        onPageChange={setPage}
        sorting={sorting}
        onSortingChange={setSorting}
        rowSelection={rowSelection}
        onRowSelectionChange={setRowSelection}
        isLoading={isLoading}
      />
    </div>
  );
}
```

## Key Rules

1. **Server-side everything** -- sorting, filtering, pagination are query params sent to API. React never sorts or filters client-side.
2. **manualSorting + manualPagination** -- always `true` in `useReactTable`. TanStack Table is headless; the API does the work.
3. **keepPreviousData** -- use `placeholderData: keepPreviousData` in React Query to prevent layout flicker when changing pages.
4. **Skeleton on first load** -- shimmer rows matching the page size. Never a full-page spinner.
5. **Responsive fallback** -- `hidden md:block` for table, `md:hidden` for card list. No horizontal scroll on mobile.
6. **Column definitions are separate** -- one file per entity's columns. Keeps the page component clean.
7. **Row actions in a dropdown** -- MoreHorizontal icon opens a menu. Destructive actions show a confirmation dialog.
8. **Selection state lifted** -- `rowSelection` lives in the page component so bulk actions can use it.
9. **Typed columns** -- `ColumnDef<User>` ensures type safety. Every `row.original` is typed.
10. **Empty state is a component** -- never just "No data". Illustration + message + optional action button.
