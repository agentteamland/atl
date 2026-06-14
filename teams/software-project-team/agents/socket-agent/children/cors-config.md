---
knowledge-base-summary: "Which origins can establish WebSocket connections? Dev: localhost:5173. Production: real domain. Misconfigured CORS = WebSocket upgrade fails silently. Environment-based configuration."
---
# CORS Configuration

## Why CORS Matters for WebSocket

WebSocket connections begin with an HTTP upgrade request. This initial handshake is a regular HTTP request, so the browser enforces CORS on it. If CORS is not configured or is misconfigured, the upgrade request is rejected and the WebSocket connection never establishes.

The failure mode is particularly insidious: **the WebSocket upgrade fails silently**. The browser console may show a generic connection error, but the root cause (CORS) is not obvious. This is the number one reason new developers struggle to get SignalR working in development.

## The HTTP Upgrade Flow

```
Browser                          Socket Server
   |                                  |
   |-- GET /hubs/notifications ------>|   (1) HTTP upgrade request
   |   Origin: http://localhost:5173  |
   |   Upgrade: websocket            |
   |   Connection: Upgrade           |
   |                                  |
   |<-- 101 Switching Protocols ------|   (2) Only if CORS allows this origin
   |   Access-Control-Allow-Origin   |
   |                                  |
   |<========= WebSocket ==========> |   (3) Persistent connection
```

If the server does not include `Access-Control-Allow-Origin` matching the request's `Origin`, the browser kills the connection at step 2. No error event. No retry. Silent death.

## Environment-Based Configuration

### appsettings.json (Defaults)

```json
{
  "Cors": {
    "AllowedOrigins": "http://localhost:5173,http://localhost:3000"
  }
}
```

### appsettings.Production.json

```json
{
  "Cors": {
    "AllowedOrigins": "https://example-app.com,https://app.example-app.com"
  }
}
```

### Environment Variable Override (Docker)

```yaml
# docker-compose.yml
services:
  socket:
    environment:
      - Cors__AllowedOrigins=https://example-app.com,https://app.example-app.com
```

Environment variables override appsettings values. In Docker, always use environment variables for production configuration.

## Program.cs Implementation

```csharp
var builder = WebApplication.CreateBuilder(args);

// --- CORS Configuration ---
var allowedOrigins = builder.Configuration["Cors:AllowedOrigins"]?
    .Split(',', StringSplitOptions.RemoveEmptyEntries | StringSplitOptions.TrimEntries)
    ?? ["http://localhost:5173", "http://localhost:3000"];

builder.Services.AddCors(options =>
{
    options.AddPolicy("SignalRPolicy", policy =>
    {
        policy
            .WithOrigins(allowedOrigins)
            .AllowAnyHeader()
            .AllowAnyMethod()
            .AllowCredentials(); // Required for SignalR
    });
});

// ... other service registrations (SignalR, JWT, etc.) ...

var app = builder.Build();

// CORS must be before Authentication and Authorization
app.UseCors("SignalRPolicy");
app.UseAuthentication();
app.UseAuthorization();

app.MapHub<NotificationHub>("/hubs/notifications");
app.MapInternalEndpoints();

app.Run();
```

## Critical Details

### AllowCredentials Is Mandatory

SignalR uses cookies and/or authorization headers. Without `AllowCredentials()`, the browser will not send the `access_token` query parameter or cookies on the upgrade request.

```csharp
// This WILL NOT work for SignalR:
policy.AllowAnyOrigin(); // Cannot combine with AllowCredentials

// This WILL work:
policy.WithOrigins("http://localhost:5173").AllowCredentials();
```

`AllowAnyOrigin()` and `AllowCredentials()` are mutually exclusive in the CORS spec. You MUST list specific origins when using credentials.

### Middleware Order Matters

CORS middleware must execute before Authentication and before endpoint mapping. The correct order:

```csharp
app.UseCors("SignalRPolicy");      // 1. CORS first
app.UseAuthentication();            // 2. Auth second
app.UseAuthorization();             // 3. Authz third
app.MapHub<NotificationHub>("/hubs/notifications");  // 4. Endpoints last
```

If CORS comes after Authentication, the preflight OPTIONS request will be rejected before CORS headers are added.

### Development vs Production Logging

Add a startup log to confirm which origins are configured. This saves debugging time:

```csharp
var logger = app.Services.GetRequiredService<ILogger<Program>>();
logger.LogInformation("CORS configured for origins: {Origins}", string.Join(", ", allowedOrigins));
```

In production logs, you will see:
```
CORS configured for origins: https://example-app.com, https://app.example-app.com
```

## Hub-Level CORS (Alternative)

Instead of a global CORS policy, you can apply CORS per hub:

```csharp
app.MapHub<NotificationHub>("/hubs/notifications")
    .RequireCors("SignalRPolicy");
```

This is useful if internal endpoints (like `/api/internal/broadcast`) should NOT have CORS (they are server-to-server only).

```csharp
// Hub gets CORS (browser clients connect here)
app.MapHub<NotificationHub>("/hubs/notifications")
    .RequireCors("SignalRPolicy");

// Internal endpoints have NO CORS (server-to-server only)
app.MapInternalEndpoints();
```

## Common Mistakes

### 1. Forgetting CORS entirely

Symptom: SignalR client gets `Error: Failed to start the connection` with no additional detail.
Fix: Add CORS with the correct origin.

### 2. Using AllowAnyOrigin with credentials

Symptom: Runtime exception: "The CORS protocol does not allow specifying a wildcard origin when the request's credentials mode is 'include'."
Fix: Replace `AllowAnyOrigin()` with `WithOrigins(...)`.

### 3. Wrong port in allowed origins

Symptom: Works on `localhost:3000` but not `localhost:5173`.
Fix: Add ALL client origins, including Vite dev server, Flutter web, etc.

### 4. HTTPS vs HTTP mismatch

Symptom: Works locally (HTTP) but fails in production (HTTPS).
Fix: Production origins must use `https://`.

### 5. Trailing slash in origin

Symptom: CORS rejection despite origin looking correct.
Fix: Origins must NOT have a trailing slash. Use `https://example-app.com`, NOT `https://example-app.com/`.

### 6. CORS middleware after UseAuthentication

Symptom: Preflight OPTIONS returns 401.
Fix: Move `UseCors()` before `UseAuthentication()`.

## Flutter / Mobile Note

Native mobile apps (iOS/Android) do NOT enforce CORS. CORS is a browser-only security mechanism. Flutter mobile clients connect directly without CORS restrictions.

However, Flutter Web DOES enforce CORS (it runs in a browser). If you have a Flutter Web target, include its dev server origin (usually `http://localhost:PORT`) in the allowed origins.

## Security Checklist

- [ ] Never use `AllowAnyOrigin()` in production
- [ ] List every legitimate client origin explicitly
- [ ] Use HTTPS origins in production
- [ ] No trailing slashes in origin URLs
- [ ] `AllowCredentials()` is present
- [ ] CORS middleware is before Authentication middleware
- [ ] Startup log confirms active origins
- [ ] Internal endpoints do NOT have CORS applied
