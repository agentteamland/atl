---
knowledge-base-summary: "Serilog → RabbitMQ → LogIngest → Elasticsearch → Kibana pipeline. Docker Compose configuration for the full logging stack. Index lifecycle, retention, Kibana setup. LogIngest uses console-only logging (prevents infinite loop)."
---
# Logging Infrastructure

The full logging pipeline: Serilog -> RabbitMQ -> LogIngest -> Elasticsearch -> Kibana.

## Pipeline Overview

```
.NET Service (Serilog)
    |
    v
RabbitMQ (logs.fanout exchange)
    |
    v
LogIngest (consumer) ---- CONSOLE ONLY logging (no Serilog, no RMQ sink)
    |
    v
Elasticsearch (index: {project}-logs-{yyyy.MM.dd})
    |
    v
Kibana (visualization, search, dashboards)
```

## Elasticsearch Service

```yaml
# docker-compose.yml
elasticsearch:
  image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
  container_name: ${PROJECT_PREFIX}-elasticsearch
  environment:
    - discovery.type=single-node
    - xpack.security.enabled=false
    - xpack.security.enrollment.enabled=false
    - xpack.security.http.ssl.enabled=false
    - xpack.security.transport.ssl.enabled=false
    - ES_JAVA_OPTS=-Xms512m -Xmx512m
    - cluster.name=${PROJECT_PREFIX}-cluster
    - bootstrap.memory_lock=true
  ulimits:
    memlock:
      soft: -1
      hard: -1
  volumes:
    - elasticsearch_data:/usr/share/elasticsearch/data
  ports:
    - "${ELASTICSEARCH_PORT:-9200}:9200"
  healthcheck:
    test: ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 10
    start_period: 30s
  restart: unless-stopped
```

### Key Configuration

| Setting | Value | Why |
|---------|-------|-----|
| `discovery.type` | `single-node` | Dev environment, no cluster needed |
| `xpack.security.enabled` | `false` | No auth overhead in dev |
| `ES_JAVA_OPTS` | `-Xms512m -Xmx512m` | MUST limit or ES eats all RAM |
| `bootstrap.memory_lock` | `true` | Prevents swapping, improves performance |

## Kibana Service

```yaml
kibana:
  image: docker.elastic.co/kibana/kibana:8.15.0
  container_name: ${PROJECT_PREFIX}-kibana
  environment:
    - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    - SERVER_NAME=${PROJECT_PREFIX}-kibana
    - MONITORING_ENABLED=false
  ports:
    - "${KIBANA_PORT:-5601}:5601"
  depends_on:
    elasticsearch:
      condition: service_healthy
  healthcheck:
    test: ["CMD-SHELL", "curl -sf http://localhost:5601/api/status || exit 1"]
    interval: 15s
    timeout: 10s
    retries: 10
    start_period: 60s
  restart: unless-stopped
```

## RabbitMQ Topology for Logging

RabbitMQ is configured with a fanout exchange for logs. The topology is declared by the application at startup (via RabbitMQ.Client), not manually.

### Exchange & Queue Structure

```
Exchange: logs.fanout (type: fanout, durable: true)
    |
    +---> Queue: logs.elasticsearch (durable: true, auto-ack: false)
    |         ^--- LogIngest consumer binds here
    |
    +---> (future queues can be added: logs.alerting, logs.archive, etc.)
```

### Why Fanout?

- Any new consumer can bind without changing producers
- Add a logs.alerting queue later without touching existing code
- Producers just publish to the exchange, routing is declarative

### Declaration in Code (at startup)

```csharp
// Shared topology declaration (called by both publisher and consumer)
channel.ExchangeDeclare("logs.fanout", ExchangeType.Fanout, durable: true);
channel.QueueDeclare("logs.elasticsearch", durable: true, exclusive: false, autoDelete: false);
channel.QueueBind("logs.elasticsearch", "logs.fanout", routingKey: "");
```

## LogIngest Service

**CRITICAL RULE: LogIngest logs ONLY to console. Never via Serilog RMQ sink.**

If LogIngest uses the same Serilog -> RMQ pipeline, it creates an infinite loop:
1. LogIngest writes a log
2. Serilog sends it to RMQ
3. LogIngest consumes it from RMQ
4. LogIngest writes a log about consuming
5. Serilog sends THAT to RMQ
6. Infinite loop, RMQ fills up, system crashes

### LogIngest Docker Compose

```yaml
log-ingest:
  build:
    context: .
    dockerfile: src/ExampleApp.LogIngest/Dockerfile.dev
  container_name: ${PROJECT_PREFIX}-log-ingest
  environment:
    - ASPNETCORE_ENVIRONMENT=Development
    - RabbitMQ__Host=rabbitmq
    - RabbitMQ__Port=5672
    - RabbitMQ__User=${RABBITMQ_USER}
    - RabbitMQ__Password=${RABBITMQ_PASSWORD}
    - Elasticsearch__Url=http://elasticsearch:9200
    - Elasticsearch__IndexPrefix=${PROJECT_PREFIX}-logs
    - Logging__LogLevel__Default=Information
  depends_on:
    rabbitmq:
      condition: service_healthy
    elasticsearch:
      condition: service_healthy
  restart: unless-stopped
```

### LogIngest Logging Configuration

```json
// appsettings.json for LogIngest ONLY
{
  "Logging": {
    "LogLevel": {
      "Default": "Information",
      "Microsoft": "Warning"
    },
    "Console": {
      "FormatterName": "Simple",
      "FormatterOptions": {
        "TimestampFormat": "yyyy-MM-dd HH:mm:ss ",
        "SingleLine": true
      }
    }
  }
}
```

LogIngest uses `Microsoft.Extensions.Logging` (ILogger) with console provider only. No Serilog. No RMQ sink. Period.

## Index Naming Convention

```
Pattern: {project-prefix}-logs-{yyyy.MM.dd}

Examples:
  wfm-logs-2026.04.14
  wfm-logs-2026.04.15
  pg-logs-2026.04.14
```

### Why Daily Indices?

- Easy retention: delete indices older than N days
- Search scope: query only recent days for faster results
- Size management: each index stays manageable

### Index Template (applied by LogIngest at startup)

```json
PUT _index_template/project-logs-template
{
  "index_patterns": ["*-logs-*"],
  "template": {
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 0,
      "index.lifecycle.name": "logs-retention-policy"
    },
    "mappings": {
      "properties": {
        "@timestamp": { "type": "date" },
        "level": { "type": "keyword" },
        "message": { "type": "text" },
        "exception": { "type": "text" },
        "source": { "type": "keyword" },
        "traceId": { "type": "keyword" },
        "userId": { "type": "keyword" },
        "action": { "type": "keyword" },
        "environment": { "type": "keyword" }
      }
    }
  }
}
```

## Retention: Index Lifecycle Management (ILM)

```json
PUT _ilm/policy/logs-retention-policy
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_age": "1d",
            "max_primary_shard_size": "5gb"
          }
        }
      },
      "delete": {
        "min_age": "30d",
        "actions": {
          "delete": {}
        }
      }
    }
  }
}
```

For development, 30 days is generous. In production, adjust based on storage and compliance requirements.

### Manual Cleanup (dev convenience)

```bash
# Delete indices older than 7 days (dev cleanup)
curl -X DELETE "http://localhost:9200/wfm-logs-2026.04.0*"

# Delete ALL log indices (full reset)
curl -X DELETE "http://localhost:9200/*-logs-*"
```

## Kibana Initial Setup

After first `docker compose up`, Kibana needs an index pattern to display logs.

### Step 1: Wait for Data

```bash
# Check if any log indices exist
curl -s "http://localhost:9200/_cat/indices/*-logs-*?v"
```

### Step 2: Create Data View (Kibana 8.x)

1. Open Kibana: http://localhost:5601
2. Go to Management -> Stack Management -> Data Views
3. Create data view:
   - Name: `Project Logs`
   - Index pattern: `*-logs-*`
   - Timestamp field: `@timestamp`

### Automated Setup via API

```bash
# Create data view via Kibana API (can be scripted)
curl -X POST "http://localhost:5601/api/data_views/data_view" \
  -H "kbn-xsrf: true" \
  -H "Content-Type: application/json" \
  -d '{
    "data_view": {
      "title": "*-logs-*",
      "name": "Project Logs",
      "timeFieldName": "@timestamp"
    }
  }'
```

## Full Logging Stack in docker-compose.yml

```yaml
# --- Logging Stack ---

elasticsearch:
  image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
  container_name: ${PROJECT_PREFIX}-elasticsearch
  environment:
    - discovery.type=single-node
    - xpack.security.enabled=false
    - xpack.security.enrollment.enabled=false
    - xpack.security.http.ssl.enabled=false
    - xpack.security.transport.ssl.enabled=false
    - ES_JAVA_OPTS=-Xms512m -Xmx512m
    - cluster.name=${PROJECT_PREFIX}-cluster
    - bootstrap.memory_lock=true
  ulimits:
    memlock:
      soft: -1
      hard: -1
  volumes:
    - elasticsearch_data:/usr/share/elasticsearch/data
  ports:
    - "${ELASTICSEARCH_PORT:-9200}:9200"
  healthcheck:
    test: ["CMD-SHELL", "curl -sf http://localhost:9200/_cluster/health || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 10
    start_period: 30s
  restart: unless-stopped

kibana:
  image: docker.elastic.co/kibana/kibana:8.15.0
  container_name: ${PROJECT_PREFIX}-kibana
  environment:
    - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    - SERVER_NAME=${PROJECT_PREFIX}-kibana
    - MONITORING_ENABLED=false
  ports:
    - "${KIBANA_PORT:-5601}:5601"
  depends_on:
    elasticsearch:
      condition: service_healthy
  healthcheck:
    test: ["CMD-SHELL", "curl -sf http://localhost:5601/api/status || exit 1"]
    interval: 15s
    timeout: 10s
    retries: 10
    start_period: 60s
  restart: unless-stopped

log-ingest:
  build:
    context: .
    dockerfile: src/ExampleApp.LogIngest/Dockerfile.dev
  container_name: ${PROJECT_PREFIX}-log-ingest
  environment:
    - ASPNETCORE_ENVIRONMENT=Development
    - RabbitMQ__Host=rabbitmq
    - RabbitMQ__Port=5672
    - RabbitMQ__User=${RABBITMQ_USER}
    - RabbitMQ__Password=${RABBITMQ_PASSWORD}
    - Elasticsearch__Url=http://elasticsearch:9200
    - Elasticsearch__IndexPrefix=${PROJECT_PREFIX}-logs
  depends_on:
    rabbitmq:
      condition: service_healthy
    elasticsearch:
      condition: service_healthy
  restart: unless-stopped

# --- Volumes ---
volumes:
  elasticsearch_data:
    name: ${PROJECT_PREFIX}_elasticsearch_data
```

## Troubleshooting

### Elasticsearch won't start

```bash
# Check if vm.max_map_count is sufficient (Linux/WSL)
sysctl vm.max_map_count
# If less than 262144:
sudo sysctl -w vm.max_map_count=262144

# macOS Docker Desktop: already configured, no action needed
```

### No logs appearing in Kibana

1. Check LogIngest is running: `docker compose logs log-ingest`
2. Check RabbitMQ queue has consumers: http://localhost:15672 -> Queues -> logs.elasticsearch
3. Check indices exist: `curl http://localhost:9200/_cat/indices/*-logs-*?v`
4. Check Kibana data view is configured (see Kibana Initial Setup above)

### Elasticsearch eating too much RAM

Always set `ES_JAVA_OPTS=-Xms512m -Xmx512m`. Without this, ES will claim 50% of system RAM. See `resource-limits.md` for full resource management.
