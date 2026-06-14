---
knowledge-base-summary: "The primary production unit of this agent. Template + checklist + naming conventions for creating new scheduled jobs. Read this FIRST when adding any new job. Covers: BackgroundService skeleton, cron parsing, API call, distributed lock, error handling, logging, idempotency."
---
# Job Blueprint: The Primary Production Unit

## Purpose

This is the blueprint for every scheduled job in the Worker. Every new job follows this exact template. No shortcuts, no variations. A job is a thin scheduler shell: wait for cron tick, acquire lock, call API, log result. That's it.

> **i18n note:** jobs that trigger user-facing emails or push notifications don't localize themselves — they call the API, and the API publishes EmailJob / PushNotificationJob RMQ messages with the per-locale envelope defined in `api-agent/children/user-facing-strings.md`. A job must NEVER assemble user-facing text in English and hand it to MailSender directly; always go through the keyed envelope so the user's locale is respected.

## Naming Conventions

| What | Convention | Example |
|------|-----------|---------|
| Class name | `{Feature}Job` | `ExpiredOrderCleanupJob`, `DailyReportJob`, `InactiveUserReminderJob` |
| File path | `Jobs/{JobName}.cs` | `Jobs/ExpiredOrderCleanupJob.cs` |
| Settings key (cron) | `jobs:{kebab-case}:cron` | `jobs:expired-order-cleanup:cron` |
| Settings key (lock TTL) | `jobs:{kebab-case}:lock-ttl-seconds` | `jobs:expired-order-cleanup:lock-ttl-seconds` |
| Lock key | `lock:jobs:{kebab-case}` | `lock:jobs:expired-order-cleanup` |
| Health key | `jobs:{kebab-case}:health` | `jobs:expired-order-cleanup:health` |
| Log prefix | `{JobName}` | Every log message starts with the job name |

## Full Job Template

```csharp
using System.Diagnostics;
using Cronos;

namespace ProjectName.Worker.Jobs;

public sealed class ExpiredOrderCleanupJob : BackgroundService
{
    private readonly IApiClient _apiClient;
    private readonly IDistributedLock _distributedLock;
    private readonly IJobHealthTracker _healthTracker;
    private readonly ISettingsService _settings;
    private readonly ILogger<ExpiredOrderCleanupJob> _logger;

    private const string JobName = "expired-order-cleanup";
    private const string LockKey = $"lock:jobs:{JobName}";
    private const string CronSettingsKey = $"jobs:{JobName}:cron";
    private const string LockTtlSettingsKey = $"jobs:{JobName}:lock-ttl-seconds";
    private const string DefaultCron = "0 */6 * * *"; // every 6 hours
    private const int DefaultLockTtlSeconds = 300; // 5 minutes

    public ExpiredOrderCleanupJob(
        IApiClient apiClient,
        IDistributedLock distributedLock,
        IJobHealthTracker healthTracker,
        ISettingsService settings,
        ILogger<ExpiredOrderCleanupJob> logger)
    {
        _apiClient = apiClient;
        _distributedLock = distributedLock;
        _healthTracker = healthTracker;
        _settings = settings;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("{Job} started, waiting for first cron tick", JobName);

        while (!stoppingToken.IsCancellationRequested)
        {
            // 1. Read cron expression from dynamic settings (hot-reload)
            var cronExpr = await _settings.GetAsync(CronSettingsKey, stoppingToken)
                           ?? DefaultCron;
            var cron = CronExpression.Parse(cronExpr);

            // 2. Calculate next occurrence and wait
            var now = DateTime.UtcNow;
            var nextOccurrence = cron.GetNextOccurrence(now, TimeZoneInfo.Utc);
            if (!nextOccurrence.HasValue)
            {
                _logger.LogWarning("{Job} cron returned no next occurrence, stopping", JobName);
                break;
            }

            var delay = nextOccurrence.Value - now;
            _logger.LogDebug("{Job} next run at {NextRun} (in {Delay})",
                JobName, nextOccurrence.Value, delay);

            try
            {
                await Task.Delay(delay, stoppingToken);
            }
            catch (OperationCanceledException)
            {
                break;
            }

            // 3. Acquire distributed lock
            var lockTtlSeconds = await _settings.GetAsync<int>(
                LockTtlSettingsKey, stoppingToken) ?? DefaultLockTtlSeconds;
            var lockTtl = TimeSpan.FromSeconds(lockTtlSeconds);

            var lockAcquired = await _distributedLock.TryAcquireAsync(
                LockKey, lockTtl, stoppingToken);

            if (!lockAcquired)
            {
                _logger.LogInformation(
                    "{Job} skipping cycle, another instance holds the lock", JobName);
                continue;
            }

            // 4. Execute the job
            var sw = Stopwatch.StartNew();
            try
            {
                _logger.LogInformation("{Job} execution started", JobName);
                await _healthTracker.RecordRunStartAsync(JobName, stoppingToken);

                await ExecuteJobAsync(stoppingToken);

                sw.Stop();
                _logger.LogInformation(
                    "{Job} execution completed in {ElapsedMs}ms",
                    JobName, sw.ElapsedMilliseconds);
                await _healthTracker.RecordSuccessAsync(JobName, stoppingToken);
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                _logger.LogWarning("{Job} cancelled during execution", JobName);
                break;
            }
            catch (Exception ex)
            {
                sw.Stop();
                _logger.LogError(ex,
                    "{Job} execution failed after {ElapsedMs}ms",
                    JobName, sw.ElapsedMilliseconds);
                await _healthTracker.RecordFailureAsync(JobName, ex.Message, stoppingToken);
            }
            finally
            {
                await _distributedLock.ReleaseAsync(LockKey, stoppingToken);
            }
        }

        _logger.LogInformation("{Job} stopped", JobName);
    }

    private async Task ExecuteJobAsync(CancellationToken ct)
    {
        // THIS is the only part that changes per job.
        // Call the API — the API does the actual work.
        var response = await _apiClient.PostAsync(
            "/api/internal/orders/cleanup-expired",
            new { },
            ct);

        if (!response.IsSuccessStatusCode)
        {
            var body = await response.Content.ReadAsStringAsync(ct);
            throw new InvalidOperationException(
                $"API returned {response.StatusCode}: {body}");
        }

        var result = await response.Content
            .ReadFromJsonAsync<CleanupResult>(ct);

        _logger.LogInformation(
            "{Job} cleaned up {Count} expired orders",
            JobName, result?.CleanedCount ?? 0);
    }

    private record CleanupResult(int CleanedCount);
}
```

## Registration in Program.cs

Every new job gets a single line in Program.cs:

```csharp
// Jobs
builder.Services.AddHostedService<ExpiredOrderCleanupJob>();
builder.Services.AddHostedService<DailyReportJob>();
builder.Services.AddHostedService<InactiveUserReminderJob>();
```

**Order does not matter.** Each job runs independently on its own cron schedule.

## New Job Checklist

Before merging any new job, verify EVERY item:

### Settings

- [ ] Cron expression defined in dynamic settings? Key: `jobs:{job-name}:cron`
- [ ] Lock TTL defined in dynamic settings? Key: `jobs:{job-name}:lock-ttl-seconds`
- [ ] Default cron and default lock TTL defined as `const` in the job class?

### Reliability

- [ ] Distributed lock key defined? Pattern: `lock:jobs:{job-name}`
- [ ] Idempotent? Running twice in a row produces the same result?
- [ ] Lock TTL is longer than maximum expected job duration?

### Cancellation

- [ ] `CancellationToken` passed to every async call?
- [ ] `OperationCanceledException` caught in the main loop?
- [ ] `OperationCanceledException` caught separately in the execution block (to distinguish from job errors)?

### Logging

- [ ] Job start log? (`{Job} execution started`)
- [ ] Job end log with duration? (`{Job} execution completed in {ElapsedMs}ms`)
- [ ] Job error log with exception? (`{Job} execution failed after {ElapsedMs}ms`)
- [ ] Lock skip log? (`{Job} skipping cycle, another instance holds the lock`)
- [ ] Business result log? (e.g., `{Job} cleaned up {Count} expired orders`)

### API

- [ ] API endpoint exists that the job calls?
- [ ] API endpoint is internal-only? (protected by `X-Internal-Token`)
- [ ] Response type defined? (record for deserialization)

### Health

- [ ] `RecordRunStartAsync` called before execution?
- [ ] `RecordSuccessAsync` called on success?
- [ ] `RecordFailureAsync` called on failure?

### Registration

- [ ] `builder.Services.AddHostedService<{JobName}>()` added in Program.cs?

## Anti-Patterns

### Never: Database access in the job

```csharp
// WRONG — Worker has no DbContext, no Infrastructure reference
var orders = await _db.Orders.Where(o => o.IsExpired).ToListAsync(ct);
foreach (var order in orders) { order.Status = "Cancelled"; }
await _db.SaveChangesAsync(ct);

// CORRECT — Call the API, the API does the work
await _apiClient.PostAsync("/api/internal/orders/cleanup-expired", new { }, ct);
```

### Never: Business logic in the job

```csharp
// WRONG — Business logic belongs in the API handler
if (order.CreatedAt < DateTime.UtcNow.AddDays(-30) && order.Status == "Pending")
{
    // calculate refund, send email, update status...
}

// CORRECT — The job only triggers; all logic is in the API
await _apiClient.PostAsync("/api/internal/orders/cleanup-expired", new { }, ct);
```

### Never: Hardcoded cron expression

```csharp
// WRONG — Cron hardcoded, requires redeployment to change
private readonly CronExpression _cron = CronExpression.Parse("0 */6 * * *");

// CORRECT — Read from dynamic settings, hot-reloadable
var cronExpr = await _settings.GetAsync(CronSettingsKey, ct) ?? DefaultCron;
var cron = CronExpression.Parse(cronExpr);
```

### Never: Swallowing exceptions silently

```csharp
// WRONG — Exception swallowed, health not tracked
catch (Exception) { /* do nothing */ }

// CORRECT — Log the error, track in health
catch (Exception ex)
{
    _logger.LogError(ex, "{Job} execution failed after {ElapsedMs}ms",
        JobName, sw.ElapsedMilliseconds);
    await _healthTracker.RecordFailureAsync(JobName, ex.Message, ct);
}
```

### Never: Missing distributed lock

```csharp
// WRONG — No lock, two pods run the same job simultaneously
await ExecuteJobAsync(stoppingToken);

// CORRECT — Lock acquired first, only one pod runs
var lockAcquired = await _distributedLock.TryAcquireAsync(LockKey, lockTtl, ct);
if (!lockAcquired) { /* skip cycle */ continue; }
try { await ExecuteJobAsync(ct); }
finally { await _distributedLock.ReleaseAsync(LockKey, ct); }
```

## Job Lifecycle Flow

```
Host starts
    ↓
ExecuteAsync(stoppingToken) begins
    ↓
LOOP:
    ├── Read cron from settings (dynamic, hot-reloadable)
    ├── Calculate next occurrence (UTC)
    ├── Task.Delay until next tick
    ├── Try acquire distributed lock
    │   ├── Lock NOT acquired → log "skipping", continue to LOOP
    │   └── Lock acquired ↓
    ├── Record health: run started
    ├── Stopwatch.Start
    ├── Call API endpoint
    │   ├── Success → log result + record health success
    │   └── Failure → log error + record health failure
    ├── Stopwatch.Stop, log duration
    └── Release lock (in finally block)
    ↓
stoppingToken cancelled → break out of LOOP
    ↓
Log "{Job} stopped"
```
