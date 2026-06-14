---
knowledge-base-summary: "Service categories: infrastructure (db, rabbitmq, redis, elasticsearch), .NET hosts (api, socket, worker, log-ingest, mail-sender), frontend (react app), tools (adminer, kibana, mailpit, redis-commander). Dependency chain and startup order."
---
# Compose Architecture

## Service Tiers

Docker Compose services are organized into four tiers. Each tier has a distinct role and startup order.

### Tier 1: Infrastructure Services

Stateful, long-running services that store data or route messages. These start first and must be healthy before anything else runs.

| Service | Image | Purpose | Port | Persistent Volume |
|---------|-------|---------|------|-------------------|
| `db` | `postgres:17-alpine` | Primary database | 5432 | `postgres_data` |
| `rabbitmq` | `rabbitmq:3-management-alpine` | Message broker | 5672, 15672 | `rabbitmq_data` |
| `redis` | `redis:7-alpine` | Cache, token store, rate limiter | 6379 | `redis_data` |
| `elasticsearch` | `elasticsearch:8.15.0` | Log storage, full-text search | 9200 | `elasticsearch_data` |
| `mailpit` | `axllent/mailpit` | Dev SMTP server (catches all email) | 1025, 8025 | none |
| `minio` | `minio/minio` | S3-compatible object storage | 9000, 9001 | `minio_data` |

### Tier 2: Tool Services (Optional UI)

Web UIs for infrastructure services. Not required for the system to function. Can be commented out to save resources.

| Service | Image | Purpose | Port | Depends On |
|---------|-------|---------|------|------------|
| `db-ui` | `adminer` | Database web UI | 8083 | db (healthy) |
| `kibana` | `kibana:8.15.0` | Elasticsearch dashboard | 5601 | elasticsearch (healthy) |
| `redis-ui` | `rediscommander/redis-commander` | Redis web UI | 8082 | redis (healthy) |

### Tier 3: .NET Host Services

Application services built with .NET 9 SDK. All use `dotnet watch run` for hot reload in development. Source code is mounted via volume, not copied into the image.

| Service | Dockerfile | Purpose | Port | Depends On |
|---------|-----------|---------|------|------------|
| `api` | `src/{Project}.Api/Dockerfile.dev` | REST API (composition root) | 3000 | db, rabbitmq, redis |
| `socket` | `src/{Project}.Socket/Dockerfile.dev` | SignalR WebSocket hub | 3002 | rabbitmq |
| `worker` | `src/{Project}.Worker/Dockerfile.dev` | Background jobs (Cronos) | none | rabbitmq |
| `log-ingest` | `src/{Project}.LogIngest/Dockerfile.dev` | RMQ -> Elasticsearch log pipeline | none | rabbitmq, elasticsearch |
| `mail-sender` | `src/{Project}.MailSender/Dockerfile.dev` | RMQ -> SMTP email sender | none | rabbitmq, redis |

### Tier 4: Frontend Services

Client-side dev servers. Start last since they depend on API availability.

| Service | Dockerfile | Purpose | Port | Depends On |
|---------|-----------|---------|------|------------|
| `web` | `web/Dockerfile.dev` | React/Vite dev server | 3001 | api (started) |

**Note:** Flutter does NOT run in Docker. It runs natively on the host machine with `flutter run`. This is the only exception to the "everything in Docker" rule.

---

## Dependency Chain

```
                    ┌─────────────┐
                    │  PostgreSQL  │
                    │     (db)     │
                    └──────┬──────┘
                           │ healthy
              ┌────────────┼────────────────┐
              │            │                │
         ┌────▼────┐  ┌───▼───┐       ┌────▼────┐
         │   API   │  │ db-ui │       │         │
         └────┬────┘  └───────┘       │         │
              │ started               │         │
         ┌────▼────┐                  │         │
         │   web   │                  │         │
         └─────────┘                  │         │
                                      │         │
                    ┌─────────────┐   │         │
                    │  RabbitMQ   │◄──┘         │
                    └──────┬──────┘             │
                           │ healthy            │
         ┌─────────┬───────┼───────┬────────┐   │
         │         │       │       │        │   │
    ┌────▼────┐┌───▼──┐┌───▼───┐┌──▼──┐┌────▼───▼────┐
    │   API   ││socket││worker ││ log ││ mail-sender │
    └─────────┘└──────┘└───────┘│ingest│└─────────────┘
                                └──┬──┘
                                   │ healthy
                    ┌──────────────┐│
                    │Elasticsearch │◄┘
                    └──────┬───────┘
                           │ healthy
                      ┌────▼────┐
                      │ kibana  │
                      └─────────┘

                    ┌─────────────┐
                    │    Redis    │
                    └──────┬──────┘
                           │ healthy
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼────┐  ┌───▼──────┐ ┌───▼──────┐
         │   API   │  │mail-sender│ │ redis-ui │
         └─────────┘  └──────────┘ └──────────┘
```

### Reading the Diagram

- API depends on PostgreSQL, RabbitMQ, and Redis (all must be healthy)
- Socket depends on RabbitMQ only
- Worker depends on RabbitMQ only
- LogIngest depends on RabbitMQ and Elasticsearch
- MailSender depends on RabbitMQ and Redis
- Web (frontend) depends on API being started (not necessarily healthy)
- Tool UIs depend on their respective infrastructure services

---

## Startup Order

Docker Compose resolves `depends_on` into a topological order. With `condition: service_healthy`, the actual startup sequence is:

```
Phase 1 (parallel):  db, rabbitmq, redis, elasticsearch, mailpit, minio
Phase 2 (parallel):  db-ui, kibana, redis-ui          (wait for Phase 1 healthy)
Phase 3 (parallel):  api, socket, worker, log-ingest, mail-sender  (wait for Phase 1 healthy)
Phase 4 (parallel):  web                              (wait for api started)
```

Services within the same phase start simultaneously. A service only waits for its specific dependencies, not for the entire previous phase.

---

## Network

All services share the default Compose network (`{project}_default`). Services reference each other by **service name** (not container name):

```
# From API connecting to PostgreSQL:
Host=db          # NOT wfm-db, NOT localhost

# From LogIngest connecting to Elasticsearch:
http://elasticsearch:9200    # NOT wfm-elasticsearch

# From MailSender connecting to RabbitMQ:
amqp://guest:guest@rabbitmq:5672    # NOT wfm-rabbitmq
```

**Container names** (e.g., `wfm-api`) are for human identification in `docker ps` output. **Service names** (e.g., `api`) are for inter-container DNS resolution.

### External Access (from host machine or browser)

```
# From host / browser:
http://localhost:3000      # API
http://localhost:3002      # Socket
http://localhost:5601      # Kibana
http://localhost:8025      # Mailpit
ws://localhost:3002/hubs   # SignalR
```

The port mapping `"${API_PORT:-3000}:3000"` maps host port (left) to container port (right). When accessing from the host, use `localhost:{host_port}`.

---

## docker-compose.yml Section Order Convention

Maintain this ordering in the compose file for readability:

```yaml
services:
  # --- Infrastructure -----------------------------------------------
  db:
  rabbitmq:
  redis:
  elasticsearch:
  mailpit:
  minio:

  # --- Tools (optional) ---------------------------------------------
  db-ui:
  kibana:
  redis-ui:

  # --- .NET Services -------------------------------------------------
  api:
  socket:
  worker:
  log-ingest:
  mail-sender:

  # --- Frontend ------------------------------------------------------
  web:

volumes:
  # Persistent data
  # Shadow volumes (bin/obj)
  # NuGet cache volumes
  # Frontend volumes
```

Each section is separated by a comment header. Infrastructure first, tools second, .NET services third, frontend last. Volumes are grouped by purpose.

---

## Adding a New Tier

If a new service type emerges (e.g., a Go microservice, a Python ML service), create a new section header and place it between the .NET services and frontend sections. Apply the same patterns: healthcheck, depends_on with condition, parameterized ports, named volumes.

---
