---
knowledge-base-summary: "API-to-Socket broadcast channel. POST `/api/internal/broadcast` with X-Internal-Token header. Supports three target types: single user, group, all connected clients. InternalSecretFilter validates the shared secret."
---
# Internal Endpoints: API-to-Socket Broadcast

## Purpose

The API needs to push real-time events to connected clients. It cannot do this directly because SignalR hubs live in the Socket service. The solution: the API makes an HTTP POST to the Socket's internal broadcast endpoint, which then uses `IHubContext` to push the event to the appropriate clients.

```
API Handler                   Socket                        Client
    |                           |                              |
    |  POST /api/internal/      |                              |
    |  broadcast                |                              |
    |  X-Internal-Token: xxx    |                              |
    | ------------------------> |                              |
    |                           |  IHubContext.Clients         |
    |                           |  .User(userId)               |
    |                           |  .SendAsync("event", data)   |
    |                           | ---------------------------> |
    |  200 OK                   |                              |
    | <------------------------ |                              |
```

## BroadcastRequest Model

The request model supports three target types: a specific user, a group, or all connected clients.

```csharp
namespace ExampleApp.Socket.Endpoints;

public sealed record BroadcastRequest(
    string Event,              // Event name: "order-confirmed", "user-typing"
    object? Payload,           // Event data (serialized as JSON)
    string? UserId = null,     // Target user ID (for user-targeted broadcasts)
    string? Group = null,      // Target group name (for group-targeted broadcasts)
    string TargetType = "all"  // "user" | "group" | "all"
);
```

## InternalSecretFilter

The broadcast endpoint is **not** protected by JWT — it is an internal system-to-system endpoint. Instead, it validates a shared secret via the `X-Internal-Token` header.

```csharp
namespace ExampleApp.Socket.Endpoints;

public sealed class InternalSecretFilter : IEndpointFilter
{
    private readonly IConfiguration _configuration;

    public InternalSecretFilter(IConfiguration configuration)
    {
        _configuration = configuration;
    }

    public async ValueTask<object?> InvokeAsync(
        EndpointFilterInvocationContext context,
        EndpointFilterDelegate next)
    {
        var expectedToken = _configuration["InternalToken"];
        var token = context.HttpContext.Request.Headers["X-Internal-Token"]
            .FirstOrDefault();

        if (string.IsNullOrEmpty(token) || token != expectedToken)
        {
            return Results.Unauthorized();
        }

        return await next(context);
    }
}
```

The `InternalToken` value is shared between API and Socket via environment variables:

```yaml
# docker-compose.yml
services:
  api:
    environment:
      - InternalToken=${INTERNAL_TOKEN}
  socket:
    environment:
      - InternalToken=${INTERNAL_TOKEN}
```

## Broadcast Endpoint

The endpoint receives the request, validates the internal token (via filter), and routes the event to the correct target using `IHubContext`.

```csharp
using Microsoft.AspNetCore.SignalR;
using ExampleApp.Socket.Hubs;

namespace ExampleApp.Socket.Endpoints;

public static class InternalEndpoints
{
    public static void MapInternalEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/internal")
            .WithTags("Internal");

        group.MapPost("/broadcast", async (
            BroadcastRequest request,
            IHubContext<NotificationHub> hubContext,
            ILogger<NotificationHub> logger) =>
        {
            logger.LogInformation(
                "Broadcast received: event {Event}, targetType {TargetType}, "
                + "userId {UserId}, group {Group}",
                request.Event, request.TargetType,
                request.UserId, request.Group);

            switch (request.TargetType)
            {
                case "user":
                    if (string.IsNullOrEmpty(request.UserId))
                        return Results.BadRequest("UserId is required for user target");

                    await hubContext.Clients
                        .User(request.UserId)
                        .SendAsync(request.Event, request.Payload);
                    break;

                case "group":
                    if (string.IsNullOrEmpty(request.Group))
                        return Results.BadRequest("Group is required for group target");

                    await hubContext.Clients
                        .Group(request.Group)
                        .SendAsync(request.Event, request.Payload);
                    break;

                case "all":
                    await hubContext.Clients.All
                        .SendAsync(request.Event, request.Payload);
                    break;

                default:
                    return Results.BadRequest(
                        $"Unknown target type: {request.TargetType}");
            }

            logger.LogInformation(
                "Broadcast sent: event {Event} to {TargetType}",
                request.Event, request.TargetType);

            return Results.Ok(new { sent = true });
        })
        .AddEndpointFilter<InternalSecretFilter>();
    }
}
```

## Registration in Program.cs

```csharp
app.MapInternalEndpoints();
```

This is called alongside `app.MapHub<NotificationHub>(...)` in Program.cs. The internal endpoints do NOT require `app.UseAuthentication()` or `app.UseAuthorization()` — they use the `InternalSecretFilter` instead.

## How the API Calls This Endpoint

On the API side, the `INotificationService` (in Application layer) and `HttpNotificationService` (in Infrastructure layer) encapsulate the HTTP call:

```csharp
// API Infrastructure layer — calls Socket's internal broadcast endpoint
public class HttpNotificationService : INotificationService
{
    private readonly HttpClient _httpClient;
    // HttpClient is pre-configured with BaseUrl = "http://socket:3002"
    // and X-Internal-Token header via DelegatingHandler

    public async Task SendToUserAsync(
        string eventName, object data, string userId, CancellationToken ct)
    {
        await _httpClient.PostAsJsonAsync("/api/internal/broadcast", new
        {
            Event = eventName,
            Payload = data,
            UserId = userId,
            TargetType = "user",
        }, ct);
    }

    public async Task SendToGroupAsync(
        string eventName, object data, string groupName, CancellationToken ct)
    {
        await _httpClient.PostAsJsonAsync("/api/internal/broadcast", new
        {
            Event = eventName,
            Payload = data,
            Group = groupName,
            TargetType = "group",
        }, ct);
    }

    public async Task BroadcastAsync(
        string eventName, object data, CancellationToken ct)
    {
        await _httpClient.PostAsJsonAsync("/api/internal/broadcast", new
        {
            Event = eventName,
            Payload = data,
            TargetType = "all",
        }, ct);
    }
}
```

## Target Type Decision Table

| Target Type | When | IHubContext Call |
|-------------|------|-----------------|
| `"user"` | Notification for a specific user (order confirmed, message received) | `Clients.User(userId).SendAsync(...)` |
| `"group"` | Notification for a tenant, room, or role group | `Clients.Group(groupName).SendAsync(...)` |
| `"all"` | System-wide announcement (maintenance, version update) | `Clients.All.SendAsync(...)` |

## Security Rules

1. **Never expose internal endpoints publicly.** The `/api/internal/*` prefix is only accessible within the Docker network. In production, the reverse proxy (nginx/traefik) must NOT route external traffic to these endpoints.

2. **InternalToken must be strong in production.** Use a 64+ character random string. The dev default (`internal-secret-change-in-production`) is only for local development.

3. **No JWT on internal endpoints.** These are system-to-system calls — the API authenticates the user, the Socket trusts the API via the shared secret.

4. **Log every broadcast.** Every incoming broadcast request is logged with event name and target. This creates an audit trail of all real-time events pushed to clients.

## Testing

```bash
# Test broadcast to all — from within Docker network
curl -X POST http://socket:3002/api/internal/broadcast \
  -H "Content-Type: application/json" \
  -H "X-Internal-Token: internal-secret-change-in-production" \
  -d '{
    "event": "test-event",
    "payload": {"message": "hello"},
    "targetType": "all"
  }'

# Test broadcast to a specific user
curl -X POST http://socket:3002/api/internal/broadcast \
  -H "Content-Type: application/json" \
  -H "X-Internal-Token: internal-secret-change-in-production" \
  -d '{
    "event": "order-confirmed",
    "payload": {"orderId": "abc-123", "total": 42.50},
    "userId": "user-guid-here",
    "targetType": "user"
  }'
```
