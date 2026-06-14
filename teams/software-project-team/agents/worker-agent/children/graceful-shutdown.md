---
knowledge-base-summary: "CancellationToken handling throughout the job lifecycle. Host.StopAsync signals cancellation → jobs finish current iteration → clean exit. No mid-operation kills. Drain pattern for long-running batch operations."
---
# Graceful Shutdown: CancellationToken Discipline

## The Problem

When a container is stopped (deploy, scale-down, rolling restart), the host sends a shutdown signal. If a job is mid-execution (waiting for an API response, processing a batch), it must stop cleanly. A hard kill mid-operation can leave data in an inconsistent state, hold distributed locks indefinitely (until TTL), or produce partial results.

## How .NET Generic Host Shutdown Works

```
SIGTERM / docker stop / Ctrl+C
    ↓
IHostApplicationLifetime.StopApplication()
    ↓
Host calls StopAsync() on all hosted services
    ↓
BackgroundService.StopAsync() triggers the CancellationToken (stoppingToken)
    ↓
ExecuteAsync should observe stoppingToken and exit
    ↓
Host waits up to ShutdownTimeout (default: 30s) for all services to stop
    ↓
If services haven't stopped → force kill
```

## ShutdownTimeout Configuration

```csharp
// In Program.cs
builder.Services.Configure<HostOptions>(options =>
{
    options.ShutdownTimeout = TimeSpan.FromSeconds(30); // default is 30s, adjust if needed
});
```

For jobs that process large batches, increase this to give them time to finish the current item:

```csharp
// For long-running batch jobs
builder.Services.Configure<HostOptions>(options =>
{
    options.ShutdownTimeout = TimeSpan.FromSeconds(60);
});
```

**Never set this too high.** Kubernetes/Docker has its own grace period (default 30s for Kubernetes). If your app's ShutdownTimeout exceeds the orchestrator's grace period, the container gets force-killed anyway.

## CancellationToken Rules

### Rule 1: Pass stoppingToken to EVERY async call

```csharp
// CORRECT -- every async call gets the token
await Task.Delay(delay, stoppingToken);
await _apiClient.PostAsync("/api/internal/cleanup", body, stoppingToken);
await _distributedLock.TryAcquireAsync(key, ttl, stoppingToken);
await _healthTracker.RecordSuccessAsync(jobName, stoppingToken);

// WRONG -- no cancellation token, call will hang during shutdown
await Task.Delay(delay);
await _apiClient.PostAsync("/api/internal/cleanup", body);
```

### Rule 2: Check IsCancellationRequested in the main loop

```csharp
while (!stoppingToken.IsCancellationRequested)
{
    // ... wait for cron tick, acquire lock, execute job ...
}
```

### Rule 3: Catch OperationCanceledException at two levels

```csharp
protected override async Task ExecuteAsync(CancellationToken stoppingToken)
{
    while (!stoppingToken.IsCancellationRequested)
    {
        // Level 1: Cancellation during Task.Delay (waiting for cron tick)
        try
        {
            await Task.Delay(delay, stoppingToken);
        }
        catch (OperationCanceledException)
        {
            break; // clean exit from loop
        }

        try
        {
            await ExecuteJobAsync(stoppingToken);
        }
        // Level 2: Cancellation during job execution
        catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
        {
            _logger.LogWarning("{Job} cancelled during execution", JobName);
            break; // clean exit
        }
        catch (Exception ex)
        {
            // This is a real error, not a shutdown
            _logger.LogError(ex, "{Job} execution failed", JobName);
        }
    }
}
```

**Why two levels?**

- **Level 1 (during delay):** The job is sleeping, waiting for the next cron tick. Cancellation here is clean -- nothing was started, just break.
- **Level 2 (during execution):** The job is running. The `when (stoppingToken.IsCancellationRequested)` guard distinguishes "shutdown cancellation" from "HTTP timeout TaskCanceledException". Without the guard, a genuine HTTP timeout would silently stop the job.

## Drain Pattern for Batch Jobs

Some jobs process a collection of items (e.g., "send reminder to all inactive users"). The drain pattern processes items one at a time and checks cancellation between items:

```csharp
private async Task ExecuteJobAsync(CancellationToken ct)
{
    // Get the list of items to process from API
    var response = await _apiClient.GetAsync(
        "/api/internal/users/inactive-list", ct);
    response.EnsureSuccessStatusCode();

    var users = await response.Content
        .ReadFromJsonAsync<List<InactiveUser>>(ct);

    if (users is null || users.Count == 0)
    {
        _logger.LogInformation("{Job} no inactive users to process", JobName);
        return;
    }

    _logger.LogInformation(
        "{Job} processing {Count} inactive users", JobName, users.Count);

    var processed = 0;
    var failed = 0;

    foreach (var user in users)
    {
        // CHECK CANCELLATION BEFORE STARTING NEXT ITEM
        if (ct.IsCancellationRequested)
        {
            _logger.LogWarning(
                "{Job} shutdown requested, stopping after {Processed}/{Total} users",
                JobName, processed, users.Count);
            break; // finish current, don't start next
        }

        try
        {
            await _apiClient.PostAsync(
                "/api/internal/users/send-reminder",
                new { UserId = user.Id },
                ct);
            processed++;
        }
        catch (OperationCanceledException) when (ct.IsCancellationRequested)
        {
            _logger.LogWarning(
                "{Job} cancelled mid-request for user {UserId}", JobName, user.Id);
            break;
        }
        catch (Exception ex)
        {
            _logger.LogError(ex,
                "{Job} failed to process user {UserId}", JobName, user.Id);
            failed++;
            // Continue to next user -- don't let one failure stop the batch
        }
    }

    _logger.LogInformation(
        "{Job} batch completed: {Processed} processed, {Failed} failed, {Remaining} remaining",
        JobName, processed, failed, users.Count - processed - failed);
}
```

### Key points of the drain pattern:

1. **Check `ct.IsCancellationRequested` before each item.** Not after -- before. "Finish current, don't start next."
2. **Individual item failure does not stop the batch.** Log the error, increment failure counter, continue.
3. **Cancellation mid-request is caught separately.** If we're cancelled while the HTTP call is in flight, we catch it, log it, and break.
4. **Final summary log includes remaining count.** This tells operators how many items were left unprocessed due to shutdown. They'll be picked up on the next run.

## Chunk Processing Pattern

For very large batches, process in chunks rather than one at a time. This reduces memory usage and allows the API to process items in bulk:

```csharp
private async Task ExecuteJobAsync(CancellationToken ct)
{
    const int chunkSize = 100;
    var offset = 0;
    var totalProcessed = 0;

    while (!ct.IsCancellationRequested)
    {
        // Fetch a chunk
        var response = await _apiClient.GetAsync(
            $"/api/internal/orders/stale?offset={offset}&limit={chunkSize}", ct);
        response.EnsureSuccessStatusCode();

        var chunk = await response.Content
            .ReadFromJsonAsync<List<StaleOrder>>(ct);

        if (chunk is null || chunk.Count == 0)
        {
            _logger.LogInformation(
                "{Job} no more items, total processed: {Total}",
                JobName, totalProcessed);
            break; // no more items
        }

        // Process the chunk via API
        var result = await _apiClient.PostAsync(
            "/api/internal/orders/process-stale-batch",
            new { OrderIds = chunk.Select(o => o.Id).ToList() },
            ct);
        result.EnsureSuccessStatusCode();

        totalProcessed += chunk.Count;
        offset += chunkSize;

        _logger.LogInformation(
            "{Job} processed chunk of {ChunkSize}, total so far: {Total}",
            JobName, chunk.Count, totalProcessed);

        // Check cancellation BETWEEN chunks
        if (ct.IsCancellationRequested)
        {
            _logger.LogWarning(
                "{Job} shutdown requested after processing {Total} items",
                JobName, totalProcessed);
            break;
        }
    }
}
```

## Lock Release During Shutdown

The distributed lock must be released even during shutdown. The `finally` block in the main loop handles this:

```csharp
try
{
    await ExecuteJobAsync(stoppingToken);
}
catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
{
    _logger.LogWarning("{Job} cancelled during execution", JobName);
    break;
}
finally
{
    // Release lock even during shutdown
    await _distributedLock.ReleaseAsync(LockKey, CancellationToken.None);
    // ^^^ Note: CancellationToken.None here, not stoppingToken
    // We MUST release the lock even when shutting down
}
```

**Why `CancellationToken.None` for lock release?** Because the stoppingToken is already cancelled at this point. If we pass it to `ReleaseAsync`, the Redis call would be cancelled immediately and the lock would stay in Redis until TTL expires. Lock release is a critical cleanup step that must complete regardless of shutdown state.

## Timeline of a Graceful Shutdown

```
T+0s   SIGTERM received
         ↓
T+0s   Host calls StopAsync() on all BackgroundServices
         ↓
T+0s   stoppingToken is cancelled
         ↓
       Job A: sleeping (Task.Delay) → OperationCanceledException → breaks loop → exits
       Job B: mid-execution → finishes current API call → checks ct → breaks loop → exits
       Job C: processing batch item 45/100 → finishes item 45 → checks ct → breaks → exits
         ↓
T+2s   All jobs have exited their ExecuteAsync methods
         ↓
T+2s   Locks released in finally blocks
         ↓
T+2s   Host completes shutdown
```

Worst case (job mid-HTTP-call):
```
T+0s   SIGTERM received, stoppingToken cancelled
T+0s   HTTP call in flight (already sent request to API)
T+~1s  API responds, job gets response
T+~1s  Job checks ct.IsCancellationRequested → true → breaks loop
T+~1s  Lock released, job exits
T+~2s  Host completes shutdown
```

## Important Rules

1. **Every async call gets `stoppingToken`.** No exceptions. Missing tokens mean operations that hang during shutdown.
2. **Two-level cancellation catching.** Level 1 for delay, Level 2 for execution. Each has different semantics.
3. **Drain pattern for batches.** Check cancellation BEFORE starting next item, not after.
4. **Lock release uses `CancellationToken.None`.** This is the one place where we intentionally ignore cancellation.
5. **ShutdownTimeout must be realistic.** Long enough for the longest single operation, short enough that the orchestrator doesn't force-kill.
6. **Never use `Environment.Exit()` or `Process.Kill()`.** Always let the host manage shutdown through the cancellation token.
7. **Log the shutdown.** When a job stops due to cancellation, log what was in progress and what was left unfinished.
