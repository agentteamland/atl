---
knowledge-base-summary: "Primary blueprint for designing new tables and entities. Contains the IEntityTypeConfiguration template, complete checklist (PK, indexes, FK cascade, string lengths, required/optional, unique constraints, audit fields), and code examples for every configuration scenario."
---
# Schema Design Blueprint

This is the primary blueprint for designing new tables and entities. Every new entity follows this template and checklist.

## IEntityTypeConfiguration Template

```csharp
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;

namespace ProjectName.Infrastructure.Persistence.Configurations;

public class OrderConfiguration : IEntityTypeConfiguration<Order>
{
    public void Configure(EntityTypeBuilder<Order> builder)
    {
        // ── Table ──────────────────────────────────────────
        builder.ToTable("orders"); // explicit table name (snake_case handled by convention)

        // ── Primary Key ────────────────────────────────────
        builder.HasKey(x => x.Id);
        builder.Property(x => x.Id)
            .HasDefaultValueSql("gen_random_uuid()");

        // ── Properties ─────────────────────────────────────
        builder.Property(x => x.OrderNumber)
            .IsRequired()
            .HasMaxLength(50);

        builder.Property(x => x.Description)
            .HasMaxLength(500); // optional — no IsRequired()

        builder.Property(x => x.TotalAmount)
            .HasPrecision(18, 2); // decimal precision for money

        builder.Property(x => x.Status)
            .IsRequired()
            .HasConversion<string>() // enum stored as string
            .HasMaxLength(50);

        // ── Audit Fields ───────────────────────────────────
        builder.Property(x => x.CreatedAt)
            .IsRequired()
            .HasDefaultValueSql("now()");

        builder.Property(x => x.UpdatedAt);

        builder.Property(x => x.CreatedBy)
            .IsRequired()
            .HasMaxLength(100);

        builder.Property(x => x.ModifiedBy)
            .HasMaxLength(100);

        // ── Relationships ──────────────────────────────────
        builder.HasOne(x => x.Customer)
            .WithMany(x => x.Orders)
            .HasForeignKey(x => x.CustomerId)
            .OnDelete(DeleteBehavior.Restrict); // ALWAYS explicit

        // ── Indexes ────────────────────────────────────────
        // Query: "get orders by customer" — used in customer detail page
        builder.HasIndex(x => x.CustomerId)
            .HasDatabaseName("ix_orders_customer_id");

        // Query: "list orders by date" — used in order listing with pagination
        builder.HasIndex(x => x.CreatedAt)
            .HasDatabaseName("ix_orders_created_at")
            .IsDescending();

        // Query: "find order by number" — unique lookup
        builder.HasIndex(x => x.OrderNumber)
            .HasDatabaseName("ix_orders_order_number")
            .IsUnique();

        // ── Unique Constraints ─────────────────────────────
        // (covered by unique index above)

        // ── Query Filters ──────────────────────────────────
        builder.HasQueryFilter(x => !x.IsDeleted); // soft delete filter
    }
}
```

## New Entity Checklist

Run through this checklist for EVERY new entity configuration. Do not skip items — mark each as done.

### Primary Key
- [ ] PK defined explicitly via `HasKey()`
- [ ] UUID with `HasDefaultValueSql("gen_random_uuid()")` (preferred over auto-increment)

### Properties
- [ ] Every `string` property has `HasMaxLength()` — no unbounded strings
- [ ] Required properties marked with `IsRequired()`
- [ ] Optional properties left without `IsRequired()` (C# nullable reference types should match)
- [ ] `decimal` properties have `HasPrecision(precision, scale)`
- [ ] Enum properties use `HasConversion<string>()` with `HasMaxLength()`

### Relationships
- [ ] Every FK has explicit `OnDelete(DeleteBehavior.X)` — NEVER rely on default
- [ ] Navigation properties configured (both sides if bidirectional)
- [ ] FK property defined on the dependent entity (`CustomerId` on `Order`)

### Indexes
- [ ] Index on every foreign key column (mandatory)
- [ ] Index for every known query pattern (list, search, filter)
- [ ] Unique index for natural keys (email, order number, slug)
- [ ] Index names follow `ix_{table}_{columns}` convention
- [ ] Comment above each index explaining which query it supports

### Audit Fields
- [ ] `CreatedAt` (timestamptz, required, default `now()`)
- [ ] `UpdatedAt` (timestamptz, nullable)
- [ ] `CreatedBy` (string, required if auditable)
- [ ] `ModifiedBy` (string, nullable)

### Soft Delete (if applicable)
- [ ] `IsDeleted` (bool, default false)
- [ ] `DeletedAt` (timestamptz, nullable)
- [ ] `DeletedBy` (string, nullable)
- [ ] Global query filter: `HasQueryFilter(x => !x.IsDeleted)`

### Constraints
- [ ] Unique constraints for business-unique fields
- [ ] Check constraints for value ranges (if applicable)
- [ ] Default values for fields that need them

### Concurrency
- [ ] `RowVersion` / `xmin` configured if entity supports concurrent updates

## Configuration Registration

Every configuration is auto-discovered via `ApplyConfigurationsFromAssembly`:

```csharp
// In ApplicationDbContext.OnModelCreating
protected override void OnModelCreating(ModelBuilder modelBuilder)
{
    modelBuilder.ApplyConfigurationsFromAssembly(
        typeof(ApplicationDbContext).Assembly);

    base.OnModelCreating(modelBuilder);
}
```

No manual registration needed — just create the file in `Configurations/` and it is picked up.

## File Location

```
src/{ProjectName}.Infrastructure/
  Persistence/
    Configurations/
      OrderConfiguration.cs          ← one file per entity
      CustomerConfiguration.cs
      OrderItemConfiguration.cs
    ApplicationDbContext.cs
    Migrations/
      ...
```

## Naming Convention

- File: `{Entity}Configuration.cs`
- Class: `{Entity}Configuration`
- Namespace: `{ProjectName}.Infrastructure.Persistence.Configurations`

## When to Create a Configuration

- **New entity added to Domain** → always create a Configuration
- **Existing entity changed** (new property, new relationship) → update its Configuration
- **Value object** → configure as owned type inside the parent entity's Configuration
- **Join entity** (many-to-many) → gets its own Configuration

## Common Pitfalls

1. **Forgetting MaxLength on strings** — PostgreSQL defaults to `text` (unlimited). While valid, it prevents client-side validation alignment and can mask data issues.

2. **Missing FK index** — EF Core does NOT create indexes on FK columns by default in PostgreSQL. Always add them explicitly.

3. **Relying on cascade delete defaults** — EF Core defaults vary by relationship type. Be explicit every time.

4. **No query filter for soft delete** — If the entity implements `ISoftDeletable`, the global filter MUST be added in the Configuration, not in every query.

5. **Enum stored as int** — Use `HasConversion<string>()` so the database is human-readable and migration-safe when enum order changes.
