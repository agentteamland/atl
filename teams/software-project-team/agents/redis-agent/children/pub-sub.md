---
knowledge-base-summary: "Redis pub/sub for real-time inter-service events. Cache invalidation across pods, real-time notifications, event broadcasting. Fire-and-forget semantics — no persistence, no replay. When to use Redis pub/sub vs RabbitMQ: pub/sub for ephemeral signals, RMQ for durable messages that must not be lost."
---
# Redis Pub/Sub

Redis pub/sub provides real-time messaging between services. Messages are fire-and-forget: if no subscriber is listening, the message is lost. This makes it suitable for ephemeral signals, not durable messaging.

## When to Use Redis Pub/Sub vs RabbitMQ

| Criteria | Redis Pub/Sub | RabbitMQ |
|----------|--------------|----------|
| **Durability** | No — lost if no subscriber | Yes — queued until consumed |
| **Delivery guarantee** | None (fire-and-forget) | At-least-once (with ack) |
| **Use case** | Ephemeral signals, cache invalidation | Business events, emails, logs |
| **Persistence** | No | Yes (disk-backed queues) |
| **Pattern** | Broadcast to all subscribers | Fan-out, routing, work queues |
| **Performance** | Very fast (~microseconds) | Fast (~milliseconds) |
| **Retry on failure** | No | Yes (DLX, requeue) |

### Decision Rule
- **"If this message is lost, does anything break?"**
  - **No** -> Redis pub/sub (cache invalidation, typing indicators, online status)
  - **Yes** -> RabbitMQ (emails, audit logs, order processing)

## Use Cases

### 1. Cache Invalidation Across Pods

When running multiple API instances behind a load balancer, one instance invalidates a cache key but other instances still have stale data. Redis pub/sub broadcasts the invalidation signal.

```csharp
// Infrastructure/Redis/CacheInvalidationPublisher.cs
public class CacheInvalidationPublisher : ICacheInvalidationPublisher
{
    private readonly ISubscriber _subscriber;
    private const string Channel = "cache:invalidation";

    public CacheInvalidationPublisher(IConnectionMultiplexer redis)
    {
        _subscriber = redis.GetSubscriber();
    }

    public async Task PublishInvalidationAsync(string cacheKey)
    {
        var message = JsonSerializer.Serialize(new CacheInvalidationMessage
        {
            Key = cacheKey,
            Timestamp = DateTimeOffset.UtcNow,
            SourceInstanceId = Environment.MachineName
        });

        await _subscriber.PublishAsync(RedisChannel.Literal(Channel), message);
    }

    public async Task PublishInvalidationAsync(IEnumerable<string> cacheKeys)
    {
        foreach (var key in cacheKeys)
        {
            await PublishInvalidationAsync(key);
        }
    }
}

public record CacheInvalidationMessage
{
    public string Key { get; init; } = string.Empty;
    public DateTimeOffset Timestamp { get; init; }
    public string SourceInstanceId { get; init; } = string.Empty;
}
```

### Subscriber (runs on every instance at startup)

```csharp
// Infrastructure/Redis/CacheInvalidationSubscriber.cs
public class CacheInvalidationSubscriber : IHostedService
{
    private readonly ISubscriber _subscriber;
    private readonly IDatabase _redis;
    private readonly ILogger<CacheInvalidationSubscriber> _logger;
    private const string Channel = "cache:invalidation";

    public CacheInvalidationSubscriber(
        IConnectionMultiplexer multiplexer,
        ILogger<CacheInvalidationSubscriber> logger)
    {
        _subscriber = multiplexer.GetSubscriber();
        _redis = multiplexer.GetDatabase();
        _logger = logger;
    }

    public async Task StartAsync(CancellationToken ct)
    {
        await _subscriber.SubscribeAsync(RedisChannel.Literal(Channel), async (channel, message) =>
        {
            try
            {
                var invalidation = JsonSerializer.Deserialize<CacheInvalidationMessage>(message!);
                if (invalidation is null) return;

                // Skip if we published this message (we already invalidated locally)
                if (invalidation.SourceInstanceId == Environment.MachineName)
                    return;

                if (invalidation.Key.Contains('*'))
                {
                    // Wildcard pattern — use SCAN
                    var server = _redis.Multiplexer.GetServer(
                        _redis.Multiplexer.GetEndPoints().First());
                    await foreach (var key in server.KeysAsync(pattern: invalidation.Key))
                    {
                        await _redis.KeyDeleteAsync(key);
                    }
                }
                else
                {
                    await _redis.KeyDeleteAsync(invalidation.Key);
                }

                _logger.LogDebug(
                    "Cache invalidated {Key} from instance {Source}",
                    invalidation.Key,
                    invalidation.SourceInstanceId);
            }
            catch (Exception ex)
            {
                _logger.LogWarning(ex, "Failed to process cache invalidation message");
            }
        });

        _logger.LogInformation("Cache invalidation subscriber started on channel {Channel}", Channel);
    }

    public async Task StopAsync(CancellationToken ct)
    {
        await _subscriber.UnsubscribeAsync(RedisChannel.Literal(Channel));
        _logger.LogInformation("Cache invalidation subscriber stopped");
    }
}
```

### 2. Real-Time Notifications (Typing Indicator)

```csharp
// Publish when user starts/stops typing
public class TypingNotificationService
{
    private readonly ISubscriber _subscriber;

    public TypingNotificationService(IConnectionMultiplexer redis)
    {
        _subscriber = redis.GetSubscriber();
    }

    public async Task PublishTypingAsync(Guid channelId, Guid userId, bool isTyping)
    {
        var message = JsonSerializer.Serialize(new
        {
            ChannelId = channelId,
            UserId = userId,
            IsTyping = isTyping,
            Timestamp = DateTimeOffset.UtcNow
        });

        await _subscriber.PublishAsync(
            RedisChannel.Literal($"typing:{channelId}"),
            message
        );
    }
}
```

### 3. Online Status Broadcasting

```csharp
// Broadcast user online/offline status changes
public async Task PublishStatusChangeAsync(Guid userId, bool isOnline)
{
    var message = JsonSerializer.Serialize(new
    {
        UserId = userId,
        IsOnline = isOnline,
        Timestamp = DateTimeOffset.UtcNow
    });

    await _subscriber.PublishAsync(
        RedisChannel.Literal("user:status"),
        message
    );
}
```

## Channel Naming Convention

Follow the same pattern as key naming:

```
cache:invalidation          → cache invalidation signals
typing:{channelId}          → typing indicators per channel
user:status                 → online/offline status changes
settings:changed            → dynamic setting changed notification
deployment:notify           → deployment/restart notifications
```

## DI Registration

```csharp
// Program.cs
builder.Services.AddSingleton<ICacheInvalidationPublisher, CacheInvalidationPublisher>();
builder.Services.AddHostedService<CacheInvalidationSubscriber>();
```

## Limitations and Gotchas

### 1. No Message Persistence
If a subscriber is down when a message is published, that message is lost forever. No replay, no catch-up.

### 2. No Acknowledgment
Publisher does not know if anyone received the message. No acks, no nacks.

### 3. All Subscribers Receive All Messages
Unlike RabbitMQ work queues, every subscriber on a channel receives every message. There is no load balancing across subscribers.

### 4. Connection-Dependent
Subscriptions are tied to the Redis connection. If the connection drops, subscriptions are lost. StackExchange.Redis handles reconnection automatically, but messages during the disconnection window are lost.

### 5. Pattern Subscriptions Are Expensive
```csharp
// Avoid pattern subscriptions in high-traffic scenarios
// BAD — Redis evaluates this pattern for every published message
await _subscriber.SubscribeAsync(RedisChannel.Pattern("cache:*"), handler);

// GOOD — subscribe to specific channels
await _subscriber.SubscribeAsync(RedisChannel.Literal("cache:invalidation"), handler);
```

## Testing Pub/Sub

```csharp
// Integration test — verify message delivery
[Fact]
public async Task CacheInvalidation_PublishAndSubscribe_ShouldDeliverMessage()
{
    var received = new TaskCompletionSource<string>();

    await _subscriber.SubscribeAsync(
        RedisChannel.Literal("cache:invalidation"),
        (channel, message) => received.SetResult(message!)
    );

    await _publisher.PublishInvalidationAsync("cache:product:123");

    var result = await received.Task.WaitAsync(TimeSpan.FromSeconds(5));
    var msg = JsonSerializer.Deserialize<CacheInvalidationMessage>(result);
    Assert.Equal("cache:product:123", msg!.Key);
}
```
