---
knowledge-base-summary: "PostgreSQL pg_dump/pg_restore for database backup. Volume backup for persistent data. Automated backup via Worker job. Restore procedure for development (import production snapshot)."
---
# Backup & Restore

Database and volume backup strategies. From manual dev snapshots to automated production backups.

## PostgreSQL Backup

### Manual Backup (Development)

```bash
# Backup entire database to SQL file
docker compose exec db pg_dump \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  --no-owner \
  --no-privileges \
  > backup_$(date +%Y%m%d_%H%M%S).sql

# Example with concrete values
docker compose exec db pg_dump \
  -U example_app \
  -d example_app \
  --no-owner \
  --no-privileges \
  > backup_20260414_120000.sql
```

### Compressed Backup

```bash
# Backup with gzip compression (recommended for large databases)
docker compose exec db pg_dump \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  --no-owner \
  --no-privileges \
  | gzip > backup_$(date +%Y%m%d_%H%M%S).sql.gz
```

### Custom Format Backup (Parallel Restore)

```bash
# Custom format -- supports parallel restore, selective table restore
docker compose exec db pg_dump \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  -Fc \
  > backup_$(date +%Y%m%d_%H%M%S).dump
```

### Backup Specific Tables

```bash
# Backup only specific tables
docker compose exec db pg_dump \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  -t users \
  -t tasks \
  -t achievements \
  > partial_backup.sql
```

### Schema-Only Backup

```bash
# Backup only the schema (no data) -- useful for migrations reference
docker compose exec db pg_dump \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  --schema-only \
  > schema_backup.sql
```

## PostgreSQL Restore

### Restore from SQL File

```bash
# Restore SQL dump (stdin redirect requires -T flag for non-interactive)
docker compose exec -T db psql \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  < backup_20260414_120000.sql
```

### Restore from Compressed Backup

```bash
# Restore gzipped SQL dump
gunzip -c backup_20260414_120000.sql.gz | \
  docker compose exec -T db psql \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB}
```

### Restore Custom Format

```bash
# Restore custom format (supports parallel restore with -j)
docker compose exec -T db pg_restore \
  -U ${POSTGRES_USER} \
  -d ${POSTGRES_DB} \
  --no-owner \
  --no-privileges \
  -j 4 \
  < backup_20260414_120000.dump
```

### Clean Restore (Drop & Recreate)

```bash
# Drop existing database and recreate from backup
# WARNING: This deletes all current data

# Step 1: Drop and recreate database
docker compose exec db psql -U ${POSTGRES_USER} -c "
  SELECT pg_terminate_backend(pid) FROM pg_stat_activity
  WHERE datname = '${POSTGRES_DB}' AND pid <> pg_backend_pid();
"
docker compose exec db dropdb -U ${POSTGRES_USER} ${POSTGRES_DB}
docker compose exec db createdb -U ${POSTGRES_USER} ${POSTGRES_DB}

# Step 2: Restore
docker compose exec -T db psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} < backup.sql
```

## Volume Backup

For any Docker volume (not just PostgreSQL).

### Backup a Named Volume

```bash
# Generic volume backup using alpine container
docker run --rm \
  -v ${PROJECT_PREFIX}_postgres_data:/data:ro \
  -v $(pwd)/backups:/backup \
  alpine \
  tar czf /backup/postgres_data_$(date +%Y%m%d_%H%M%S).tar.gz -C /data .
```

### Restore a Volume

```bash
# Stop services using the volume first
docker compose stop db

# Restore volume from tar
docker run --rm \
  -v ${PROJECT_PREFIX}_postgres_data:/data \
  -v $(pwd)/backups:/backup \
  alpine \
  sh -c "rm -rf /data/* && tar xzf /backup/postgres_data_20260414_120000.tar.gz -C /data"

# Start services
docker compose start db
```

### Backup All Project Volumes

```bash
#!/bin/bash
# scripts/backup-all-volumes.sh

BACKUP_DIR="./backups/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"
PREFIX="${PROJECT_PREFIX:-wfm}"

VOLUMES=(
  "${PREFIX}_postgres_data"
  "${PREFIX}_redis_data"
  "${PREFIX}_elasticsearch_data"
  "${PREFIX}_minio_data"
  "${PREFIX}_rabbitmq_data"
)

for VOL in "${VOLUMES[@]}"; do
  if docker volume inspect "$VOL" > /dev/null 2>&1; then
    echo "Backing up $VOL..."
    docker run --rm \
      -v "$VOL":/data:ro \
      -v "$BACKUP_DIR":/backup \
      alpine \
      tar czf "/backup/${VOL}.tar.gz" -C /data .
    echo "  Done: $BACKUP_DIR/${VOL}.tar.gz"
  else
    echo "  Skipping $VOL (not found)"
  fi
done

echo "All backups saved to $BACKUP_DIR"
```

## Automated Backup

### Option 1: Worker Job (Recommended)

The Worker service runs scheduled jobs via Cronos. Add a backup job that runs daily.

```csharp
// In Worker project -- the backup job calls pg_dump inside the db container
// via a backup endpoint on the API, or directly via connection string

public class DatabaseBackupJob : IScheduledJob
{
    public string CronExpression => "0 2 * * *"; // Daily at 2 AM

    public async Task ExecuteAsync(CancellationToken ct)
    {
        // Option A: Call API backup endpoint
        await _apiClient.PostAsync("/internal/backup", null, ct);

        // Option B: Direct pg_dump via Npgsql
        // (requires backup logic in the Worker itself)
    }
}
```

### Option 2: Cron in Docker Compose

```yaml
# Dedicated backup service with cron
db-backup:
  image: postgres:17-alpine
  container_name: ${PROJECT_PREFIX}-db-backup
  environment:
    - PGHOST=db
    - PGUSER=${POSTGRES_USER}
    - PGPASSWORD=${POSTGRES_PASSWORD}
    - PGDATABASE=${POSTGRES_DB}
  volumes:
    - ./backups:/backups
    - ./scripts/backup-cron.sh:/backup.sh:ro
  entrypoint: >
    /bin/sh -c "
    echo '0 2 * * * /bin/sh /backup.sh >> /var/log/backup.log 2>&1' | crontab -;
    crond -f -l 2;
    "
  depends_on:
    db:
      condition: service_healthy
  restart: unless-stopped
```

```bash
#!/bin/sh
# scripts/backup-cron.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="/backups/${PGDATABASE}_${TIMESTAMP}.sql.gz"

pg_dump --no-owner --no-privileges | gzip > "$BACKUP_FILE"

# Keep only last 7 days of backups
find /backups -name "*.sql.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE"
```

### Option 3: Upload to S3/MinIO

```bash
#!/bin/bash
# scripts/backup-to-s3.sh

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="/tmp/${POSTGRES_DB}_${TIMESTAMP}.sql.gz"

# Create backup
docker compose exec -T db pg_dump \
  -U ${POSTGRES_USER} -d ${POSTGRES_DB} \
  --no-owner --no-privileges \
  | gzip > "$BACKUP_FILE"

# Upload to MinIO/S3
docker run --rm \
  --network ${PROJECT_PREFIX}_default \
  -v "$BACKUP_FILE":/backup.sql.gz:ro \
  -e MC_HOST_s3="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@minio:9000" \
  minio/mc:latest \
  cp /backup.sql.gz s3/backups/db/${POSTGRES_DB}_${TIMESTAMP}.sql.gz

rm "$BACKUP_FILE"
echo "Backup uploaded to S3: ${POSTGRES_DB}_${TIMESTAMP}.sql.gz"
```

## Restore to Development

Download a production backup and restore it locally for development.

### Full Workflow

```bash
# Step 1: Download production backup (from S3, server, etc.)
scp production-server:/backups/latest.sql.gz ./backups/

# Step 2: Stop API and other services that use the database
docker compose stop api socket worker

# Step 3: Drop and recreate database
docker compose exec db psql -U ${POSTGRES_USER} -c "
  SELECT pg_terminate_backend(pid) FROM pg_stat_activity
  WHERE datname = '${POSTGRES_DB}' AND pid <> pg_backend_pid();
"
docker compose exec db dropdb -U ${POSTGRES_USER} ${POSTGRES_DB} --if-exists
docker compose exec db createdb -U ${POSTGRES_USER} ${POSTGRES_DB}

# Step 4: Restore
gunzip -c ./backups/latest.sql.gz | \
  docker compose exec -T db psql -U ${POSTGRES_USER} -d ${POSTGRES_DB}

# Step 5: Run pending migrations (if any)
docker compose run --rm api dotnet ef database update

# Step 6: Start services
docker compose up -d
```

### Data Sanitization (Optional)

After restoring production data to development, sanitize sensitive data:

```sql
-- Anonymize user emails and passwords
UPDATE users SET
  email = 'user' || id || '@dev.local',
  password_hash = '$2a$11$devhashplaceholder',
  phone_number = NULL
WHERE email NOT LIKE '%@dev.local';

-- Clear sessions and tokens
TRUNCATE refresh_tokens;
TRUNCATE verification_tokens;
```

## Backup Directory Structure

```
backups/
  20260414_120000/
    wfm_postgres_data.tar.gz
    wfm_redis_data.tar.gz
    wfm_elasticsearch_data.tar.gz
  20260413_020000/
    example_app_20260413_020000.sql.gz
```

### .gitignore Entry

```
# Backup files
backups/
*.sql
*.sql.gz
*.dump
*.tar.gz
```

## Verification Checklist

After every backup, verify:

```bash
# 1. Check file exists and has non-zero size
ls -lh backups/latest.sql.gz

# 2. Test restore to a throwaway database
docker compose exec db createdb -U ${POSTGRES_USER} test_restore
gunzip -c backups/latest.sql.gz | \
  docker compose exec -T db psql -U ${POSTGRES_USER} -d test_restore
docker compose exec db psql -U ${POSTGRES_USER} -d test_restore \
  -c "SELECT count(*) FROM users;"
docker compose exec db dropdb -U ${POSTGRES_USER} test_restore

# 3. Log result
echo "Backup verified at $(date)"
```

A backup that has never been tested is not a backup.
