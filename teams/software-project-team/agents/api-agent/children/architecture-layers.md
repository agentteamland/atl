---
knowledge-base-summary: "Domain (pure C#, no deps), Application (Mediator, FluentValidation, pipeline behaviors), Infrastructure (EF Core, Auth, RMQ, Redis), Api (composition root, Minimal API endpoints). Vertical Slice structure: Command + Handler + Validator in same folder."
---
# Architecture Layers (Detail)

## Domain
- Pure C#, no NuGet dependencies
- Entities derive from `BaseEntity` (Id, CreatedAt, UpdatedAt)
- `IAuditableEntity` (CreatedBy, ModifiedBy) is supported
- Enums, ValueObjects, Domain Exceptions live here
- Domain Events are defined (MediatR INotification)

## Application
- Mediator (MediatR or martinothamar/Mediator — pick one; MediatR is the default in our scaffolds)
- FluentValidation (AbstractValidator<TCommand>)
- Pipeline Behaviors order: Logging → UnhandledException → Validation → Performance
- `Common/Interfaces/` — all abstractions live here
- `Common/Behaviors/` — cross-cutting pipeline behaviors
- `Common/Exceptions/` — NotFoundException, ValidationException, ForbiddenException
- `Common/Mail/` — EmailJob contract (shared with the consumer)
- `Features/{Feature}/` — Vertical Slices

### Application .csproj — required package set

Application expresses the DB-facing interface as `IApplicationDbContext { DbSet<User> Users; ... }`. That means Application DOES depend on EF Core's abstractions (it imports `Microsoft.EntityFrameworkCore` to get `DbSet<>`). This is the pragmatic choice we ship. Pure-clean variants hide `DbSet` behind `IQueryable<T>` — we do not.

Consequence: Application.csproj must reference:

```xml
<ItemGroup>
  <PackageReference Include="MediatR" Version="12.*" />
  <PackageReference Include="FluentValidation" Version="11.*" />
  <PackageReference Include="FluentValidation.DependencyInjectionExtensions" Version="11.*" />
  <PackageReference Include="Microsoft.EntityFrameworkCore" Version="9.*" />
  <PackageReference Include="Microsoft.Extensions.DependencyInjection.Abstractions" Version="9.*" />
  <PackageReference Include="Microsoft.Extensions.Logging.Abstractions" Version="9.*" />
  <PackageReference Include="Microsoft.Extensions.Localization.Abstractions" Version="9.*" />
</ItemGroup>
```

- **`FluentValidation.DependencyInjectionExtensions`** is required for the `AddValidatorsFromAssembly(...)` extension. The base `FluentValidation` package does NOT include DI wiring.
- **`Microsoft.EntityFrameworkCore`** is required because `IApplicationDbContext` uses `DbSet<>`. If you go the IQueryable-only route, you can drop this — but then you lose `.Entry()`, `SaveChangesAsync` on the interface, etc.

## Infrastructure
- EF Core + PostgreSQL (Npgsql)
- `Persistence/ApplicationDbContext.cs` — `IApplicationDbContext` implementation
- `Persistence/Configurations/` — `IEntityTypeConfiguration<T>` files
- Audit trail + soft-delete logic lives **inside the DbContext's `SaveChangesAsync` override**, NOT as separate Singleton interceptor classes. See `audit-trail.md` for the pattern — constructor-injected `ICurrentUser` (Scoped) + DbContext (Scoped) = lifetimes align cleanly. Registering interceptors as Singletons that consume Scoped services fails DI validation.
- `Auth/` — JWT, BCrypt, Redis token stores
- `Messaging/` — RabbitMQ connection, producers, `RmqLogPublisher : BackgroundService`
- `Services/` — external service implementations
- `DependencyInjection.cs` — all Infrastructure DI registration

### Infrastructure .csproj — required package set

Because Infrastructure hosts a `BackgroundService` (the log publisher), reads options from config, AND implements `ICurrentUser` via `IHttpContextAccessor`, it needs both hosting abstractions and an ASP.NET Core framework reference:

```xml
<ItemGroup>
  <FrameworkReference Include="Microsoft.AspNetCore.App" />
</ItemGroup>

<ItemGroup>
  <PackageReference Include="Microsoft.EntityFrameworkCore.Relational" Version="9.*" />
  <PackageReference Include="Microsoft.EntityFrameworkCore.Design" Version="9.*">
    <IncludeAssets>runtime; build; native; contentfiles; analyzers; buildtransitive</IncludeAssets>
    <PrivateAssets>all</PrivateAssets>
  </PackageReference>
  <PackageReference Include="Npgsql.EntityFrameworkCore.PostgreSQL" Version="9.*" />
  <PackageReference Include="Microsoft.Extensions.Hosting" Version="9.*" />
  <PackageReference Include="Microsoft.Extensions.Options.ConfigurationExtensions" Version="9.*" />
  <PackageReference Include="StackExchange.Redis" Version="2.*" />
  <PackageReference Include="AWSSDK.S3" Version="3.7.*" />     <!-- preferred over the `Minio` package; works against MinIO and real S3 -->
  <PackageReference Include="RabbitMQ.Client" Version="6.*" /> <!-- 6.x sync API; 7.x is async-first and NOT compatible -->
  <PackageReference Include="BCrypt.Net-Next" Version="4.*" />
  <PackageReference Include="Microsoft.IdentityModel.Tokens" Version="8.*" />
  <PackageReference Include="System.IdentityModel.Tokens.Jwt" Version="8.*" />
</ItemGroup>
```

- **`<FrameworkReference Include="Microsoft.AspNetCore.App" />`** — required because `Infrastructure/Auth/CurrentUser.cs` uses `IHttpContextAccessor` (from `Microsoft.AspNetCore.Http`). Without this, cold build fails with `CS0234: namespace 'AspNetCore' does not exist in 'Microsoft'`. The framework reference brings in the full ASP.NET Core shared framework, which exposes `IHttpContextAccessor` along with all related abstractions. The clean-arch alternative (move `CurrentUser` to Api, expose `ICurrentUser` interface from Infrastructure) is fine architecturally but the team's convention places `CurrentUser` in `Infrastructure/Auth/` to keep auth implementation co-located.
- **`Microsoft.Extensions.Hosting`** — required for `BackgroundService` (used by `RmqLogPublisher`). Without it, Infrastructure fails to compile.
- **`Microsoft.Extensions.Options.ConfigurationExtensions`** — required for `.Bind()` on config sections in DI registration.
- **`AWSSDK.S3` over `Minio`** — the AWS SDK is the canonical choice because it talks to both MinIO and real S3 with the same code. The `Minio` package's API has churned between 5.x and 6.x (e.g., `ListObjectsAsync` → `ListObjectsEnumAsync`), which creates scaffold drift.
- **`RabbitMQ.Client` 6.x** — stick to 6.x sync API. The 7.x line is async-first and not drop-in-compatible with our patterns.

## Api (Host)
- Minimal API endpoints (`Endpoints/{Feature}Endpoints.cs`)
- `Program.cs` — composition root (DI, middleware, pipeline order)
- Global exception handler (`IExceptionHandler` implementation)
- Rate limiting (per endpoint)
- JWT Bearer authentication setup
- Internal token validation (X-Internal-Token) for system-to-system
- EF auto-migrate (Development only)
- Serilog + RMQ log pipeline

### Api .csproj — required package set

Because Api is the **startup project** for `dotnet ef` migration tooling, it needs the EF Design package even though Infrastructure also has it. EF Design must be on the project that runs the tool, not just the project containing the `DbContext`:

```xml
<ItemGroup>
  <PackageReference Include="Microsoft.EntityFrameworkCore.Design" Version="9.*">
    <IncludeAssets>runtime; build; native; contentfiles; analyzers; buildtransitive</IncludeAssets>
    <PrivateAssets>all</PrivateAssets>
  </PackageReference>
  <PackageReference Include="Microsoft.AspNetCore.Authentication.JwtBearer" Version="9.*" />
  <PackageReference Include="Swashbuckle.AspNetCore" Version="6.*" />
  <PackageReference Include="Serilog.AspNetCore" Version="8.*" />
  <PackageReference Include="Mediator.SourceGenerator" Version="3.*" />
</ItemGroup>
```

- **`Microsoft.EntityFrameworkCore.Design` on Api**: without it, `dotnet ef migrations add` fails with `Your startup project doesn't reference Microsoft.EntityFrameworkCore.Design`. Cold-build verification stalls because the initial migration can't be scaffolded.

## Worker / LogIngest / MailSender (`Microsoft.NET.Sdk.Worker` projects)

These three projects use the Worker SDK and call `Host.CreateApplicationBuilder(args)` in `Program.cs`. **In .NET 9 the Worker SDK does NOT transitively expose `Host`** — projects must explicitly reference `Microsoft.Extensions.Hosting`:

```xml
<Project Sdk="Microsoft.NET.Sdk.Worker">
  <ItemGroup>
    <PackageReference Include="Microsoft.Extensions.Hosting" Version="9.*" />
    <!-- … other refs … -->
  </ItemGroup>
</Project>
```

Without this reference, cold build fails with `CS0103: The name 'Host' does not exist in the current context`. This was a working transitive in earlier .NET versions; .NET 9's Worker SDK trimmed it. Affects `Worker`, `LogIngest`, `MailSender` — all three host-builder-based services.

## Vertical Slice Structure

```
Application/Features/{FeatureName}/
├── Commands/
│   └── {Action}/
│       ├── {Action}Command.cs      → record : IRequest<{Action}Response>
│       ├── {Action}Handler.cs      → IRequestHandler<Command, Response>
│       └── {Action}Validator.cs    → AbstractValidator<Command> (MANDATORY)
├── Queries/
│   └── {Query}/
│       ├── {Query}Query.cs         → record : IRequest<{Query}Response>
│       └── {Query}Handler.cs       → IRequestHandler<Query, Response>
└── Common/
    └── {Feature}Dto.cs             → shared DTOs (optional)
```

