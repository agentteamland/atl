---
knowledge-base-summary: "Check for missing logs, string interpolation instead of message templates, sensitive data in logs, missing log levels, and inconsistent property naming. Every handler must tell its story through logs."
---
# Logging Review

## Missing Logs

### The Rule

Every handler must tell its story through logs. Production has no breakpoints -- logs are the debugger.

### What to Look For

- **Handler with no log statements:** Every handler should have at minimum an entry log and an exit log.
- **Decision points without logs:** If there's an if/else or switch in the handler, each branch should log which path was taken and why.
- **External calls without logs:** Before and after calling an external service, a log should record what was requested and what was received.

### Expected Pattern

```csharp
public async ValueTask<CreateOrderResponse> Handle(CreateOrderCommand request, CancellationToken ct)
{
    _logger.LogInformation("Creating order for user {UserId} with {ItemCount} items",
        request.UserId, request.Items.Count);

    var user = await _db.Users.FindAsync(request.UserId, ct)
        ?? throw new NotFoundException(nameof(User), request.UserId);

    _logger.LogDebug("User {UserId} found, applying discount {DiscountRate}",
        user.Id, user.DiscountRate);

    var order = Order.Create(user, request.Items);
    _db.Orders.Add(order);
    await _db.SaveChangesAsync(ct);

    _logger.LogInformation("Order {OrderId} created for user {UserId}, total: {Total}",
        order.Id, user.Id, order.Total);

    return new CreateOrderResponse(order.Id);
}
```

## String Interpolation vs Message Template

### The Rule

NEVER use string interpolation (`$"..."`) in log messages. Always use message templates with named placeholders.

### What to Look For

```csharp
// BAD: String interpolation -- loses structured logging capability
_logger.LogInformation($"Order {orderId} created for user {userId}");

// BAD: String.Format -- same problem
_logger.LogInformation(string.Format("Order {0} created", orderId));

// BAD: String concatenation
_logger.LogInformation("Order " + orderId + " created");

// CORRECT: Message template with named placeholders
_logger.LogInformation("Order {OrderId} created for user {UserId}", orderId, userId);
```

### Why It Matters

Message templates allow structured logging backends (Elasticsearch, Seq) to index and search by individual properties. String interpolation produces a flat string that cannot be queried.

## Sensitive Data in Logs

### What to Look For

- **Passwords:**
  ```csharp
  // CRITICAL: Password in logs
  _logger.LogDebug("Login attempt for {Email} with password {Password}", email, password);
  ```
- **Tokens/Secrets:**
  ```csharp
  // CRITICAL: JWT token in logs
  _logger.LogInformation("Generated token {Token} for user {UserId}", token, userId);
  ```
- **Credit card / financial data:**
  ```csharp
  // CRITICAL: Credit card in logs
  _logger.LogInformation("Processing payment with card {CardNumber}", cardNumber);
  ```
- **Full request bodies:** Logging entire request objects that may contain sensitive fields.
  ```csharp
  // DANGEROUS: Request may contain password, token, etc.
  _logger.LogDebug("Request: {@Request}", request);
  ```

### Safe Logging Pattern

```csharp
// Log only non-sensitive identifiers
_logger.LogInformation("Login attempt for {Email}", email);
// Mask sensitive data if it must be logged
_logger.LogDebug("Token generated for user {UserId}, expires {Expiry}", userId, expiry);
// Be selective about request logging
_logger.LogDebug("Creating order for user {UserId} with {ItemCount} items",
    request.UserId, request.Items.Count);
```

## Log Level Misuse

### What to Look For

| Level | When to Use | Common Misuse |
|-------|------------|---------------|
| `Trace` | Step-by-step debugging detail | Used for important events |
| `Debug` | Diagnostic info, values of variables | Used in production for normal flow |
| `Information` | Normal application flow | Used for errors or warnings |
| `Warning` | Unexpected but recoverable situation | Used for normal flow |
| `Error` | Operation failed, needs attention | Used for expected business exceptions |
| `Critical` | Application crash, unrecoverable | Used for regular errors |

### Red Flags

- `LogError` for expected business cases (user not found, validation failed)
- `LogInformation` for errors or exceptions
- `LogWarning` for normal successful operations
- `LogDebug` or `LogTrace` for important business events
- Inconsistent levels for the same type of event across handlers

## Inconsistent Property Naming

### What to Look For

- Same concept, different names in log properties:
  ```csharp
  // BAD: UserId vs User_Id vs userId vs user
  _logger.LogInformation("Processing order for {UserId}", userId);
  _logger.LogInformation("Fetched data for {User_Id}", userId);
  _logger.LogInformation("Completed for {user}", userId);
  ```
- Property names should be PascalCase and consistent:
  ```csharp
  // CORRECT: Always {UserId}
  _logger.LogInformation("Processing order for {UserId}", userId);
  _logger.LogInformation("Fetched data for {UserId}", userId);
  _logger.LogInformation("Completed for {UserId}", userId);
  ```

## Review Checklist

- [ ] Every handler has entry and exit log statements
- [ ] Decision points (if/else/switch) are logged
- [ ] External service calls have before/after logs
- [ ] No string interpolation in log messages (use message templates)
- [ ] No sensitive data in logs (passwords, tokens, PII)
- [ ] Log levels are correct (Error for errors, Info for flow, Debug for detail)
- [ ] Property names are PascalCase and consistent across handlers
- [ ] Request objects are not logged in full (selective property logging)
