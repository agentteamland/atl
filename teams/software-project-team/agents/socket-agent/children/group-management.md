---
knowledge-base-summary: "Tenant/room-based groups for targeted broadcasting. OnConnectedAsync → join tenant group. OnDisconnectedAsync → leave. Groups enable \"broadcast to all users of Dealer X\" without hitting every connected client."
---
# Group Management: Tenant/Room-Based Groups

## Purpose

SignalR groups enable targeted broadcasting. Instead of sending a message to all connected clients, you send it to a specific group — all users of a tenant, all members of a chat room, all admins of an organization. Groups are the backbone of multi-tenant, room-based real-time communication.

## How Groups Work in SignalR

- Groups are identified by a **string name**.
- A connection can be in **multiple groups** simultaneously.
- Groups are **ephemeral** — they exist only in memory, tied to the SignalR server. No persistence.
- When a connection disconnects, it is automatically removed from all groups.
- Groups are managed via `Groups.AddToGroupAsync()` and `Groups.RemoveFromGroupAsync()`.

## Group Naming Convention

Use a prefixed naming convention to avoid collisions and clarify intent:

| Pattern | Example | Use Case |
|---------|---------|----------|
| `tenant:{tenantId}` | `tenant:acme-corp` | All users in a tenant/organization |
| `room:{roomId}` | `room:support-chat-42` | All users in a chat room |
| `role:{roleName}` | `role:admin` | All users with a specific role |
| `user:{userId}` | `user:guid-123` | All connections of a single user (multi-tab) |

**Why prefixed?** Without prefixes, a group named `"42"` is ambiguous — is it a room? A tenant? A user? The prefix makes it self-documenting and prevents accidental cross-type collisions.

## Automatic Group Assignment on Connect

When a user connects, extract their tenant from the JWT claims and add them to the tenant group. This happens in `OnConnectedAsync` — no manual hub method call needed.

```csharp
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.SignalR;
using System.Security.Claims;

namespace ExampleApp.Socket.Hubs;

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
        var userId = Context.UserIdentifier!;
        var connectionId = Context.ConnectionId;

        // Extract tenant from JWT claims
        var tenantId = Context.User?.FindFirstValue("tenantId");

        // Auto-join tenant group
        if (!string.IsNullOrEmpty(tenantId))
        {
            await Groups.AddToGroupAsync(connectionId, $"tenant:{tenantId}");
            _logger.LogInformation(
                "User {UserId} joined group tenant:{TenantId}, connection {ConnectionId}",
                userId, tenantId, connectionId);
        }

        // Auto-join user-level group (for multi-tab/device support)
        await Groups.AddToGroupAsync(connectionId, $"user:{userId}");
        _logger.LogInformation(
            "User {UserId} connected, connection {ConnectionId}",
            userId, connectionId);

        await base.OnConnectedAsync();
    }

    public override async Task OnDisconnectedAsync(Exception? exception)
    {
        var userId = Context.UserIdentifier!;
        var connectionId = Context.ConnectionId;
        var tenantId = Context.User?.FindFirstValue("tenantId");

        // Groups.RemoveFromGroupAsync is technically not needed here —
        // SignalR auto-removes disconnected connections from all groups.
        // But explicit removal makes the code self-documenting and
        // ensures cleanup if custom logic depends on it.
        if (!string.IsNullOrEmpty(tenantId))
        {
            await Groups.RemoveFromGroupAsync(connectionId, $"tenant:{tenantId}");
            _logger.LogInformation(
                "User {UserId} left group tenant:{TenantId}, connection {ConnectionId}",
                userId, tenantId, connectionId);
        }

        await Groups.RemoveFromGroupAsync(connectionId, $"user:{userId}");
        _logger.LogInformation(
            "User {UserId} disconnected, connection {ConnectionId}, exception: {Exception}",
            userId, connectionId, exception?.Message);

        await base.OnDisconnectedAsync(exception);
    }
}
```

## Manual Group Join/Leave (Room-Based)

For dynamic groups like chat rooms, the client explicitly joins and leaves via hub methods. The hub validates room membership through the API before adding to the group.

```csharp
public async Task JoinRoom(string roomId)
{
    var userId = Context.UserIdentifier!;
    _logger.LogInformation(
        "JoinRoom request: user {UserId}, room {RoomId}",
        userId, roomId);

    // Validate membership via API — Socket never decides access
    var response = await _apiClient.GetAsync(
        $"/api/rooms/{roomId}/membership/{userId}");

    if (!response.IsSuccessStatusCode)
    {
        _logger.LogWarning(
            "JoinRoom denied: user {UserId}, room {RoomId}, status {StatusCode}",
            userId, roomId, (int)response.StatusCode);
        throw new HubException("Not authorized to join this room");
    }

    await Groups.AddToGroupAsync(Context.ConnectionId, $"room:{roomId}");

    // Notify room members
    await Clients.Group($"room:{roomId}").SendAsync("UserJoined", new
    {
        UserId = userId,
        RoomId = roomId,
        JoinedAt = DateTime.UtcNow,
    });

    _logger.LogInformation(
        "User {UserId} joined room:{RoomId}", userId, roomId);
}

public async Task LeaveRoom(string roomId)
{
    var userId = Context.UserIdentifier!;

    await Groups.RemoveFromGroupAsync(Context.ConnectionId, $"room:{roomId}");

    // Notify remaining room members
    await Clients.Group($"room:{roomId}").SendAsync("UserLeft", new
    {
        UserId = userId,
        RoomId = roomId,
        LeftAt = DateTime.UtcNow,
    });

    _logger.LogInformation(
        "User {UserId} left room:{RoomId}", userId, roomId);
}
```

## Broadcasting to Groups

### From within a Hub method

```csharp
// Send to everyone in the tenant
await Clients.Group($"tenant:{tenantId}")
    .SendAsync("OrderCreated", new { OrderId = orderId });

// Send to everyone in a room
await Clients.Group($"room:{roomId}")
    .SendAsync("MessageReceived", messageDto);

// Send to everyone in a room EXCEPT the caller
await Clients.GroupExcept($"room:{roomId}", Context.ConnectionId)
    .SendAsync("UserTyping", new { UserId = userId });
```

### From the internal broadcast endpoint (via IHubContext)

```csharp
// In InternalEndpoints — API pushes event to a group
case "group":
    await hubContext.Clients
        .Group(request.Group!)
        .SendAsync(request.Event, request.Payload);
    break;
```

The API calls this with a request like:

```json
{
    "event": "new-order",
    "payload": { "orderId": "abc-123", "total": 42.50 },
    "group": "tenant:acme-corp",
    "targetType": "group"
}
```

## Multiple Groups Per User

A user can be in many groups at the same time. This is normal and expected.

```
User "mesut" connected from Tab 1:
  - tenant:acme-corp        (auto-joined on connect)
  - user:mesut-guid          (auto-joined on connect)
  - room:support-chat-42     (manually joined)
  - room:team-general        (manually joined)
  - role:admin               (auto-joined based on claims)
```

A broadcast to `tenant:acme-corp` reaches all of the user's connections. A broadcast to `room:support-chat-42` reaches only the connections that joined that room.

## Role-Based Groups

If the JWT contains a role claim, auto-join the role group on connect:

```csharp
public override async Task OnConnectedAsync()
{
    var userId = Context.UserIdentifier!;
    var connectionId = Context.ConnectionId;

    // Role-based group
    var role = Context.User?.FindFirstValue(ClaimTypes.Role);
    if (!string.IsNullOrEmpty(role))
    {
        await Groups.AddToGroupAsync(connectionId, $"role:{role}");
        _logger.LogInformation(
            "User {UserId} joined group role:{Role}",
            userId, role);
    }

    await base.OnConnectedAsync();
}
```

This enables broadcasts like "notify all admins" without knowing individual admin user IDs.

## Important Caveats

1. **Groups are per-server.** In a multi-server deployment, groups need a backplane (Redis, Azure SignalR Service) to work across servers. Without a backplane, a user connected to server A will not receive broadcasts sent to a group on server B.

2. **No group membership queries.** SignalR does not provide a way to ask "who is in this group?" or "how many connections are in this group?" If you need this, track it separately in Redis (see `connection-tracking.md`).

3. **Group names are case-sensitive.** `"tenant:ACME"` and `"tenant:acme"` are different groups. Normalize to lowercase before using.

4. **Empty groups cost nothing.** A group with no connections simply has no effect — broadcasting to it is a no-op. You do not need to create or destroy groups explicitly.

5. **Auto-cleanup on disconnect.** When a connection drops, SignalR automatically removes it from all groups. The explicit `RemoveFromGroupAsync` in `OnDisconnectedAsync` is for clarity and any custom cleanup logic, not strictly necessary.
