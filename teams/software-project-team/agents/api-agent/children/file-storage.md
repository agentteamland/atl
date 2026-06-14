---
knowledge-base-summary: "MinIO (S3-compatible) in dev, AWS S3 in production. Entity-based path default: `{entity}/{id}/{purpose}-{guid}.{ext}`. Handler can override with custom path. `IStorageService` in Application, `S3StorageService` in Infrastructure. Integrates with soft delete — hard delete phase removes entire entity directory."
---
# File Upload & Storage Pattern

## Infrastructure

In the development environment, **MinIO** (S3-compatible object storage) is used — same API as production. There are no two different behaviors like "save to disk locally / save to S3 in production."

In Docker Compose:
```yaml
minio:
  image: minio/minio
  container_name: wfm-minio
  ports:
    - "${MINIO_API_PORT:-9000}:9000"
    - "${MINIO_CONSOLE_PORT:-9001}:9001"
  environment:
    MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
    MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin}
  volumes:
    - minio_data:/data
  command: server /data --console-address ":9001"
  healthcheck:
    test: ["CMD", "mc", "ready", "local"]
    interval: 5s
    timeout: 5s
    retries: 5
```

## IStorageService (Application layer)

```csharp
public interface IStorageService
{
    Task<StorageResult> UploadAsync(
        Stream file,
        string path,
        string contentType,
        CancellationToken ct = default);

    Task<Stream> DownloadAsync(
        string path,
        CancellationToken ct = default);

    Task DeleteAsync(
        string path,
        CancellationToken ct = default);

    Task DeleteDirectoryAsync(
        string prefix,
        CancellationToken ct = default);

    string GetPublicUrl(string path);

    Task<string> GetSignedUrlAsync(
        string path,
        TimeSpan expiry,
        CancellationToken ct = default);
}

public record StorageResult(
    string Path,
    string Url,
    long SizeBytes,
    string ContentType);
```

## Implementation (Infrastructure layer)

`S3StorageService` — uses AWS SDK, compatible with both MinIO and AWS S3:

```csharp
public class S3StorageService : IStorageService
{
    private readonly IAmazonS3 _s3;
    private readonly StorageOptions _options;

    // MinIO locally, AWS S3 in production — only the endpoint URL changes
}
```

## File Path Convention

### Default: Entity-Based Path

If the handler does not specify a path, the generic structure takes over: `{entity}/{id}/{filename}`

```
products/abc-123/photo-a1b2c3d4.jpg
products/abc-123/gallery-d5e6f7g8.jpg
users/def-456/avatar-h9i0j1k2.png
orders/ghi-789/invoice-l3m4n5o6.pdf
```

### Custom Path Override

The handler can specify a different path if desired. Some processes may require a custom directory structure:

```csharp
// Default — generic structure generates the path:
var path = StoragePathHelper.Generate("products", product.Id, "photo", ext);
// → "products/abc-123/photo-a1b2c3d4.jpg"

// Custom path — handler determines it:
var path = $"exports/reports/{DateTime.UtcNow:yyyy/MM}/{reportId}{ext}";
// → "exports/reports/2026/04/abc123.pdf"

// Shared/public directory:
var path = $"public/banners/{campaignId}-{Guid.NewGuid()}{ext}";
// → "public/banners/camp-123-a1b2c3d4.jpg"
```

### StoragePathHelper (Generic Path Generator)

```csharp
public static class StoragePathHelper
{
    public static string Generate(string entity, Guid entityId, string purpose, string extension)
    {
        return $"{entity}/{entityId}/{purpose}-{Guid.NewGuid()}{extension}";
    }
}
```

If the handler does not provide a path, `StoragePathHelper.Generate()` is used. If the handler provides a custom path, it is used directly.

### Why Entity-Based Default:
- Works with soft delete — when the entity is deleted (at the hard delete stage), all files are deleted in a single move with `DeleteDirectoryAsync("products/abc-123/")`
- Readable — the file path reveals which entity it belongs to
- Easy bulk cleanup — directory deletion by entity

### Filename Convention:
- Original filename is not preserved (security + collision risk)
- `{purpose}-{guid}.{ext}` format: `photo-a1b2c3.jpg`, `avatar-d4e5f6.png`
- Purpose examples: `photo`, `avatar`, `document`, `invoice`, `gallery`

## Upload Flow

```
Client (multipart/form-data)
    ↓
API Endpoint: parse file + metadata
    ↓
Handler:
    1. Validate: size limit, allowed extensions, content type
    2. Generate path: {entity}/{id}/{purpose}-{guid}.{ext}
    3. _storageService.UploadAsync(stream, path, contentType)
    4. Save URL/path to entity (DB)
    5. Log: file uploaded, size, path
    ↓
Response: StorageResult (path, url, size)
```

## Handler Example

```csharp
public async Task<UploadProductPhotoResponse> Handle(
    UploadProductPhotoCommand request, CancellationToken ct)
{
    _logger.LogInformation(
        "Uploading product photo for {ProductId}, size {Size} bytes",
        request.ProductId, request.File.Length);

    var product = await _db.Products.FindAsync(request.ProductId, ct)
        ?? throw new NotFoundException(nameof(Product), request.ProductId);

    // Validate
    var maxSize = await _settings.GetAsync<int>("upload:max-file-size-mb", 10, ct);
    if (request.File.Length > maxSize * 1024 * 1024)
        throw new ValidationException($"File size exceeds {maxSize}MB limit");

    var allowedTypes = new[] { "image/jpeg", "image/png", "image/webp" };
    if (!allowedTypes.Contains(request.File.ContentType))
        throw new ValidationException("Invalid file type");

    // Upload
    var ext = Path.GetExtension(request.File.FileName);
    var path = $"products/{product.Id}/photo-{Guid.NewGuid()}{ext}";

    var result = await _storageService.UploadAsync(
        request.File.OpenReadStream(), path, request.File.ContentType, ct);

    // Update DB
    product.PhotoPath = result.Path;
    product.PhotoUrl = result.Url;
    await _db.SaveChangesAsync(ct);

    _logger.LogInformation(
        "Product photo uploaded: {ProductId}, path {Path}, size {Size}",
        product.Id, result.Path, result.SizeBytes);

    return new UploadProductPhotoResponse(result.Path, result.Url);
}
```

## Validation Rules

File validation is done in the handler, read from dynamic settings:

| Setting | Settings Key | Default |
|---------|-------------|---------|
| Max file size | `upload:max-file-size-mb` | 10 MB |
| Allowed image types | `upload:allowed-image-types` | `image/jpeg,image/png,image/webp` |
| Allowed document types | `upload:allowed-document-types` | `application/pdf` |

## Bulk Cleanup (Soft Delete Integration)

When an entity reaches the hard-delete stage (bulk cleanup Worker job):

```csharp
// Worker job or admin endpoint:
var deletedProducts = await _db.Products
    .IgnoreQueryFilters()
    .Where(p => p.IsDeleted && p.DeletedAt < DateTime.UtcNow.AddDays(-90))
    .ToListAsync(ct);

foreach (var product in deletedProducts)
{
    // Delete all files in a single move
    await _storageService.DeleteDirectoryAsync($"products/{product.Id}/", ct);

    // Physically delete the entity
    _db.Products.Remove(product);

    _logger.LogInformation(
        "Hard-deleted product {ProductId} and all storage files", product.Id);
}

await _db.SaveChangesAsync(ct);
```

## Signed URLs

For private files (invoice, private document), pre-signed URLs:

```csharp
// In handler:
var signedUrl = await _storageService.GetSignedUrlAsync(
    order.InvoicePath,
    TimeSpan.FromMinutes(15),  // valid for 15 min
    ct);
```

This URL redirects directly to MinIO/S3 — no proxy through the API is needed, performant.

