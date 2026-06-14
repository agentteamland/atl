---
knowledge-base-summary: "Cursor-based infinite scroll. No page numbers. Encoded cursor: Base64(`{sortValue}|{id}`) — supports sorting by any field. `IncludeCount` optional (default off — `COUNT(*)` is expensive). Fetch `PageSize + 1` to determine `HasMore`. Filtering is NOT generic — each feature writes its own Where clauses in the handler."
---
# Pagination Pattern: Cursor-Based Infinite Scroll

## Philosophy

Page-numbered pagination is not used. All lists work with infinite scroll logic: new data loads as the user scrolls down. This applies to both mobile and web.

## Generic Structure

### PaginatedQuery (Application/Common/)

Base parameters for all list queries:

```csharp
public record PaginatedQuery
{
    public string? Cursor { get; init; }          // cursor of the last item (first page: null)
    public int PageSize { get; init; } = 20;      // how many items to fetch (max 100)
    public string? Search { get; init; }           // general search (optional)
    public string? SortBy { get; init; }           // sort field (optional)
    public bool SortDesc { get; init; } = false;   // sort direction
    public bool IncludeCount { get; init; } = false; // include total count?
}
```

### PaginatedResponse<T> (Application/Common/)

Wrapper for all list responses:

```csharp
public record PaginatedResponse<T>(
    List<T> Items,              // items
    string? NextCursor,         // next page cursor (null = last page)
    bool HasMore,               // are there more?
    int? TotalCount             // populated only when IncludeCount=true
);
```

## Cursor Mechanism

The cursor is in Base64 encoded `{sortValue}|{id}` format. This supports sorting by any field (CreatedAt DESC, Price ASC, Name ASC, etc.).

### Cursor Encode/Decode Utility

```csharp
public static class CursorHelper
{
    public static string Encode(string sortValue, Guid id)
    {
        var raw = $"{sortValue}|{id}";
        return Convert.ToBase64String(Encoding.UTF8.GetBytes(raw));
    }

    public static (string SortValue, Guid Id) Decode(string cursor)
    {
        var raw = Encoding.UTF8.GetString(Convert.FromBase64String(cursor));
        var parts = raw.Split('|', 2);
        return (parts[0], Guid.Parse(parts[1]));
    }
}
```

## Frontend Flow

1. **First request:** `cursor=null` → first 20 items + `nextCursor="abc123"`
2. **Scroll down:** `cursor="abc123"` → next 20 + `nextCursor="def456"`
3. **Last page:** `nextCursor=null`, `hasMore=false` → loading ends

## Usage Example in Handler

```csharp
public async Task<PaginatedResponse<OrderDto>> Handle(
    GetOrdersQuery request, CancellationToken ct)
{
    var query = _db.Orders.AsQueryable();

    // Feature-specific filtering (each query writes its own Where conditions)
    if (request.Status.HasValue)
        query = query.Where(o => o.Status == request.Status.Value);

    // Search (generic)
    if (!string.IsNullOrWhiteSpace(request.Search))
        query = query.Where(o => o.CustomerName.Contains(request.Search));

    // Sort
    var sortBy = request.SortBy ?? "CreatedAt";
    query = request.SortDesc
        ? query.OrderByDescending(e => EF.Property<object>(e, sortBy))
        : query.OrderBy(e => EF.Property<object>(e, sortBy));

    // Cursor (if present, continue from that point)
    if (!string.IsNullOrEmpty(request.Cursor))
    {
        var (sortValue, lastId) = CursorHelper.Decode(request.Cursor);
        // Cursor-based WHERE condition (depends on sort direction)
        // ...implementation detail varies by sort field
    }

    // TotalCount (optional, only when requested)
    int? totalCount = request.IncludeCount
        ? await query.CountAsync(ct)
        : null;

    // Fetch (pull PageSize + 1 — to determine hasMore)
    var items = await query
        .Take(request.PageSize + 1)
        .ToListAsync(ct);

    var hasMore = items.Count > request.PageSize;
    if (hasMore) items.RemoveAt(items.Count - 1);

    // Generate cursor
    string? nextCursor = null;
    if (hasMore && items.Count > 0)
    {
        var lastItem = items.Last();
        var sortFieldValue = lastItem.GetType()
            .GetProperty(sortBy)?.GetValue(lastItem)?.ToString() ?? "";
        nextCursor = CursorHelper.Encode(sortFieldValue, lastItem.Id);
    }

    // Map to DTO
    var dtos = items.Select(o => new OrderDto(o.Id, o.CustomerName, o.Total, o.CreatedAt)).ToList();

    return new PaginatedResponse<OrderDto>(dtos, nextCursor, hasMore, totalCount);
}
```

## Important Rules

1. **Fetch PageSize + 1.** Fetch one extra item to determine `hasMore`, then remove the extra from the list.
2. **TotalCount is off by default.** `COUNT(*)` is expensive in PostgreSQL (MVCC full scan). Only compute when `IncludeCount=true` is sent. Usually not needed on mobile, may be needed in admin panels.
3. **PageSize max limit.** Limit `PageSize` to max 100 in the Validator — a malicious client shouldn't be able to fetch 10000.
4. **Filtering is NOT generic.** Each feature writes its own Where conditions in its handler. There is no generic filter system — consistent with the Vertical Slice philosophy.
5. **Cursor is opaque.** The frontend only stores the cursor and sends it back — it does not decode it or care about its contents.

