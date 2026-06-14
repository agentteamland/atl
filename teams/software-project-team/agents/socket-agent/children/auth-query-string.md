---
knowledge-base-summary: "Browser WebSocket limitation — cannot send headers during upgrade. Token extracted from `?access_token={token}` via JwtBearerEvents.OnMessageReceived. Full validation same as API. Shared JWT secret between API and Socket."
---
# Auth: JWT from WebSocket Query String

## The Problem

Browsers **cannot** send custom HTTP headers during the WebSocket upgrade handshake. The `Authorization: Bearer {token}` header that works for REST API calls is not available when establishing a SignalR WebSocket connection.

This is a browser limitation, not a SignalR limitation. The WebSocket API in browsers (`new WebSocket(url)`) provides no mechanism to attach headers. SignalR's JavaScript client uses this API under the hood.

## The Solution

Send the JWT token as a **query string parameter** during the WebSocket connection:

```
wss://socket-host:3002/hubs/notifications?access_token=eyJhbGciOiJIUzI1NiIs...
```

Then, on the server side, extract this token from the query string and feed it to the ASP.NET Core JWT authentication pipeline via `JwtBearerEvents.OnMessageReceived`.

## How It Works

```
Client                              Server
  |                                    |
  |  GET /hubs/notifications           |
  |  ?access_token=eyJ...              |
  |  Upgrade: websocket                |
  | ---------------------------------> |
  |                                    |  1. OnMessageReceived fires
  |                                    |  2. Extracts token from query string
  |                                    |  3. Sets context.Token = token
  |                                    |  4. Normal JWT validation runs
  |                                    |  5. ClaimsPrincipal set on context
  |  101 Switching Protocols           |
  | <--------------------------------- |
  |                                    |
  |  WebSocket connection established  |
  |  (authenticated, claims available) |
```

## Program.cs Configuration

This is the full JWT configuration for the Socket project. The critical part is the `OnMessageReceived` event handler.

```csharp
using System.Text;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.IdentityModel.Tokens;

// JWT secret MUST match the API's JWT secret — same tokens, same validation
var jwtSecret = builder.Configuration["Jwt:Secret"]
    ?? "super-secret-key-that-should-be-at-least-32-chars-long!";
var jwtIssuer = builder.Configuration["Jwt:Issuer"] ?? "ExampleApp";
var jwtAudience = builder.Configuration["Jwt:Audience"] ?? "ExampleApp";

builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        // Validation parameters — identical to API
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidateAudience = true,
            ValidateLifetime = true,
            ValidateIssuerSigningKey = true,
            ValidIssuer = jwtIssuer,
            ValidAudience = jwtAudience,
            IssuerSigningKey = new SymmetricSecurityKey(
                Encoding.UTF8.GetBytes(jwtSecret)),
            ClockSkew = TimeSpan.Zero, // No grace period — token expires exactly on time
        };

        // THIS IS THE KEY PART — extract token from query string for SignalR
        options.Events = new JwtBearerEvents
        {
            OnMessageReceived = context =>
            {
                var accessToken = context.Request.Query["access_token"];
                var path = context.HttpContext.Request.Path;

                // Only extract from query string for hub endpoints
                if (!string.IsNullOrEmpty(accessToken)
                    && path.StartsWithSegments("/hubs"))
                {
                    context.Token = accessToken;
                }

                return Task.CompletedTask;
            },
        };
    });

builder.Services.AddAuthorization();
```

## Why the Path Check Matters

```csharp
path.StartsWithSegments("/hubs")
```

Without this check, the query string extraction would run for **every** request — including the internal broadcast endpoint (`/api/internal/broadcast`). That endpoint uses `X-Internal-Token`, not JWT. The path check ensures query string extraction only activates for hub routes.

## Shared JWT Secret

The API and Socket MUST share the same JWT configuration:

| Setting | API | Socket | Must Match? |
|---------|-----|--------|-------------|
| `Jwt:Secret` | Signs tokens | Validates tokens | **YES** |
| `Jwt:Issuer` | Sets `iss` claim | Validates `iss` claim | **YES** |
| `Jwt:Audience` | Sets `aud` claim | Validates `aud` claim | **YES** |
| `ClockSkew` | - | Validation tolerance | Should match |

In Docker Compose, both services read from the same environment variable:

```yaml
services:
  api:
    environment:
      - Jwt__Secret=${JWT_SECRET}
      - Jwt__Issuer=ExampleApp
      - Jwt__Audience=ExampleApp
  
  socket:
    environment:
      - Jwt__Secret=${JWT_SECRET}
      - Jwt__Issuer=ExampleApp
      - Jwt__Audience=ExampleApp
```

## Accessing User Identity in Hub

After JWT validation, the user's claims are available via `Context.User`:

```csharp
[Authorize]
public sealed class NotificationHub : Hub
{
    // UserIdentifier is auto-populated from ClaimTypes.NameIdentifier
    // This is what SignalR uses for Clients.User(userId)
    public override async Task OnConnectedAsync()
    {
        var userId = Context.UserIdentifier;              // From NameIdentifier claim
        var email = Context.User?.FindFirstValue("email"); // Custom claim
        var tenantId = Context.User?.FindFirstValue("tenantId"); // Custom claim

        _logger.LogInformation(
            "User {UserId} connected, tenant {TenantId}",
            userId, tenantId);

        await base.OnConnectedAsync();
    }
}
```

## UserIdentifier Mapping

SignalR uses `IUserIdProvider` to determine the user's identity. By default, it maps `ClaimTypes.NameIdentifier`. If the API sets a different claim for the user ID, override the provider:

```csharp
// Only needed if your JWT uses a non-standard claim for user ID
public sealed class CustomUserIdProvider : IUserIdProvider
{
    public string? GetUserId(HubConnectionContext connection)
    {
        return connection.User?.FindFirstValue("sub")  // or "userId", etc.
            ?? connection.User?.FindFirstValue(ClaimTypes.NameIdentifier);
    }
}

// Register in Program.cs
builder.Services.AddSingleton<IUserIdProvider, CustomUserIdProvider>();
```

## Client-Side Connection

### Flutter (signalr_netcore package)

```dart
final connection = HubConnectionBuilder()
    .withUrl(
      'http://socket-host:3002/hubs/notifications',
      options: HttpConnectionOptions(
        accessTokenFactory: () async => await getAccessToken(),
      ),
    )
    .withAutomaticReconnect()
    .build();

await connection.start();
```

The `accessTokenFactory` appends the token as `?access_token={token}` automatically.

### JavaScript

```javascript
const connection = new signalR.HubConnectionBuilder()
    .withUrl("/hubs/notifications", {
        accessTokenFactory: () => localStorage.getItem("access_token")
    })
    .withAutomaticReconnect()
    .build();

await connection.start();
```

## Security Considerations

1. **HTTPS is mandatory in production.** The `?access_token=` in the URL is visible in server logs, proxy logs, and browser history. HTTPS encrypts the URL in transit but the token still appears in server access logs. Consider log scrubbing.

2. **Short-lived tokens.** Access tokens should be short-lived (15 minutes). The WebSocket connection stays authenticated after the initial handshake, but if the client reconnects after token expiry, it must obtain a fresh token.

3. **Token is validated once** — at connection time. A token that expires during an active WebSocket session does NOT cause disconnection. The connection remains authenticated until it drops. On reconnect, the client must supply a fresh token.

4. **Never log the token value.** Log userId and connectionId, never the raw JWT.
