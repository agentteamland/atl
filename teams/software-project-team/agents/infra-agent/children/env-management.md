---
knowledge-base-summary: ".env + .env.example pattern. Variable naming convention (SCREAMING_SNAKE_CASE). Secret management (gitignore .env, commit .env.example). Environment-specific overrides. Docker Compose variable interpolation."
---
# Environment Management

All runtime configuration lives in environment variables. No hardcoded connection strings, no config files committed with real secrets. Two files govern everything: `.env` (real values, gitignored) and `.env.example` (template, committed).

## The Two-File Pattern

```
.env              → Real values. Gitignored. Every developer creates their own.
.env.example      → Template with safe defaults and explanatory comments. Committed to git.
```

### .gitignore Entry

```gitignore
# Environment
.env
.env.local
.env.*.local
# DO NOT add .env.example — it must be committed
```

### Developer Workflow

```bash
# First time setup: copy template and adjust if needed
cp .env.example .env

# Most of the time, defaults work for local development — no changes needed
docker compose up
```

---

## Variable Naming Convention

All variables use **SCREAMING_SNAKE_CASE**, grouped by the service they configure.

### Grouping Rules

1. Group variables by the service they belong to
2. Add a comment header for each group
3. Within a group, order: identifiers first, credentials second, ports last
4. Every variable that a developer might need to change gets a comment

---

## Docker Compose Interpolation

Docker Compose reads `.env` automatically (no `env_file:` directive needed). Variables are referenced with the `${VAR:-default}` syntax:

```yaml
environment:
  POSTGRES_USER: ${POSTGRES_USER:-postgres}        # Use .env value, fallback to "postgres"
  POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
ports:
  - "${POSTGRES_PORT:-5432}:5432"                  # Use .env value, fallback to 5432
```

### Interpolation Syntax

| Syntax | Behavior |
|--------|----------|
| `${VAR}` | Use .env value, empty string if missing |
| `${VAR:-default}` | Use .env value, use "default" if missing or empty |
| `${VAR-default}` | Use .env value, use "default" only if undefined (empty string = valid) |

**Always use `${VAR:-default}`** for all variables. This ensures `docker compose up` works even without a `.env` file (all defaults kick in).

### Connection String Construction

Connection strings are assembled inside docker-compose.yml using interpolation:

```yaml
environment:
  ConnectionStrings__DefaultConnection: "Host=db;Port=5432;Database=${POSTGRES_DB:-example_app};Username=${POSTGRES_USER:-postgres};Password=${POSTGRES_PASSWORD:-postgres}"
  ConnectionStrings__Redis: "redis:6379"
  ConnectionStrings__RabbitMq: "amqp://${RABBITMQ_DEFAULT_USER:-guest}:${RABBITMQ_DEFAULT_PASS:-guest}@rabbitmq:5672"
```

Note: service hostnames (`db`, `redis`, `rabbitmq`) are Docker Compose service names, not configurable — they are internal DNS names on the Compose network.

---

## Environment Variable Categories

### 1. Credentials

```bash
# PostgreSQL
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# RabbitMQ
RABBITMQ_DEFAULT_USER=guest
RABBITMQ_DEFAULT_PASS=guest

# JWT
JWT_SECRET=super-secret-key-that-should-be-at-least-32-chars-long!

# Internal Token (system-to-system auth: Worker->API, API->Socket)
INTERNAL_TOKEN=internal-secret-change-in-production
```

**Rule:** .env.example contains safe defaults (dev passwords). Production uses completely different values managed by the deployment platform (Kubernetes secrets, AWS Parameter Store, etc.).

### 2. Identifiers

```bash
# Database name
POSTGRES_DB=example_app

# JWT claims
JWT_ISSUER=ExampleApp
JWT_AUDIENCE=ExampleApp
```

### 3. Ports

```bash
# Infrastructure
POSTGRES_PORT=5432
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672
REDIS_PORT=6379
ELASTICSEARCH_PORT=9200

# .NET Services
API_PORT=3000
SOCKET_PORT=3002

# Tools
KIBANA_PORT=5601
ADMINER_PORT=8083
REDIS_COMMANDER_PORT=8082
MAILPIT_UI_PORT=8025
SMTP_PORT=1025
```

### 4. Service Configuration

```bash
# Runtime environment
ASPNETCORE_ENVIRONMENT=Development

# SMTP (Mailpit in dev, real SMTP in prod)
SMTP_HOST=mailpit
SMTP_PORT=1025
```

### 5. Feature Flags

```bash
# Optional features (add as needed)
ENABLE_SWAGGER=true
ENABLE_SEED_DATA=true
ENABLE_RATE_LIMITING=false
```

---

## Complete .env.example Template

```bash
# ================================================================
# Environment Configuration
# ================================================================
# Copy this file to .env and adjust values as needed.
# Default values work for local development — no changes required.
# ================================================================

# ── PostgreSQL ──────────────────────────────────────────────────
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=example_app
POSTGRES_PORT=5432

# ── RabbitMQ ────────────────────────────────────────────────────
RABBITMQ_DEFAULT_USER=guest
RABBITMQ_DEFAULT_PASS=guest
RABBITMQ_PORT=5672
RABBITMQ_MANAGEMENT_PORT=15672

# ── Redis ───────────────────────────────────────────────────────
REDIS_PORT=6379

# ── Elasticsearch ───────────────────────────────────────────────
ELASTICSEARCH_PORT=9200

# ── Kibana ──────────────────────────────────────────────────────
KIBANA_PORT=5601

# ── API ─────────────────────────────────────────────────────────
API_PORT=3000
ASPNETCORE_ENVIRONMENT=Development

# ── Socket (SignalR) ────────────────────────────────────────────
SOCKET_PORT=3002

# ── JWT ─────────────────────────────────────────────────────────
# MUST be at least 32 characters for HS256
JWT_SECRET=super-secret-key-that-should-be-at-least-32-chars-long!
JWT_ISSUER=ExampleApp
JWT_AUDIENCE=ExampleApp

# ── Internal Token ──────────────────────────────────────────────
# Used for system-to-system auth (Worker->API, API->Socket)
INTERNAL_TOKEN=internal-secret-change-in-production

# ── SMTP (Mailpit in dev) ──────────────────────────────────────
SMTP_HOST=mailpit
SMTP_PORT=1025
MAILPIT_UI_PORT=8025

# ── Tool UIs ────────────────────────────────────────────────────
ADMINER_PORT=8083
REDIS_COMMANDER_PORT=8082

# ── MinIO (S3-compatible storage) ──────────────────────────────
# MINIO_ROOT_USER=minioadmin
# MINIO_ROOT_PASSWORD=minioadmin
# MINIO_API_PORT=9000
# MINIO_CONSOLE_PORT=9001

# ── Frontend ────────────────────────────────────────────────────
# WEB_PORT=3001
```

---

## Secret Management Rules

### What Goes in .env

- Database passwords
- JWT secrets
- Internal tokens
- API keys for third-party services (dev keys only)
- SMTP credentials (if not Mailpit)

### What NEVER Goes in .env.example

- Real production secrets
- Personal API keys
- Private certificates or keys
- Tokens with actual access to resources

### What Goes in docker-compose.yml (Not .env)

- Service hostnames (they are Compose DNS names: `db`, `redis`, `rabbitmq`)
- Container-internal ports (right side of port mapping)
- Volume mount paths
- Connection string templates (assembled from .env vars)

---

## Adding a New Environment Variable

1. Add the variable to `.env.example` with a safe default and a comment
2. Add the variable to `.env` (your local copy) with the actual value
3. Reference it in `docker-compose.yml` using `${VAR:-default}`
4. If the variable is a port, follow the port management conventions

```bash
# Step 1: .env.example
# ── New Service ─────────────────────────────────────────────────
NEW_SERVICE_API_KEY=dev-key-not-real
NEW_SERVICE_PORT=4000
```

```yaml
# Step 2: docker-compose.yml
environment:
  NEW_SERVICE_API_KEY: ${NEW_SERVICE_API_KEY:-dev-key-not-real}
ports:
  - "${NEW_SERVICE_PORT:-4000}:4000"
```

---

## Debugging Environment Variables

```bash
# See what Docker Compose resolves for a service:
docker compose config | grep -A 20 "api:"

# See the raw environment inside a running container:
docker compose exec api env | sort

# Check if .env is being loaded:
docker compose config --format json | jq '.services.api.environment'
```

---
