---
knowledge-base-summary: "Malicious client can call hub methods thousands of times per second. Throttling on hub method invocations. Per-user, per-method limits. Exceeding → disconnect or ignore."
---
# Rate Limiting

## The Problem

A malicious or buggy client can call hub methods thousands of times per second. Without throttling:

- Server CPU spikes processing garbage requests.
- Redis / API get hammered with forwarded calls.
- Other users experience degraded real-time performance.
- A single bad actor can bring down the entire Socket service.

SignalR does not have built-in rate limiting. You must implement it yourself.

## Architecture

```
Client                    Hub Filter                    Hub Method
  |                          |                              |
  |-- SendMessage ---------> |                              |
  |                          |-- Check Redis counter -----> |
  |                          |   INCR user:method:window    |
  |                          |                              |
  |                          |-- Counter <= limit?          |
  |                          |   YES -> proceed ----------> | Execute method
  |                          |   NO  -> reject              |
  |<-- rate-limit-exceeded --|                              |
```

## Redis-Based Rate Limiter

### Rate Limit Store

```csharp
// Services/IRateLimiter.cs
namespace ExampleApp.Socket.Services;

public interface IHubRateLimiter
{
    /// <summary>
    /// Returns true if the request is allowed, false if rate limit exceeded.
    /// </summary>
    Task<bool> IsAllowedAsync(string userId, string methodName);
}
```

```csharp
// Services/RedisHubRateLimiter.cs
using StackExchange.Redis;

namespace ExampleApp.Socket.Services;

public sealed class RedisHubRateLimiter : IHubRateLimiter
{
    private readonly IDatabase _redis;
    private readonly ILogger<RedisHubRateLimiter> _logger;
    private readonly Dictionary<string, RateLimitRule> _rules;

    public RedisHubRateLimiter(
        IConnectionMultiplexer redis,
        ILogger<RedisHubRateLimiter> logger)
    {
        _redis = redis.GetDatabase();
        _logger = logger;

        // Per-method rate limits
        _rules = new Dictionary<string, RateLimitRule>(StringComparer.OrdinalIgnoreCase)
        {
            ["SendMessage"]    = new(MaxRequests: 10, WindowSeconds: 1),   // 10/sec
            ["StartTyping"]    = new(MaxRequests: 5,  WindowSeconds: 1),   // 5/sec
            ["JoinRoom"]       = new(MaxRequests: 3,  WindowSeconds: 10),  // 3 per 10 sec
            ["LeaveRoom"]      = new(MaxRequests: 3,  WindowSeconds: 10),  // 3 per 10 sec
        };
    }

    /// <summary>
    /// Default rule for methods not explicitly configured.
    /// </summary>
    private static readonly RateLimitRule DefaultRule = new(MaxRequests: 20, WindowSeconds: 1);

    public async Task<bool> IsAllowedAsync(string userId, string methodName)
    {
        var rule = _rules.GetValueOrDefault(methodName, DefaultRule);

        // Sliding window key: rate:{userId}:{method}:{windowSlot}
        var windowSlot = DateTimeOffset.UtcNow.ToUnixTimeSeconds() / rule.WindowSeconds;
        var key = $"rate:{userId}:{methodName}:{windowSlot}";

        var count = await _redis.StringIncrementAsync(key);

        if (count == 1)
        {
            // First request in this window -- set expiry
            await _redis.KeyExpireAsync(key, TimeSpan.FromSeconds(rule.WindowSeconds * 2));
        }

        if (count > rule.MaxRequests)
        {
            _logger.LogWarning(
                "Rate limit exceeded: User {UserId}, Method {Method}, Count {Count}/{Max} in {Window}s window",
                userId, methodName, count, rule.MaxRequests, rule.WindowSeconds);
            return false;
        }

        return true;
    }
}

public sealed record RateLimitRule(int MaxRequests, int WindowSeconds);
```

## Hub Filter (SignalR Pipeline)

SignalR supports hub filters (similar to ASP.NET action filters). This is the cleanest way to intercept every hub method invocation.

```csharp
// Filters/RateLimitHubFilter.cs
using Microsoft.AspNetCore.SignalR;

namespace ExampleApp.Socket.Filters;

public sealed class RateLimitHubFilter : IHubFilter
{
    private readonly IHubRateLimiter _rateLimiter;
    private readonly ILogger<RateLimitHubFilter> _logger;

    public RateLimitHubFilter(
        IHubRateLimiter rateLimiter,
        ILogger<RateLimitHubFilter> logger)
    {
        _rateLimiter = rateLimiter;
        _logger = logger;
    }

    public async ValueTask<object?> InvokeMethodAsync(
        HubInvocationContext invocationContext,
        Func<HubInvocationContext, ValueTask<object?>> next)
    {
        var userId = invocationContext.Context.UserIdentifier;
        var methodName = invocationContext.HubMethodName;

        if (string.IsNullOrEmpty(userId))
        {
            // Anonymous connections should not reach [Authorize] hubs,
            // but reject defensively
            throw new HubException("Authentication required");
        }

        var allowed = await _rateLimiter.IsAllowedAsync(userId, methodName);

        if (!allowed)
        {
            // Notify the client that they hit the limit
            await invocationContext.Hub.Clients
                .Client(invocationContext.Context.ConnectionId)
                .SendAsync("rate-limit-exceeded", new
                {
                    method = methodName,
                    message = "Too many requests. Please slow down.",
                    timestamp = DateTimeOffset.UtcNow
                });

            _logger.LogWarning(
                "Blocked hub call: User {UserId}, Method {Method}, ConnectionId {ConnectionId}",
                userId, methodName, invocationContext.Context.ConnectionId);

            // Return null to prevent the method from executing
            return null;
        }

        return await next(invocationContext);
    }
}
```

## Registration in Program.cs

```csharp
// Program.cs

// Register Redis
builder.Services.AddSingleton<IConnectionMultiplexer>(
    ConnectionMultiplexer.Connect(
        builder.Configuration.GetConnectionString("Redis") ?? "redis:6379"));

// Register rate limiter
builder.Services.AddSingleton<IHubRateLimiter, RedisHubRateLimiter>();

// Register SignalR with hub filter
builder.Services.AddSignalR(options =>
{
    options.AddFilter<RateLimitHubFilter>();
});
```

## Escalation: Disconnect Abusive Clients

For persistent abusers, counting violations and disconnecting is appropriate:

```csharp
// Filters/RateLimitHubFilter.cs -- escalation variant
public async ValueTask<object?> InvokeMethodAsync(
    HubInvocationContext invocationContext,
    Func<HubInvocationContext, ValueTask<object?>> next)
{
    var userId = invocationContext.Context.UserIdentifier!;
    var methodName = invocationContext.HubMethodName;

    var allowed = await _rateLimiter.IsAllowedAsync(userId, methodName);

    if (!allowed)
    {
        // Track consecutive violations
        var violationKey = $"rate:violations:{userId}";
        var violations = await _redis.StringIncrementAsync(violationKey);
        await _redis.KeyExpireAsync(violationKey, TimeSpan.FromMinutes(5));

        if (violations >= 10)
        {
            // 10 violations in 5 minutes -- force disconnect
            _logger.LogWarning(
                "Force disconnecting abusive client: User {UserId}, Violations {Count}",
                userId, violations);

            invocationContext.Context.Abort();
            return null;
        }

        await invocationContext.Hub.Clients
            .Client(invocationContext.Context.ConnectionId)
            .SendAsync("rate-limit-exceeded", new
            {
                method = methodName,
                message = "Too many requests. Please slow down.",
                violationCount = violations,
                timestamp = DateTimeOffset.UtcNow
            });

        return null;
    }

    return await next(invocationContext);
}
```

## Alternative: In-Memory Rate Limiter (No Redis)

If Redis is not available, use an in-memory counter. This only works for single-instance deployments:

```csharp
// Services/InMemoryHubRateLimiter.cs
using System.Collections.Concurrent;

namespace ExampleApp.Socket.Services;

public sealed class InMemoryHubRateLimiter : IHubRateLimiter
{
    private readonly ConcurrentDictionary<string, (int Count, long WindowSlot)> _counters = new();

    public Task<bool> IsAllowedAsync(string userId, string methodName)
    {
        var windowSlot = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
        var key = $"{userId}:{methodName}";

        var result = _counters.AddOrUpdate(
            key,
            _ => (1, windowSlot),
            (_, existing) =>
            {
                if (existing.WindowSlot != windowSlot)
                    return (1, windowSlot); // New window
                return (existing.Count + 1, windowSlot);
            });

        return Task.FromResult(result.Count <= 20); // 20/sec default
    }
}
```

**Warning:** In-memory counters are lost on restart and do not work across multiple Socket instances. Use Redis for production.

## Rate Limit Configuration

For production, externalize rate limits to configuration:

```json
// appsettings.json
{
  "RateLimits": {
    "SendMessage": { "MaxRequests": 10, "WindowSeconds": 1 },
    "StartTyping": { "MaxRequests": 5, "WindowSeconds": 1 },
    "JoinRoom": { "MaxRequests": 3, "WindowSeconds": 10 },
    "Default": { "MaxRequests": 20, "WindowSeconds": 1 }
  }
}
```

```csharp
// Bind in Program.cs
var rateLimitConfig = builder.Configuration
    .GetSection("RateLimits")
    .Get<Dictionary<string, RateLimitRule>>()
    ?? new();

builder.Services.AddSingleton(rateLimitConfig);
```

## Client-Side Handling

The client should listen for the `rate-limit-exceeded` event and implement exponential backoff:

```
on("rate-limit-exceeded"):
    show warning toast
    disable send button for 2 seconds
    if violationCount >= 5:
        show persistent warning
        disable send for 30 seconds
```

## What NOT to Rate Limit

- `OnConnectedAsync` / `OnDisconnectedAsync` -- these are lifecycle events, not hub method calls. Rate limiting connections is done at the reverse proxy level (nginx, Cloudflare).
- Internal broadcast endpoints -- these are server-to-server and already protected by X-Internal-Token.
- Heartbeat / ping methods -- these are infrastructure, not user actions.

## Key Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| Counter backend | Redis | Multi-instance support |
| Window type | Fixed window (second-aligned) | Simple, predictable, good enough |
| Enforcement point | Hub filter | Intercepts all methods uniformly |
| Violation response | Send error event | Client can react gracefully |
| Escalation | Disconnect after 10 violations | Stops persistent abuse |
