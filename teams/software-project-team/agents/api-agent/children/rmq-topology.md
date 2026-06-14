---
knowledge-base-summary: "Every consumer (MailSender, LogIngest, etc.) MUST declare its own RMQ topology at startup — exchange, queue, binding. Consumer may start before API — cannot assume topology exists. Declare with the SAME arguments (including DLX if any), otherwise RabbitMQ throws `PRECONDITION_FAILED`. Idempotent — declaring twice is safe."
---
# RMQ Consumer Topology Rule

Every consumer (MailSender, LogIngest, etc.) must **idempotently declare** its own RMQ topology at startup: exchange, queue, and binding. There is a chance the consumer starts before the API — it cannot assume the topology exists.

When declaring a queue, it must declare with **the same arguments** (e.g., if DLX exists, the DLX argument must also be passed). Otherwise RabbitMQ throws `PRECONDITION_FAILED`.

## Example: Email Consumer

```csharp
// Inside consumer ConnectAsync(), after creating the channel:
await _channel.ExchangeDeclareAsync("emails.fanout", ExchangeType.Fanout, durable: true, cancellationToken: ct);
await _channel.ExchangeDeclareAsync("emails.dlx", ExchangeType.Fanout, durable: true, cancellationToken: ct);
var queueArgs = new Dictionary<string, object?>
{
    { "x-dead-letter-exchange", "emails.dlx" },
};
await _channel.QueueDeclareAsync("emails.smtp", durable: true, exclusive: false,
    autoDelete: false, arguments: queueArgs, cancellationToken: ct);
await _channel.QueueBindAsync("emails.smtp", "emails.fanout", "", cancellationToken: ct);
```

## Example: Log Consumer

```csharp
await _channel.ExchangeDeclareAsync("logs.fanout", ExchangeType.Fanout, durable: true, cancellationToken: ct);
await _channel.QueueDeclareAsync("logs.elasticsearch", durable: true, exclusive: false,
    autoDelete: false, cancellationToken: ct);
await _channel.QueueBindAsync("logs.elasticsearch", "logs.fanout", "", cancellationToken: ct);
```

## Rule

This rule applies to both the producer side (Infrastructure/Messaging) and the consumer side. Both sides can declare the same topology — RabbitMQ behaves idempotently (re-declaring with the same arguments is not a problem).

