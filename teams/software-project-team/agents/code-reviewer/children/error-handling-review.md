---
knowledge-base-summary: "Verify error handling patterns: no try-catch in handlers, no swallowed exceptions, no generic catches, no missing validation, no error messages exposing internals. Error handling should be consistent with the project's exception hierarchy."
---
# Error Handling Review

## No Try-Catch in Handlers

### The Rule

Handlers MUST NOT contain try-catch blocks. The global exception handler in the Api layer is responsible for catching and mapping exceptions.

### What to Look For

```csharp
// BAD: Handler has try-catch
public async ValueTask<CreateOrderResponse> Handle(CreateOrderCommand request, CancellationToken ct)
{
    try
    {
        var order = new Order(request.Name);
        _db.Orders.Add(order);
        await _db.SaveChangesAsync(ct);
        return new CreateOrderResponse(order.Id);
    }
    catch (Exception ex)
    {
        _logger.LogError(ex, "Failed to create order");
        throw; // Even re-throwing is wrong -- the try-catch serves no purpose
    }
}
```

```csharp
// CORRECT: No try-catch, throw custom exceptions on business error
public async ValueTask<CreateOrderResponse> Handle(CreateOrderCommand request, CancellationToken ct)
{
    var user = await _db.Users.FindAsync(request.UserId, ct)
        ?? throw new NotFoundException(nameof(User), request.UserId);

    var order = new Order(request.Name, user.Id);
    _db.Orders.Add(order);
    await _db.SaveChangesAsync(ct);
    return new CreateOrderResponse(order.Id);
}
```

### Exceptions to the Rule

Try-catch is acceptable ONLY when:
- Catching a specific external service exception to convert it to a domain exception
- Implementing retry logic for transient failures (but prefer Polly policies instead)
- Handling `DbUpdateConcurrencyException` for optimistic concurrency

## Swallowed Exceptions

### What to Look For

- **Catch with no action:**
  ```csharp
  // CRITICAL: Exception is completely lost
  try { await DoWork(); }
  catch (Exception) { }
  ```
- **Catch with only logging (no rethrow):**
  ```csharp
  // BAD: Exception is logged but the caller thinks everything succeeded
  try { await DoWork(); }
  catch (Exception ex) { _logger.LogError(ex, "Error"); }
  ```
- **Catch returning a default value:**
  ```csharp
  // BAD: Caller gets null and has no idea why
  try { return await GetUser(id); }
  catch { return null; }
  ```

### Correct Pattern

If you must catch (rare cases), always rethrow or throw a domain exception:

```csharp
catch (HttpRequestException ex)
{
    _logger.LogError(ex, "External service failed for {OrderId}", orderId);
    throw new ExternalServiceException("Payment service unavailable", ex);
}
```

## Generic Exception Catch

### What to Look For

- **Catching `Exception` instead of specific types:**
  ```csharp
  // BAD: Catches everything, including OutOfMemoryException, StackOverflowException
  catch (Exception ex) { ... }
  ```
- **Catching `System.Exception` when only `DbUpdateException` is expected:**
  ```csharp
  // BAD: Too broad
  catch (Exception ex) { /* handle db error */ }
  // CORRECT: Specific
  catch (DbUpdateException ex) { /* handle db error */ }
  ```

### Rule

Catch the most specific exception type possible. If you don't know which exception to expect, you probably shouldn't be catching at all -- let the global handler deal with it.

## Missing Validation

### What to Look For

- **Command without a Validator class:** Every command MUST have a corresponding `AbstractValidator<TCommand>`. If a command file exists without a validator file in the same directory, flag it.
- **Missing null checks on critical inputs:** If the validator doesn't check for null/empty on required fields.
- **Missing range validation:** Numeric inputs without min/max checks.
- **Missing format validation:** Email, phone, URL fields without format validation.
- **Missing business rule validation:** Rules that can be checked statically (e.g., "end date must be after start date") should be in the validator, not the handler.

### Check Pattern

For every Command file, verify:
1. A Validator file exists in the same directory
2. The validator covers all required fields (not just one or two)
3. String fields have MaximumLength
4. Numeric fields have range constraints where applicable
5. Email/URL fields have format validation

## Error Messages Exposing Internals

### What to Look For

- **Stack traces in API responses:**
  ```csharp
  // BAD: Stack trace sent to client
  return Results.Problem(detail: ex.ToString());
  ```
- **SQL errors in responses:**
  ```csharp
  // BAD: Database schema leaked
  return Results.Problem(detail: ex.InnerException?.Message);
  ```
- **File paths in error messages:**
  ```csharp
  // BAD: Server file structure exposed
  throw new Exception($"File not found at /app/data/uploads/{filename}");
  ```
- **Connection strings or config in errors:**
  ```csharp
  // BAD: Database credentials in error
  _logger.LogError("Failed to connect to {ConnectionString}", connectionString);
  ```

### Correct Pattern

Production error responses should be generic:
```csharp
// Development: detailed error for debugging
// Production: generic message, details only in logs
return Results.Problem(
    statusCode: 500,
    title: "An error occurred",
    detail: env.IsDevelopment() ? ex.Message : "An unexpected error occurred. Please try again."
);
```

## Review Checklist

- [ ] No try-catch blocks in handlers (except documented exceptions)
- [ ] No swallowed exceptions (catch without rethrow)
- [ ] No generic Exception catches (catch specific types)
- [ ] Every Command has a corresponding Validator
- [ ] Validators cover all required fields with appropriate rules
- [ ] No stack traces or SQL errors in API responses
- [ ] No file paths or connection strings in error messages
- [ ] Production error messages are generic (details only in Development)
