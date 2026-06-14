---
knowledge-base-summary: "Two-layer configuration: static (appsettings = defaults) + dynamic (DB → Redis = override). Redis-only, no dictionary cache, no RMQ, no periodic refresh. Set: API endpoint → write DB + Redis → instant propagation. Get: read Redis → fallback DB → fallback appsettings. Other hosts access via API endpoints."
---
# Dynamic Settings: DB + Redis Centralized Configuration System

## Philosophy

Application settings are two-layered:

**Layer 1 — Static (appsettings / .env):** Things required for the application to start up. Determined at deploy-time, does not change at runtime.
- Port numbers, connection strings, JWT secret, RMQ/Redis/ES URLs

**Layer 2 — Dynamic (DB + Redis):** Everything that can change while the application is running. Does not require redeployment.
- Cache TTL durations, rate limit values, feature flags, mail settings, pagination defaults, file upload limits, maintenance mode, etc.

**Rule:** Static settings are the default values. DB settings override them. If a new setting does not exist in the DB, the default from appsettings is used.

## Flow

```
Set: Admin panel → API endpoint → write to DB + write to Redis → done
Get: Anywhere → read from Redis → done
```

- No RMQ, no dictionary cache, no periodic refresh
- All pods read the same Redis — synchronization is automatic
- Both DB and Redis are updated at set time — instant propagation

## SystemSetting Entity

```csharp
public class SystemSetting : BaseEntity
{
    public required string Key { get; set; }
    public required string Value { get; set; }
    public string ValueType { get; set; } = "string";  // string | int | bool | decimal | json
    public string? Description { get; set; }
    public string Category { get; set; } = "general";
}
```

**Key convention:** Category-based, `:` separated:
- `cache:product:ttl-minutes` → "30"
- `auth:login-attempt-limit` → "5"
- `auth:lockout-duration-minutes` → "15"
- `mail:from-address` → "noreply@example.com"
- `mail:bcc-address` → "archive@example.com"
- `general:maintenance-mode` → "false"
- `upload:max-file-size-mb` → "10"
- `pagination:default-page-size` → "20"

## ISettingsService (Application layer)

```csharp
public interface ISettingsService
{
    Task<T> GetAsync<T>(string key, CancellationToken ct = default);
    Task<T> GetAsync<T>(string key, T defaultValue, CancellationToken ct = default);
    Task SetAsync<T>(string key, T value, string? description = null, CancellationToken ct = default);
    Task<Dictionary<string, string>> GetByCategoryAsync(string category, CancellationToken ct = default);
}
```

## SettingsService (Infrastructure layer)

```csharp
public class SettingsService : ISettingsService
{
    private readonly IApplicationDbContext _db;
    private readonly IConnectionMultiplexer _redis;
    private readonly IConfiguration _configuration;
    private const string RedisPrefix = "settings:";

    public async Task<T> GetAsync<T>(string key, T defaultValue, CancellationToken ct = default)
    {
        // 1. Read from Redis
        var redisDb = _redis.GetDatabase();
        var cached = await redisDb.StringGetAsync($"{RedisPrefix}{key}");

        if (cached.HasValue)
            return Deserialize<T>(cached!);

        // 2. If not in Redis, fetch from DB and write to Redis
        var setting = await _db.SystemSettings
            .FirstOrDefaultAsync(s => s.Key == key, ct);

        if (setting != null)
        {
            await redisDb.StringSetAsync($"{RedisPrefix}{key}", setting.Value);
            return Deserialize<T>(setting.Value);
        }

        // 3. If not in DB either, default from appsettings
        var configValue = _configuration[key.Replace(":", "__")];
        if (configValue != null)
            return Deserialize<T>(configValue);

        // 4. If nowhere, use the provided default
        return defaultValue;
    }

    public async Task SetAsync<T>(string key, T value, string? description = null, CancellationToken ct = default)
    {
        var stringValue = Serialize(value);

        // 1. Write to DB (upsert)
        var setting = await _db.SystemSettings
            .FirstOrDefaultAsync(s => s.Key == key, ct);

        if (setting == null)
        {
            setting = new SystemSetting
            {
                Key = key,
                Value = stringValue,
                ValueType = typeof(T).Name.ToLowerInvariant(),
                Description = description,
                Category = key.Split(':')[0],
            };
            _db.SystemSettings.Add(setting);
        }
        else
        {
            setting.Value = stringValue;
        }

        await _db.SaveChangesAsync(ct);

        // 2. Write to Redis (instant propagation)
        var redisDb = _redis.GetDatabase();
        await redisDb.StringSetAsync($"{RedisPrefix}{key}", stringValue);
    }
}
```

## Seeding at Application Startup

When the API starts, it seeds default values from appsettings into the DB (if they don't exist), then loads all of them into Redis:

```csharp
// In Program.cs or a HostedService:
public async Task SeedSettingsAsync()
{
    var defaults = new Dictionary<string, (string Value, string Type, string Desc, string Category)>
    {
        ["cache:product:ttl-minutes"] = ("15", "int", "Product cache duration (minutes)", "cache"),
        ["cache:default-ttl-minutes"] = ("10", "int", "Default cache duration (minutes)", "cache"),
        ["auth:login-attempt-limit"] = ("5", "int", "Max login attempts", "auth"),
        ["auth:lockout-duration-minutes"] = ("15", "int", "Account lockout duration (minutes)", "auth"),
        ["pagination:default-page-size"] = ("20", "int", "Default page size", "pagination"),
        ["pagination:max-page-size"] = ("100", "int", "Max page size", "pagination"),
        ["upload:max-file-size-mb"] = ("10", "int", "Max file size (MB)", "upload"),
        ["general:maintenance-mode"] = ("false", "bool", "Maintenance mode", "general"),
        ["mail:from-address"] = ("noreply@example.com", "string", "Mail sender address", "mail"),
    };

    foreach (var (key, (value, type, desc, category)) in defaults)
    {
        // Add if not in DB (don't touch if exists — admin may have changed it)
        if (!await _db.SystemSettings.AnyAsync(s => s.Key == key))
        {
            _db.SystemSettings.Add(new SystemSetting
            {
                Key = key, Value = value, ValueType = type,
                Description = desc, Category = category,
            });
        }
    }
    await _db.SaveChangesAsync();

    // Load all into Redis
    var allSettings = await _db.SystemSettings.ToListAsync();
    var redisDb = _redis.GetDatabase();
    foreach (var s in allSettings)
    {
        await redisDb.StringSetAsync($"{RedisPrefix}{s.Key}", s.Value);
    }
}
```

## Usage in Handler

```csharp
public async Task<PaginatedResponse<ProductDto>> Handle(
    GetProductsQuery request, CancellationToken ct)
{
    // Get cache TTL from settings
    var ttl = await _settingsService.GetAsync<int>(
        "cache:product:ttl-minutes", defaultValue: 15, ct);

    _logger.LogInformation("Using product cache TTL: {TtlMinutes} minutes", ttl);

    // ... handler logic
}
```

## API Endpoints

```csharp
// GET /api/settings/{key} — read a single setting
// GET /api/settings?category=cache — list by category
// PUT /api/settings/{key} — update setting (admin)
// POST /api/settings/seed — re-seed defaults (admin)
```

All settings endpoints require admin authorization (`.RequireAuthorization(p => p.RequireRole("Admin"))`).

## Other Hosts (Socket, Worker, etc.)

Other hosts access settings via the API endpoint:

```csharp
// In Socket or Worker:
var response = await _apiClient.GetAsync<SettingResponse>("/api/settings/general:maintenance-mode");
```

This preserves the "API is the brain" principle — other hosts do not directly access DB or Redis.

## Cache Integration

Dynamic settings work in integration with cache behavior. TTL is read from settings in `ICacheable` queries:

```csharp
record GetProductQuery(Guid ProductId) : IRequest<ProductDto>, ICacheable
{
    public string CacheKey => $"product:{ProductId}";
    // TTL is read from dynamic settings — inside CachingBehavior
    public string? CacheTtlSettingKey => "cache:product:ttl-minutes";
}
```

`CachingBehavior` sees this key and retrieves the TTL via `ISettingsService.GetAsync<int>(settingKey)`.

