---
knowledge-base-summary: "DLX configuration for failed messages. Retry with delay using DLX + TTL queue that routes back to the original queue. Max retry count tracking via `x-death` message headers. Poison message handling — move to parking lot queue after N retries. Code examples."
---
# Retry & Dead Letter Exchange

Failed messages must never silently disappear. DLX (Dead Letter Exchange) is the safety net, and retry with delay gives transient failures a chance to recover.

## Dead Letter Exchange (DLX) Basics

A message is dead-lettered when:
1. Consumer sends `BasicNack` or `BasicReject` with `requeue: false`
2. Message TTL expires
3. Queue length limit is exceeded

The queue's `x-dead-letter-exchange` argument determines where dead-lettered messages go.

### DLX Setup

```csharp
// 1. Declare the DLX (fanout — routes to all bound DLQs)
await channel.ExchangeDeclareAsync("emails.dlx", ExchangeType.Fanout, durable: true);

// 2. Declare the DLQ (Dead Letter Queue)
await channel.QueueDeclareAsync("emails.smtp.dlq", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("emails.smtp.dlq", "emails.dlx", routingKey: "");

// 3. Declare the main queue with DLX argument
var args = new Dictionary<string, object?>
{
    { "x-dead-letter-exchange", "emails.dlx" }
};
await channel.QueueDeclareAsync("emails.smtp", durable: true, exclusive: false, autoDelete: false, arguments: args);
```

## Retry with Delay (DLX + TTL Queue)

For transient failures (network timeout, temporary service unavailability), you want to retry after a delay. The pattern uses a TTL queue that dead-letters back to the original exchange.

### How It Works

```
                   nack (requeue: false)
Main Queue  ──────────────────────────>  Retry Exchange
                                              │
                                              ▼
                                       Retry Queue (TTL: 30s)
                                              │
                                        TTL expires (dead-letter)
                                              │
                                              ▼
                                       Main Exchange
                                              │
                                              ▼
                                        Main Queue (re-consumed)
```

### Retry Topology Setup

```csharp
public static class EmailRetryTopology
{
    public const string MainExchange = "emails.fanout";
    public const string MainQueue = "emails.smtp";

    public const string RetryExchange = "emails.retry";
    public const string RetryQueue = "emails.smtp.retry";
    public const int RetryDelayMs = 30_000; // 30 seconds

    public const string DlxExchange = "emails.dlx";
    public const string DlqQueue = "emails.smtp.dlq";

    public const int MaxRetries = 3;

    public static async Task DeclareAsync(IChannel channel)
    {
        // 1. DLX (final resting place for poison messages)
        await channel.ExchangeDeclareAsync(DlxExchange, ExchangeType.Fanout, durable: true);
        await channel.QueueDeclareAsync(DlqQueue, durable: true, exclusive: false, autoDelete: false);
        await channel.QueueBindAsync(DlqQueue, DlxExchange, routingKey: "");

        // 2. Main exchange
        await channel.ExchangeDeclareAsync(MainExchange, ExchangeType.Fanout, durable: true);

        // 3. Main queue — DLX points to retry exchange (not final DLX)
        var mainArgs = new Dictionary<string, object?>
        {
            { "x-dead-letter-exchange", RetryExchange }
        };
        await channel.QueueDeclareAsync(MainQueue, durable: true, exclusive: false, autoDelete: false, arguments: mainArgs);
        await channel.QueueBindAsync(MainQueue, MainExchange, routingKey: "");

        // 4. Retry exchange
        await channel.ExchangeDeclareAsync(RetryExchange, ExchangeType.Fanout, durable: true);

        // 5. Retry queue — TTL + dead-letter back to main exchange
        var retryArgs = new Dictionary<string, object?>
        {
            { "x-message-ttl", RetryDelayMs },
            { "x-dead-letter-exchange", MainExchange }
        };
        await channel.QueueDeclareAsync(RetryQueue, durable: true, exclusive: false, autoDelete: false, arguments: retryArgs);
        await channel.QueueBindAsync(RetryQueue, RetryExchange, routingKey: "");
    }
}
```

## Retry Count Tracking via x-death Headers

When a message is dead-lettered, RabbitMQ adds (or updates) the `x-death` header. This header contains the history of dead-lettering events, including a count.

### Reading Retry Count

```csharp
public static int GetRetryCount(IReadOnlyBasicProperties? props)
{
    if (props?.Headers is null) return 0;

    if (!props.Headers.TryGetValue("x-death", out var xDeathObj)) return 0;

    if (xDeathObj is not List<object> xDeathList || xDeathList.Count == 0) return 0;

    // x-death is a list of entries, each representing a dead-letter event.
    // Sum up the counts to get total retries.
    var totalRetries = 0;

    foreach (var entry in xDeathList)
    {
        if (entry is Dictionary<string, object> dict
            && dict.TryGetValue("count", out var countObj))
        {
            totalRetries += Convert.ToInt32(countObj);
        }
    }

    return totalRetries;
}
```

### Using Retry Count in Consumer

```csharp
private async Task ProcessMessageAsync(BasicDeliverEventArgs ea, CancellationToken ct)
{
    var messageId = ea.BasicProperties?.MessageId ?? ea.DeliveryTag.ToString();

    try
    {
        // ... process message ...
        await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
    }
    catch (TransientException ex)
    {
        // Transient failure — retry
        var retries = GetRetryCount(ea.BasicProperties);
        _logger.LogWarning(ex, "Transient failure on {MessageId}, retry {Count}/{Max}",
            messageId, retries + 1, MaxRetries);

        if (retries >= MaxRetries)
        {
            // Exceeded max retries — this is now a poison message
            // Nack goes to retry exchange (per queue config),
            // but we manually route to the final DLX instead
            await PublishToParkingLotAsync(ea);
            await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
        }
        else
        {
            // Nack without requeue — goes to retry exchange via DLX
            await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: false);
        }
    }
    catch (PermanentException ex)
    {
        // Permanent failure — no point retrying, go straight to DLQ
        _logger.LogError(ex, "Permanent failure on {MessageId}, moving to parking lot", messageId);
        await PublishToParkingLotAsync(ea);
        await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
    }
}
```

## Poison Message Handling (Parking Lot)

Messages that exceed max retries are "poison messages." They go to a parking lot (final DLQ) for manual inspection.

```csharp
private async Task PublishToParkingLotAsync(BasicDeliverEventArgs ea)
{
    // Publish to the final DLX with additional diagnostic headers
    var props = new BasicProperties
    {
        ContentType = ea.BasicProperties?.ContentType,
        ContentEncoding = ea.BasicProperties?.ContentEncoding,
        DeliveryMode = DeliveryModes.Persistent,
        MessageId = ea.BasicProperties?.MessageId,
        Headers = new Dictionary<string, object?>
        {
            { "x-original-exchange", ea.Exchange },
            { "x-original-routing-key", ea.RoutingKey },
            { "x-failure-timestamp", DateTimeOffset.UtcNow.ToString("O") },
            { "x-retry-count", GetRetryCount(ea.BasicProperties).ToString() }
        }
    };

    await _channel!.BasicPublishAsync(
        EmailRetryTopology.DlxExchange,
        routingKey: "",
        mandatory: false,
        props,
        ea.Body);
}
```

## Exponential Backoff (Advanced)

For more sophisticated retry, use multiple retry queues with increasing TTLs:

```csharp
// Retry queue 1: 5 seconds
// Retry queue 2: 30 seconds
// Retry queue 3: 5 minutes

// Determine which retry queue based on retry count:
var retryCount = GetRetryCount(ea.BasicProperties);
var retryExchange = retryCount switch
{
    0 => "emails.retry.5s",
    1 => "emails.retry.30s",
    2 => "emails.retry.5m",
    _ => null // max retries exceeded
};
```

This requires declaring separate retry queues with different TTLs, each dead-lettering back to the main exchange.

## Rules

1. **Every queue MUST have a DLX.** No exceptions. Failed messages must go somewhere inspectable.
2. **Retry delay via TTL queues, not Thread.Sleep.** The consumer must stay responsive. TTL queue handles the delay at the RabbitMQ level.
3. **Track retry count via x-death headers.** Do not use custom headers for retry count — x-death is the standard mechanism.
4. **Max 3-5 retries.** More than that means the failure is not transient. Move to parking lot.
5. **Parking lot is for manual inspection.** Set up alerts when messages arrive in the parking lot queue. These need human attention.
6. **Distinguish transient vs permanent failures.** Transient (timeout, temporary unavailability) -> retry. Permanent (bad data, validation failure) -> parking lot immediately.
