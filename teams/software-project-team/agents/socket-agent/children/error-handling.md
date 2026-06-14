---
knowledge-base-summary: "Hub method throws exception → what does the client see? Structured error format for SignalR — the Socket equivalent of API's ProblemDetails. Consistent, parseable, client-friendly error responses."
---
# Error Handling on Hub Methods

## The Problem

When a hub method throws an unhandled exception, the client receives an opaque, unhelpful error. In development mode SignalR leaks the full exception message; in production it says "An unexpected error occurred." Neither is useful for the client to react programmatically.

The API has a global exception handler that maps exceptions to ProblemDetails. The Socket needs the same thing -- a global error wrapper that produces structured, parseable error responses for every hub method.

---

## Structured Error Model

Every error sent to the client follows this contract:

```csharp
public sealed record HubError
{
    /// <summary>
    /// Machine-readable error code. Kebab-case.
    /// Examples: "not-found", "validation-error", "unauthorized", "rate-limited"
    /// </summary>
    public required string Code { get; init; }

    /// <summary>
    /// Human-readable error message. For display or debugging.
    /// </summary>
    public required string Message { get; init; }

    /// <summary>
    /// Optional structured details. Validation errors, field-level errors, etc.
    /// </summary>
    public object? Details { get; init; }
}
```

### Error Code Convention

Error codes are **kebab-case** strings. They are stable identifiers that the client can switch on:

| Code | HTTP Equivalent | When |
|------|----------------|------|
| `not-found` | 404 | Entity not found (forwarded from API) |
| `validation-error` | 422 | Invalid input from client |
| `unauthorized` | 401 | Token expired or invalid |
| `forbidden` | 403 | Authenticated but no access |
| `rate-limited` | 429 | Too many requests from this connection |
| `conflict` | 409 | Concurrent modification conflict |
| `internal-error` | 500 | Unexpected server error |
| `api-unavailable` | 503 | API returned error or is unreachable |

---

## Hub Filter: Global Error Wrapper

SignalR provides `IHubFilter` -- the equivalent of middleware/exception handlers for hub methods. Implement a single filter that catches all exceptions and wraps them in a `HubException` with the structured payload.

### Implementation

```csharp
public sealed class HubExceptionFilter : IHubFilter
{
    private readonly ILogger<HubExceptionFilter> _logger;
    private readonly IHostEnvironment _environment;

    public HubExceptionFilter(
        ILogger<HubExceptionFilter> logger,
        IHostEnvironment environment)
    {
        _logger = logger;
        _environment = environment;
    }

    public async ValueTask<object?> InvokeMethodAsync(
        HubInvocationContext invocationContext,
        Func<HubInvocationContext, ValueTask<object?>> next)
    {
        try
        {
            return await next(invocationContext);
        }
        catch (HubException)
        {
            // Already a HubException with structured payload -- rethrow as-is
            throw;
        }
        catch (Exception ex)
        {
            var hubError = MapExceptionToHubError(ex);

            _logger.LogError(ex,
                "Hub method {MethodName} failed for user {UserId}. Code: {ErrorCode}, Message: {ErrorMessage}",
                invocationContext.HubMethodName,
                invocationContext.Context.UserIdentifier,
                hubError.Code,
                hubError.Message);

            // HubException is the only exception type that SignalR forwards to the client.
            // The message becomes the error message on the client side.
            throw new HubException(
                System.Text.Json.JsonSerializer.Serialize(hubError));
        }
    }

    private HubError MapExceptionToHubError(Exception ex)
    {
        return ex switch
        {
            // Map known exception types to structured errors
            NotFoundException notFound => new HubError
            {
                Code = "not-found",
                Message = notFound.Message,
            },

            ValidationException validation => new HubError
            {
                Code = "validation-error",
                Message = "One or more validation errors occurred.",
                Details = validation.Errors, // Dictionary<string, string[]>
            },

            ForbiddenException forbidden => new HubError
            {
                Code = "forbidden",
                Message = forbidden.Message,
            },

            UnauthorizedAccessException => new HubError
            {
                Code = "unauthorized",
                Message = "Authentication required.",
            },

            HttpRequestException httpEx => new HubError
            {
                Code = "api-unavailable",
                Message = "The API is temporarily unavailable. Please try again.",
                Details = _environment.IsDevelopment()
                    ? new { InnerMessage = httpEx.Message }
                    : null,
            },

            // Unknown exception -- generic error, no details leaked in production
            _ => new HubError
            {
                Code = "internal-error",
                Message = _environment.IsDevelopment()
                    ? ex.Message
                    : "An unexpected error occurred.",
                Details = _environment.IsDevelopment()
                    ? new { ExceptionType = ex.GetType().Name, ex.StackTrace }
                    : null,
            },
        };
    }
}
```

### Exception Types

Since the Socket project does NOT reference the Application layer, define lightweight exception types in the Socket project itself:

```csharp
// Socket/Exceptions/NotFoundException.cs
public sealed class NotFoundException : Exception
{
    public NotFoundException(string message) : base(message) { }
    public NotFoundException(string entity, object key)
        : base($"{entity} with key '{key}' was not found.") { }
}

// Socket/Exceptions/ValidationException.cs
public sealed class ValidationException : Exception
{
    public Dictionary<string, string[]> Errors { get; }

    public ValidationException(Dictionary<string, string[]> errors)
        : base("One or more validation errors occurred.")
    {
        Errors = errors;
    }

    public ValidationException(string field, string error)
        : this(new Dictionary<string, string[]> { [field] = [error] }) { }
}

// Socket/Exceptions/ForbiddenException.cs
public sealed class ForbiddenException : Exception
{
    public ForbiddenException(string message = "Access denied.") : base(message) { }
}
```

### When API Returns an Error

Hub methods call the API via `IApiClient`. When the API returns a non-success status, the hub method should translate it:

```csharp
public async Task<object> GetOrderStatus(string orderId)
{
    var response = await _apiClient.GetAsync($"/api/orders/{orderId}", default);

    if (response.StatusCode == System.Net.HttpStatusCode.NotFound)
        throw new NotFoundException("Order", orderId);

    if (response.StatusCode == System.Net.HttpStatusCode.Forbidden)
        throw new ForbiddenException("You do not have access to this order.");

    if (!response.IsSuccessStatusCode)
        throw new HttpRequestException(
            $"API returned {(int)response.StatusCode} for GET /api/orders/{orderId}");

    return await response.Content.ReadFromJsonAsync<object>()
        ?? throw new InvalidOperationException("API returned null body.");
}
```

---

## Registration

```csharp
// Program.cs
builder.Services.AddSignalR(options =>
{
    options.AddFilter<HubExceptionFilter>();
});

// Register the filter for DI
builder.Services.AddSingleton<HubExceptionFilter>();
```

---

## Client-Side Handling

### Option 1: Catch HubException (Invoke Pattern)

When the client uses `.invoke()` (request-response), the error is thrown as an exception:

```dart
try {
  final result = await hub.invoke('GetOrderStatus', args: ['order-123']);
} on Exception catch (e) {
  // Parse the structured error from the exception message
  final errorJson = jsonDecode(e.toString());
  final code = errorJson['code']; // "not-found"
  final message = errorJson['message']; // "Order with key 'order-123' was not found."

  switch (code) {
    case 'not-found':
      showNotFoundDialog(message);
      break;
    case 'validation-error':
      showValidationErrors(errorJson['details']);
      break;
    case 'unauthorized':
      redirectToLogin();
      break;
    default:
      showGenericError(message);
  }
}
```

### Option 2: Listen for Error Events (Fire-and-Forget Pattern)

For hub methods that don't return a value, the client can listen for a dedicated error event:

```csharp
// Hub method sends error as an event instead of throwing
public async Task SendMessage(string conversationId, string content)
{
    try
    {
        await _apiClient.PostAsync("/api/messages", new { conversationId, content }, default);
    }
    catch (Exception ex)
    {
        var hubError = MapExceptionToHubError(ex);
        // Send error back to the caller only
        await Clients.Caller.SendAsync("error", hubError);
    }
}
```

```dart
// Client listens for error events
hub.on('error', (error) {
  final code = error['code'];
  final message = error['message'];
  handleError(code, message);
});
```

---

## Rules

1. **Every hub method is wrapped by the filter.** No hub method should have its own try-catch for error formatting. The filter handles it globally, just like the API's global exception handler.
2. **HubException is the only exception SignalR forwards to clients.** All other exception types are swallowed by SignalR. The filter must always convert to `HubException`.
3. **Structured JSON payload in HubException.Message.** The error code, message, and details are serialized as JSON in the HubException message string. The client deserializes it.
4. **No sensitive data in production errors.** Stack traces, inner exception messages, and detailed error info are only included when `IsDevelopment()` is true.
5. **Error codes are stable contracts.** Once published, an error code like `not-found` must not change meaning. New error conditions get new codes.
6. **Log every error.** The filter logs every exception before converting it. This is the Socket's equivalent of the API's exception handler logging.
