---
knowledge-base-summary: "Four volume types: shadow volumes (bin/obj isolation), NuGet cache (per-service package cache), persistent data (postgres, redis, elasticsearch), source mounts (live code editing). Naming convention and cleanup."
---
# Volume Strategy

Four distinct volume types serve different purposes. Understanding when to use each prevents data loss, artifact corruption, and performance issues.

## Type 1: Shadow Volumes (bin/obj Isolation)

**Problem:** .NET compiles to `bin/` and `obj/` directories. When source code is mounted from macOS into a Linux container, the host's bin/obj (compiled for macOS) conflicts with the container's bin/obj (compiled for Linux). The result: build errors, assembly loading failures, and mysterious crashes.

**Solution:** Shadow volumes overlay the bin/obj directories inside the container. The host's bin/obj is hidden behind a Docker named volume, so each platform compiles independently.

```yaml
# docker-compose.yml — shadow volumes for the API service
volumes:
  - ./src:/src/src                              # Source code mounted from host
  - api_bin:/src/src/ExampleApp.Api/bin       # Shadows host's bin/
  - api_obj:/src/src/ExampleApp.Api/obj       # Shadows host's obj/
```

### How It Works

```
Host filesystem (macOS):
  src/ExampleApp.Api/
    ├── bin/         ← macOS build artifacts (if any)
    ├── obj/         ← macOS restore artifacts
    └── Program.cs   ← source code

Container filesystem (Linux):
  /src/src/ExampleApp.Api/
    ├── bin/         ← Docker volume (api_bin) — Linux artifacts only
    ├── obj/         ← Docker volume (api_obj) — Linux artifacts only
    └── Program.cs   ← mounted from host via ./src:/src/src
```

The source files come from the host mount. The bin/obj directories come from the Docker volume. They never mix.

### Rules for Shadow Volumes

1. **Every .NET project referenced by a service needs shadow volumes** — not just the service itself. If the API references Domain, Application, Infrastructure, and Logging, ALL of their bin/obj need shadows:

```yaml
# API service references these shared projects:
- domain_bin:/src/src/ExampleApp.Domain/bin
- domain_obj:/src/src/ExampleApp.Domain/obj
- app_bin:/src/src/ExampleApp.Application/bin
- app_obj:/src/src/ExampleApp.Application/obj
- infra_bin:/src/src/ExampleApp.Infrastructure/bin
- infra_obj:/src/src/ExampleApp.Infrastructure/obj
- logging_bin:/src/src/ExampleApp.Logging/bin
- logging_obj:/src/src/ExampleApp.Logging/obj
```

2. **Shared project volumes are reused across services.** Multiple services can mount `domain_bin:` — Docker named volumes support concurrent access. This means Domain is compiled once, and all services share the compiled output.

3. **Service-specific volumes are unique.** `api_bin:` is only used by the API. `worker_bin:` is only used by the Worker.

### Naming Convention

```
{short_name}_bin    → e.g., api_bin, socket_bin, worker_bin, logingest_bin
{short_name}_obj    → e.g., api_obj, socket_obj, worker_obj, logingest_obj
```

For shared projects:
```
domain_bin, domain_obj
app_bin, app_obj
infra_bin, infra_obj
logging_bin, logging_obj
```

---

## Type 2: NuGet Cache Volumes

**Problem:** Every time a container is recreated (after `docker compose down` or image rebuild), NuGet packages are re-downloaded from the internet. On a slow connection, this adds minutes to startup.

**Solution:** A named volume persists the NuGet package cache across container recreations.

```yaml
volumes:
  - nuget_api:/root/.nuget/packages
```

### Why Per-Service?

NuGet cache volumes are per service, not shared:

```yaml
nuget_api:
nuget_socket:
nuget_worker:
nuget_logingest:
nuget_mailsender:
```

The reason: different services may restore simultaneously. Sharing a single NuGet cache volume can cause lock contention during parallel `dotnet restore` operations. Per-service volumes eliminate this risk.

### When to Clear

If NuGet resolution fails or packages seem corrupted:

```bash
# Remove all NuGet cache volumes
docker volume rm $(docker volume ls -q | grep nuget_)

# Or remove everything and start fresh
docker compose down -v
```

---

## Type 3: Persistent Data Volumes

**Problem:** Infrastructure services store data (database rows, search indices, queue messages). Without persistent volumes, all data is lost on `docker compose down`.

**Solution:** Named volumes for data directories survive container removal. Data persists across `docker compose down` and `docker compose up`.

```yaml
# Persistent data volumes
postgres_data:/var/lib/postgresql/data
rabbitmq_data:/var/lib/rabbitmq
redis_data:/data
elasticsearch_data:/usr/share/elasticsearch/data
minio_data:/data
```

### Volume Lifecycle

```bash
# Stop containers but KEEP data volumes:
docker compose down              # Volumes survive

# Stop containers AND DELETE all data volumes:
docker compose down -v           # Volumes destroyed — full reset

# Remove a specific volume:
docker volume rm example_app_postgres_data
```

### Naming Convention

```
{service}_data    → e.g., postgres_data, redis_data, elasticsearch_data
```

### Data Survival Matrix

| Command | Containers | Volumes | Networks |
|---------|-----------|---------|----------|
| `docker compose stop` | Stopped | Kept | Kept |
| `docker compose down` | Removed | Kept | Removed |
| `docker compose down -v` | Removed | Removed | Removed |
| `docker compose restart` | Restarted | Kept | Kept |

---

## Type 4: Source Mounts (Bind Mounts)

**Problem:** Developers edit code on their host machine (with IDE, editor). The container needs to see those changes instantly for hot reload.

**Solution:** Bind mounts map a host directory directly into the container. Changes on either side are reflected immediately.

```yaml
volumes:
  - ./src:/src/src                                  # All source projects
  - ./ExampleApp.sln:/src/ExampleApp.sln        # Solution file
```

### Bind Mounts vs Named Volumes

| Property | Bind Mount | Named Volume |
|----------|-----------|--------------|
| Syntax | `./host/path:/container/path` | `volume_name:/container/path` |
| Data source | Host filesystem | Docker-managed |
| Survives `down -v` | Yes (it is on host) | No |
| Performance on macOS | Slower (filesystem translation) | Fast (native) |
| Use case | Live code editing | Cached data, build artifacts |

### Bind Mount Performance on macOS

Docker Desktop for macOS uses a virtualized Linux kernel. File system operations between macOS and the Linux VM have overhead. For source mounts this is acceptable (file watcher triggers on save, not continuous I/O). For bin/obj directories with thousands of tiny files, it is unacceptably slow — which is another reason shadow volumes exist.

---

## Volume Declaration

Every named volume used in a service definition MUST be declared in the top-level `volumes:` section. Organize by purpose:

```yaml
volumes:
  # ── Persistent data ──────────────────────────────
  postgres_data:
  rabbitmq_data:
  redis_data:
  elasticsearch_data:

  # ── Shadow volumes (bin/obj) ─────────────────────
  api_bin:
  api_obj:
  domain_bin:
  domain_obj:
  app_bin:
  app_obj:
  infra_bin:
  infra_obj:
  logging_bin:
  logging_obj:
  socket_bin:
  socket_obj:
  worker_bin:
  worker_obj:
  logingest_bin:
  logingest_obj:
  mailsender_bin:
  mailsender_obj:

  # ── NuGet cache ──────────────────────────────────
  nuget_api:
  nuget_socket:
  nuget_worker:
  nuget_logingest:
  nuget_mailsender:

  # ── Frontend ─────────────────────────────────────
  web_node_modules:
```

---

## Volume Cleanup Commands

```bash
# See all volumes for this project
docker volume ls | grep example_app

# Nuclear reset: remove everything (data, caches, build artifacts)
docker compose down -v

# Remove only build-related volumes (keep data)
docker volume rm $(docker volume ls -q | grep -E '_(bin|obj|nuget_)')

# Remove only data volumes (keep build cache)
docker volume rm $(docker volume ls -q | grep '_data')

# Prune all unused volumes globally (DANGER: affects all projects)
docker volume prune
```

---

## Checklist for New Service Volumes

- [ ] Shadow volumes for the service's own bin/obj?
- [ ] Shadow volumes for all shared projects it references?
- [ ] NuGet cache volume (per service)?
- [ ] Source mount (`./src:/src/src`)?
- [ ] Solution file mount (`./{SolutionName}.sln:/src/{SolutionName}.sln`)?
- [ ] All volumes declared in the top-level `volumes:` section?
- [ ] Persistent data volume (only for infrastructure services)?

---
