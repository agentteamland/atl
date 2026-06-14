---
knowledge-base-summary: "Detailed examples for each exchange type. Fanout for broadcast (logs, notifications). Direct for targeted routing. Topic for pattern-based routing. Headers exchange (rare). Default exchange for simple point-to-point. Complete C# code examples for each pattern."
---
# Exchange Patterns

Detailed examples for each RabbitMQ exchange type with complete C# code.

## 1. Fanout Exchange (Broadcast)

Every bound queue receives every message. Routing key is ignored.

**Use cases:** Logs, notifications, event broadcasting, cache invalidation.

### Producer

```csharp
public sealed class LogPublisher
{
    private readonly IRabbitMqConnection _connection;

    public async Task PublishAsync(LogEntry entry, CancellationToken ct = default)
    {
        await using var channel = await _connection.GetChannelAsync(ct);

        // Declare topology (idempotent)
        await channel.ExchangeDeclareAsync("logs.fanout", ExchangeType.Fanout, durable: true);

        var envelope = new MessageEnvelope<LogEntry>
        {
            Type = "log.entry",
            Version = 1,
            Data = entry,
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString()
        };

        var body = JsonSerializer.SerializeToUtf8Bytes(envelope);

        var props = new BasicProperties
        {
            ContentType = "application/json",
            ContentEncoding = "utf-8",
            DeliveryMode = DeliveryModes.Persistent,
            MessageId = Guid.NewGuid().ToString(),
            Timestamp = new AmqpTimestamp(DateTimeOffset.UtcNow.ToUnixTimeSeconds())
        };

        // Routing key is empty — fanout ignores it
        await channel.BasicPublishAsync("logs.fanout", routingKey: "", mandatory: false, props, body);
    }
}
```

### Consumer (Elasticsearch)

```csharp
// Queue: logs.elasticsearch — gets ALL logs
await channel.QueueDeclareAsync("logs.elasticsearch", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("logs.elasticsearch", "logs.fanout", routingKey: "");
```

### Consumer (File backup)

```csharp
// Queue: logs.file — also gets ALL logs (independent consumer)
await channel.QueueDeclareAsync("logs.file", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("logs.file", "logs.fanout", routingKey: "");
```

**Key point:** Adding a new consumer is just adding a new queue + binding. No producer changes needed.

## 2. Direct Exchange (Targeted Routing)

Routes messages to queues whose binding key exactly matches the routing key.

**Use cases:** Task queues by priority, routing by type, targeted delivery.

### Producer

```csharp
public sealed class TaskPublisher
{
    public async Task PublishAsync(TaskMessage task, string priority, CancellationToken ct = default)
    {
        await using var channel = await _connection.GetChannelAsync(ct);

        await channel.ExchangeDeclareAsync("tasks.direct", ExchangeType.Direct, durable: true);

        var body = JsonSerializer.SerializeToUtf8Bytes(WrapEnvelope(task));
        var props = CreatePersistentProperties();

        // Routing key determines which queue gets the message
        await channel.BasicPublishAsync("tasks.direct", routingKey: priority, mandatory: false, props, body);
    }
}

// Usage:
await publisher.PublishAsync(task, priority: "high");    // -> tasks.high queue
await publisher.PublishAsync(task, priority: "low");     // -> tasks.low queue
```

### Consumers

```csharp
// High-priority queue
await channel.QueueDeclareAsync("tasks.high", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("tasks.high", "tasks.direct", routingKey: "high");

// Low-priority queue
await channel.QueueDeclareAsync("tasks.low", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("tasks.low", "tasks.direct", routingKey: "low");
```

**Key point:** A queue can bind with multiple routing keys:

```csharp
// This queue receives BOTH high and medium priority
await channel.QueueBindAsync("tasks.urgent", "tasks.direct", routingKey: "high");
await channel.QueueBindAsync("tasks.urgent", "tasks.direct", routingKey: "medium");
```

## 3. Topic Exchange (Pattern-Based Routing)

Routes based on routing key patterns. `.` separates words, `*` matches one word, `#` matches zero or more.

**Use cases:** Event systems, multi-dimensional routing, selective subscriptions.

### Producer

```csharp
public sealed class EventPublisher
{
    public async Task PublishAsync<T>(string eventType, T data, CancellationToken ct = default)
    {
        await using var channel = await _connection.GetChannelAsync(ct);

        await channel.ExchangeDeclareAsync("events.topic", ExchangeType.Topic, durable: true);

        var body = JsonSerializer.SerializeToUtf8Bytes(WrapEnvelope(data));
        var props = CreatePersistentProperties();

        // Routing key is the event type: entity.action
        await channel.BasicPublishAsync("events.topic", routingKey: eventType, mandatory: false, props, body);
    }
}

// Usage:
await publisher.PublishAsync("user.created", userData);
await publisher.PublishAsync("user.deleted", userId);
await publisher.PublishAsync("order.created", orderData);
await publisher.PublishAsync("order.shipped", shipmentData);
await publisher.PublishAsync("payment.completed", paymentData);
```

### Consumers with Pattern Matching

```csharp
// User service — gets all user events
await channel.QueueDeclareAsync("events.user-service", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("events.user-service", "events.topic", routingKey: "user.*");
// Matches: user.created, user.deleted, user.updated

// Order service — gets all order events
await channel.QueueDeclareAsync("events.order-service", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("events.order-service", "events.topic", routingKey: "order.*");
// Matches: order.created, order.shipped, order.cancelled

// Audit service — gets EVERYTHING
await channel.QueueDeclareAsync("events.audit", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("events.audit", "events.topic", routingKey: "#");
// Matches: all events

// Notification service — gets only creation events
await channel.QueueDeclareAsync("events.notifications", durable: true, exclusive: false, autoDelete: false);
await channel.QueueBindAsync("events.notifications", "events.topic", routingKey: "*.created");
// Matches: user.created, order.created, payment.created
```

### Multi-Level Routing Keys

```csharp
// With 3 segments: entity.action.region
await publisher.PublishAsync("order.created.eu", orderData);
await publisher.PublishAsync("order.created.us", orderData);

// EU-only consumer
await channel.QueueBindAsync("events.eu-orders", "events.topic", routingKey: "order.*.eu");

// All EU events
await channel.QueueBindAsync("events.eu-all", "events.topic", routingKey: "*.*.eu");
```

## 4. Headers Exchange (Rare)

Routes based on message header attributes instead of routing key.

**When to use:** Almost never. Topic exchange handles most pattern-matching. Headers exchange is useful when routing needs to match on multiple independent attributes simultaneously.

```csharp
// Declare
await channel.ExchangeDeclareAsync("reports.headers", ExchangeType.Headers, durable: true);

// Bind with header matching
var bindArgs = new Dictionary<string, object?>
{
    { "x-match", "all" },        // "all" = AND, "any" = OR
    { "format", "pdf" },
    { "department", "finance" }
};
await channel.QueueBindAsync("reports.finance-pdf", "reports.headers", routingKey: "", arguments: bindArgs);

// Publish with matching headers
var props = new BasicProperties
{
    Headers = new Dictionary<string, object?>
    {
        { "format", "pdf" },
        { "department", "finance" }
    }
};
await channel.BasicPublishAsync("reports.headers", routingKey: "", mandatory: false, props, body);
```

## 5. Default Exchange (Simple Point-to-Point)

The default (nameless) exchange routes directly to a queue by name. Every queue is automatically bound to it with the queue name as routing key.

**When to use:** Simple scenarios with one producer and one consumer, no routing logic needed.

```csharp
// No exchange declaration needed — default exchange always exists

// Declare the queue
await channel.QueueDeclareAsync("simple-tasks", durable: true, exclusive: false, autoDelete: false);

// Publish directly to queue (empty exchange name, routing key = queue name)
await channel.BasicPublishAsync(
    exchange: "",
    routingKey: "simple-tasks",
    mandatory: false,
    props,
    body);
```

## Choosing the Right Pattern

| Scenario | Exchange Type | Why |
|----------|--------------|-----|
| Log pipeline | Fanout | Every log consumer gets every log |
| Email sending | Fanout | Currently one consumer, but fanout allows adding SMS/push later |
| Task by priority | Direct | Exact match on priority level |
| Event system | Topic | Consumers selectively subscribe by pattern |
| Simple job queue | Default | One producer, one consumer, no routing |
| Multi-attribute routing | Headers | Complex matching on multiple fields (rare) |

## Anti-Patterns

1. **Using direct when you need fanout.** If you hardcode a single queue name in the producer, adding a second consumer requires changing the producer. Use fanout — adding consumers is just adding queues.

2. **Using topic when direct is sufficient.** If routing keys are simple exact matches, topic adds unnecessary overhead. Use direct.

3. **Publishing directly to a queue (default exchange) in production.** This tightly couples producer to consumer. Use a named exchange for flexibility.

4. **Creating a new exchange per message type.** Exchanges should group related messages. One exchange for all user events, not one exchange per user event.
