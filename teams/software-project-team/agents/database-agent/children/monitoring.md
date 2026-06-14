---
knowledge-base-summary: "pg_stat_statements for slow query detection. Index usage statistics (pg_stat_user_indexes). Table bloat detection. Connection pool monitoring. VACUUM and auto-maintenance. Kibana dashboard integration for database-related logs."
---
# Monitoring

## pg_stat_statements — Slow Query Detection

The most important extension for query performance monitoring. Tracks execution statistics for all SQL statements.

### Enable the Extension

```sql
-- Run once (in migration or manually)
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
```

In `postgresql.conf` (or Docker environment):
```
shared_preload_libraries = 'pg_stat_statements'
pg_stat_statements.track = all
pg_stat_statements.max = 10000
```

### Find Slow Queries

```sql
-- Top 10 slowest queries by total time
SELECT 
    calls,
    round(total_exec_time::numeric, 2) AS total_time_ms,
    round(mean_exec_time::numeric, 2) AS avg_time_ms,
    round(max_exec_time::numeric, 2) AS max_time_ms,
    rows,
    query
FROM pg_stat_statements
ORDER BY total_exec_time DESC
LIMIT 10;

-- Top 10 most frequently called queries
SELECT 
    calls,
    round(mean_exec_time::numeric, 2) AS avg_time_ms,
    round(total_exec_time::numeric, 2) AS total_time_ms,
    query
FROM pg_stat_statements
ORDER BY calls DESC
LIMIT 10;

-- Queries with high average execution time (potential optimization targets)
SELECT 
    calls,
    round(mean_exec_time::numeric, 2) AS avg_time_ms,
    round(stddev_exec_time::numeric, 2) AS stddev_ms,
    query
FROM pg_stat_statements
WHERE calls > 10
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### Reset Statistics

```sql
-- Reset after making optimizations to get fresh data
SELECT pg_stat_statements_reset();
```

## Index Usage Statistics

### Are Indexes Being Used?

```sql
-- Index usage statistics
SELECT 
    schemaname,
    tablename,
    indexrelname AS index_name,
    idx_scan AS times_used,
    idx_tup_read AS rows_read,
    idx_tup_fetch AS rows_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;
```

### Find Unused Indexes

```sql
-- Indexes that have never been scanned (candidates for removal)
SELECT 
    schemaname,
    tablename,
    indexrelname AS index_name,
    pg_size_pretty(pg_relation_size(indexrelid)) AS index_size,
    idx_scan AS scans
FROM pg_stat_user_indexes
WHERE idx_scan = 0
  AND schemaname = 'public'
  AND indexrelname NOT LIKE 'pk_%'  -- don't flag primary keys
ORDER BY pg_relation_size(indexrelid) DESC;
```

### Find Missing Indexes

```sql
-- Tables with high sequential scans (may need indexes)
SELECT 
    schemaname,
    relname AS table_name,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    CASE WHEN (seq_scan + idx_scan) > 0 
        THEN round(100.0 * idx_scan / (seq_scan + idx_scan), 1) 
        ELSE 0 
    END AS index_usage_pct,
    n_live_tup AS estimated_rows
FROM pg_stat_user_tables
WHERE schemaname = 'public'
  AND n_live_tup > 1000  -- only tables with significant data
ORDER BY seq_scan DESC;
```

## Table Bloat

Dead tuples accumulate after UPDATE and DELETE operations. VACUUM reclaims space, but if it falls behind, tables bloat.

### Check Table Bloat

```sql
-- Dead tuples per table
SELECT 
    schemaname,
    relname AS table_name,
    n_live_tup AS live_rows,
    n_dead_tup AS dead_rows,
    CASE WHEN n_live_tup > 0 
        THEN round(100.0 * n_dead_tup / n_live_tup, 1) 
        ELSE 0 
    END AS dead_pct,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
WHERE schemaname = 'public'
ORDER BY n_dead_tup DESC;
```

### Check Table Sizes

```sql
-- Table sizes including indexes and TOAST
SELECT 
    tablename,
    pg_size_pretty(pg_total_relation_size('public.' || tablename)) AS total_size,
    pg_size_pretty(pg_relation_size('public.' || tablename)) AS table_size,
    pg_size_pretty(pg_total_relation_size('public.' || tablename) - pg_relation_size('public.' || tablename)) AS index_size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size('public.' || tablename) DESC;
```

## Connection Pool Monitoring

### Check Active Connections

```sql
-- Current connections by state
SELECT 
    state,
    COUNT(*) AS connections,
    MAX(now() - state_change) AS longest_in_state
FROM pg_stat_activity
WHERE datname = current_database()
GROUP BY state
ORDER BY connections DESC;

-- Long-running queries (potential issues)
SELECT 
    pid,
    now() - pg_stat_activity.query_start AS duration,
    state,
    query
FROM pg_stat_activity
WHERE datname = current_database()
  AND state != 'idle'
  AND now() - pg_stat_activity.query_start > interval '30 seconds'
ORDER BY duration DESC;
```

### Connection Limits

```sql
-- Current vs max connections
SELECT 
    max_conn, 
    used, 
    max_conn - used AS available
FROM 
    (SELECT COUNT(*) AS used FROM pg_stat_activity) t1,
    (SELECT setting::int AS max_conn FROM pg_settings WHERE name = 'max_connections') t2;
```

### EF Core Connection Pool Configuration

```csharp
// In Program.cs or DbContext configuration
services.AddDbContext<ApplicationDbContext>(options =>
{
    options.UseNpgsql(connectionString, npgsql =>
    {
        npgsql.CommandTimeout(30); // query timeout in seconds
    });
});

// Connection pool settings in connection string
// "Host=...;Port=5432;Database=...;Username=...;Password=...;
//  Minimum Pool Size=5;Maximum Pool Size=20;Connection Idle Lifetime=300"
```

## VACUUM and Maintenance

### Autovacuum Settings

PostgreSQL runs autovacuum automatically. Check if it is keeping up:

```sql
-- Check autovacuum settings
SELECT name, setting, short_desc
FROM pg_settings
WHERE name LIKE '%autovacuum%'
ORDER BY name;

-- Tables where autovacuum may be falling behind
SELECT 
    relname AS table_name,
    n_dead_tup,
    last_autovacuum,
    autovacuum_count
FROM pg_stat_user_tables
WHERE n_dead_tup > 10000
ORDER BY n_dead_tup DESC;
```

### Manual VACUUM (Rarely Needed)

```sql
-- Analyze + vacuum a specific table
VACUUM (ANALYZE, VERBOSE) orders;

-- Full vacuum (rewrites table, requires exclusive lock — use with caution)
VACUUM (FULL, ANALYZE) orders;
```

**Warning:** `VACUUM FULL` locks the table exclusively. Only use during maintenance windows.

## Kibana Dashboard for DB Logs

### What to Log from EF Core

In `appsettings.json`, configure EF Core logging:

```json
{
  "Serilog": {
    "MinimumLevel": {
      "Override": {
        "Microsoft.EntityFrameworkCore.Database.Command": "Information",
        "Microsoft.EntityFrameworkCore.Infrastructure": "Warning"
      }
    }
  }
}
```

This sends all SQL commands to the log pipeline (Serilog -> RMQ -> LogIngest -> Elasticsearch).

### Kibana Query Patterns

**Find slow EF Core queries:**
```
logger: "Microsoft.EntityFrameworkCore.Database.Command" AND elapsed_ms: >100
```

**Find N+1 patterns (many similar queries in sequence):**
```
logger: "Microsoft.EntityFrameworkCore.Database.Command" AND message: "SELECT" AND correlation_id: "<request-id>"
```
Group by `correlation_id` and look for requests with unusually high query counts.

**Find connection issues:**
```
logger: "Npgsql" AND level: "Error"
```

### Key Metrics to Track

| Metric | Source | Alert Threshold |
|--------|--------|----------------|
| Slow queries (>100ms) | pg_stat_statements | >10/minute |
| Dead tuples ratio | pg_stat_user_tables | >20% dead tuples |
| Connection count | pg_stat_activity | >80% of max_connections |
| Index hit ratio | pg_statio_user_tables | <95% |
| Sequential scans on large tables | pg_stat_user_tables | >100/minute |
| Lock waits | pg_stat_activity | Any wait >5s |

### Index Hit Ratio

```sql
-- Overall index hit ratio (should be >95%)
SELECT 
    sum(idx_blks_hit) AS index_hits,
    sum(idx_blks_read) AS index_reads,
    CASE WHEN sum(idx_blks_hit + idx_blks_read) > 0
        THEN round(100.0 * sum(idx_blks_hit) / sum(idx_blks_hit + idx_blks_read), 2)
        ELSE 0
    END AS hit_ratio_pct
FROM pg_statio_user_indexes;

-- Cache hit ratio (should be >99%)
SELECT 
    sum(heap_blks_hit) AS cache_hits,
    sum(heap_blks_read) AS disk_reads,
    CASE WHEN sum(heap_blks_hit + heap_blks_read) > 0
        THEN round(100.0 * sum(heap_blks_hit) / sum(heap_blks_hit + heap_blks_read), 2)
        ELSE 0
    END AS hit_ratio_pct
FROM pg_statio_user_tables;
```

## Periodic Health Check Script

Run this periodically (daily or weekly) to catch issues early:

```sql
-- 1. Unused indexes (wasting space and slowing writes)
-- 2. Tables with high dead tuple ratio
-- 3. Long-running queries
-- 4. Connection count
-- 5. Table sizes
-- 6. Slow query patterns from pg_stat_statements
```

These queries are listed in the sections above. Combine them into a monitoring script or scheduled job via the Worker Agent.
