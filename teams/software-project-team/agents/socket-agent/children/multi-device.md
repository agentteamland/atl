---
knowledge-base-summary: "Same user connected from phone + desktop + multiple browser tabs. \"Order confirmed\" → all devices receive it, not just one. User-level broadcasting via userId, not connectionId. Connection tracking supports multiple connections per user."
---
# Multi-Tab and Multi-Device Support

## The Problem

A single user can be connected from multiple places simultaneously:

- Phone (Flutter iOS app)
- Tablet (Flutter Android app)
- Browser tab 1 (React web app)
- Browser tab 2 (same React web app, different tab)

Each connection has a unique `connectionId`. But the user is one person. When their task completes, ALL four connections must receive the `task-completed` event. Not just the one that triggered it.

SignalR assigns a new `connectionId` per connection. One user = N connectionIds. If you broadcast to a specific connectionId, only one device gets the event. The others miss it.

## SignalR's User Concept

SignalR has built-in user-level broadcasting. When you call:

```csharp
await Clients.User(userId).SendAsync("task-completed", payload);
```

SignalR internally resolves `userId` to ALL active connectionIds for that user and sends the event to every one of them. You do not need to track connections yourself for this to work.

The key is telling SignalR HOW to map a connection to a userId. This is done via `IUserIdProvider`.

## Custom IUserIdProvider

By default, SignalR uses `ClaimTypes.NameIdentifier` from the JWT claims. If your JWT uses a different claim name for the user ID, you must provide a custom `IUserIdProvider`.

### Implementation

```csharp
// Services/JwtUserIdProvider.cs
using Microsoft.AspNetCore.SignalR;
using System.Security.Claims;

namespace ExampleApp.Socket.Services;

/// <summary>
/// Extracts the userId from JWT claims for SignalR user-level broadcasting.
/// SignalR uses this to map connectionId → userId.
/// </summary>
public sealed class JwtUserIdProvider : IUserIdProvider
{
    public string? GetUserId(HubConnectionContext connection)
    {
        // Option 1: Standard claim (if your JWT uses "sub" or NameIdentifier)
        var userId = connection.User?.FindFirst(ClaimTypes.NameIdentifier)?.Value;

        // Option 2: Custom claim name (if your JWT uses "userId" or "uid")
        // var userId = connection.User?.FindFirst("userId")?.Value;

        return userId;
    }
}
```

### Registration in Program.cs

```csharp
// Program.cs

// Register BEFORE AddSignalR
builder.Services.AddSingleton<IUserIdProvider, JwtUserIdProvider>();

builder.Services.AddSignalR();
```

The `IUserIdProvider` must be registered as a singleton. SignalR calls it for every connection to determine the user identity.

## Three Levels of Broadcasting

### 1. User-Level: Clients.User(userId)

Sends to ALL connections of a specific user across all devices and tabs.

```csharp
// Send to ALL of Ali's devices (phone + tablet + browser tabs)
await Clients.User(userId).SendAsync("task-completed", new TaskCompletedEvent
{
    EventId = Guid.NewGuid().ToString("N"),
    TaskId = taskId,
    DistanceMeters = 5420,
    Timestamp = DateTimeOffset.UtcNow
});
```

**When to use:**
- Notifications (friend request, achievement, reminder)
- Status changes (order confirmed, payment processed)
- Sync signals (profile updated, settings changed)
- Any event that the USER should see regardless of which device they are on

**This is the default for 90% of events.**

### 2. Group-Level: Clients.Group(groupName)

Sends to all connections in a specific group (room, channel, tenant).

```csharp
// Send to everyone in the chat room, regardless of user
await Clients.Group($"room:{roomId}").SendAsync("chat-message-received", new ChatMessageReceivedEvent
{
    EventId = Guid.NewGuid().ToString("N"),
    MessageId = messageId,
    SenderId = senderId,
    Content = "Let's task at 5pm!",
    RoomId = roomId,
    Timestamp = DateTimeOffset.UtcNow
});
```

**When to use:**
- Chat messages in a room
- Typing indicators in a room
- Collaborative features (multiple users viewing the same page)
- Tenant-scoped events (all users of Organization X)

### 3. Connection-Level: Clients.Client(connectionId)

Sends to ONE specific connection. Not the user -- just that one browser tab or device.

```csharp
// Send ONLY to the specific connection that made a request
await Clients.Client(Context.ConnectionId).SendAsync("error", new
{
    Message = "Invalid room ID",
    Timestamp = DateTimeOffset.UtcNow
});
```

**When to use:**
- Error responses to a specific hub method call
- Acknowledgments for a specific action taken on ONE device
- Rate limit warnings (only warn the offending connection)

**Rarely needed.** Most events should be user-level.

## Decision Matrix

| Scenario                                   | Level      | Method                                  |
|-------------------------------------------|------------|-----------------------------------------|
| Task completed                             | User       | `Clients.User(userId)`                  |
| New notification                           | User       | `Clients.User(userId)`                  |
| Profile updated on another device          | User       | `Clients.User(userId)`                  |
| Chat message in a room                     | Group      | `Clients.Group(roomId)`                 |
| Typing indicator in a room                 | Group      | `Clients.GroupExcept(roomId, connId)`    |
| 5 users viewing this task route            | Group      | `Clients.Group($"view:{routeId}")`      |
| Error response to a hub method call        | Connection | `Clients.Client(connectionId)`          |
| Rate limit exceeded warning                | Connection | `Clients.Client(connectionId)`          |
| System announcement to all users           | All        | `Clients.All`                           |

## Excluding the Sender

When a user types a message, they do not need to receive their own typing indicator. Use `GroupExcept` or `AllExcept`:

```csharp
// In the Hub
public async Task StartTyping(Guid roomId)
{
    var userId = Context.UserIdentifier!;

    // Send to the room, but NOT to the connection that sent it
    await Clients.GroupExcept($"room:{roomId}", Context.ConnectionId)
        .SendAsync("chat-user-typing", new ChatUserTypingEvent
        {
            UserId = Guid.Parse(userId),
            RoomId = roomId,
            Timestamp = DateTimeOffset.UtcNow
        });
}
```

Note: `GroupExcept` excludes by connectionId, not userId. If the same user has 3 connections in the room, only the initiating connection is excluded. The other 2 connections of the same user WILL receive the event. This is usually the desired behavior (phone should show "you are typing" from the desktop session).

## Internal Broadcast Endpoint

When the API sends a broadcast to Socket via the internal endpoint, it typically uses user-level targeting:

```csharp
// InternalEndpoints.cs
app.MapPost("/api/internal/broadcast", async (
    BroadcastRequest request,
    IHubContext<NotificationHub> hubContext) =>
{
    switch (request.TargetType)
    {
        case "user":
            // All devices of this user
            await hubContext.Clients.User(request.TargetId)
                .SendAsync(request.EventName, request.Payload);
            break;

        case "group":
            // All connections in this group
            await hubContext.Clients.Group(request.TargetId)
                .SendAsync(request.EventName, request.Payload);
            break;

        case "all":
            // Every connected client
            await hubContext.Clients.All
                .SendAsync(request.EventName, request.Payload);
            break;
    }

    return Results.Ok();
});
```

The API never needs to know about connectionIds. It always targets by userId or groupName. SignalR resolves the rest.

## Connection Tracking for Multi-Device

To answer "how many devices does this user have connected?", use Redis-based connection tracking. This is documented in `children/connection-tracking.md`, but the relevant parts for multi-device:

### OnConnectedAsync

```csharp
public override async Task OnConnectedAsync()
{
    var userId = Context.UserIdentifier!;
    var connectionId = Context.ConnectionId;

    // Add this connectionId to the user's set
    await _redis.SetAddAsync($"connected:{userId}", connectionId);

    _logger.LogInformation(
        "User {UserId} connected with ConnectionId {ConnectionId}. Total connections: {Count}",
        userId, connectionId,
        await _redis.SetLengthAsync($"connected:{userId}"));

    await base.OnConnectedAsync();
}
```

### OnDisconnectedAsync

```csharp
public override async Task OnDisconnectedAsync(Exception? exception)
{
    var userId = Context.UserIdentifier!;
    var connectionId = Context.ConnectionId;

    // Remove this connectionId from the user's set
    await _redis.SetRemoveAsync($"connected:{userId}", connectionId);

    var remaining = await _redis.SetLengthAsync($"connected:{userId}");

    if (remaining == 0)
    {
        // User has no more connections -- truly offline
        await _redis.KeyDeleteAsync($"connected:{userId}");

        _logger.LogInformation("User {UserId} is now fully offline", userId);

        // Optionally broadcast presence change
        // await Clients.All.SendAsync("presence-changed", ...);
    }
    else
    {
        _logger.LogInformation(
            "User {UserId} disconnected ConnectionId {ConnectionId}. {Remaining} connections remain",
            userId, connectionId, remaining);
    }

    await base.OnDisconnectedAsync(exception);
}
```

### Key Insight

A user is only truly "offline" when their connection set in Redis is empty. Disconnecting one browser tab does not make them offline if their phone is still connected.

## Multiple Socket Instances (Scale-Out)

When running multiple Socket instances behind a load balancer, a user's connections may be distributed across different instances:

```
User Ali:
  - Connection A → Socket Instance 1
  - Connection B → Socket Instance 2
  - Connection C → Socket Instance 1
```

`Clients.User("ali")` on Instance 1 only knows about connections A and C. Connection B on Instance 2 is invisible.

### Solution: Redis Backplane

SignalR's Redis backplane synchronizes events across instances:

```csharp
// Program.cs
builder.Services.AddSignalR()
    .AddStackExchangeRedis(
        builder.Configuration.GetConnectionString("Redis") ?? "redis:6379",
        options =>
        {
            options.Configuration.ChannelPrefix =
                RedisChannel.Literal("ExampleApp:SignalR:");
        });
```

With the Redis backplane:
1. Instance 1 calls `Clients.User("ali").SendAsync(...)`.
2. The event is published to Redis pub/sub.
3. Instance 2 picks it up and delivers to Connection B.
4. All three connections receive the event.

**Always configure the Redis backplane in production.** Without it, multi-instance Socket deployments will lose events.

## Summary

| Concept | Implementation |
|---------|---------------|
| Map connection to user | `IUserIdProvider` extracts userId from JWT |
| Send to all user devices | `Clients.User(userId)` |
| Send to a room | `Clients.Group(groupName)` |
| Send to one connection | `Clients.Client(connectionId)` |
| Exclude sender | `GroupExcept(group, connectionId)` |
| Track connections per user | Redis Set: `connected:{userId}` |
| True offline detection | Redis Set length = 0 |
| Multi-instance support | SignalR Redis backplane |
| Default broadcasting level | User-level (90% of events) |
