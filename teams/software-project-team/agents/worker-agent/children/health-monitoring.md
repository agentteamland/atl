---
knowledge-base-summary: "Job execution tracking: last run time, last success, last failure, consecutive failure count. Redis-based health state. Health endpoint for orchestration tools. Stale job detection: \"this job hasn't run in 2x its expected interval → alert.\""
---
# Health Monitoring: Job Execution Tracking

## Purpose

Every job writes its health state to Redis after each execution. This allows operators, health checks, and alerting systems to answer three questions at any time:

1. **Is this job running?** When was the last run?
2. **Is this job healthy?** Did the last run succeed or fail?
3. **Is this job stuck?** Has it been too long since the last run?

## Redis Key Convention

```
jobs:{job-name}:health
```

Examples:
- `jobs:expired-order-cleanup:health`
- `jobs:daily-report:health`
- `jobs:inactive-user-reminder:health`

## Health State Structure

Each job's health is stored as a Redis hash with these fields:

| Field | Type | Description |
|-------|------|-------------|
| `last_run` | ISO 8601 UTC | Timestamp of the most recent execution start |
| `last_success` | ISO 8601 UTC | Timestamp of the most recent successful execution |
| `last_failure` | ISO 8601 UTC | Timestamp of the most recent failed execution |
| `last_failure_reason` | string | Error message from the most recent failure |
| `consecutive_failures` | int | Number of consecutive failures (reset to 0 on success) |
| `last_duration_ms` | long | Duration of the most recent execution in milliseconds |
| `status` | string | Current status: `running`, `succeeded`, `failed` |

```bash
# Redis CLI inspection:
HGETALL jobs:expired-order-cleanup:health
# 1) "last_run"
# 2) "2026-04-14T06:00:01Z"
# 3) "last_success"
# 4) "2026-04-14T06:00:03Z"
# 5) "consecutive_failures"
# 6) "0"
# 7) "last_duration_ms"
# 8) "1823"
# 9) "status"
# 10) "succeeded"
```

## IJobHealthTracker Interface

```csharp
namespace ProjectName.Worker.Services;

public interface IJobHealthTracker
{
    /// <summary>
    /// Called at the start of job execution.
    /// Sets status to "running" and updates last_run timestamp.
    /// </summary>
    Task RecordRunStartAsync(
        string jobName,
        CancellationToken ct = default);

    /// <summary>
    /// Called after successful job execution.
    /// Sets status to "succeeded", updates last_success, resets consecutive_failures to 0.
    /// </summary>
    Task RecordSuccessAsync(
        string jobName,
        CancellationToken ct = default);

    /// <summary>
    /// Called after failed job execution.
    /// Sets status to "failed", updates last_failure, increments consecutive_failures.
    /// Triggers alert check if consecutive_failures exceeds threshold.
    /// </summary>
    Task RecordFailureAsync(
        string jobName,
        string reason,
        CancellationToken ct = default);

    /// <summary>
    /// Returns the current health state of a specific job.
    /// </summary>
    Task<JobHealthState?> GetHealthAsync(
        string jobName,
        CancellationToken ct = default);

    /// <summary>
    /// Returns the health state of all known jobs.
    /// Used by the health endpoint.
    /// </summary>
    Task<IReadOnlyList<JobHealthState>> GetAllHealthAsync(
        CancellationToken ct = default);
}

public record JobHealthState(
    string JobName,
    string Status,
    DateTime? LastRun,
    DateTime? LastSuccess,
    DateTime? LastFailure,
    string? LastFailureReason,
    int ConsecutiveFailures,
    long LastDurationMs);
```

## RedisJobHealthTracker Implementation

```csharp
using StackExchange.Redis;
using System.Text.Json;

namespace ProjectName.Worker.Services;

public sealed class RedisJobHealthTracker : IJobHealthTracker
{
    private readonly IConnectionMultiplexer _redis;
    private readonly ILogger<RedisJobHealthTracker> _logger;
    private const string KeyPrefix = "jobs:";
    private const string KeySuffix = ":health";
    private const string JobRegistryKey = "jobs:__registry";
    private const int AlertThreshold = 3; // consecutive failures before alerting

    public RedisJobHealthTracker(
        IConnectionMultiplexer redis,
        ILogger<RedisJobHealthTracker> logger)
    {
        _redis = redis;
        _logger = logger;
    }

    public async Task RecordRunStartAsync(
        string jobName,
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var key = GetKey(jobName);
        var now = DateTime.UtcNow.ToString("O");

        await db.HashSetAsync(key, new[]
        {
            new HashEntry("last_run", now),
            new HashEntry("status", "running"),
        });

        // Register job name in the global registry (for GetAllHealthAsync)
        await db.SetAddAsync(JobRegistryKey, jobName);

        _logger.LogDebug("{Job} health: run started at {Time}", jobName, now);
    }

    public async Task RecordSuccessAsync(
        string jobName,
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var key = GetKey(jobName);
        var now = DateTime.UtcNow.ToString("O");

        // Calculate duration from last_run
        var lastRun = await db.HashGetAsync(key, "last_run");
        long durationMs = 0;
        if (lastRun.HasValue && DateTime.TryParse(lastRun, out var runStart))
        {
            durationMs = (long)(DateTime.UtcNow - runStart).TotalMilliseconds;
        }

        await db.HashSetAsync(key, new[]
        {
            new HashEntry("last_success", now),
            new HashEntry("consecutive_failures", 0),
            new HashEntry("last_duration_ms", durationMs),
            new HashEntry("status", "succeeded"),
        });

        _logger.LogDebug(
            "{Job} health: success at {Time}, duration {DurationMs}ms",
            jobName, now, durationMs);
    }

    public async Task RecordFailureAsync(
        string jobName,
        string reason,
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var key = GetKey(jobName);
        var now = DateTime.UtcNow.ToString("O");

        // Calculate duration from last_run
        var lastRun = await db.HashGetAsync(key, "last_run");
        long durationMs = 0;
        if (lastRun.HasValue && DateTime.TryParse(lastRun, out var runStart))
        {
            durationMs = (long)(DateTime.UtcNow - runStart).TotalMilliseconds;
        }

        // Increment consecutive failures
        var newFailureCount = await db.HashIncrementAsync(
            key, "consecutive_failures");

        await db.HashSetAsync(key, new[]
        {
            new HashEntry("last_failure", now),
            new HashEntry("last_failure_reason", reason ?? "Unknown"),
            new HashEntry("last_duration_ms", durationMs),
            new HashEntry("status", "failed"),
        });

        _logger.LogDebug(
            "{Job} health: failure at {Time}, consecutive failures: {Count}",
            jobName, now, newFailureCount);

        // Alert check
        if (newFailureCount >= AlertThreshold)
        {
            _logger.LogError(
                "{Job} ALERT: {Count} consecutive failures (threshold: {Threshold}). "
                + "Last error: {Reason}",
                jobName, newFailureCount, AlertThreshold, reason);
        }
    }

    public async Task<JobHealthState?> GetHealthAsync(
        string jobName,
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var key = GetKey(jobName);
        var hash = await db.HashGetAllAsync(key);

        if (hash.Length == 0) return null;

        return MapToState(jobName, hash);
    }

    public async Task<IReadOnlyList<JobHealthState>> GetAllHealthAsync(
        CancellationToken ct = default)
    {
        var db = _redis.GetDatabase();
        var jobNames = await db.SetMembersAsync(JobRegistryKey);
        var results = new List<JobHealthState>();

        foreach (var name in jobNames)
        {
            var key = GetKey(name!);
            var hash = await db.HashGetAllAsync(key);
            if (hash.Length > 0)
            {
                results.Add(MapToState(name!, hash));
            }
        }

        return results;
    }

    private static string GetKey(string jobName) =>
        $"{KeyPrefix}{jobName}{KeySuffix}";

    private static JobHealthState MapToState(string jobName, HashEntry[] hash)
    {
        var dict = hash.ToDictionary(
            h => h.Name.ToString(),
            h => h.Value.ToString());

        return new JobHealthState(
            JobName: jobName,
            Status: dict.GetValueOrDefault("status", "unknown"),
            LastRun: ParseDate(dict.GetValueOrDefault("last_run")),
            LastSuccess: ParseDate(dict.GetValueOrDefault("last_success")),
            LastFailure: ParseDate(dict.GetValueOrDefault("last_failure")),
            LastFailureReason: dict.GetValueOrDefault("last_failure_reason"),
            ConsecutiveFailures: int.TryParse(
                dict.GetValueOrDefault("consecutive_failures"), out var cf) ? cf : 0,
            LastDurationMs: long.TryParse(
                dict.GetValueOrDefault("last_duration_ms"), out var dur) ? dur : 0);
    }

    private static DateTime? ParseDate(string? value) =>
        DateTime.TryParse(value, out var d) ? d : null;
}
```

## Registration in Program.cs

```csharp
// Health tracker (requires Redis IConnectionMultiplexer already registered)
builder.Services.AddSingleton<IJobHealthTracker, RedisJobHealthTracker>();
```

## Usage in a Job

Health tracking is called at three points in the job lifecycle:

```csharp
// 1. Before execution
await _healthTracker.RecordRunStartAsync(JobName, stoppingToken);

var sw = Stopwatch.StartNew();
try
{
    await ExecuteJobAsync(stoppingToken);

    // 2. After successful execution
    sw.Stop();
    await _healthTracker.RecordSuccessAsync(JobName, stoppingToken);
}
catch (Exception ex)
{
    // 3. After failed execution
    sw.Stop();
    await _healthTracker.RecordFailureAsync(JobName, ex.Message, stoppingToken);
    throw; // or handle based on job's error strategy
}
```

## Stale Job Detection

A job is considered stale if it hasn't run within 2x its expected interval. This catches scenarios where the job is silently not running (cron misconfigured, pod not starting, settings key missing).

### How to detect:

```csharp
public static bool IsStale(JobHealthState health, TimeSpan expectedInterval)
{
    if (health.LastRun is null) return true; // never ran

    var timeSinceLastRun = DateTime.UtcNow - health.LastRun.Value;
    return timeSinceLastRun > expectedInterval * 2;
}
```

### Example:

| Job | Expected Interval | Last Run | Now | Time Since | Stale? |
|-----|-------------------|----------|-----|------------|--------|
| cleanup | 6 hours | 5 hours ago | - | 5h | No (< 12h threshold) |
| cleanup | 6 hours | 13 hours ago | - | 13h | Yes (> 12h threshold) |
| report | 24 hours | 20 hours ago | - | 20h | No (< 48h threshold) |
| report | 24 hours | 50 hours ago | - | 50h | Yes (> 48h threshold) |

## Consecutive Failure Alerting

The `consecutive_failures` counter is the primary alerting mechanism:

- Reset to 0 on every success.
- Incremented by 1 on every failure.
- When it reaches the threshold (default: 3), `LogError` is emitted with the `ALERT` prefix.

This LogError is picked up by the log pipeline (Serilog -> RMQ -> Elasticsearch) and can trigger alerts in Kibana or any monitoring system that watches for error-level logs.

### Threshold Configuration

The alert threshold can be made dynamic via settings:

```
jobs:{job-name}:alert-threshold
```

Default: 3. For critical jobs (payment processing), set to 1. For non-critical jobs (report generation), set to 5.

## Health Endpoint

The Worker exposes a health endpoint that returns the status of all jobs. This is used by Kubernetes liveness/readiness probes and monitoring dashboards.

### Implementation in Program.cs

```csharp
// Minimal API health endpoint in Worker
var app = builder.Build();

app.MapGet("/health/jobs", async (IJobHealthTracker tracker, CancellationToken ct) =>
{
    var allHealth = await tracker.GetAllHealthAsync(ct);

    var response = allHealth.Select(h => new
    {
        h.JobName,
        h.Status,
        h.LastRun,
        h.LastSuccess,
        h.LastFailure,
        h.LastFailureReason,
        h.ConsecutiveFailures,
        h.LastDurationMs,
    });

    return Results.Ok(new { jobs = response });
});
```

### Response Example

```json
{
  "jobs": [
    {
      "jobName": "expired-order-cleanup",
      "status": "succeeded",
      "lastRun": "2026-04-14T06:00:01Z",
      "lastSuccess": "2026-04-14T06:00:03Z",
      "lastFailure": null,
      "lastFailureReason": null,
      "consecutiveFailures": 0,
      "lastDurationMs": 1823
    },
    {
      "jobName": "daily-report",
      "status": "failed",
      "lastRun": "2026-04-14T00:00:01Z",
      "lastSuccess": "2026-04-13T00:00:02Z",
      "lastFailure": "2026-04-14T00:00:05Z",
      "lastFailureReason": "API returned 500: Internal Server Error",
      "consecutiveFailures": 2,
      "lastDurationMs": 4521
    }
  ]
}
```

## Job Registry

The `jobs:__registry` Redis set tracks all known job names. When a job calls `RecordRunStartAsync` for the first time, its name is added to the registry via `SADD`. This enables `GetAllHealthAsync` to discover all jobs without hardcoding a list.

```bash
# Redis CLI:
SMEMBERS jobs:__registry
# 1) "expired-order-cleanup"
# 2) "daily-report"
# 3) "inactive-user-reminder"
```

## Important Rules

1. **Every job calls all three health methods.** `RecordRunStartAsync` before execution, `RecordSuccessAsync` or `RecordFailureAsync` after.
2. **Health data has no TTL.** It persists until the next run updates it. This is intentional -- stale detection relies on comparing `last_run` to current time.
3. **Consecutive failures reset on success.** A single success clears the failure counter. This is the desired behavior -- transient failures should not accumulate across successful runs.
4. **Alert threshold is configurable.** Different jobs have different criticality levels.
5. **Health endpoint is unauthenticated.** It is an internal endpoint, exposed only within the Docker network. Kubernetes probes and monitoring tools access it directly.
6. **Health tracking failures should not crash the job.** If Redis is down and health tracking fails, the job should still attempt to run. Wrap health calls in try-catch if Redis availability is uncertain.
