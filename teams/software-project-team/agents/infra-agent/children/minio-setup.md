---
knowledge-base-summary: "S3-compatible object storage for development. Bucket initialization. Console UI access. Environment variables for API integration. Production equivalent: AWS S3, GCP Cloud Storage."
---
# MinIO Setup

S3-compatible object storage for development. Use MinIO locally, swap to AWS S3 / GCP Cloud Storage in production with zero code changes.

## Why MinIO?

- Same S3 API as AWS -- your code uses `AWSSDK.S3` or any S3-compatible SDK
- Local development without cloud credentials or costs
- Web console for visual file browsing
- Bucket versioning, lifecycle policies, presigned URLs -- all work identically

## Docker Compose Service

```yaml
minio:
  image: minio/minio:latest
  container_name: ${PROJECT_PREFIX}-minio
  command: server /data --console-address ":9001"
  environment:
    - MINIO_ROOT_USER=${MINIO_ROOT_USER:-minioadmin}
    - MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD:-minioadmin}
  ports:
    - "${MINIO_API_PORT:-9000}:9000"       # S3 API
    - "${MINIO_CONSOLE_PORT:-9001}:9001"   # Web Console
  volumes:
    - minio_data:/data
  healthcheck:
    test: ["CMD", "mc", "ready", "local"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 10s
  restart: unless-stopped

volumes:
  minio_data:
    name: ${PROJECT_PREFIX}_minio_data
```

## Environment Variables

```bash
# .env
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001

# Application configuration (for .NET services)
S3__ServiceUrl=http://minio:9000
S3__AccessKey=minioadmin
S3__SecretKey=minioadmin
S3__BucketName=uploads
S3__ForcePathStyle=true    # Required for MinIO (not virtual-hosted style)
```

### .env.example Entry

```bash
# MinIO (S3-compatible storage)
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
```

## Bucket Initialization

Buckets must exist before the application writes to them. Use a one-shot init container.

### Option 1: Init Container (Recommended)

```yaml
minio-init:
  image: minio/mc:latest
  container_name: ${PROJECT_PREFIX}-minio-init
  depends_on:
    minio:
      condition: service_healthy
  entrypoint: >
    /bin/sh -c "
    mc alias set local http://minio:9000 ${MINIO_ROOT_USER:-minioadmin} ${MINIO_ROOT_PASSWORD:-minioadmin};
    mc mb local/uploads --ignore-existing;
    mc mb local/avatars --ignore-existing;
    mc mb local/exports --ignore-existing;
    mc anonymous set download local/avatars;
    echo 'Buckets initialized successfully';
    exit 0;
    "
```

### What This Does

1. Creates an alias pointing `mc` to the MinIO instance
2. Creates buckets (`--ignore-existing` makes it idempotent)
3. Sets public read policy on avatars bucket (profile pictures)
4. Exits with 0 -- the container stops after init, not restarted

### Option 2: Application-Level Initialization

```csharp
// In your .NET startup or a hosted service
public class MinioInitializer : IHostedService
{
    public async Task StartAsync(CancellationToken ct)
    {
        var client = new AmazonS3Client(
            accessKey, secretKey,
            new AmazonS3Config
            {
                ServiceURL = "http://minio:9000",
                ForcePathStyle = true
            });

        var buckets = new[] { "uploads", "avatars", "exports" };
        foreach (var bucket in buckets)
        {
            if (!await AmazonS3Util.DoesS3BucketExistV2Async(client, bucket))
                await client.PutBucketAsync(bucket, ct);
        }
    }

    public Task StopAsync(CancellationToken ct) => Task.CompletedTask;
}
```

## Console Access

```
URL:  http://localhost:9001
User: minioadmin (value of MINIO_ROOT_USER)
Pass: minioadmin (value of MINIO_ROOT_PASSWORD)
```

From the console you can:
- Browse buckets and objects
- Upload/download files manually
- Set bucket policies (public/private)
- View access logs
- Manage users and service accounts

## API Integration

The same S3 SDK works with both MinIO (dev) and AWS S3 (prod). The only difference is configuration.

### .NET Configuration

```csharp
// appsettings.Development.json (MinIO)
{
  "S3": {
    "ServiceUrl": "http://minio:9000",
    "AccessKey": "minioadmin",
    "SecretKey": "minioadmin",
    "BucketName": "uploads",
    "ForcePathStyle": true
  }
}

// appsettings.Production.json (AWS S3)
{
  "S3": {
    "Region": "eu-central-1",
    "BucketName": "myproject-uploads",
    "ForcePathStyle": false
    // AccessKey/SecretKey from IAM role, not config
  }
}
```

### Service Registration

```csharp
// DI registration -- works for both MinIO and AWS
builder.Services.AddSingleton<IAmazonS3>(sp =>
{
    var config = new AmazonS3Config();
    var s3Config = builder.Configuration.GetSection("S3");

    if (!string.IsNullOrEmpty(s3Config["ServiceUrl"]))
    {
        config.ServiceURL = s3Config["ServiceUrl"];
        config.ForcePathStyle = s3Config.GetValue<bool>("ForcePathStyle");
        return new AmazonS3Client(
            s3Config["AccessKey"],
            s3Config["SecretKey"],
            config);
    }

    // Production: uses IAM role or environment credentials
    config.RegionEndpoint = RegionEndpoint.GetBySystemName(s3Config["Region"]);
    return new AmazonS3Client(config);
});
```

## Volume for Persistent Storage

```yaml
volumes:
  minio_data:
    name: ${PROJECT_PREFIX}_minio_data
```

Data persists across `docker compose down` / `docker compose up` cycles.

### To Reset Storage

```bash
# Remove the volume (deletes all uploaded files)
docker volume rm ${PROJECT_PREFIX}_minio_data

# Or nuke everything
docker compose down -v
```

## Presigned URLs (for Client Uploads)

```csharp
// Generate a presigned URL for direct client upload
var request = new GetPreSignedUrlRequest
{
    BucketName = "uploads",
    Key = $"user-{userId}/{Guid.NewGuid()}.jpg",
    Verb = HttpVerb.PUT,
    Expires = DateTime.UtcNow.AddMinutes(15),
    ContentType = "image/jpeg"
};

string presignedUrl = _s3Client.GetPreSignedURL(request);
// Return this URL to the client -- they upload directly to MinIO/S3
```

## Full Docker Compose Snippet

```yaml
# --- Object Storage ---

minio:
  image: minio/minio:latest
  container_name: ${PROJECT_PREFIX}-minio
  command: server /data --console-address ":9001"
  environment:
    - MINIO_ROOT_USER=${MINIO_ROOT_USER:-minioadmin}
    - MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD:-minioadmin}
  ports:
    - "${MINIO_API_PORT:-9000}:9000"
    - "${MINIO_CONSOLE_PORT:-9001}:9001"
  volumes:
    - minio_data:/data
  healthcheck:
    test: ["CMD", "mc", "ready", "local"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 10s
  restart: unless-stopped

minio-init:
  image: minio/mc:latest
  container_name: ${PROJECT_PREFIX}-minio-init
  depends_on:
    minio:
      condition: service_healthy
  entrypoint: >
    /bin/sh -c "
    mc alias set local http://minio:9000 ${MINIO_ROOT_USER:-minioadmin} ${MINIO_ROOT_PASSWORD:-minioadmin};
    mc mb local/uploads --ignore-existing;
    mc mb local/avatars --ignore-existing;
    mc mb local/exports --ignore-existing;
    mc anonymous set download local/avatars;
    echo 'Buckets initialized';
    exit 0;
    "

volumes:
  minio_data:
    name: ${PROJECT_PREFIX}_minio_data
```

## Troubleshooting

### "Access Denied" from application

- Check `ForcePathStyle=true` is set (required for MinIO)
- Verify `ServiceUrl` uses `http://minio:9000` (Docker network name, not localhost)
- Confirm bucket exists: check via console at http://localhost:9001

### "Bucket does not exist"

- minio-init container may not have run yet
- Check: `docker compose logs minio-init`
- Run manually: `docker compose run --rm minio-init`

### Files disappear after restart

- Ensure volume is declared and attached (not using tmpfs or anonymous volume)
- Check: `docker volume inspect ${PROJECT_PREFIX}_minio_data`
