---
knowledge-base-summary: "Detect N+1 queries, unbounded queries, missing async/await, large allocations in loops, missing index hints, synchronous I/O, and unnecessary materializations. Performance issues are silent killers -- they work fine in dev and explode in production."
---
# Performance Check

Performance issues are silent killers -- they work fine in development with 10 rows and explode in production with 10,000.

## N+1 Queries

### What to Look For

- **Missing Include/ThenInclude:** Loading a collection of entities, then accessing a navigation property in a loop.
  ```csharp
  // N+1: Each iteration triggers a separate query
  var orders = await _db.Orders.ToListAsync(ct);
  foreach (var order in orders)
  {
      var items = order.Items; // Lazy load = 1 query per order
  }
  ```
- **LINQ projection without Include:** Using `.Select()` that accesses navigation properties without eager loading.
- **Nested loops with database access:** Any loop inside a loop where the inner loop queries the database.

### Fix Pattern

```csharp
// Correct: Eager load with Include
var orders = await _db.Orders
    .Include(o => o.Items)
    .ToListAsync(ct);

// Or: Projection (even better -- only loads needed fields)
var orders = await _db.Orders
    .Select(o => new OrderResponse(o.Id, o.Items.Count))
    .ToListAsync(ct);
```

## Unbounded Queries

### What to Look For

- **ToListAsync without Take or pagination:**
  ```csharp
  // DANGEROUS: What if there are 1 million users?
  var users = await _db.Users.ToListAsync(ct);
  ```
- **Missing pagination on list endpoints:** Any endpoint that returns a collection without limit.
- **COUNT(*) without necessity:** Using `.CountAsync()` when you only need to know "are there any?" (use `.AnyAsync()` instead).
- **Loading all to filter in memory:**
  ```csharp
  // BAD: Loads ALL orders, then filters in C#
  var allOrders = await _db.Orders.ToListAsync(ct);
  var filtered = allOrders.Where(o => o.Status == "active");
  ```

### Fix Pattern

```csharp
// Always limit results
var users = await _db.Users
    .Where(u => u.IsActive)
    .Take(pageSize + 1) // +1 to determine HasMore
    .ToListAsync(ct);
```

## Missing Async/Await

### What to Look For

- **Sync over async:** Calling `.Result` or `.Wait()` on an async method, which blocks the thread.
  ```csharp
  // BAD: Blocks thread, can cause deadlock
  var user = _db.Users.FindAsync(id).Result;
  ```
- **Missing await on async call:** Forgetting to await an async method, losing the result or exception.
  ```csharp
  // BUG: Task is started but never awaited -- exception is swallowed
  _emailSender.SendAsync(email, template, ct);
  ```
- **Async void methods:** Methods marked `async void` instead of `async Task` -- exceptions cannot be caught.

### Fix Pattern

```csharp
// Always await async calls
var user = await _db.Users.FindAsync(id, ct);
await _emailSender.SendAsync(email, template, ct);
```

## Large Object Allocation in Loops

### What to Look For

- **String concatenation in loops:** Using `+=` on strings inside a loop instead of `StringBuilder`.
  ```csharp
  // BAD: Creates a new string object on every iteration
  var result = "";
  foreach (var item in items)
      result += item.Name + ", ";
  ```
- **Creating collections in loops:** Instantiating `new List<T>()` or `new Dictionary<K,V>()` inside a loop when it could be reused.
- **Boxing value types:** Casting value types to `object` repeatedly in a loop.

## Missing Index Hints

### What to Look For

- **Queries filtering on non-indexed columns:** If a handler queries by a field frequently (e.g., `WHERE Email = @email`), check if that column has an index.
- **Composite queries without composite index:** If a query filters on `TenantId + Status`, a composite index is more effective than two separate indexes.
- **OrderBy on non-indexed column:** Sorting by a column that doesn't have an index causes a full table sort.

### How to Check

Look at the EF Core configuration files for index definitions:

```csharp
// In EntityConfiguration.cs
builder.HasIndex(e => e.Email).IsUnique();
builder.HasIndex(e => new { e.TenantId, e.Status });
```

If a query filters or sorts by a column and there's no corresponding index, flag it as a suggestion.

## Synchronous I/O

### What to Look For

- **File.ReadAllText instead of File.ReadAllTextAsync**
- **HttpClient.Send instead of HttpClient.SendAsync**
- **Stream.Read instead of Stream.ReadAsync**
- **Any I/O operation without the Async suffix** in an async context

## Unnecessary Materializations

### What to Look For

- **ToList() before further LINQ operations:**
  ```csharp
  // BAD: Materializes all users, then filters in memory
  var users = await _db.Users.ToListAsync(ct);
  var active = users.Where(u => u.IsActive).ToList();
  ```
- **Multiple ToList() calls in the same query chain:**
  ```csharp
  // BAD: Two materializations
  var items = orders.SelectMany(o => o.Items).ToList().GroupBy(i => i.Category).ToList();
  ```
- **AsEnumerable() or AsQueryable() misuse:** Converting between IQueryable and IEnumerable at the wrong point, causing the rest of the query to run in memory instead of SQL.

### Fix Pattern

```csharp
// Correct: Filter at the database level, materialize once
var activeUsers = await _db.Users
    .Where(u => u.IsActive)
    .Select(u => new UserResponse(u.Id, u.Name))
    .ToListAsync(ct);
```

## Review Checklist

- [ ] No N+1 queries (all navigation properties are eagerly loaded or projected)
- [ ] No unbounded queries (all list queries have Take/pagination)
- [ ] No .Result or .Wait() on async methods
- [ ] No missing await on async calls
- [ ] No string concatenation in loops (use StringBuilder)
- [ ] Frequently queried columns have indexes
- [ ] No synchronous I/O in async methods
- [ ] No unnecessary ToList() before further LINQ operations
- [ ] No loading all records to filter in memory
- [ ] CountAsync used only when the count is needed (not just existence -- use AnyAsync)
