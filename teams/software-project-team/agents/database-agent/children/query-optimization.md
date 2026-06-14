---
knowledge-base-summary: "Preventing N+1 queries with Include and projection. AsNoTracking for read-only paths. Projection (Select) vs full entity load tradeoffs. Pagination performance. Batch operations. Query splitting for complex includes. When raw SQL is justified."
---
# Query Optimization

## N+1 Problem

The most common performance killer. Happens when EF Core loads a collection one item at a time instead of in a single query.

### Detection

```csharp
// BAD — N+1: 1 query for orders + N queries for order items
var orders = await context.Orders.ToListAsync();
foreach (var order in orders)
{
    var items = order.Items; // lazy load triggers a query per order
}
```

### Solution 1: Eager Loading (Include)

```csharp
// GOOD — 1 query with JOIN
var orders = await context.Orders
    .Include(x => x.Items)
    .Include(x => x.Customer)
    .ToListAsync();
```

### Solution 2: Projection (Best for Read-Only)

```csharp
// BEST — only fetch what you need
var orders = await context.Orders
    .Select(x => new OrderDto
    {
        Id = x.Id,
        OrderNumber = x.OrderNumber,
        CustomerName = x.Customer.Name,
        ItemCount = x.Items.Count,
        TotalAmount = x.Items.Sum(i => i.Price * i.Quantity)
    })
    .ToListAsync();
```

Projection avoids loading full entities and their navigation properties. It generates a single optimized SQL query.

### Solution 3: Explicit Loading (Rare)

```csharp
// When you conditionally need related data
var order = await context.Orders.FindAsync(orderId);
if (order.Status == OrderStatus.Completed)
{
    await context.Entry(order)
        .Collection(x => x.Items)
        .LoadAsync();
}
```

## AsNoTracking

For read-only queries, disable change tracking. Reduces memory allocation and speeds up materialization.

```csharp
// Read-only queries — always use AsNoTracking
var orders = await context.Orders
    .AsNoTracking()
    .Where(x => x.Status == OrderStatus.Active)
    .ToListAsync();

// Or set at DbContext level for query-only contexts
public class ReadOnlyDbContext : ApplicationDbContext
{
    public ReadOnlyDbContext(DbContextOptions options) : base(options)
    {
        ChangeTracker.QueryTrackingBehavior = QueryTrackingBehavior.NoTracking;
    }
}
```

**When to use:**
- All GET/query handlers
- Reports and dashboards
- Export operations
- Any operation that does not modify entities

**When NOT to use:**
- Command handlers that modify and save entities
- When you need to call `Update()` or `Remove()` on the returned entity

## Projection vs Full Entity Load

| Approach | Use When |
|----------|----------|
| Full entity (`Include`) | Updating the entity in the same operation |
| Projection (`Select`) | Returning data to the client (read-only) |
| `AsNoTracking` + full entity | Reading full entity but not modifying it |

```csharp
// Projection — generates optimized SELECT with only needed columns
var dto = await context.Orders
    .Where(x => x.Id == orderId)
    .Select(x => new OrderDetailResponse
    {
        Id = x.Id,
        OrderNumber = x.OrderNumber,
        Status = x.Status.ToString(),
        Customer = new CustomerSummary
        {
            Id = x.Customer.Id,
            Name = x.Customer.Name
        },
        Items = x.Items.Select(i => new OrderItemDto
        {
            ProductName = i.Product.Name,
            Quantity = i.Quantity,
            UnitPrice = i.UnitPrice
        }).ToList()
    })
    .FirstOrDefaultAsync();
```

## Pagination Performance

### Cursor-Based Pagination (Preferred)

```csharp
// Efficient — uses index seek, constant performance regardless of page
var query = context.Orders
    .AsNoTracking()
    .Where(x => x.TenantId == tenantId);

if (cursor is not null)
{
    var (sortValue, id) = DecodeCursor(cursor);
    query = query.Where(x => 
        x.CreatedAt < sortValue || 
        (x.CreatedAt == sortValue && x.Id.CompareTo(id) < 0));
}

var items = await query
    .OrderByDescending(x => x.CreatedAt)
    .ThenByDescending(x => x.Id)
    .Take(pageSize + 1) // fetch one extra to determine HasMore
    .Select(x => new OrderListItem { ... })
    .ToListAsync();

var hasMore = items.Count > pageSize;
if (hasMore) items.RemoveAt(items.Count - 1);
```

### Offset Pagination (Avoid for Large Datasets)

```csharp
// SLOW on large tables — PostgreSQL must scan and discard offset rows
var items = await context.Orders
    .OrderByDescending(x => x.CreatedAt)
    .Skip(page * pageSize)  // gets slower as page increases
    .Take(pageSize)
    .ToListAsync();
```

**Required index for cursor pagination:**
```csharp
builder.HasIndex(x => new { x.TenantId, x.CreatedAt })
    .HasDatabaseName("ix_orders_tenant_id_created_at");
```

## Batch Operations

### Bulk Insert

```csharp
// EF Core 7+ — ExecuteUpdate / ExecuteDelete (no entity loading)
// Update all orders for a customer
await context.Orders
    .Where(x => x.CustomerId == customerId && x.Status == OrderStatus.Pending)
    .ExecuteUpdateAsync(x => x
        .SetProperty(o => o.Status, OrderStatus.Cancelled)
        .SetProperty(o => o.UpdatedAt, DateTime.UtcNow));

// Delete all expired tokens
await context.RefreshTokens
    .Where(x => x.ExpiresAt < DateTime.UtcNow)
    .ExecuteDeleteAsync();
```

### Bulk Insert with AddRange

```csharp
// For inserting many entities at once
var items = orders.Select(o => new OrderItem
{
    OrderId = o.Id,
    ProductId = productId,
    Quantity = 1
}).ToList();

context.OrderItems.AddRange(items);
await context.SaveChangesAsync();
```

## Query Splitting

When `Include` generates a cartesian explosion (multiple collection navigations), use `AsSplitQuery`:

```csharp
// WITHOUT split: 1 query with cartesian product (rows = orders * items * comments)
var orders = await context.Orders
    .Include(x => x.Items)
    .Include(x => x.Comments)
    .ToListAsync();

// WITH split: 3 separate queries (1 for orders, 1 for items, 1 for comments)
var orders = await context.Orders
    .Include(x => x.Items)
    .Include(x => x.Comments)
    .AsSplitQuery()
    .ToListAsync();
```

**When to use `AsSplitQuery`:**
- Including 2+ collection navigations
- Collection navigations have many rows
- Network latency is low (same datacenter — which Docker provides)

**When NOT to use:**
- Single collection include (JOIN is fine)
- Reference navigation includes (no cartesian issue)
- When transaction consistency between queries matters

## Raw SQL (Last Resort)

When LINQ cannot express the query or performance is critical:

```csharp
// Via DbContext — still returns tracked entities
var orders = await context.Orders
    .FromSqlRaw("""
        SELECT * FROM orders 
        WHERE created_at > {0}
        AND status = ANY({1})
    """, cutoffDate, statuses)
    .ToListAsync();

// For complex reports — use a view instead
// Step 1: Create view in migration
migrationBuilder.Sql("""
    CREATE VIEW order_summary AS
    SELECT 
        o.customer_id,
        COUNT(*) as order_count,
        SUM(oi.quantity * oi.unit_price) as total_spent
    FROM orders o
    JOIN order_items oi ON oi.order_id = o.id
    WHERE o.is_deleted = false
    GROUP BY o.customer_id;
""");

// Step 2: Map to entity
[Keyless]
public class OrderSummary
{
    public Guid CustomerId { get; set; }
    public int OrderCount { get; set; }
    public decimal TotalSpent { get; set; }
}

// Step 3: Query via DbContext
modelBuilder.Entity<OrderSummary>().ToView("order_summary");
var summaries = await context.OrderSummaries.ToListAsync();
```

## Performance Checklist for Every Query

- [ ] Is `AsNoTracking` used for read-only queries?
- [ ] Is projection used instead of loading full entities (when possible)?
- [ ] Are related entities loaded via `Include` or projection (no N+1)?
- [ ] Is the query supported by an index? (Check with EXPLAIN ANALYZE)
- [ ] For lists: is cursor-based pagination used?
- [ ] For multiple collection includes: is `AsSplitQuery` considered?
- [ ] Are batch operations used instead of loops for updates/deletes?
