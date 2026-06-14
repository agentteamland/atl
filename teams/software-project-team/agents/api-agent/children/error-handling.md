---
knowledge-base-summary: "Exception hierarchy: `NotFoundException`→404, `ValidationException`→422, `ForbiddenException`→403, unhandled→500. No try-catch in handlers — throw the appropriate exception, global `IExceptionHandler` maps to HTTP status + ProblemDetails. Development shows details, production shows generic message."
---
# Error Handling Strategy

## Exception Hierarchy

Custom exceptions are defined in the Application layer:

| Exception | HTTP Status | When |
|-----------|-------------|------|
| `NotFoundException` | 404 | When the entity is not found |
| `ValidationException` | 422 | FluentValidation or manual validation error |
| `ForbiddenException` | 403 | Authorized but no access to this resource |
| `UnauthorizedAccessException` | 401 | Identity could not be verified |

## Error Handling in Handler

Try-catch is **NOT written** in the handler. On error conditions, the appropriate exception is thrown and the upper layer catches it:

```csharp
// ✅ Correct: Throw exception, upper layer catches it
var user = await _db.Users.FindAsync(request.UserId, ct)
    ?? throw new NotFoundException(nameof(User), request.UserId);

// ❌ Wrong: Writing try-catch in handler and doing result wrapping
try { ... } catch (Exception ex) { return Result.Failure(ex.Message); }
```

## Global Exception Handler (Api layer)

The `IExceptionHandler` implementation in the Api maps exception types to HTTP status codes:

```csharp
NotFoundException      → 404 + ProblemDetails
ValidationException    → 422 + ProblemDetails (errors array)
ForbiddenException     → 403 + ProblemDetails
_                      → 500 + ProblemDetails (detail in Development, generic in Production)
```

Neither the handler nor the endpoint ever writes try-catch. This responsibility belongs entirely to the global exception handler.

