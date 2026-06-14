---
knowledge-base-summary: "Multiple projects on the same machine (ExampleApp + AdminApp). Port offset strategy (project B = default + 10000). Container naming convention prevents conflicts. Network isolation. Shared vs isolated infrastructure services."
---
# Multi-Project Coexistence

Running multiple Docker Compose projects on the same development machine without port conflicts, naming collisions, or resource fights.

## The Problem

Developer works on two projects:
- **ExampleApp** (wfm): PostgreSQL, Redis, RabbitMQ, Elasticsearch
- **PG Project** (pg): PostgreSQL, Redis, RabbitMQ, Elasticsearch

Both want port 5432 (PostgreSQL), 6379 (Redis), 5672 (RabbitMQ), 9200 (Elasticsearch). Without a strategy, only one project can run at a time.

## The Solution: Port Offset + Naming Prefix

Each project gets:
1. A **prefix** for container names (prevents Docker name collisions)
2. A **port offset** for host ports (prevents port binding collisions)
3. Its own **Docker network** (automatic with Compose, provides service isolation)

### Strategy

| Project | Prefix | Offset | PostgreSQL | Redis | RabbitMQ | ES |
|---------|--------|--------|-----------|-------|----------|-----|
| ExampleApp | wfm | 0 | 5432 | 6379 | 5672 | 9200 |
| PG Project | pg | +10000 | 15432 | 16379 | 15672 | 19200 |
| Project C | c | +20000 | 25432 | 26379 | 25672 | 29200 |

## Implementation

### .env for ExampleApp (offset 0 -- defaults)

```bash
# .env
PROJECT_PREFIX=wfm

# Ports (default, no offset)
API_PORT=3000
SOCKET_PORT=3002
POSTGRES_PORT=5432
REDIS_PORT=6379
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
ELASTICSEARCH_PORT=9200
KIBANA_PORT=5601
ADMINER_PORT=8083
REDIS_COMMANDER_PORT=8082
MAILPIT_SMTP_PORT=1025
MAILPIT_UI_PORT=8025
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
```

### .env for PG Project (offset +10000)

```bash
# .env
PROJECT_PREFIX=pg

# Ports (offset +10000)
API_PORT=13000
SOCKET_PORT=13002
POSTGRES_PORT=15432
REDIS_PORT=16379
RABBITMQ_PORT=15672
RABBITMQ_MANAGEMENT_PORT=25672
ELASTICSEARCH_PORT=19200
KIBANA_PORT=15601
ADMINER_PORT=18083
REDIS_COMMANDER_PORT=18082
MAILPIT_SMTP_PORT=11025
MAILPIT_UI_PORT=18025
MINIO_API_PORT=19000
MINIO_CONSOLE_PORT=19001
```

### docker-compose.yml Using Variables

```yaml
services:
  db:
    image: postgres:17-alpine
    container_name: ${PROJECT_PREFIX}-db
    ports:
      - "${POSTGRES_PORT:-5432}:5432"      # Host port from .env, container always 5432
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    container_name: ${PROJECT_PREFIX}-redis
    ports:
      - "${REDIS_PORT:-6379}:6379"
    volumes:
      - redis_data:/data

  rabbitmq:
    image: rabbitmq:3-management-alpine
    container_name: ${PROJECT_PREFIX}-rabbitmq
    ports:
      - "${RABBITMQ_PORT:-5672}:5672"
      - "${RABBITMQ_MANAGEMENT_PORT:-15672}:15672"
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
    container_name: ${PROJECT_PREFIX}-elasticsearch
    ports:
      - "${ELASTICSEARCH_PORT:-9200}:9200"
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data

  api:
    container_name: ${PROJECT_PREFIX}-api
    ports:
      - "${API_PORT:-3000}:3000"

volumes:
  postgres_data:
    name: ${PROJECT_PREFIX}_postgres_data
  redis_data:
    name: ${PROJECT_PREFIX}_redis_data
  rabbitmq_data:
    name: ${PROJECT_PREFIX}_rabbitmq_data
  elasticsearch_data:
    name: ${PROJECT_PREFIX}_elasticsearch_data
```

### Key Pattern

```yaml
ports:
  - "${POSTGRES_PORT:-5432}:5432"
  #   ^^^^^^^^^^^^^^^^^^^^^^^^ Host port (varies per project)
  #                         ^^^^ Container port (always default)
```

The container internally always uses the default port. Only the host-side mapping changes. This means application code always connects to the default port (via Docker network), not the offset port.

## Container Naming Convention

Every container name is prefixed with `PROJECT_PREFIX`:

```
wfm-db              pg-db
wfm-redis           pg-redis
wfm-rabbitmq        pg-rabbitmq
wfm-api             pg-api
wfm-elasticsearch   pg-elasticsearch
```

### Why Prefix Matters

Without prefix, both projects try to create a container named `db`:

```
Error: Conflict. The container name "/db" is already in use.
```

With prefix, they coexist:

```bash
docker ps --format "table {{.Names}}\t{{.Ports}}"
# NAMES               PORTS
# wfm-db              0.0.0.0:5432->5432/tcp
# wfm-redis           0.0.0.0:6379->6379/tcp
# pg-db               0.0.0.0:15432->5432/tcp
# pg-redis             0.0.0.0:16379->6379/tcp
```

## Volume Naming Convention

```yaml
volumes:
  postgres_data:
    name: ${PROJECT_PREFIX}_postgres_data
```

This produces:
- `wfm_postgres_data` for ExampleApp
- `pg_postgres_data` for PG Project

Without explicit names, Docker Compose uses the project directory name as prefix, which can be unpredictable.

## Network Isolation

Docker Compose automatically creates a network per project:

```
example-app_default    (wfm services talk to each other here)
pg-project_default      (pg services talk to each other here)
```

Services from different projects cannot reach each other by default. This is the desired behavior -- each project is fully isolated.

### Internal Connectivity

Within the same Compose project, services connect by service name:

```
# ExampleApp API connects to its own database
Host=db      (resolves to wfm-db within example-app_default network)
Port=5432    (default port, not the offset)

# PG API connects to its own database
Host=db      (resolves to pg-db within pg-project_default network)
Port=5432    (default port, not the offset)
```

Application configuration does NOT change between projects. The offset only affects the host machine.

## Shared vs Isolated Services

### Isolated (Recommended)

Each project runs its own instances of all services. Pros:
- Complete independence
- No version conflicts
- Easy to `docker compose down -v` without affecting other projects
- Different PostgreSQL versions per project

### Shared (Advanced)

Multiple projects share one PostgreSQL instance with different databases. Only do this if machine resources are extremely tight.

```yaml
# Shared postgres (runs independently)
# docker-compose.shared.yml
services:
  shared-db:
    image: postgres:17-alpine
    container_name: shared-db
    ports:
      - "5432:5432"
    volumes:
      - shared_postgres_data:/var/lib/postgresql/data
    environment:
      POSTGRES_USER: admin
      POSTGRES_PASSWORD: admin
```

```bash
# Create databases for each project
docker exec shared-db createdb -U admin example-app
docker exec shared-db createdb -U admin pg_project
```

**Recommendation: Always use isolated.** The resource cost of running separate PostgreSQL instances is minimal, and the operational simplicity is worth it.

## Quick Reference: "Which Port Is What?"

Keep a reference card in each project's `.env.example`:

```bash
# ==========================================
# PORT REFERENCE (ExampleApp, offset: 0)
# ==========================================
# API:                 http://localhost:3000
# API Swagger:         http://localhost:3000/swagger
# Socket:              http://localhost:3002
# PostgreSQL:          localhost:5432
# Redis:               localhost:6379
# Redis Commander:     http://localhost:8082
# RabbitMQ:            localhost:5672
# RabbitMQ Management: http://localhost:15672
# Elasticsearch:       http://localhost:9200
# Kibana:              http://localhost:5601
# Adminer:             http://localhost:8083
# Mailpit SMTP:        localhost:1025
# Mailpit UI:          http://localhost:8025
# MinIO API:           http://localhost:9000
# MinIO Console:       http://localhost:9001
# ==========================================
```

## Complete Port Offset Table

| Service | Default | +10000 | +20000 |
|---------|---------|--------|--------|
| API | 3000 | 13000 | 23000 |
| Socket | 3002 | 13002 | 23002 |
| PostgreSQL | 5432 | 15432 | 25432 |
| Redis | 6379 | 16379 | 26379 |
| RabbitMQ AMQP | 5672 | 15672 | 25672 |
| RabbitMQ Mgmt | 15672 | 25672 | 35672 |
| Elasticsearch | 9200 | 19200 | 29200 |
| Kibana | 5601 | 15601 | 25601 |
| Adminer | 8083 | 18083 | 28083 |
| Redis Commander | 8082 | 18082 | 28082 |
| Mailpit SMTP | 1025 | 11025 | 21025 |
| Mailpit UI | 8025 | 18025 | 28025 |
| MinIO API | 9000 | 19000 | 29000 |
| MinIO Console | 9001 | 19001 | 29001 |

## Switching Between Projects

```bash
# Terminal 1: ExampleApp
cd ~/projects/example-app && docker compose up

# Terminal 2: PG Project
cd ~/projects/pg-project && docker compose up

# Both run simultaneously, no conflicts
```

### Stopping Everything

```bash
# Stop one project
cd ~/projects/example-app && docker compose down

# Stop all Docker containers (nuclear option)
docker stop $(docker ps -q)
```

## Troubleshooting

### "Port already in use"

```bash
# Find what's using a port
lsof -i :5432
# or
docker ps --format "{{.Names}} {{.Ports}}" | grep 5432
```

Fix: check both projects' `.env` files for port conflicts.

### "Container name already in use"

```bash
# Remove the orphaned container
docker rm -f wfm-db

# Or prune all stopped containers
docker container prune
```

Fix: ensure `PROJECT_PREFIX` is different in each project's `.env`.
