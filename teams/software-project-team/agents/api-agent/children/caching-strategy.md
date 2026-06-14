---
knowledge-base-summary: "Cache-Aside pattern as a Mediator pipeline behavior. Query implements `ICacheable` → CachingBehavior checks Redis → handler runs only on cache miss. TTL read from dynamic settings. Cache invalidation via `ICacheInvalidator` on commands — deklarative, handler stays clean."
---
# Caching Strategy: Pipeline Behavior + Redis

## Philosophy

Cache-Aside pattern, implemented as a Mediator pipeline behavior. The handler is unaware of the cache — adding the `ICacheable` interface to the query is sufficient. Cache behavior is declarative.

## ICacheable Interface

```csharp
public interface ICacheable
{
    string CacheKey { get; }
    string? CacheTtlSettingKey { get; }  // dynamic settings key (optional)
}
```

If `CacheTtlSettingKey` is provided, the TTL is read from dynamic settings. If not provided, the default TTL is used (`cache:default-ttl-minutes` settings key).

## Usage in Query Definition

```csharp
// Simple — default TTL
record GetProductQuery(Guid ProductId) : IRequest<ProductDto>, ICacheable
{
    public string CacheKey => $"product:{ProductId}";
    public string? CacheTtlSettingKey => "cache:product:ttl-minutes";
}

// When TTL is not specified, default is used
record GetCategoriesQuery() : IRequest<List<CategoryDto>>, ICacheable
{
    public string CacheKey => "categories:all";
    public string? CacheTtlSettingKey => null;  // default TTL is used
}
```

## CachingBehavior (Pipeline)

```csharp
public sealed class CachingBehavior<TRequest, TResponse>
    : IPipelineBehavior<TRequest, TResponse>
    where TRequest : ICacheable
{
    private readonly IConnectionMultiplexer _redis;
    private readonly ISettingsService _settings;
    private readonly ILogger<CachingBehavior<TRequest, TResponse>> _logger;

    public async ValueTask<TResponse> Handle(
        TRequest request,
        MessageHandlerDelegate<TRequest, TResponse> next,
        CancellationToken ct)
    {
        var cacheKey = request.CacheKey;
        var redisDb = _redis.GetDatabase();

        // 1. Is it in cache?
        var cached = await redisDb.StringGetAsync(cacheKey);
        if (cached.HasValue)
        {
            _logger.LogInformation("Cache HIT for {CacheKey}", cacheKey);
            return JsonSerializer.Deserialize<TResponse>(cached!)!;
        }

        // 2. Run the handler
        _logger.LogInformation("Cache MISS for {CacheKey}", cacheKey);
        var response = await next(request, ct);

        // 3. Determine TTL (from dynamic settings or default)
        var ttlMinutes = request.CacheTtlSettingKey != null
            ? await _settings.GetAsync<int>(request.CacheTtlSettingKey, 10, ct)
            : await _settings.GetAsync<int>("cache:default-ttl-minutes", 10, ct);

        // 4. Write to cache
        await redisDb.StringSetAsync(
            cacheKey,
            JsonSerializer.Serialize(response),
            TimeSpan.FromMinutes(ttlMinutes));

        _logger.LogInformation(
            "Cached {CacheKey} for {TtlMinutes} minutes", cacheKey, ttlMinutes);

        return response;
    }
}
```

## Cache Invalidation

When data changes in command handlers, the related cache keys are invalidated. Declarative via the `ICacheInvalidator` interface:

```csharp
public interface ICacheInvalidator
{
    string[] CacheKeysToInvalidate { get; }
}

record UpdateProductCommand(Guid ProductId, ...) 
    : IRequest<ProductDto>, ICacheInvalidator
{
    public string[] CacheKeysToInvalidate => 
        [$"product:{ProductId}", "categories:all"];
}
```

`CacheInvalidationBehavior`:

```csharp
public sealed class CacheInvalidationBehavior<TRequest, TResponse>
    : IPipelineBehavior<TRequest, TResponse>
    where TRequest : ICacheInvalidator
{
    public async ValueTask<TResponse> Handle(
        TRequest request,
        MessageHandlerDelegate<TRequest, TResponse> next,
        CancellationToken ct)
    {
        var response = await next(request, ct);

        // Invalidate after successful command
        var redisDb = _redis.GetDatabase();
        foreach (var key in request.CacheKeysToInvalidate)
        {
            if (key.Contains('*'))
            {
                // Pattern-based invalidation (SCAN + DEL)
                await InvalidatePatternAsync(redisDb, key);
            }
            else
            {
                await redisDb.KeyDeleteAsync(key);
            }
        }

        _logger.LogInformation(
            "Cache invalidated: {Keys}", 
            string.Join(", ", request.CacheKeysToInvalidate));

        return response;
    }
}
```

## Pipeline Behavior Order (Updated)

```
Logging → UnhandledException → Validation → Caching → Performance
                                              ↑
                                    (cache check on queries)

Logging → UnhandledException → Validation → CacheInvalidation → Performance
                                              ↑
                                    (cache clearing on commands)
```

## What to Cache, What Not to Cache

**Cache:**
- Frequently read, rarely changed: product list, category tree, settings
- Expensive to compute: dashboard statistics, reports
- External API responses: exchange rates, shipping prices

**Don't cache:**
- User-specific, frequently changing: cart, notifications, unread messages
- Security-critical: authorization checks, token validation
- Already fast: single row ID lookup (FindAsync)

## Redis Key Convention

```
cache:product:{id}                → single product
cache:products:list:{cursor}      → paginated product list
cache:categories:all              → all categories
cache:dashboard:stats:{date}      → daily statistics
settings:{key}                    → dynamic settings (separate prefix)
```

