---
knowledge-base-summary: "SignalR hub structure. [Authorize] attribute, hub method naming, method grouping. Hub methods are bridges — parse client input, call API via IApiClient, return result. No logic in hubs."
---
# Hub Design: SignalR Hub Structure

## Core Rule

The hub is a **bridge** — it receives a call from the client, forwards it to the API via `IApiClient`, and returns the result. No business logic, no database access, no service orchestration. If you find yourself writing an `if` statement that makes a business decision, you are doing it wrong — that logic belongs in the API.

## Hub Class Structure

Every hub:
- Has the `[Authorize]` attribute (no anonymous hub access)
- Injects `IApiClient` and `ILogger<T>` via constructor
- Contains only bridge methods
- Overrides `OnConnectedAsync` and `OnDisconnectedAsync` for lifecycle logging

```csharp
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.SignalR;
using ExampleApp.Socket.Services;

namespace ExampleApp.Socket.Hubs;

[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly IApiClient _apiClient;
    private readonly ILogger<NotificationHub> _logger;

    public NotificationHub(IApiClient apiClient, ILogger<NotificationHub> logger)
    {
        _apiClient = apiClient;
        _logger = logger;
    }

    public override async Task OnConnectedAsync()
    {
        var userId = Context.UserIdentifier;
        _logger.LogInformation(
            "User {UserId} connected, connectionId {ConnectionId}",
            userId, Context.ConnectionId);
        await base.OnConnectedAsync();
    }

    public override async Task OnDisconnectedAsync(Exception? exception)
    {
        var userId = Context.UserIdentifier;
        _logger.LogInformation(
            "User {UserId} disconnected, connectionId {ConnectionId}, exception: {Exception}",
            userId, Context.ConnectionId, exception?.Message);
        await base.OnDisconnectedAsync(exception);
    }
}
```

## Hub Method Pattern

Every hub method follows the same three-step pattern:

1. **Parse** — extract data from the client's call
2. **Forward** — call API via `_apiClient`
3. **Return** — send the API's response back to the caller (and optionally broadcast to others)

```csharp
// Example 1: Client sends a message to a room
public async Task SendMessage(string roomId, string content)
{
    var userId = Context.UserIdentifier!;
    _logger.LogInformation(
        "SendMessage from {UserId} to room {RoomId}",
        userId, roomId);

    var response = await _apiClient.PostAsync("/api/messages", new
    {
        RoomId = roomId,
        Content = content,
        SenderId = userId,
    });

    if (!response.IsSuccessStatusCode)
    {
        _logger.LogWarning(
            "SendMessage failed for {UserId}, status {StatusCode}",
            userId, (int)response.StatusCode);
        throw new HubException("Failed to send message");
    }

    var message = await response.Content.ReadFromJsonAsync<MessageDto>();

    // Broadcast to everyone in the room (including sender)
    await Clients.Group($"room:{roomId}").SendAsync("MessageReceived", message);

    _logger.LogInformation(
        "Message sent by {UserId} to room {RoomId}, messageId {MessageId}",
        userId, roomId, message?.Id);
}

// Example 2: Client joins a room
public async Task JoinRoom(string roomId)
{
    var userId = Context.UserIdentifier!;
    _logger.LogInformation(
        "JoinRoom request from {UserId} for room {RoomId}",
        userId, roomId);

    // Validate membership via API
    var response = await _apiClient.GetAsync(
        $"/api/rooms/{roomId}/membership/{userId}");

    if (!response.IsSuccessStatusCode)
    {
        _logger.LogWarning(
            "JoinRoom denied for {UserId} in room {RoomId}, status {StatusCode}",
            userId, roomId, (int)response.StatusCode);
        throw new HubException("Not authorized to join this room");
    }

    await Groups.AddToGroupAsync(Context.ConnectionId, $"room:{roomId}");

    // Notify others in the room
    await Clients.Group($"room:{roomId}").SendAsync("UserJoined", new
    {
        UserId = userId,
        RoomId = roomId,
        JoinedAt = DateTime.UtcNow,
    });

    _logger.LogInformation(
        "User {UserId} joined room {RoomId}",
        userId, roomId);
}

// Example 3: Client requests their unread count
public async Task GetUnreadCount()
{
    var userId = Context.UserIdentifier!;

    var response = await _apiClient.GetAsync(
        $"/api/notifications/unread-count?userId={userId}");

    if (!response.IsSuccessStatusCode)
    {
        throw new HubException("Failed to fetch unread count");
    }

    var result = await response.Content.ReadFromJsonAsync<UnreadCountDto>();

    // Return only to the caller
    await Clients.Caller.SendAsync("UnreadCount", result);
}
```

## Method Naming Conventions

| Convention | Example | Explanation |
|------------|---------|-------------|
| Hub methods | `SendMessage`, `JoinRoom`, `GetUnreadCount` | PascalCase, verb-first |
| Client-side event names | `"MessageReceived"`, `"UserJoined"`, `"UnreadCount"` | PascalCase strings in `SendAsync()` |
| Group names | `"room:{roomId}"`, `"tenant:{tenantId}"` | Prefixed with type, kebab-case |

## What Does NOT Belong in a Hub

| Prohibited | Why | Where It Belongs |
|-----------|-----|-----------------|
| `DbContext` / EF Core queries | Socket has no Infrastructure reference | API handlers |
| Business rules / validation | Socket is a bridge, not a decision-maker | Application layer |
| Direct Redis calls for data | Data access goes through API | Infrastructure layer |
| Email / RMQ publishing | Side effects belong in handlers | API + MailSender/Worker |
| Complex object mapping | Keep hub methods flat and simple | API response DTOs |

**Exception:** Redis calls for connection tracking (online status) are acceptable in the hub because they are Socket-level infrastructure, not business data. See `connection-tracking.md`.

## Hub Registration in Program.cs

```csharp
builder.Services.AddSignalR();

// ...

app.MapHub<NotificationHub>("/hubs/notifications");
```

The hub path follows the pattern `/hubs/{hubName}`. Client connects to `wss://socket-host:3002/hubs/notifications?access_token={jwt}`.

## Error Handling in Hub Methods

Hub methods throw `HubException` for client-visible errors. The SignalR framework serializes this to the client. Never throw raw exceptions — they leak stack traces.

```csharp
// Correct: HubException with a client-friendly message
if (!response.IsSuccessStatusCode)
{
    _logger.LogWarning("API returned {StatusCode}", (int)response.StatusCode);
    throw new HubException("Operation failed. Please try again.");
}

// Wrong: letting any exception bubble up
var data = await response.Content.ReadFromJsonAsync<SomeDto>()
    ?? throw new NullReferenceException(); // Leaks internal details to client
```

## DTOs

Hub methods use simple record DTOs for deserialization. These live in the Socket project — they are lightweight mirrors of API responses, not shared contracts.

```csharp
namespace ExampleApp.Socket.Hubs;

public sealed record MessageDto(Guid Id, string Content, string SenderId, DateTime CreatedAt);
public sealed record UnreadCountDto(int Count);
```

## Checklist for Writing a New Hub Method

- [ ] Method is PascalCase, verb-first
- [ ] Only `IApiClient` and `ILogger<T>` injected (no DbContext, no services)
- [ ] Three-step pattern: parse, forward to API, return/broadcast result
- [ ] Errors wrapped in `HubException` (no raw exceptions)
- [ ] Entry log with userId and parameters
- [ ] Result log with outcome
- [ ] No business logic — if you need an `if` for a business rule, it goes in the API
