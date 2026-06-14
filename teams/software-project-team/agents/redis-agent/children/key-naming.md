---
knowledge-base-summary: "Key naming pattern: `{scope}:{entity}:{identifier}`. Six standard scopes: cache, lock, session, rate, settings, connections. Wildcard patterns for bulk operations with SCAN. Key expiry conventions aligned with TTL strategy. Naming must be human-readable in Redis Commander."
---
# Key Naming Convention

Every Redis key in the project follows a strict naming pattern. Consistent naming makes keys human-readable in Redis Commander, enables wildcard operations, and prevents collisions.

## Pattern

```
{scope}:{entity}:{identifier}
```

### Parts

| Part | Description | Example |
|------|-------------|---------|
| `scope` | The purpose/category of the key | `cache`, `lock`, `session` |
| `entity` | What the key represents | `product`, `user`, `jobs` |
| `identifier` | Unique ID (usually UUID or composite) | `550e8400-...`, `cleanup`, `user123:endpoint` |

## Standard Scopes

### `cache:` — Cached query results and computed values
```
cache:product:{productId}              → single product cache
cache:products:list:{cursorHash}       → paginated product list
cache:dashboard:stats                  → global dashboard statistics
cache:settings:all                     → all dynamic settings
```

### `lock:` — Distributed locks
```
lock:jobs:cleanup                      → cleanup job lock (only one instance runs)
lock:order:{orderId}                   → order processing lock
lock:import:products                   → product import lock
```

### `session:` — User sessions and auth tokens
```
session:refresh:{userId}               → refresh token
session:verify:{token}                 → email/phone verification token
session:otp:{userId}                   → one-time password
```

### `rate:` — Rate limiting counters
```
rate:login:{ip}                        → login attempt counter per IP
rate:api:{userId}:{endpoint}           → API rate limit per user per endpoint
rate:email:{userId}                    → email sending rate limit
```

### `settings:` — Dynamic application settings
```
settings:{key}                         → single setting value
settings:cache:default-ttl             → cache default TTL setting
settings:feature:{featureName}         → feature flag
```

### `connections:` — Real-time connection tracking
```
connections:user:{userId}              → user's active SignalR connection IDs
connections:online                     → set of currently online user IDs
connections:typing:{channelId}         → users currently typing in a channel
```

## Naming Rules

### 1. Always lowercase
```
GOOD: cache:product:abc-123
BAD:  Cache:Product:ABC-123
```

### 2. Use colon `:` as separator
```
GOOD: cache:product:abc-123
BAD:  cache.product.abc-123
BAD:  cache/product/abc-123
BAD:  cache_product_abc-123
```

### 3. Use hyphens `-` within segments (not underscores)
```
GOOD: lock:jobs:daily-cleanup
BAD:  lock:jobs:daily_cleanup
```

### 4. UUIDs as-is (with hyphens)
```
GOOD: cache:product:550e8400-e29b-41d4-a716-446655440000
BAD:  cache:product:550e8400e29b41d4a716446655440000
```

### 5. Composite identifiers use colon
```
GOOD: rate:api:user123:get-products
BAD:  rate:api:user123-get-products
```

## Key Helper Class

Centralize key generation to prevent typos and enforce conventions:

```csharp
// Infrastructure/Redis/RedisKeys.cs
public static class RedisKeys
{
    // Cache
    public static string ProductCache(Guid id) => $"cache:product:{id}";
    public static string ProductListCache(string cursorHash) => $"cache:products:list:{cursorHash}";
    public static string DashboardStats() => "cache:dashboard:stats";
    public static string SettingsCache() => "cache:settings:all";

    // Locks
    public static string JobLock(string jobName) => $"lock:jobs:{jobName}";
    public static string OrderLock(Guid orderId) => $"lock:order:{orderId}";

    // Sessions
    public static string RefreshToken(Guid userId) => $"session:refresh:{userId}";
    public static string VerificationToken(string token) => $"session:verify:{token}";
    public static string Otp(Guid userId) => $"session:otp:{userId}";

    // Rate limiting
    public static string LoginAttempts(string ip) => $"rate:login:{ip}";
    public static string ApiRateLimit(Guid userId, string endpoint) => $"rate:api:{userId}:{endpoint}";

    // Settings
    public static string Setting(string key) => $"settings:{key}";
    public static string FeatureFlag(string name) => $"settings:feature:{name}";

    // Connections
    public static string UserConnections(Guid userId) => $"connections:user:{userId}";
    public static string OnlineUsers() => "connections:online";

    // Patterns (for SCAN / invalidation)
    public static string ProductCachePattern() => "cache:product:*";
    public static string ProductListCachePattern() => "cache:products:list:*";
    public static string AllCachePattern() => "cache:*";
    public static string UserSessionPattern(Guid userId) => $"session:*:{userId}";
}
```

### Usage

```csharp
// In a handler or service — ALWAYS use RedisKeys, never raw strings
var key = RedisKeys.ProductCache(productId);
var cached = await _redis.StringGetAsync(key);

// In cache invalidation
public IEnumerable<string> CacheKeysToInvalidate =>
[
    RedisKeys.ProductCache(Id),
    RedisKeys.ProductListCachePattern() // wildcard
];
```

## Wildcard Patterns for Bulk Operations

Redis supports glob-style patterns with `SCAN` (never `KEYS` in production):

```csharp
// Find all product cache keys
var server = _multiplexer.GetServer(_multiplexer.GetEndPoints().First());
await foreach (var key in server.KeysAsync(pattern: "cache:product:*"))
{
    await _redis.KeyDeleteAsync(key);
}

// Find all rate limit keys for a specific user
await foreach (var key in server.KeysAsync(pattern: $"rate:*:{userId}:*"))
{
    // process
}
```

### Pattern Syntax

| Pattern | Matches |
|---------|---------|
| `cache:product:*` | All product cache keys |
| `cache:*:list:*` | All list caches for any entity |
| `rate:login:192.168.*` | Login attempts from 192.168.x.x |
| `session:*` | All session keys regardless of type |

## Key Length Considerations

- Keep keys reasonably short — they consume memory
- UUID keys are fine (36 chars) — readability matters more than saving 10 bytes
- For high-cardinality sets (millions of keys), consider shorter prefixes: `c:` instead of `cache:`
- Document any abbreviations in this file if used

## Key Expiry Convention

Every scope has a default TTL range (see TTL Strategy for details):

| Scope | Default TTL Range |
|-------|-------------------|
| `cache:` | 15-60 minutes |
| `lock:` | 30 seconds - 5 minutes |
| `session:` | 15 minutes - 30 days |
| `rate:` | 1-5 minutes |
| `settings:` | 1-7 days |
| `connections:` | Managed by connect/disconnect events |
