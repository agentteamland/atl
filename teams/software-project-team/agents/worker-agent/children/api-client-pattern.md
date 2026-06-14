---
knowledge-base-summary: "Worker → API communication via typed HttpClient. IApiClient interface, ApiClient implementation. InternalTokenHandler (DelegatingHandler) auto-injects X-Internal-Token on every request. Same pattern as Socket Agent but from Worker context."
---
# API Client Pattern: Worker to API Communication

## Core Principle

The Worker NEVER accesses the database directly. It has no reference to Domain, Application, or Infrastructure. Every operation goes through the API via HTTP. The Worker is a scheduler that triggers API endpoints at the right time -- the API contains all business logic.

## Architecture

```
Worker (scheduler)
    ↓ HTTP (with X-Internal-Token header)
API (business logic)
    ↓
Database / Redis / RMQ / etc.
```

## IApiClient Interface

Defined in the Worker project. Generic enough for all jobs, typed enough for safety:

```csharp
namespace ProjectName.Worker.Services;

public interface IApiClient
{
    Task<HttpResponseMessage> GetAsync(string path, CancellationToken ct = default);

    Task<HttpResponseMessage> PostAsync<T>(
        string path, T body, CancellationToken ct = default);

    Task<HttpResponseMessage> PutAsync<T>(
        string path, T body, CancellationToken ct = default);

    Task<HttpResponseMessage> DeleteAsync(
        string path, CancellationToken ct = default);
}
```

**Why generic `HttpResponseMessage` return type?** Because different jobs need different response types. The job deserializes the response itself based on its specific needs. The client stays generic.

## ApiClient Implementation

```csharp
using System.Net.Http.Json;

namespace ProjectName.Worker.Services;

public sealed class ApiClient : IApiClient
{
    private readonly HttpClient _httpClient;

    public ApiClient(HttpClient httpClient)
    {
        _httpClient = httpClient;
    }

    public async Task<HttpResponseMessage> GetAsync(
        string path, CancellationToken ct = default)
    {
        return await _httpClient.GetAsync(path, ct);
    }

    public async Task<HttpResponseMessage> PostAsync<T>(
        string path, T body, CancellationToken ct = default)
    {
        return await _httpClient.PostAsJsonAsync(path, body, ct);
    }

    public async Task<HttpResponseMessage> PutAsync<T>(
        string path, T body, CancellationToken ct = default)
    {
        return await _httpClient.PutAsJsonAsync(path, body, ct);
    }

    public async Task<HttpResponseMessage> DeleteAsync(
        string path, CancellationToken ct = default)
    {
        return await _httpClient.DeleteAsync(path, ct);
    }
}
```

No retry logic, no error handling, no logging inside the client. The client is a thin HTTP wrapper. Retry and error decisions belong to the individual job because different jobs have different strategies (retry cleanup vs skip report vs alert on payment failure).

## InternalTokenHandler (DelegatingHandler)

The `X-Internal-Token` header is injected automatically on every outgoing request. No job ever sets this header manually:

```csharp
namespace ProjectName.Worker.Services;

public sealed class InternalTokenHandler : DelegatingHandler
{
    private readonly string _internalToken;

    public InternalTokenHandler(string internalToken)
    {
        _internalToken = internalToken;
    }

    protected override async Task<HttpResponseMessage> SendAsync(
        HttpRequestMessage request,
        CancellationToken cancellationToken)
    {
        request.Headers.Remove("X-Internal-Token");
        request.Headers.Add("X-Internal-Token", _internalToken);
        return await base.SendAsync(request, cancellationToken);
    }
}
```

**Why `Remove` then `Add`?** Defensive coding. If somehow the header already exists (bug, test double), we replace it rather than throwing a duplicate header exception.

## Registration in Program.cs

```csharp
// IApiClient with InternalTokenHandler
var apiBaseUrl = builder.Configuration["ApiClient:BaseUrl"]
    ?? "http://api:3000";
var internalToken = builder.Configuration["InternalToken"]
    ?? "internal-secret-change-in-production";

builder.Services.AddTransient(_ => new InternalTokenHandler(internalToken));
builder.Services.AddHttpClient<IApiClient, ApiClient>(client =>
{
    client.BaseAddress = new Uri(apiBaseUrl);
    client.Timeout = TimeSpan.FromSeconds(30);
})
.AddHttpMessageHandler<InternalTokenHandler>();
```

### What this does:

1. `AddHttpClient<IApiClient, ApiClient>` -- registers a typed HttpClient. Each `IApiClient` injection gets a properly configured `HttpClient` from the factory.
2. `client.BaseAddress = new Uri(apiBaseUrl)` -- base URL set once. Jobs use relative paths: `/api/internal/orders/cleanup`.
3. `.AddHttpMessageHandler<InternalTokenHandler>()` -- the handler is part of the HTTP pipeline. Every request automatically gets the `X-Internal-Token` header.
4. `client.Timeout = TimeSpan.FromSeconds(30)` -- global timeout. Individual jobs can override per-request if needed.

### Configuration (appsettings.json / environment variables)

```json
{
  "ApiClient": {
    "BaseUrl": "http://api:3000"
  },
  "InternalToken": "internal-secret-change-in-production"
}
```

In Docker Compose, these come from environment variables:
```yaml
worker:
  environment:
    - ApiClient__BaseUrl=http://api:3000
    - InternalToken=${INTERNAL_TOKEN}
```

## Usage in a Job

```csharp
private async Task ExecuteJobAsync(CancellationToken ct)
{
    // POST with body
    var response = await _apiClient.PostAsync(
        "/api/internal/orders/cleanup-expired",
        new { OlderThanDays = 30 },
        ct);

    if (!response.IsSuccessStatusCode)
    {
        var errorBody = await response.Content.ReadAsStringAsync(ct);
        _logger.LogError(
            "{Job} API returned {StatusCode}: {Body}",
            JobName, (int)response.StatusCode, errorBody);
        throw new InvalidOperationException(
            $"API returned {response.StatusCode}");
    }

    var result = await response.Content
        .ReadFromJsonAsync<CleanupResult>(ct);

    _logger.LogInformation(
        "{Job} API returned: {CleanedCount} orders cleaned",
        JobName, result?.CleanedCount ?? 0);
}

// GET example
private async Task CheckHealthAsync(CancellationToken ct)
{
    var response = await _apiClient.GetAsync("/api/health", ct);
    response.EnsureSuccessStatusCode();
}
```

## Error Handling Strategy

The job decides what to do when the API returns an error. Different jobs have different strategies:

### Strategy 1: Log and Skip (most common)

The job logs the error, records a health failure, and waits for the next cron tick. Used for non-critical periodic jobs (cleanup, reports).

```csharp
catch (Exception ex)
{
    _logger.LogError(ex, "{Job} failed, will retry on next cycle", JobName);
    await _healthTracker.RecordFailureAsync(JobName, ex.Message, ct);
    // No rethrow -- loop continues to next cycle
}
```

### Strategy 2: Retry with Backoff

The job retries a fixed number of times with increasing delay. Used for important but recoverable operations (sync, data push).

```csharp
private async Task ExecuteWithRetryAsync(CancellationToken ct)
{
    var maxRetries = 3;
    for (var attempt = 1; attempt <= maxRetries; attempt++)
    {
        try
        {
            await ExecuteJobAsync(ct);
            return; // success
        }
        catch (Exception ex) when (attempt < maxRetries)
        {
            var backoff = TimeSpan.FromSeconds(Math.Pow(2, attempt));
            _logger.LogWarning(ex,
                "{Job} attempt {Attempt}/{MaxRetries} failed, retrying in {Backoff}",
                JobName, attempt, maxRetries, backoff);
            await Task.Delay(backoff, ct);
        }
    }
    // Final attempt -- let exception propagate to the outer catch
    await ExecuteJobAsync(ct);
}
```

### Strategy 3: Alert on Failure

The job logs an error at critical level or triggers a notification. Used for business-critical operations (payment processing, SLA-bound tasks). The alert mechanism is a separate API call or a health tracker threshold.

```csharp
catch (Exception ex)
{
    _logger.LogCritical(ex,
        "{Job} CRITICAL FAILURE, requires attention", JobName);
    await _healthTracker.RecordFailureAsync(JobName, ex.Message, ct);
    // Health tracker will trigger alert when consecutive failures exceed threshold
}
```

## API Endpoint Convention for Worker Jobs

Worker-triggered API endpoints follow a consistent pattern:

```
/api/internal/{feature}/{action}
```

Examples:
- `POST /api/internal/orders/cleanup-expired`
- `POST /api/internal/reports/generate-daily`
- `POST /api/internal/users/send-inactive-reminders`
- `GET /api/internal/settings/{key}`

All internal endpoints are protected by `X-Internal-Token` middleware on the API side. They are NOT accessible to external clients.

## Important Rules

1. **Worker NEVER accesses DB.** All data operations go through API endpoints.
2. **X-Internal-Token is injected automatically.** No job touches auth headers.
3. **Error handling belongs to the job**, not the client. Different jobs have different strategies.
4. **CancellationToken passed to every HTTP call.** If the host shuts down mid-request, the HTTP call is cancelled.
5. **Base URL comes from configuration.** In Docker: `http://api:3000`. In local dev: `http://localhost:3000`. Never hardcoded.
6. **One IApiClient for all jobs.** Do not create separate HTTP clients per job -- the typed client factory handles connection pooling.
