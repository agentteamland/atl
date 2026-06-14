---
knowledge-base-summary: "Modal, drawer (side panel), sheet, toast notifications (sonner/react-hot-toast), confirmation dialog. When to use which. Accessible: focus trap, keyboard navigation, ESC to close. Headless UI or Radix primitives."
---
# Modal & Dialog Patterns

## Philosophy

Overlays interrupt the user's flow. Use them intentionally. A modal demands attention (create/edit forms). A drawer shows detail without leaving the page. A sheet is a mobile-native bottom overlay. A toast is transient feedback. A confirmation dialog prevents destructive mistakes. Every overlay must be keyboard-accessible: ESC to close, focus trap inside, return focus on close.

## Dependencies

```json
{
  "@radix-ui/react-dialog": "^1.x",
  "sonner": "^1.x",
  "clsx": "^2.x"
}
```

Radix primitives are headless -- they handle accessibility (focus trap, aria attributes, portal rendering). We supply the Tailwind styling.

## When to Use Which

| Overlay | Use Case | Example |
|---------|----------|---------|
| **Modal** | Create/edit forms, critical info | "Create User" form, payment confirmation |
| **Drawer** | Detail view, filters, settings | User profile panel, filter sidebar |
| **Sheet** | Mobile-only bottom overlay | Mobile action menu, mobile filters |
| **Toast** | Transient feedback (auto-dismiss) | "Saved successfully", "Link copied" |
| **Confirmation** | Destructive or irreversible action | "Delete this user?", "Discard changes?" |

**Decision rule:** If the user must respond before continuing, use modal or confirmation. If the content is supplementary, use drawer. If it is feedback only, use toast.

## Modal Component

```typescript
// components/ui/modal.tsx
import * as Dialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import clsx from 'clsx';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeMap = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
};

export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  size = 'md',
}: ModalProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <Dialog.Portal>
        {/* Backdrop */}
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        {/* Content */}
        <Dialog.Content
          className={clsx(
            'fixed left-1/2 top-1/2 z-50 w-full -translate-x-1/2 -translate-y-1/2',
            'rounded-lg bg-white shadow-xl',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            'data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%]',
            'data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%]',
            sizeMap[size]
          )}
        >
          {/* Header */}
          <div className="flex items-center justify-between border-b px-6 py-4">
            <div>
              <Dialog.Title className="text-lg font-semibold text-gray-900">
                {title}
              </Dialog.Title>
              {description && (
                <Dialog.Description className="mt-1 text-sm text-gray-500">
                  {description}
                </Dialog.Description>
              )}
            </div>
            <Dialog.Close asChild>
              <button
                className="rounded-md p-1 text-gray-400 hover:text-gray-600 hover:bg-gray-100"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          {/* Body */}
          <div className="px-6 py-4">{children}</div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

// Footer helper -- placed inside Modal children
export function ModalFooter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex justify-end gap-3 border-t px-6 py-4 -mx-6 -mb-4 mt-4">
      {children}
    </div>
  );
}
```

## Modal Usage

```typescript
// features/users/create-user-modal.tsx
import { Modal, ModalFooter } from '@/components/ui/modal';
import { useCreateUser } from './use-create-user';

interface CreateUserModalProps {
  open: boolean;
  onClose: () => void;
}

export function CreateUserModal({ open, onClose }: CreateUserModalProps) {
  const createUser = useCreateUser();

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    createUser.mutate(
      { name: formData.get('name') as string, email: formData.get('email') as string },
      { onSuccess: () => onClose() }
    );
  };

  return (
    <Modal open={open} onClose={onClose} title="Create User" size="md">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">Name</label>
          <input name="name" required className="mt-1 w-full rounded-md border px-3 py-2 text-sm" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700">Email</label>
          <input name="email" type="email" required className="mt-1 w-full rounded-md border px-3 py-2 text-sm" />
        </div>
        <ModalFooter>
          <button type="button" onClick={onClose} className="rounded-md border px-4 py-2 text-sm hover:bg-gray-50">
            Cancel
          </button>
          <button
            type="submit"
            disabled={createUser.isPending}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {createUser.isPending ? 'Creating...' : 'Create'}
          </button>
        </ModalFooter>
      </form>
    </Modal>
  );
}
```

## Drawer Component

Slides in from the right. Used for detail views, filter panels, settings.

```typescript
// components/ui/drawer.tsx
import * as Dialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';
import clsx from 'clsx';

interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  side?: 'right' | 'left';
  width?: 'sm' | 'md' | 'lg';
}

const widthMap = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
};

export function Drawer({
  open,
  onClose,
  title,
  children,
  side = 'right',
  width = 'md',
}: DrawerProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        <Dialog.Content
          className={clsx(
            'fixed top-0 z-50 h-full w-full bg-white shadow-xl',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'duration-300',
            widthMap[width],
            side === 'right' && 'right-0 data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right',
            side === 'left' && 'left-0 data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left'
          )}
        >
          {/* Header */}
          <div className="flex items-center justify-between border-b px-6 py-4">
            <Dialog.Title className="text-lg font-semibold text-gray-900">
              {title}
            </Dialog.Title>
            <Dialog.Close asChild>
              <button className="rounded-md p-1 text-gray-400 hover:text-gray-600" aria-label="Close">
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          {/* Scrollable body */}
          <div className="overflow-y-auto p-6" style={{ height: 'calc(100% - 65px)' }}>
            {children}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
```

## Drawer Usage

```typescript
// features/users/user-detail-drawer.tsx
import { Drawer } from '@/components/ui/drawer';
import { useUser } from './use-user';

interface UserDetailDrawerProps {
  userId: string | null;
  onClose: () => void;
}

export function UserDetailDrawer({ userId, onClose }: UserDetailDrawerProps) {
  const { data: user, isLoading } = useUser(userId!, { enabled: !!userId });

  return (
    <Drawer open={!!userId} onClose={onClose} title="User Details" width="md">
      {isLoading ? (
        <div className="space-y-4 animate-pulse">
          <div className="h-16 w-16 rounded-full bg-gray-100" />
          <div className="h-4 w-1/2 bg-gray-100 rounded" />
          <div className="h-4 w-3/4 bg-gray-100 rounded" />
        </div>
      ) : user ? (
        <div className="space-y-6">
          <div className="flex items-center gap-4">
            <div className="h-16 w-16 rounded-full bg-gray-200 flex items-center justify-center text-xl font-semibold">
              {user.name.charAt(0)}
            </div>
            <div>
              <h2 className="text-lg font-semibold">{user.name}</h2>
              <p className="text-sm text-gray-500">{user.email}</p>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-gray-500">Role</span>
              <p className="font-medium">{user.role}</p>
            </div>
            <div>
              <span className="text-gray-500">Status</span>
              <p className="font-medium">{user.status}</p>
            </div>
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}
```

## Bottom Sheet (Mobile)

```typescript
// components/ui/sheet.tsx
import * as Dialog from '@radix-ui/react-dialog';
import { X } from 'lucide-react';

interface SheetProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export function Sheet({ open, onClose, title, children }: SheetProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-40 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        <Dialog.Content
          className={
            'fixed bottom-0 left-0 right-0 z-50 rounded-t-2xl bg-white shadow-xl ' +
            'data-[state=open]:animate-in data-[state=closed]:animate-out ' +
            'data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom ' +
            'duration-300 max-h-[85vh]'
          }
        >
          {/* Drag handle */}
          <div className="flex justify-center pt-3 pb-2">
            <div className="h-1 w-10 rounded-full bg-gray-300" />
          </div>

          {/* Header */}
          {title && (
            <div className="flex items-center justify-between px-6 pb-3">
              <Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
              <Dialog.Close asChild>
                <button className="rounded-md p-1 text-gray-400 hover:text-gray-600" aria-label="Close">
                  <X className="h-5 w-5" />
                </button>
              </Dialog.Close>
            </div>
          )}

          {/* Body */}
          <div className="overflow-y-auto px-6 pb-6" style={{ maxHeight: 'calc(85vh - 80px)' }}>
            {children}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
```

## Toast Notifications (Sonner)

```typescript
// lib/toast.ts -- re-export with project defaults
import { toast as sonnerToast } from 'sonner';

export const toast = {
  success: (message: string) => sonnerToast.success(message),
  error: (message: string) => sonnerToast.error(message),
  info: (message: string) => sonnerToast.info(message),
  loading: (message: string) => sonnerToast.loading(message),
  promise: <T>(
    promise: Promise<T>,
    opts: { loading: string; success: string; error: string }
  ) => sonnerToast.promise(promise, opts),
};
```

```typescript
// App.tsx or layout -- mount the Toaster once
import { Toaster } from 'sonner';

export function App() {
  return (
    <>
      <RouterProvider router={router} />
      <Toaster
        position="bottom-right"
        richColors
        closeButton
        toastOptions={{
          duration: 4000,
          className: 'text-sm',
        }}
      />
    </>
  );
}
```

```typescript
// Usage in mutations
import { toast } from '@/lib/toast';

export function useDeleteUser() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => api.delete(`/users/${userId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User deleted successfully');
    },
    onError: () => {
      toast.error('Failed to delete user. Please try again.');
    },
  });
}
```

## Confirmation Dialog

```typescript
// components/ui/confirm-dialog.tsx
import * as AlertDialog from '@radix-ui/react-alert-dialog';
import clsx from 'clsx';

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'default' | 'destructive';
  loading?: boolean;
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'default',
  loading = false,
}: ConfirmDialogProps) {
  return (
    <AlertDialog.Root open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />

        <AlertDialog.Content className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg bg-white p-6 shadow-xl data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95">
          <AlertDialog.Title className="text-lg font-semibold text-gray-900">
            {title}
          </AlertDialog.Title>
          <AlertDialog.Description className="mt-2 text-sm text-gray-500">
            {description}
          </AlertDialog.Description>

          <div className="mt-6 flex justify-end gap-3">
            <AlertDialog.Cancel asChild>
              <button className="rounded-md border px-4 py-2 text-sm hover:bg-gray-50">
                {cancelLabel}
              </button>
            </AlertDialog.Cancel>
            <AlertDialog.Action asChild>
              <button
                onClick={onConfirm}
                disabled={loading}
                className={clsx(
                  'rounded-md px-4 py-2 text-sm text-white disabled:opacity-50',
                  variant === 'destructive'
                    ? 'bg-red-600 hover:bg-red-700'
                    : 'bg-blue-600 hover:bg-blue-700'
                )}
              >
                {loading ? 'Processing...' : confirmLabel}
              </button>
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
```

## Confirmation Usage

```typescript
const [showConfirm, setShowConfirm] = useState(false);
const deleteUser = useDeleteUser();

<button onClick={() => setShowConfirm(true)}>Delete</button>

<ConfirmDialog
  open={showConfirm}
  onClose={() => setShowConfirm(false)}
  onConfirm={() => {
    deleteUser.mutate(userId, {
      onSuccess: () => setShowConfirm(false),
    });
  }}
  title="Delete User"
  description="This action cannot be undone. All data associated with this user will be permanently deleted."
  confirmLabel="Delete"
  variant="destructive"
  loading={deleteUser.isPending}
/>
```

## Key Rules

1. **Radix primitives** -- use `@radix-ui/react-dialog` and `@radix-ui/react-alert-dialog`. They handle focus trap, aria attributes, portal rendering, and ESC to close out of the box.
2. **ESC to close** -- every overlay must close on ESC. Radix handles this by default.
3. **Backdrop click closes** -- modals and drawers close when clicking outside. Confirmation dialogs should NOT close on backdrop click (use AlertDialog).
4. **Focus returns** -- when overlay closes, focus returns to the element that triggered it. Radix handles this.
5. **Toast for feedback** -- "Saved", "Deleted", "Copied". Sonner handles auto-dismiss, stacking, and rich styling.
6. **Confirmation for destruction** -- delete, discard, irreversible actions always show a confirmation dialog first.
7. **Drawer for supplementary content** -- detail view, filters, settings. Does not block the main content.
8. **Sheet on mobile only** -- bottom sheet replaces drawer on mobile. Use `md:hidden` for sheet, `hidden md:block` for drawer.
9. **Loading state on confirm** -- disable the confirm button and show "Processing..." while the mutation is pending.
10. **One Toaster mount** -- `<Toaster />` is mounted once at the app root. Never mount multiple Toasters.
