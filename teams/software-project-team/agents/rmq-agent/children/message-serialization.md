---
knowledge-base-summary: "JSON serialization with System.Text.Json. Message envelope format: `{ type, version, data, timestamp, correlationId }`. Versioning strategy: additive changes only, new message type for breaking changes. Content-type and encoding headers."
---
# Message Serialization

Consistent message format ensures producers and consumers speak the same language, and messages remain compatible as the system evolves.

## JSON Serialization

Use `System.Text.Json` — it is built-in, fast, and source-generator compatible.

### Serializer Configuration

```csharp
public static class RmqJsonOptions
{
    public static readonly JsonSerializerOptions Default = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        PropertyNameCaseInsensitive = true,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull,
        WriteIndented = false,   // Compact for network transfer
        Converters =
        {
            new JsonStringEnumConverter(JsonNamingPolicy.CamelCase)
        }
    };
}
```

Use the same options on both producer and consumer. Register as a singleton or static readonly field.

## Message Envelope

Every message is wrapped in a standard envelope. Never publish raw data — always wrap it.

### Envelope Structure

```csharp
public sealed record MessageEnvelope<T>
{
    /// <summary>
    /// Message type identifier (e.g., "email.send", "log.entry").
    /// Used by consumers to determine how to deserialize Data.
    /// </summary>
    public required string Type { get; init; }

    /// <summary>
    /// Schema version. Starts at 1. Incremented on additive changes.
    /// </summary>
    public required int Version { get; init; }

    /// <summary>
    /// The actual message payload.
    /// </summary>
    public required T Data { get; init; }

    /// <summary>
    /// When the message was created (UTC).
    /// </summary>
    public required DateTimeOffset Timestamp { get; init; }

    /// <summary>
    /// Correlation ID for tracing a message through the system.
    /// </summary>
    public required string CorrelationId { get; init; }
}
```

### Usage

```csharp
// Producer
var envelope = new MessageEnvelope<EmailMessage>
{
    Type = "email.send",
    Version = 1,
    Data = new EmailMessage
    {
        To = "user@example.com",
        Subject = "Welcome",
        Body = "Hello!"
    },
    Timestamp = DateTimeOffset.UtcNow,
    CorrelationId = Guid.NewGuid().ToString()
};

var body = JsonSerializer.SerializeToUtf8Bytes(envelope, RmqJsonOptions.Default);

// Consumer
var json = Encoding.UTF8.GetString(ea.Body.Span);
var envelope = JsonSerializer.Deserialize<MessageEnvelope<EmailMessage>>(json, RmqJsonOptions.Default);
```

## Message Properties (AMQP Headers)

Set these properties on every published message:

```csharp
var props = new BasicProperties
{
    // Required
    ContentType = "application/json",
    ContentEncoding = "utf-8",
    DeliveryMode = DeliveryModes.Persistent,  // Survives RabbitMQ restart
    MessageId = Guid.NewGuid().ToString(),     // Unique ID for idempotency

    // Recommended
    Timestamp = new AmqpTimestamp(DateTimeOffset.UtcNow.ToUnixTimeSeconds()),
    AppId = "example-app-api",                 // Identifies the publisher
    Type = "email.send",                       // Message type (mirrors envelope)

    // Optional
    CorrelationId = correlationId,             // For request-reply or tracing
    ReplyTo = null,                            // For request-reply pattern
    Expiration = null                          // Message-level TTL (ms as string)
};
```

### Property Checklist

| Property | Required | Purpose |
|----------|----------|---------|
| ContentType | Yes | `application/json` |
| ContentEncoding | Yes | `utf-8` |
| DeliveryMode | Yes | `Persistent` (always) |
| MessageId | Yes | Idempotency key |
| Timestamp | Yes | When message was created |
| AppId | Recommended | Which service published |
| Type | Recommended | Message type for routing/deserialization |
| CorrelationId | Recommended | End-to-end tracing |

## Versioning Strategy

### Rule: Additive Changes Only

Once a message schema is published, it can only be extended, never modified.

**Allowed changes (increment version):**
- Add a new optional field
- Add a new enum value

**Breaking changes (require a new message type):**
- Rename a field
- Change a field type
- Remove a field
- Change the meaning of a field

### Example: Adding a Field

```csharp
// Version 1
public sealed record EmailMessage
{
    public required string To { get; init; }
    public required string Subject { get; init; }
    public required string Body { get; init; }
}

// Version 2 — added optional CC field (additive, non-breaking)
public sealed record EmailMessage
{
    public required string To { get; init; }
    public required string Subject { get; init; }
    public required string Body { get; init; }
    public string? Cc { get; init; }           // New optional field
    public string? ReplyTo { get; init; }      // Another new optional field
}
```

The consumer receiving a v1 message ignores unknown fields (`PropertyNameCaseInsensitive = true` + `DefaultIgnoreCondition.WhenWritingNull`). A v2 consumer can handle both v1 and v2 messages.

### Example: Breaking Change

```csharp
// WRONG — renaming "Body" to "HtmlBody" breaks existing consumers
public sealed record EmailMessage
{
    public required string To { get; init; }
    public required string Subject { get; init; }
    public required string HtmlBody { get; init; }  // BREAKING!
}

// CORRECT — create a new message type
// Old: "email.send" (version 1) with Body
// New: "email.send-html" (version 1) with HtmlBody
```

### Version-Based Deserialization

```csharp
// Consumer checks version before processing
var envelope = JsonSerializer.Deserialize<MessageEnvelope<JsonElement>>(json, RmqJsonOptions.Default);

var data = envelope.Type switch
{
    "email.send" when envelope.Version == 1 =>
        envelope.Data.Deserialize<EmailMessageV1>(RmqJsonOptions.Default),
    "email.send" when envelope.Version >= 2 =>
        envelope.Data.Deserialize<EmailMessageV2>(RmqJsonOptions.Default),
    _ => throw new UnknownMessageTypeException(envelope.Type, envelope.Version)
};
```

## Encoding

All messages use UTF-8 encoding. This is set in both the AMQP properties and the serialization.

```csharp
// Serialization (producer)
var body = JsonSerializer.SerializeToUtf8Bytes(envelope, RmqJsonOptions.Default);
// SerializeToUtf8Bytes returns a ReadOnlyMemory<byte> in UTF-8

// Deserialization (consumer)
var json = Encoding.UTF8.GetString(ea.Body.Span);
var envelope = JsonSerializer.Deserialize<MessageEnvelope<T>>(json, RmqJsonOptions.Default);
```

## Helper Methods

```csharp
public static class RmqMessageHelper
{
    public static (byte[] Body, BasicProperties Props) CreateMessage<T>(
        string type, T data, string? correlationId = null)
    {
        var envelope = new MessageEnvelope<T>
        {
            Type = type,
            Version = 1,
            Data = data,
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = correlationId ?? Guid.NewGuid().ToString()
        };

        var body = JsonSerializer.SerializeToUtf8Bytes(envelope, RmqJsonOptions.Default);

        var props = new BasicProperties
        {
            ContentType = "application/json",
            ContentEncoding = "utf-8",
            DeliveryMode = DeliveryModes.Persistent,
            MessageId = Guid.NewGuid().ToString(),
            Timestamp = new AmqpTimestamp(DateTimeOffset.UtcNow.ToUnixTimeSeconds()),
            Type = type,
            CorrelationId = envelope.CorrelationId
        };

        return (body, props);
    }

    public static MessageEnvelope<T>? DeserializeEnvelope<T>(ReadOnlyMemory<byte> body)
    {
        var json = Encoding.UTF8.GetString(body.Span);
        return JsonSerializer.Deserialize<MessageEnvelope<T>>(json, RmqJsonOptions.Default);
    }
}
```

## Rules

1. **Always use the envelope.** Never publish raw data without Type, Version, Timestamp, CorrelationId.
2. **Always set ContentType and ContentEncoding** in AMQP properties.
3. **Always use `DeliveryMode.Persistent`.** Transient messages are lost on RabbitMQ restart.
4. **Additive changes only.** New fields must be optional. Renaming or removing fields requires a new message type.
5. **Same JsonSerializerOptions on both sides.** Mismatched options (e.g., different naming policy) cause deserialization failures.
6. **MessageId is mandatory.** It is the idempotency key. Without it, deduplication is impossible.
7. **CorrelationId for tracing.** Pass it through the entire pipeline so logs can be correlated end-to-end.
