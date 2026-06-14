---
knowledge-base-summary: "Why idempotency matters (network partitions, consumer crashes, redelivery). Redis SETNX pattern for deduplication. Message ID as the idempotency key. TTL for idempotency keys (24h). Check-before-process, ack-after-process pattern. Code examples."
---
# Idempotency

Processing the same message twice must produce the same result. This is the most critical reliability guarantee in message-based systems.

## Why Idempotency Matters

Messages WILL be delivered more than once. This is not a bug — it is a fundamental property of distributed messaging.

**Scenarios that cause redelivery:**
1. **Consumer crashes** after processing but before ack — RabbitMQ redelivers
2. **Network partition** between consumer and RabbitMQ — connection drops, messages are requeued
3. **Channel timeout** — consumer takes too long, RabbitMQ assumes it is dead
4. **Manual requeue** from dead letter queue during incident recovery
5. **RabbitMQ failover** in clustered setups — messages may be redelivered during leader election

**Without idempotency:**
- Email sent twice to the customer
- Payment charged twice
- Duplicate records in the database
- Counters incremented twice

## Redis SETNX Pattern

The standard approach: use Redis `SETNX` (Set if Not eXists) with the message ID as key. If the key already exists, the message was already processed — skip it.

### IIdempotencyStore Interface

```csharp
public interface IIdempotencyStore
{
    /// <summary>
    /// Try to acquire an idempotency lock for the given message ID.
    /// Returns true if this is the first time (proceed with processing).
    /// Returns false if already processed (skip).
    /// </summary>
    Task<bool> TryAcquireAsync(string messageId);

    /// <summary>
    /// Release the idempotency lock (used when processing fails and retry is needed).
    /// </summary>
    Task ReleaseAsync(string messageId);
}
```

### Redis Implementation

```csharp
using StackExchange.Redis;

public sealed class RedisIdempotencyStore : IIdempotencyStore
{
    private readonly IConnectionMultiplexer _redis;
    private readonly string _prefix;
    private readonly TimeSpan _ttl;

    public RedisIdempotencyStore(string connectionString, string prefix, TimeSpan ttl)
    {
        _redis = ConnectionMultiplexer.Connect(connectionString);
        _prefix = prefix;
        _ttl = ttl;
    }

    public async Task<bool> TryAcquireAsync(string messageId)
    {
        var db = _redis.GetDatabase();
        var key = $"{_prefix}:{messageId}";

        // SETNX with TTL — atomic operation
        // Returns true if key was set (first time), false if it already existed
        return await db.StringSetAsync(key, "1", _ttl, When.NotExists);
    }

    public async Task ReleaseAsync(string messageId)
    {
        var db = _redis.GetDatabase();
        var key = $"{_prefix}:{messageId}";

        await db.KeyDeleteAsync(key);
    }
}
```

### Registration

```csharp
// In Program.cs
builder.Services.AddSingleton<IIdempotencyStore>(sp =>
    new RedisIdempotencyStore(
        connectionString: builder.Configuration.GetConnectionString("Redis")!,
        prefix: "email-idem",        // Different prefix per consumer type
        ttl: TimeSpan.FromHours(24)   // 24h TTL — after that, key expires
    ));
```

## Message ID as Idempotency Key

The message ID is set by the producer and must be unique per logical operation.

### Producer Side

```csharp
var props = new BasicProperties
{
    MessageId = Guid.NewGuid().ToString(),  // Unique per message
    // ... other properties
};

await channel.BasicPublishAsync(exchange, routingKey, mandatory: false, props, body);
```

### Consumer Side

```csharp
// Use MessageId from properties, fall back to DeliveryTag
var messageId = ea.BasicProperties?.MessageId ?? ea.DeliveryTag.ToString();
```

**Important:** If the producer uses a business-level idempotency key (e.g., order ID + operation), use that instead of a random GUID. This prevents duplicate processing even if the producer publishes the same message twice.

```csharp
// Better: business-level idempotency key
var props = new BasicProperties
{
    MessageId = $"send-email:{order.Id}:{emailType}",
    // ...
};
```

## TTL for Idempotency Keys

The Redis key TTL determines how long deduplication is active.

| TTL | Use Case |
|-----|----------|
| 1 hour | High-throughput, short-lived operations |
| 24 hours | Standard — covers most redelivery scenarios |
| 7 days | Critical operations (payments, user creation) |
| No TTL | Never — keys would grow indefinitely |

**Rule:** Default to 24 hours. Only use longer for critical financial/user operations. Never use no-TTL.

## Check-Before-Process, Ack-After-Process

The order of operations is critical:

```
1. Receive message
2. Check idempotency (TryAcquire) ─── duplicate? → ack and skip
3. Process message                 ─── failure? → release key + nack
4. Ack message
```

### Complete Pattern

```csharp
private async Task ProcessMessageAsync(BasicDeliverEventArgs ea, CancellationToken ct)
{
    var messageId = ea.BasicProperties?.MessageId ?? ea.DeliveryTag.ToString();

    try
    {
        // STEP 1: Idempotency check
        if (!await _idempotencyStore.TryAcquireAsync(messageId))
        {
            _logger.LogWarning("Duplicate message skipped: {MessageId}", messageId);
            // Ack the duplicate — it was already processed successfully
            await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
            return;
        }

        // STEP 2: Deserialize and validate
        var body = Encoding.UTF8.GetString(ea.Body.Span);
        var envelope = JsonSerializer.Deserialize<MessageEnvelope<EmailMessage>>(body);

        if (envelope?.Data is null)
        {
            _logger.LogError("Invalid message body: {MessageId}", messageId);
            // Don't release key — invalid message should not be retried
            await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: false);
            return;
        }

        // STEP 3: Process
        await DoWorkAsync(envelope.Data, ct);

        // STEP 4: Ack AFTER successful processing
        await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
        _logger.LogInformation("Message processed: {MessageId}", messageId);
    }
    catch (Exception ex) when (ex is not OperationCanceledException)
    {
        _logger.LogError(ex, "Failed to process: {MessageId}", messageId);

        // CRITICAL: Release the idempotency key so retry can process again
        await _idempotencyStore.ReleaseAsync(messageId);

        // Nack — requeue or DLQ based on retry count
        var retries = GetRetryCount(ea.BasicProperties);
        if (retries >= MaxRetries)
        {
            await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: false);
        }
        else
        {
            await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: true);
        }
    }
}
```

## Edge Cases

### 1. Consumer crashes between process and ack
The idempotency key is set, but the message is not acked. RabbitMQ redelivers. The consumer sees the key and skips.

**Problem:** The message was processed but never acked. The idempotency key prevents reprocessing.

**Solution:** This is the correct behavior IF the processing was successful. If the processing was only partial (e.g., wrote to DB but did not send HTTP request), the consumer needs to be designed to handle partial completion — either make the entire operation atomic, or make each sub-operation independently idempotent.

### 2. Redis is down
If Redis is unavailable, `TryAcquireAsync` will throw.

**Solution:** Catch Redis exceptions and decide: either nack (wait for Redis to come back) or process without idempotency (risky but keeps the pipeline moving). The choice depends on the business criticality.

```csharp
bool isFirstTime;
try
{
    isFirstTime = await _idempotencyStore.TryAcquireAsync(messageId);
}
catch (RedisConnectionException ex)
{
    _logger.LogWarning(ex, "Redis unavailable for idempotency check, requeueing: {MessageId}", messageId);
    await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: true);
    return;
}
```

### 3. Key expires before redelivery
If the TTL is too short and redelivery happens after key expiration, the message will be processed again.

**Solution:** Use a TTL longer than the maximum expected redelivery delay. 24 hours covers virtually all scenarios.

## Rules

1. **Every consumer MUST be idempotent.** No exceptions.
2. **Use Redis SETNX** — it is atomic and exactly-once by design.
3. **TTL is mandatory** — keys without TTL grow indefinitely and leak memory.
4. **Release key on failure** — so retry can process the message again.
5. **Do NOT release key on success** — let it expire naturally. Releasing opens a window for duplicate processing.
6. **Business-level keys when possible** — `order:123:confirm` is better than a random UUID for preventing duplicate operations.
7. **Different prefix per consumer** — `email-idem`, `log-idem`, `payment-idem`. Prevents key collision across consumers.
