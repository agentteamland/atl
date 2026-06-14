---
knowledge-base-summary: "CPU and memory limits for containers. Elasticsearch/Kibana tuning (JVM heap size). Preventing single container from consuming all host resources. Development vs production resource profiles."
---
# Resource Limits

Container CPU and memory limits. Without limits, a single container (especially Elasticsearch) can consume all host resources and freeze the machine.

## Why Limits Matter

### The Horror Story

1. Developer starts `docker compose up`
2. Elasticsearch starts with default JVM settings
3. ES claims 50% of system RAM (8 GB on a 16 GB machine)
4. Kibana starts, claims another 1.5 GB
5. PostgreSQL, Redis, RabbitMQ, 5 .NET services each take their share
6. Total: 14+ GB claimed, machine starts swapping
7. macOS beach ball, nothing responds, force restart

### The Fix

Set explicit limits for every resource-hungry service. The development environment runs fine on 8 GB total if limits are configured.

## Docker Compose Resource Limits

### Syntax (Compose v3.x with deploy)

```yaml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 1G
        reservations:
          cpus: "0.5"
          memory: 512M
```

**Note:** `deploy.resources` works with `docker compose up` in recent versions of Docker Compose (v2.x CLI). No need for Swarm mode.

### Key Concepts

| Setting | Meaning |
|---------|---------|
| `limits.memory` | Hard cap -- container is OOM-killed if exceeded |
| `limits.cpus` | Maximum CPU cores the container can use |
| `reservations.memory` | Minimum guaranteed memory |
| `reservations.cpus` | Minimum guaranteed CPU |

## Recommended Limits Per Service

### Development Profile

For a 16 GB machine running all services simultaneously:

| Service | Memory Limit | CPU Limit | JVM/Special Config | Notes |
|---------|-------------|-----------|-------------------|-------|
| PostgreSQL | 512M | 1.0 | `shared_buffers=128MB` | Rarely needs more in dev |
| Redis | 256M | 0.5 | `maxmemory 128mb` | In-memory, cap explicitly |
| RabbitMQ | 512M | 1.0 | `vm_memory_high_watermark.relative = 0.4` | Paging threshold |
| Elasticsearch | 1G | 2.0 | `ES_JAVA_OPTS=-Xms512m -Xmx512m` | MUST set JVM heap |
| Kibana | 1G | 1.0 | - | UI is memory-hungry |
| API (.NET) | 512M | 1.0 | - | Typical .NET service |
| Socket (.NET) | 256M | 0.5 | - | Lightweight relay |
| Worker (.NET) | 256M | 0.5 | - | Background jobs |
| LogIngest (.NET) | 256M | 0.5 | - | RMQ consumer |
| MailSender (.NET) | 256M | 0.5 | - | RMQ consumer |
| Adminer | 128M | 0.25 | - | Simple PHP app |
| Redis Commander | 128M | 0.25 | - | Simple Node app |
| Mailpit | 128M | 0.25 | - | Dev SMTP |
| **Total** | **~5.5 GB** | | | Leaves room for Docker + OS |

### Production Profile

| Service | Memory Limit | CPU Limit | Notes |
|---------|-------------|-----------|-------|
| PostgreSQL | 4G | 4.0 | More for query caching |
| Redis | 1G | 1.0 | Depends on cache size |
| RabbitMQ | 1G | 2.0 | Message throughput |
| Elasticsearch | 4G | 4.0 | `ES_JAVA_OPTS=-Xms2g -Xmx2g` |
| Kibana | 2G | 2.0 | Dashboard rendering |
| API (.NET) | 1G | 2.0 | Concurrent requests |
| Socket (.NET) | 512M | 1.0 | WebSocket connections |
| Worker (.NET) | 512M | 1.0 | Job processing |

## Elasticsearch: The Most Critical Limit

Elasticsearch is the single biggest resource hog. Without JVM heap limits, it claims 50% of available RAM.

```yaml
elasticsearch:
  image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
  environment:
    - ES_JAVA_OPTS=-Xms512m -Xmx512m    # MUST SET THIS
  deploy:
    resources:
      limits:
        memory: 1G       # Container limit > JVM heap (overhead for native memory)
        cpus: "2.0"
```

### Rules for ES Memory

1. **Set `ES_JAVA_OPTS` explicitly** -- never rely on defaults
2. **Xms = Xmx** -- always set min and max heap to the same value (prevents resize pauses)
3. **Container limit > JVM heap** -- ES uses native memory beyond the JVM heap (Lucene segments, network buffers). Set container limit to roughly 1.5x-2x the JVM heap
4. **Never exceed 50% of available RAM** -- the OS needs memory for filesystem cache

### Development: 512m Heap

```
ES_JAVA_OPTS=-Xms512m -Xmx512m
Container limit: 1G
```

### Production: 2g Heap

```
ES_JAVA_OPTS=-Xms2g -Xmx2g
Container limit: 4G
```

## Redis: In-Memory Limit

Redis holds everything in memory. Without a limit, it grows until the machine runs out of RAM.

```yaml
redis:
  image: redis:7-alpine
  command: redis-server --maxmemory 128mb --maxmemory-policy allkeys-lru
  deploy:
    resources:
      limits:
        memory: 256M
```

### maxmemory-policy Options

| Policy | Behavior |
|--------|----------|
| `allkeys-lru` | Evict least recently used keys (recommended for cache) |
| `volatile-lru` | Evict LRU among keys with TTL set |
| `noeviction` | Return error when memory limit reached |

For cache use (refresh tokens, rate limits), `allkeys-lru` is appropriate. For persistent data, use `noeviction` and monitor memory usage.

## RabbitMQ: Memory Watermark

```yaml
rabbitmq:
  image: rabbitmq:3-management-alpine
  environment:
    - RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS=-rabbit vm_memory_high_watermark 0.4
  deploy:
    resources:
      limits:
        memory: 512M
```

The `vm_memory_high_watermark` (0.4 = 40% of available memory) triggers flow control -- producers are throttled when memory exceeds this threshold. This prevents RabbitMQ from consuming all memory and getting OOM-killed.

## PostgreSQL: Shared Buffers

```yaml
db:
  image: postgres:17-alpine
  command: >
    postgres
      -c shared_buffers=128MB
      -c work_mem=4MB
      -c maintenance_work_mem=64MB
      -c effective_cache_size=256MB
  deploy:
    resources:
      limits:
        memory: 512M
```

### Key PostgreSQL Memory Settings

| Setting | Dev Value | What It Does |
|---------|-----------|-------------|
| `shared_buffers` | 128MB | Data cache in shared memory (25% of available) |
| `work_mem` | 4MB | Per-operation sort/hash memory |
| `maintenance_work_mem` | 64MB | VACUUM, CREATE INDEX memory |
| `effective_cache_size` | 256MB | Planner's estimate of OS cache |

## Full Docker Compose with Limits

```yaml
services:
  db:
    image: postgres:17-alpine
    container_name: ${PROJECT_PREFIX}-db
    command: postgres -c shared_buffers=128MB -c work_mem=4MB
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M

  redis:
    image: redis:7-alpine
    container_name: ${PROJECT_PREFIX}-redis
    command: redis-server --maxmemory 128mb --maxmemory-policy allkeys-lru
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M

  rabbitmq:
    image: rabbitmq:3-management-alpine
    container_name: ${PROJECT_PREFIX}-rabbitmq
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
    container_name: ${PROJECT_PREFIX}-elasticsearch
    environment:
      - ES_JAVA_OPTS=-Xms512m -Xmx512m
    deploy:
      resources:
        limits:
          cpus: "2.0"
          memory: 1G

  kibana:
    image: docker.elastic.co/kibana/kibana:8.15.0
    container_name: ${PROJECT_PREFIX}-kibana
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 1G

  api:
    container_name: ${PROJECT_PREFIX}-api
    deploy:
      resources:
        limits:
          cpus: "1.0"
          memory: 512M

  socket:
    container_name: ${PROJECT_PREFIX}-socket
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M

  worker:
    container_name: ${PROJECT_PREFIX}-worker
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M

  log-ingest:
    container_name: ${PROJECT_PREFIX}-log-ingest
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M

  mail-sender:
    container_name: ${PROJECT_PREFIX}-mail-sender
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 256M
```

## Monitoring: docker stats

```bash
# Real-time resource usage for all containers
docker stats

# Output:
# CONTAINER     CPU %   MEM USAGE / LIMIT   MEM %   NET I/O       BLOCK I/O
# wfm-es        1.23%   487MiB / 1GiB       47.56%  1.2MB/800KB   50MB/10MB
# wfm-db        0.15%   45MiB / 512MiB      8.79%   500KB/200KB   5MB/1MB
# wfm-api       0.30%   120MiB / 512MiB     23.44%  2MB/1MB       1MB/500KB

# Filter to specific containers
docker stats wfm-elasticsearch wfm-kibana wfm-db

# One-shot (non-streaming)
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}"
```

## Symptoms of Missing Limits

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Machine freezes, beach ball | ES using all RAM | Set `ES_JAVA_OPTS` |
| Container restarts randomly | OOM kill (no limit or limit too low) | Check `docker inspect` for OOMKilled, increase limit |
| Slow IDE, slow builds | Docker using too much CPU | Set CPU limits |
| "Cannot allocate memory" errors | Total container memory > available | Reduce limits or stop unused services |
| Docker Desktop "out of resources" | Docker VM memory limit too low | Increase Docker Desktop memory allocation |

### Docker Desktop Memory Allocation

On macOS/Windows, Docker runs inside a VM. Default is often 2 GB -- far too low for a full stack.

**Recommended: 8 GB for Docker Desktop VM** (on a 16 GB machine).

Settings -> Resources -> Memory -> 8 GB

## Development vs Production Profiles

Use Compose profiles or override files:

```yaml
# docker-compose.yml (base)
services:
  elasticsearch:
    environment:
      - ES_JAVA_OPTS=${ES_JAVA_OPTS:--Xms512m -Xmx512m}
```

```bash
# .env (development)
ES_JAVA_OPTS=-Xms512m -Xmx512m

# .env.production
ES_JAVA_OPTS=-Xms2g -Xmx2g
```

Or use override files:

```yaml
# docker-compose.prod.yml
services:
  elasticsearch:
    environment:
      - ES_JAVA_OPTS=-Xms2g -Xmx2g
    deploy:
      resources:
        limits:
          memory: 4G
          cpus: "4.0"
```

```bash
# Production deploy
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```
