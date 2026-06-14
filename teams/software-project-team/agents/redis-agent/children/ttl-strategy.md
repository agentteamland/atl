---
knowledge-base-summary: "Four TTL categories: short (1-5 min: rate limits, OTP), medium (15-60 min: query cache, verification tokens), long (hours-days: sessions, refresh tokens), very long (days-weeks: feature flags, settings). Dynamic TTL from settings service. Sliding vs absolute expiry and when to use each."
---
# TTL Strategy

Every Redis key MUST have a TTL. No immortal keys. This document defines the TTL categories, when to use each, and how TTLs are managed dynamically.

## TTL Categories

### Short (1-5 minutes)
**Use for:** data that changes frequently or must be near-real-time.

| Key Pattern | TTL | Reason |
|-------------|-----|--------|
| `rate:login:{ip}` | 5 min | Rate limit window resets |
| `rate:api:{userId}:{endpoint}` | 1 min | Per-minute API rate limit |
| `session:otp:{userId}` | 3 min | OTP must expire quickly for security |
| `lock:*` | 30s - 2 min | Locks must release even if holder crashes |
| `cache:search:{hash}` | 2 min | Search results go stale fast |

### Medium (15-60 minutes)
**Use for:** query cache, verification tokens, temporary state.

| Key Pattern | TTL | Reason |
|-------------|-----|--------|
| `cache:product:{id}` | 30 min | Products change infrequently |
| `cache:products:list:{cursor}` | 15 min | List cache, moderate staleness OK |
| `cache:dashboard:stats` | 5-15 min | Stats are approximate anyway |
| `session:verify:{token}` | 30 min | Email verification link validity |
| `cache:user:profile:{id}` | 15 min | User profile, invalidated on update |

### Long (hours-days)
**Use for:** sessions, refresh tokens, data that rarely changes.

| Key Pattern | TTL | Reason |
|-------------|-----|--------|
| `session:refresh:{userId}` | 30 days | Refresh token lifespan |
| `cache:settings:all` | 24 hours | Settings change rarely, invalidated on update |
| `idempotency:{requestId}` | 24 hours | Prevent reprocessing for a day |

### Very Long (days-weeks)
**Use for:** feature flags, system settings. The closest to "permanent" we allow.

| Key Pattern | TTL | Reason |
|-------------|-----|--------|
| `settings:feature:{name}` | 7 days | Feature flags, re-checked weekly |
| `settings:maintenance-mode` | 7 days | System flag, refreshed periodically |
| `cache:static:{resource}` | 7 days | Truly static content (countries, currencies) |

## Dynamic TTL from Settings

TTL values are NOT hardcoded. They come from the dynamic settings service, with fallbacks:

```csharp
// Resolution order:
// 1. Query-level override (ICacheable.CacheDuration)
// 2. Dynamic setting from Redis/DB (settings service)
// 3. Hardcoded fallback (last resort)

public async Task<TimeSpan> ResolveTtl(ICacheable request, CancellationToken ct)
{
    // 1. Query override
    if (request.CacheDuration.HasValue)
        return request.CacheDuration.Value;

    // 2. Dynamic setting
    var settingTtl = await _settings.GetAsync<TimeSpan?>("Cache:DefaultTtl", ct);
    if (settingTtl.HasValue)
        return settingTtl.Value;

    // 3. Hardcoded fallback
    return TimeSpan.FromMinutes(30);
}
```

### Settings Keys for TTL

```
Cache:DefaultTtl              = "00:30:00"   (30 minutes)
Cache:ShortTtl                = "00:05:00"   (5 minutes)
Cache:LongTtl                 = "24:00:00"   (24 hours)
Auth:RefreshTokenTtl          = "30.00:00:00" (30 days)
Auth:VerificationTokenTtl     = "00:30:00"   (30 minutes)
Auth:OtpTtl                   = "00:03:00"   (3 minutes)
RateLimit:LoginWindowTtl      = "00:05:00"   (5 minutes)
RateLimit:ApiWindowTtl        = "00:01:00"   (1 minute)
Lock:DefaultTtl               = "00:00:30"   (30 seconds)
```

## Sliding vs Absolute Expiry

### Absolute Expiry (default)
Key expires at a fixed time after creation, regardless of access.

```csharp
// Absolute: expires 30 minutes from NOW, no matter what
await _redis.StringSetAsync(key, value, TimeSpan.FromMinutes(30));
```

**Use for:**
- Cache entries (prevents stale data from living forever)
- Rate limit windows (must reset at fixed intervals)
- Verification tokens (security: must expire regardless of use)
- Idempotency keys (24-hour window from first request)

### Sliding Expiry
Key's TTL resets every time it is accessed. Stays alive as long as it is used.

```csharp
// Sliding: read and reset TTL
var value = await _redis.StringGetAsync(key);
if (value.HasValue)
{
    // Reset the TTL on access
    await _redis.KeyExpireAsync(key, TimeSpan.FromMinutes(30));
}
```

**Use for:**
- User sessions (keep alive while user is active)
- Connection tracking (reset on each heartbeat)
- Shopping cart (keep alive while user is browsing)

### Implementation Pattern for Sliding

```csharp
// Infrastructure/Redis/SlidingCacheService.cs
public class SlidingCacheService
{
    private readonly IDatabase _redis;

    public async Task<T?> GetWithSlideAsync<T>(string key, TimeSpan slidingWindow)
    {
        var value = await _redis.StringGetAsync(key);
        if (!value.HasValue)
            return default;

        // Reset TTL on access (fire-and-forget for performance)
        _ = _redis.KeyExpireAsync(key, slidingWindow, CommandFlags.FireAndForget);

        return JsonSerializer.Deserialize<T>(value!);
    }

    public async Task SetWithSlideAsync<T>(string key, T value, TimeSpan slidingWindow)
    {
        var json = JsonSerializer.Serialize(value);
        await _redis.StringSetAsync(key, json, slidingWindow);
    }
}
```

## TTL Anti-Patterns

### 1. No TTL (immortal key)
```csharp
// BAD — key lives forever, leaks memory
await _redis.StringSetAsync(key, value);

// GOOD — always set expiry
await _redis.StringSetAsync(key, value, TimeSpan.FromMinutes(30));
```

### 2. TTL too long for cache
```csharp
// BAD — caching a product for 7 days means 7 days of stale data
await _redis.StringSetAsync(key, value, TimeSpan.FromDays(7));

// GOOD — reasonable TTL with invalidation on update
await _redis.StringSetAsync(key, value, TimeSpan.FromMinutes(30));
```

### 3. TTL too short for sessions
```csharp
// BAD — user has to re-login every 2 minutes
await _redis.StringSetAsync(sessionKey, value, TimeSpan.FromMinutes(2));

// GOOD — reasonable session with sliding expiry
await _redis.StringSetAsync(sessionKey, value, TimeSpan.FromMinutes(30));
// + slide on every request
```

### 4. Hardcoded TTL everywhere
```csharp
// BAD — changing TTL requires code deployment
await _redis.StringSetAsync(key, value, TimeSpan.FromMinutes(30));

// GOOD — TTL from dynamic settings
var ttl = await _settings.GetAsync<TimeSpan>("Cache:DefaultTtl", ct);
await _redis.StringSetAsync(key, value, ttl);
```

## Monitoring TTL

```csharp
// Check remaining TTL on a key
var ttl = await _redis.KeyTimeToLiveAsync(key);
if (ttl.HasValue)
{
    _logger.LogDebug("Key {Key} expires in {Seconds}s", key, ttl.Value.TotalSeconds);
}
else
{
    // Key has no TTL (immortal) — this is a bug, report it
    _logger.LogWarning("Key {Key} has no TTL — this should not happen", key);
}
```
