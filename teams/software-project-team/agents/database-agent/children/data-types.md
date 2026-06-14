---
knowledge-base-summary: "PostgreSQL-specific type mappings: uuid with gen_random_uuid(), timestamptz (always UTC), text vs varchar, jsonb for semi-structured data, decimal for money (never float), enum mapping via HasConversion, array types. EF Core configuration for each."
---
# Data Types

## PostgreSQL-Specific Type Mappings

This document covers PostgreSQL-specific data types and how to configure them in EF Core.

## UUID (Primary Keys)

Always use `uuid` for primary keys. Never use auto-increment integers.

```csharp
// Domain entity
public class Order : BaseEntity
{
    // BaseEntity provides: public Guid Id { get; set; }
}

// Configuration
builder.HasKey(x => x.Id);
builder.Property(x => x.Id)
    .HasDefaultValueSql("gen_random_uuid()");
```

**Why UUID:**
- No sequential guessing (security)
- No coordination needed for distributed systems
- Client can generate the ID before sending to server
- `gen_random_uuid()` is a PostgreSQL built-in (v13+), no extension needed

**Performance note:** UUID v4 is random, which can cause B-tree index fragmentation on high-write tables. For such cases, consider UUID v7 (time-sorted) when PostgreSQL 17+ is available, or accept the tradeoff.

## Timestamps (Always UTC)

Always use `timestamptz` (timestamp with time zone). Never use `timestamp` (without timezone).

```csharp
// Configuration
builder.Property(x => x.CreatedAt)
    .IsRequired()
    .HasDefaultValueSql("now()");

builder.Property(x => x.UpdatedAt);

builder.Property(x => x.ExpiresAt)
    .IsRequired();
```

EF Core maps `DateTime` to `timestamptz` by default with Npgsql. To be explicit:

```csharp
builder.Property(x => x.CreatedAt)
    .HasColumnType("timestamptz");
```

**Rules:**
- Always store in UTC (`DateTime.UtcNow` in C#)
- Never use `timestamp` without timezone — it creates ambiguity
- Display timezone conversion happens in the client (Flutter/React), never in the DB
- Use `DateTimeOffset` in C# if you need to preserve the original offset

## Text vs Varchar

| Type | PostgreSQL | Use When |
|------|-----------|----------|
| `varchar(n)` | `character varying(n)` | Known max length (email, phone, slug) |
| `text` | `text` | Unknown or very long content (bio, description) |

```csharp
// Bounded string — always prefer when max length is known
builder.Property(x => x.Email)
    .IsRequired()
    .HasMaxLength(256); // generates varchar(256)

builder.Property(x => x.PhoneNumber)
    .HasMaxLength(20);

// Unbounded string — for long-form text
builder.Property(x => x.Description)
    .HasColumnType("text"); // explicit, though EF defaults to text for no-MaxLength strings
```

**Guideline:** If you can define a reasonable max length, use `HasMaxLength()`. It aligns client validation with database constraints. Use `text` only for truly variable-length content like user bios, descriptions, or notes.

## JSONB (Semi-Structured Data)

Use `jsonb` for flexible, queryable semi-structured data. Never use `json` (no indexing, no deduplication).

```csharp
// Domain
public class Product : BaseEntity
{
    public string Name { get; set; } = default!;
    public Dictionary<string, object> Metadata { get; set; } = new();
    public List<string> Tags { get; set; } = new();
}

// Configuration
builder.Property(x => x.Metadata)
    .HasColumnType("jsonb")
    .HasDefaultValueSql("'{}'::jsonb");

builder.Property(x => x.Tags)
    .HasColumnType("jsonb")
    .HasDefaultValueSql("'[]'::jsonb");

// GIN index for JSONB queries
builder.HasIndex(x => x.Metadata)
    .HasDatabaseName("ix_products_metadata")
    .HasMethod("gin");
```

**Querying JSONB in PostgreSQL:**
```sql
-- Contains key
SELECT * FROM products WHERE metadata ? 'color';

-- Contains key-value
SELECT * FROM products WHERE metadata @> '{"color": "red"}'::jsonb;

-- Access nested value
SELECT * FROM products WHERE metadata->>'brand' = 'Nike';
```

**When to use JSONB:**
- Dynamic attributes that vary per record (product metadata, user preferences)
- Storing external API responses
- Configuration/settings that don't warrant their own columns

**When NOT to use JSONB:**
- Data that needs FK references (use a proper table)
- Data that is queried in every request (make it a column)
- Data with a fixed, known schema (use columns)

## Decimal (Money)

**NEVER use `float` or `double` for money.** Use `decimal` with explicit precision.

```csharp
// Domain
public class Order : BaseEntity
{
    public decimal TotalAmount { get; set; }
    public decimal TaxRate { get; set; }
    public decimal DiscountAmount { get; set; }
}

// Configuration
builder.Property(x => x.TotalAmount)
    .HasPrecision(18, 2); // 18 total digits, 2 decimal places

builder.Property(x => x.TaxRate)
    .HasPrecision(5, 4); // e.g., 0.1850 (18.50%)

builder.Property(x => x.DiscountAmount)
    .HasPrecision(18, 2)
    .HasDefaultValue(0m);
```

PostgreSQL uses `numeric(precision, scale)` which maps to C# `decimal`.

## Enum Mapping

Store enums as strings in the database. Never as integers (order changes break data).

```csharp
// Domain
public enum OrderStatus
{
    Pending,
    Confirmed,
    Processing,
    Shipped,
    Delivered,
    Cancelled
}

// Configuration
builder.Property(x => x.Status)
    .IsRequired()
    .HasConversion<string>()
    .HasMaxLength(50);
```

**Why string over int:**
- Human-readable in database queries
- Adding new enum values doesn't shift existing values
- Removing/reordering enum values doesn't corrupt data
- Debugging is easier (seeing "Cancelled" vs "5")

### PostgreSQL Native Enum (Alternative)

For stricter type safety at DB level:

```csharp
// In migration
migrationBuilder.Sql("""
    CREATE TYPE order_status AS ENUM (
        'Pending', 'Confirmed', 'Processing', 
        'Shipped', 'Delivered', 'Cancelled'
    );
""");

// In DbContext OnModelCreating
modelBuilder.HasPostgresEnum<OrderStatus>();

// In Configuration
builder.Property(x => x.Status)
    .IsRequired();
```

**Tradeoff:** PostgreSQL native enums are more type-safe but harder to modify (adding values requires `ALTER TYPE`). Prefer string conversion for simplicity unless you need DB-level enum enforcement.

## Array Types

PostgreSQL supports native arrays. Useful for simple lists without join tables.

```csharp
// Domain
public class User : BaseEntity
{
    public List<string> Roles { get; set; } = new();
    public int[] PermissionIds { get; set; } = Array.Empty<int>();
}

// Configuration
builder.Property(x => x.Roles)
    .HasColumnType("text[]");

builder.Property(x => x.PermissionIds)
    .HasColumnType("integer[]");

// GIN index for array containment queries
builder.HasIndex(x => x.Roles)
    .HasDatabaseName("ix_users_roles")
    .HasMethod("gin");
```

**Querying arrays in PostgreSQL:**
```sql
-- Contains element
SELECT * FROM users WHERE 'Admin' = ANY(roles);

-- Contains all elements
SELECT * FROM users WHERE roles @> ARRAY['Admin', 'Manager'];

-- Overlaps (any match)
SELECT * FROM users WHERE roles && ARRAY['Admin', 'Manager'];
```

**When to use arrays:**
- Simple value lists (tags, roles, labels)
- Small, bounded lists
- Queried with containment operators

**When NOT to use arrays:**
- Items need their own properties (use a join table)
- List grows unbounded (use a related table)
- Items need FK references

## Boolean

```csharp
builder.Property(x => x.IsActive)
    .IsRequired()
    .HasDefaultValue(true);

builder.Property(x => x.IsDeleted)
    .IsRequired()
    .HasDefaultValue(false);
```

For boolean columns, consider partial indexes instead of full indexes:

```csharp
// Only index active records — much smaller than indexing all rows
builder.HasIndex(x => x.Email)
    .HasDatabaseName("ix_users_email_active")
    .IsUnique()
    .HasFilter("is_active = true");
```

## Type Mapping Quick Reference

| C# Type | PostgreSQL Type | EF Configuration |
|---------|----------------|------------------|
| `Guid` | `uuid` | `HasDefaultValueSql("gen_random_uuid()")` |
| `DateTime` | `timestamptz` | Default with Npgsql |
| `DateTimeOffset` | `timestamptz` | Default with Npgsql |
| `string` (bounded) | `varchar(n)` | `HasMaxLength(n)` |
| `string` (unbounded) | `text` | Default (no MaxLength) |
| `decimal` | `numeric(p,s)` | `HasPrecision(p, s)` |
| `bool` | `boolean` | Default |
| `int` | `integer` | Default |
| `long` | `bigint` | Default |
| `enum` | `varchar(n)` | `HasConversion<string>().HasMaxLength(n)` |
| `Dictionary<string,object>` | `jsonb` | `HasColumnType("jsonb")` |
| `List<string>` | `text[]` | `HasColumnType("text[]")` |
| `byte[]` | `bytea` | Default (avoid for large files — use object storage) |
