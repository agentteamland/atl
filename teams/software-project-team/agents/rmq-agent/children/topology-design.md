---
knowledge-base-summary: "Exchange types (fanout, direct, topic) and when to use each. Queue naming convention: `{purpose}.{consumer}` (e.g., `emails.smtp`). Exchange naming: `{purpose}.{type}` (e.g., `emails.fanout`). Binding patterns and routing key strategies."
---
# Topology Design

RabbitMQ topology is the arrangement of exchanges, queues, and bindings. Good topology design determines message flow, decoupling, and scalability.

## Exchange Types

### Fanout Exchange
Routes to ALL bound queues, ignoring routing keys.

**When to use:**
- Broadcast scenarios: every consumer gets every message
- Logs (every log goes to Elasticsearch, potentially also to file, alerting)
- Notifications (email + push + SMS all get the same event)

**Example:** `logs.fanout` -> bound to `logs.elasticsearch`, `logs.file`, `logs.alerting`

### Direct Exchange
Routes to queues whose binding key EXACTLY matches the routing key.

**When to use:**
- Targeted routing: only one specific consumer should get the message
- Task queues with specific routing (e.g., route by priority or region)
- When you need to selectively deliver based on an exact key

**Example:** `tasks.direct` with routing key `priority.high` -> only the `tasks.high-priority` queue

### Topic Exchange
Routes based on pattern matching with `.` delimiters, `*` (one word), `#` (zero or more words).

**When to use:**
- Multi-dimensional routing (e.g., by region AND type)
- When consumers want to subscribe to a subset of messages
- Event systems where consumers filter by event category

**Example:** `events.topic` with patterns:
- `user.created` -> user service queue
- `order.*` -> order service queue (matches `order.created`, `order.cancelled`)
- `#` -> audit queue (gets everything)

### Headers Exchange (Rare)
Routes based on message header attributes instead of routing key.

**When to use:** Almost never in practice. Topic exchange covers most pattern-matching needs.

### Default Exchange
Nameless direct exchange where routing key = queue name. Every queue is automatically bound.

**When to use:** Simple point-to-point when you just need one producer and one consumer with no routing logic.

## Naming Conventions

### Exchange Names
```
{purpose}.{type}
```

| Exchange | Type | Example |
|----------|------|---------|
| Email delivery | fanout | `emails.fanout` |
| Log pipeline | fanout | `logs.fanout` |
| Task routing | direct | `tasks.direct` |
| Event broadcasting | topic | `events.topic` |
| Dead letter | fanout | `emails.dlx` |

### Queue Names
```
{purpose}.{consumer}
```

| Queue | Consumer | Example |
|-------|----------|---------|
| Email via SMTP | MailSender | `emails.smtp` |
| Logs to Elasticsearch | LogIngest | `logs.elasticsearch` |
| High-priority tasks | Worker | `tasks.high-priority` |
| Dead letter queue | Manual inspection | `emails.smtp.dlq` |
| Retry with delay | Retry mechanism | `emails.smtp.retry` |

### Routing Key Patterns (for direct/topic)
```
{entity}.{action}             # user.created, order.cancelled
{entity}.{action}.{region}    # user.created.eu, order.cancelled.us
```

## Binding Patterns

### One-to-One (Direct/Default)
```
Producer -> Exchange -> Queue -> Consumer
```
Simple. One message, one consumer. Use direct exchange or default exchange.

### One-to-Many (Fanout)
```
Producer -> Exchange -> Queue A -> Consumer A
                    \-> Queue B -> Consumer B
                    \-> Queue C -> Consumer C
```
Every consumer gets every message. Each consumer has its own queue.

### Selective (Topic)
```
Producer -> Exchange -> Queue A (pattern: user.*)     -> User Consumer
                    \-> Queue B (pattern: order.*)    -> Order Consumer
                    \-> Queue C (pattern: #)          -> Audit Consumer
```
Consumers subscribe to the patterns they care about.

### Competing Consumers (Work Queue)
```
Producer -> Exchange -> Queue -> Consumer A (instance 1)
                            \-> Consumer A (instance 2)
                            \-> Consumer A (instance 3)
```
Multiple instances of the same consumer share a single queue. RabbitMQ round-robins messages. Use `prefetchCount: 1` for fair dispatch.

## Topology Declaration in Code

```csharp
// BOTH producer and consumer declare the same topology.
// Declarations are idempotent — if it exists with the same config, RabbitMQ ignores it.
// This eliminates startup order dependencies.

public static class EmailTopology
{
    public const string Exchange = "emails.fanout";
    public const string Queue = "emails.smtp";
    public const string DlxExchange = "emails.dlx";
    public const string DlqQueue = "emails.smtp.dlq";

    public static async Task DeclareAsync(IChannel channel)
    {
        // 1. DLX infrastructure first
        await channel.ExchangeDeclareAsync(DlxExchange, ExchangeType.Fanout, durable: true);
        await channel.QueueDeclareAsync(DlqQueue, durable: true, exclusive: false, autoDelete: false);
        await channel.QueueBindAsync(DlqQueue, DlxExchange, routingKey: "");

        // 2. Main exchange
        await channel.ExchangeDeclareAsync(Exchange, ExchangeType.Fanout, durable: true);

        // 3. Main queue with DLX argument
        var args = new Dictionary<string, object?>
        {
            { "x-dead-letter-exchange", DlxExchange }
        };

        await channel.QueueDeclareAsync(Queue, durable: true, exclusive: false, autoDelete: false, arguments: args);
        await channel.QueueBindAsync(Queue, Exchange, routingKey: "");
    }
}
```

## Rules

1. **Topology constants in a shared static class.** Both producer and consumer reference the same constants. No magic strings.
2. **Declare DLX/DLQ before the main queue.** The main queue's `x-dead-letter-exchange` argument references the DLX — it must exist.
3. **Never use auto-delete queues in production.** If the consumer disconnects temporarily, the queue (and its messages) would be deleted.
4. **Never use exclusive queues in production.** Exclusive queues are connection-scoped and deleted when the connection closes.
5. **Always set durable: true** for exchanges and queues. Transient entities are lost on RabbitMQ restart.
