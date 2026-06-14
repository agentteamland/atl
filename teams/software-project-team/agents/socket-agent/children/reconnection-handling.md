---
knowledge-base-summary: "Mobile apps lose connection frequently. Client reconnects but missed events during downtime. Strategies: missed events queue (Redis), \"fetch last N events\" endpoint, or client-side last-event-id tracking."
---
# Reconnection Handling

Mobile clients lose connection constantly -- tunnel, elevator, airplane mode, app backgrounded by OS. The critical question: what events did the user miss while offline?

## The Problem

1. User is connected, receives live events.
2. Connection drops (network switch, sleep, etc.).
3. Events fire while user is offline. They are lost.
4. User reconnects. Their UI is stale.

SignalR does NOT replay missed events. Once a connection drops, any event sent during downtime is gone. The client must have a strategy to recover.

## Strategy 1: REST Catch-Up (Recommended Default)

The simplest and most reliable approach. Socket is not responsible for recovery -- the client calls REST after reconnecting to fetch what it missed.

### How It Works

1. Client connects via SignalR.
2. Client stores a local timestamp (`lastSyncedAt`) every time it processes an event.
3. Connection drops.
4. Client reconnects to SignalR.
5. Immediately after reconnection, client calls a REST endpoint: `GET /api/notifications?since={lastSyncedAt}`.
6. API returns all missed items. Client merges them into local state.

### Socket Side (No Changes Needed)

The Socket does nothing special. It broadcasts as usual. Recovery is entirely client + API responsibility.

```csharp
// NotificationHub.cs -- no reconnection logic needed
[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly ILogger<NotificationHub> _logger;

    public NotificationHub(ILogger<NotificationHub> logger)
    {
        _logger = logger;
    }

    public override async Task OnConnectedAsync()
    {
        var userId = Context.UserIdentifier;
        _logger.LogInformation("User {UserId} connected. Client should fetch missed events via REST", userId);
        await base.OnConnectedAsync();
    }
}
```

### API Side (Endpoint)

The API exposes a catch-up endpoint. This is not the Socket agent's responsibility, but here is the contract the client expects:

```
GET /api/notifications?since=2026-04-14T10:30:00Z&limit=50

Response:
{
    "items": [ ... ],
    "hasMore": false,
    "syncedAt": "2026-04-14T11:00:00Z"
}
```

### Client Side (Flutter/React Pseudo-Code)

```
onReconnected:
    missedItems = await GET /api/notifications?since={lastSyncedAt}
    merge(missedItems)
    lastSyncedAt = now()
```

### Why This Is the Default

- Zero Socket complexity. Socket stays a thin bridge.
- REST endpoints already exist (or should). No new infrastructure.
- Works for all event types -- notifications, orders, messages.
- Paginated. Client can fetch 50 at a time if thousands were missed.
- Idempotent. Client can call it multiple times safely.

---

## Strategy 2: Redis Missed Events Queue

For chat-like applications where message ordering and completeness are critical. Socket maintains a per-user event buffer in Redis.

### How It Works

1. Every event sent to a user is also written to a Redis list: `missed:{userId}`.
2. When the user has an active connection, events are sent via SignalR AND written to Redis (with TTL).
3. On reconnect, Socket reads the Redis list and replays buffered events.
4. After replay, the list is trimmed.

### Implementation

```csharp
// Services/MissedEventStore.cs
using StackExchange.Redis;
using System.Text.Json;

namespace ExampleApp.Socket.Services;

public interface IMissedEventStore
{
    Task BufferEventAsync(string userId, string eventName, object payload);
    Task<IReadOnlyList<BufferedEvent>> FlushEventsAsync(string userId, string? lastEventId = null);
}

public sealed record BufferedEvent(
    string EventId,
    string EventName,
    string PayloadJson,
    DateTimeOffset Timestamp
);

public sealed class RedisMissedEventStore : IMissedEventStore
{
    private readonly IDatabase _redis;
    private const int MaxEventsPerUser = 200;
    private static readonly TimeSpan BufferTtl = TimeSpan.FromHours(24);

    public RedisMissedEventStore(IConnectionMultiplexer redis)
    {
        _redis = redis.GetDatabase();
    }

    public async Task BufferEventAsync(string userId, string eventName, object payload)
    {
        var key = $"missed:{userId}";
        var buffered = new BufferedEvent(
            EventId: Guid.NewGuid().ToString("N"),
            EventName: eventName,
            PayloadJson: JsonSerializer.Serialize(payload),
            Timestamp: DateTimeOffset.UtcNow
        );

        var json = JsonSerializer.Serialize(buffered);

        await _redis.ListRightPushAsync(key, json);
        await _redis.ListTrimAsync(key, -MaxEventsPerUser, -1); // Keep last N
        await _redis.KeyExpireAsync(key, BufferTtl);
    }

    public async Task<IReadOnlyList<BufferedEvent>> FlushEventsAsync(
        string userId, string? lastEventId = null)
    {
        var key = $"missed:{userId}";
        var values = await _redis.ListRangeAsync(key);

        var events = values
            .Select(v => JsonSerializer.Deserialize<BufferedEvent>(v.ToString())!)
            .ToList();

        // If client sends lastEventId, skip everything up to and including that event
        if (!string.IsNullOrEmpty(lastEventId))
        {
            var index = events.FindIndex(e => e.EventId == lastEventId);
            if (index >= 0)
            {
                events = events.Skip(index + 1).ToList();
            }
        }

        // Clear the buffer after flush
        await _redis.KeyDeleteAsync(key);

        return events;
    }
}
```

### Hub Integration

```csharp
// NotificationHub.cs with replay support
[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly ILogger<NotificationHub> _logger;
    private readonly IMissedEventStore _missedEvents;

    public NotificationHub(
        ILogger<NotificationHub> logger,
        IMissedEventStore missedEvents)
    {
        _logger = logger;
        _missedEvents = missedEvents;
    }

    public override async Task OnConnectedAsync()
    {
        var userId = Context.UserIdentifier!;
        _logger.LogInformation("User {UserId} connected, replaying missed events", userId);

        // Replay missed events
        var lastEventId = Context.GetHttpContext()?.Request.Query["lastEventId"].ToString();
        var missed = await _missedEvents.FlushEventsAsync(userId, lastEventId);

        foreach (var evt in missed)
        {
            await Clients.Caller.SendAsync(evt.EventName, evt.PayloadJson);
        }

        _logger.LogInformation("Replayed {Count} missed events for User {UserId}",
            missed.Count, userId);

        await base.OnConnectedAsync();
    }
}
```

### When to Use This

- Chat applications where every message must be delivered.
- Collaborative editing where event order matters.
- Auction / bidding systems where missed bids are critical.

### Trade-Offs

- **Adds Redis dependency** to Socket (may already exist for connection tracking).
- **Memory pressure** -- 200 events x 10K users = 2M entries in Redis.
- **Ordering** -- Redis lists preserve insertion order, but clock skew between Socket instances can cause minor reordering.
- **Duplication** -- Client may receive an event both live AND in the replay. Client must deduplicate by eventId.

---

## Strategy 3: Client-Side Last-Event-ID Tracking

The client tracks the last event it successfully processed and sends it on reconnect. The server replays from that point.

### How It Works

1. Every event includes an `eventId` field.
2. Client stores `lastProcessedEventId` locally (SharedPreferences, localStorage).
3. On reconnect, client sends `lastEventId` as a query parameter: `?access_token=X&lastEventId=abc123`.
4. Socket reads from persistent store (Redis or DB) and replays events after that ID.

This is essentially Strategy 2 with the cursor driven by the client.

```csharp
// Client connects with lastEventId
// ws://socket:3002/hubs/notifications?access_token=JWT&lastEventId=abc123

public override async Task OnConnectedAsync()
{
    var userId = Context.UserIdentifier!;
    var lastEventId = Context.GetHttpContext()?.Request.Query["lastEventId"].ToString();

    if (!string.IsNullOrEmpty(lastEventId))
    {
        _logger.LogInformation(
            "User {UserId} reconnecting with lastEventId {LastEventId}",
            userId, lastEventId);

        var missed = await _missedEvents.FlushEventsAsync(userId, lastEventId);
        foreach (var evt in missed)
        {
            await Clients.Caller.SendAsync(evt.EventName, evt.PayloadJson);
        }
    }

    await base.OnConnectedAsync();
}
```

### Trade-Offs vs Strategy 2

- Client has more control over what it considers "missed".
- Requires disciplined client-side storage of lastEventId.
- If client storage is cleared (app reinstall), falls back to full fetch.

---

## Decision Matrix

| Criteria                    | Strategy 1 (REST) | Strategy 2 (Redis Queue) | Strategy 3 (Last-Event-ID) |
|-----------------------------|--------------------|--------------------------|-----------------------------|
| Socket complexity           | None               | Medium                   | Medium                      |
| New infrastructure needed   | None               | Redis                    | Redis                       |
| Works for all event types   | Yes                | Yes                      | Yes                         |
| Message ordering guarantee  | API handles it     | Redis list order         | Redis list order            |
| Max offline duration        | Unlimited (DB)     | 24h (TTL)                | 24h (TTL)                   |
| Client complexity           | Medium             | Low                      | Medium                      |
| Socket stays thin           | Yes                | No                       | No                          |

## Recommendation

**Start with Strategy 1 (REST catch-up).** It requires zero Socket changes, keeps the Socket thin, and works for 90% of use cases.

Move to Strategy 2 only if you build a chat feature or auction system where sub-second event delivery matters. Even then, combine it with Strategy 1 as a fallback for long offline periods (> 24h).

Never skip Strategy 1. It is the foundation. Strategies 2 and 3 are optimizations layered on top.
