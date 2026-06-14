---
knowledge-base-summary: "Creating, applying, and reverting migrations inside Docker. Naming convention: `{Timestamp}_{Description}`. Auto-migrate in Development, manual apply in Production. Handling merge conflicts in migration snapshots. Empty migrations for data seeding."
---
# Migration Management

## Golden Rule

**Migrations are ALWAYS created inside the Docker container.** Never use local dotnet tooling.

```bash
# Create a new migration
docker compose exec api dotnet ef migrations add {MigrationName} \
  --project ../ExampleApp.Infrastructure \
  --startup-project .

# Apply pending migrations
docker compose exec api dotnet ef database update \
  --project ../ExampleApp.Infrastructure \
  --startup-project .

# Revert to a specific migration
docker compose exec api dotnet ef database update {TargetMigrationName} \
  --project ../ExampleApp.Infrastructure \
  --startup-project .

# Remove the last migration (if not yet applied)
docker compose exec api dotnet ef migrations remove \
  --project ../ExampleApp.Infrastructure \
  --startup-project .

# Generate SQL script (for production review)
docker compose exec api dotnet ef migrations script \
  --project ../ExampleApp.Infrastructure \
  --startup-project . \
  --idempotent
```

## Naming Convention

Format: `{Timestamp}_{Description}`

EF Core auto-generates the timestamp. The description should be clear and concise:

| Pattern | Example |
|---------|---------|
| Add entity | `AddOrders` |
| Add column | `AddStatusToOrders` |
| Add index | `AddIndexOnOrdersCreatedAt` |
| Add relationship | `AddCustomerOrderRelationship` |
| Remove column | `RemoveTemporaryFieldFromUsers` |
| Rename column | `RenameUserNameToUsernameInUsers` |
| Add constraint | `AddUniqueConstraintOnOrderNumber` |
| Seed data | `SeedDefaultRoles` |

**Rules:**
- PascalCase, no spaces or special characters
- Verb + Noun pattern
- Be specific about what changed and where
- Never use generic names like `UpdateDatabase` or `FixStuff`

## Development Workflow

### Auto-Migrate in Development

In `Program.cs`, Development environment auto-applies migrations on startup:

```csharp
if (app.Environment.IsDevelopment())
{
    using var scope = app.Services.CreateScope();
    var db = scope.ServiceProvider.GetRequiredService<ApplicationDbContext>();
    await db.Database.MigrateAsync();
}
```

This means: create the migration, restart the container, schema is updated.

### Step-by-Step: New Migration

1. Make changes to entity / configuration
2. Rebuild the container: `docker compose up --build api`
3. Create migration:
   ```bash
   docker compose exec api dotnet ef migrations add AddStatusToOrders \
     --project ../ExampleApp.Infrastructure \
     --startup-project .
   ```
4. Review the generated migration file — check for unintended changes
5. Restart to auto-apply: `docker compose restart api`
6. Verify: check the database via Adminer (http://localhost:8083) or psql

### Step-by-Step: Fix a Bad Migration (Not Yet Applied in Prod)

1. Revert the migration:
   ```bash
   docker compose exec api dotnet ef database update {PreviousMigration} \
     --project ../ExampleApp.Infrastructure \
     --startup-project .
   ```
2. Remove the migration:
   ```bash
   docker compose exec api dotnet ef migrations remove \
     --project ../ExampleApp.Infrastructure \
     --startup-project .
   ```
3. Fix the entity/configuration
4. Create a new migration

## Production Workflow

### Never Auto-Migrate in Production

Production uses explicit migration scripts:

```bash
# Generate idempotent SQL script
docker compose exec api dotnet ef migrations script \
  --project ../ExampleApp.Infrastructure \
  --startup-project . \
  --idempotent \
  -o /tmp/migration.sql
```

The script is reviewed, approved, and applied manually or via CI/CD pipeline.

### Production Checklist

- [ ] Migration script generated and reviewed
- [ ] No data-destructive operations (column drops, type changes)
- [ ] Backward-compatible changes (add nullable column, add index)
- [ ] Estimated execution time for large tables
- [ ] Rollback script prepared
- [ ] Database backup taken before applying

## Handling Merge Conflicts

### ModelSnapshot Conflicts

The `ApplicationDbContextModelSnapshot.cs` file is the most common source of merge conflicts.

**Resolution strategy:**

1. Accept the incoming snapshot entirely (theirs)
2. Remove the conflicting migration
3. Recreate your migration on top of the latest snapshot

```bash
# Step 1: Accept their snapshot
git checkout --theirs src/ExampleApp.Infrastructure/Persistence/Migrations/ApplicationDbContextModelSnapshot.cs

# Step 2: Remove your migration files (the .cs and .Designer.cs)
rm src/.../Migrations/{YourMigration}.cs
rm src/.../Migrations/{YourMigration}.Designer.cs

# Step 3: Recreate your migration
docker compose exec api dotnet ef migrations add {YourMigrationName} \
  --project ../ExampleApp.Infrastructure \
  --startup-project .
```

### Conflicting Migrations with Same Timestamp

If two developers create migrations simultaneously, one will need to regenerate. The developer whose migration is not yet in `main` recreates theirs.

## Empty Migration for Data Seeding

Use empty migrations for seed data that must run in a specific order:

```bash
docker compose exec api dotnet ef migrations add SeedDefaultRoles \
  --project ../ExampleApp.Infrastructure \
  --startup-project .
```

Then edit the generated migration:

```csharp
public partial class SeedDefaultRoles : Migration
{
    protected override void Up(MigrationBuilder migrationBuilder)
    {
        migrationBuilder.Sql("""
            INSERT INTO roles (id, name, created_at)
            VALUES
                (gen_random_uuid(), 'Admin', now()),
                (gen_random_uuid(), 'User', now())
            ON CONFLICT (name) DO NOTHING;
        """);
    }

    protected override void Down(MigrationBuilder migrationBuilder)
    {
        migrationBuilder.Sql("""
            DELETE FROM roles WHERE name IN ('Admin', 'User');
        """);
    }
}
```

**Rules for seed migrations:**
- Always include `ON CONFLICT DO NOTHING` for idempotency
- Always implement `Down()` for reversibility
- Use raw SQL for seed data — EF's `HasData()` generates noisy diffs

## PostgreSQL Extensions (PostGIS, etc.)

Some features require Postgres extensions. The extension binaries ship with the Docker image (e.g., `imresamu/postgis:17-3.5-alpine` includes PostGIS), but the extension isn't active in the database until `CREATE EXTENSION` runs. Do this in a migration — not in seed.sql, not in application startup — so every environment (dev, CI, prod) gets it deterministically.

### PostGIS

If the project was scaffolded with Q4 "Spatial/GIS support = Yes" (see `/create-new-project`), the db image is `imresamu/postgis:*-alpine` and the FIRST migration must enable the extension before any geometry columns appear. Place it ahead of the initial schema so column types like `geography(Point,4326)` resolve:

```bash
# Inside the API container
docker compose exec api dotnet ef migrations add EnablePostgis \
  --project ../{ProjectName}.Infrastructure \
  --startup-project .
```

Edit the generated migration:

```csharp
public partial class EnablePostgis : Migration
{
    protected override void Up(MigrationBuilder migrationBuilder)
    {
        migrationBuilder.Sql("CREATE EXTENSION IF NOT EXISTS postgis;");
    }

    protected override void Down(MigrationBuilder migrationBuilder)
    {
        migrationBuilder.Sql("DROP EXTENSION IF EXISTS postgis;");
    }
}
```

**Why `IF NOT EXISTS`:** the migration must be idempotent — a restore-from-backup or a re-run on a partially-migrated DB shouldn't fail.

**EF Core side:** add `.HasPostgresExtension("postgis")` to `OnModelCreating` so the snapshot records it; then `NetTopologySuite` types (Point, LineString, Polygon) map directly to `geography`/`geometry` columns via `UseNetTopologySuite()` on the `NpgsqlDataSourceBuilder`.

### Other extensions

Same pattern — one migration per extension, `CREATE EXTENSION IF NOT EXISTS {name}` up, `DROP EXTENSION IF EXISTS {name}` down. Common candidates: `pg_trgm` (fuzzy text search), `pgcrypto` (hashing), `uuid-ossp` (UUID generation — usually unnecessary on Postgres 13+ since `gen_random_uuid()` is built in).

## Migration File Structure

```
src/{ProjectName}.Infrastructure/
  Persistence/
    Migrations/
      {Timestamp}_InitialCreate.cs
      {Timestamp}_InitialCreate.Designer.cs
      {Timestamp}_AddOrders.cs
      {Timestamp}_AddOrders.Designer.cs
      ApplicationDbContextModelSnapshot.cs    ← auto-generated, never edit manually
```

**Never manually edit `*.Designer.cs` or `ApplicationDbContextModelSnapshot.cs`.** These are auto-generated. Only edit the main migration file (`Up` / `Down` methods) when needed.
