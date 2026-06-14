---
knowledge-base-summary: "SignalR client (@microsoft/signalr). Connection setup with JWT token. Event listeners. Auto-reconnection handling. React hook for SignalR (useSignalR). Connection state management."
---
# WebSocket Integration: SignalR Client

## Philosophy

Real-time communication goes through SignalR. The React app connects to the Socket service (not the API directly). The connection is established once, managed centrally, and shared across the app. React Query cache invalidation is the primary action on incoming events -- the API is still the source of truth, SignalR just tells us "something changed, refetch."

## Package

```bash
npm install @microsoft/signalr
```

## Connection Builder

A single factory function creates the HubConnection. JWT token is injected via `accessTokenFactory`. The connection is never created inside a component -- it lives in a service layer.

```typescript
// src/lib/signalr.ts
import {
  HubConnectionBuilder,
  HubConnection,
  HubConnectionState,
  HttpTransportType,
  LogLevel,
} from '@microsoft/signalr';
import { useAuthStore } from '@/stores/auth-store';

const SOCKET_URL = import.meta.env.VITE_SOCKET_URL ?? 'http://localhost:3002';

export function createHubConnection(hubPath: string): HubConnection {
  const connection = new HubConnectionBuilder()
    .withUrl(`${SOCKET_URL}/${hubPath}`, {
      accessTokenFactory: () => {
        const token = useAuthStore.getState().accessToken;
        return token ?? '';
      },
      transport: HttpTransportType.WebSockets,
      skipNegotiation: true,
    })
    .withAutomaticReconnect({
      nextRetryDelayInMilliseconds: (retryContext) => {
        // Exponential backoff: 0s, 2s, 4s, 8s, 16s, then every 30s
        if (retryContext.previousRetryCount < 5) {
          return Math.pow(2, retryContext.previousRetryCount) * 1000;
        }
        return 30_000;
      },
    })
    .configureLogging(
      import.meta.env.DEV ? LogLevel.Information : LogLevel.Warning
    )
    .build();

  // Server timeout: if no message in 60s, connection is considered lost
  connection.serverTimeoutInMilliseconds = 60_000;

  // Keep-alive: ping every 15s
  connection.keepAliveIntervalInMilliseconds = 15_000;

  return connection;
}
```

## Connection State Store (Zustand)

Connection state is global. Components read it to show "connected/disconnected" badges or disable real-time features when offline.

```typescript
// src/stores/signalr-store.ts
import { create } from 'zustand';

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

interface SignalRState {
  status: ConnectionStatus;
  error: string | null;
  setStatus: (status: ConnectionStatus) => void;
  setError: (error: string | null) => void;
}

export const useSignalRStore = create<SignalRState>((set) => ({
  status: 'disconnected',
  error: null,
  setStatus: (status) => set({ status, error: null }),
  setError: (error) => set({ error }),
}));
```

## useSignalR Hook

The core hook. Manages the entire lifecycle: connect, listen, invoke, disconnect. One instance per hub, created at a high level (layout or app root).

```typescript
// src/hooks/use-signalr.ts
import { useEffect, useRef, useCallback } from 'react';
import { HubConnection, HubConnectionState } from '@microsoft/signalr';
import { createHubConnection } from '@/lib/signalr';
import { useSignalRStore } from '@/stores/signalr-store';
import { useAuthStore } from '@/stores/auth-store';

interface UseSignalROptions {
  hubPath: string;
  enabled?: boolean; // don't connect if not authenticated
}

interface UseSignalRReturn {
  connection: HubConnection | null;
  status: string;
  on: (event: string, handler: (...args: any[]) => void) => void;
  off: (event: string, handler: (...args: any[]) => void) => void;
  invoke: <T = void>(method: string, ...args: unknown[]) => Promise<T>;
}

export function useSignalR({ hubPath, enabled = true }: UseSignalROptions): UseSignalRReturn {
  const connectionRef = useRef<HubConnection | null>(null);
  const { setStatus, setError, status } = useSignalRStore();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  // Start connection
  useEffect(() => {
    if (!enabled || !isAuthenticated) return;

    const connection = createHubConnection(hubPath);
    connectionRef.current = connection;

    // Lifecycle events
    connection.onreconnecting((error) => {
      setStatus('reconnecting');
      console.warn('[SignalR] Reconnecting...', error?.message);
    });

    connection.onreconnected((connectionId) => {
      setStatus('connected');
      console.info('[SignalR] Reconnected:', connectionId);
    });

    connection.onclose((error) => {
      setStatus('disconnected');
      if (error) {
        setError(error.message);
        console.error('[SignalR] Connection closed with error:', error);
      }
    });

    // Connect
    setStatus('connecting');
    connection
      .start()
      .then(() => {
        setStatus('connected');
        console.info('[SignalR] Connected to', hubPath);
      })
      .catch((err) => {
        setStatus('disconnected');
        setError(err.message);
        console.error('[SignalR] Connection failed:', err);
      });

    // Cleanup on unmount or logout
    return () => {
      if (connection.state !== HubConnectionState.Disconnected) {
        connection.stop();
      }
      connectionRef.current = null;
      setStatus('disconnected');
    };
  }, [hubPath, enabled, isAuthenticated, setStatus, setError]);

  // Subscribe to an event
  const on = useCallback((event: string, handler: (...args: any[]) => void) => {
    connectionRef.current?.on(event, handler);
  }, []);

  // Unsubscribe from an event
  const off = useCallback((event: string, handler: (...args: any[]) => void) => {
    connectionRef.current?.off(event, handler);
  }, []);

  // Invoke a server method
  const invoke = useCallback(async <T = void>(method: string, ...args: unknown[]): Promise<T> => {
    if (!connectionRef.current || connectionRef.current.state !== HubConnectionState.Connected) {
      throw new Error('SignalR not connected');
    }
    return connectionRef.current.invoke<T>(method, ...args);
  }, []);

  return {
    connection: connectionRef.current,
    status,
    on,
    off,
    invoke,
  };
}
```

## React Query Cache Invalidation on Real-Time Events

The most common pattern: SignalR event arrives, we invalidate the relevant React Query cache so it refetches from the API. We never trust the WebSocket payload as the final data -- we use it as a signal to refetch.

```typescript
// src/hooks/use-realtime-invalidation.ts
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useSignalR } from './use-signalr';

interface InvalidationRule {
  event: string;
  queryKeys: string[][];
}

/**
 * Listens to SignalR events and invalidates React Query cache.
 * 
 * Usage:
 *   useRealtimeInvalidation([
 *     { event: 'OrderCreated', queryKeys: [['orders'], ['dashboard-stats']] },
 *     { event: 'OrderStatusChanged', queryKeys: [['orders']] },
 *   ]);
 */
export function useRealtimeInvalidation(rules: InvalidationRule[]) {
  const { on, off, status } = useSignalR({ hubPath: 'notifications' });
  const queryClient = useQueryClient();

  useEffect(() => {
    if (status !== 'connected') return;

    const handlers = rules.map((rule) => {
      const handler = () => {
        rule.queryKeys.forEach((key) => {
          queryClient.invalidateQueries({ queryKey: key });
        });
      };
      on(rule.event, handler);
      return { event: rule.event, handler };
    });

    return () => {
      handlers.forEach(({ event, handler }) => off(event, handler));
    };
  }, [status, rules, on, off, queryClient]);
}
```

## Typed Event Listener Hook

For cases where you need the event payload directly (e.g., toast notification, live counter update):

```typescript
// src/hooks/use-signalr-event.ts
import { useEffect } from 'react';
import { useSignalR } from './use-signalr';

/**
 * Subscribe to a specific SignalR event with a typed handler.
 * Automatically cleans up on unmount.
 *
 * Usage:
 *   useSignalREvent<{ orderId: string; status: string }>(
 *     'OrderStatusChanged',
 *     (data) => toast.info(`Order ${data.orderId} is now ${data.status}`)
 *   );
 */
export function useSignalREvent<T>(event: string, handler: (data: T) => void) {
  const { on, off, status } = useSignalR({ hubPath: 'notifications' });

  useEffect(() => {
    if (status !== 'connected') return;

    const wrappedHandler = (data: T) => handler(data);
    on(event, wrappedHandler);

    return () => off(event, wrappedHandler);
  }, [event, handler, on, off, status]);
}
```

## App-Level Setup

The SignalR connection is established at the app layout level, not in individual pages. This ensures a single connection per session.

```typescript
// src/layouts/app-layout.tsx
import { useSignalR } from '@/hooks/use-signalr';
import { ConnectionBadge } from '@/components/connection-badge';

export function AppLayout() {
  const { status } = useSignalR({
    hubPath: 'notifications',
    enabled: true,
  });

  return (
    <div className="flex h-screen">
      <Sidebar />
      <main className="flex-1 overflow-auto">
        <Header>
          <ConnectionBadge status={status} />
        </Header>
        <Outlet />
      </main>
    </div>
  );
}
```

## Connection Status Badge

Visual indicator for connection state:

```typescript
// src/components/connection-badge.tsx
const statusConfig = {
  connected: { label: 'Live', className: 'bg-green-500' },
  connecting: { label: 'Connecting...', className: 'bg-yellow-500 animate-pulse' },
  reconnecting: { label: 'Reconnecting...', className: 'bg-yellow-500 animate-pulse' },
  disconnected: { label: 'Offline', className: 'bg-red-500' },
} as const;

interface ConnectionBadgeProps {
  status: keyof typeof statusConfig;
}

export function ConnectionBadge({ status }: ConnectionBadgeProps) {
  const config = statusConfig[status];

  return (
    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <span className={`h-2 w-2 rounded-full ${config.className}`} />
      {config.label}
    </div>
  );
}
```

## Page-Level Usage Example

```typescript
// src/features/orders/pages/orders-page.tsx
import { useRealtimeInvalidation } from '@/hooks/use-realtime-invalidation';
import { useSignalREvent } from '@/hooks/use-signalr-event';
import { toast } from 'sonner';

export function OrdersPage() {
  // Auto-refetch orders list when new order arrives
  useRealtimeInvalidation([
    { event: 'OrderCreated', queryKeys: [['orders']] },
    { event: 'OrderStatusChanged', queryKeys: [['orders']] },
  ]);

  // Show toast on new order (using the event payload)
  useSignalREvent<{ orderId: string; customerName: string }>(
    'OrderCreated',
    (data) => toast.success(`New order from ${data.customerName}`)
  );

  // ... rest of the page
}
```

## Disconnect on Logout

When the user logs out, the SignalR connection must be closed. This is handled by the `useSignalR` hook's dependency on `isAuthenticated` -- when it becomes false, the cleanup function runs and stops the connection.

```typescript
// src/stores/auth-store.ts (logout action)
logout: () => {
  set({ accessToken: null, user: null, isAuthenticated: false });
  // useSignalR's useEffect cleanup will automatically disconnect
  // because isAuthenticated changed to false
},
```

## Rules

1. **One connection per hub.** Don't create multiple connections to the same hub. The `useSignalR` hook should be called once per hub at a layout level.
2. **Never trust WebSocket data as source of truth.** Use it as an invalidation signal. Refetch from the API via React Query.
3. **Type your events.** Create interfaces for every event payload. No `any`.
4. **Clean up listeners.** Every `on()` must have a corresponding `off()` in the cleanup function. The hooks handle this automatically.
5. **Don't connect if not authenticated.** Use the `enabled` flag tied to auth state.
6. **Exponential backoff for reconnection.** Don't hammer the server with instant retries.
