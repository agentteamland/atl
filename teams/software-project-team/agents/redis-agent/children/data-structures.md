---
knowledge-base-summary: "When to use which Redis data type. String for simple cache and atomic counters. Hash for object fields without full serialization. Set for unique collections and membership checks. Sorted Set for leaderboards and time-based data. List for queues and recent-items. Code examples with StackExchange.Redis for each type."
---
# Redis Data Structures

Redis is not just a key-value store. Choosing the right data structure for each use case is critical for performance and memory efficiency. This guide covers when to use each type with StackExchange.Redis code examples.

## Decision Table

| Use Case | Data Structure | Why |
|----------|---------------|-----|
| Simple cache (JSON object) | String | Most common, straightforward serialize/deserialize |
| Atomic counter | String (INCR) | Thread-safe increment without read-modify-write |
| Object with partial reads/writes | Hash | Read/write individual fields without full deserialization |
| Unique collection, membership test | Set | O(1) membership check, no duplicates |
| Leaderboard, ranking | Sorted Set | Score-based ordering, range queries |
| Time-based expiry list | Sorted Set | Score = timestamp, range delete for cleanup |
| Queue (FIFO) | List | Push right, pop left |
| Recent items (bounded) | List (LTRIM) | Keep last N items automatically |

## String

The most common type. Stores serialized JSON, counters, or simple values.

### Simple Cache

```csharp
// Store a serialized object
var product = new ProductResponse { Id = id, Name = "Widget", Price = 29.99m };
var json = JsonSerializer.Serialize(product);
await _redis.StringSetAsync(
    RedisKeys.ProductCache(id),
    json,
    TimeSpan.FromMinutes(30)
);

// Retrieve
var cached = await _redis.StringGetAsync(RedisKeys.ProductCache(id));
if (cached.HasValue)
{
    var product = JsonSerializer.Deserialize<ProductResponse>(cached!);
}
```

### Atomic Counter

```csharp
// Increment login attempts (atomic, thread-safe)
var key = RedisKeys.LoginAttempts(ipAddress);
var attempts = await _redis.StringIncrementAsync(key);

// Set expiry on first increment
if (attempts == 1)
{
    await _redis.KeyExpireAsync(key, TimeSpan.FromMinutes(5));
}

// Check threshold
if (attempts > 5)
{
    throw new TooManyAttemptsException();
}

// Decrement (e.g., on successful login)
await _redis.StringDecrementAsync(key);
```

### Set If Not Exists (SETNX)

```csharp
// Idempotency check — only process once
var key = $"idempotency:{requestId}";
var wasSet = await _redis.StringSetAsync(
    key,
    "processing",
    TimeSpan.FromHours(24),
    When.NotExists  // SETNX semantics
);

if (!wasSet)
{
    // Already processed — return cached result or skip
    return;
}
```

### When to Use String
- Single value cache (most common case)
- Counters that need atomic operations
- Simple flags (exists/not exists)
- Any value that is always read and written as a whole

## Hash

Stores an object as individual fields. Useful when you need to read or update specific fields without deserializing the entire object.

### User Session

```csharp
var key = $"session:user:{userId}";

// Store multiple fields at once
var entries = new HashEntry[]
{
    new("userId", userId.ToString()),
    new("email", user.Email),
    new("role", user.Role),
    new("tenantId", user.TenantId.ToString()),
    new("lastActivity", DateTimeOffset.UtcNow.ToUnixTimeSeconds()),
};
await _redis.HashSetAsync(key, entries);
await _redis.KeyExpireAsync(key, TimeSpan.FromMinutes(30));

// Read a single field (no deserialization needed)
var role = await _redis.HashGetAsync(key, "role");

// Read all fields
var allFields = await _redis.HashGetAllAsync(key);
var session = new UserSession
{
    UserId = Guid.Parse(allFields.First(x => x.Name == "userId").Value!),
    Email = allFields.First(x => x.Name == "email").Value!,
    Role = allFields.First(x => x.Name == "role").Value!,
};

// Update a single field (without touching others)
await _redis.HashSetAsync(key, "lastActivity", DateTimeOffset.UtcNow.ToUnixTimeSeconds());

// Increment a numeric field atomically
await _redis.HashIncrementAsync(key, "requestCount", 1);

// Check if a field exists
var exists = await _redis.HashExistsAsync(key, "tenantId");

// Delete a specific field
await _redis.HashDeleteAsync(key, "temporaryFlag");
```

### When to Use Hash
- Objects where you frequently read/write individual fields
- Session data (update lastActivity without reading entire session)
- User profiles with many fields but partial reads
- Avoid for: simple cache where you always serialize/deserialize the whole object (use String instead)

### Hash vs String for Objects
- **String**: serialize entire object, read/write all-or-nothing, simpler code
- **Hash**: field-level access, partial updates, no serialization overhead for single fields
- **Rule of thumb**: if you always read the whole object, use String. If you update individual fields, use Hash.

## Set

Unordered collection of unique values. O(1) add, remove, and membership check.

### Online Users

```csharp
var key = RedisKeys.OnlineUsers();

// Add user to online set
await _redis.SetAddAsync(key, userId.ToString());

// Remove user from online set
await _redis.SetRemoveAsync(key, userId.ToString());

// Check if user is online
var isOnline = await _redis.SetContainsAsync(key, userId.ToString());

// Get all online users
var onlineUsers = await _redis.SetMembersAsync(key);

// Count online users
var count = await _redis.SetLengthAsync(key);
```

### Tags / Categories

```csharp
// Add product to a category set
await _redis.SetAddAsync($"cache:category:{categoryId}:products", productId.ToString());

// Get all products in a category
var productIds = await _redis.SetMembersAsync($"cache:category:{categoryId}:products");

// Products in BOTH categories (intersection)
var intersection = await _redis.SetCombineAsync(
    SetOperation.Intersect,
    $"cache:category:electronics:products",
    $"cache:category:sale:products"
);

// Products in EITHER category (union)
var union = await _redis.SetCombineAsync(
    SetOperation.Union,
    $"cache:category:electronics:products",
    $"cache:category:sale:products"
);
```

### When to Use Set
- Tracking unique members (online users, active sessions)
- Tag systems (which items belong to which tags)
- Set operations needed (intersection, union, difference)
- Deduplication (ensure each item appears only once)

## Sorted Set

Like Set but each member has a score. Members are ordered by score. Perfect for rankings and time-based data.

### Leaderboard

```csharp
var key = "cache:leaderboard:weekly";

// Add/update score
await _redis.SortedSetAddAsync(key, userId.ToString(), score);

// Increment score atomically
await _redis.SortedSetIncrementAsync(key, userId.ToString(), pointsEarned);

// Get top 10 (highest scores first)
var top10 = await _redis.SortedSetRangeByRankWithScoresAsync(
    key,
    start: 0,
    stop: 9,
    order: Order.Descending
);

foreach (var entry in top10)
{
    Console.WriteLine($"User: {entry.Element}, Score: {entry.Score}");
}

// Get user's rank (0-based, highest first)
var rank = await _redis.SortedSetRankAsync(key, userId.ToString(), Order.Descending);

// Get user's score
var userScore = await _redis.SortedSetScoreAsync(key, userId.ToString());
```

### Time-Based Data (Recent Activity)

```csharp
var key = $"cache:activity:{userId}";
var timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds();

// Add activity with timestamp as score
var activityJson = JsonSerializer.Serialize(new { Action = "login", Ip = ip });
await _redis.SortedSetAddAsync(key, activityJson, timestamp);

// Get last 20 activities (most recent first)
var recent = await _redis.SortedSetRangeByRankAsync(
    key,
    start: 0,
    stop: 19,
    order: Order.Descending
);

// Remove activities older than 7 days
var sevenDaysAgo = DateTimeOffset.UtcNow.AddDays(-7).ToUnixTimeSeconds();
await _redis.SortedSetRemoveRangeByScoreAsync(key, double.NegativeInfinity, sevenDaysAgo);

// Count activities in last hour
var oneHourAgo = DateTimeOffset.UtcNow.AddHours(-1).ToUnixTimeSeconds();
var countLastHour = await _redis.SortedSetLengthAsync(key, oneHourAgo, double.PositiveInfinity);
```

### When to Use Sorted Set
- Rankings and leaderboards
- Time-series data with range queries
- Priority queues (score = priority)
- Scheduled tasks (score = execution timestamp)
- Rate limiting with sliding windows

## List

Ordered sequence. Supports push/pop from both ends. Good for queues and bounded recent-items lists.

### Recent Items (Bounded List)

```csharp
var key = $"cache:recent-searches:{userId}";

// Add to the left (most recent first)
await _redis.ListLeftPushAsync(key, searchQuery);

// Trim to keep only last 20 items
await _redis.ListTrimAsync(key, 0, 19);

// Set/reset expiry
await _redis.KeyExpireAsync(key, TimeSpan.FromDays(7));

// Get last 10 recent searches
var recent = await _redis.ListRangeAsync(key, 0, 9);
```

### Simple Queue (FIFO)

```csharp
var key = "queue:notifications";

// Enqueue (push to tail)
await _redis.ListRightPushAsync(key, notificationJson);

// Dequeue (pop from head) — blocks if empty
var item = await _redis.ListLeftPopAsync(key);

// Check queue length
var length = await _redis.ListLengthAsync(key);
```

### When to Use List
- Recent items with bounded length (LTRIM)
- Simple queues (prefer RabbitMQ for durable message queues)
- Activity feeds (push new, trim old)
- Avoid for: random access by index (O(n) for middle elements)

## Summary: Quick Reference

```
String  → cache:product:{id}         = "{json}"              TTL: 30min
Hash    → session:user:{id}          = {email, role, ...}    TTL: 30min
Set     → connections:online         = {user1, user2, ...}   TTL: none (managed)
SortedSet → cache:leaderboard:weekly = {user1:100, user2:95} TTL: 7days
List    → cache:recent:{userId}      = [search3, search2, search1]  TTL: 7days
```
