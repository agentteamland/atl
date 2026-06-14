---
knowledge-base-summary: "Check names against project conventions. Class, method, variable, file, and namespace naming. Consistency verification (same concept = same name everywhere). Abbreviation rules and magic string/number detection."
---
# Naming Review

## Pre-Review Step

Before checking names, read the project's naming conventions from:
- `.atl/docs/coding-standards/{app}.md` (project-specific)
- The relevant agent's `children/naming-conventions.md` (if it exists)

Project conventions always take precedence over general rules.

## Class Naming

### What to Check

| Pattern | Convention | Example |
|---------|-----------|---------|
| Command | `{Action}Command` | `CreateOrderCommand` |
| Query | `{Query}Query` | `GetOrderByIdQuery` |
| Handler | `{Action}Handler` | `CreateOrderHandler` |
| Validator | `{Action}Validator` | `CreateOrderValidator` |
| Entity | Singular noun | `Order`, `User`, `Product` |
| DTO/Response | `{Entity}{Action}Response` | `OrderCreateResponse` |
| Interface | `I{Noun}` | `IOrderRepository`, `IEmailSender` |
| Configuration | `{Entity}Configuration` | `OrderConfiguration` |
| Exception | `{Noun}Exception` | `NotFoundException`, `ForbiddenException` |
| Behavior | `{Noun}Behavior` | `ValidationBehavior`, `LoggingBehavior` |
| Endpoint group | `{Feature}Endpoints` | `OrderEndpoints` |

### Red Flags

- Class name does not describe what it does: `Manager`, `Helper`, `Utils`, `Processor` (too generic)
- Class name includes implementation detail: `SqlOrderRepository` in Application layer
- Plural entity name: `Orders` instead of `Order`
- Verb as class name (unless it's a command): `Validate` instead of `OrderValidator`

## Method Naming

### What to Check

| Context | Convention | Example |
|---------|-----------|---------|
| Endpoint handler | `{Action}Async` | `CreateOrderAsync` |
| Mediator handle | `Handle` | (framework-defined) |
| Query method | `Get{What}`, `Find{What}` | `GetOrderById`, `FindActiveUsers` |
| Boolean method | `Is{Condition}`, `Has{Property}`, `Can{Action}` | `IsExpired()`, `HasPermission()` |
| Factory method | `Create{What}` | `CreateFromTemplate()` |
| Conversion | `To{Target}` | `ToResponse()`, `ToDto()` |

### Red Flags

- Missing `Async` suffix on async methods
- Method name doesn't describe what it does: `Process()`, `Handle()` (outside of framework methods)
- Boolean method without `Is/Has/Can` prefix
- Inconsistent naming: `GetUser` in one place, `FetchUser` in another

## Variable Naming

### What to Check

- Local variables: camelCase (`orderItem`, `userId`)
- Constants: PascalCase for public, camelCase for private (`MaxRetryCount`, `_maxRetryCount`)
- Private fields: `_camelCase` with underscore prefix (`_orderRepository`, `_logger`)
- Parameters: camelCase matching the domain term (`orderId`, not `id` or `oId`)

### Red Flags

- Single-letter variables (except in LINQ lambdas or loop counters)
- Abbreviated names: `usr` instead of `user`, `ord` instead of `order`
- Hungarian notation: `strName`, `intCount`
- Inconsistent casing within the same file
- Variable name does not match what it holds: `var data = GetOrders();` -- should be `var orders`

## File Naming

### What to Check

| Type | Convention | Example |
|------|-----------|---------|
| Command | `{Action}Command.cs` | `CreateOrderCommand.cs` |
| Handler | `{Action}Handler.cs` | `CreateOrderHandler.cs` |
| Validator | `{Action}Validator.cs` | `CreateOrderValidator.cs` |
| Entity | `{Entity}.cs` | `Order.cs` |
| Configuration | `{Entity}Configuration.cs` | `OrderConfiguration.cs` |
| Interface | `I{Name}.cs` | `IOrderRepository.cs` |

### Red Flags

- File name doesn't match class name
- Multiple public classes in one file (exceptions: related records/enums)
- File in wrong directory for its namespace

## Namespace Naming

### What to Check

- Namespace matches folder structure exactly
- Pattern: `{Project}.{Layer}.{Feature}.{SubFeature}`
- Example: `ExampleApp.Application.Features.Users.Commands.CreateUser`

### Red Flags

- Namespace doesn't match directory path
- Flat namespace (everything in one namespace)
- Namespace importing from wrong layer direction

## Consistency Check

The most important naming rule: **the same concept must use the same name everywhere**.

### What to Check

- If it's called `userId` in one place, it must be `userId` everywhere (not `uid`, `user_id`, `UserId` in different contexts)
- If the entity is `Order`, the table is `Orders`, the endpoint is `/orders`, the feature folder is `Orders`
- If one handler uses `cancellationToken`, all handlers use `cancellationToken` (not `ct` in some, `token` in others)

### Magic String/Number Detection

- **Magic strings:** Hardcoded strings that should be constants or enums
  - `if (status == "active")` -- should be an enum
  - `"order-confirmation"` template name -- should be a constant
- **Magic numbers:** Hardcoded numbers without context
  - `if (retryCount > 3)` -- what is 3? Should be a named constant
  - `Take(100)` -- why 100? Should come from configuration or a constant

### Report Template

```
**Problem:** Inconsistent naming for {concept}: `{name-A}` in {file-A}, `{name-B}` in {file-B}.
**Fix:** Standardize to `{correct-name}` everywhere.
```
