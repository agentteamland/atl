---
knowledge-base-summary: "Naming: kebab-case events (`order-confirmed`, `user-typing`). Payload contracts: what data each event carries. Client-server agreement. Documented per event."
---
# Event Conventions

## Why Conventions Matter

Without naming conventions, events become a mess over time. One developer uses `orderConfirmed`, another uses `OrderConfirmed`, a third uses `order_confirmed`. The Flutter client handles `order-confirmed` but the event is sent as `orderConfirmed` and nothing happens. Silent failure. No error. Hours of debugging.

Conventions prevent this. Every event follows the same rules. Every developer knows what to expect.

## Event Naming Rules

### Rule 1: kebab-case for all event names

```
order-confirmed       -- correct
user-typing           -- correct
message-received      -- correct

orderConfirmed        -- wrong (camelCase)
OrderConfirmed        -- wrong (PascalCase)
order_confirmed       -- wrong (snake_case)
MESSAGE_RECEIVED      -- wrong (SCREAMING_CASE)
```

SignalR uses string-based event names. The client and server must agree exactly. kebab-case is the convention because it is readable, URL-friendly, and unambiguous.

### Rule 2: Past tense for server-to-client events

Events from server to client describe something that **already happened**:

```
order-confirmed       -- the order was confirmed
task-started          -- the task began
message-received      -- a message arrived
user-joined           -- someone joined
payment-failed        -- payment did not go through
```

### Rule 3: Present tense / imperative for client-to-server events

Events from client to server describe an **action the client wants to perform**:

```
send-message          -- client wants to send a message
start-typing          -- client started typing
join-room             -- client wants to join a room
mark-read             -- client wants to mark as read
```

### Rule 4: Feature prefix grouping

Group related events by feature prefix:

```
task-*                -- task-started, task-paused, task-completed, task-step-counted
chat-*                -- chat-message-received, chat-user-typing, chat-user-left
notification-*        -- notification-received, notification-dismissed
order-*               -- order-confirmed, order-cancelled, order-shipped
presence-*            -- presence-online, presence-offline, presence-idle
```

This makes it easy to:
- Find all events related to a feature.
- Apply rate limiting per feature group.
- Add/remove feature-level event subscriptions.

## Payload Rules

### Rule 5: Always a typed object, never a raw primitive

```csharp
// WRONG -- raw primitive
await Clients.User(userId).SendAsync("task-completed", taskId);

// WRONG -- anonymous object (no type safety on server)
await Clients.User(userId).SendAsync("task-completed", new { taskId, distance });

// CORRECT -- typed record
await Clients.User(userId).SendAsync("task-completed", new TaskCompletedEvent
{
    TaskId = taskId,
    DistanceMeters = 5420,
    DurationSeconds = 3600,
    Timestamp = DateTimeOffset.UtcNow
});
```

### Rule 6: Every payload includes a timestamp

```csharp
public sealed record TaskCompletedEvent
{
    public required Guid TaskId { get; init; }
    public required double DistanceMeters { get; init; }
    public required int DurationSeconds { get; init; }
    public required DateTimeOffset Timestamp { get; init; } // Always present
}
```

The timestamp is when the event was emitted, NOT when the underlying action happened (though they may be the same). The client uses this for:
- Ordering events correctly.
- Detecting stale events after reconnection.
- Display purposes ("2 minutes ago").

### Rule 7: Include an eventId for deduplication

For events that may be replayed (reconnection scenarios), include an ID:

```csharp
public sealed record NotificationReceivedEvent
{
    public required string EventId { get; init; }      // Unique per event instance
    public required string Type { get; init; }         // "task-reminder", "friend-request"
    public required string Title { get; init; }
    public required string Body { get; init; }
    public required DateTimeOffset Timestamp { get; init; }
}
```

The client can use `eventId` to deduplicate if the same event arrives both live and via reconnection replay.

## Payload Contracts Directory

All event payloads live in a shared contracts directory:

```
src/ExampleApp.Socket/
    Contracts/
        Events/
            Task/
                TaskStartedEvent.cs
                TaskCompletedEvent.cs
                TaskStepCountedEvent.cs
            Chat/
                ChatMessageReceivedEvent.cs
                ChatUserTypingEvent.cs
            Notification/
                NotificationReceivedEvent.cs
                NotificationDismissedEvent.cs
            Presence/
                PresenceChangedEvent.cs
```

```csharp
// Contracts/Events/Task/TaskStartedEvent.cs
namespace ExampleApp.Socket.Contracts.Events.Task;

public sealed record TaskStartedEvent
{
    public required string EventId { get; init; }
    public required Guid TaskId { get; init; }
    public required Guid UserId { get; init; }
    public required DateTimeOffset StartedAt { get; init; }
    public required DateTimeOffset Timestamp { get; init; }
}
```

```csharp
// Contracts/Events/Task/TaskCompletedEvent.cs
namespace ExampleApp.Socket.Contracts.Events.Task;

public sealed record TaskCompletedEvent
{
    public required string EventId { get; init; }
    public required Guid TaskId { get; init; }
    public required Guid UserId { get; init; }
    public required double DistanceMeters { get; init; }
    public required int DurationSeconds { get; init; }
    public required int StepCount { get; init; }
    public required DateTimeOffset CompletedAt { get; init; }
    public required DateTimeOffset Timestamp { get; init; }
}
```

```csharp
// Contracts/Events/Chat/ChatMessageReceivedEvent.cs
namespace ExampleApp.Socket.Contracts.Events.Chat;

public sealed record ChatMessageReceivedEvent
{
    public required string EventId { get; init; }
    public required Guid MessageId { get; init; }
    public required Guid SenderId { get; init; }
    public required string SenderName { get; init; }
    public required string Content { get; init; }
    public required Guid RoomId { get; init; }
    public required DateTimeOffset SentAt { get; init; }
    public required DateTimeOffset Timestamp { get; init; }
}
```

```csharp
// Contracts/Events/Notification/NotificationReceivedEvent.cs
namespace ExampleApp.Socket.Contracts.Events.Notification;

public sealed record NotificationReceivedEvent
{
    public required string EventId { get; init; }
    public required Guid NotificationId { get; init; }
    public required string Type { get; init; }       // "task-reminder", "friend-request", "achievement"
    public required string Title { get; init; }
    public required string Body { get; init; }
    public required Dictionary<string, string>? Data { get; init; }  // Extra context
    public required DateTimeOffset Timestamp { get; init; }
}
```

## Event Catalog

Document every event in a structured format. This is the contract between Socket and all clients (Flutter, React, etc.).

### Format

```
| Event Name              | Direction         | Payload Type                | Description                                    |
|-------------------------|-------------------|-----------------------------|------------------------------------------------|
| task-started            | server -> client  | TaskStartedEvent            | A task session has begun                       |
| task-completed          | server -> client  | TaskCompletedEvent          | A task session has ended                       |
| task-step-counted       | server -> client  | TaskStepCountedEvent        | Step count updated during active task          |
| chat-message-received   | server -> client  | ChatMessageReceivedEvent    | New chat message in a room                     |
| chat-user-typing        | server -> client  | ChatUserTypingEvent         | A user started typing in a room                |
| notification-received   | server -> client  | NotificationReceivedEvent   | New notification for the user                  |
| notification-dismissed  | server -> client  | NotificationDismissedEvent  | A notification was dismissed (sync across tabs)|
| presence-changed        | server -> client  | PresenceChangedEvent        | A user went online/offline/idle                |
| send-message            | client -> server  | (hub method parameter)      | Client sends a chat message                    |
| start-typing            | client -> server  | (hub method parameter)      | Client signals typing started                  |
| stop-typing             | client -> server  | (hub method parameter)      | Client signals typing stopped                  |
| join-room               | client -> server  | (hub method parameter)      | Client joins a chat room                       |
| leave-room              | client -> server  | (hub method parameter)      | Client leaves a chat room                      |
| mark-read               | client -> server  | (hub method parameter)      | Client marks notifications as read             |
| rate-limit-exceeded     | server -> client  | (inline object)             | Client exceeded rate limit for a method        |
```

## Sending Events (Server Side)

### From Internal Broadcast Endpoint

When the API sends a broadcast request to Socket:

```csharp
// InternalEndpoints.cs
app.MapPost("/api/internal/broadcast", async (
    BroadcastRequest request,
    IHubContext<NotificationHub> hubContext) =>
{
    switch (request.TargetType)
    {
        case "user":
            await hubContext.Clients.User(request.TargetId)
                .SendAsync(request.EventName, request.Payload);
            break;

        case "group":
            await hubContext.Clients.Group(request.TargetId)
                .SendAsync(request.EventName, request.Payload);
            break;

        case "all":
            await hubContext.Clients.All
                .SendAsync(request.EventName, request.Payload);
            break;
    }
});
```

The `EventName` in the broadcast request follows the same kebab-case convention: `"task-completed"`, `"notification-received"`.

### From Hub Methods

When a hub method triggers a broadcast (e.g., typing indicator):

```csharp
public async Task StartTyping(Guid roomId)
{
    var userId = Context.UserIdentifier!;

    // Broadcast to the room, excluding the sender
    await Clients.GroupExcept(roomId.ToString(), Context.ConnectionId)
        .SendAsync("chat-user-typing", new ChatUserTypingEvent
        {
            EventId = Guid.NewGuid().ToString("N"),
            UserId = Guid.Parse(userId),
            RoomId = roomId,
            Timestamp = DateTimeOffset.UtcNow
        });
}
```

## Event Name Constants

Avoid magic strings by centralizing event names:

```csharp
// Contracts/SocketEvents.cs
namespace ExampleApp.Socket.Contracts;

/// <summary>
/// Central registry of all Socket event names.
/// Every event sent or received must be defined here.
/// </summary>
public static class SocketEvents
{
    // Task
    public const string TaskStarted = "task-started";
    public const string TaskCompleted = "task-completed";
    public const string TaskStepCounted = "task-step-counted";

    // Chat
    public const string ChatMessageReceived = "chat-message-received";
    public const string ChatUserTyping = "chat-user-typing";

    // Notification
    public const string NotificationReceived = "notification-received";
    public const string NotificationDismissed = "notification-dismissed";

    // Presence
    public const string PresenceChanged = "presence-changed";

    // System
    public const string RateLimitExceeded = "rate-limit-exceeded";
}
```

Usage:

```csharp
await Clients.User(userId).SendAsync(SocketEvents.TaskCompleted, payload);
```

## Checklist for Adding a New Event

1. Define the event record in `Contracts/Events/{Feature}/`.
2. Add the event name constant to `SocketEvents.cs`.
3. Add the event to the Event Catalog table above.
4. Use the constant when calling `SendAsync`.
5. Inform the Flutter/React team about the new event and its payload shape.
6. Update rate limiting rules if the event is client-to-server.
