---
knowledge-base-summary: "Table names: PascalCase plural mapped to snake_case. Column names: PascalCase mapped to snake_case. Index: IX_{Table}_{Columns}. FK: FK_{Table}_{ReferencedTable}. Schema usage patterns. EF snake_case convention setup."
---
# Naming Conventions

## The Dual Convention

C# uses PascalCase. PostgreSQL uses snake_case. EF Core handles the mapping automatically via `UseSnakeCaseNamingConvention()`.

### Setup (Once in DbContext)

```csharp
// In ApplicationDbContext or via options configuration
protected override void OnConfiguring(DbContextOptionsBuilder optionsBuilder)
{
    optionsBuilder.UseNpgsql(connectionString)
        .UseSnakeCaseNamingConvention(); // from Npgsql.EntityFrameworkCore.PostgreSQL
}
```

With this convention, the C# property `FirstName` becomes the PostgreSQL column `first_name` automatically.

## Table Names

| C# Entity | PostgreSQL Table | Rule |
|-----------|-----------------|------|
| `User` | `users` | PascalCase singular in C#, snake_case plural in DB |
| `Order` | `orders` | |
| `OrderItem` | `order_items` | |
| `AuditLog` | `audit_logs` | |
| `UserProfile` | `user_profiles` | |

**Convention:** Entity names are singular in C# (one `Order`), table names are plural in PostgreSQL (many `orders`).

EF Core pluralizes automatically with `UseSnakeCaseNamingConvention`, but for explicitness:

```csharp
builder.ToTable("orders");
```

## Column Names

| C# Property | PostgreSQL Column | Notes |
|-------------|------------------|-------|
| `Id` | `id` | |
| `FirstName` | `first_name` | Auto-mapped by convention |
| `CreatedAt` | `created_at` | |
| `IsDeleted` | `is_deleted` | |
| `CustomerId` | `customer_id` | FK column |
| `OrderNumber` | `order_number` | |

## Index Names

Format: `ix_{table}_{columns}`

| C# Configuration | PostgreSQL Index Name |
|-------------------|---------------------|
| Index on `CreatedAt` in `Orders` | `ix_orders_created_at` |
| Composite index on `TenantId, CreatedAt` | `ix_orders_tenant_id_created_at` |
| Unique index on `Email` in `Users` | `ix_users_email` |
| Partial index on `Status` (pending only) | `ix_orders_status_pending` |

```csharp
builder.HasIndex(x => x.CreatedAt)
    .HasDatabaseName("ix_orders_created_at");

builder.HasIndex(x => new { x.TenantId, x.CreatedAt })
    .HasDatabaseName("ix_orders_tenant_id_created_at");
```

**Always use `HasDatabaseName()`.** Without it, EF Core generates names like `IX_Orders_CreatedAt` which doesn't match PostgreSQL conventions.

## Foreign Key Constraint Names

Format: `fk_{table}_{referenced_table}`

| Relationship | Constraint Name |
|-------------|----------------|
| `Order.CustomerId → Customer.Id` | `fk_orders_customers` |
| `OrderItem.OrderId → Order.Id` | `fk_order_items_orders` |
| `OrderItem.ProductId → Product.Id` | `fk_order_items_products` |

```csharp
builder.HasOne(x => x.Customer)
    .WithMany(x => x.Orders)
    .HasForeignKey(x => x.CustomerId)
    .HasConstraintName("fk_orders_customers")
    .OnDelete(DeleteBehavior.Restrict);
```

## Primary Key Constraint Names

Format: `pk_{table}`

EF Core + snake_case convention handles this automatically, but for explicitness:

```csharp
builder.HasKey(x => x.Id)
    .HasName("pk_orders");
```

## Unique Constraint Names

Format: `uq_{table}_{columns}`

```csharp
builder.HasAlternateKey(x => x.Email)
    .HasName("uq_users_email");

// Or via unique index (preferred — more flexible)
builder.HasIndex(x => x.Email)
    .HasDatabaseName("ix_users_email")
    .IsUnique();
```

## Schema Usage

Default schema is `public`. Use separate schemas for logical grouping in large projects:

```csharp
// In OnModelCreating
modelBuilder.HasDefaultSchema("public");

// Per-entity schema override
builder.ToTable("audit_logs", "audit");
builder.ToTable("email_templates", "messaging");
```

| Schema | Purpose |
|--------|---------|
| `public` | Core business entities (default) |
| `audit` | Audit trail tables |
| `auth` | Authentication/authorization tables |
| `messaging` | Email templates, notification logs |

## File Naming

| Type | Pattern | Example |
|------|---------|---------|
| Entity Configuration | `{Entity}Configuration.cs` | `OrderConfiguration.cs` |
| Migration | Auto-generated timestamp + description | `20240115_AddOrders.cs` |
| DbContext | `ApplicationDbContext.cs` | Fixed name |
| Seed migration | Timestamp + `Seed{Description}` | `20240120_SeedDefaultRoles.cs` |

## Summary Table

| Element | C# Convention | PostgreSQL Convention | Who Maps |
|---------|--------------|----------------------|----------|
| Table name | `Order` (singular) | `orders` (plural, snake_case) | `ToTable()` + convention |
| Column name | `FirstName` | `first_name` | Auto (snake_case convention) |
| Primary key | `Id` | `id` | Auto |
| Foreign key | `CustomerId` | `customer_id` | Auto |
| Index | N/A | `ix_{table}_{cols}` | `HasDatabaseName()` |
| FK constraint | N/A | `fk_{table}_{ref}` | `HasConstraintName()` |
| PK constraint | N/A | `pk_{table}` | Auto or `HasName()` |
| Unique constraint | N/A | `uq_{table}_{cols}` | `HasName()` or index |
