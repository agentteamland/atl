---
knowledge-base-summary: "Port convention to avoid conflicts between projects. Offset strategy: project A uses default ports, project B adds 10000. Port mapping in .env for easy override. Common port assignments table."
---
# Port Management

Every service exposes ports through `${VAR:-default}` in docker-compose.yml. This makes ports overridable per developer (via `.env`) and per project (via offset strategy). No port is hardcoded.

## Default Port Table

| Service | Host Port | Container Port | .env Variable | Notes |
|---------|-----------|---------------|---------------|-------|
| **Infrastructure** | | | | |
| PostgreSQL | 5432 | 5432 | `POSTGRES_PORT` | |
| RabbitMQ (AMQP) | 5672 | 5672 | `RABBITMQ_PORT` | Protocol port |
| RabbitMQ (Management) | 15672 | 15672 | `RABBITMQ_MANAGEMENT_PORT` | Web UI |
| Redis | 6379 | 6379 | `REDIS_PORT` | |
| Elasticsearch | 9200 | 9200 | `ELASTICSEARCH_PORT` | |
| Mailpit (SMTP) | 1025 | 1025 | `SMTP_PORT` | Dev SMTP |
| Mailpit (UI) | 8025 | 8025 | `MAILPIT_UI_PORT` | Web UI |
| MinIO (API) | 9000 | 9000 | `MINIO_API_PORT` | S3-compatible |
| MinIO (Console) | 9001 | 9001 | `MINIO_CONSOLE_PORT` | Web UI |
| **.NET Services** | | | | |
| API | 3000 | 3000 | `API_PORT` | REST API |
| Socket | 3002 | 3002 | `SOCKET_PORT` | SignalR |
| **Tools** | | | | |
| Kibana | 5601 | 5601 | `KIBANA_PORT` | |
| Adminer | 8083 | 8080 | `ADMINER_PORT` | Note: different container port |
| Redis Commander | 8082 | 8081 | `REDIS_COMMANDER_PORT` | Note: different container port |
| **Frontend** | | | | |
| Web (Vite) | 3001 | 3001 | `WEB_PORT` | React dev server |

### Port Mapping in Compose

```yaml
ports:
  - "${API_PORT:-3000}:3000"
  #   ^^^^^^^^^^^^^^^^  ^^^^
  #   host port          container port
  #   (configurable)     (fixed, app listens on this)
```

The **host port** (left side) is configurable via .env. The **container port** (right side) is fixed — it is what the application inside the container listens on and should never change.

---

## Offset Strategy for Multi-Project Coexistence

When multiple Docker Compose projects run on the same machine, port conflicts are inevitable. The offset strategy prevents this.

### How It Works

Pick a base offset for each project. All ports in that project are shifted by the offset:

| Project | Offset | PostgreSQL | API | Socket | RabbitMQ | Redis | Elasticsearch |
|---------|--------|-----------|-----|--------|----------|-------|--------------|
| ExampleApp | 0 | 5432 | 3000 | 3002 | 5672 | 6379 | 9200 |
| Project B | +10000 | 15432 | 13000 | 13002 | 15672 | 16379 | 19200 |
| Project C | +20000 | 25432 | 23000 | 23002 | 25672 | 26379 | 29200 |

### Implementation

Each project sets its ports in `.env`:

```bash
# Project B: .env
POSTGRES_PORT=15432
RABBITMQ_PORT=15672
RABBITMQ_MANAGEMENT_PORT=25672
REDIS_PORT=16379
ELASTICSEARCH_PORT=19200
KIBANA_PORT=15601
API_PORT=13000
SOCKET_PORT=13002
ADMINER_PORT=18083
REDIS_COMMANDER_PORT=18082
MAILPIT_UI_PORT=18025
SMTP_PORT=11025
```

Since docker-compose.yml uses `${VAR:-default}`, changing the .env file is all that is needed. No compose file changes.

### Container Name Isolation

Ports alone are not enough. Container names must also be unique across projects. Each project uses a prefix:

```yaml
# ExampleApp
container_name: wfm-api
container_name: wfm-db

# Project B
container_name: prjb-api
container_name: prjb-db
```

---

## macOS Port Conflicts

### AirPlay Receiver (Port 5000)

macOS Monterey and later use port 5000 for AirPlay Receiver. If your service needs port 5000, either:

1. **Avoid port 5000 entirely** (recommended) — use a different default
2. Disable AirPlay Receiver: System Settings > General > AirDrop & Handoff > AirPlay Receiver > Off

We never use port 5000 for any service.

### AirPlay (Port 7000)

macOS also uses port 7000 for AirPlay. Avoid this port as well.

### Control Center (Port 5000, 7000)

Both ports are used by `ControlCenter` process. Check with:

```bash
# See what is using a port on macOS:
lsof -i :5000
lsof -i :7000
```

---

## Common Port Conflicts to Watch For

| Port | Commonly Used By | Our Service |
|------|-----------------|-------------|
| 3000 | Node.js (Express, Next.js) | API |
| 3001 | Next.js dev, Create React App | Web (Vite) |
| 5000 | macOS AirPlay, Python Flask | AVOID |
| 5432 | Local PostgreSQL installation | db |
| 5672 | Local RabbitMQ installation | rabbitmq |
| 6379 | Local Redis installation | redis |
| 8080 | Many web servers, Jenkins, Spring | AVOID as default |
| 8082 | Varies | Redis Commander |
| 8083 | Varies | Adminer |
| 9200 | Local Elasticsearch | elasticsearch |

### Detecting Conflicts

```bash
# Check if a port is in use before starting:
lsof -i :3000

# Check all ports used by Docker:
docker ps --format "table {{.Names}}\t{{.Ports}}"

# Check all listening ports on the system:
lsof -nP -iTCP -sTCP:LISTEN | grep LISTEN
```

---

## Port Convention Rules

1. **Every host port uses ${VAR:-default}** — never hardcode
2. **Container ports are fixed** — the application listens on a known port internally
3. **Tool UIs use 8000+ range** — Adminer (8083), Redis Commander (8082), Mailpit (8025)
4. **.NET services use 3000+ range** — API (3000), Web (3001), Socket (3002)
5. **Infrastructure uses standard ports** — PostgreSQL (5432), Redis (6379), etc.
6. **Avoid ports 5000 and 7000** — macOS system conflicts
7. **Avoid port 8080** — too commonly used by other services
8. **All ports documented in .env.example** with comments
9. **Offset strategy** for multi-project setups (+10000 per project)
10. **Port changes only in .env** — never edit compose file for port overrides

---

## Quick Reference: Start Fresh After Port Conflict

```bash
# 1. Stop everything
docker compose down

# 2. Find what's using the conflicting port
lsof -i :3000

# 3. Either kill the process or change the port in .env
# Option A: Kill the process
kill -9 <PID>

# Option B: Change the port in .env
echo "API_PORT=3100" >> .env

# 4. Start again
docker compose up
```

---
