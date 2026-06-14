---
knowledge-base-summary: "Production-ready patterns: distributed lock (SETNX + TTL + unique token), sliding window rate limiting (INCR + EXPIRE), idempotency store (SETNX for request dedup), session storage (Hash with sliding TTL), circuit breaker state. Code examples for each pattern with StackExchange.Redis."
---
# Distributed Patterns

Production-ready patterns for distributed systems using Redis. Each pattern includes the problem it solves, the implementation, and edge cases.

## 1. Distributed Lock

### Problem
Multiple instances (pods, workers) must not execute the same operation simultaneously. Example: only one worker should run the daily cleanup job.

### Implementation

```csharp
// Infrastructure/Redis/DistributedLock.cs
public class DistributedLock : IDistributedLock
{
    private readonly IDatabase _redis;
    private readonly ILogger<DistributedLock> _logger;

    public DistributedLock(
        IConnectionMultiplexer multiplexer,
        ILogger<DistributedLock> logger)
    {
        _redis = multiplexer.GetDatabase();
        _logger = logger;
    }

    /// <summary>
    /// Acquires a distributed lock. Returns a disposable that releases the lock.
    /// Returns null if the lock could not be acquired.
    /// </summary>
    public async Task<IAsyncDisposable?> AcquireAsync(
        string resource,
        TimeSpan expiry,
        CancellationToken ct = default)
    {
        var key = $"lock:{resource}";
        var token = Guid.NewGuid().ToString(); // Unique token to prevent releasing others' locks

        var acquired = await _redis.StringSetAsync(
            key,
            token,
            expiry,
            When.NotExists // SETNX — only set if key doesn't exist
        );

        if (!acquired)
        {
            _logger.LogDebug("Lock {Resource} already held, skipping", resource);
            return null;
        }

        _logger.LogDebug("Lock {Resource} acquired with token {Token}, expires in {Expiry}",
            resource, token, expiry);

        return new LockHandle(_redis, key, token, _logger);
    }

    private class LockHandle : IAsyncDisposable
    {
        private readonly IDatabase _redis;
        private readonly string _key;
        private readonly string _token;
        private readonly ILogger _logger;

        // Lua script to release lock only if we still own it
        private const string ReleaseLuaScript = @"
            if redis.call('get', KEYS[1]) == ARGV[1] then
                return redis.call('del', KEYS[1])
            else
                return 0
            end";

        public LockHandle(IDatabase redis, string key, string token, ILogger logger)
        {
            _redis = redis;
            _key = key;
            _token = token;
            _logger = logger;
        }

        public async ValueTask DisposeAsync()
        {
            try
            {
                var result = await _redis.ScriptEvaluateAsync(
                    ReleaseLuaScript,
                    new RedisKey[] { _key },
                    new RedisValue[] { _token }
                );

                if ((int)result == 1)
                    _logger.LogDebug("Lock {Key} released", _key);
                else
                    _logger.LogWarning("Lock {Key} was already released or expired", _key);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Failed to release lock {Key}", _key);
            }
        }
    }
}
```

### Usage

```csharp
// In a Worker job or handler
await using var lockHandle = await _lock.AcquireAsync(
    resource: "jobs:daily-cleanup",
    expiry: TimeSpan.FromMinutes(5)
);

if (lockHandle is null)
{
    _logger.LogInformation("Daily cleanup already running on another instance, skipping");
    return;
}

// Safe to execute — we hold the lock
await ExecuteCleanupAsync(ct);
// Lock is automatically released when lockHandle is disposed
```

### Critical Rules
- **Always set a TTL** on the lock. If the holder crashes, the lock must expire.
- **Use a unique token** (GUID) to prevent releasing someone else's lock.
- **Use Lua script for release** — ensures atomic check-and-delete.
- **Keep lock duration short** — just enough for the operation.

### Interface

```csharp
// Application/Interfaces/IDistributedLock.cs
public interface IDistributedLock
{
    Task<IAsyncDisposable?> AcquireAsync(
        string resource,
        TimeSpan expiry,
        CancellationToken ct = default);
}
```

## 2. Rate Limiting (Sliding Window)

### Problem
Limit how many times a user can perform an action within a time window. Example: max 5 login attempts per 5 minutes.

### Implementation

```csharp
// Infrastructure/Redis/RateLimiter.cs
public class RateLimiter : IRateLimiter
{
    private readonly IDatabase _redis;

    public RateLimiter(IConnectionMultiplexer multiplexer)
    {
        _redis = multiplexer.GetDatabase();
    }

    /// <summary>
    /// Checks and increments rate limit. Returns remaining attempts.
    /// Throws TooManyRequestsException if limit exceeded.
    /// </summary>
    public async Task<RateLimitResult> CheckAsync(
        string scope,
        string identifier,
        int maxAttempts,
        TimeSpan window)
    {
        var key = $"rate:{scope}:{identifier}";

        // Increment counter
        var count = await _redis.StringIncrementAsync(key);

        // Set expiry on first increment only
        if (count == 1)
        {
            await _redis.KeyExpireAsync(key, window);
        }

        var remaining = Math.Max(0, maxAttempts - (int)count);
        var ttl = await _redis.KeyTimeToLiveAsync(key);

        return new RateLimitResult
        {
            IsAllowed = count <= maxAttempts,
            CurrentCount = (int)count,
            MaxAttempts = maxAttempts,
            Remaining = remaining,
            ResetsIn = ttl ?? window
        };
    }

    /// <summary>
    /// Resets rate limit for a specific scope and identifier.
    /// Call this on successful action (e.g., successful login resets login attempts).
    /// </summary>
    public async Task ResetAsync(string scope, string identifier)
    {
        var key = $"rate:{scope}:{identifier}";
        await _redis.KeyDeleteAsync(key);
    }
}

public record RateLimitResult
{
    public bool IsAllowed { get; init; }
    public int CurrentCount { get; init; }
    public int MaxAttempts { get; init; }
    public int Remaining { get; init; }
    public TimeSpan ResetsIn { get; init; }
}
```

### Usage in Login Handler

```csharp
public async ValueTask<LoginResponse> Handle(LoginCommand cmd, CancellationToken ct)
{
    // Check rate limit before processing
    var rateLimit = await _rateLimiter.CheckAsync(
        scope: "login",
        identifier: cmd.IpAddress,
        maxAttempts: 5,
        window: TimeSpan.FromMinutes(5)
    );

    if (!rateLimit.IsAllowed)
    {
        throw new TooManyRequestsException(
            $"Too many login attempts. Try again in {rateLimit.ResetsIn.TotalSeconds:F0}s");
    }

    // Attempt login...
    var user = await _db.Users.FirstOrDefaultAsync(u => u.Email == cmd.Email, ct);
    if (user is null || !BCrypt.Verify(cmd.Password, user.PasswordHash))
    {
        throw new UnauthorizedException("Invalid credentials");
    }

    // Success — reset rate limit
    await _rateLimiter.ResetAsync("login", cmd.IpAddress);

    return new LoginResponse { ... };
}
```

### As Mediator Pipeline Behavior

```csharp
// Application/Interfaces/IRateLimited.cs
public interface IRateLimited
{
    string RateLimitScope { get; }
    string RateLimitIdentifier { get; }
    int MaxAttempts { get; }
    TimeSpan Window { get; }
}

// Application/Behaviors/RateLimitBehavior.cs
public sealed class RateLimitBehavior<TRequest, TResponse> : IPipelineBehavior<TRequest, TResponse>
    where TRequest : IRateLimited
{
    private readonly IRateLimiter _rateLimiter;

    public async ValueTask<TResponse> Handle(
        TRequest request,
        CancellationToken ct,
        MessageHandlerDelegate<TRequest, TResponse> next)
    {
        var result = await _rateLimiter.CheckAsync(
            request.RateLimitScope,
            request.RateLimitIdentifier,
            request.MaxAttempts,
            request.Window
        );

        if (!result.IsAllowed)
        {
            throw new TooManyRequestsException(
                $"Rate limit exceeded. Try again in {result.ResetsIn.TotalSeconds:F0}s");
        }

        return await next(request, ct);
    }
}
```

## 3. Idempotency Store

### Problem
Prevent duplicate processing of the same request. Example: client retries a payment creation — the second request should return the same result, not create a duplicate payment.

### Implementation

```csharp
// Infrastructure/Redis/IdempotencyStore.cs
public class IdempotencyStore : IIdempotencyStore
{
    private readonly IDatabase _redis;
    private readonly ILogger<IdempotencyStore> _logger;

    public IdempotencyStore(
        IConnectionMultiplexer multiplexer,
        ILogger<IdempotencyStore> logger)
    {
        _redis = multiplexer.GetDatabase();
        _logger = logger;
    }

    /// <summary>
    /// Try to claim an idempotency key. Returns true if this is the first request.
    /// </summary>
    public async Task<bool> TryClaimAsync(string idempotencyKey, TimeSpan expiry)
    {
        var key = $"idempotency:{idempotencyKey}";
        var claimed = await _redis.StringSetAsync(
            key,
            "processing",
            expiry,
            When.NotExists
        );

        if (claimed)
        {
            _logger.LogDebug("Idempotency key {Key} claimed", idempotencyKey);
        }
        else
        {
            _logger.LogDebug("Idempotency key {Key} already exists", idempotencyKey);
        }

        return claimed;
    }

    /// <summary>
    /// Store the result for a claimed idempotency key.
    /// </summary>
    public async Task StoreResultAsync<T>(string idempotencyKey, T result, TimeSpan expiry)
    {
        var key = $"idempotency:{idempotencyKey}";
        var json = JsonSerializer.Serialize(result);
        await _redis.StringSetAsync(key, json, expiry);
        _logger.LogDebug("Stored result for idempotency key {Key}", idempotencyKey);
    }

    /// <summary>
    /// Get the stored result for an idempotency key. Returns null if processing or not found.
    /// </summary>
    public async Task<T?> GetResultAsync<T>(string idempotencyKey)
    {
        var key = $"idempotency:{idempotencyKey}";
        var value = await _redis.StringGetAsync(key);

        if (!value.HasValue || value == "processing")
            return default;

        return JsonSerializer.Deserialize<T>(value!);
    }
}
```

### Usage in Mediator Pipeline

```csharp
// Application/Interfaces/IIdempotent.cs
public interface IIdempotent
{
    string? IdempotencyKey { get; } // From X-Idempotency-Key header, null = skip
}

// Application/Behaviors/IdempotencyBehavior.cs
public sealed class IdempotencyBehavior<TRequest, TResponse> : IPipelineBehavior<TRequest, TResponse>
    where TRequest : IIdempotent
{
    private readonly IIdempotencyStore _store;
    private static readonly TimeSpan DefaultExpiry = TimeSpan.FromHours(24);

    public async ValueTask<TResponse> Handle(
        TRequest request,
        CancellationToken ct,
        MessageHandlerDelegate<TRequest, TResponse> next)
    {
        // No key = no idempotency check
        if (string.IsNullOrEmpty(request.IdempotencyKey))
            return await next(request, ct);

        // Check if already processed
        var existingResult = await _store.GetResultAsync<TResponse>(request.IdempotencyKey);
        if (existingResult is not null)
            return existingResult;

        // Try to claim
        var claimed = await _store.TryClaimAsync(request.IdempotencyKey, DefaultExpiry);
        if (!claimed)
        {
            // Another request is currently processing — wait briefly then check again
            await Task.Delay(500, ct);
            existingResult = await _store.GetResultAsync<TResponse>(request.IdempotencyKey);
            if (existingResult is not null)
                return existingResult;

            throw new ConflictException("Request is currently being processed");
        }

        // Execute handler
        var response = await next(request, ct);

        // Store result
        await _store.StoreResultAsync(request.IdempotencyKey, response, DefaultExpiry);

        return response;
    }
}
```

## 4. Session Storage

### Problem
Store user session data with sliding expiry that resets on each access.

### Implementation

```csharp
// Infrastructure/Redis/SessionStore.cs
public class SessionStore : ISessionStore
{
    private readonly IDatabase _redis;

    public SessionStore(IConnectionMultiplexer multiplexer)
    {
        _redis = multiplexer.GetDatabase();
    }

    public async Task CreateAsync(Guid userId, UserSessionData data, TimeSpan expiry)
    {
        var key = RedisKeys.RefreshToken(userId);
        var entries = new HashEntry[]
        {
            new("userId", userId.ToString()),
            new("email", data.Email),
            new("role", data.Role),
            new("tenantId", data.TenantId?.ToString() ?? ""),
            new("createdAt", DateTimeOffset.UtcNow.ToUnixTimeSeconds()),
            new("lastActivity", DateTimeOffset.UtcNow.ToUnixTimeSeconds()),
        };

        await _redis.HashSetAsync(key, entries);
        await _redis.KeyExpireAsync(key, expiry);
    }

    public async Task<UserSessionData?> GetAndSlideAsync(Guid userId, TimeSpan slidingExpiry)
    {
        var key = RedisKeys.RefreshToken(userId);
        var fields = await _redis.HashGetAllAsync(key);

        if (fields.Length == 0)
            return null;

        // Slide the expiry (fire-and-forget for performance)
        _ = _redis.KeyExpireAsync(key, slidingExpiry, CommandFlags.FireAndForget);

        // Update last activity
        _ = _redis.HashSetAsync(key, "lastActivity",
            DateTimeOffset.UtcNow.ToUnixTimeSeconds(), CommandFlags.FireAndForget);

        return MapToSessionData(fields);
    }

    public async Task DestroyAsync(Guid userId)
    {
        var key = RedisKeys.RefreshToken(userId);
        await _redis.KeyDeleteAsync(key);
    }

    public async Task DestroyAllAsync(Guid userId)
    {
        // Destroy all sessions for a user (e.g., on password change)
        var server = _redis.Multiplexer.GetServer(_redis.Multiplexer.GetEndPoints().First());
        await foreach (var key in server.KeysAsync(pattern: RedisKeys.UserSessionPattern(userId)))
        {
            await _redis.KeyDeleteAsync(key);
        }
    }

    private static UserSessionData MapToSessionData(HashEntry[] fields)
    {
        var dict = fields.ToDictionary(x => x.Name.ToString(), x => x.Value.ToString());
        return new UserSessionData
        {
            UserId = Guid.Parse(dict["userId"]),
            Email = dict["email"],
            Role = dict["role"],
            TenantId = string.IsNullOrEmpty(dict["tenantId"])
                ? null
                : Guid.Parse(dict["tenantId"]),
        };
    }
}
```

## 5. Circuit Breaker State

### Problem
Track the state of an external service (open/closed/half-open) across all instances. When a service fails repeatedly, stop calling it temporarily.

### Implementation

```csharp
// Infrastructure/Redis/CircuitBreakerState.cs
public class CircuitBreakerState : ICircuitBreakerState
{
    private readonly IDatabase _redis;

    public CircuitBreakerState(IConnectionMultiplexer multiplexer)
    {
        _redis = multiplexer.GetDatabase();
    }

    public async Task<bool> IsOpenAsync(string serviceName)
    {
        var key = $"circuit:{serviceName}";
        var state = await _redis.StringGetAsync(key);
        return state == "open";
    }

    public async Task RecordFailureAsync(string serviceName, int threshold, TimeSpan window)
    {
        var failureKey = $"circuit:{serviceName}:failures";
        var count = await _redis.StringIncrementAsync(failureKey);

        if (count == 1)
            await _redis.KeyExpireAsync(failureKey, window);

        if (count >= threshold)
        {
            // Trip the circuit breaker
            var circuitKey = $"circuit:{serviceName}";
            await _redis.StringSetAsync(circuitKey, "open", TimeSpan.FromMinutes(1));
            await _redis.KeyDeleteAsync(failureKey);
        }
    }

    public async Task RecordSuccessAsync(string serviceName)
    {
        var failureKey = $"circuit:{serviceName}:failures";
        var circuitKey = $"circuit:{serviceName}";
        await _redis.KeyDeleteAsync(new RedisKey[] { failureKey, circuitKey });
    }
}
```

### Usage

```csharp
// Before calling an external service
if (await _circuitBreaker.IsOpenAsync("payment-gateway"))
{
    throw new ServiceUnavailableException("Payment gateway is temporarily unavailable");
}

try
{
    var result = await _paymentGateway.ProcessAsync(payment);
    await _circuitBreaker.RecordSuccessAsync("payment-gateway");
    return result;
}
catch (HttpRequestException)
{
    await _circuitBreaker.RecordFailureAsync(
        "payment-gateway",
        threshold: 5,
        window: TimeSpan.FromMinutes(1)
    );
    throw;
}
```

## Pattern Summary

| Pattern | Redis Operation | Key Pattern | TTL |
|---------|----------------|-------------|-----|
| Distributed Lock | SETNX + Lua DEL | `lock:{resource}` | 30s - 5min |
| Rate Limiting | INCR + EXPIRE | `rate:{scope}:{id}` | 1 - 5min |
| Idempotency | SETNX + GET/SET | `idempotency:{key}` | 24h |
| Session | HSET + EXPIRE | `session:{type}:{id}` | Sliding 30min |
| Circuit Breaker | SET + INCR | `circuit:{service}` | 1min (open state) |

## Interfaces (Application Layer)

```csharp
// All interfaces live in Application/Interfaces/

public interface IDistributedLock
{
    Task<IAsyncDisposable?> AcquireAsync(string resource, TimeSpan expiry, CancellationToken ct = default);
}

public interface IRateLimiter
{
    Task<RateLimitResult> CheckAsync(string scope, string identifier, int maxAttempts, TimeSpan window);
    Task ResetAsync(string scope, string identifier);
}

public interface IIdempotencyStore
{
    Task<bool> TryClaimAsync(string idempotencyKey, TimeSpan expiry);
    Task StoreResultAsync<T>(string idempotencyKey, T result, TimeSpan expiry);
    Task<T?> GetResultAsync<T>(string idempotencyKey);
}

public interface ISessionStore
{
    Task CreateAsync(Guid userId, UserSessionData data, TimeSpan expiry);
    Task<UserSessionData?> GetAndSlideAsync(Guid userId, TimeSpan slidingExpiry);
    Task DestroyAsync(Guid userId);
    Task DestroyAllAsync(Guid userId);
}

public interface ICircuitBreakerState
{
    Task<bool> IsOpenAsync(string serviceName);
    Task RecordFailureAsync(string serviceName, int threshold, TimeSpan window);
    Task RecordSuccessAsync(string serviceName);
}
```
