---
knowledge-base-summary: "RabbitMQ Management UI (:15672) for queue inspection. Queue depth monitoring — messages piling up means consumer is too slow. Consumer utilization, unacked messages, DLX queue depth. Alerting thresholds. Kibana dashboard integration for consumer logs."
---
# Monitoring

Effective monitoring detects problems before they become incidents. A message queue without monitoring is a time bomb.

## RabbitMQ Management UI

Available at `http://localhost:15672` (default credentials: guest/guest in development).

### Key Pages

| Page | What to Check |
|------|---------------|
| Overview | Total messages, publish/deliver rates, connection count |
| Queues | Queue depth (messages ready + unacked), consumer count |
| Exchanges | Publish rate per exchange |
| Connections | Active connections, per-connection channel count |
| Channels | Prefetch count, unacked messages per channel |

### Docker Compose Setup

```yaml
services:
  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"     # AMQP
      - "15672:15672"   # Management UI
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    healthcheck:
      test: rabbitmq-diagnostics -q ping
      interval: 30s
      timeout: 10s
      retries: 3
```

## Queue Depth Monitoring

The single most important metric. Messages piling up means the consumer is slower than the producer.

### What to Watch

| Metric | Healthy | Warning | Critical |
|--------|---------|---------|----------|
| Messages Ready | < 100 | 100 - 1000 | > 1000 |
| Messages Unacked | < prefetchCount | 2x prefetchCount | > 5x prefetchCount |
| Consumer Count | >= 1 | 0 (briefly) | 0 (sustained) |
| Message Rate (in/out) | in <= out | in > out (briefly) | in >> out (sustained) |
| DLQ Depth | 0 | 1-10 | > 10 |

### Alerting Thresholds

```
RULE: queue_depth_warning
  WHEN messages_ready > 500 FOR 5 minutes
  THEN alert (Slack/email)

RULE: queue_depth_critical
  WHEN messages_ready > 2000 FOR 2 minutes
  THEN page on-call

RULE: consumer_down
  WHEN consumer_count = 0 FOR 1 minute
  THEN page on-call

RULE: dlq_messages
  WHEN dlq_messages_ready > 0 FOR 1 minute
  THEN alert (requires manual inspection)
```

## Consumer Health Tracking

Each consumer tracks its own health state in Redis for external monitoring.

### Health State Model

```csharp
public sealed record ConsumerHealthState
{
    public string ConsumerName { get; init; } = "";
    public DateTimeOffset LastHeartbeat { get; init; }
    public DateTimeOffset? LastSuccessfulProcess { get; init; }
    public DateTimeOffset? LastFailedProcess { get; init; }
    public int ConsecutiveFailures { get; init; }
    public long TotalProcessed { get; init; }
    public long TotalFailed { get; init; }
    public bool IsHealthy => ConsecutiveFailures < 5
                             && LastHeartbeat > DateTimeOffset.UtcNow.AddMinutes(-2);
}
```

### Updating Health State

```csharp
public sealed class ConsumerHealthTracker
{
    private readonly IConnectionMultiplexer _redis;
    private readonly string _consumerName;
    private long _totalProcessed;
    private long _totalFailed;
    private int _consecutiveFailures;

    public ConsumerHealthTracker(IConnectionMultiplexer redis, string consumerName)
    {
        _redis = redis;
        _consumerName = consumerName;
    }

    public async Task RecordSuccessAsync()
    {
        Interlocked.Increment(ref _totalProcessed);
        Interlocked.Exchange(ref _consecutiveFailures, 0);
        await UpdateRedisAsync(success: true);
    }

    public async Task RecordFailureAsync()
    {
        Interlocked.Increment(ref _totalFailed);
        Interlocked.Increment(ref _consecutiveFailures);
        await UpdateRedisAsync(success: false);
    }

    public async Task HeartbeatAsync()
    {
        var db = _redis.GetDatabase();
        var key = $"consumer:health:{_consumerName}";

        var state = new ConsumerHealthState
        {
            ConsumerName = _consumerName,
            LastHeartbeat = DateTimeOffset.UtcNow,
            TotalProcessed = _totalProcessed,
            TotalFailed = _totalFailed,
            ConsecutiveFailures = _consecutiveFailures
        };

        var json = JsonSerializer.Serialize(state);
        await db.StringSetAsync(key, json, TimeSpan.FromMinutes(5));
    }

    private async Task UpdateRedisAsync(bool success)
    {
        var db = _redis.GetDatabase();
        var key = $"consumer:health:{_consumerName}";
        var now = DateTimeOffset.UtcNow;

        var state = new ConsumerHealthState
        {
            ConsumerName = _consumerName,
            LastHeartbeat = now,
            LastSuccessfulProcess = success ? now : null,
            LastFailedProcess = success ? null : now,
            TotalProcessed = _totalProcessed,
            TotalFailed = _totalFailed,
            ConsecutiveFailures = _consecutiveFailures
        };

        var json = JsonSerializer.Serialize(state);
        await db.StringSetAsync(key, json, TimeSpan.FromMinutes(5));
    }
}
```

### Using Health Tracker in Consumer

```csharp
private async Task ProcessMessageAsync(BasicDeliverEventArgs ea, CancellationToken ct)
{
    try
    {
        // ... process message ...
        await _healthTracker.RecordSuccessAsync();
        await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
    }
    catch (Exception ex)
    {
        await _healthTracker.RecordFailureAsync();
        // ... error handling ...
    }
}
```

## Consumer Logging for Kibana

Every consumer logs structured events that flow through the standard log pipeline (Serilog -> RMQ -> LogIngest -> Elasticsearch -> Kibana).

### Log Events to Emit

```csharp
// Consumer started
_logger.LogInformation("Consumer started: {ConsumerName}, Queue: {QueueName}", name, queue);

// Message received
_logger.LogDebug("Message received: {MessageId}, Queue: {QueueName}", messageId, queue);

// Message processed successfully
_logger.LogInformation("Message processed: {MessageId}, Duration: {DurationMs}ms", messageId, sw.ElapsedMilliseconds);

// Duplicate skipped
_logger.LogWarning("Duplicate message skipped: {MessageId}", messageId);

// Processing failed (transient)
_logger.LogWarning(ex, "Transient failure: {MessageId}, Retry: {RetryCount}/{MaxRetries}", messageId, retries, max);

// Processing failed (permanent / DLQ)
_logger.LogError(ex, "Permanent failure, moved to DLQ: {MessageId}", messageId);

// Consumer stopped
_logger.LogInformation("Consumer stopped: {ConsumerName}", name);

// Connection lost
_logger.LogWarning("RabbitMQ connection lost, waiting for recovery...");

// Connection recovered
_logger.LogInformation("RabbitMQ connection recovered");
```

### Kibana Dashboard Queries

```
# Messages processed per minute
consumer.name: "EmailConsumer" AND message: "Message processed"

# Failed messages
consumer.name: "EmailConsumer" AND level: "Error"

# DLQ messages
message: "moved to DLQ"

# Consumer health
consumer.name: * AND message: "Consumer started" OR message: "Consumer stopped"

# Processing duration (slow messages)
consumer.name: "EmailConsumer" AND DurationMs: >5000
```

## RabbitMQ HTTP API

For programmatic monitoring, use the RabbitMQ Management HTTP API.

### Queue Stats

```bash
# Get queue depth
curl -u guest:guest http://localhost:15672/api/queues/%2F/emails.smtp

# Response includes:
# messages_ready: waiting to be consumed
# messages_unacknowledged: delivered but not acked
# consumers: number of connected consumers
# message_stats.publish_details.rate: messages/sec published
# message_stats.deliver_details.rate: messages/sec delivered
```

### Automated Health Check Script

```bash
#!/bin/bash
# Check if any queue has messages piling up

QUEUES=$(curl -s -u guest:guest http://localhost:15672/api/queues/%2F/ \
  | jq '.[] | select(.messages_ready > 500) | .name')

if [ -n "$QUEUES" ]; then
  echo "WARNING: Queues with high message count: $QUEUES"
  # Send alert
fi

# Check DLQ queues
DLQ=$(curl -s -u guest:guest http://localhost:15672/api/queues/%2F/ \
  | jq '.[] | select(.name | endswith(".dlq")) | select(.messages_ready > 0) | {name, messages_ready}')

if [ -n "$DLQ" ]; then
  echo "ALERT: Messages in Dead Letter Queues: $DLQ"
  # Page on-call
fi
```

## Troubleshooting Checklist

| Symptom | Likely Cause | Action |
|---------|-------------|--------|
| Messages piling up | Consumer too slow | Scale consumers, increase prefetch, optimize processing |
| High unacked count | Consumer hanging | Check for deadlocks, timeouts, blocked I/O |
| Consumer count = 0 | Consumer crashed | Check container logs, restart service |
| DLQ has messages | Processing errors | Inspect DLQ messages, fix consumer bug, replay |
| Connection churn | Network issues | Check heartbeat config, network stability |
| Memory alarm | Too many messages | Consumers are too slow, enable flow control |
| Disk alarm | Persistent messages filling disk | Consume faster or increase disk |

## Rules

1. **Monitor queue depth always.** It is the single most important indicator of system health.
2. **Alert on DLQ messages immediately.** Any message in the DLQ needs human attention.
3. **Consumer count = 0 is critical.** If a queue has no consumers, messages pile up indefinitely.
4. **Log every decision.** Start, stop, process, skip, fail, dlq — every consumer action must be logged.
5. **Health tracker in Redis** with TTL. If the key expires, the consumer has been silent too long.
6. **Structured logging** with consistent field names (MessageId, QueueName, ConsumerName, DurationMs). This enables Kibana dashboards and alerts.
7. **Management UI is for development.** In production, use HTTP API + automated alerting.
