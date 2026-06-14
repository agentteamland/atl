---
knowledge-base-summary: "Socket → API communication via typed HttpClient. IApiClient interface, ApiClient implementation. InternalTokenHandler (DelegatingHandler) auto-injects X-Internal-Token on every request. BaseUrl configured from environment."
---
# API Client Pattern: Socket-to-API HTTP Calls

## Why This Exists

The Socket is a thin host with no business logic, no database access, no Application/Infrastructure references. When a client calls a hub method that requires data or triggers an action, the hub forwards the request to the API via HTTP. This is the only way the Socket communicates with the backend.

```
Flutter Client              Socket Hub                  API
    |                          |                          |
    |  Hub method call         |                          |
    |  (SendMessage)           |                          |
    | -----------------------> |                          |
    |                          |  POST /api/messages      |
    |                          |  X-Internal-Token: xxx   |
    |                          | -----------------------> |
    |                          |                          |  (business logic,
    |                          |                          |   DB, validation)
    |                          |  200 OK + MessageDto     |
    |                          | <----------------------- |
    |  MessageReceived event   |                          |
    | <----------------------- |                          |
```

## IApiClient Interface

The interface defines generic HTTP methods. Hub methods call these to forward requests to the API.

```csharp
namespace ExampleApp.Socket.Services;

public interface IApiClient
{
    Task<HttpResponseMessage> GetAsync(
        string path,
        CancellationToken ct = default);

    Task<HttpResponseMessage> PostAsync<T>(
        string path,
        T body,
        CancellationToken ct = default);

    Task<HttpResponseMessage> PutAsync<T>(
        string path,
        T body,
        CancellationToken ct = default);

    Task<HttpResponseMessage> DeleteAsync(
        string path,
        CancellationToken ct = default);
}
```

## ApiClient Implementation

The implementation is intentionally thin. No retry logic, no caching, no error handling beyond what HttpClient provides. The API handles all business concerns.

```csharp
using System.Net.Http.Json;

namespace ExampleApp.Socket.Services;

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

## ApiClientOptions

Configuration for the API's base URL and the internal token for system-to-system calls.

```csharp
namespace ExampleApp.Socket.Services;

public sealed class ApiClientOptions
{
    public string BaseUrl { get; set; } = "http://api:3000";
    public string InternalToken { get; set; } = string.Empty;
}
```

## InternalTokenHandler (DelegatingHandler)

This handler automatically injects the `X-Internal-Token` header on every outgoing HTTP request from the Socket to the API. Hub methods never need to think about authentication — the handler does it transparently.

```csharp
namespace ExampleApp.Socket.Services;

public sealed class InternalTokenHandler : DelegatingHandler
{
    private readonly ApiClientOptions _options;

    public InternalTokenHandler(ApiClientOptions options)
    {
        _options = options;
    }

    protected override async Task<HttpResponseMessage> SendAsync(
        HttpRequestMessage request,
        CancellationToken cancellationToken)
    {
        request.Headers.Add("X-Internal-Token", _options.InternalToken);
        return await base.SendAsync(request, cancellationToken);
    }
}
```

## Registration in Program.cs

There are two approaches to register the HttpClient with the internal token header. Choose one.

### Approach 1: DelegatingHandler (Recommended)

Uses the `InternalTokenHandler` to inject the header automatically via the HttpClient pipeline.

```csharp
// ApiClientOptions
var apiClientOptions = new ApiClientOptions
{
    BaseUrl = builder.Configuration["ApiClient:BaseUrl"] ?? "http://api:3000",
    InternalToken = builder.Configuration["InternalToken"]
        ?? "internal-secret-change-in-production",
};

builder.Services.AddSingleton(apiClientOptions);

// Register the DelegatingHandler
builder.Services.AddTransient<InternalTokenHandler>();

// Typed HttpClient with handler in the pipeline
builder.Services.AddHttpClient<IApiClient, ApiClient>(client =>
{
    client.BaseAddress = new Uri(apiClientOptions.BaseUrl);
    client.Timeout = TimeSpan.FromSeconds(10);
})
.AddHttpMessageHandler<InternalTokenHandler>();
```

### Approach 2: Direct Header (Simpler)

Sets the header directly on the HttpClient's default headers. Simpler, fewer files, but less flexible.

```csharp
var apiClientOptions = new ApiClientOptions
{
    BaseUrl = builder.Configuration["ApiClient:BaseUrl"] ?? "http://api:3000",
    InternalToken = builder.Configuration["InternalToken"]
        ?? "internal-secret-change-in-production",
};

builder.Services.AddSingleton(apiClientOptions);

builder.Services.AddHttpClient<IApiClient, ApiClient>(client =>
{
    client.BaseAddress = new Uri(apiClientOptions.BaseUrl);
    client.Timeout = TimeSpan.FromSeconds(10);
    client.DefaultRequestHeaders.Add("X-Internal-Token", apiClientOptions.InternalToken);
});
```

The current codebase uses Approach 2 for simplicity. Switch to Approach 1 when you need more control (e.g., adding logging, retry, or correlation headers to the pipeline).

## Docker Compose Configuration

```yaml
services:
  socket:
    environment:
      - ApiClient__BaseUrl=http://api:3000
      - InternalToken=${INTERNAL_TOKEN}
```

The `ApiClient__BaseUrl` uses the Docker service name `api` — within the Docker network, services resolve each other by name.

## Usage in Hub Methods

```csharp
[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly IApiClient _apiClient;
    private readonly ILogger<NotificationHub> _logger;

    public NotificationHub(IApiClient apiClient, ILogger<NotificationHub> logger)
    {
        _apiClient = apiClient;
        _logger = logger;
    }

    public async Task MarkNotificationRead(Guid notificationId)
    {
        var userId = Context.UserIdentifier!;
        _logger.LogInformation(
            "MarkNotificationRead: user {UserId}, notification {NotificationId}",
            userId, notificationId);

        var response = await _apiClient.PutAsync(
            $"/api/notifications/{notificationId}/read",
            new { UserId = userId });

        if (!response.IsSuccessStatusCode)
        {
            _logger.LogWarning(
                "MarkNotificationRead failed: {StatusCode}",
                (int)response.StatusCode);
            throw new HubException("Failed to mark notification as read");
        }

        // Confirm back to the caller
        await Clients.Caller.SendAsync("NotificationMarked", new
        {
            NotificationId = notificationId,
            MarkedAt = DateTime.UtcNow,
        });
    }
}
```

## Reading API Responses

Use `ReadFromJsonAsync<T>` to deserialize API responses into lightweight DTOs defined in the Socket project.

```csharp
var response = await _apiClient.GetAsync($"/api/users/{userId}/profile");

if (response.IsSuccessStatusCode)
{
    var profile = await response.Content.ReadFromJsonAsync<UserProfileDto>();
    await Clients.Caller.SendAsync("ProfileLoaded", profile);
}
else
{
    throw new HubException("Failed to load profile");
}
```

Socket DTOs are simple records — they mirror API response shapes but are owned by the Socket project:

```csharp
namespace ExampleApp.Socket.Models;

public sealed record UserProfileDto(
    Guid Id,
    string DisplayName,
    string? AvatarUrl);
```

## What NOT to Do

| Wrong | Why | Correct |
|-------|-----|---------|
| Inject `DbContext` into hub | Socket has no Infrastructure reference | Use `_apiClient.GetAsync(...)` |
| Add retry logic to `ApiClient` | Business resilience is API's concern | Let API return proper error codes |
| Cache API responses in Socket | Socket is stateless, cache belongs in API/Redis | Call API every time |
| Pass user's JWT to API | API trusts internal calls via X-Internal-Token | Token is injected by handler |
| Construct complex request bodies | Hub methods should be flat | Keep it simple, forward as-is |
