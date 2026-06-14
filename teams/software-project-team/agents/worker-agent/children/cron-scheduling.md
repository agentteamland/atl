---
knowledge-base-summary: "Cronos library for cron expression parsing. Expressions stored in dynamic settings (not hardcoded). Timezone handling. Standard cron format (5-field). Common patterns: every minute, hourly, daily at midnight, weekly, monthly."
---
# Cron Scheduling: Cronos Library

## Library

[Cronos](https://github.com/HangfireIO/Cronos) is the cron expression parser. It is lightweight, has no dependencies, supports standard 5-field and optional 6-field (with seconds) formats, and handles timezone conversions. It does NOT schedule anything by itself -- it only parses expressions and calculates next occurrences. The scheduling loop is our own `BackgroundService` code.

## Core API

### Parsing a Cron Expression

```csharp
using Cronos;

// Standard 5-field format: minute hour day-of-month month day-of-week
var cron = CronExpression.Parse("0 */6 * * *"); // every 6 hours

// 6-field format with seconds (opt-in)
var cronWithSeconds = CronExpression.Parse(
    "*/30 * * * * *", CronFormat.IncludeSeconds); // every 30 seconds
```

**We use 5-field format by default.** The 6-field format with seconds is only used when sub-minute precision is genuinely needed (rare). Do not default to seconds format -- it creates unnecessary complexity and most jobs run on minute-or-longer intervals.

### Calculating Next Occurrence

```csharp
var now = DateTime.UtcNow;
var next = cron.GetNextOccurrence(now, TimeZoneInfo.Utc);

if (next.HasValue)
{
    var delay = next.Value - now;
    await Task.Delay(delay, stoppingToken);
    // cron tick fired -- execute the job
}
```

**Always pass `TimeZoneInfo.Utc`.** Internally everything is UTC. The cron expression itself is written in UTC terms. If the business requirement says "run at 9am Istanbul time", convert to UTC when defining the cron expression (6am UTC in winter, 6am UTC in summer -- handle DST at definition time, not at runtime).

## Common Cron Patterns

| Pattern | Expression | Notes |
|---------|-----------|-------|
| Every minute | `* * * * *` | Use sparingly -- high frequency |
| Every 5 minutes | `*/5 * * * *` | Good for polling-style jobs |
| Every 15 minutes | `*/15 * * * *` | Common for sync jobs |
| Every 30 minutes | `*/30 * * * *` | |
| Every hour (on the hour) | `0 * * * *` | |
| Every 2 hours | `0 */2 * * *` | |
| Every 6 hours | `0 */6 * * *` | Common for cleanup jobs |
| Daily at midnight UTC | `0 0 * * *` | Common for reports, daily cleanup |
| Daily at 6am UTC | `0 6 * * *` | Common for morning tasks |
| Weekdays at 9am UTC | `0 9 * * 1-5` | Mon-Fri only |
| Weekly on Monday at midnight | `0 0 * * 1` | Weekly reports |
| First day of month at midnight | `0 0 1 * *` | Monthly reports |
| Every Sunday at 3am UTC | `0 3 * * 0` | Weekly maintenance |

## Dynamic Settings Integration

Cron expressions are NEVER hardcoded in the job class. They come from dynamic settings (stored in DB, cached in Redis, served via API). This allows changing a job's schedule without redeployment.

### Settings Key Convention

```
jobs:{job-name}:cron
```

Examples:
- `jobs:expired-order-cleanup:cron` = `"0 */6 * * *"`
- `jobs:daily-report:cron` = `"0 0 * * *"`
- `jobs:inactive-user-reminder:cron` = `"0 9 * * 1-5"`

### Reading Cron from Settings

```csharp
private const string CronSettingsKey = "jobs:expired-order-cleanup:cron";
private const string DefaultCron = "0 */6 * * *";

// Inside the main loop -- read EVERY cycle for hot-reload
while (!stoppingToken.IsCancellationRequested)
{
    var cronExpr = await _settings.GetAsync(CronSettingsKey, stoppingToken)
                   ?? DefaultCron;
    var cron = CronExpression.Parse(cronExpr);

    var now = DateTime.UtcNow;
    var next = cron.GetNextOccurrence(now, TimeZoneInfo.Utc);
    // ...
}
```

### Why read on every cycle?

Because the cron expression might change in dynamic settings while the job is running. By reading at the top of each loop iteration:

1. Admin changes cron from `"0 */6 * * *"` to `"*/30 * * * *"` via the settings API.
2. The job finishes the current delay, wakes up, reads the new cron at the top of the loop.
3. Next occurrence is calculated with the new expression -- no restart needed.

The cost of reading a string from settings (Redis GET) is negligible compared to the job's execution or the delay duration.

### Default Value is Mandatory

Every job defines a `const string DefaultCron` as a fallback. If the settings service is unavailable or the key does not exist, the job still runs on a sensible default. The default should be conservative (less frequent rather than more frequent).

## Timezone Handling

### Rule: UTC everywhere internally

```csharp
// CORRECT
var now = DateTime.UtcNow;
var next = cron.GetNextOccurrence(now, TimeZoneInfo.Utc);

// WRONG -- local time will drift on server timezone changes
var now = DateTime.Now;
var next = cron.GetNextOccurrence(now, TimeZoneInfo.Local);
```

### DST and Business Hours

If the business requires "run at 9am local time" where local time has DST:

1. Option A (recommended): Define two cron expressions in settings -- one for summer, one for winter. Switch via settings when DST changes. Simple, explicit, no surprises.
2. Option B: Use `TimeZoneInfo.FindSystemTimeZoneById("Europe/Istanbul")` in `GetNextOccurrence`. Works but creates implicit DST handling that can surprise operators.

For most background jobs, the exact hour does not matter ("roughly every 6 hours", "roughly at midnight"). UTC is sufficient.

## Error Handling for Invalid Cron Expressions

If someone puts an invalid string in the settings, `CronExpression.Parse()` throws `CronFormatException`. Handle this gracefully:

```csharp
CronExpression cron;
try
{
    var cronExpr = await _settings.GetAsync(CronSettingsKey, stoppingToken)
                   ?? DefaultCron;
    cron = CronExpression.Parse(cronExpr);
}
catch (CronFormatException ex)
{
    _logger.LogError(ex,
        "{Job} invalid cron expression in settings key {Key}, falling back to default {Default}",
        JobName, CronSettingsKey, DefaultCron);
    cron = CronExpression.Parse(DefaultCron);
}
```

Never let an invalid settings value crash the job. Log the error, fall back to default, keep running.

## Important Rules

1. **5-field format is the default.** Only use `CronFormat.IncludeSeconds` when the job genuinely needs sub-minute precision.
2. **UTC everywhere.** No local time, no timezone surprises.
3. **Read cron from settings every cycle.** Enables hot-reload without restart.
4. **Always have a default cron fallback.** Settings unavailable should not crash the job.
5. **Validate cron expression from settings.** Catch `CronFormatException` and fall back gracefully.
6. **Do not use Cronos for scheduling.** It is a parser only. The scheduling loop (`while + Task.Delay`) is our own code in `BackgroundService`.
