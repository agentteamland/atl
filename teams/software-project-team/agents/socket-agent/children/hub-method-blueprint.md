---
knowledge-base-summary: "The primary production unit of this agent. Template + checklist + naming conventions for creating new hub methods. Read this FIRST when adding any new hub method. Covers the full lifecycle: define request → write method → define events → register → test."
---
# Hub Method Blueprint

## Overview

This is the blueprint for creating new hub methods. Every hub method follows this template and checklist. When you need to add a new hub method, read this file and follow it step by step.

## Template

```csharp
// In Hubs/NotificationHub.cs (or feature-specific hub)

/// <summary>
/// {Brief description of what this method does}
/// </summary>
public async Task {MethodName}({RequestType} request)
{
    var userId = Context.UserIdentifier;
    _logger.LogInformation(
        "{MethodName} called by user {UserId} with {@Request}",
        nameof({MethodName}), userId, request);

    try
    {
        // 1. Input validation (if needed)
        if (string.IsNullOrEmpty(request.SomeField))
        {
            throw new HubException(JsonSerializer.Serialize(new
            {
                code = "validation-error",
                message = "SomeField is required"
            }));
        }

        // 2. Call API via IApiClient (the bridge)
        var result = await _apiClient.{ApiMethod}(request, Context.ConnectionAborted);

        // 3. Return result to caller (if request-response)
        await Clients.Caller.SendAsync("{response-event-name}", result, Context.ConnectionAborted);

        // 4. Broadcast to others (if needed)
        await Clients.Group("tenant:{tenantId}").SendAsync("{broadcast-event-name}", new
        {
            // minimal payload — just enough for UI to react
        }, Context.ConnectionAborted);

        _logger.LogInformation(
            "{MethodName} completed for user {UserId}",
            nameof({MethodName}), userId);
    }
    catch (HubException)
    {
        throw; // already structured, let it pass
    }
    catch (Exception ex)
    {
        _logger.LogError(ex, "{MethodName} failed for user {UserId}", nameof({MethodName}), userId);
        throw new HubException(JsonSerializer.Serialize(new
        {
            code = "internal-error",
            message = "An unexpected error occurred"
        }));
    }
}
```

## Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Hub method | PascalCase, verb-first | `SendMessage`, `JoinRoom`, `GetUnreadCount` |
| Response event | kebab-case, past tense | `message-sent`, `room-joined`, `unread-count-updated` |
| Broadcast event | kebab-case, past tense | `message-received`, `user-joined-room` |
| Request type | PascalCase + Request suffix | `SendMessageRequest`, `JoinRoomRequest` |

## Lifecycle

1. **Define request type** — create a record for the input
2. **Write hub method** — follow the template above
3. **Define events** — add event names to the event catalog (event-conventions.md)
4. **Register** — if new hub, map in Program.cs
5. **Test** — call from client, verify logs, verify broadcast reaches targets

## Checklist

Before considering a hub method complete, verify:

- [ ] **Bridge only:** Method contains no business logic — only calls IApiClient and broadcasts results
- [ ] **Authorization:** Hub has `[Authorize]` attribute (or method-level if mixed auth)
- [ ] **Input validation:** Required fields checked, invalid input → structured HubException
- [ ] **API call:** Uses `_apiClient` for all data/logic operations
- [ ] **CancellationToken:** `Context.ConnectionAborted` passed to all async calls
- [ ] **Error handling:** Catches exceptions, wraps in structured HubException
- [ ] **Logging:** Start log with parameters, end log with result, error log with exception
- [ ] **Event naming:** Follows kebab-case convention, documented in event catalog
- [ ] **Broadcast target:** Correct target used (Caller, User, Group, All) — see decision table:

| Scenario | Target | Method |
|----------|--------|--------|
| Return result to sender | Caller only | `Clients.Caller.SendAsync()` |
| Notify specific user (all devices) | User | `Clients.User(userId).SendAsync()` |
| Notify room/tenant | Group | `Clients.Group(name).SendAsync()` |
| Notify room except sender | Group except | `Clients.GroupExcept(name, [connId]).SendAsync()` |
| Notify everyone | All | `Clients.All.SendAsync()` |

- [ ] **Rate limit:** Method added to rate limit config if it's user-callable (see rate-limiting.md)
- [ ] **Payload size:** Response/broadcast payload is minimal — IDs + summary, not full entities

## Anti-Patterns

```csharp
// ❌ WRONG: Business logic in hub method
public async Task CreateOrder(CreateOrderRequest request)
{
    var order = new Order { ... };
    _dbContext.Orders.Add(order);  // NO — hub has no DbContext
    await _dbContext.SaveChangesAsync();
}

// ❌ WRONG: No error handling
public async Task SendMessage(SendMessageRequest request)
{
    var result = await _apiClient.SendMessageAsync(request); // exception = opaque error to client
    await Clients.Group(request.RoomId).SendAsync("message-received", result);
}

// ✅ CORRECT: Bridge pattern with full lifecycle
public async Task SendMessage(SendMessageRequest request)
{
    _logger.LogInformation("SendMessage called by {UserId} in room {RoomId}",
        Context.UserIdentifier, request.RoomId);
    try
    {
        var result = await _apiClient.SendMessageAsync(request, Context.ConnectionAborted);
        await Clients.Caller.SendAsync("message-sent", result, Context.ConnectionAborted);
        await Clients.GroupExcept($"room:{request.RoomId}", new[] { Context.ConnectionId })
            .SendAsync("message-received", result, Context.ConnectionAborted);
        _logger.LogInformation("Message sent in room {RoomId}", request.RoomId);
    }
    catch (Exception ex)
    {
        _logger.LogError(ex, "SendMessage failed for {UserId}", Context.UserIdentifier);
        throw new HubException(JsonSerializer.Serialize(new { code = "send-failed", message = "Failed to send message" }));
    }
}
```
