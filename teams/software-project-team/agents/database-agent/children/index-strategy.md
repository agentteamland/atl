---
knowledge-base-summary: "When and how to add indexes. Composite indexes, partial indexes (PostgreSQL WHERE clause), covering indexes. B-tree vs GIN vs GiST selection. Mandatory index on every foreign key. EXPLAIN ANALYZE for verification. Monitoring slow queries via pg_stat_statements."
---
# Index Strategy

## Core Principle

Every index must have a documented reason. No speculative indexes. Every query pattern must have a supporting index.

## When to Add an Index

| Scenario | Action |
|----------|--------|
| Foreign key column | **Always** — mandatory, no exceptions |
| Column used in WHERE clause | Add B-tree index |
| Column used in ORDER BY | Add B-tree index (match sort direction) |
| Column used in GROUP BY | Add B-tree index |
| Unique business key (email, slug) | Add unique index |
| Full-text search column | Add GIN index |
| JSONB column queried with `@>`, `?`, `?&` | Add GIN index |
| Geospatial data | Add GiST index |
| Column with very low cardinality (bool, status) | Consider partial index instead |
| Column only used in JOINs (as FK) | Already covered by FK index rule |

## Index Types in PostgreSQL

### B-tree (Default)

The workhorse. Supports `=`, `<`, `>`, `<=`, `>=`, `BETWEEN`, `IN`, `IS NULL`, `LIKE 'prefix%'`.

```csharp
// Simple B-tree
builder.HasIndex(x => x.CreatedAt)
    .HasDatabaseName("ix_orders_created_at");

// Descending (for ORDER BY ... DESC queries)
builder.HasIndex(x => x.CreatedAt)
    .HasDatabaseName("ix_orders_created_at_desc")
    .IsDescending();
```

### GIN (Generalized Inverted Index)

Best for: full-text search, JSONB queries, array containment.

```csharp
// GIN on JSONB column
builder.HasIndex(x => x.Metadata)
    .HasDatabaseName("ix_orders_metadata")
    .HasMethod("gin");

// GIN for full-text search (via migration SQL)
migrationBuilder.Sql("""
    CREATE INDEX ix_products_search 
    ON products USING gin(to_tsvector('english', name || ' ' || description));
""");
```

### GiST (Generalized Search Tree)

Best for: geometric/geospatial data, range types, nearest-neighbor search.

```csharp
// GiST for geospatial (via migration SQL — EF doesn't support GiST natively)
migrationBuilder.Sql("""
    CREATE INDEX ix_locations_coordinates 
    ON locations USING gist(coordinates);
""");
```

## Composite Indexes

Order matters. The leftmost column should be the most selective (most unique values) or the column used in equality conditions.

```csharp
// Query: WHERE tenant_id = X AND created_at > Y ORDER BY created_at DESC
// tenant_id first (equality), then created_at (range + sort)
builder.HasIndex(x => new { x.TenantId, x.CreatedAt })
    .HasDatabaseName("ix_orders_tenant_id_created_at");
```

### Composite Index Rules

1. **Equality columns first, range columns last** — `WHERE status = 'Active' AND created_at > '2024-01-01'` → index on `(status, created_at)`
2. **Most selective first** (within equality group)
3. **Maximum 3-4 columns** — wider indexes have diminishing returns and slow down writes
4. **The index supports queries on its leading columns** — an index on `(a, b, c)` supports queries on `(a)`, `(a, b)`, and `(a, b, c)`, but NOT `(b)` or `(c)` alone

## Partial Indexes (PostgreSQL WHERE Clause)

Index only rows that match a condition. Smaller index, faster lookups for specific queries.

```csharp
// Only index active orders — inactive orders are rarely queried
builder.HasIndex(x => x.CreatedAt)
    .HasDatabaseName("ix_orders_created_at_active")
    .HasFilter("is_deleted = false");

// Only index unprocessed items — processed items don't need fast lookup
builder.HasIndex(x => x.Status)
    .HasDatabaseName("ix_orders_status_pending")
    .HasFilter("status = 'Pending'");
```

### When to Use Partial Indexes

- Soft delete: filter out `is_deleted = true` rows
- Status-based queries: only index certain statuses
- Boolean flags: index only `true` or `false` rows
- Temporal: index only recent/active records

## Covering Indexes (INCLUDE)

Add non-indexed columns to the index so the query can be answered from the index alone (index-only scan).

```csharp
// Query: SELECT name, email FROM users WHERE tenant_id = X
// Include name and email so PostgreSQL doesn't need to read the table
builder.HasIndex(x => x.TenantId)
    .HasDatabaseName("ix_users_tenant_id")
    .IncludeProperties(x => new { x.Name, x.Email });
```

### When to Use Covering Indexes

- Query selects a small number of columns
- Those columns are frequently co-queried with the indexed column
- Table is large and avoiding heap access matters for performance
- Do NOT over-include — every included column increases index size

## Foreign Key Index (Mandatory)

**Every foreign key MUST have an index.** EF Core on PostgreSQL does NOT create FK indexes automatically.

```csharp
// Relationship
builder.HasOne(x => x.Customer)
    .WithMany(x => x.Orders)
    .HasForeignKey(x => x.CustomerId)
    .OnDelete(DeleteBehavior.Restrict);

// Index on FK — MANDATORY
builder.HasIndex(x => x.CustomerId)
    .HasDatabaseName("ix_orders_customer_id");
```

## Unique Indexes

```csharp
// Unique business key
builder.HasIndex(x => x.Email)
    .HasDatabaseName("ix_users_email")
    .IsUnique();

// Composite unique (tenant + slug must be unique together)
builder.HasIndex(x => new { x.TenantId, x.Slug })
    .HasDatabaseName("ix_products_tenant_id_slug")
    .IsUnique();

// Unique but only for non-deleted records
builder.HasIndex(x => x.Email)
    .HasDatabaseName("ix_users_email_active")
    .IsUnique()
    .HasFilter("is_deleted = false");
```

## Verifying Index Effectiveness

### EXPLAIN ANALYZE

```sql
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT * FROM orders 
WHERE customer_id = '...' 
ORDER BY created_at DESC 
LIMIT 20;
```

**What to look for:**
- `Index Scan` or `Index Only Scan` — good, index is being used
- `Seq Scan` — bad on large tables, likely missing index
- `Bitmap Index Scan` — acceptable, happens when multiple conditions combine
- `Rows Removed by Filter` — high number means index is not selective enough

### Check Index Usage

```sql
-- Which indexes are being used?
SELECT 
    indexrelname AS index_name,
    idx_scan AS times_used,
    idx_tup_read AS rows_read,
    idx_tup_fetch AS rows_fetched
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Unused indexes (candidates for removal)
SELECT 
    indexrelname AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND schemaname = 'public'
ORDER BY pg_relation_size(indexrelid) DESC;
```

## Anti-Patterns

1. **Index on every column** — slows down writes, wastes disk space. Only index what queries need.
2. **Missing FK index** — causes slow JOINs and cascading delete performance issues.
3. **Wrong column order in composite index** — equality columns must come before range columns.
4. **Indexing low-cardinality columns alone** — a B-tree on a boolean column is useless. Use a partial index instead.
5. **Duplicate indexes** — an index on `(a, b)` makes a standalone index on `(a)` redundant.
6. **Never checking if indexes are used** — run `pg_stat_user_indexes` regularly to find unused indexes.
