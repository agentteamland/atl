---
knowledge-base-summary: "App updated, event payload changed. Old client doesn't understand new format. Backward-compatible event design, optional fields, version negotiation. Don't break existing clients when adding new fields."
---
# Event Versioning: Backward-Compatible Event Evolution

## The Problem

The Flutter app is on version 2.3. You deploy a backend change that adds new fields to the `order-confirmed` event. Now every user still on v2.2 receives a payload their code doesn't understand. If you remove a field, their app crashes. If you rename a field, their app silently ignores the data.

WebSocket events are a **public API contract** between backend and clients. Unlike REST endpoints where you can version the URL, WebSocket events are pushed to all connected clients regardless of their version. You must evolve events without breaking existing consumers.

---

## Three Rules of Event Evolution

### Rule 1: Never Remove Fields

Once a field is published in an event payload, it stays forever (or until the event itself is deprecated). Old clients depend on it.

```csharp
// v1 payload
{
    "orderId": "abc-123",
    "total": 49.99,
    "status": "confirmed"
}

// v2 payload -- WRONG: removed 'total'
{
    "orderId": "abc-123",
    "status": "confirmed",
    "summary": { "total": 49.99, "itemCount": 3 }
}

// v2 payload -- CORRECT: 'total' stays, new field added
{
    "orderId": "abc-123",
    "total": 49.99,
    "status": "confirmed",
    "summary": { "total": 49.99, "itemCount": 3 }
}
```

### Rule 2: New Fields Are Always Optional/Nullable

Old clients will not send or expect the new field. New clients must handle the field being absent (for events from old backend versions during rolling deploys).

```csharp
// Event record with evolution
public sealed record OrderConfirmedEvent
{
    public required string OrderId { get; init; }
    public required decimal Total { get; init; }
    public required string Status { get; init; }

    // v2: added later -- nullable, old clients ignore it
    public OrderSummary? Summary { get; init; }

    // v3: added later -- nullable, defaults to null
    public string? EstimatedDelivery { get; init; }
}
```

### Rule 3: If a Breaking Change Is Unavoidable, Create a New Event

When you must fundamentally restructure a payload (rename fields, change types, alter semantics), create a new event with a versioned name.

```csharp
// Old event -- still sent for backward compatibility
"order-confirmed"    → { orderId, total, status }

// New event -- sent alongside the old one during transition period
"order-confirmed-v2" → { orderId, pricing: { subtotal, tax, total }, status, items: [...] }
```

During the transition period, the backend sends BOTH events. Once all clients have migrated to v2, stop sending the old event.

---

## Payload Envelope Pattern

For events that may undergo significant evolution, wrap the data in an envelope with a version number:

```csharp
public sealed record EventEnvelope<T>
{
    /// <summary>
    /// Payload version. Increment on structural changes.
    /// Clients use this to pick the right deserializer.
    /// </summary>
    public required int Version { get; init; }

    /// <summary>
    /// The event name (redundant with SignalR event name but useful for logging).
    /// </summary>
    public required string Event { get; init; }

    /// <summary>
    /// The actual payload.
    /// </summary>
    public required T Data { get; init; }

    /// <summary>
    /// When the event was produced.
    /// </summary>
    public DateTimeOffset Timestamp { get; init; } = DateTimeOffset.UtcNow;
}
```

### Usage in Hub/Broadcast

```csharp
await hubContext.Clients
    .User(userId)
    .SendAsync("order-confirmed", new EventEnvelope<OrderConfirmedEvent>
    {
        Version = 1,
        Event = "order-confirmed",
        Data = new OrderConfirmedEvent
        {
            OrderId = "abc-123",
            Total = 49.99m,
            Status = "confirmed",
        },
    });
```

### Client-Side Envelope Handling (Flutter/Dart)

```dart
hub.on('order-confirmed', (envelope) {
  final version = envelope['version'] as int;
  final data = envelope['data'] as Map<String, dynamic>;

  switch (version) {
    case 1:
      handleOrderConfirmedV1(data);
      break;
    case 2:
      handleOrderConfirmedV2(data);
      break;
    default:
      // Unknown version -- log and use best effort
      log.warning('Unknown order-confirmed version: $version');
      handleOrderConfirmedV1(data); // Fall back to v1 handler
  }
});
```

---

## JSON Deserialization: Be Lenient

Clients MUST ignore unknown fields. This is the default behavior in most JSON libraries, but it must be explicitly confirmed and never overridden.

### C# (System.Text.Json) -- Backend

```csharp
// Default behavior: unknown properties are ignored during deserialization.
// Do NOT add [JsonExtensionData] or strict validation that rejects unknown fields.

var options = new JsonSerializerOptions
{
    PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
    // Do NOT set: UnmappedMemberHandling = JsonUnmappedMemberHandling.Disallow
};
```

### Dart (json_serializable) -- Client

```dart
// In model classes, unknown fields are ignored by default.
// Do NOT use @JsonSerializable(disallowUnrecognizedKeys: true)

@JsonSerializable()
class OrderConfirmedEvent {
  final String orderId;
  final double total;
  final String status;
  // New fields from v2 -- nullable, ignored if absent
  final OrderSummary? summary;

  OrderConfirmedEvent({
    required this.orderId,
    required this.total,
    required this.status,
    this.summary, // Optional -- old events won't have it
  });

  factory OrderConfirmedEvent.fromJson(Map<String, dynamic> json) =>
      _$OrderConfirmedEventFromJson(json);
}
```

---

## Event Catalog: Documentation as Contract

Maintain an event catalog that documents every event, its current version, and its payload. This is the single source of truth for the client-server contract.

### Format (in code or a shared doc)

```csharp
/// <summary>
/// EVENT CATALOG
/// =============
///
/// order-confirmed (v1) -- Since: 2026-01
///   Fired when an order is confirmed by the system.
///   Payload: { orderId: string, total: decimal, status: string }
///   Target: single user (order owner)
///
/// order-confirmed (v2) -- Since: 2026-04
///   Added: summary (optional), estimatedDelivery (optional)
///   Payload: { orderId, total, status, summary?: { subtotal, tax, total, itemCount }, estimatedDelivery?: string }
///   Target: single user (order owner)
///
/// user-typing (v1) -- Since: 2026-01, Ephemeral
///   Fired when a user is typing in a conversation.
///   Payload: { userId: string, conversationId: string, timestamp: DateTimeOffset }
///   Target: conversation group (excluding sender)
///   Note: NOT versioned with envelope (ephemeral, not worth the overhead)
///
/// [DEPRECATED] order-status-changed (v1) -- Since: 2025-12, Deprecated: 2026-03
///   Superseded by: order-confirmed, order-shipped, order-delivered (split into specific events)
///   Removal target: 2026-06
/// </summary>
```

### Deprecation Lifecycle

```
[Active]  →  [Deprecated]  →  [Removed]
              (still sent)     (no longer sent)
              mark in catalog   remove from code
              warn in logs      after N months
```

1. **Deprecate:** Mark the event as deprecated in the catalog. Add a log warning when it is sent. Continue sending it.
2. **Notify:** Publish a timeline for removal. Give clients at least 2 release cycles to migrate.
3. **Remove:** Stop sending the event. Remove the code. Old clients that still listen for it simply never receive it -- no crash, no error.

```csharp
// During deprecation period -- log a warning each time the deprecated event is sent
_logger.LogWarning(
    "Sending deprecated event 'order-status-changed'. " +
    "Clients should migrate to specific events. Removal target: 2026-06.");
```

---

## When to Use Each Strategy

| Change Type | Strategy | Example |
|-------------|----------|---------|
| Add a new field | Add as nullable, no version bump | Add `estimatedDelivery` to order event |
| Change field type | New event name with version suffix | `order-confirmed-v2` |
| Remove a field | Do NOT remove. Deprecate + keep sending | Mark `legacyStatus` as deprecated |
| Rename a field | Add new field, keep old field, deprecate old | Add `pricing`, keep `total` |
| Split event into multiple | Create new events, deprecate old | `order-status-changed` → `order-confirmed` + `order-shipped` |
| Restructure entire payload | Envelope version bump OR new event name | Bump envelope version to 2 |

---

## When NOT to Use Envelope Versioning

Not every event needs the full envelope pattern. Use it only for events that:
- Carry significant data payloads
- Are consumed by multiple client types (Flutter, React, third-party)
- Have a history of changing or are likely to change

Ephemeral signals (typing indicators, heartbeats, viewer counts) do NOT need envelope versioning. They are simple, stable, and if they change, all clients update within days.

---

## Rules

1. **Additive only.** Adding fields is always safe. Removing or renaming is always dangerous.
2. **New fields are always nullable.** Old serialized payloads won't have them. Old clients won't send them.
3. **Breaking change = new event name.** `order-confirmed-v2` is a new event. The old `order-confirmed` continues to exist until deprecated and removed.
4. **Clients ignore unknown fields.** This is enforced by convention and by using lenient JSON deserialization settings.
5. **Deprecation has a timeline.** Mark deprecated in catalog, log warnings, give clients N months, then remove.
6. **Event catalog is the contract.** Every event, its version, its payload shape, and its deprecation status must be documented. If it's not in the catalog, it doesn't exist.
7. **No envelope for ephemeral signals.** Typing, heartbeat, viewer count -- these are simple and stable. Envelope overhead is not worth it.
