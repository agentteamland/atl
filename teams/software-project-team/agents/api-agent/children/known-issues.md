---
knowledge-base-summary: "Issues discovered during real project scaffolding. EF Core Relational package must be explicit (transitive dependency unreliable in .NET 9). MinIO init container exits after bucket creation (expected, not a bug). Read before every new project."
---
# Known Issues & Lessons Learned

Issues discovered during real project scaffolding and development. These must be considered when creating new projects or features.

## EF Core Relational Package Must Be Explicit

**Discovered:** 2026-04-15 during reference-project scaffolding

**Problem:** `Npgsql.EntityFrameworkCore.PostgreSQL` should transitively bring `Microsoft.EntityFrameworkCore.Relational`, but in .NET 9 a version mismatch can occur causing `FileNotFoundException: Could not load file or assembly 'Microsoft.EntityFrameworkCore.Relational'` at startup during migration.

**Fix:** Always explicitly add `Microsoft.EntityFrameworkCore.Relational` to the Infrastructure project:

```xml
<PackageReference Include="Microsoft.EntityFrameworkCore.Relational" Version="9.*" />
```

**Apply when:** Every new project scaffolding. Never rely on transitive dependency for EF Core Relational.

---

## MinIO Init Container Shows as "Exited"

**Discovered:** 2026-04-15 during reference-project scaffolding

**Problem:** `wfm-minio-init` container appears as "exited" in Docker Desktop, which can cause concern. This is normal behavior — it's a one-shot init container that creates the default bucket and then exits.

**Fix:** Document in CLAUDE.md and README that `minio-init` is expected to exit after first run. Not a bug.

**Apply when:** Every project using MinIO/S3 file storage.

---

## Infrastructure Needs ASP.NET Core Framework Reference

**Discovered:** 2026-04-27 during a fresh project's cold build verification

**Problem:** `Infrastructure/Auth/CurrentUser.cs` uses `IHttpContextAccessor` (from `Microsoft.AspNetCore.Http`). On cold build, the .NET SDK can't resolve the namespace and the Infrastructure project fails to compile with:

```
error CS0234: The type or namespace name 'AspNetCore' does not exist in the namespace 'Microsoft'
error CS0246: The type or namespace name 'IHttpContextAccessor' could not be found
```

The errors block every dependent project (Application, Api, Socket, Worker — everything) and verify-system can't proceed.

**Fix:** Add `<FrameworkReference Include="Microsoft.AspNetCore.App" />` to `Infrastructure.csproj`:

```xml
<ItemGroup>
  <FrameworkReference Include="Microsoft.AspNetCore.App" />
</ItemGroup>
```

This brings in the full ASP.NET Core shared framework (which exposes `IHttpContextAccessor` and friends) without pulling individual NuGet packages.

**Apply when:** Every new project scaffolding. The convention is documented in `architecture-layers.md` § "Infrastructure .csproj — required package set".

---

## Api Needs EF Core Design Package (Startup Project Rule)

**Discovered:** 2026-04-27 during a fresh project's initial migration scaffold

**Problem:** Running `dotnet ef migrations add InitialCreate --project ../<Project>.Infrastructure --startup-project .` from the Api project fails with:

```
Your startup project '<Project>.Api' doesn't reference Microsoft.EntityFrameworkCore.Design.
This package is required for the Entity Framework Core Tools to work. Ensure your startup project is correct, install the package, and try again.
```

EF Core's design-time tools must be reachable from the **startup project** (the one running the EF command), not just the project containing the `DbContext`. Even if `Infrastructure` has the package, the tool searches the startup assembly.

**Fix:** Add `Microsoft.EntityFrameworkCore.Design` to `<Project>.Api.csproj` with `PrivateAssets`:

```xml
<PackageReference Include="Microsoft.EntityFrameworkCore.Design" Version="9.*">
  <IncludeAssets>runtime; build; native; contentfiles; analyzers; buildtransitive</IncludeAssets>
  <PrivateAssets>all</PrivateAssets>
</PackageReference>
```

**Apply when:** Every new project scaffolding. The convention is documented in `architecture-layers.md` § "Api .csproj — required package set".

---

## Worker SDK Doesn't Transitively Expose Host (.NET 9)

**Discovered:** 2026-04-27 during a fresh project's cold build verification

**Problem:** `Worker`, `LogIngest`, and `MailSender` use the `Microsoft.NET.Sdk.Worker` SDK and call `Host.CreateApplicationBuilder(args)` in `Program.cs`. On .NET 9 cold build, this fails:

```
error CS0103: The name 'Host' does not exist in the current context
```

In .NET 8 and earlier, the Worker SDK brought `Microsoft.Extensions.Hosting` transitively. **.NET 9 trimmed that transitive** — the SDK no longer exposes the `Host` static class without an explicit package reference.

**Fix:** Add `Microsoft.Extensions.Hosting` to each affected csproj:

```xml
<PackageReference Include="Microsoft.Extensions.Hosting" Version="9.*" />
```

Apply to all three Worker-SDK projects: `Worker`, `LogIngest`, `MailSender`.

**Apply when:** Every new project scaffolding on .NET 9 or later. Documented in `architecture-layers.md` § "Worker / LogIngest / MailSender".

---

## Initial EF Migration Required for Cold-Build Verification

**Discovered:** 2026-04-27 during a fresh project's verify-system run

**Problem:** A freshly scaffolded project has the `DbContext` and entity classes in place, but no migrations folder exists yet. When `docker compose up -d` brings up the API, EF auto-migrate has nothing to apply — the database stays empty (no `Users`, `AuditLogs`, or `__EFMigrationsHistory` tables). Verify-system Level 3 fails because login/register endpoints can't write to non-existent tables.

**Fix:** As the **last step of Phase 2** in `/create-new-project`, scaffold an initial migration before `docker compose up -d`:

```bash
docker compose run --rm api dotnet ef migrations add InitialCreate \
  --project ../<Project>.Infrastructure \
  --startup-project .
```

The migration files land in `Infrastructure/Persistence/Migrations/`. EF auto-migrate picks them up on next API startup.

**Note:** This requires the two prior fixes (Infrastructure framework reference + Api EF Design package) to already be in place — the migration scaffold itself uses EF design-time tools.

**Apply when:** Every new project scaffolding. The `/create-new-project` skill's Phase 2/3 boundary.
