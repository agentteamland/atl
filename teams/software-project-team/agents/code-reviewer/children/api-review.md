---
knowledge-base-summary: "Evaluate endpoint design for RESTful conventions, response format consistency, missing authorization, missing rate limiting, missing idempotency on POST, and inconsistent error responses. The API is the public contract."
---
# API Review

## Endpoint Design (RESTful Conventions)

### What to Check

| HTTP Method | Purpose | URL Pattern | Example |
|-------------|---------|-------------|---------|
| GET | Read (single) | `/{resource}/{id}` | `GET /orders/123` |
| GET | Read (list) | `/{resource}` | `GET /orders` |
| POST | Create | `/{resource}` | `POST /orders` |
| PUT | Full update | `/{resource}/{id}` | `PUT /orders/123` |
| PATCH | Partial update | `/{resource}/{id}` | `PATCH /orders/123` |
| DELETE | Delete | `/{resource}/{id}` | `DELETE /orders/123` |

### Red Flags

- **Verb in URL:** `POST /create-order` -- the HTTP method IS the verb
- **Singular resource name:** `GET /order/123` -- should be plural `/orders/123`
- **Inconsistent pluralization:** `/orders` in one place, `/user` in another
- **Action endpoints:** `POST /orders/123/activate` -- acceptable for non-CRUD operations, but should be rare
- **Deeply nested URLs:** `GET /users/123/orders/456/items/789` -- more than 2 levels deep is a smell
- **Query parameters in POST body:** Using POST for what should be a GET with query params
- **GET with side effects:** A GET endpoint that modifies data

## Response Format Consistency

### What to Check

All endpoints should return responses in the same format:

**Success responses:**
```json
{
  "id": "uuid",
  "name": "Order #1",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

**List responses:**
```json
{
  "items": [...],
  "cursor": "encoded-cursor",
  "hasMore": true
}
```

**Error responses:**
```json
{
  "type": "https://tools.ietf.org/html/rfc7231#section-6.5.4",
  "title": "Not Found",
  "status": 404,
  "detail": "Order with ID 123 was not found"
}
```

### Red Flags

- Some endpoints return `{ data: ... }` wrapper, others return raw objects
- Inconsistent date format (ISO 8601 vs Unix timestamp vs custom format)
- Inconsistent null handling (some return `null`, others omit the field)
- Error responses not using ProblemDetails format
- List endpoints with different pagination formats

## Missing Authorization

### What to Check

- Every endpoint that returns user-specific data MUST have `.RequireAuthorization()`
- Endpoints that modify data MUST have role-based or policy-based authorization
- Admin-only endpoints MUST require specific roles
- Internal endpoints MUST use `InternalSecretFilter`
- Public endpoints (login, register, health check) should be explicitly documented as public

### Red Flags

- Endpoint returns user data but has no auth requirement
- Endpoint modifies data but anyone can call it
- All endpoints use the same generic authorization -- no role differentiation
- Authorization checked in the handler instead of the endpoint declaration (inconsistent with project pattern)

## Missing Rate Limiting

### What to Check

| Endpoint Type | Rate Limit Needed | Why |
|---------------|-------------------|-----|
| Login | Yes (strict) | Brute force prevention |
| Register | Yes (strict) | Spam prevention |
| Password reset | Yes (strict) | Email bombing |
| Verification | Yes (strict) | Code guessing |
| File upload | Yes | Resource exhaustion |
| Search/List | Yes (moderate) | Scraping prevention |
| Create/Update | Optional | Depends on business logic |
| Read (single) | Optional | Usually not needed |

### Red Flags

- Auth endpoints without `.RequireRateLimiting()`
- File upload endpoint without size and rate limits
- Search endpoint without rate limiting (enables data scraping)

## Missing Idempotency on POST

### What to Check

- **Create endpoints:** POST endpoints that create resources should support idempotency via `X-Idempotency-Key` header.
- **Payment/financial endpoints:** Any endpoint that processes money MUST be idempotent.
- **Notification triggers:** Any endpoint that sends emails or notifications MUST be idempotent (prevents duplicate notifications).

### Implementation Check

Commands that need idempotency should implement `IIdempotent`:

```csharp
public record CreateOrderCommand(...) : ICommand<CreateOrderResponse>, IIdempotent;
```

### Red Flags

- POST endpoint that creates a resource without idempotency support
- Financial transaction endpoint without idempotency
- Notification-triggering endpoint without idempotency

## Inconsistent Error Responses

### What to Check

- All endpoints should return errors via the global exception handler (ProblemDetails format)
- No endpoint should manually construct error responses
- Error status codes should be consistent (404 for not found, 422 for validation, 403 for forbidden)

### Red Flags

```csharp
// BAD: Manual error response -- bypasses global handler
if (user == null)
    return Results.NotFound(new { message = "User not found" });

// CORRECT: Throw exception, global handler produces ProblemDetails
var user = await _db.Users.FindAsync(id, ct)
    ?? throw new NotFoundException(nameof(User), id);
```

- Different error formats from different endpoints
- Some endpoints return 400 for validation, others return 422
- Some endpoints return `{ error: "..." }`, others return ProblemDetails

## Endpoint Structure

### What to Check

Endpoints should be thin bridges -- parse HTTP, call mediator, return response:

```csharp
static async Task<IResult> CreateOrderAsync(
    CreateOrderRequest request,
    IMediator mediator,
    CancellationToken ct)
{
    var response = await mediator.Send(
        new CreateOrderCommand(request.Name, request.Items), ct);
    return Results.Created($"/orders/{response.Id}", response);
}
```

### Red Flags

- Business logic in the endpoint method
- Database access in the endpoint
- Try-catch in the endpoint
- Multiple mediator calls in one endpoint
- More than 10 lines of code in an endpoint method

## Review Checklist

- [ ] URLs follow RESTful conventions (plural nouns, no verbs)
- [ ] Response format is consistent across all endpoints (ProblemDetails for errors)
- [ ] All non-public endpoints have RequireAuthorization
- [ ] Role/policy-based auth where needed
- [ ] Rate limiting on auth and sensitive endpoints
- [ ] Idempotency support on POST endpoints that create resources
- [ ] Error responses use global exception handler (no manual error construction)
- [ ] Endpoints are thin (parse, send, return -- no logic)
- [ ] Date format is ISO 8601 consistently
- [ ] Pagination format is consistent across list endpoints
