---
knowledge-base-summary: "Verify implementation matches documented architecture. Check for layer violations, dependency direction, feature organization, thin host compliance, and code in the wrong layer. The architecture document is the source of truth."
---
# Architecture Review

## Pre-Review Step

Read CLAUDE.md and any architecture documentation to understand the intended design. The review measures actual implementation against documented intent.

## Implementation vs Documentation

### What to Check

1. **Read the documented architecture** from CLAUDE.md (Project Structure section)
2. **Scan the actual directory structure** and compare
3. **Flag any discrepancy** -- missing folders, extra folders, renamed folders, different organization

### Common Findings

- CLAUDE.md says the project has a `Worker` service, but the directory doesn't exist (or is empty)
- CLAUDE.md describes Vertical Slice organization, but some features use a different structure
- CLAUDE.md lists a technology (e.g., Redis), but there's no code using it
- A new service exists in the code but is not documented in CLAUDE.md

## Layer Violations

### What to Check

Verify that imports (using statements) respect layer boundaries:

```
Domain       -> No external dependencies (pure C#)
Application  -> Domain only (+ framework packages like Mediator, FluentValidation)
Infrastructure -> Application, Domain (implements Application interfaces)
Api/Socket/Worker -> Application (composition root, wires everything)
```

### Violation Detection

Scan for these patterns:

| Violation | Pattern to Search |
|-----------|------------------|
| Application importing Infrastructure | `using {Project}.Infrastructure` in Application layer |
| Domain importing Application | `using {Project}.Application` in Domain layer |
| Domain importing Infrastructure | `using {Project}.Infrastructure` in Domain layer |
| Api containing business logic | Classes in Api that do more than endpoint routing |
| Host importing another host | Socket importing Worker, Worker importing Api, etc. |

### How to Scan

```bash
# Check for Application -> Infrastructure violation
grep -r "using.*\.Infrastructure" src/{Project}.Application/

# Check for Domain -> Application violation
grep -r "using.*\.Application" src/{Project}.Domain/

# Check for Domain -> Infrastructure violation
grep -r "using.*\.Infrastructure" src/{Project}.Domain/
```

## Dependency Direction

### The Rule

Dependencies always flow inward:

```
Api -> Application -> Domain
       Application <- Infrastructure
```

Infrastructure implements Application interfaces. Application never knows about Infrastructure's concrete types.

### What to Check

- Application layer defines interfaces (`IEmailSender`, `IStorageService`)
- Infrastructure layer implements them (`SmtpEmailSender`, `S3StorageService`)
- No concrete Infrastructure type is referenced in Application
- DI registration happens in the composition root (Api/Program.cs), not in Application

## Feature Organization

### What to Check

If the project uses Vertical Slice architecture:

```
Features/
  Orders/
    Commands/
      CreateOrder/
        CreateOrderCommand.cs
        CreateOrderHandler.cs
        CreateOrderValidator.cs
    Queries/
      GetOrderById/
        GetOrderByIdQuery.cs
        GetOrderByIdHandler.cs
    Common/
      OrderResponse.cs
```

- Each feature is self-contained in its own folder
- Commands and Queries are separated
- Handler, Command, and Validator live together
- Shared types (DTOs, responses) are in `Common/`

### Red Flags

- Feature files scattered across multiple directories
- All handlers in one file or folder
- Shared DTOs mixed with feature-specific code
- Inconsistent organization (some features follow the pattern, others don't)

## Thin Host Principle

### The Rule

Host applications (Api, Socket, Worker, LogIngest, MailSender) should be thin:
- Api: Parse HTTP request -> call Mediator -> return response
- Socket: Receive SignalR message -> call API via HTTP -> push response
- Worker: Timer fires -> call API via HTTP -> log result
- Consumers: Receive message -> process -> acknowledge

### What to Check

- No business logic in endpoint methods (Api)
- No database access in Socket hub methods
- No business rules in Worker job execution
- No domain logic in consumer message handlers

### Red Flags

- Endpoint method with more than 10 lines of code
- Hub method that queries the database directly
- Worker job that calculates business rules instead of calling the API
- Consumer that does complex data transformation

## Code in Wrong Layer

### Common Misplacements

| Code | Wrong Layer | Correct Layer |
|------|------------|---------------|
| Validation rules | Handler or Domain | Application (Validator) |
| Business logic | Endpoint or Worker | Application (Handler) |
| Database query | Endpoint | Infrastructure or Handler (via DbContext) |
| Email sending | Handler (concrete) | Infrastructure (via interface) |
| JWT generation | Application | Infrastructure |
| Entity definition | Application | Domain |
| Interface definition | Infrastructure | Application |
| Exception definition | Api | Application |

### How to Detect

- Scan for database-related imports in Api layer
- Scan for HTTP/JWT imports in Domain layer
- Check if handlers instantiate concrete infrastructure classes
- Verify entities have no dependencies on external packages

## Review Checklist

- [ ] Directory structure matches documented architecture
- [ ] No layer violations (using statements respect boundaries)
- [ ] Dependency direction is always inward
- [ ] Features organized consistently (Vertical Slice if documented)
- [ ] Host applications are thin (no business logic)
- [ ] No code in the wrong layer
- [ ] New services/features documented in CLAUDE.md
- [ ] All documented technologies are actually used
