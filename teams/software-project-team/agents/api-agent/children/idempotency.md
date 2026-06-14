---
knowledge-base-summary: "Client sends `X-Idempotency-Key` header (UUID v4). `IIdempotent` interface on commands → IdempotencyBehavior checks Redis SETNX. Already processed → return cached response, handler doesn't run. Race condition protection via \"processing\" flag. Key is optional — null means behavior is skipped."
---
# Idempotency: When the Same Request Arrives Twice

## Problem

In mobile applications and distributed systems, the same request can arrive multiple times:
- Network timeout → client retries
- User presses button twice
- Load balancer retry
- Queue consumer crash → message redelivery

Result: order created twice, payment charged twice, email sent twice.

## Solution: Idempotency Key

The client sends a unique `X-Idempotency-Key` header with every mutating request (POST, PUT, DELETE). The API checks this key in Redis — if already processed, returns the same response; if not processed, processes it and caches the result.

## Flow

```
Client → POST /api/orders (X-Idempotency-Key: "abc-123")
    ↓
IdempotencyBehavior (Mediator pipeline):
    ↓
Redis: SETNX idempotency:abc-123
    ├── Key already exists → return cached response (handler does not run)
    └── Key doesn't exist → handler runs → response written to Redis → response returned
```

## Which Operations Should Be Idempotent?

| Operation | Idempotency | Why |
|-----------|-------------|-----|
| Record creation (Create) | **YES** | Repeat → duplication |
| Payment transaction | **YES** | Repeat → double charge |
| Email/notification trigger | **YES** | Repeat → double email |
| Record update (Update) | Optional | Updating with the same value may be harmless |
| Record deletion (Delete) | Optional | Already deleted → NotFoundException |
| Read (Query/GET) | **NO** | Idempotent by nature |

## IIdempotent Interface

Create commands implement the `IIdempotent` interface — the pipeline behavior kicks in automatically:

```csharp
public interface IIdempotent
{
    // Idempotency key is taken from the Header and bound to the command
    string? IdempotencyKey { get; }
}

record CreateOrderCommand(
    Guid CustomerId,
    Guid CartId,
    string? IdempotencyKey = null
) : IRequest<CreateOrderResponse>, IIdempotent;
```

## IdempotencyBehavior (Pipeline)

```csharp
public sealed class IdempotencyBehavior<TRequest, TResponse>
    : IPipelineBehavior<TRequest, TResponse>
    where TRequest : IIdempotent
{
    private readonly IConnectionMultiplexer _redis;
    private readonly ILogger<IdempotencyBehavior<TRequest, TResponse>> _logger;
    private const string Prefix = "idempotency:";
    private static readonly TimeSpan DefaultTtl = TimeSpan.FromHours(24);

    public async ValueTask<TResponse> Handle(
        TRequest request,
        MessageHandlerDelegate<TRequest, TResponse> next,
        CancellationToken ct)
    {
        // If no idempotency key, normal flow (optional usage)
        if (string.IsNullOrEmpty(request.IdempotencyKey))
            return await next(request, ct);

        var redisDb = _redis.GetDatabase();
        var cacheKey = $"{Prefix}{request.IdempotencyKey}";

        // 1. Was it already processed?
        var cached = await redisDb.StringGetAsync(cacheKey);
        if (cached.HasValue)
        {
            _logger.LogInformation(
                "Idempotency HIT: key {Key} already processed, returning cached response",
                request.IdempotencyKey);
            return JsonSerializer.Deserialize<TResponse>(cached!)!;
        }

        // 2. Set processing flag (race condition prevention)
        var acquired = await redisDb.StringSetAsync(
            cacheKey, "processing", DefaultTtl, When.NotExists);

        if (!acquired)
        {
            // Another pod is processing the same key simultaneously — wait briefly and read from cache
            _logger.LogWarning(
                "Idempotency CONFLICT: key {Key} is being processed by another instance",
                request.IdempotencyKey);
            await Task.Delay(500, ct);
            cached = await redisDb.StringGetAsync(cacheKey);
            if (cached.HasValue && cached != "processing")
                return JsonSerializer.Deserialize<TResponse>(cached!)!;

            throw new ConflictException("Request is already being processed");
        }

        // 3. Run the handler
        var response = await next(request, ct);

        // 4. Cache the result (processing → actual result)
        await redisDb.StringSetAsync(
            cacheKey,
            JsonSerializer.Serialize(response),
            DefaultTtl);

        _logger.LogInformation(
            "Idempotency SET: key {Key} processed and cached",
            request.IdempotencyKey);

        return response;
    }
}
```

## Header Binding at the Endpoint

The `X-Idempotency-Key` header is bound to the command at the API endpoint:

```csharp
group.MapPost("/orders", async (
    [FromBody] CreateOrderRequest request,
    [FromHeader(Name = "X-Idempotency-Key")] string? idempotencyKey,
    IMediator mediator,
    CancellationToken ct) =>
{
    var command = new CreateOrderCommand(
        request.CustomerId,
        request.CartId,
        IdempotencyKey: idempotencyKey);

    var response = await mediator.Send(command, ct);
    return Results.Created($"/api/orders/{response.Id}", response);
})
.WithName("CreateOrder");
```

## Client Side (Flutter/React)

The client generates a UUID v4 and sends it as a header with every mutating request:

```dart
// Flutter:
final response = await dio.post('/api/orders',
  data: orderData,
  options: Options(headers: {
    'X-Idempotency-Key': Uuid().v4(),
  }),
);
```

On retry, the same key is sent again — this way the same operation does not run again.

## Redis Key Convention

```
idempotency:{key}  →  "processing" (in progress) or JSON response (completed)
TTL: 24 hours (if the same key arrives again within 24 hours, it returns from cache)
```

## Pipeline Behavior Order (Updated)

```
Logging → UnhandledException → Validation → Idempotency → Caching → Performance
                                                ↑
                                    (duplicate prevention on create commands)
```

Idempotency comes AFTER Validation — invalid requests are already filtered at the Validator, preventing unnecessary key writes to Redis.

## Important Rules

1. **Idempotency is NOT used on GET requests.** Queries are idempotent by nature.
2. **Key is optional.** If `IdempotencyKey` comes as null, the behavior is skipped and normal flow continues. This provides flexibility — it doesn't force every endpoint.
3. **TTL is 24 hours.** Long enough to catch retries, short enough to prevent Redis from bloating.
4. **Race condition protection.** `SETNX` + "processing" flag → two pods cannot start processing the same key at the same time.
5. **Response is cached.** When the second request arrives, the handler does not run, and the first response is returned as-is — the client does not see a different result.

