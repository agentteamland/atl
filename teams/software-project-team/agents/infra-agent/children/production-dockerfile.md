---
knowledge-base-summary: "Multi-stage build: SDK restore+publish → runtime image. Security: non-root user, no shell, minimal base image. Health endpoint exposed. Environment-configurable. Image size optimization."
---
# Production Dockerfile

Multi-stage Docker build for production .NET 9 services. Small, secure, fast.

## Philosophy

- **Multi-stage**: SDK for building, aspnet runtime for running
- **Non-root**: Application runs as `appuser`, not root
- **Minimal**: No SDK, no shell abuse, exec-form entrypoint
- **Layer-optimized**: .csproj copied first, restore cached, then source

## Complete Production Dockerfile

```dockerfile
# ============================================================
# Stage 1: Restore (cached unless .csproj files change)
# ============================================================
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS restore
WORKDIR /src

# Copy solution and project files first (layer cache optimization)
COPY ExampleApp.sln ./
COPY src/ExampleApp.Domain/*.csproj             src/ExampleApp.Domain/
COPY src/ExampleApp.Application/*.csproj         src/ExampleApp.Application/
COPY src/ExampleApp.Infrastructure/*.csproj       src/ExampleApp.Infrastructure/
COPY src/ExampleApp.Logging/*.csproj             src/ExampleApp.Logging/
COPY src/ExampleApp.Api/*.csproj                 src/ExampleApp.Api/

# Restore NuGet packages (this layer is cached until a .csproj changes)
RUN dotnet restore src/ExampleApp.Api/ExampleApp.Api.csproj

# ============================================================
# Stage 2: Build & Publish
# ============================================================
FROM restore AS build
WORKDIR /src

# Copy ALL source code
COPY src/ src/

# Publish in Release mode
RUN dotnet publish src/ExampleApp.Api/ExampleApp.Api.csproj \
    -c Release \
    -o /app/publish \
    --no-restore

# ============================================================
# Stage 3: Runtime (final image)
# ============================================================
FROM mcr.microsoft.com/dotnet/aspnet:9.0 AS runtime
WORKDIR /app

# Install curl for health checks
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN adduser --disabled-password --gecos "" --home /app appuser

# Copy published output from build stage
COPY --from=build /app/publish .

# Set ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Configure ASP.NET
ENV ASPNETCORE_URLS=http://+:8080
ENV ASPNETCORE_ENVIRONMENT=Production
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -sf http://localhost:8080/health || exit 1

# Exec form entrypoint (no shell wrapping, proper signal handling)
ENTRYPOINT ["dotnet", "ExampleApp.Api.dll"]
```

## Base Image Choice — Debian vs Alpine

**The template above uses `mcr.microsoft.com/dotnet/aspnet:9.0` (Debian-based).** That is the default recommendation. Alpine (`-alpine` tag) is tempting for smaller images but has a sharp edge: it ships with **globalization-invariant mode enabled**, which breaks any app that uses real culture identifiers like `"en"` or `"tr"`.

If your project uses `AddLocalization()` / `UseRequestLocalization()` / `CultureInfo("en")` anywhere (every multi-language project does), Alpine will crash at startup with:

```
System.Globalization.CultureNotFoundException:
Only the invariant culture is supported in globalization-invariant mode.
```

Two ways to use Alpine safely:

### Option A — Install ICU + disable invariant mode (recommended if you need Alpine)

```dockerfile
FROM mcr.microsoft.com/dotnet/aspnet:9.0-alpine AS runtime
ENV DOTNET_SYSTEM_GLOBALIZATION_INVARIANT=false
RUN apk add --no-cache curl icu-libs icu-data-full
# ... rest of the stage ...
```

Both lines are required: the ENV flips the runtime switch, and `icu-libs` + `icu-data-full` provide the culture data the runtime expects.

### Option B — Stay on Debian (the template above)

If you don't have a specific reason to use Alpine (image size sensitivity, hardening policy), keep the Debian base. It has no globalization trap and Microsoft ships it as the default. The size difference (~60MB vs ~30MB compressed) is rarely worth the Alpine gotchas.

**Never deliver an Alpine-based host without the ICU + ENV fix for a project that ships multi-language UI.** The crash happens on `app.Run()`, so CI health checks don't catch it until the container actually tries to accept traffic.

## Stage-by-Stage Breakdown

### Stage 1: Restore

```dockerfile
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS restore
WORKDIR /src

COPY ExampleApp.sln ./
COPY src/ExampleApp.Domain/*.csproj             src/ExampleApp.Domain/
COPY src/ExampleApp.Application/*.csproj         src/ExampleApp.Application/
COPY src/ExampleApp.Infrastructure/*.csproj       src/ExampleApp.Infrastructure/
COPY src/ExampleApp.Logging/*.csproj             src/ExampleApp.Logging/
COPY src/ExampleApp.Api/*.csproj                 src/ExampleApp.Api/

RUN dotnet restore src/ExampleApp.Api/ExampleApp.Api.csproj
```

**Why copy .csproj files separately?**

Docker caches layers. If only C# source code changes (not package references), the restore layer stays cached. This saves 30-60 seconds per build.

### Stage 2: Build

```dockerfile
FROM restore AS build
COPY src/ src/
RUN dotnet publish ... -c Release -o /app/publish --no-restore
```

- Builds on top of the restore stage (packages already restored)
- `--no-restore` prevents redundant restore
- Output goes to `/app/publish` (clean, flat directory)

### Stage 3: Runtime

```dockerfile
FROM mcr.microsoft.com/dotnet/aspnet:9.0 AS runtime
```

- `aspnet` image, NOT `sdk` -- no build tools, no source code
- Roughly 220MB vs 900MB+ for SDK image
- Contains only the .NET runtime and ASP.NET Core shared framework

## Security: Non-Root User

```dockerfile
RUN adduser --disabled-password --gecos "" --home /app appuser
USER appuser
```

**Why?**

- Running as root inside a container means a container escape = root on host
- `adduser --disabled-password` creates a user that cannot log in via password
- `USER appuser` switches all subsequent commands (and the running process) to this user
- If an attacker breaks into the container, they're limited to appuser privileges

## Port Configuration

```dockerfile
ENV ASPNETCORE_URLS=http://+:8080
EXPOSE 8080
```

**Why 8080 instead of 80?**

- Non-root users cannot bind to ports below 1024
- Port 8080 is the convention for non-root containers
- `.NET 8+` defaults to port 8080 for non-root, but explicit is better

### Docker Compose Mapping

```yaml
# docker-compose.prod.yml
api:
  image: ghcr.io/owner/example-app-api:latest
  ports:
    - "80:8080"      # Host port 80 maps to container port 8080
    - "443:8080"     # Or behind a reverse proxy
```

## Entrypoint: Exec Form

```dockerfile
# CORRECT: exec form
ENTRYPOINT ["dotnet", "ExampleApp.Api.dll"]

# WRONG: shell form (wraps in /bin/sh -c, breaks signal handling)
# ENTRYPOINT dotnet ExampleApp.Api.dll
```

**Exec form** runs the process directly as PID 1 -- receives SIGTERM for graceful shutdown. **Shell form** wraps in `/bin/sh -c` which swallows signals, causing ungraceful 10-second timeout kills.

## Health Check

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -sf http://localhost:8080/health || exit 1
```

This requires a `/health` endpoint in the application:

```csharp
// In the API's endpoint registration
app.MapGet("/health", () => Results.Ok(new { status = "healthy" }))
    .WithTags("Health");
```

## .dockerignore

Create `.dockerignore` in the project root to exclude unnecessary files from the build context:

```
# .dockerignore
**/bin/
**/obj/
**/node_modules/
**/.git
**/.vs
**/.vscode
**/.idea
**/Dockerfile*
**/docker-compose*
**/*.md
**/LICENSE
**/.env
**/.env.*
**/coverage/
**/TestResults/
```

### Why .dockerignore Matters

- Build context is sent to Docker daemon -- large contexts = slow builds
- `bin/` and `obj/` from host conflict with container builds
- `.git/` directory can be hundreds of MB
- Smaller context = faster `COPY . .` = faster builds

## Image Size Comparison

| Image | Approximate Size |
|-------|-----------------|
| sdk:9.0 (build stage) | ~900 MB |
| aspnet:9.0 (runtime) | ~220 MB |
| aspnet:9.0-alpine (runtime) | ~110 MB |
| Final production image | ~230-250 MB |

### Alpine Variant

For smaller images, use Alpine base:

```dockerfile
FROM mcr.microsoft.com/dotnet/aspnet:9.0-alpine AS runtime

# Alpine uses adduser differently
RUN adduser -D -h /app appuser

# Alpine uses apk instead of apt
RUN apk add --no-cache curl
```

**Caveat**: Alpine uses musl libc, not glibc. Some NuGet packages with native dependencies may not work. Test thoroughly before switching.

## Per-Service Dockerfiles

Each .NET service gets its own production Dockerfile. They share the same pattern, only the project path and DLL name change.

### API Dockerfile

```
src/ExampleApp.Api/Dockerfile
ENTRYPOINT ["dotnet", "ExampleApp.Api.dll"]
```

### Worker Dockerfile

```
src/ExampleApp.Worker/Dockerfile
ENTRYPOINT ["dotnet", "ExampleApp.Worker.dll"]
# No EXPOSE -- Worker has no HTTP endpoint
# No HEALTHCHECK via curl -- use a different check or omit
```

### LogIngest Dockerfile

```
src/ExampleApp.LogIngest/Dockerfile
ENTRYPOINT ["dotnet", "ExampleApp.LogIngest.dll"]
# No EXPOSE -- LogIngest is a consumer, no HTTP
```

### Key Differences

| Service | EXPOSE | HEALTHCHECK | Notes |
|---------|--------|-------------|-------|
| Api | 8080 | curl /health | Full HTTP service |
| Socket | 8080 | curl /health | SignalR over HTTP |
| Worker | none | none or custom | Background service, no HTTP |
| LogIngest | none | none | RMQ consumer only |
| MailSender | none | none | RMQ consumer only |

## Building Locally

```bash
# Build a single service
docker build -t example-app-api:local -f src/ExampleApp.Api/Dockerfile .

# Build with build arguments
docker build \
  --build-arg DOTNET_VERSION=9.0 \
  -t example-app-api:local \
  -f src/ExampleApp.Api/Dockerfile .

# Check image size
docker images example-app-api:local
```

## Troubleshooting

### "Could not find file ExampleApp.sln"

Build context must be the project root (where the .sln is), not the service directory:

```bash
# CORRECT: context is root, dockerfile is specified
docker build -f src/ExampleApp.Api/Dockerfile .

# WRONG: context is the service directory
docker build src/ExampleApp.Api/
```

### "Permission denied" at runtime

Ensure `chown -R appuser:appuser /app` runs BEFORE `USER appuser`.

### Large image size

1. Check `.dockerignore` is present and comprehensive
2. Ensure using `aspnet` base, not `sdk`
3. Run `docker history <image>` to find large layers
