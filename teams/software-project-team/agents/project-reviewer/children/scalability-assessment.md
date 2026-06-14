---
knowledge-base-summary: "Evaluate database query scalability, cache effectiveness, stateless service compliance, connection pooling, rate limiting coverage, and resource limits. Will this project handle growth?"
---
# Scalability Assessment

## Database Query Scalability

### What to Check

For every significant database query in the codebase, ask: "Will this work with 1 million rows?"

| Pattern | Problem at Scale | Fix |
|---------|-----------------|-----|
| `ToListAsync()` without `Take()` | Loads entire table into memory | Add pagination |
| `CountAsync()` on large tables | Full table scan | Use approximate counts or cache |
| `WHERE` on non-indexed column | Full table scan | Add index |
| `LIKE '%search%'` | Cannot use index (leading wildcard) | Full-text search |
| `OrderBy` on non-indexed column | Full table sort | Add index on sort column |
| `JOIN` without index on FK | Nested loop join | Add index on foreign key |
| `SELECT *` (loading all columns) | Wasted I/O and memory | Project only needed columns |
| `DISTINCT` on large result set | Sorting and dedup in memory | Redesign query or add unique constraint |

### How to Assess

1. **List all DbContext queries:** Search for `_db.` or `_context.` usage patterns
2. **Check each for pagination:** Does it have `Take()` or cursor-based pagination?
3. **Check for index support:** Cross-reference WHERE and OrderBy columns with EF Configuration index definitions
4. **Estimate table size:** Based on the entity type, estimate how many rows are expected

### Severity

- Unbounded query on a table expected to grow: Red
- Missing index on frequently queried column: Amber
- SELECT * when only 2 fields needed: Green (suggestion)

## Cache Effectiveness

### What to Check

1. **Cache hit rate potential:** Are the right things being cached?
   - Frequently read, rarely changed data: Good cache candidate
   - Rarely read, frequently changed data: Bad cache candidate
   - User-specific data: Cache only if access is frequent

2. **Cache invalidation:** When data changes, is the cache invalidated?
   - Commands that modify data should invalidate related query caches
   - Look for `ICacheInvalidator` implementation on commands
   - Missing invalidation = stale data served to users

3. **TTL configuration:** Are cache durations reasonable?
   - Too short: Cache misses, no benefit
   - Too long: Stale data served
   - No TTL: Memory grows indefinitely

4. **Cache key design:** Are cache keys specific enough?
   - `"users"` as cache key: Dangerous -- one invalidation clears all user caches
   - `"user:{id}"` as cache key: Better -- targeted invalidation

### Common Findings

- Query marked `ICacheable` but corresponding command doesn't implement `ICacheInvalidator`
- Cache keys too broad (clearing too much on invalidation)
- No TTL set (memory leak potential)
- Caching user-specific data without user ID in cache key
- Caching data that changes every request (wasted memory)

## Stateless Service Check

### The Rule

Every service should be stateless -- meaning you can run multiple instances behind a load balancer without problems.

### What to Check

| State Type | Problem | Fix |
|-----------|---------|-----|
| In-memory cache (Dictionary, ConcurrentDictionary) | Not shared across instances | Use Redis |
| File system storage (local disk) | Not shared across instances | Use S3/MinIO |
| Static variables holding state | Not shared, race conditions | Use distributed store |
| Session state (in-process) | Lost on restart, not shared | Use Redis session store |
| Background task tracking | Not shared across instances | Use distributed scheduling |

### How to Detect

```bash
# Find in-memory caching
grep -rn "ConcurrentDictionary\|MemoryCache\|Dictionary<.*,.*>" src/ --include="*.cs"

# Find static mutable state
grep -rn "static.*=.*new\|static.*List\|static.*Dictionary" src/ --include="*.cs"

# Find local file operations
grep -rn "File.Write\|File.Read\|StreamWriter\|Path.Combine" src/ --include="*.cs"
```

### Exceptions

Some state is acceptable:
- Channel/Queue for internal producer-consumer (within the same process)
- Configuration loaded at startup (read-only after init)
- Logger instances (stateless by design)

## Connection Pooling

### What to Check

| Resource | Pooling Method | Configuration |
|----------|---------------|---------------|
| PostgreSQL | EF Core default (Npgsql pool) | `MaxPoolSize`, `MinPoolSize` in connection string |
| Redis | StackExchange.Redis multiplexer | `ConnectionMultiplexer` (singleton) |
| RabbitMQ | Connection per service (not per message) | Single connection, multiple channels |
| HTTP clients | `IHttpClientFactory` | Named/typed clients with pooling |

### Common Findings

- **New connection per request:** Creating a new DB connection or HTTP client per request instead of using a pool.
  ```csharp
  // BAD: New HttpClient per request
  var client = new HttpClient();
  
  // CORRECT: Injected via IHttpClientFactory
  var client = _httpClientFactory.CreateClient("ExternalApi");
  ```

- **Missing connection string pool settings:** Default pool size might be too small for production.

- **Redis multiplexer created per request:** `ConnectionMultiplexer` should be a singleton.

- **RabbitMQ connection per message:** Creating a new connection for each message instead of reusing.

## Rate Limiting Coverage

### What to Check

| Endpoint Category | Rate Limit Needed | Suggested Limit |
|-------------------|-------------------|-----------------|
| Authentication (login, register) | Mandatory | 5-10 req/min per IP |
| Password reset / verification | Mandatory | 3-5 req/min per IP |
| File upload | Mandatory | 10-20 req/min per user |
| Search / listing | Recommended | 60 req/min per user |
| CRUD write operations | Recommended | 30 req/min per user |
| CRUD read operations | Optional | 120 req/min per user |
| Health check | Not needed | N/A |

### How to Verify

1. List all endpoints
2. Check which ones have `.RequireRateLimiting()`
3. Flag unprotected endpoints that should have limits

### Common Findings

- Auth endpoints without rate limiting (brute force risk)
- No global rate limit as fallback
- Rate limit configured but policy name doesn't match any registered policy
- Rate limiting based on IP only (shared IP = shared limit)

## Resource Limits

### What to Check

1. **Docker memory limits:**
   ```yaml
   services:
     api:
       deploy:
         resources:
           limits:
             memory: 512M
           reservations:
             memory: 256M
   ```

2. **Database connection limits:** Is `MaxPoolSize` set appropriately?

3. **Request size limits:** Is there a max request body size configured?
   ```csharp
   // Kestrel max request body size
   builder.WebHost.ConfigureKestrel(options => {
       options.Limits.MaxRequestBodySize = 10 * 1024 * 1024; // 10MB
   });
   ```

4. **File upload limits:** Max file size, max number of files per request.

5. **Query result limits:** Is there a maximum page size enforced?

### Common Findings

- No Docker memory limits (one service can consume all memory)
- No request body size limit (DoS via large payload)
- No file upload size limit
- No maximum page size on pagination endpoints
- Default connection pool size used without assessment

## Review Checklist

- [ ] All database queries assessed for scalability
- [ ] Pagination present on all list queries
- [ ] Indexes exist for frequently queried columns
- [ ] Cache usage is effective (right data cached, invalidation in place)
- [ ] Services are stateless (no in-memory mutable state)
- [ ] Connection pooling configured for all external resources
- [ ] Rate limiting covers auth and sensitive endpoints
- [ ] Docker resource limits configured
- [ ] Request/upload size limits configured
- [ ] Maximum page size enforced on pagination
