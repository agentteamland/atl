---
knowledge-base-summary: "Redis-based online user tracking. Who's connected right now? `connected:{userId}` → set of connectionIds. Updated on connect/disconnect. Enables \"is user online?\" queries and online user counts."
---
# Connection Tracking: Redis-Based Online Status

## Purpose

SignalR does not provide a built-in way to query "is user X online?" or "how many users are online?" Groups are fire-and-forget — you can broadcast to them but cannot enumerate their members. Connection tracking solves this by maintaining user connection state in Redis.

## Data Model in Redis

Two Redis structures per user:

| Key | Type | Value | Purpose |
|-----|------|-------|---------|
| `connections:{userId}` | Set | `{connectionId1, connectionId2, ...}` | All active connections for this user |
| `online:{userId}` | String | `"true"` | Fast "is online?" check |

### Why two keys?

- `connections:{userId}` tracks individual connections (multi-tab/multi-device). Needed for cleanup.
- `online:{userId}` is a fast existence check. Querying `EXISTS online:{userId}` is O(1) and avoids loading the entire set when you only need a boolean.

## Connection Lifecycle

```
User opens Tab 1:
  Redis: SADD connections:user-123 "conn-abc"       → {conn-abc}
  Redis: SET online:user-123 "true"

User opens Tab 2:
  Redis: SADD connections:user-123 "conn-def"       → {conn-abc, conn-def}
  (online:user-123 already exists, no change)

User closes Tab 1:
  Redis: SREM connections:user-123 "conn-abc"        → {conn-def}
  (set not empty → user still online, no change to online key)

User closes Tab 2:
  Redis: SREM connections:user-123 "conn-def"        → {}
  Redis: DEL connections:user-123
  Redis: DEL online:user-123                         → user is offline
```

## IConnectionTracker Interface

```csharp
namespace ExampleApp.Socket.Services;

public interface IConnectionTracker
{
    Task OnConnectedAsync(string userId, string connectionId);
    Task OnDisconnectedAsync(string userId, string connectionId);
    Task<bool> IsOnlineAsync(string userId);
    Task<int> GetOnlineCountAsync();
    Task<IReadOnlySet<string>> GetConnectionsAsync(string userId);
}
```

## ConnectionTracker Implementation

```csharp
using StackExchange.Redis;

namespace ExampleApp.Socket.Services;

public sealed class ConnectionTracker : IConnectionTracker
{
    private readonly IConnectionMultiplexer _redis;
    private readonly ILogger<ConnectionTracker> _logger;

    private static readonly RedisKey OnlinePrefix = "online:";
    private static readonly RedisKey ConnectionsPrefix = "connections:";

    public ConnectionTracker(
        IConnectionMultiplexer redis,
        ILogger<ConnectionTracker> logger)
    {
        _redis = redis;
        _logger = logger;
    }

    public async Task OnConnectedAsync(string userId, string connectionId)
    {
        var db = _redis.GetDatabase();

        // Add connectionId to the user's connection set
        await db.SetAddAsync($"connections:{userId}", connectionId);

        // Mark user as online
        await db.StringSetAsync($"online:{userId}", "true");

        _logger.LogInformation(
            "Connection tracked: user {UserId}, connection {ConnectionId}",
            userId, connectionId);
    }

    public async Task OnDisconnectedAsync(string userId, string connectionId)
    {
        var db = _redis.GetDatabase();

        // Remove this connectionId from the user's set
        await db.SetRemoveAsync($"connections:{userId}", connectionId);

        // Check if the user has any remaining connections
        var remainingCount = await db.SetLengthAsync($"connections:{userId}");

        if (remainingCount == 0)
        {
            // No more connections — user is offline
            await db.KeyDeleteAsync($"connections:{userId}");
            await db.KeyDeleteAsync($"online:{userId}");

            _logger.LogInformation(
                "User {UserId} went offline (last connection {ConnectionId} removed)",
                userId, connectionId);
        }
        else
        {
            _logger.LogInformation(
                "Connection removed: user {UserId}, connection {ConnectionId}, "
                + "{RemainingCount} connections remaining",
                userId, connectionId, remainingCount);
        }
    }

    public async Task<bool> IsOnlineAsync(string userId)
    {
        var db = _redis.GetDatabase();
        return await db.KeyExistsAsync($"online:{userId}");
    }

    public async Task<int> GetOnlineCountAsync()
    {
        var server = _redis.GetServer(_redis.GetEndPoints().First());

        var count = 0;
        await foreach (var key in server.KeysAsync(pattern: "online:*"))
        {
            count++;
        }

        return count;
    }

    public async Task<IReadOnlySet<string>> GetConnectionsAsync(string userId)
    {
        var db = _redis.GetDatabase();
        var members = await db.SetMembersAsync($"connections:{userId}");
        return members.Select(m => m.ToString()).ToHashSet();
    }
}
```

## Integration in NotificationHub

```csharp
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.SignalR;
using ExampleApp.Socket.Services;

namespace ExampleApp.Socket.Hubs;

[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly IApiClient _apiClient;
    private readonly IConnectionTracker _connectionTracker;
    private readonly ILogger<NotificationHub> _logger;

    public NotificationHub(
        IApiClient apiClient,
        IConnectionTracker connectionTracker,
        ILogger<NotificationHub> logger)
    {
        _apiClient = apiClient;
        _connectionTracker = connectionTracker;
        _logger = logger;
    }

    public override async Task OnConnectedAsync()
    {
        var userId = Context.UserIdentifier!;
        var connectionId = Context.ConnectionId;

        await _connectionTracker.OnConnectedAsync(userId, connectionId);

        _logger.LogInformation(
            "User {UserId} connected, connection {ConnectionId}",
            userId, connectionId);

        await base.OnConnectedAsync();
    }

    public override async Task OnDisconnectedAsync(Exception? exception)
    {
        var userId = Context.UserIdentifier!;
        var connectionId = Context.ConnectionId;

        await _connectionTracker.OnDisconnectedAsync(userId, connectionId);

        _logger.LogInformation(
            "User {UserId} disconnected, connection {ConnectionId}, exception: {Exception}",
            userId, connectionId, exception?.Message);

        await base.OnDisconnectedAsync(exception);
    }
}
```

## Registration in Program.cs

```csharp
using StackExchange.Redis;

// Redis connection
var redisConnection = builder.Configuration.GetConnectionString("Redis")
    ?? "redis:6379";
builder.Services.AddSingleton<IConnectionMultiplexer>(
    ConnectionMultiplexer.Connect(redisConnection));

// Connection tracker
builder.Services.AddSingleton<IConnectionTracker, ConnectionTracker>();
```

## Querying Online Status

### From a hub method

```csharp
public async Task CheckUserOnline(string targetUserId)
{
    var isOnline = await _connectionTracker.IsOnlineAsync(targetUserId);

    await Clients.Caller.SendAsync("UserOnlineStatus", new
    {
        UserId = targetUserId,
        IsOnline = isOnline,
    });
}
```

### From an internal endpoint

Expose an endpoint that the API can call to check online status:

```csharp
// In InternalEndpoints
group.MapGet("/online/{userId}", async (
    string userId,
    IConnectionTracker connectionTracker) =>
{
    var isOnline = await connectionTracker.IsOnlineAsync(userId);
    return Results.Ok(new { userId, isOnline });
})
.AddEndpointFilter<InternalSecretFilter>();

group.MapGet("/online-count", async (
    IConnectionTracker connectionTracker) =>
{
    var count = await connectionTracker.GetOnlineCountAsync();
    return Results.Ok(new { onlineCount = count });
})
.AddEndpointFilter<InternalSecretFilter>();
```

## Multi-Tab / Multi-Device Support

The set-based approach naturally handles multiple connections per user:

```
User opens browser Tab 1 → connections:user-123 = {conn-A}
User opens browser Tab 2 → connections:user-123 = {conn-A, conn-B}
User opens mobile app    → connections:user-123 = {conn-A, conn-B, conn-C}

All three connections receive broadcasts via Clients.User(userId)
Online status: true (until ALL connections are closed)
```

This is why we use a Redis **set** instead of a simple key — a single string key would only track the last connection, losing awareness of the others.

## Edge Cases and Reliability

### Server crash / ungraceful shutdown

If the Socket server crashes, `OnDisconnectedAsync` never fires and stale connections remain in Redis. Two mitigation strategies:

**Strategy 1: TTL with refresh (recommended)**

Set a TTL on the online key and refresh it periodically:

```csharp
public async Task OnConnectedAsync(string userId, string connectionId)
{
    var db = _redis.GetDatabase();
    await db.SetAddAsync($"connections:{userId}", connectionId);
    // Auto-expire after 5 minutes — heartbeat refreshes it
    await db.StringSetAsync($"online:{userId}", "true", TimeSpan.FromMinutes(5));
}

// Called periodically (e.g., every 2 minutes via a BackgroundService or SignalR keep-alive)
public async Task RefreshOnlineStatusAsync(string userId)
{
    var db = _redis.GetDatabase();
    var hasConnections = await db.SetLengthAsync($"connections:{userId}") > 0;
    if (hasConnections)
    {
        await db.StringSetAsync($"online:{userId}", "true", TimeSpan.FromMinutes(5));
    }
}
```

**Strategy 2: Startup cleanup**

On Socket server startup, clear all connection data and let clients reconnect:

```csharp
// In Program.cs or a hosted service — runs once at startup
var server = redis.GetServer(redis.GetEndPoints().First());
await foreach (var key in server.KeysAsync(pattern: "connections:*"))
{
    await redis.GetDatabase().KeyDeleteAsync(key);
}
await foreach (var key in server.KeysAsync(pattern: "online:*"))
{
    await redis.GetDatabase().KeyDeleteAsync(key);
}
```

### Race condition: connect and disconnect in rapid succession

If a user closes and reopens a tab instantly, the disconnect and connect events might race. The set-based approach handles this correctly — `SADD` and `SREM` are atomic Redis operations and operate on different connection IDs, so they do not conflict.

## Performance Considerations

| Operation | Redis Command | Time Complexity |
|-----------|---------------|-----------------|
| Track connection | `SADD` | O(1) |
| Remove connection | `SREM` | O(1) |
| Check online | `EXISTS` | O(1) |
| Count online users | `KEYS online:*` | O(N) - use sparingly |
| Get user connections | `SMEMBERS` | O(N) where N = connections per user |

For `GetOnlineCountAsync`, the `KEYS` pattern scan is O(N) across all keys. This is acceptable for admin dashboards called infrequently. For high-frequency queries, maintain a separate counter:

```csharp
// Alternative: increment/decrement a counter
await db.StringIncrementAsync("stats:online-count");   // on connect
await db.StringDecrementAsync("stats:online-count");   // on last disconnect
```

## Docker Compose Configuration

```yaml
services:
  socket:
    environment:
      - ConnectionStrings__Redis=redis:6379
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```
