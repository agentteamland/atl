---
knowledge-base-summary: "maxmemory configuration and eviction policies (allkeys-lru recommended for cache workloads). Memory monitoring via INFO memory. Key size estimation rules. Avoiding large values (>1MB). SCAN instead of KEYS in production. Memory optimization patterns: compression, shorter keys for high-cardinality sets."
---
# Memory Management

Redis stores everything in memory. Without careful management, Redis can consume all available RAM, get OOM-killed, or evict critical keys. This document covers configuration, monitoring, and optimization.

## maxmemory Configuration

### Docker Compose

```yaml
# docker-compose.yml
redis:
  image: redis:7-alpine
  command: >
    redis-server
    --maxmemory 256mb
    --maxmemory-policy allkeys-lru
    --save ""
    --appendonly no
  ports:
    - "6379:6379"
  volumes:
    - redis_data:/data
```

### Configuration Options

| Setting | Dev | Production | Reason |
|---------|-----|------------|--------|
| `maxmemory` | 256mb | 1-4gb | Based on workload, never more than 75% of system RAM |
| `maxmemory-policy` | `allkeys-lru` | `allkeys-lru` | Evict least recently used keys when full |
| `save` | `""` (disabled) | `""` (disabled) | No RDB snapshots — Redis is cache, not storage |
| `appendonly` | `no` | `no` | No AOF — data is ephemeral |

### Why `allkeys-lru`?

Since Redis is used only for ephemeral data (cache, locks, sessions), all keys are eviction candidates. `allkeys-lru` evicts the least recently used key when memory is full, regardless of whether it has a TTL.

| Policy | Behavior | When to Use |
|--------|----------|-------------|
| `allkeys-lru` | Evict any LRU key | Cache workloads (our default) |
| `volatile-lru` | Evict LRU among keys with TTL | Mixed persistent + cache (not our case) |
| `allkeys-lfu` | Evict least frequently used | When frequency matters more than recency |
| `noeviction` | Return error on writes | When data loss is unacceptable (not for cache) |

## Memory Monitoring

### INFO Memory Command

```csharp
// Infrastructure/Redis/RedisHealthCheck.cs
public class RedisHealthCheck : IHealthCheck
{
    private readonly IConnectionMultiplexer _redis;

    public async Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context,
        CancellationToken ct = default)
    {
        try
        {
            var server = _redis.GetServer(_redis.GetEndPoints().First());
            var info = await server.InfoAsync("memory");

            var memorySection = info.FirstOrDefault(s => s.Key == "memory");
            if (memorySection.Key is null)
                return HealthCheckResult.Degraded("Cannot read memory info");

            var usedMemory = memorySection
                .FirstOrDefault(x => x.Key == "used_memory")?.Value;
            var maxMemory = memorySection
                .FirstOrDefault(x => x.Key == "maxmemory")?.Value;

            if (long.TryParse(usedMemory, out var used) &&
                long.TryParse(maxMemory, out var max) && max > 0)
            {
                var percentage = (double)used / max * 100;
                var data = new Dictionary<string, object>
                {
                    ["used_mb"] = used / 1024 / 1024,
                    ["max_mb"] = max / 1024 / 1024,
                    ["usage_percent"] = Math.Round(percentage, 2)
                };

                if (percentage > 90)
                    return HealthCheckResult.Unhealthy(
                        $"Redis memory at {percentage:F1}%", data: data);
                if (percentage > 75)
                    return HealthCheckResult.Degraded(
                        $"Redis memory at {percentage:F1}%", data: data);

                return HealthCheckResult.Healthy(
                    $"Redis memory at {percentage:F1}%", data: data);
            }

            return HealthCheckResult.Healthy("Redis is reachable");
        }
        catch (Exception ex)
        {
            return HealthCheckResult.Unhealthy("Redis unreachable", ex);
        }
    }
}
```

### Key Metrics to Monitor

| Metric | Where | Alert Threshold |
|--------|-------|-----------------|
| `used_memory` | INFO memory | >75% of maxmemory |
| `evicted_keys` | INFO stats | >0 consistently (means memory pressure) |
| `connected_clients` | INFO clients | >100 (possible connection leak) |
| `keyspace_hits` / `keyspace_misses` | INFO stats | Hit ratio <80% (caching is ineffective) |
| `mem_fragmentation_ratio` | INFO memory | >1.5 (fragmentation issue) |

## Key Size Estimation

### Rules of Thumb

| Data Type | Size | Example |
|-----------|------|---------|
| Simple string key | ~100 bytes overhead + value | `cache:product:uuid` = ~150 bytes |
| JSON value (small) | 200-500 bytes | Serialized ProductResponse |
| JSON value (medium) | 1-5 KB | Serialized list of 20 items |
| Hash (5 fields) | ~300 bytes + field data | User session |
| Set member | ~100 bytes per member | Online user ID |

### Size Checking

```csharp
// Check memory usage of a specific key
var server = _redis.GetServer(_redis.GetEndPoints().First());
var memoryUsage = await _redis.ExecuteAsync("MEMORY", "USAGE", key);
_logger.LogDebug("Key {Key} uses {Bytes} bytes", key, memoryUsage);
```

## Avoiding Large Values

### The 1MB Rule
Never store values larger than 1MB in Redis. Large values cause:
- Slow serialization/deserialization
- Network latency spikes
- Memory fragmentation
- Blocks other operations (Redis is single-threaded)

### What to Do Instead

```csharp
// BAD — storing a huge list as a single key
var allProducts = await _db.Products.ToListAsync();
await _redis.StringSetAsync("cache:products:all", JsonSerializer.Serialize(allProducts));
// This could be 10MB+

// GOOD — store individual items and a reference set
foreach (var product in products)
{
    await _redis.StringSetAsync(
        RedisKeys.ProductCache(product.Id),
        JsonSerializer.Serialize(product),
        TimeSpan.FromMinutes(30)
    );
}
```

### Compression for Medium-Large Values

```csharp
// For values between 10KB-1MB, consider compression
public static class RedisCompression
{
    public static byte[] Compress(string json)
    {
        var bytes = Encoding.UTF8.GetBytes(json);
        using var output = new MemoryStream();
        using (var gzip = new GZipStream(output, CompressionLevel.Fastest))
        {
            gzip.Write(bytes, 0, bytes.Length);
        }
        return output.ToArray();
    }

    public static string Decompress(byte[] compressed)
    {
        using var input = new MemoryStream(compressed);
        using var gzip = new GZipStream(input, CompressionMode.Decompress);
        using var output = new MemoryStream();
        gzip.CopyTo(output);
        return Encoding.UTF8.GetString(output.ToArray());
    }
}
```

## SCAN Instead of KEYS

**NEVER use `KEYS *` in production.** It blocks Redis (single-threaded) and scans every key.

```csharp
// BAD — blocks Redis for the entire scan
var keys = server.Keys(pattern: "cache:*"); // This calls KEYS internally on small datasets

// GOOD — uses SCAN with cursor-based iteration
await foreach (var key in server.KeysAsync(pattern: "cache:product:*", pageSize: 100))
{
    await _redis.KeyDeleteAsync(key);
}
```

### SCAN Configuration

```csharp
// pageSize controls how many keys are returned per SCAN iteration
// Lower = less blocking per iteration, more round trips
// Higher = more blocking per iteration, fewer round trips
// Default: 250, recommended: 100-500

await foreach (var key in server.KeysAsync(
    pattern: "cache:*",
    pageSize: 100,     // keys per SCAN iteration
    database: 0))
{
    // process key
}
```

## Memory Optimization Patterns

### 1. Avoid Storing Redundant Data
```csharp
// BAD — caching the full entity with all relations
await _redis.StringSetAsync(key, JsonSerializer.Serialize(productWithAllRelations));

// GOOD — cache only the response DTO (minimal data)
await _redis.StringSetAsync(key, JsonSerializer.Serialize(productResponse));
```

### 2. Use Appropriate Data Structures
```csharp
// BAD — storing a list of IDs as a JSON array in a String
await _redis.StringSetAsync("online-users", "[\"id1\",\"id2\",\"id3\"]");

// GOOD — use a Set (less memory, O(1) operations)
await _redis.SetAddAsync("connections:online", userId.ToString());
```

### 3. Shorter Keys for High-Cardinality Sets
When you have millions of keys with the same prefix, consider abbreviating:

```csharp
// Standard (fine for thousands of keys)
"cache:product:550e8400-e29b-41d4-a716-446655440000"  // 52 chars

// Abbreviated (for millions of keys — saves ~20 bytes per key)
"c:p:550e8400-e29b-41d4-a716-446655440000"  // 42 chars

// Document abbreviations if used
```

### 4. Pipeline Bulk Operations
```csharp
// BAD — N round trips
foreach (var id in productIds)
{
    await _redis.KeyDeleteAsync(RedisKeys.ProductCache(id));
}

// GOOD — single pipeline, single round trip
var batch = _redis.CreateBatch();
var tasks = productIds
    .Select(id => batch.KeyDeleteAsync(RedisKeys.ProductCache(id)))
    .ToArray();
batch.Execute();
await Task.WhenAll(tasks);
```

## Redis Memory Troubleshooting

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Memory keeps growing | Keys without TTL | Audit with `OBJECT ENCODING key`, add TTLs |
| Sudden memory spike | Large value stored | Check `MEMORY USAGE key` for big keys |
| High fragmentation | Many small allocs/frees | Restart Redis (cache is ephemeral) |
| Evicted keys > 0 | Memory limit reached | Increase maxmemory or reduce TTLs |
| Slow operations | Large key or KEYS command | Use SCAN, avoid large values |
