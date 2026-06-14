---
knowledge-base-summary: "SQL-file-based development seeding. Single `Seeds/seed.sql` file, append-only, idempotent INSERTs. Auto-executed on startup in Development. Reset-db endpoint for clean slate. Agent should ask \"does this feature need seed data?\" on every new feature — if yes, append to seed.sql."
---
# Seed Data: SQL-Based Development Seeding

## Philosophy

Development databases start empty. Seed data provides a known starting state — admin user, test tenant, sample data. The seed mechanism is **SQL-file-based**, not code-based. This makes it:
- Easy to append (just add INSERT statements)
- Transparent (read the SQL, know the data)
- Portable (works with any PostgreSQL tool)
- Flexible (project-specific data without code changes)

## Seed File Location

```
src/{ProjectName}.Api/Seeds/seed.sql
```

Single file, append-only. Grows as the project develops. Every INSERT is idempotent (safe to run multiple times).

## Idempotent INSERT Pattern

Every seed statement MUST be idempotent — running it twice produces the same result:

```sql
-- ✅ CORRECT: Insert only if not exists
INSERT INTO "Users" ("Id", "Email", "PasswordHash", "FirstName", "LastName", "EmailVerified", "CreatedAt")
SELECT 
    'a1b2c3d4-0000-0000-0000-000000000001',
    'admin@test.com',
    '$2a$11$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ012', -- password: Admin123!
    'Admin',
    'User',
    true,
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "Users" WHERE "Email" = 'admin@test.com'
);

-- ❌ WRONG: Plain INSERT — fails on second run
INSERT INTO "Users" ("Id", "Email", ...) VALUES (...);
```

## Fixed UUIDs for Seed Data

Use deterministic UUIDs for seed entities so they can be referenced across statements:

```sql
-- Convention: a1b2c3d4-0000-0000-0000-{sequential}
-- Users
-- 'a1b2c3d4-0000-0000-0000-000000000001' = admin user
-- 'a1b2c3d4-0000-0000-0000-000000000002' = test dealer owner
-- 'a1b2c3d4-0000-0000-0000-000000000003' = test customer

-- Tenants
-- 'b1b2c3d4-0000-0000-0000-000000000001' = test tenant (dealer)

-- Profiles
-- 'c1b2c3d4-0000-0000-0000-000000000001' = admin profile
-- 'c1b2c3d4-0000-0000-0000-000000000002' = dealer owner profile
-- 'c1b2c3d4-0000-0000-0000-000000000003' = customer profile
```

This way seed statements can reference each other reliably.

## Seed File Structure

```sql
-- =============================================================
-- SEED DATA — Development Only
-- This file is executed on API startup in Development mode.
-- All statements MUST be idempotent (safe to re-run).
-- =============================================================

-- ─── System Settings ─────────────────────────────────────────
-- (Dynamic settings defaults are seeded by SettingsService.SeedAsync)

-- ─── Users ───────────────────────────────────────────────────

INSERT INTO "Users" ("Id", "Email", "PasswordHash", "FirstName", "LastName", "EmailVerified", "CreatedAt")
SELECT 'a1b2c3d4-0000-0000-0000-000000000001', 'admin@test.com',
    '$2a$11$...', 'Admin', 'User', true, NOW()
WHERE NOT EXISTS (SELECT 1 FROM "Users" WHERE "Id" = 'a1b2c3d4-0000-0000-0000-000000000001');

-- ─── Tenants ─────────────────────────────────────────────────

-- (Project-specific: Dealer, Company, Seller, etc.)

-- ─── Profiles ────────────────────────────────────────────────

-- (Link users to tenants with roles)

-- ─── Sample Data ─────────────────────────────────────────────

-- (Project-specific: products, orders, etc.)
-- Add new seed data below this line ↓
```

## Startup Execution

In `Program.cs`, after migration:

```csharp
if (app.Environment.IsDevelopment())
{
    using var scope = app.Services.CreateScope();
    var db = scope.ServiceProvider.GetRequiredService<AppDbContext>();
    
    // 1. Run EF migrations
    await db.Database.MigrateAsync();
    
    // 2. Seed dynamic settings defaults
    var settings = scope.ServiceProvider.GetRequiredService<ISettingsService>();
    await settings.SeedDefaultsAsync();
    
    // 3. Run seed.sql if exists
    var seedPath = Path.Combine(AppContext.BaseDirectory, "Seeds", "seed.sql");
    if (File.Exists(seedPath))
    {
        var sql = await File.ReadAllTextAsync(seedPath);
        await db.Database.ExecuteSqlRawAsync(sql);
        logger.LogInformation("Seed data applied from {Path}", seedPath);
    }
}
```

**IMPORTANT:** Ensure `seed.sql` is copied to output directory. In `.csproj`:

```xml
<ItemGroup>
  <None Update="Seeds\seed.sql">
    <CopyToOutputDirectory>PreserveNewest</CopyToOutputDirectory>
  </None>
</ItemGroup>
```

## Reset-DB Endpoint

Development-only endpoint to reset the database to a clean, seeded state:

```csharp
// In HealthEndpoints.cs or a dedicated DevEndpoints.cs
if (app.Environment.IsDevelopment())
{
    app.MapPost("/api/internal/reset-db", async (
        AppDbContext db,
        ISettingsService settings,
        ILogger<Program> logger,
        CancellationToken ct) =>
    {
        logger.LogWarning("reset-db called — dropping and recreating database");

        // 1. Drop and recreate
        await db.Database.EnsureDeletedAsync(ct);
        await db.Database.MigrateAsync(ct);

        // 2. Seed settings
        await settings.SeedDefaultsAsync(ct);

        // 3. Run seed.sql
        var seedPath = Path.Combine(AppContext.BaseDirectory, "Seeds", "seed.sql");
        if (File.Exists(seedPath))
        {
            var sql = await File.ReadAllTextAsync(seedPath, ct);
            await db.Database.ExecuteSqlRawAsync(sql, ct);
        }

        logger.LogInformation("Database reset and seeded successfully");
        return Results.Ok(new { message = "Database reset and seeded", timestamp = DateTime.UtcNow });
    })
    .AddEndpointFilter<InternalSecretFilter>()
    .WithName("ResetDatabase")
    .WithDescription("Development only: drops DB, runs migrations, applies seed.sql");
}
```

## When to Add Seed Data

The API Agent should consider adding seed data when:

- [ ] New entity type created → add at least one sample record
- [ ] New role or user type → add a test user with that role
- [ ] New tenant type (SaaS) → add a test tenant + profile
- [ ] Feature requires reference data → add the reference records
- [ ] User explicitly asks for seed data
- [ ] Testing a flow requires specific data state

## How to Add Seed Data

Append to the bottom of `seed.sql`:

```sql
-- ─── {Feature Name} (added {date}) ──────────────────────────

INSERT INTO "Products" ("Id", "Name", "Price", "TenantId", "CreatedAt")
SELECT 
    'd1b2c3d4-0000-0000-0000-000000000001',
    'Sample Product',
    29.99,
    'b1b2c3d4-0000-0000-0000-000000000001',  -- test tenant
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM "Products" WHERE "Id" = 'd1b2c3d4-0000-0000-0000-000000000001'
);
```

**Rules:**
1. **Always append, never modify existing entries** — other devs might depend on existing seed IDs
2. **Always idempotent** — `WHERE NOT EXISTS` or `ON CONFLICT DO NOTHING`
3. **Comment with feature name and date** — know when and why it was added
4. **Use fixed UUIDs** — deterministic, referenceable across statements
5. **Passwords in comments** — note the plain text password next to the hash for dev convenience

## Test Credentials (Convention)

```
Admin:     admin@test.com     / Admin123!
Dealer:    dealer@test.com    / Dealer123!
Customer:  customer@test.com  / Customer123!
```

Same credentials in every project for dev convenience. Never used outside development.

## Important Rules

1. **seed.sql is Development ONLY.** Never runs in Staging or Production.
2. **reset-db is Development ONLY.** Protected by InternalSecretFilter AND environment check.
3. **Idempotent or bust.** If the INSERT isn't idempotent, it will break on second `compose up`.
4. **Agent responsibility:** When creating a new feature, always ask "does this need seed data?" If yes, append to seed.sql.
5. **seed.sql is committed to git.** It's part of the project, not a secret. (No real passwords, only test hashes.)
