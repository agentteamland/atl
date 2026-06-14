---
knowledge-base-summary: "Lightweight ephemeral signals: \"Alice is typing...\", \"3 users viewing this page\". Not persisted to DB — fire-and-forget through Socket. Useful for chat, live support, collaborative features."
---
# Presence & Typing Indicators

## Core Principle

Presence signals are **ephemeral** -- they are NEVER persisted to a database. They exist only in-memory and in Redis with short TTLs. When the Socket process restarts, all presence data is gone, and that is by design. Clients must tolerate stale presence gracefully.

## Two Signal Types

| Signal | Purpose | Scope | Lifetime |
|--------|---------|-------|----------|
| Typing indicator | "Alice is typing..." | Per conversation/room | 3 seconds (auto-expire) |
| Viewer count | "3 users viewing this page" | Per page/resource | Until disconnect |

---

## 1. Typing Indicators

### How It Works

1. Client starts typing in a chat/conversation
2. Client calls `SendTyping(conversationId)` hub method
3. Hub broadcasts `user-typing` event to the conversation group (excluding sender)
4. Client shows "User is typing..." indicator
5. If no new `SendTyping` signal arrives within 3 seconds, client hides the indicator
6. Client stops typing -- no explicit "stopped typing" event needed, timeout handles it

### Hub Method

```csharp
[Authorize]
public sealed class NotificationHub : Hub
{
    // Typing indicator -- ephemeral, no API call, no persistence
    public async Task SendTyping(string conversationId)
    {
        var userId = Context.UserIdentifier!;
        var connectionId = Context.ConnectionId;

        // Broadcast to conversation group, excluding the sender
        await Clients
            .GroupExcept($"conversation:{conversationId}", connectionId)
            .SendAsync("user-typing", new
            {
                UserId = userId,
                ConversationId = conversationId,
                Timestamp = DateTimeOffset.UtcNow,
            });
    }
}
```

### Why No "Stopped Typing" Event?

Sending an explicit "stopped typing" event creates two problems:
1. If the client crashes or disconnects, the "stopped typing" event never fires -- other clients see "typing..." forever
2. Debouncing "stopped typing" on the client side is error-prone

Instead, the client sets a 3-second timer on receiving `user-typing`. If a new `user-typing` arrives, the timer resets. If it expires, the indicator disappears. This is self-healing -- even if the sender disconnects abruptly, the indicator vanishes after 3 seconds.

### Client-Side Pseudocode (Flutter/Dart)

```dart
// Receiving side
final Map<String, Timer> _typingTimers = {};

hub.on('user-typing', (data) {
  final userId = data['userId'];
  final conversationId = data['conversationId'];

  // Cancel existing timer for this user
  _typingTimers[userId]?.cancel();

  // Show typing indicator
  setState(() => _typingUsers.add(userId));

  // Auto-hide after 3 seconds
  _typingTimers[userId] = Timer(Duration(seconds: 3), () {
    setState(() => _typingUsers.remove(userId));
  });
});

// Sending side -- throttle to max 1 signal per second
void _onTextChanged(String text) {
  if (_lastTypingSignal == null ||
      DateTime.now().difference(_lastTypingSignal!) > Duration(seconds: 1)) {
    hub.invoke('SendTyping', args: [conversationId]);
    _lastTypingSignal = DateTime.now();
  }
}
```

### Throttling on the Sender

The client should NOT send a typing signal on every keystroke. Throttle to **1 signal per second**. The 3-second timeout on the receiver gives enough buffer -- even if the sender skips a second, the indicator stays visible.

---

## 2. Viewer Count (Page Presence)

### How It Works

1. Client navigates to a page (e.g., product detail, auction listing)
2. Client calls `JoinViewers(resourceId)` hub method
3. Hub adds the connection to a group and increments the viewer count
4. Hub broadcasts updated count to the group
5. Client navigates away or disconnects -- `LeaveViewers` or `OnDisconnectedAsync` removes them

### Viewer Tracking with Redis

Viewer count requires a central counter because multiple Socket instances may run behind a load balancer. Redis is the source of truth for the count.

```csharp
public sealed class ViewerTracker
{
    private readonly IConnectionMultiplexer _redis;
    private const string Prefix = "viewers:";

    public ViewerTracker(IConnectionMultiplexer redis)
    {
        _redis = redis;
    }

    // Add a viewer, returns the new count
    public async Task<long> AddViewerAsync(string resourceId, string connectionId)
    {
        var db = _redis.GetDatabase();
        var key = $"{Prefix}{resourceId}";

        // SADD -- set guarantees no duplicates if same connection joins twice
        await db.SetAddAsync(key, connectionId);
        // Set a safety TTL -- if all clients crash, the key self-cleans
        await db.KeyExpireAsync(key, TimeSpan.FromHours(1));

        return await db.SetLengthAsync(key);
    }

    // Remove a viewer, returns the new count
    public async Task<long> RemoveViewerAsync(string resourceId, string connectionId)
    {
        var db = _redis.GetDatabase();
        var key = $"{Prefix}{resourceId}";

        await db.SetRemoveAsync(key, connectionId);
        var count = await db.SetLengthAsync(key);

        // Clean up empty sets
        if (count == 0)
            await db.KeyDeleteAsync(key);

        return count;
    }

    // Get current viewer count
    public async Task<long> GetViewerCountAsync(string resourceId)
    {
        var db = _redis.GetDatabase();
        return await db.SetLengthAsync($"{Prefix}{resourceId}");
    }
}
```

### Hub Methods

```csharp
[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly ViewerTracker _viewerTracker;

    public NotificationHub(ViewerTracker viewerTracker)
    {
        _viewerTracker = viewerTracker;
    }

    public async Task JoinViewers(string resourceId)
    {
        var groupName = $"viewers:{resourceId}";
        await Groups.AddToGroupAsync(Context.ConnectionId, groupName);

        var count = await _viewerTracker.AddViewerAsync(resourceId, Context.ConnectionId);

        // Broadcast updated count to everyone viewing this resource
        await Clients.Group(groupName).SendAsync("viewer-count-updated", new
        {
            ResourceId = resourceId,
            ViewerCount = count,
        });
    }

    public async Task LeaveViewers(string resourceId)
    {
        var groupName = $"viewers:{resourceId}";

        var count = await _viewerTracker.RemoveViewerAsync(resourceId, Context.ConnectionId);
        await Groups.RemoveFromGroupAsync(Context.ConnectionId, groupName);

        // Broadcast updated count to remaining viewers
        await Clients.Group(groupName).SendAsync("viewer-count-updated", new
        {
            ResourceId = resourceId,
            ViewerCount = count,
        });
    }

    public override async Task OnDisconnectedAsync(Exception? exception)
    {
        // Clean up all viewer groups this connection was part of
        // The connection-to-resources mapping is tracked separately
        // (see Connection Tracking children doc)
        await base.OnDisconnectedAsync(exception);
    }
}
```

### Tracking Which Resources a Connection Is Viewing

When a connection disconnects, we need to know which resources it was viewing so we can decrement the counts. Store a reverse mapping in Redis:

```csharp
// When joining:
await db.SetAddAsync($"connection-resources:{connectionId}", resourceId);

// On disconnect -- get all resources and clean up:
var resources = await db.SetMembersAsync($"connection-resources:{connectionId}");
foreach (var resource in resources)
{
    await RemoveViewerAsync(resource.ToString(), connectionId);
    // Also broadcast updated count to the group
}
await db.KeyDeleteAsync($"connection-resources:{connectionId}");
```

---

## Use Cases

| Use Case | Signal Type | Group Key |
|----------|-------------|-----------|
| Chat: "Alice is typing..." | Typing | `conversation:{conversationId}` |
| Live support: "Agent is typing..." | Typing | `ticket:{ticketId}` |
| Product page: "12 people viewing this" | Viewer count | `viewers:product:{productId}` |
| Live auction: "45 watching" | Viewer count | `viewers:auction:{auctionId}` |
| Collaborative doc: "3 editors online" | Viewer count | `viewers:document:{docId}` |

---

## Rules

1. **No database writes.** Presence is ephemeral. Redis with TTL at most.
2. **No API calls.** Typing and viewer counts are handled entirely within the Socket. The API does not need to know who is typing.
3. **Graceful degradation.** If Redis is down, typing still works (broadcast-only, no viewer count). If the Socket restarts, all presence data resets -- clients rebuild on reconnect.
4. **Throttle on the sender.** Typing: max 1 signal/second. Viewer join: once per navigation.
5. **Timeout on the receiver.** Typing indicator: 3-second auto-expire. No stale indicators.
6. **Use connectionId, not userId for viewer sets.** One user can have multiple tabs open -- each tab is a separate viewer.
