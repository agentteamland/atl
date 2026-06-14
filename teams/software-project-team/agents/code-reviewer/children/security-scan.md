---
knowledge-base-summary: "Check for SQL injection, XSS, auth bypass, data leaks, secrets in code, CORS misconfiguration, and missing rate limiting. Every security finding is critical severity. When in doubt, flag it."
---
# Security Scan

Every security finding is **Critical** severity by default. When in doubt, flag it -- a false positive is better than a missed vulnerability.

## SQL Injection

### What to Look For

- **Raw SQL with string concatenation:**
  ```csharp
  // CRITICAL: SQL injection
  var sql = $"SELECT * FROM Users WHERE Name = '{name}'";
  _db.Database.ExecuteSqlRaw(sql);
  ```
- **String interpolation in SQL:**
  ```csharp
  // CRITICAL: SQL injection via interpolation
  _db.Database.ExecuteSqlRaw($"DELETE FROM Orders WHERE Id = {id}");
  ```
- **FromSqlRaw with user input:**
  ```csharp
  // CRITICAL: User input directly in SQL
  _db.Users.FromSqlRaw($"SELECT * FROM Users WHERE Email = '{request.Email}'");
  ```

### Safe Alternatives

```csharp
// Parameterized query
_db.Database.ExecuteSqlInterpolated($"DELETE FROM Orders WHERE Id = {id}");
// EF Core LINQ (always safe)
_db.Users.Where(u => u.Email == request.Email);
// FromSqlInterpolated (parameterized)
_db.Users.FromSqlInterpolated($"SELECT * FROM Users WHERE Email = {request.Email}");
```

## XSS (Cross-Site Scripting)

### What to Look For

- **Unsanitized user input in responses:** Returning raw HTML or script content from user input without encoding.
- **Rich text fields without sanitization:** If users can submit HTML (e.g., bio, description), it must be sanitized before storage and before rendering.
- **User input in error messages:** Error messages that echo back user input verbatim.

### Check Points

- Any field that accepts free-text from users
- Any response that includes user-generated content
- Any template that renders dynamic content

## Auth Bypass

### What to Look For

- **Missing RequireAuthorization:** An endpoint that should be protected but lacks `.RequireAuthorization()`.
- **Missing role/policy check:** An endpoint with auth but no role restriction when it should have one.
- **Broken object-level authorization:** A handler that fetches data by ID without checking if the current user owns/can access it.
  ```csharp
  // CRITICAL: No ownership check -- any authenticated user can access any order
  var order = await _db.Orders.FindAsync(request.OrderId);
  ```
- **Internal endpoints without InternalSecretFilter:** System-to-system endpoints accessible without `X-Internal-Token` validation.
- **JWT validation bypass:** Custom token validation that skips expiry check or issuer validation.

### Critical Pattern

Every endpoint that returns or modifies data for a specific user MUST verify ownership:

```csharp
// Correct: ownership check
var order = await _db.Orders
    .Where(o => o.Id == request.OrderId && o.UserId == currentUserId)
    .FirstOrDefaultAsync(ct)
    ?? throw new NotFoundException(nameof(Order), request.OrderId);
```

## Data Leak

### What to Look For

- **Returning entity instead of DTO:**
  ```csharp
  // CRITICAL: Entity returned directly -- exposes all fields including internal ones
  return Results.Ok(user);
  // CORRECT:
  return Results.Ok(new UserResponse(user.Id, user.Name, user.Email));
  ```
- **Exposing internal IDs:** Sequential integer IDs in responses that allow enumeration.
- **Verbose error messages:** Error responses that include stack traces, SQL queries, or internal paths in production.
- **Over-fetching:** Querying `SELECT *` when only a few fields are needed, then returning the full object.
- **Sensitive fields in response:** Password hashes, security tokens, internal flags in API responses.
- **Logging PII:** Passwords, tokens, credit card numbers in log messages.

## Secrets in Code

### What to Look For

- Hardcoded connection strings
- API keys or tokens in source code
- JWT signing keys in code (should be in configuration/secrets)
- Password or secret in variable assignment
- Base64-encoded secrets (they are NOT encrypted)
- `.env` files committed to git

### Where to Check

- `appsettings.json` (should use placeholders, not real values)
- `docker-compose.yml` (environment variables with real secrets)
- Any string that looks like a key, token, or password

## CORS Misconfiguration

### What to Look For

- **AllowAnyOrigin with credentials:**
  ```csharp
  // CRITICAL: Allows any website to make authenticated requests
  builder.WithOrigins("*").AllowCredentials();
  ```
- **Wildcard origin in production:** `AllowAnyOrigin()` should never be used in production.
- **Missing CORS on sensitive endpoints:** API endpoints that should restrict which domains can call them.

## Rate Limiting

### What to Look For

- **Missing rate limiting on auth endpoints:** Login, register, password reset, and verification endpoints MUST have rate limiting.
- **Missing rate limiting on sensitive operations:** Create, update, delete operations on important resources.
- **Missing rate limiting on search/listing:** Endpoints that can be used for data scraping.

### Critical Endpoints (Must Have Rate Limiting)

| Endpoint | Why |
|----------|-----|
| Login | Brute force prevention |
| Register | Spam account prevention |
| Password reset | Email bombing prevention |
| Verification code | Code guessing prevention |
| File upload | Resource exhaustion prevention |

## Review Checklist

- [ ] No raw SQL with string concatenation or interpolation
- [ ] No unsanitized user input in responses
- [ ] All endpoints that need auth have RequireAuthorization
- [ ] All handlers verify object-level authorization (ownership)
- [ ] No entities returned directly (always DTO/Response)
- [ ] No secrets hardcoded in code
- [ ] CORS configured correctly (no wildcard with credentials)
- [ ] Rate limiting on auth and sensitive endpoints
- [ ] Internal endpoints protected with InternalSecretFilter
- [ ] No PII in log messages
