---
knowledge-base-summary: "Two-layer change tracking. Layer 1: IAuditableEntity (CreatedBy, ModifiedBy) on every entity. Layer 2: AuditLog table — automatic change history via ChangeTracker (entity, field, old value, new value, who, when). No extra code in handlers — interceptor handles everything."
---
# Audit Trail: Two-Layer Change Tracking

## Layer 1: IAuditableEntity (Default on Every Entity)

Tracks the last state of each entity:

```csharp
public interface IAuditableEntity
{
    string? CreatedBy { get; set; }
    DateTime CreatedAt { get; set; }
    string? ModifiedBy { get; set; }
    DateTime? ModifiedAt { get; set; }
}
```

`AuditableEntityInterceptor` fills these fields automatically during `SaveChanges` — no extra code needed in the handler.

## Layer 2: AuditLog Table (Change History)

A separate record is written for each change. Answers the question "who changed what, when, and on which entity" after the fact.

### Entity

```csharp
public class AuditLog
{
    public Guid Id { get; set; }
    public string EntityName { get; set; } = null!;   // "Order", "Customer"
    public string EntityId { get; set; } = null!;      // entity's Id (as string)
    public string Action { get; set; } = null!;        // "Created" | "Modified" | "Deleted"
    public string? Changes { get; set; }               // JSON: changed fields
    public string? UserId { get; set; }                // who did it
    public string? UserEmail { get; set; }             // who did it (human-readable)
    public DateTime Timestamp { get; set; }
}
```

### Changes JSON Format

```json
[
  { "Field": "Status", "Old": "Draft", "New": "Confirmed" },
  { "Field": "Total", "Old": "150.00", "New": "175.00" }
]
```

For Created operations, `Old` is null. For Deleted operations, `New` is null.

### Automatic Collection via Interceptor

Automatically collected from `ChangeTracker` in the `SaveChanges` override:

```csharp
private List<AuditLog> CollectAuditEntries()
{
    var entries = new List<AuditLog>();

    foreach (var entry in ChangeTracker.Entries<IAuditableEntity>())
    {
        if (entry.State == EntityState.Unchanged)
            continue;

        var auditLog = new AuditLog
        {
            Id = Guid.NewGuid(),
            EntityName = entry.Entity.GetType().Name,
            EntityId = entry.Property("Id").CurrentValue?.ToString() ?? "",
            UserId = _currentUser?.UserId,
            UserEmail = _currentUser?.Email,
            Timestamp = DateTime.UtcNow,
        };

        switch (entry.State)
        {
            case EntityState.Added:
                auditLog.Action = "Created";
                auditLog.Changes = SerializeProperties(
                    entry.Properties
                        .Where(p => p.CurrentValue != null)
                        .Select(p => new
                        {
                            Field = p.Metadata.Name,
                            Old = (string?)null,
                            New = p.CurrentValue?.ToString(),
                        }));
                break;

            case EntityState.Modified:
                auditLog.Action = "Modified";
                auditLog.Changes = SerializeProperties(
                    entry.Properties
                        .Where(p => p.IsModified)
                        .Select(p => new
                        {
                            Field = p.Metadata.Name,
                            Old = p.OriginalValue?.ToString(),
                            New = p.CurrentValue?.ToString(),
                        }));
                break;

            case EntityState.Deleted:
                auditLog.Action = "Deleted";
                break;
        }

        entries.Add(auditLog);
    }

    return entries;
}

private static string SerializeProperties(IEnumerable<object> changes)
{
    return JsonSerializer.Serialize(changes);
}
```

### SaveChanges Flow

```csharp
public override async Task<int> SaveChangesAsync(CancellationToken ct = default)
{
    // 1. Collect audit logs (BEFORE — because ChangeTracker resets after SaveChanges)
    var auditEntries = CollectAuditEntries();

    // 2. Fill auditable fields (CreatedBy, ModifiedBy)
    UpdateAuditableFields();

    // 3. Soft delete conversion
    ConvertSoftDeletes();

    // 4. Actual save
    var result = await base.SaveChangesAsync(ct);

    // 5. Save audit logs (separate SaveChanges — doesn't block the main operation)
    if (auditEntries.Count > 0)
    {
        AuditLogs.AddRange(auditEntries);
        await base.SaveChangesAsync(ct);
    }

    return result;
}
```

### No Extra Code Needed in Handler

The audit trail works entirely at the interceptor/DbContext level. The handler does normal CRUD — audit is automatic:

```csharp
// In the handler:
var order = await _db.Orders.FindAsync(request.OrderId, ct)
    ?? throw new NotFoundException(nameof(Order), request.OrderId);

order.Status = OrderStatus.Confirmed;  // just this
await _db.SaveChangesAsync(ct);        // audit log is created automatically

// In the AuditLog table:
// EntityName: "Order"
// EntityId: "abc-123"
// Action: "Modified"
// Changes: [{"Field":"Status","Old":"Draft","New":"Confirmed"}]
// UserId: "user-456"
// Timestamp: 2026-04-12T10:30:00Z
```

### Querying Audit Logs

To view the history of an entity:

```csharp
// In the query handler:
var history = await _db.AuditLogs
    .Where(a => a.EntityName == "Order" && a.EntityId == orderId.ToString())
    .OrderByDescending(a => a.Timestamp)
    .ToListAsync(ct);
```

### Sensitive Field Protection

Sensitive fields are masked when writing to audit logs:

```csharp
private static readonly HashSet<string> SensitiveFields = new(StringComparer.OrdinalIgnoreCase)
{
    "PasswordHash", "Password", "Secret", "Token",
    "ApiKey", "CreditCard", "Cvv", "SecretHash"
};

// Inside CollectAuditEntries:
var value = SensitiveFields.Contains(p.Metadata.Name)
    ? "***REDACTED***"
    : p.CurrentValue?.ToString();
```

### Bulk Cleanup

Audit logs grow over time. Periodic cleanup can be done via a Worker job:
- Records older than 90 days or 1 year are deleted (determined per project)
- This topic will be addressed separately when the need arises

