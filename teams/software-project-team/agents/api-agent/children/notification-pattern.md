---
knowledge-base-summary: "Two methods — choose based on decision table. HTTP (instant): most cases, single user/group/all, 5-10ms. RMQ (async): batch notifications, message must not be lost, handler shouldn't slow down. `INotificationService` for HTTP, `INotificationPublisher` for RMQ. Check the decision table for every handler that needs notifications."
---
# Notification Pattern: API → Socket Broadcast

## Two Methods EXIST — Choose the Right One Based on the Situation

In this pattern, there are two different methods. When writing each handler, if notification is needed, **look at the decision table and choose the right method**. Never use one as the default — evaluate each case.

### Decision Table

| Situation | Method | Why |
|-----------|--------|-----|
| Notification to a single user ("your order has been confirmed") | **HTTP (Instant)** | Fast, simple, user sees it immediately |
| Notification to a group ("new order received" → admins) | **HTTP (Instant)** | Group is small, should see immediately |
| Broadcast to everyone ("maintenance mode starting") | **HTTP (Instant)** | Urgent, everyone should see immediately |
| Multiple notifications from a single handler (in a loop) | **RMQ (Async)** | Handler shouldn't slow down, shouldn't make N HTTP calls |
| Notification after batch operation ("100 products imported") | **RMQ (Async)** | Batch operation, handler is already long-running |
| Notification failure is not critical (best-effort) | **RMQ (Async)** | Safe with retry/DLX, doesn't block the handler |
| Notification cannot be lost (e.g., payment confirmation) | **RMQ (Async)** | RMQ is durable, won't be lost, waits in queue even if Socket is down |

**Rule:** When in doubt, use HTTP (Instant) — sufficient and simple for most cases. Only switch to RMQ in the specific situations listed in the table above.

---

## Method 1: HTTP (Instant Notification)

The handler makes a direct HTTP call to Socket. 5-10ms overhead.

### INotificationService (Application layer)

```csharp
public interface INotificationService
{
    // To a single user
    Task SendToUserAsync(
        string eventName,
        object data,
        string userId,
        CancellationToken ct = default);

    // To a group (role-based)
    Task SendToGroupAsync(
        string eventName,
        object data,
        string groupName,
        CancellationToken ct = default);

    // To everyone
    Task BroadcastAsync(
        string eventName,
        object data,
        CancellationToken ct = default);
}
```

### Implementation (Infrastructure layer)

```csharp
public class HttpNotificationService : INotificationService
{
    private readonly HttpClient _httpClient; // directed to Socket, X-Internal-Token injected

    public async Task SendToUserAsync(
        string eventName, object data, string userId, CancellationToken ct)
    {
        await _httpClient.PostAsJsonAsync("/api/internal/broadcast", new
        {
            Event = eventName,
            Data = data,
            TargetUserId = userId,
            TargetType = "user",
        }, ct);
    }

    public async Task SendToGroupAsync(
        string eventName, object data, string groupName, CancellationToken ct)
    {
        await _httpClient.PostAsJsonAsync("/api/internal/broadcast", new
        {
            Event = eventName,
            Data = data,
            TargetGroup = groupName,
            TargetType = "group",
        }, ct);
    }

    public async Task BroadcastAsync(
        string eventName, object data, CancellationToken ct)
    {
        await _httpClient.PostAsJsonAsync("/api/internal/broadcast", new
        {
            Event = eventName,
            Data = data,
            TargetType = "all",
        }, ct);
    }
}
```

### Usage in Handler

```csharp
// Order confirmed → notify the user
await _notificationService.SendToUserAsync(
    "order-confirmed",
    new { OrderId = order.Id, Total = order.Total },
    order.CustomerId.ToString(),
    ct);

_logger.LogInformation(
    "Notification sent: order-confirmed to user {UserId} for order {OrderId}",
    order.CustomerId, order.Id);
```

---

## Method 2: RMQ (Async Notification)

The handler publishes an event to RMQ, Socket consumes and broadcasts it. Handler does not slow down.

### NotificationEvent (Application/Common/)

```csharp
public record NotificationEvent
{
    public string EventName { get; init; } = null!;
    public object Data { get; init; } = null!;
    public string TargetType { get; init; } = "all";  // "user" | "group" | "all"
    public string? TargetUserId { get; init; }
    public string? TargetGroup { get; init; }
    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
}
```

### INotificationPublisher (Application layer)

```csharp
public interface INotificationPublisher
{
    Task PublishAsync(NotificationEvent notification, CancellationToken ct = default);
    Task PublishManyAsync(IEnumerable<NotificationEvent> notifications, CancellationToken ct = default);
}
```

### Implementation (Infrastructure layer)

```csharp
public class RmqNotificationPublisher : INotificationPublisher
{
    private readonly IRabbitMqConnection _connection;

    public async Task PublishAsync(NotificationEvent notification, CancellationToken ct)
    {
        // Publish to the notifications.fanout exchange
        // On the Socket side, the consumer listens to this queue and broadcasts via Hub
    }

    public async Task PublishManyAsync(
        IEnumerable<NotificationEvent> notifications, CancellationToken ct)
    {
        // Batch publish — each one is sent as a separate message to the queue
    }
}
```

### Usage in Handler (Batch/Async Case)

```csharp
// 100 products imported → notify each product owner
var notifications = importedProducts.Select(p => new NotificationEvent
{
    EventName = "product-imported",
    Data = new { ProductId = p.Id, ProductName = p.Name },
    TargetType = "user",
    TargetUserId = p.OwnerId.ToString(),
}).ToList();

await _notificationPublisher.PublishManyAsync(notifications, ct);

_logger.LogInformation(
    "Published {Count} product-imported notifications via RMQ",
    notifications.Count);
```

---

## RMQ Topology

```
Exchange: notifications.fanout (fanout, durable)
Queue: notifications.socket (durable)
Binding: notifications.fanout → notifications.socket
```

The Socket project consumes this queue and delivers messages to clients via Hub.

---

## Socket Side (InternalEndpoints)

The `/api/internal/broadcast` endpoint in the Socket project:

```csharp
// For the HTTP method:
app.MapPost("/api/internal/broadcast", async (
    BroadcastRequest request,
    IHubContext<NotificationHub> hubContext) =>
{
    switch (request.TargetType)
    {
        case "user":
            await hubContext.Clients
                .User(request.TargetUserId!)
                .SendAsync(request.Event, request.Data);
            break;
        case "group":
            await hubContext.Clients
                .Group(request.TargetGroup!)
                .SendAsync(request.Event, request.Data);
            break;
        case "all":
            await hubContext.Clients.All
                .SendAsync(request.Event, request.Data);
            break;
    }
    return Results.Ok();
})
.AddEndpointFilter<InternalSecretFilter>();
```

---

## Handler Writing Checklist

Check the following for every handler that requires notification:

- [ ] Is notification needed? (not every handler requires it)
- [ ] Look at the decision table: HTTP or RMQ?
- [ ] Single user / group / everyone?
- [ ] Event name: kebab-case (`order-confirmed`, `product-imported`)
- [ ] Data: only necessary fields (not the entire entity, think of it like a DTO)
- [ ] Log: notification sent/published log
- [ ] Sensitive data: no password, token, etc. in notification data

---
