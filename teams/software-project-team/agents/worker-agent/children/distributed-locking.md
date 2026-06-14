---
knowledge-base-summary: "Redis-based distributed lock to prevent concurrent execution of the same job across multiple pods. SETNX with TTL. Lock acquired → run job → release. Lock not acquired → skip this cycle. Handles lock expiry, dead locks, and crash recovery."
---
# Distributed Locking: One Job, One Pod

## Problem

In multi-pod deployments (Kubernetes, Docker Swarm, manual scaling), every pod runs the same set of `BackgroundService` instances. Without coordination, a job scheduled for midnight runs on ALL pods at midnight simultaneously. Result: duplicate API calls, duplicate processing, wasted resources, potential data corruption if the API endpoint is not perfectly idempotent.

## Solution: Redis Distributed Lock

Before executing, the job attempts to acquire a Redis lock (SETNX). Only the pod that acquires the lock runs the job. All other pods skip the cycle and wait for the next cron tick.

```
Pod A: TryAcquire("lock:jobs:cleanup") → true  → runs job → releases lock
Pod B: TryAcquire("lock:jobs:cleanup") → false → logs "skipping" → waits for next tick
Pod C: TryAcquire("lock:jobs:cleanup") → false → logs "skipping" → waits for next tick
```

## IDistributedLock Interface

```csharp
namespace ProjectName.Worker.Services;

public interface IDistributedLock
{
    /// <summary>
    /// Attempts to acquire a distributed lock.
    /// Returns true if the lock was acquired, false if another instance holds it.
    /// </summary>
    Task<bool> TryAcquireAsync(
        string key,
        TimeSpan ttl,
        CancellationToken ct = default);

    /// <summary>
    /// Releases a previously acquired lock.
    /// Safe to call even if the lock has already expired.
    /// </summary>
    Task ReleaseAsync(
        string key,
        CancellationToken ct = default);
}
```

## RedisDistributedLock Implementation

```csharp
using StackExchange.Redis;

namespace ProjectName.Worker.Services;

public sealed class RedisDistributedLock : IDistributedLock
{
    private readonly IConnectionMultiplexer _redis;
    private readonly ILogger<RedisDistributedLock> _logger;
    private readonly string _instanceId;

    public RedisDistributedLock(
        IConnectionMultiplexer redis,
        ILogger<RedisDistributedLock> logger)
    {
        _redis = redis;
        _logger = logger;
        // Unique identifier for this pod/process instance
        _instanceId = $"{Environment.MachineName}:{Environment.ProcessId}";
    }

    public async Task<bool> TryAcquireAsync(
        string key,
        TimeSpan ttl,
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();

        // SETNX: set only if key does not exist
        // Value = instance ID (for debugging: who holds the lock?)
        // TTL = auto-expire (dead lock protection)
        var acquired = await db.StringSetAsync(
            key,
            _instanceId,
            ttl,
            When.NotExists);

        if (acquired)
        {
            _logger.LogDebug(
                "Lock acquired: {Key} by {Instance}, TTL {TtlSeconds}s",
                key, _instanceId, ttl.TotalSeconds);
        }
        else
        {
            var holder = await db.StringGetAsync(key);
            _logger.LogDebug(
                "Lock NOT acquired: {Key} held by {Holder}",
                key, holder.ToString());
        }

        return acquired;
    }

    public async Task ReleaseAsync(
        string key,
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();

        // Only delete if WE hold the lock (prevent releasing another pod's lock)
        var currentHolder = await db.StringGetAsync(key);
        if (currentHolder == _instanceId)
        {
            await db.KeyDeleteAsync(key);
            _logger.LogDebug("Lock released: {Key} by {Instance}", key, _instanceId);
        }
        else
        {
            _logger.LogDebug(
                "Lock release skipped: {Key} held by {Holder}, we are {Instance}",
                key, currentHolder.ToString(), _instanceId);
        }
    }
}
```

### Why check the holder on release?

Consider this scenario:
1. Pod A acquires lock, starts job.
2. Job takes longer than TTL -- lock auto-expires.
3. Pod B acquires the (now-expired) lock, starts its own job run.
4. Pod A finishes, calls `ReleaseAsync` -- if we blindly delete, we release Pod B's lock.

By checking `currentHolder == _instanceId`, we only release our own lock. If the lock expired and someone else holds it, we leave it alone.

## Registration in Program.cs

```csharp
// Redis connection
builder.Services.AddSingleton<IConnectionMultiplexer>(
    ConnectionMultiplexer.Connect(
        builder.Configuration.GetConnectionString("Redis")
        ?? "redis:6379"));

// Distributed lock
builder.Services.AddSingleton<IDistributedLock, RedisDistributedLock>();
```

## Key Convention

```
lock:jobs:{job-name}
```

Examples:
- `lock:jobs:expired-order-cleanup`
- `lock:jobs:daily-report`
- `lock:jobs:inactive-user-reminder`

The key is always defined as a `const` in the job class to prevent typos and ensure consistency:

```csharp
private const string JobName = "expired-order-cleanup";
private const string LockKey = $"lock:jobs:{JobName}";
```

## Lock Value

The lock value is the instance identifier: `{MachineName}:{ProcessId}`. This serves two purposes:

1. **Debugging:** When a lock is held, you can see which pod holds it by inspecting the Redis key.
2. **Safe release:** Only the holder can release its own lock.

```bash
# In Redis CLI:
GET lock:jobs:expired-order-cleanup
# → "worker-pod-abc:12345"
```

## TTL: The Critical Safety Parameter

### Rule: TTL must be longer than the maximum expected job duration

```
TTL = max_expected_duration + buffer
```

| Job Duration | Buffer | TTL |
|-------------|--------|-----|
| < 10 seconds | +50s | 60 seconds |
| 10-60 seconds | +120s | 180 seconds |
| 1-5 minutes | +5m | 10 minutes |
| 5-30 minutes | +15m | 45 minutes |
| 30+ minutes | +30m | 60 minutes |

### Settings Key

```
jobs:{job-name}:lock-ttl-seconds
```

Stored in dynamic settings so it can be adjusted without redeployment:

```csharp
private const string LockTtlSettingsKey = "jobs:expired-order-cleanup:lock-ttl-seconds";
private const int DefaultLockTtlSeconds = 300; // 5 minutes

var lockTtlSeconds = await _settings.GetAsync<int>(
    LockTtlSettingsKey, stoppingToken) ?? DefaultLockTtlSeconds;
var lockTtl = TimeSpan.FromSeconds(lockTtlSeconds);
```

### What happens if TTL is too short?

1. Pod A acquires lock, starts job.
2. Job takes longer than TTL -- lock auto-expires while job is still running.
3. Pod B acquires the expired lock, starts the SAME job.
4. Both pods are now running the same job simultaneously -- the exact problem we're trying to prevent.

**If you see this happening:** Increase the TTL in settings. The safe-release check prevents lock corruption, but it does not prevent the actual duplicate execution.

### What happens if TTL is too long?

1. Pod A acquires lock, crashes mid-job.
2. Lock sits in Redis for the full TTL duration.
3. No other pod can run the job until the lock expires.
4. Job effectively pauses for the TTL duration after a crash.

**Trade-off:** Too short = risk of concurrent execution. Too long = longer recovery time after a crash. Err on the side of longer TTL -- a job skipping one cycle is much less harmful than running twice.

## Dead Lock Protection

The TTL is the primary dead lock protection mechanism. If a pod crashes while holding a lock:

1. The pod is gone -- it cannot release the lock.
2. The lock sits in Redis with its TTL countdown.
3. After TTL expires, Redis automatically deletes the key.
4. The next pod to check `TryAcquireAsync` gets the lock.

**No manual intervention needed.** The system self-heals after at most one TTL duration.

## Usage in a Job (Full Pattern)

```csharp
// Inside the main loop, after Task.Delay:

var lockAcquired = await _distributedLock.TryAcquireAsync(
    LockKey, lockTtl, stoppingToken);

if (!lockAcquired)
{
    _logger.LogInformation(
        "{Job} skipping cycle, another instance holds the lock", JobName);
    continue; // back to top of while loop -- wait for next cron tick
}

try
{
    await ExecuteJobAsync(stoppingToken);
}
finally
{
    // ALWAYS release in finally -- even if the job throws
    await _distributedLock.ReleaseAsync(LockKey, stoppingToken);
}
```

**The `finally` block is critical.** If the job throws an exception, the lock must still be released. Without `finally`, a failed job would hold the lock until TTL expires, blocking all pods for that duration.

## Important Rules

1. **Every job uses a distributed lock.** No exceptions, even in single-pod deployments. Tomorrow you might scale to two pods.
2. **TTL is always longer than max job duration.** Measure your slowest runs and add buffer.
3. **Lock release is always in `finally`.** Never leave locks dangling on job failure.
4. **Safe release: only release your own lock.** Check the holder before deleting.
5. **Lock key is a `const` in the job class.** Never construct lock keys dynamically from runtime values.
6. **TTL comes from dynamic settings.** Allows adjustment without redeployment.
7. **`SETNX` is atomic.** Two pods calling `TryAcquireAsync` simultaneously -- exactly one gets the lock. Redis guarantees this.
