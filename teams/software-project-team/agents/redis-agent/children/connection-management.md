---
knowledge-base-summary: "IConnectionMultiplexer as singleton — never per-request. Connection string configuration with timeout, retry, and keepAlive. Multiple database usage (db0-db15) for logical separation. Pipeline batching for multiple operations. Fire-and-forget flags for non-critical writes. Reconnection strategy."
---
# Connection Management

StackExchange.Redis manages connections through `IConnectionMultiplexer`. Proper configuration is critical for performance and reliability.

## IConnectionMultiplexer as Singleton

The multiplexer is designed to be shared and reused. It manages an internal pool of connections and handles reconnection automatically. **Never create per-request connections.**

### DI Registration

```csharp
// Program.cs
builder.Services.AddSingleton<IConnectionMultiplexer>(sp =>
{
    var configuration = builder.Configuration.GetConnectionString("Redis")
        ?? "localhost:6379";

    var options = ConfigurationOptions.Parse(configuration);
    options.AbortOnConnectFail = false;     // Don't throw on startup if Redis is down
    options.ConnectRetry = 3;               // Retry connection 3 times
    options.ConnectTimeout = 5000;          // 5 second connection timeout
    options.SyncTimeout = 3000;             // 3 second sync operation timeout
    options.AsyncTimeout = 5000;            // 5 second async operation timeout
    options.KeepAlive = 60;                 // Send keepalive every 60 seconds
    options.ReconnectRetryPolicy = new ExponentialRetry(5000); // Exponential backoff

    var multiplexer = ConnectionMultiplexer.Connect(options);

    multiplexer.ConnectionFailed += (sender, args) =>
    {
        var logger = sp.GetRequiredService<ILogger<IConnectionMultiplexer>>();
        logger.LogWarning(
            args.Exception,
            "Redis connection failed: {FailureType} to {EndPoint}",
            args.FailureType,
            args.EndPoint);
    };

    multiplexer.ConnectionRestored += (sender, args) =>
    {
        var logger = sp.GetRequiredService<ILogger<IConnectionMultiplexer>>();
        logger.LogInformation(
            "Redis connection restored to {EndPoint}",
            args.EndPoint);
    };

    return multiplexer;
});
```

### Connection String Format

```
# Basic
localhost:6379

# With password
localhost:6379,password=secret

# With database selection
localhost:6379,defaultDatabase=1

# Full production config
redis-server:6379,password=secret,ssl=true,abortConnect=false,connectTimeout=5000,syncTimeout=3000,asyncTimeout=5000,keepAlive=60,connectRetry=3
```

### appsettings.json

```json
{
  "ConnectionStrings": {
    "Redis": "redis:6379,abortConnect=false,connectTimeout=5000"
  }
}
```

## Getting the Database

```csharp
// Inject IConnectionMultiplexer, get IDatabase from it
public class ProductCacheService
{
    private readonly IDatabase _redis;

    public ProductCacheService(IConnectionMultiplexer multiplexer)
    {
        _redis = multiplexer.GetDatabase(); // Default: db0
    }

    public async Task<ProductResponse?> GetAsync(Guid id)
    {
        var cached = await _redis.StringGetAsync(RedisKeys.ProductCache(id));
        return cached.HasValue
            ? JsonSerializer.Deserialize<ProductResponse>(cached!)
            : null;
    }
}
```

## Multiple Databases (db0-db15)

Redis supports 16 logical databases (db0 through db15). Use them for logical separation within the same Redis instance.

### Database Allocation

| Database | Purpose | Reason |
|----------|---------|--------|
| db0 | Cache (default) | Most common operations |
| db1 | Sessions/Auth | Separate from cache for independent FLUSHDB |
| db2 | Rate limiting | Can be flushed independently during maintenance |
| db3 | Pub/Sub control | Pub/sub is connection-based, not db-based, but metadata can go here |

### Accessing Different Databases

```csharp
// Get specific database
var cacheDb = multiplexer.GetDatabase(0);   // Cache
var sessionDb = multiplexer.GetDatabase(1); // Sessions
var rateDb = multiplexer.GetDatabase(2);    // Rate limiting

// DI registration for multiple databases
builder.Services.AddKeyedSingleton<IDatabase>("cache",
    (sp, _) => sp.GetRequiredService<IConnectionMultiplexer>().GetDatabase(0));
builder.Services.AddKeyedSingleton<IDatabase>("session",
    (sp, _) => sp.GetRequiredService<IConnectionMultiplexer>().GetDatabase(1));
builder.Services.AddKeyedSingleton<IDatabase>("rate",
    (sp, _) => sp.GetRequiredService<IConnectionMultiplexer>().GetDatabase(2));

// Inject with [FromKeyedServices]
public class SessionStore(
    [FromKeyedServices("session")] IDatabase redis)
{
    // Uses db1 automatically
}
```

### Important Note
In production with Redis Cluster, only db0 is available. Plan accordingly. Multiple databases are a dev/single-instance convenience.

## Pipeline Batching

When performing multiple Redis operations, batch them into a single round trip.

### IBatch (Send all at once, await results)

```csharp
// Multiple cache reads in one round trip
public async Task<Dictionary<Guid, ProductResponse?>> GetMultipleAsync(IEnumerable<Guid> ids)
{
    var batch = _redis.CreateBatch();
    var tasks = ids.ToDictionary(
        id => id,
        id => batch.StringGetAsync(RedisKeys.ProductCache(id))
    );
    batch.Execute();

    var results = new Dictionary<Guid, ProductResponse?>();
    foreach (var (id, task) in tasks)
    {
        var value = await task;
        results[id] = value.HasValue
            ? JsonSerializer.Deserialize<ProductResponse>(value!)
            : null;
    }
    return results;
}
```

### Multiple Writes in One Round Trip

```csharp
// Bulk cache invalidation
public async Task InvalidateMultipleAsync(IEnumerable<string> keys)
{
    var batch = _redis.CreateBatch();
    var tasks = keys.Select(key => batch.KeyDeleteAsync(key)).ToArray();
    batch.Execute();
    await Task.WhenAll(tasks);
}
```

### When to Use Batching
- **3+ operations** to the same Redis instance
- **Bulk reads** (loading multiple cache entries)
- **Bulk writes** (invalidating multiple keys)
- **Initialization** (warming cache at startup)

## Fire-and-Forget

For non-critical writes where you don't need confirmation, use `CommandFlags.FireAndForget` to avoid waiting for the response.

```csharp
// Cache write — if it fails, TTL will handle it eventually
await _redis.StringSetAsync(
    key, value, ttl,
    flags: CommandFlags.FireAndForget
);

// TTL reset on sliding session — not critical if one reset is lost
await _redis.KeyExpireAsync(
    sessionKey, slidingWindow,
    CommandFlags.FireAndForget
);

// Counter increment for analytics — approximate is fine
await _redis.StringIncrementAsync(
    "stats:page-views",
    flags: CommandFlags.FireAndForget
);
```

### When to Use Fire-and-Forget
- Cache writes (cache miss is acceptable)
- TTL resets (one missed reset is not critical)
- Analytics counters (approximate counts are fine)
- **Never for**: locks, rate limits, session creation, idempotency checks

## Timeout Configuration

| Timeout | Default | Recommended | Description |
|---------|---------|-------------|-------------|
| `ConnectTimeout` | 5000ms | 5000ms | Time to establish initial connection |
| `SyncTimeout` | 5000ms | 3000ms | Time for synchronous operations |
| `AsyncTimeout` | 5000ms | 5000ms | Time for async operations |
| `KeepAlive` | -1 (disabled) | 60s | TCP keepalive interval |
| `ConnectRetry` | 3 | 3 | Number of connection retry attempts |

### Timeout Handling

```csharp
try
{
    var result = await _redis.StringGetAsync(key);
    // success
}
catch (RedisTimeoutException ex)
{
    _logger.LogWarning(ex, "Redis timeout for key {Key}", key);
    // Fall back to database
}
catch (RedisConnectionException ex)
{
    _logger.LogWarning(ex, "Redis connection failed for key {Key}", key);
    // Fall back to database
}
```

## Reconnection Strategy

StackExchange.Redis handles reconnection automatically. The key configurations:

```csharp
options.AbortOnConnectFail = false;     // CRITICAL: don't crash on startup
options.ReconnectRetryPolicy = new ExponentialRetry(
    deltaBackOffMilliseconds: 5000      // Start at 5s, exponentially back off
);
```

### What Happens During Disconnection
1. All operations throw `RedisConnectionException`
2. Your code catches these and falls back to the database
3. StackExchange.Redis retries connection in the background
4. Once reconnected, operations resume normally
5. No manual intervention needed

### Health Check for Monitoring

```csharp
// Program.cs
builder.Services.AddHealthChecks()
    .AddRedis(
        builder.Configuration.GetConnectionString("Redis")!,
        name: "redis",
        failureStatus: HealthStatus.Degraded, // Degraded, not Unhealthy
        tags: ["cache", "infrastructure"]
    );
```

Note: Redis being down is `Degraded`, not `Unhealthy`. The application should continue functioning without Redis — just slower.

## Anti-Patterns

### 1. Creating connections per request
```csharp
// BAD — creates a new connection for every request
public async Task<string?> Get(string key)
{
    using var redis = await ConnectionMultiplexer.ConnectAsync("localhost");
    return await redis.GetDatabase().StringGetAsync(key);
}

// GOOD — inject singleton multiplexer
public class CacheService(IConnectionMultiplexer redis)
{
    private readonly IDatabase _db = redis.GetDatabase();
}
```

### 2. Blocking on sync operations
```csharp
// BAD — blocks the thread
var result = _redis.StringGet(key); // sync

// GOOD — async all the way
var result = await _redis.StringGetAsync(key);
```

### 3. Not handling connection failures
```csharp
// BAD — crashes the request if Redis is down
var cached = await _redis.StringGetAsync(key);

// GOOD — graceful degradation
try
{
    var cached = await _redis.StringGetAsync(key);
    if (cached.HasValue) return Deserialize(cached!);
}
catch (RedisConnectionException)
{
    // Redis down — proceed to database
}
```
