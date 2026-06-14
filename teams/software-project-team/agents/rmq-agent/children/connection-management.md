---
knowledge-base-summary: "IRabbitMqConnection singleton with lazy initialization. AutoRecovery enabled for resilience. Connection string from environment variables. Channel-per-consumer (never shared). Publisher confirms for reliable publishing. Heartbeat and prefetch configuration."
---
# Connection Management

RabbitMQ connections are expensive to create and should be managed carefully. One connection per application, one channel per consumer/producer context.

## IRabbitMqConnection Singleton

A single connection is shared across the entire application. Channels are created from this connection as needed.

### Interface

```csharp
public interface IRabbitMqConnection : IAsyncDisposable
{
    /// <summary>
    /// Gets a new channel. Each consumer/producer should have its own channel.
    /// Channels are lightweight and meant to be used per-context.
    /// </summary>
    Task<IChannel> GetChannelAsync(CancellationToken ct = default);

    /// <summary>
    /// Whether the connection is currently open.
    /// </summary>
    bool IsConnected { get; }
}
```

### Implementation

```csharp
public sealed class RabbitMqConnection : IRabbitMqConnection
{
    private readonly ConnectionFactory _factory;
    private IConnection? _connection;
    private readonly SemaphoreSlim _lock = new(1, 1);
    private readonly ILogger<RabbitMqConnection> _logger;

    public RabbitMqConnection(string connectionString, ILogger<RabbitMqConnection> logger)
    {
        _logger = logger;
        _factory = new ConnectionFactory
        {
            Uri = new Uri(connectionString),

            // Auto-recovery: reconnect automatically on connection failure
            AutomaticRecoveryEnabled = true,
            NetworkRecoveryInterval = TimeSpan.FromSeconds(10),

            // Topology recovery: re-declare exchanges, queues, bindings after reconnect
            TopologyRecoveryEnabled = true,

            // Heartbeat: detect dead connections
            RequestedHeartbeat = TimeSpan.FromSeconds(30),

            // Client-provided name for identification in Management UI
            ClientProvidedName = Environment.GetEnvironmentVariable("SERVICE_NAME")
                                ?? "unknown-service",

            // Async consumer dispatch
            DispatchConsumersAsync = true
        };
    }

    public bool IsConnected => _connection is { IsOpen: true };

    public async Task<IChannel> GetChannelAsync(CancellationToken ct = default)
    {
        if (_connection is { IsOpen: true })
        {
            return await _connection.CreateChannelAsync(cancellationToken: ct);
        }

        await _lock.WaitAsync(ct);
        try
        {
            // Double-check after acquiring lock
            if (_connection is { IsOpen: true })
            {
                return await _connection.CreateChannelAsync(cancellationToken: ct);
            }

            _logger.LogInformation("Creating RabbitMQ connection...");
            _connection = await _factory.CreateConnectionAsync(ct);

            _connection.ConnectionShutdownAsync += (sender, args) =>
            {
                _logger.LogWarning("RabbitMQ connection shutdown: {Reason}", args.ReplyText);
                return Task.CompletedTask;
            };

            _connection.RecoverySucceededAsync += (sender, args) =>
            {
                _logger.LogInformation("RabbitMQ connection recovered");
                return Task.CompletedTask;
            };

            _connection.ConnectionRecoveryErrorAsync += (sender, args) =>
            {
                _logger.LogError(args.Exception, "RabbitMQ connection recovery failed");
                return Task.CompletedTask;
            };

            _logger.LogInformation("RabbitMQ connection established");
            return await _connection.CreateChannelAsync(cancellationToken: ct);
        }
        finally
        {
            _lock.Release();
        }
    }

    public async ValueTask DisposeAsync()
    {
        if (_connection is not null)
        {
            await _connection.CloseAsync();
            _connection.Dispose();
        }
        _lock.Dispose();
    }
}
```

### Registration

```csharp
// Program.cs — register as singleton
builder.Services.AddSingleton<IRabbitMqConnection>(sp =>
    new RabbitMqConnection(
        builder.Configuration.GetConnectionString("RabbitMq")!,
        sp.GetRequiredService<ILogger<RabbitMqConnection>>()));
```

## Connection String

Always from environment variables or configuration. Never hardcoded.

```json
// appsettings.json
{
  "ConnectionStrings": {
    "RabbitMq": "amqp://guest:guest@rabbitmq:5672/"
  }
}
```

```yaml
# docker-compose.yml
services:
  mailsender:
    environment:
      - ConnectionStrings__RabbitMq=amqp://guest:guest@rabbitmq:5672/
```

### MailSender service — package pins

The MailSender consumer host (`Microsoft.NET.Sdk.Worker`) sends transactional emails via SMTP using **MailKit**. Pin to the canonical minimum from [`software-project-team/dependency-versions.md`](../../../dependency-versions.md):

```xml
<!-- src/<App>.MailSender/<App>.MailSender.csproj -->
<PackageReference Include="MailKit"             Version="4.16.0" />
<PackageReference Include="RabbitMQ.Client"     Version="6.8.1" />
<PackageReference Include="StackExchange.Redis" Version="2.8.16" />
```

> **Security note — minimum 4.16.0 (medium CVE).** MailKit < 4.16.0 has a STARTTLS Response Injection vulnerability that enables SASL mechanism downgrade. Any new MailSender host MUST start at 4.16.0 or later. Discovered while scaffolding a downstream MailSender host; the bug is not project-specific. For the centralized pin and any future bumps, see `software-project-team/dependency-versions.md`.

### Connection String Format
```
amqp://{user}:{password}@{host}:{port}/{vhost}
```

| Field | Dev | Production |
|-------|-----|------------|
| user | guest | service-specific user |
| password | guest | strong password from secrets |
| host | rabbitmq (Docker service name) | hostname/IP |
| port | 5672 | 5672 (or 5671 for TLS) |
| vhost | / (default) | /app or /service-specific |

## Channel Management

### Rules

1. **One channel per consumer.** A consumer's channel is created at startup and kept open for the lifetime of the consumer.
2. **One channel per publish context.** For producers, create a channel per publish operation or use a channel pool.
3. **Never share channels across threads.** Channels are NOT thread-safe.
4. **Close channels explicitly** in `StopAsync` or `DisposeAsync`.

### Consumer Channel (Long-lived)

```csharp
public sealed class EmailConsumer : BackgroundService
{
    private IChannel? _channel;

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        // Create once, use for the lifetime of the consumer
        _channel = await _rmqConnection.GetChannelAsync(stoppingToken);

        // Declare topology, set QoS, start consuming...
    }

    public override async Task StopAsync(CancellationToken ct)
    {
        if (_channel is not null)
        {
            await _channel.CloseAsync();
            _channel.Dispose();
        }
        await base.StopAsync(ct);
    }
}
```

### Producer Channel (Short-lived or Pooled)

```csharp
public sealed class EmailPublisher
{
    private readonly IRabbitMqConnection _connection;

    public async Task PublishAsync(EmailMessage message, CancellationToken ct)
    {
        // Option 1: Channel per publish (simple, slightly more overhead)
        await using var channel = await _connection.GetChannelAsync(ct);
        // ... publish ...

        // Option 2: Reuse a channel (better performance, need thread safety)
        // Use a channel pool or dedicated publisher channel
    }
}
```

## Publisher Confirms

For reliable publishing, enable publisher confirms. The broker confirms that it has received and persisted the message.

```csharp
public async Task PublishWithConfirmAsync(byte[] body, BasicProperties props, CancellationToken ct)
{
    await using var channel = await _connection.GetChannelAsync(ct);

    // Enable publisher confirms on this channel
    await channel.ConfirmSelectAsync(ct);

    await channel.BasicPublishAsync(
        exchange: "emails.fanout",
        routingKey: "",
        mandatory: false,
        basicProperties: props,
        body: body,
        cancellationToken: ct);

    // Wait for broker confirmation
    await channel.WaitForConfirmsOrDieAsync(ct);
}
```

**When to use publisher confirms:**
- Critical messages (payments, user creation, important notifications)
- When you need to guarantee the message reached the broker

**When NOT to use:**
- High-throughput scenarios where occasional message loss is acceptable (logs)
- Fire-and-forget patterns where performance is more important than guaranteed delivery

## Prefetch Configuration

`BasicQos` controls how many unacknowledged messages the broker delivers to a consumer.

```csharp
// Process one message at a time
await channel.BasicQosAsync(prefetchSize: 0, prefetchCount: 1, global: false);
```

| Prefetch Count | Use Case |
|----------------|----------|
| 1 | Default — fair dispatch, predictable memory usage |
| 5-20 | Higher throughput when message processing is fast |
| 0 (unlimited) | NEVER — consumer can be overwhelmed |

**Rule:** Start with `prefetchCount: 1`. Only increase if you have measured that the consumer is underutilized and message processing is I/O-bound.

## Heartbeat

Heartbeats detect dead connections. If RabbitMQ does not receive a heartbeat within 2x the interval, it closes the connection.

```csharp
_factory = new ConnectionFactory
{
    RequestedHeartbeat = TimeSpan.FromSeconds(30),  // Send heartbeat every 30s
    // RabbitMQ closes connection if no heartbeat for 60s (2x interval)
};
```

| Environment | Heartbeat |
|-------------|-----------|
| Development | 60 seconds |
| Production | 30 seconds |
| Containers (short-lived) | 15 seconds |

## Health Check

Expose connection health for orchestration tools (Docker health check, Kubernetes readiness probe).

```csharp
public sealed class RabbitMqHealthCheck : IHealthCheck
{
    private readonly IRabbitMqConnection _connection;

    public RabbitMqHealthCheck(IRabbitMqConnection connection)
    {
        _connection = connection;
    }

    public Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context, CancellationToken ct = default)
    {
        return Task.FromResult(_connection.IsConnected
            ? HealthCheckResult.Healthy("RabbitMQ connection is open")
            : HealthCheckResult.Unhealthy("RabbitMQ connection is closed"));
    }
}

// Registration
builder.Services.AddHealthChecks()
    .AddCheck<RabbitMqHealthCheck>("rabbitmq");
```

## Rules

1. **One connection per application.** Connections are expensive (TCP + AMQP handshake). Reuse via singleton.
2. **AutoRecovery always enabled.** Network blips happen. Let the client library reconnect automatically.
3. **TopologyRecovery always enabled.** After reconnect, exchanges/queues/bindings are re-declared automatically.
4. **Never share channels across threads.** Channels are NOT thread-safe.
5. **Close channels explicitly.** Do not rely on garbage collection.
6. **DispatchConsumersAsync = true.** Required for `AsyncEventingBasicConsumer`. Without this, async consumers deadlock.
7. **ClientProvidedName for debugging.** Shows up in Management UI — essential for identifying which service holds which connection.
