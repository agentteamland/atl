---
knowledge-base-summary: "Health check configuration for every service type. PostgreSQL: pg_isready. RabbitMQ: rabbitmq-diagnostics ping. Redis: redis-cli ping. Elasticsearch: curl cluster health. .NET: /health endpoint. depends_on with condition: service_healthy."
---
# Health Checks

Every infrastructure service MUST have a health check. Without health checks, `depends_on` only waits for the container to start, not for the service inside to be ready. This leads to race conditions: API tries to connect to PostgreSQL before it accepts connections, consumers try to bind to RabbitMQ before it has booted.

## The Problem Without Health Checks

```yaml
# BAD: API starts the moment the db container process starts
depends_on:
  - db

# GOOD: API starts only after PostgreSQL accepts connections
depends_on:
  db:
    condition: service_healthy
```

With bare `depends_on`, Docker waits for the container entrypoint to start, not for the application to be ready. PostgreSQL can take 2-5 seconds to initialize, RabbitMQ can take 10-15 seconds, Elasticsearch can take 30+ seconds.

---

## Health Check Configuration Per Service

### PostgreSQL

```yaml
db:
  image: postgres:17-alpine
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres}"]
    interval: 5s
    timeout: 5s
    retries: 5
```

- `pg_isready` is a built-in utility that checks if PostgreSQL is accepting connections
- `-U` specifies the user to check against
- Fast check: returns immediately, low overhead

### RabbitMQ

```yaml
rabbitmq:
  image: rabbitmq:3-management-alpine
  healthcheck:
    test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
    interval: 10s
    timeout: 10s
    retries: 5
```

- `rabbitmq-diagnostics ping` checks if the Erlang VM and RabbitMQ application are running
- `-q` suppresses output (quiet mode)
- Longer interval (10s) because RabbitMQ startup is slower than PostgreSQL
- Longer timeout (10s) because the diagnostics command itself can be slow during startup

### Redis

```yaml
redis:
  image: redis:7-alpine
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 5s
    retries: 5
```

- `redis-cli ping` returns `PONG` if Redis is accepting connections
- Redis starts very quickly (under 1 second), but the health check ensures the TCP listener is ready

### Elasticsearch

```yaml
elasticsearch:
  image: elasticsearch:8.15.0
  healthcheck:
    test: ["CMD-SHELL", "curl -f http://localhost:9200/_cluster/health || exit 1"]
    interval: 10s
    timeout: 10s
    retries: 5
```

- `curl -f` fails silently on HTTP errors (no output, just exit code)
- `_cluster/health` returns cluster status (green, yellow, red)
- `-f` flag makes curl return exit code 22 on HTTP errors, which Docker interprets as unhealthy
- Elasticsearch is the slowest to start (15-30 seconds), so 10s interval with 5 retries gives 50 seconds total

### MinIO

```yaml
minio:
  image: minio/minio
  healthcheck:
    test: ["CMD", "mc", "ready", "local"]
    interval: 10s
    timeout: 5s
    retries: 5
```

- `mc ready local` checks if the MinIO server is ready to accept requests
- Alternative: `curl -f http://localhost:9000/minio/health/live`

### Mailpit

```yaml
mailpit:
  image: axllent/mailpit
  healthcheck:
    test: ["CMD-SHELL", "wget -q --spider http://localhost:8025 || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 3
```

- Mailpit does not include curl by default, so use `wget --spider` (header-only request)
- Checks the web UI port (8025), not the SMTP port (1025)
- Mailpit starts quickly, 3 retries is sufficient

### .NET Services (Custom /health Endpoint)

For .NET web hosts (API, Socket), implement a health endpoint and configure the health check in compose:

```yaml
api:
  healthcheck:
    test: ["CMD-SHELL", "curl -f http://localhost:3000/health || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 30s
```

The `start_period` is important for .NET services: `dotnet watch` needs time to compile on first start. During the start period, failed health checks do not count against the retry limit.

**Implementation in Program.cs:**

```csharp
// Minimal health endpoint — no dependencies, just "I'm alive"
app.MapGet("/health", () => Results.Ok(new { status = "healthy" }))
   .ExcludeFromDescription();
```

For richer health checks (database connectivity, RMQ connection), use `Microsoft.Extensions.Diagnostics.HealthChecks`:

```csharp
builder.Services.AddHealthChecks()
    .AddNpgSql(connectionString, name: "postgresql")
    .AddRabbitMQ(rabbitConnectionString, name: "rabbitmq")
    .AddRedis(redisConnectionString, name: "redis");

app.MapHealthChecks("/health");
```

---

## Health Check Parameters

| Parameter | Description | Recommended |
|-----------|-------------|-------------|
| `test` | Command to run inside the container | Service-specific (see above) |
| `interval` | Time between checks | 5s for fast services, 10s for slow |
| `timeout` | Max time for a single check to complete | 5s for fast, 10s for slow |
| `retries` | Consecutive failures before "unhealthy" | 3-5 |
| `start_period` | Grace period during startup (failures don't count) | 0s for infra, 30s for .NET |

### Timing Calculation

Total time before a service is marked unhealthy:
```
start_period + (interval * retries) + (timeout * retries)
```

For PostgreSQL: `0 + (5 * 5) + (5 * 5) = 50 seconds max`
For .NET API: `30 + (10 * 5) + (5 * 5) = 105 seconds max`

---

## depends_on with condition

The `condition` key controls when a dependent service starts:

```yaml
depends_on:
  db:
    condition: service_healthy       # Wait for health check to pass
  api:
    condition: service_started       # Wait for container to start (no health check needed)
```

| Condition | When to Use |
|-----------|-------------|
| `service_healthy` | Infrastructure services with health checks (db, rabbitmq, redis, elasticsearch) |
| `service_started` | Services without health checks, or when you just need the process running |
| `service_completed_successfully` | Init containers that run once and exit (migrations, seeders) |

### Dependency Examples

```yaml
# API depends on three infrastructure services, all must be healthy
api:
  depends_on:
    db:
      condition: service_healthy
    rabbitmq:
      condition: service_healthy
    redis:
      condition: service_healthy

# Frontend depends on API being started (not necessarily healthy)
web:
  depends_on:
    api:
      condition: service_started

# LogIngest depends on both RMQ and Elasticsearch
log-ingest:
  depends_on:
    rabbitmq:
      condition: service_healthy
    elasticsearch:
      condition: service_healthy

# Tool UIs depend on their infrastructure service
kibana:
  depends_on:
    elasticsearch:
      condition: service_healthy
```

---

## Startup Order Guaranteed by Health Checks

The final startup order, guaranteed by health check dependencies:

```
1. db, rabbitmq, redis, elasticsearch, mailpit     → Start simultaneously
   │                                                  (no dependencies)
   ▼ All report healthy
2. db-ui, kibana, redis-ui                          → Start simultaneously
   │                                                  (depend on infra healthy)
   ▼
3. api, socket, worker, log-ingest, mail-sender     → Start simultaneously
   │                                                  (depend on infra healthy)
   ▼ api reports started
4. web                                              → Starts last
                                                      (depends on api started)
```

Without health checks, steps 2-4 would race against step 1, causing connection errors during the first 10-30 seconds of startup.

---

## Debugging Health Checks

```bash
# Check the health status of all containers:
docker compose ps

# Output includes a STATUS column: "healthy", "unhealthy", "starting"
# NAME            STATUS              PORTS
# wfm-db          Up 2 min (healthy)  0.0.0.0:5432->5432/tcp
# wfm-rabbitmq    Up 2 min (healthy)  0.0.0.0:5672->5672/tcp

# Inspect a specific container's health check details:
docker inspect --format='{{json .State.Health}}' wfm-db | jq

# Run the health check command manually:
docker compose exec db pg_isready -U postgres
docker compose exec rabbitmq rabbitmq-diagnostics -q ping
docker compose exec redis redis-cli ping

# Watch health check logs:
docker compose logs -f db 2>&1 | grep -i health
```

---
