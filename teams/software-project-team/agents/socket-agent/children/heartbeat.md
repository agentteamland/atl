---
knowledge-base-summary: "Is the client truly connected or did the connection silently drop? SignalR has built-in keep-alive but custom heartbeat logic may be needed for \"user went idle after 30s of no signal → mark offline\". Ties into connection tracking."
---
# Heartbeat & Connection Health Monitoring

## Two Layers of "Alive"

There is a critical difference between "connected" and "active":

| State | Meaning | Detected By |
|-------|---------|-------------|
| **Connected** | TCP connection is alive, SignalR can send messages | SignalR built-in keep-alive |
| **Active** | User is actively using the app (screen on, interacting) | Custom heartbeat from client |
| **Idle** | Connected but not interacting (app backgrounded, tab inactive) | No heartbeat for 60 seconds |
| **Away** | Connected but absent for a long time | No heartbeat for 5 minutes |
| **Disconnected** | TCP connection is dead | SignalR `OnDisconnectedAsync` |

SignalR's built-in keep-alive only tells you about the first and last states. The middle three require custom heartbeat logic.

---

## 1. SignalR Built-In Keep-Alive

SignalR sends ping frames automatically. This is **transport-level** -- it keeps the TCP connection alive and detects dead connections.

### Configuration

```csharp
builder.Services.AddSignalR(options =>
{
    // Server sends ping to client every 15 seconds
    options.KeepAliveInterval = TimeSpan.FromSeconds(15);

    // If client doesn't respond within 30 seconds, connection is considered dead
    options.ClientTimeoutInterval = TimeSpan.FromSeconds(30);

    // How long to wait for a hub method to complete
    options.HandshakeTimeout = TimeSpan.FromSeconds(15);
});
```

### What This Handles

- Detecting dead connections (client crashed, network dropped)
- Triggering `OnDisconnectedAsync` when a connection is truly gone
- Preventing proxies/load balancers from closing idle WebSocket connections

### What This Does NOT Handle

- Whether the user is actively using the app
- Whether the app is in the foreground or background
- User availability status (active/idle/away)

---

## 2. Custom Application-Level Heartbeat

### How It Works

1. Client sends `Heartbeat()` hub method call every 30 seconds while the user is active
2. Hub receives the heartbeat and updates the user's last-activity timestamp in Redis
3. A background checker (or lazy check) evaluates: is the user idle or away?
4. When status changes, broadcast to relevant groups

### Heartbeat Hub Method

```csharp
[Authorize]
public sealed class NotificationHub : Hub
{
    private readonly IPresenceStore _presenceStore;
    private readonly ILogger<NotificationHub> _logger;

    public NotificationHub(
        IPresenceStore presenceStore,
        ILogger<NotificationHub> logger)
    {
        _presenceStore = presenceStore;
        _logger = logger;
    }

    public async Task Heartbeat()
    {
        var userId = Context.UserIdentifier!;
        var previousStatus = await _presenceStore.GetStatusAsync(userId);

        await _presenceStore.RecordHeartbeatAsync(userId);

        // If user was idle/away and is now active again, broadcast status change
        if (previousStatus != UserPresenceStatus.Active)
        {
            _logger.LogInformation(
                "User {UserId} became active (was {PreviousStatus})", userId, previousStatus);

            await BroadcastStatusChange(userId, UserPresenceStatus.Active);
        }
    }

    public override async Task OnConnectedAsync()
    {
        var userId = Context.UserIdentifier!;
        await _presenceStore.RecordHeartbeatAsync(userId);
        await _presenceStore.SetStatusAsync(userId, UserPresenceStatus.Active);
        await BroadcastStatusChange(userId, UserPresenceStatus.Active);
        await base.OnConnectedAsync();
    }

    public override async Task OnDisconnectedAsync(Exception? exception)
    {
        var userId = Context.UserIdentifier!;
        await _presenceStore.SetStatusAsync(userId, UserPresenceStatus.Disconnected);
        await BroadcastStatusChange(userId, UserPresenceStatus.Disconnected);
        await base.OnDisconnectedAsync(exception);
    }

    private async Task BroadcastStatusChange(string userId, UserPresenceStatus status)
    {
        // Broadcast to all clients who care about this user's presence
        // (e.g., contacts, team members -- determined by group membership)
        await Clients.All.SendAsync("user-presence-changed", new
        {
            UserId = userId,
            Status = status.ToString().ToLowerInvariant(),
            Timestamp = DateTimeOffset.UtcNow,
        });
    }
}
```

### Presence Status Enum

```csharp
public enum UserPresenceStatus
{
    Active,       // User is actively using the app
    Idle,         // Connected but no heartbeat for 60s
    Away,         // Connected but no heartbeat for 5min
    Disconnected, // TCP connection lost
}
```

### Redis Presence Store

```csharp
public interface IPresenceStore
{
    Task RecordHeartbeatAsync(string userId);
    Task<UserPresenceStatus> GetStatusAsync(string userId);
    Task SetStatusAsync(string userId, UserPresenceStatus status);
    Task<DateTimeOffset?> GetLastHeartbeatAsync(string userId);
}

public sealed class RedisPresenceStore : IPresenceStore
{
    private readonly IConnectionMultiplexer _redis;
    private const string HeartbeatPrefix = "presence:heartbeat:";
    private const string StatusPrefix = "presence:status:";

    public RedisPresenceStore(IConnectionMultiplexer redis)
    {
        _redis = redis;
    }

    public async Task RecordHeartbeatAsync(string userId)
    {
        var db = _redis.GetDatabase();
        var now = DateTimeOffset.UtcNow.ToUnixTimeSeconds();

        // Store last heartbeat timestamp
        await db.StringSetAsync(
            $"{HeartbeatPrefix}{userId}",
            now,
            expiry: TimeSpan.FromMinutes(10)); // Self-clean if user disappears

        // Update status to active
        await db.StringSetAsync(
            $"{StatusPrefix}{userId}",
            UserPresenceStatus.Active.ToString(),
            expiry: TimeSpan.FromMinutes(10));
    }

    public async Task<UserPresenceStatus> GetStatusAsync(string userId)
    {
        var db = _redis.GetDatabase();
        var status = await db.StringGetAsync($"{StatusPrefix}{userId}");

        if (!status.HasValue)
            return UserPresenceStatus.Disconnected;

        return Enum.Parse<UserPresenceStatus>(status!);
    }

    public async Task SetStatusAsync(string userId, UserPresenceStatus status)
    {
        var db = _redis.GetDatabase();
        await db.StringSetAsync(
            $"{StatusPrefix}{userId}",
            status.ToString(),
            expiry: TimeSpan.FromMinutes(10));
    }

    public async Task<DateTimeOffset?> GetLastHeartbeatAsync(string userId)
    {
        var db = _redis.GetDatabase();
        var timestamp = await db.StringGetAsync($"{HeartbeatPrefix}{userId}");

        if (!timestamp.HasValue)
            return null;

        return DateTimeOffset.FromUnixTimeSeconds((long)timestamp);
    }
}
```

---

## 3. Idle/Away Detection: Background Service

A `BackgroundService` periodically scans active users and transitions them to idle or away if their heartbeat has expired.

```csharp
public sealed class PresenceMonitorService : BackgroundService
{
    private readonly IConnectionMultiplexer _redis;
    private readonly IHubContext<NotificationHub> _hubContext;
    private readonly ILogger<PresenceMonitorService> _logger;
    private readonly TimeSpan _checkInterval = TimeSpan.FromSeconds(15);
    private readonly TimeSpan _idleThreshold = TimeSpan.FromSeconds(60);
    private readonly TimeSpan _awayThreshold = TimeSpan.FromMinutes(5);

    public PresenceMonitorService(
        IConnectionMultiplexer redis,
        IHubContext<NotificationHub> hubContext,
        ILogger<PresenceMonitorService> logger)
    {
        _redis = redis;
        _hubContext = hubContext;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation(
            "Presence monitor started. Idle: {IdleThreshold}s, Away: {AwayThreshold}s",
            _idleThreshold.TotalSeconds, _awayThreshold.TotalSeconds);

        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await CheckPresenceAsync();
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Error in presence monitor cycle");
            }

            await Task.Delay(_checkInterval, stoppingToken);
        }
    }

    private async Task CheckPresenceAsync()
    {
        var db = _redis.GetDatabase();
        var server = _redis.GetServer(_redis.GetEndPoints().First());
        var now = DateTimeOffset.UtcNow;

        // Scan all heartbeat keys
        await foreach (var key in server.KeysAsync(pattern: "presence:heartbeat:*"))
        {
            var userId = key.ToString().Replace("presence:heartbeat:", "");
            var timestampValue = await db.StringGetAsync(key);

            if (!timestampValue.HasValue) continue;

            var lastHeartbeat = DateTimeOffset.FromUnixTimeSeconds((long)timestampValue);
            var elapsed = now - lastHeartbeat;

            var currentStatus = await db.StringGetAsync($"presence:status:{userId}");
            if (!currentStatus.HasValue) continue;

            var status = Enum.Parse<UserPresenceStatus>(currentStatus!);
            UserPresenceStatus? newStatus = null;

            if (elapsed >= _awayThreshold && status != UserPresenceStatus.Away)
            {
                newStatus = UserPresenceStatus.Away;
            }
            else if (elapsed >= _idleThreshold && elapsed < _awayThreshold
                     && status == UserPresenceStatus.Active)
            {
                newStatus = UserPresenceStatus.Idle;
            }

            if (newStatus.HasValue)
            {
                await db.StringSetAsync(
                    $"presence:status:{userId}",
                    newStatus.Value.ToString(),
                    expiry: TimeSpan.FromMinutes(10));

                _logger.LogInformation(
                    "User {UserId} transitioned from {OldStatus} to {NewStatus} (last heartbeat {Elapsed}s ago)",
                    userId, status, newStatus.Value, (int)elapsed.TotalSeconds);

                await _hubContext.Clients.All.SendAsync("user-presence-changed", new
                {
                    UserId = userId,
                    Status = newStatus.Value.ToString().ToLowerInvariant(),
                    Timestamp = DateTimeOffset.UtcNow,
                });
            }
        }
    }
}
```

### Registration

```csharp
// Program.cs
builder.Services.AddSingleton<IPresenceStore, RedisPresenceStore>();
builder.Services.AddHostedService<PresenceMonitorService>();
```

---

## 4. Configuration from Dynamic Settings

Timeouts should be configurable without redeployment. Read from environment variables or dynamic settings:

```csharp
public sealed class HeartbeatOptions
{
    // How often the client should send heartbeat (guidance, not enforced)
    public int ClientIntervalSeconds { get; set; } = 30;

    // No heartbeat for this long -> idle
    public int IdleTimeoutSeconds { get; set; } = 60;

    // No heartbeat for this long -> away
    public int AwayTimeoutSeconds { get; set; } = 300;

    // How often the background service checks for idle/away transitions
    public int CheckIntervalSeconds { get; set; } = 15;
}
```

```csharp
// Program.cs
builder.Services.Configure<HeartbeatOptions>(
    builder.Configuration.GetSection("Heartbeat"));
```

```jsonc
// appsettings.json
{
  "Heartbeat": {
    "ClientIntervalSeconds": 30,
    "IdleTimeoutSeconds": 60,
    "AwayTimeoutSeconds": 300,
    "CheckIntervalSeconds": 15
  }
}
```

---

## Client-Side Heartbeat (Flutter/Dart)

```dart
class HeartbeatManager {
  final HubConnection _hub;
  Timer? _heartbeatTimer;
  bool _isActive = true;

  HeartbeatManager(this._hub);

  void start() {
    _heartbeatTimer = Timer.periodic(Duration(seconds: 30), (_) {
      if (_isActive) {
        _hub.invoke('Heartbeat');
      }
    });

    // Listen to app lifecycle
    AppLifecycleListener(
      onResume: () {
        _isActive = true;
        _hub.invoke('Heartbeat'); // Immediate heartbeat on resume
      },
      onPause: () {
        _isActive = false;
      },
    );
  }

  void dispose() {
    _heartbeatTimer?.cancel();
  }
}
```

---

## State Machine

```
            connect
  [none] ──────────► [active]
                        │
                        │ no heartbeat 60s
                        ▼
                      [idle]
                        │
                        │ no heartbeat 5min
                        ▼
                      [away]
                        │
    heartbeat ◄─────────┤ (any state -> active)
                        │
                        │ disconnect
                        ▼
                    [disconnected]
```

Any heartbeat from idle or away transitions the user back to active immediately. Disconnection can happen from any state.

---

## Rules

1. **SignalR keep-alive is transport-level.** Do not disable it. It handles dead connection detection. Custom heartbeat is for application-level activity tracking.
2. **Client sends heartbeat only when active.** If the app is backgrounded, stop sending. This prevents a backgrounded app from appearing "active."
3. **Redis is the source of truth.** Multiple Socket instances may exist behind a load balancer. All read/write presence from Redis.
4. **Self-cleaning keys.** All Redis keys have TTL. If the Socket crashes, keys expire on their own. No orphaned "active" users.
5. **Background service is lightweight.** It scans keys, not connections. Runs every 15 seconds. If it falls behind, the worst case is a slightly delayed idle transition -- no data loss.
6. **Broadcast status changes, not heartbeats.** Clients receive `user-presence-changed` events only when the status actually transitions. They do NOT receive every heartbeat -- that would be too noisy.
