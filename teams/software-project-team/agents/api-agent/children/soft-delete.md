---
knowledge-base-summary: "No hard deletes. `ISoftDeletable` interface (IsDeleted, DeletedAt, DeletedBy) + global query filter + SaveChanges interceptor that converts `Delete` to `Modified`. Handler calls `Remove()` normally — interceptor handles the rest. `IgnoreQueryFilters()` for admin/recovery access. Physical cleanup via separate Worker job."
---
# Soft Delete Strategy

## Rule

Physical deletion (hard delete) is not performed. All entities use soft delete: `IsDeleted` flag + `DeletedAt` timestamp. Deleted records stay in the DB and are automatically filtered out in queries.

## Entity Structure

Added to `BaseEntity` (or the `ISoftDeletable` interface):

```csharp
public interface ISoftDeletable
{
    bool IsDeleted { get; set; }
    DateTime? DeletedAt { get; set; }
    string? DeletedBy { get; set; }
}
```

## Global Query Filter

Automatic filter for all `ISoftDeletable` entities inside `ApplicationDbContext.OnModelCreating`:

```csharp
protected override void OnModelCreating(ModelBuilder modelBuilder)
{
    // Apply global filter to all ISoftDeletable entities
    foreach (var entityType in modelBuilder.Model.GetEntityTypes())
    {
        if (typeof(ISoftDeletable).IsAssignableFrom(entityType.ClrType))
        {
            modelBuilder.Entity(entityType.ClrType)
                .HasQueryFilter(
                    BuildSoftDeleteFilter(entityType.ClrType));
        }
    }
}

// Lambda expression builder for generic filter
private static LambdaExpression BuildSoftDeleteFilter(Type type)
{
    var parameter = Expression.Parameter(type, "e");
    var property = Expression.Property(parameter, nameof(ISoftDeletable.IsDeleted));
    var condition = Expression.Equal(property, Expression.Constant(false));
    return Expression.Lambda(condition, parameter);
}
```

## SaveChanges Interceptor

The `Delete` operation is converted to soft delete in the interceptor:

```csharp
// Inside DbContext.SaveChanges override or interceptor:
var deletedEntries = ChangeTracker.Entries<ISoftDeletable>()
    .Where(e => e.State == EntityState.Deleted);

foreach (var entry in deletedEntries)
{
    entry.State = EntityState.Modified;
    entry.Entity.IsDeleted = true;
    entry.Entity.DeletedAt = DateTime.UtcNow;
    entry.Entity.DeletedBy = _currentUser?.UserId;
}
```

## Accessing Deleted Records

In rare cases where you need to see deleted records (admin panel, recovery):

```csharp
// Bypass the global filter
var allOrders = await _db.Orders
    .IgnoreQueryFilters()
    .Where(o => o.IsDeleted)
    .ToListAsync(ct);
```

## Deletion in Handler

Normal `Remove` is called in the handler — the interceptor automatically converts it to soft delete:

```csharp
// In the handler:
var order = await _db.Orders.FindAsync(request.OrderId, ct)
    ?? throw new NotFoundException(nameof(Order), request.OrderId);

_db.Orders.Remove(order);  // interceptor converts to soft delete
await _db.SaveChangesAsync(ct);

_logger.LogInformation("Order {OrderId} soft-deleted by {UserId}",
    order.Id, _currentUser.UserId);
```

## Bulk Cleanup

Physical cleanup of soft-deleted records is done through a separate mechanism (Worker job, admin endpoint). This topic is currently out of scope — it will be addressed when the need arises.

## Index Recommendation

Add a composite index on the soft delete field — for query filter performance:

```csharp
builder.HasIndex(e => e.IsDeleted)
    .HasFilter("\"IsDeleted\" = false");  // partial index, only active records
```

