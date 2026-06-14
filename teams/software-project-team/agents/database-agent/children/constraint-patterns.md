---
knowledge-base-summary: "Unique constraints (single and composite). Check constraints for data integrity. Default values. Not null enforcement. Global query filter for soft delete. Concurrency token (RowVersion/xmin). Code examples for each pattern."
---
# Constraint Patterns

## Unique Constraints

### Single Column Unique

```csharp
// Via unique index (preferred — more flexible, supports partial and conditional)
builder.HasIndex(x => x.Email)
    .HasDatabaseName("ix_users_email")
    .IsUnique();

// Via alternate key (creates a constraint, not an index — less flexible)
builder.HasAlternateKey(x => x.Email)
    .HasName("uq_users_email");
```

**Prefer unique index over alternate key.** Unique indexes support partial filters, covering columns, and are more commonly used in PostgreSQL.

### Composite Unique

```csharp
// Tenant + Slug must be unique together
builder.HasIndex(x => new { x.TenantId, x.Slug })
    .HasDatabaseName("ix_products_tenant_id_slug")
    .IsUnique();
```

### Unique with Soft Delete Filter

A common pattern: email must be unique among active records, but a deleted user's email can be reused.

```csharp
builder.HasIndex(x => x.Email)
    .HasDatabaseName("ix_users_email_active")
    .IsUnique()
    .HasFilter("is_deleted = false");
```

Without the filter, restoring a soft-deleted user or creating a new user with the same email would fail.

## Check Constraints

Enforce business rules at the database level. Defense-in-depth — the handler validates too, but the DB is the last line of defense.

```csharp
// In Configuration
builder.ToTable(t =>
{
    // Price must be positive
    t.HasCheckConstraint("ck_products_price_positive", "price > 0");

    // Quantity must be non-negative
    t.HasCheckConstraint("ck_order_items_quantity_positive", "quantity > 0");

    // Discount must be between 0 and 100
    t.HasCheckConstraint("ck_orders_discount_range", 
        "discount_percentage >= 0 AND discount_percentage <= 100");

    // End date must be after start date
    t.HasCheckConstraint("ck_events_date_range", 
        "end_date > start_date");

    // Status must be a known value
    t.HasCheckConstraint("ck_orders_status_valid",
        "status IN ('Pending', 'Confirmed', 'Processing', 'Shipped', 'Delivered', 'Cancelled')");
});
```

### Naming Convention

Format: `ck_{table}_{description}`

| Constraint | Name |
|-----------|------|
| Price > 0 | `ck_products_price_positive` |
| Quantity > 0 | `ck_order_items_quantity_positive` |
| Date range valid | `ck_events_date_range` |
| Status enum check | `ck_orders_status_valid` |

## Default Values

### Static Defaults

```csharp
builder.Property(x => x.IsActive)
    .IsRequired()
    .HasDefaultValue(true);

builder.Property(x => x.ViewCount)
    .IsRequired()
    .HasDefaultValue(0);

builder.Property(x => x.Role)
    .IsRequired()
    .HasDefaultValue("User")
    .HasMaxLength(50);
```

### Dynamic Defaults (SQL Expressions)

```csharp
builder.Property(x => x.Id)
    .HasDefaultValueSql("gen_random_uuid()");

builder.Property(x => x.CreatedAt)
    .HasDefaultValueSql("now()");

// PostgreSQL-specific: default to empty JSONB object
builder.Property(x => x.Metadata)
    .HasColumnType("jsonb")
    .HasDefaultValueSql("'{}'::jsonb");

// Default to empty array
builder.Property(x => x.Tags)
    .HasColumnType("text[]")
    .HasDefaultValueSql("'{}'::text[]");
```

## Not Null Enforcement

### Required Properties

```csharp
// Explicit required
builder.Property(x => x.Name)
    .IsRequired()
    .HasMaxLength(100);

// Required FK
builder.Property(x => x.CustomerId)
    .IsRequired();
```

### Nullable Properties

```csharp
// Optional — no IsRequired()
builder.Property(x => x.Description)
    .HasMaxLength(500);

// Nullable FK
builder.Property(x => x.AssignedToId); // no IsRequired = nullable
```

### Alignment with C# Nullable Reference Types

```csharp
// Domain entity should match DB constraints
public class Order : BaseEntity
{
    public string OrderNumber { get; set; } = default!;  // required (non-nullable)
    public string? Notes { get; set; }                    // optional (nullable)
    public Guid CustomerId { get; set; }                  // required FK (non-nullable Guid)
    public Guid? AssignedToId { get; set; }               // optional FK (nullable Guid)
}
```

## Soft Delete Global Query Filter

Every entity that implements `ISoftDeletable` MUST have the global query filter:

```csharp
// Interface
public interface ISoftDeletable
{
    bool IsDeleted { get; set; }
    DateTime? DeletedAt { get; set; }
    string? DeletedBy { get; set; }
}

// Configuration
builder.Property(x => x.IsDeleted)
    .IsRequired()
    .HasDefaultValue(false);

builder.Property(x => x.DeletedAt);

builder.Property(x => x.DeletedBy)
    .HasMaxLength(100);

// Global filter — automatically excludes deleted records from ALL queries
builder.HasQueryFilter(x => !x.IsDeleted);
```

### Bypassing the Filter

```csharp
// Admin access to see deleted records
var allOrders = await context.Orders
    .IgnoreQueryFilters()
    .ToListAsync();

// Recovery/restore
var deletedOrder = await context.Orders
    .IgnoreQueryFilters()
    .FirstOrDefaultAsync(x => x.Id == orderId && x.IsDeleted);
```

### Multi-Tenant + Soft Delete Combined Filter

```csharp
builder.HasQueryFilter(x => !x.IsDeleted && x.TenantId == _tenantId);
```

## Concurrency Token (Optimistic Locking)

### Option 1: PostgreSQL xmin (Recommended)

PostgreSQL's `xmin` system column changes on every row update. No extra column needed.

```csharp
// Domain entity
public class Order : BaseEntity
{
    public uint RowVersion { get; set; }
}

// Configuration
builder.Property(x => x.RowVersion)
    .HasColumnName("xmin")
    .HasColumnType("xid")
    .ValueGeneratedOnAddOrUpdate()
    .IsConcurrencyToken();
```

### Option 2: Manual RowVersion (byte[])

```csharp
// Domain entity
public class Order : BaseEntity
{
    public byte[] RowVersion { get; set; } = default!;
}

// Configuration
builder.Property(x => x.RowVersion)
    .IsRowVersion(); // auto-configured as concurrency token
```

### How It Works

```csharp
// Handler
var order = await context.Orders.FindAsync(orderId);
order.Status = newStatus;
order.UpdatedAt = DateTime.UtcNow;

try
{
    await context.SaveChangesAsync(ct);
}
catch (DbUpdateConcurrencyException)
{
    throw new ConflictException("Order was modified by another user. Please refresh and try again.");
}
```

EF Core automatically adds `WHERE xmin = @originalValue` to the UPDATE statement. If the row was modified since it was read, the WHERE doesn't match, zero rows are updated, and EF throws `DbUpdateConcurrencyException`.

## Constraint Patterns Checklist

For every new entity:

- [ ] Unique constraints on business-unique fields (email, slug, order number)
- [ ] Check constraints for value ranges (price > 0, percentage 0-100)
- [ ] Default values for fields that need them (IsActive, CreatedAt, empty JSONB)
- [ ] Not null on all required fields
- [ ] Nullable only on explicitly optional fields
- [ ] C# nullable annotations match DB constraints
- [ ] Soft delete filter if entity implements ISoftDeletable
- [ ] Concurrency token if entity supports concurrent updates
