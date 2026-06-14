---
knowledge-base-summary: "Ctrl+S (save), Ctrl+K (command palette), Ctrl+Z (undo). Hotkey registration and management. Clipboard API (copy to clipboard). Key combination display in UI. Accessible: doesn't conflict with screen readers."
---
# Keyboard Shortcuts: Hotkeys & Command Palette

## Philosophy

Keyboard shortcuts are power-user accelerators. Every shortcut must have a mouse/touch alternative -- shortcuts are never the only way to do something. We never override browser-native shortcuts (Ctrl+T, Ctrl+W, Ctrl+L, Ctrl+N). The shortcut registry is centralized so we can display them in the UI and avoid conflicts.

## Package

```bash
npm install react-hotkeys-hook
```

Lightweight, hook-based, handles modifier keys and scope management out of the box.

## Shortcut Registry

Central definition of all shortcuts. Single source of truth for key bindings, descriptions, and categories. Used by the command palette, tooltips, and documentation.

```typescript
// src/config/shortcuts.ts

export interface ShortcutDef {
  id: string;
  keys: string;           // hotkey string: 'ctrl+s', 'ctrl+k', 'escape'
  label: string;          // human-readable description
  category: ShortcutCategory;
  global?: boolean;       // true = works everywhere, false = scoped to a component
}

export type ShortcutCategory = 'general' | 'navigation' | 'editing' | 'table';

export const SHORTCUTS: Record<string, ShortcutDef> = {
  save: {
    id: 'save',
    keys: 'ctrl+s',
    label: 'Save',
    category: 'editing',
    global: true,
  },
  commandPalette: {
    id: 'commandPalette',
    keys: 'ctrl+k',
    label: 'Command Palette',
    category: 'general',
    global: true,
  },
  search: {
    id: 'search',
    keys: 'ctrl+shift+f',
    label: 'Global Search',
    category: 'general',
    global: true,
  },
  closeModal: {
    id: 'closeModal',
    keys: 'escape',
    label: 'Close',
    category: 'general',
    global: true,
  },
  undo: {
    id: 'undo',
    keys: 'ctrl+z',
    label: 'Undo',
    category: 'editing',
  },
  newItem: {
    id: 'newItem',
    keys: 'ctrl+shift+n',
    label: 'New Item',
    category: 'editing',
  },
  nextPage: {
    id: 'nextPage',
    keys: 'alt+right',
    label: 'Next Page',
    category: 'navigation',
  },
  prevPage: {
    id: 'prevPage',
    keys: 'alt+left',
    label: 'Previous Page',
    category: 'navigation',
  },
  copyId: {
    id: 'copyId',
    keys: 'ctrl+shift+c',
    label: 'Copy ID',
    category: 'table',
  },
} as const;

/**
 * Format a key string for display. 'ctrl+s' → 'Ctrl + S'
 */
export function formatShortcut(keys: string): string {
  return keys
    .split('+')
    .map((key) => {
      if (key === 'ctrl') return navigator.platform.includes('Mac') ? 'Cmd' : 'Ctrl';
      if (key === 'alt') return navigator.platform.includes('Mac') ? 'Option' : 'Alt';
      if (key === 'shift') return 'Shift';
      if (key === 'escape') return 'Esc';
      return key.charAt(0).toUpperCase() + key.slice(1);
    })
    .join(' + ');
}

/**
 * Get shortcuts grouped by category (for help dialog / command palette).
 */
export function getShortcutsByCategory(): Record<ShortcutCategory, ShortcutDef[]> {
  const grouped: Record<ShortcutCategory, ShortcutDef[]> = {
    general: [],
    navigation: [],
    editing: [],
    table: [],
  };

  Object.values(SHORTCUTS).forEach((shortcut) => {
    grouped[shortcut.category].push(shortcut);
  });

  return grouped;
}
```

## useAppHotkey Hook

Wraps `react-hotkeys-hook` with our conventions: prevents browser default, integrates with registry.

```typescript
// src/hooks/use-app-hotkey.ts
import { useHotkeys, type Options } from 'react-hotkeys-hook';
import { SHORTCUTS } from '@/config/shortcuts';

/**
 * Register a keyboard shortcut using a registry key.
 * Automatically prevents default browser behavior.
 *
 * Usage:
 *   useAppHotkey('save', () => handleSave());
 *   useAppHotkey('commandPalette', () => setOpen(true));
 */
export function useAppHotkey(
  shortcutId: keyof typeof SHORTCUTS,
  handler: (e: KeyboardEvent) => void,
  options?: Omit<Options, 'preventDefault'>
) {
  const shortcut = SHORTCUTS[shortcutId];

  useHotkeys(
    shortcut.keys,
    (e) => {
      e.preventDefault();
      handler(e);
    },
    {
      ...options,
      preventDefault: true,
      enableOnFormTags: shortcut.id === 'closeModal' || shortcut.id === 'save',
    }
  );
}
```

## Shortcut Hint Component

Show the keyboard shortcut next to a button or in a tooltip.

```typescript
// src/components/shortcut-hint.tsx
import { formatShortcut } from '@/config/shortcuts';

interface ShortcutHintProps {
  keys: string;
  className?: string;
}

/**
 * Inline keyboard shortcut badge.
 *
 * Usage:
 *   <Button>
 *     Save <ShortcutHint keys="ctrl+s" />
 *   </Button>
 */
export function ShortcutHint({ keys, className = '' }: ShortcutHintProps) {
  return (
    <kbd
      className={`ml-2 inline-flex items-center gap-0.5 rounded border bg-muted px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground ${className}`}
    >
      {formatShortcut(keys)}
    </kbd>
  );
}
```

### Usage in Buttons and Tooltips

```typescript
import { SHORTCUTS } from '@/config/shortcuts';

// In a button
<Button onClick={handleSave}>
  Save
  <ShortcutHint keys={SHORTCUTS.save.keys} />
</Button>

// In a tooltip (with Radix or similar)
<Tooltip content={`Search ${formatShortcut(SHORTCUTS.search.keys)}`}>
  <Button variant="ghost" onClick={openSearch}>
    <SearchIcon className="h-4 w-4" />
  </Button>
</Tooltip>
```

## Command Palette

A search overlay triggered by Ctrl+K. Lists available actions, navigable by keyboard (arrow keys + Enter).

```typescript
// src/components/command-palette.tsx
import { useState, useEffect, useRef, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAppHotkey } from '@/hooks/use-app-hotkey';
import { SHORTCUTS, formatShortcut, getShortcutsByCategory } from '@/config/shortcuts';

interface CommandItem {
  id: string;
  label: string;
  shortcut?: string;
  action: () => void;
  category: string;
}

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  // Open/close
  useAppHotkey('commandPalette', () => setOpen((prev) => !prev));
  useAppHotkey('closeModal', () => setOpen(false));

  // Build command list
  const commands = useMemo<CommandItem[]>(() => [
    { id: 'nav-dashboard', label: 'Go to Dashboard', action: () => navigate('/dashboard'), category: 'Navigation' },
    { id: 'nav-orders', label: 'Go to Orders', action: () => navigate('/orders'), category: 'Navigation' },
    { id: 'nav-products', label: 'Go to Products', action: () => navigate('/products'), category: 'Navigation' },
    { id: 'nav-users', label: 'Go to Users', action: () => navigate('/users'), category: 'Navigation' },
    { id: 'nav-settings', label: 'Go to Settings', action: () => navigate('/settings'), category: 'Navigation' },
    { id: 'act-new-order', label: 'Create New Order', shortcut: SHORTCUTS.newItem.keys, action: () => navigate('/orders/new'), category: 'Actions' },
    // Add more commands as the app grows
  ], [navigate]);

  // Filter by search query
  const filtered = useMemo(() => {
    if (!query) return commands;
    const lower = query.toLowerCase();
    return commands.filter((cmd) => cmd.label.toLowerCase().includes(lower));
  }, [query, commands]);

  // Reset selection when filtered list changes
  useEffect(() => setSelectedIndex(0), [filtered]);

  // Focus input when opened
  useEffect(() => {
    if (open) {
      setQuery('');
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [open]);

  // Keyboard navigation
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setSelectedIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter' && filtered[selectedIndex]) {
      filtered[selectedIndex].action();
      setOpen(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={() => setOpen(false)} />

      {/* Palette */}
      <div className="relative w-full max-w-lg rounded-xl border bg-card shadow-2xl">
        <input
          ref={inputRef}
          type="text"
          placeholder="Type a command..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          className="w-full rounded-t-xl border-b bg-transparent px-4 py-3 text-sm outline-none placeholder:text-muted-foreground"
        />

        <ul className="max-h-72 overflow-auto p-2">
          {filtered.map((cmd, index) => (
            <li
              key={cmd.id}
              onClick={() => { cmd.action(); setOpen(false); }}
              className={`flex cursor-pointer items-center justify-between rounded-md px-3 py-2 text-sm ${
                index === selectedIndex ? 'bg-primary/10 text-primary' : 'text-foreground'
              }`}
            >
              <span>{cmd.label}</span>
              {cmd.shortcut && (
                <kbd className="rounded border bg-muted px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground">
                  {formatShortcut(cmd.shortcut)}
                </kbd>
              )}
            </li>
          ))}

          {filtered.length === 0 && (
            <li className="px-3 py-6 text-center text-sm text-muted-foreground">
              No commands found.
            </li>
          )}
        </ul>
      </div>
    </div>
  );
}
```

## Clipboard API: Copy to Clipboard

```typescript
// src/lib/clipboard.ts
import { toast } from 'sonner';

/**
 * Copy text to clipboard with success/error toast.
 *
 * Usage:
 *   copyToClipboard(orderId, 'Order ID copied');
 */
export async function copyToClipboard(text: string, successMessage = 'Copied to clipboard') {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(successMessage);
  } catch {
    // Fallback for older browsers or insecure contexts
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
    toast.success(successMessage);
  }
}
```

### Usage with Shortcut

```typescript
// In a table row action
import { copyToClipboard } from '@/lib/clipboard';
import { useAppHotkey } from '@/hooks/use-app-hotkey';

function OrderRow({ order }: { order: Order }) {
  useAppHotkey('copyId', () => copyToClipboard(order.id, 'Order ID copied'));

  return (
    <tr>
      <td>
        <button onClick={() => copyToClipboard(order.id, 'Order ID copied')}>
          {order.id}
        </button>
      </td>
      {/* ... */}
    </tr>
  );
}
```

## Keyboard Shortcuts Help Dialog

Shows all registered shortcuts grouped by category. Triggered by `?` key or from help menu.

```typescript
// src/components/shortcuts-help.tsx
import { useState } from 'react';
import { getShortcutsByCategory, formatShortcut } from '@/config/shortcuts';
import { useHotkeys } from 'react-hotkeys-hook';

const categoryLabels: Record<string, string> = {
  general: 'General',
  navigation: 'Navigation',
  editing: 'Editing',
  table: 'Table',
};

export function ShortcutsHelp() {
  const [open, setOpen] = useState(false);
  const groups = getShortcutsByCategory();

  useHotkeys('shift+/', () => setOpen(true)); // '?' key

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={() => setOpen(false)} />
      <div className="relative w-full max-w-md rounded-xl border bg-card p-6 shadow-2xl">
        <h2 className="mb-4 text-lg font-semibold">Keyboard Shortcuts</h2>

        {Object.entries(groups).map(([category, shortcuts]) =>
          shortcuts.length > 0 ? (
            <div key={category} className="mb-4">
              <h3 className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {categoryLabels[category]}
              </h3>
              <div className="space-y-1">
                {shortcuts.map((s) => (
                  <div key={s.id} className="flex items-center justify-between py-1">
                    <span className="text-sm">{s.label}</span>
                    <kbd className="rounded border bg-muted px-2 py-0.5 text-xs font-mono text-muted-foreground">
                      {formatShortcut(s.keys)}
                    </kbd>
                  </div>
                ))}
              </div>
            </div>
          ) : null
        )}

        <button
          onClick={() => setOpen(false)}
          className="mt-2 w-full rounded-md bg-muted py-2 text-sm text-muted-foreground hover:bg-muted/80"
        >
          Close (Esc)
        </button>
      </div>
    </div>
  );
}
```

## Browser Shortcut Conflict Avoidance

These shortcuts are reserved by browsers and must NEVER be overridden:

| Shortcut | Browser Action |
|----------|---------------|
| Ctrl+T | New tab |
| Ctrl+W | Close tab |
| Ctrl+N | New window |
| Ctrl+L | Focus address bar |
| Ctrl+Q | Quit browser |
| Ctrl+Tab | Switch tab |
| Ctrl+Shift+T | Reopen closed tab |
| F5 | Refresh |
| F11 | Fullscreen |
| F12 | DevTools |

Safe to use: `Ctrl+S`, `Ctrl+K`, `Ctrl+Z`, `Ctrl+Shift+F`, `Ctrl+Shift+N`, `Alt+Arrow`, `Escape`.

## Rules

1. **Every shortcut has a mouse alternative.** A shortcut is an accelerator, never the only way to trigger an action.
2. **Never override browser defaults.** See the conflict table above.
3. **Centralized registry.** All shortcuts defined in `src/config/shortcuts.ts`. No inline hotkey strings scattered across components.
4. **Show shortcuts in UI.** Buttons should display their shortcut via `ShortcutHint`. Tooltips should mention shortcuts.
5. **Scope matters.** Global shortcuts (save, command palette) work everywhere. Component-scoped shortcuts (table row copy) only work when that component is focused.
6. **Mac awareness.** Display "Cmd" instead of "Ctrl" on macOS. The `formatShortcut` utility handles this.
7. **Accessibility.** Shortcuts must not conflict with screen reader controls. Always provide non-keyboard alternatives.
