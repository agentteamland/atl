---
knowledge-base-summary: "Optimistic locking via `RowVersion` property + `IsConcurrencyToken()`. EF Core adds `WHERE RowVersion = @original` automatically. Handler needs no extra code. `DbUpdateConcurrencyException` → 409 Conflict. Client sends RowVersion in update requests, gets new version back."
---
# Concurrency Handling: Optimistic Locking

## Problem

Two users update the same entity at the same time. The last writer wins, and the first user's changes are silently lost. This is unacceptable.

## Solution: Optimistic Concurrency

A `RowVersion` field is kept on the entity. EF Core checks during `SaveChanges` whether "the version I read is still the same." If it has changed, it throws `DbUpdateConcurrencyException`.

```
User A: read product (RowVersion=1)
User B: read product (RowVersion=1)
User A: change price → save → RowVersion=2 ✅
User B: change stock → save → expected RowVersion=1 but found 2 → CONFLICT ✅
```

**Why not Pessimistic (locking):**
- DB row lock → deadlock risk, scaling issues
- User opens form and thinks for 10 min → lock is held for 10 min
- Managing DB locks in a distributed system is a nightmare

## Adding RowVersion to BaseEntity

```csharp
public abstract class BaseEntity
{
    public Guid Id { get; set; }
    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;
    public DateTime? UpdatedAt { get; set; }
    public uint RowVersion { get; set; }  // concurrency token
}
```

## EF Core Configuration

In PostgreSQL the `xmin` system column can be used, but an explicit `RowVersion` is clearer and more portable:

```csharp
// In base entity configuration:
public class BaseEntityConfiguration<T> : IEntityTypeConfiguration<T>
    where T : BaseEntity
{
    public virtual void Configure(EntityTypeBuilder<T> builder)
    {
        builder.HasKey(e => e.Id);
        builder.Property(e => e.Id).HasDefaultValueSql("gen_random_uuid()");
        builder.Property(e => e.RowVersion).IsConcurrencyToken();
    }
}
```

`IsConcurrencyToken()` → EF Core adds `WHERE "RowVersion" = @originalVersion` to every UPDATE query. If the row is not updated (version has changed), it throws an exception.

## Automatic RowVersion Increment

In the `SaveChanges` override or interceptor:

```csharp
// Inside AppDbContext.SaveChanges:
var modifiedEntries = ChangeTracker.Entries<BaseEntity>()
    .Where(e => e.State == EntityState.Modified);

foreach (var entry in modifiedEntries)
{
    entry.Entity.RowVersion++;
}
```

## Catching in Global Exception Handler

`DbUpdateConcurrencyException` → `409 Conflict`:

```csharp
// Added inside Api/Infrastructure/GlobalExceptionHandler.cs:
DbUpdateConcurrencyException => (
    StatusCodes.Status409Conflict,
    "Conflict",
    "This record has been modified by another user. Please reload and try again."
),
```

## No Extra Code Needed in Handler

The handler does normal CRUD — concurrency control is entirely at the EF Core + interceptor level:

```csharp
public async Task<UpdateProductResponse> Handle(
    UpdateProductCommand request, CancellationToken ct)
{
    var product = await _db.Products.FindAsync(request.ProductId, ct)
        ?? throw new NotFoundException(nameof(Product), request.ProductId);

    _logger.LogInformation(
        "Updating product {ProductId}, current RowVersion {Version}",
        product.Id, product.RowVersion);

    product.Name = request.Name;
    product.Price = request.Price;

    // SaveChanges → RowVersion++ (interceptor)
    // → UPDATE ... WHERE Id = @id AND RowVersion = @originalVersion
    // → If version has changed, DbUpdateConcurrencyException
    await _db.SaveChangesAsync(ct);

    _logger.LogInformation(
        "Product {ProductId} updated, new RowVersion {Version}",
        product.Id, product.RowVersion);

    return new UpdateProductResponse(product.Id, product.RowVersion);
}
```

## Client Side

### Sending RowVersion in the Request

The client receives `RowVersion` when reading the entity and sends it back in the update request:

```csharp
// Update command:
record UpdateProductCommand(
    Guid ProductId,
    string Name,
    decimal Price,
    uint RowVersion  // the version the client knows
) : IRequest<UpdateProductResponse>;
```

### Version Check in Handler

```csharp
var product = await _db.Products.FindAsync(request.ProductId, ct)
    ?? throw new NotFoundException(nameof(Product), request.ProductId);

// The version sent by the client must match the version in the DB
if (product.RowVersion != request.RowVersion)
{
    _logger.LogWarning(
        "Concurrency conflict for product {ProductId}: "
        + "client version {ClientVersion}, db version {DbVersion}",
        product.Id, request.RowVersion, product.RowVersion);
    throw new ConcurrencyException(nameof(Product), product.Id);
}
```

### 409 Handling in Flutter/React

```dart
// Flutter:
try {
  await dio.put('/api/products/$id', data: updateData);
} on DioException catch (e) {
  if (e.response?.statusCode == 409) {
    // "This record has been modified by someone else. Please reload."
    showConflictDialog();
    await reloadProduct();
  }
}
```

## RowVersion in DTOs

`RowVersion` is always present in DTOs of entities that can be updated:

```csharp
// In the response:
record ProductDto(
    Guid Id,
    string Name,
    decimal Price,
    uint RowVersion  // client stores this and sends it back on update
);

// In the update response:
record UpdateProductResponse(
    Guid Id,
    uint RowVersion  // new version, client updates its copy
);
```

## ConcurrencyException (Application layer)

```csharp
public class ConcurrencyException : Exception
{
    public ConcurrencyException(string entityName, object entityId)
        : base($"{entityName} with id {entityId} has been modified by another user")
    {
        EntityName = entityName;
        EntityId = entityId;
    }

    public string EntityName { get; }
    public object EntityId { get; }
}
```

In the global exception handler:
```csharp
ConcurrencyException → 409 + ProblemDetails
DbUpdateConcurrencyException → 409 + ProblemDetails
```

