---
knowledge-base-summary: "Two patterns: Dev (SDK + dotnet watch, source mounted) and Prod (multi-stage, restore→build→publish→runtime, non-root user). Layer caching optimization: copy .csproj first, restore, then copy source."
---
# Dockerfile Patterns

Two Dockerfile patterns exist: **Dev** (for local development with hot reload) and **Prod** (for production deployment with minimal image). Every .NET service has both.

## Dev Pattern: Dockerfile.dev

The dev Dockerfile is designed for `dotnet watch run`. Source code is NOT copied into the image — it is mounted via Docker Compose volumes at runtime. The Dockerfile only handles restore so that `dotnet watch` can start quickly.

### Template

```dockerfile
# src/{ProjectName}.{ServiceName}/Dockerfile.dev
FROM mcr.microsoft.com/dotnet/sdk:9.0

WORKDIR /src

# Copy ALL .csproj files + solution for restore (layer caching)
COPY src/{ProjectName}.Domain/*.csproj src/{ProjectName}.Domain/
COPY src/{ProjectName}.Application/*.csproj src/{ProjectName}.Application/
COPY src/{ProjectName}.Infrastructure/*.csproj src/{ProjectName}.Infrastructure/
COPY src/{ProjectName}.Logging/*.csproj src/{ProjectName}.Logging/
COPY src/{ProjectName}.Api/*.csproj src/{ProjectName}.Api/
COPY src/{ProjectName}.Socket/*.csproj src/{ProjectName}.Socket/
COPY src/{ProjectName}.Worker/*.csproj src/{ProjectName}.Worker/
COPY src/{ProjectName}.LogIngest/*.csproj src/{ProjectName}.LogIngest/
COPY src/{ProjectName}.MailSender/*.csproj src/{ProjectName}.MailSender/
COPY {SolutionName}.sln .

RUN dotnet restore

ENTRYPOINT ["dotnet", "watch", "run", "--project", "src/{ProjectName}.{ServiceName}/{ProjectName}.{ServiceName}.csproj"]
```

### Concrete Example (API)

```dockerfile
# src/ExampleApp.Api/Dockerfile.dev
FROM mcr.microsoft.com/dotnet/sdk:9.0

WORKDIR /src

COPY src/ExampleApp.Domain/*.csproj src/ExampleApp.Domain/
COPY src/ExampleApp.Application/*.csproj src/ExampleApp.Application/
COPY src/ExampleApp.Infrastructure/*.csproj src/ExampleApp.Infrastructure/
COPY src/ExampleApp.Logging/*.csproj src/ExampleApp.Logging/
COPY src/ExampleApp.Api/*.csproj src/ExampleApp.Api/
COPY src/ExampleApp.Socket/*.csproj src/ExampleApp.Socket/
COPY src/ExampleApp.Worker/*.csproj src/ExampleApp.Worker/
COPY src/ExampleApp.LogIngest/*.csproj src/ExampleApp.LogIngest/
COPY src/ExampleApp.MailSender/*.csproj src/ExampleApp.MailSender/
COPY ExampleApp.sln .

RUN dotnet restore

ENTRYPOINT ["dotnet", "watch", "run", "--project", "src/ExampleApp.Api/ExampleApp.Api.csproj", "--urls", "http://0.0.0.0:3000"]
```

### Why Copy ALL .csproj Files?

Even though this Dockerfile is for the API, we copy every project's .csproj. The reason: `dotnet restore` resolves the entire solution dependency graph. If any .csproj is missing, restore fails. Copying all of them once creates a cached layer that only invalidates when a project reference changes (rare).

### Why Source Is NOT Copied

In the dev pattern, source code is mounted as a volume in docker-compose.yml:

```yaml
volumes:
  - ./src:/src/src                    # Real source code from host
  - ./ExampleApp.sln:/src/ExampleApp.sln
```

This means:
- Edit code locally, container sees changes immediately
- `dotnet watch` detects the change and rebuilds
- No `docker compose build` needed for code changes
- Only `docker compose build` when a .csproj or Dockerfile changes

### Web Host vs Non-Web Host

For services that expose HTTP (API, Socket), add `--urls` to the entrypoint:

```dockerfile
# Web host (API, Socket)
ENTRYPOINT ["dotnet", "watch", "run", "--project", "src/ExampleApp.Api/ExampleApp.Api.csproj", "--urls", "http://0.0.0.0:3000"]
```

For background services (Worker, Consumer), no `--urls` needed:

```dockerfile
# Non-web host (Worker, LogIngest, MailSender)
ENTRYPOINT ["dotnet", "watch", "run", "--project", "src/ExampleApp.Worker/ExampleApp.Worker.csproj"]
```

---

## Prod Pattern: Dockerfile (Multi-Stage)

The production Dockerfile is a multi-stage build optimized for small, secure images. Source code IS copied and compiled inside the image. No volumes, no SDK in the final image.

### Template

```dockerfile
# src/{ProjectName}.{ServiceName}/Dockerfile
# ──── Stage 1: Build ────────────────────────────────────────────
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /src

# Copy .csproj files first (layer cache for restore)
COPY src/{ProjectName}.Domain/*.csproj src/{ProjectName}.Domain/
COPY src/{ProjectName}.Application/*.csproj src/{ProjectName}.Application/
COPY src/{ProjectName}.Infrastructure/*.csproj src/{ProjectName}.Infrastructure/
COPY src/{ProjectName}.Logging/*.csproj src/{ProjectName}.Logging/
COPY src/{ProjectName}.{ServiceName}/*.csproj src/{ProjectName}.{ServiceName}/
COPY {SolutionName}.sln .

RUN dotnet restore "src/{ProjectName}.{ServiceName}/{ProjectName}.{ServiceName}.csproj"

# Copy all source code
COPY src/ src/

# Publish (Release, self-contained=false, trimmed output)
RUN dotnet publish "src/{ProjectName}.{ServiceName}/{ProjectName}.{ServiceName}.csproj" \
    -c Release \
    -o /app/publish \
    --no-restore

# ──── Stage 2: Runtime ──────────────────────────────────────────
FROM mcr.microsoft.com/dotnet/aspnet:9.0-alpine AS runtime
WORKDIR /app

# Security: run as non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

COPY --from=build /app/publish .

EXPOSE {PORT}

ENTRYPOINT ["dotnet", "{ProjectName}.{ServiceName}.dll"]
```

### Concrete Example (API)

```dockerfile
# src/ExampleApp.Api/Dockerfile
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /src

COPY src/ExampleApp.Domain/*.csproj src/ExampleApp.Domain/
COPY src/ExampleApp.Application/*.csproj src/ExampleApp.Application/
COPY src/ExampleApp.Infrastructure/*.csproj src/ExampleApp.Infrastructure/
COPY src/ExampleApp.Logging/*.csproj src/ExampleApp.Logging/
COPY src/ExampleApp.Api/*.csproj src/ExampleApp.Api/
COPY ExampleApp.sln .

RUN dotnet restore "src/ExampleApp.Api/ExampleApp.Api.csproj"

COPY src/ src/

RUN dotnet publish "src/ExampleApp.Api/ExampleApp.Api.csproj" \
    -c Release \
    -o /app/publish \
    --no-restore

FROM mcr.microsoft.com/dotnet/aspnet:9.0-alpine AS runtime
WORKDIR /app

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

COPY --from=build /app/publish .

EXPOSE 3000

ENTRYPOINT ["dotnet", "ExampleApp.Api.dll"]
```

### Non-Web Host Production Dockerfile

For services without HTTP (Worker, LogIngest, MailSender), use the base runtime image instead of aspnet:

```dockerfile
# Stage 2 for non-web hosts:
FROM mcr.microsoft.com/dotnet/runtime:9.0-alpine AS runtime
# No EXPOSE needed
```

---

## Layer Caching: Why .csproj First Matters

Docker builds layers top-to-bottom. Each `COPY` or `RUN` creates a new layer. If a layer changes, all subsequent layers are invalidated.

```
Layer 1: FROM sdk:9.0              → cached (almost always)
Layer 2: COPY *.csproj             → cached unless project references change
Layer 3: RUN dotnet restore        → cached if Layer 2 is cached
Layer 4: COPY src/ src/            → invalidated on every code change
Layer 5: RUN dotnet publish        → invalidated because Layer 4 changed
```

By copying .csproj files and restoring BEFORE copying source code, we ensure that NuGet restore only runs when project references change. Source code changes (the most frequent case) only invalidate layers 4-5, skipping the expensive restore step.

**Bad pattern (invalidates restore on every code change):**

```dockerfile
# DO NOT DO THIS
COPY . .
RUN dotnet restore
RUN dotnet publish
```

---

## .dockerignore

Every project should have a `.dockerignore` at the repository root:

```
# .dockerignore
**/bin/
**/obj/
**/.vs/
**/.vscode/
**/node_modules/
**/.git/
**/.gitignore
**/docker-compose*.yml
**/.env
**/.env.*
!**/.env.example
**/README.md
**/LICENSE
**/*.md
**/Dockerfile*
```

This prevents unnecessary files from being sent to the Docker daemon during `docker build`, speeding up the build and reducing context size.

---

## Image Selection Reference

| Use Case | Image | Size |
|----------|-------|------|
| Dev (dotnet watch) | `mcr.microsoft.com/dotnet/sdk:9.0` | ~900MB |
| Prod web host | `mcr.microsoft.com/dotnet/aspnet:9.0-alpine` | ~100MB |
| Prod non-web host | `mcr.microsoft.com/dotnet/runtime:9.0-alpine` | ~90MB |

Always use `-alpine` for production. The SDK image is only needed for building and dev — never ship it to production.

---
