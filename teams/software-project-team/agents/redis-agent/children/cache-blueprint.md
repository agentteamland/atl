---
knowledge-base-summary: "Primary blueprint for implementing cache. Cache-aside pattern with Redis as the cache layer. When to cache (read-heavy, rarely changing data) vs when not to (user-specific, frequently changing). ICacheable interface on queries, CachingBehavior as Mediator pipeline, TTL from dynamic settings, invalidation via ICacheInvalidator on commands."
---
# Cache Blueprint

This is the primary blueprint for implementing caching in the project. Every cache implementation follows this pattern.

## When to Cache vs When Not to Cache

### Cache (green light)
- **Read-heavy, write-light data** — product catalogs, settings, lookup tables
- **Expensive computations** — aggregations, statistics, report summaries
- **Shared across users** — global configuration, feature flags, public content
- **Rarely changing** — data that updates less than once per minute

### Do NOT Cache (red light)
- **User-specific data that changes frequently** — shopping cart mid-checkout, active form state
- **Data that must be real-time consistent** — account balance, inventory count during purchase
- **Large result sets** — full table dumps, unbounded lists
- **Data with strict regulatory requirements** — if caching violates data residency rules

### Gray Area (cache with short TTL)
- **User profiles** — cache for 5-15 minutes, invalidate on update
- **Search results** — cache for 1-5 minutes for identical queries
- **Permissions/roles** — cache for 5-10 minutes, invalidate on role change

## Cache-Aside Pattern

The standard pattern. Application checks cache first, falls back to database on miss, writes to cache after fetch.

```
Request → CachingBehavior → [Cache Hit?] → Yes → Return cached
                                         → No  → Handler → DB → Write to cache → Return
```

### Never Use Cache-Through or Write-Behind
We explicitly use cache-aside. The application owns both the read and write path. Redis is never the source of truth.

## ICacheable Interface

Queries that want caching implement this interface in the Application layer:

```csharp
// Application/Interfaces/ICacheable.cs
public interface ICacheable
{
    string CacheKey { get; }
    TimeSpan? CacheDuration { get; } // null = use default from settings
}
```

### Query Implementation

```csharp
// Application/Features/Products/Queries/GetProduct/GetProductQuery.cs
public sealed record GetProductQuery(Guid Id) : IQuery<ProductResponse>, ICacheable
{
    public string CacheKey => $"cache:product:{Id}";
    public TimeSpan? CacheDuration => null; // uses default from settings
}
```

### Custom TTL Override

```csharp
public sealed record GetDashboardStatsQuery : IQuery<DashboardStatsResponse>, ICacheable
{
    public string CacheKey => "cache:dashboard:stats";
    public TimeSpan? CacheDuration => TimeSpan.FromMinutes(5); // override default
}
```

## CachingBehavior (Mediator Pipeline)

This behavior intercepts all queries that implement `ICacheable`. It runs BEFORE the handler.

```csharp
// Application/Behaviors/CachingBehavior.cs
public sealed class CachingBehavior<TRequest, TResponse> : IPipelineBehavior<TRequest, TResponse>
    where TRequest : ICacheable
{
    private readonly IDatabase _redis;
    private readonly ISettingsService _settings;
    private readonly ILogger<CachingBehavior<TRequest, TResponse>> _logger;

    public CachingBehavior(
        IConnectionMultiplexer redis,
        ISettingsService settings,
        ILogger<CachingBehavior<TRequest, TResponse>> logger)
    {
        _redis = redis.GetDatabase();
        _settings = settings;
        _logger = logger;
    }

    public async ValueTask<TResponse> Handle(
        TRequest request,
        CancellationToken ct,
        MessageHandlerDelegate<TRequest, TResponse> next)
    {
        var cacheKey = request.CacheKey;

        // Try cache first
        try
        {
            var cached = await _redis.StringGetAsync(cacheKey);
            if (cached.HasValue)
            {
                _logger.LogDebug("Cache HIT for {CacheKey}", cacheKey);
                return JsonSerializer.Deserialize<TResponse>(cached!)!;
            }
        }
        catch (RedisConnectionException ex)
        {
            // Redis down — degrade gracefully, proceed to handler
            _logger.LogWarning(ex, "Redis unavailable, skipping cache for {CacheKey}", cacheKey);
        }

        _logger.LogDebug("Cache MISS for {CacheKey}", cacheKey);

        // Execute handler
        var response = await next(request, ct);

        // Write to cache
        try
        {
            var ttl = request.CacheDuration
                ?? await _settings.GetAsync<TimeSpan>("Cache:DefaultTtl", ct)
                ?? TimeSpan.FromMinutes(30);

            var serialized = JsonSerializer.Serialize(response);
            await _redis.StringSetAsync(cacheKey, serialized, ttl);

            _logger.LogDebug("Cached {CacheKey} with TTL {Ttl}", cacheKey, ttl);
        }
        catch (RedisConnectionException ex)
        {
            _logger.LogWarning(ex, "Redis unavailable, could not cache {CacheKey}", cacheKey);
        }

        return response;
    }
}
```

### Key Points
- **try-catch around every Redis call** — Redis being down never crashes the request
- **TTL resolution order**: query override -> dynamic settings -> hardcoded fallback (30 min)
- **Serialization**: System.Text.Json, not Newtonsoft

## Cache Invalidation

### ICacheInvalidator Interface

Commands that modify data implement this to declare which cache keys to invalidate:

```csharp
// Application/Interfaces/ICacheInvalidator.cs
public interface ICacheInvalidator
{
    IEnumerable<string> CacheKeysToInvalidate { get; }
}
```

### Command Implementation

```csharp
// Application/Features/Products/Commands/UpdateProduct/UpdateProductCommand.cs
public sealed record UpdateProductCommand(
    Guid Id,
    string Name,
    decimal Price
) : ICommand<ProductResponse>, ICacheInvalidator
{
    public IEnumerable<string> CacheKeysToInvalidate =>
    [
        $"cache:product:{Id}",
        "cache:products:list:*"  // wildcard — invalidates all product list caches
    ];
}
```

### CacheInvalidationBehavior

```csharp
// Application/Behaviors/CacheInvalidationBehavior.cs
public sealed class CacheInvalidationBehavior<TRequest, TResponse> : IPipelineBehavior<TRequest, TResponse>
    where TRequest : ICacheInvalidator
{
    private readonly IDatabase _redis;
    private readonly IConnectionMultiplexer _multiplexer;
    private readonly ILogger<CacheInvalidationBehavior<TRequest, TResponse>> _logger;

    public CacheInvalidationBehavior(
        IConnectionMultiplexer multiplexer,
        ILogger<CacheInvalidationBehavior<TRequest, TResponse>> logger)
    {
        _redis = multiplexer.GetDatabase();
        _multiplexer = multiplexer;
        _logger = logger;
    }

    public async ValueTask<TResponse> Handle(
        TRequest request,
        CancellationToken ct,
        MessageHandlerDelegate<TRequest, TResponse> next)
    {
        // Execute handler first — only invalidate on success
        var response = await next(request, ct);

        // Invalidate cache keys
        try
        {
            foreach (var key in request.CacheKeysToInvalidate)
            {
                if (key.Contains('*'))
                {
                    // Wildcard — use SCAN to find matching keys
                    var server = _multiplexer.GetServer(_multiplexer.GetEndPoints().First());
                    await foreach (var matchedKey in server.KeysAsync(pattern: key))
                    {
                        await _redis.KeyDeleteAsync(matchedKey);
                        _logger.LogDebug("Invalidated cache key {Key} (pattern: {Pattern})", matchedKey, key);
                    }
                }
                else
                {
                    await _redis.KeyDeleteAsync(key);
                    _logger.LogDebug("Invalidated cache key {Key}", key);
                }
            }
        }
        catch (RedisConnectionException ex)
        {
            _logger.LogWarning(ex, "Redis unavailable during cache invalidation");
            // Not fatal — cache will expire via TTL
        }

        return response;
    }
}
```

### Invalidation Rules

1. **Invalidation runs AFTER the handler succeeds** — never invalidate on failure
2. **Wildcard patterns use SCAN** — never KEYS in production
3. **Failure to invalidate is NOT fatal** — TTL will handle eventual expiry
4. **Be specific** — invalidate only what changed, not the entire cache namespace

## Checklist for Adding Cache to a Query

1. Does this query benefit from caching? (read-heavy, shared, not user-specific)
2. Add `ICacheable` to the query record
3. Define `CacheKey` using the `cache:{entity}:{id}` pattern
4. Set `CacheDuration` (null for default, or explicit TimeSpan)
5. Identify all commands that modify this data
6. Add `ICacheInvalidator` to those commands
7. List all cache keys that should be invalidated (including wildcard patterns)
8. Test: verify cache hit/miss logging works
9. Test: verify invalidation clears the right keys
10. Test: verify behavior when Redis is down (graceful degradation)

## Anti-Patterns to Avoid

- **Caching everything** — only cache what benefits from it
- **Infinite TTL** — every cache key must expire
- **Cache stampede** — when many requests hit a cold cache simultaneously; consider cache warming for critical keys
- **Stale data tolerance undefined** — document how stale is acceptable for each cached entity
- **Serializing entities directly** — always cache DTOs/Response records, never domain entities
