---
knowledge-base-summary: "The primary production unit of this agent. Template + checklist + naming conventions for creating new RMQ consumers. Read this FIRST when adding any new consumer. Covers: BackgroundService skeleton, connect, declare topology, consume, process, ack/nack, DLX, idempotency, error handling, health tracking."
---
# Consumer Blueprint (Primary Production Unit)

This is the RMQ Agent's primary blueprint. Every new RabbitMQ consumer follows this template and checklist.

## File Structure Convention

```
src/{ProjectName}.{ConsumerName}/
  Program.cs                          # Host setup, DI registration
  {Purpose}Consumer.cs                # BackgroundService — connect, consume, process
  appsettings.json                    # RMQ connection string, consumer-specific settings
  Dockerfile                          # Multi-stage build
```

## Naming Conventions

| Item | Pattern | Example |
|------|---------|---------|
| Project | `{ProjectName}.{ConsumerName}` | `ExampleApp.MailSender` |
| Consumer class | `{Purpose}Consumer` | `EmailConsumer` |
| Exchange | `{purpose}.{type}` | `emails.fanout` |
| Queue | `{purpose}.{consumer}` | `emails.smtp` |
| DLX | `{purpose}.dlx` | `emails.dlx` |
| DLQ | `{purpose}.{consumer}.dlq` | `emails.smtp.dlq` |
| Retry queue | `{purpose}.{consumer}.retry` | `emails.smtp.retry` |

## Consumer Template

Every consumer follows this exact structure:

```csharp
// src/{ProjectName}.{ConsumerName}/{Purpose}Consumer.cs

using System.Text;
using System.Text.Json;
using RabbitMQ.Client;
using RabbitMQ.Client.Events;

public sealed class EmailConsumer : BackgroundService
{
    private readonly IRabbitMqConnection _rmqConnection;
    private readonly IIdempotencyStore _idempotencyStore;
    private readonly ILogger<EmailConsumer> _logger;
    private IChannel? _channel;

    public EmailConsumer(
        IRabbitMqConnection rmqConnection,
        IIdempotencyStore idempotencyStore,
        ILogger<EmailConsumer> logger)
    {
        _rmqConnection = rmqConnection;
        _idempotencyStore = idempotencyStore;
        _logger = logger;
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        // 1. Get channel
        _channel = await _rmqConnection.GetChannelAsync(stoppingToken);

        // 2. Declare topology (idempotent — safe to call on both producer and consumer)
        await DeclareTopologyAsync();

        // 3. Set prefetch (process one message at a time)
        await _channel.BasicQosAsync(prefetchSize: 0, prefetchCount: 1, global: false);

        // 4. Create async consumer
        var consumer = new AsyncEventingBasicConsumer(_channel);
        consumer.ReceivedAsync += async (sender, ea) =>
        {
            await ProcessMessageAsync(ea, stoppingToken);
        };

        // 5. Start consuming
        await _channel.BasicConsumeAsync(
            queue: "emails.smtp",
            autoAck: false,  // ALWAYS manual ack
            consumer: consumer);

        _logger.LogInformation("EmailConsumer started, listening on queue: emails.smtp");

        // 6. Keep alive until cancellation
        await Task.Delay(Timeout.Infinite, stoppingToken);
    }

    private async Task DeclareTopologyAsync()
    {
        // DLX + DLQ first (must exist before main queue references them)
        await _channel!.ExchangeDeclareAsync(
            exchange: "emails.dlx",
            type: ExchangeType.Fanout,
            durable: true);

        await _channel.QueueDeclareAsync(
            queue: "emails.smtp.dlq",
            durable: true,
            exclusive: false,
            autoDelete: false);

        await _channel.QueueBindAsync(
            queue: "emails.smtp.dlq",
            exchange: "emails.dlx",
            routingKey: "");

        // Main exchange
        await _channel.ExchangeDeclareAsync(
            exchange: "emails.fanout",
            type: ExchangeType.Fanout,
            durable: true);

        // Main queue with DLX argument
        var args = new Dictionary<string, object?>
        {
            { "x-dead-letter-exchange", "emails.dlx" }
        };

        await _channel.QueueDeclareAsync(
            queue: "emails.smtp",
            durable: true,
            exclusive: false,
            autoDelete: false,
            arguments: args);

        await _channel.QueueBindAsync(
            queue: "emails.smtp",
            exchange: "emails.fanout",
            routingKey: "");
    }

    private async Task ProcessMessageAsync(BasicDeliverEventArgs ea, CancellationToken ct)
    {
        var messageId = ea.BasicProperties?.MessageId ?? ea.DeliveryTag.ToString();

        try
        {
            // Idempotency check — skip if already processed
            if (!await _idempotencyStore.TryAcquireAsync(messageId))
            {
                _logger.LogWarning("Duplicate message skipped: {MessageId}", messageId);
                await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
                return;
            }

            // Deserialize
            var body = Encoding.UTF8.GetString(ea.Body.Span);
            var envelope = JsonSerializer.Deserialize<MessageEnvelope<EmailMessage>>(body);

            if (envelope?.Data is null)
            {
                _logger.LogError("Invalid message body, sending to DLQ: {MessageId}", messageId);
                await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: false);
                return;
            }

            // Process (the actual work)
            await ProcessEmailAsync(envelope.Data, ct);

            // Ack AFTER successful processing
            await _channel!.BasicAckAsync(ea.DeliveryTag, multiple: false);
            _logger.LogInformation("Message processed: {MessageId}", messageId);
        }
        catch (Exception ex) when (ex is not OperationCanceledException)
        {
            _logger.LogError(ex, "Failed to process message: {MessageId}", messageId);

            // Check retry count from x-death header
            var retryCount = GetRetryCount(ea.BasicProperties);

            if (retryCount >= 3)
            {
                // Poison message — send to DLQ (nack without requeue)
                _logger.LogError("Max retries exceeded for {MessageId}, moving to DLQ", messageId);
                await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: false);
            }
            else
            {
                // Requeue for retry
                await _channel!.BasicNackAsync(ea.DeliveryTag, multiple: false, requeue: true);
            }

            // Remove idempotency key so retry can process again
            await _idempotencyStore.ReleaseAsync(messageId);
        }
    }

    private int GetRetryCount(IReadOnlyBasicProperties? props)
    {
        if (props?.Headers is null) return 0;
        if (!props.Headers.TryGetValue("x-death", out var xDeath)) return 0;

        if (xDeath is List<object> deaths && deaths.Count > 0
            && deaths[0] is Dictionary<string, object> first
            && first.TryGetValue("count", out var count))
        {
            return Convert.ToInt32(count);
        }

        return 0;
    }

    private async Task ProcessEmailAsync(EmailMessage email, CancellationToken ct)
    {
        // Actual email sending logic here
        // This is where SmtpClient or IEmailSender is called
    }

    public override async Task StopAsync(CancellationToken cancellationToken)
    {
        _logger.LogInformation("EmailConsumer stopping...");

        if (_channel is not null)
        {
            await _channel.CloseAsync();
            _channel.Dispose();
        }

        await base.StopAsync(cancellationToken);
    }
}
```

## Program.cs Template

Required file headers (put these at the top of every consumer host's Program.cs):

```csharp
using Microsoft.Extensions.Hosting;
```

Without `using Microsoft.Extensions.Hosting;`, `Host.CreateApplicationBuilder(args)` fails to compile even though `ImplicitUsings` is enabled — the SDK's implicit usings cover `Microsoft.Extensions.*` for ASP.NET hosts (Sdk.Web) but NOT reliably for `Microsoft.NET.Sdk.Worker` across all minor versions. Always add it explicitly.

Also, every Worker-SDK consumer host must pin the hosting package explicitly in its `.csproj`:

```xml
<ItemGroup>
  <PackageReference Include="Microsoft.Extensions.Hosting" Version="9.0.0" />
  <!-- plus consumer-specific deps (RabbitMQ.Client, StackExchange.Redis, etc.) -->
</ItemGroup>
```

Don't rely on the SDK's transitive reference alone — it's been observed to miss in .NET 9 with Alpine base images. A one-line package reference is cheap insurance.

```csharp
// src/{ProjectName}.{ConsumerName}/Program.cs

using Microsoft.Extensions.Hosting;

var builder = Host.CreateApplicationBuilder(args);

// RabbitMQ connection (singleton)
builder.Services.AddSingleton<IRabbitMqConnection>(sp =>
    new RabbitMqConnection(builder.Configuration.GetConnectionString("RabbitMq")!));

// Idempotency store (Redis)
builder.Services.AddSingleton<IIdempotencyStore>(sp =>
    new RedisIdempotencyStore(
        builder.Configuration.GetConnectionString("Redis")!,
        prefix: "email-idem",
        ttl: TimeSpan.FromHours(24)));

// Register the consumer as a hosted service
builder.Services.AddHostedService<EmailConsumer>();

// Logging
builder.Services.AddSerilogLogging(builder.Configuration);

var host = builder.Build();
await host.RunAsync();
```

## Creation Checklist

Before considering a new consumer complete, verify ALL of the following:

### Topology
- [ ] Exchange declared (durable: true)
- [ ] Queue declared (durable: true)
- [ ] Queue bound to exchange
- [ ] DLX exchange declared
- [ ] DLQ declared and bound to DLX
- [ ] Topology declared in BOTH producer AND consumer (idempotent)

### Consumer Logic
- [ ] `autoAck: false` (manual acknowledgment)
- [ ] `prefetchCount` set (default: 1)
- [ ] Idempotency check BEFORE processing
- [ ] Ack AFTER successful processing
- [ ] Nack without requeue on permanent failure (goes to DLQ)
- [ ] Nack with requeue on transient failure (retry)
- [ ] Retry count checked via x-death header
- [ ] Poison message detection (max retries exceeded -> DLQ)

### Error Handling
- [ ] Try-catch around message processing
- [ ] OperationCanceledException excluded from error handling
- [ ] Deserialization failure -> DLQ (permanent error)
- [ ] Business logic failure -> retry or DLQ based on error type
- [ ] Idempotency key released on failure (so retry can process)

### Infrastructure
- [ ] IRabbitMqConnection injected (singleton)
- [ ] IIdempotencyStore injected (Redis-backed)
- [ ] CancellationToken respected
- [ ] Graceful shutdown (StopAsync closes channel)
- [ ] Logging at every decision point (start, process, skip, fail, dlq)

### Deployment
- [ ] Dockerfile created (multi-stage build)
- [ ] Added to docker-compose.yml
- [ ] Environment variables for RMQ and Redis connection strings
- [ ] Health check endpoint (if applicable)

## Lifecycle

1. **Create** the consumer project with the template above
2. **Declare topology** in both the producer (Infrastructure/Messaging) and consumer
3. **Register** in docker-compose.yml with proper service dependencies
4. **Test** by publishing a message and verifying it is consumed
5. **Verify DLQ** by intentionally causing a failure and checking the dead letter queue
6. **Verify idempotency** by sending the same message twice and confirming single processing
