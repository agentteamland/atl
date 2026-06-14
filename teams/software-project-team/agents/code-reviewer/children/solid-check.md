---
knowledge-base-summary: "Detect SOLID principle violations in code changes. Single responsibility (class doing too much), Open/closed (switch statements that should be polymorphism), Liskov (base class contract violations), Interface segregation (fat interfaces), Dependency inversion (concrete dependencies)."
---
# SOLID Principle Check

## Single Responsibility Principle (SRP)

A class should have only one reason to change.

### What to Look For

- **Handler doing too much:** A handler that validates, fetches data, applies business logic, sends notifications, and publishes events is doing too many things. Validation belongs in the validator, notifications belong in a service, events are published via the pipeline.
- **Fat service classes:** A service with 10+ methods or 5+ injected dependencies is a red flag. It should be split by responsibility.
- **Entity with behavior it doesn't own:** An entity that sends emails or calls external services is violating SRP. Entities hold state and invariants, nothing more.
- **Endpoint with logic:** An endpoint that does anything beyond parse-request, call-mediator, return-response has too much responsibility.

### Report Template

```
**Problem:** {ClassName} has {n} responsibilities: {list them}
**Why:** When {responsibility-A} changes, {ClassName} must change too, even though {responsibility-B} is unrelated.
**Fix:** Extract {responsibility} into {NewClassName}.
```

## Open/Closed Principle (OCP)

Software entities should be open for extension, closed for modification.

### What to Look For

- **Switch/if-else chains on type:** A switch statement that checks entity type and runs different logic for each is a sign that polymorphism should be used instead.
- **Modifying existing code to add a new variant:** If adding a new payment method requires modifying the existing handler with another `if` branch, the design is not open for extension.
- **Magic strings controlling behavior:** If behavior changes based on string comparisons (`if (type == "premium")`), this should be an enum or a strategy pattern.

### Report Template

```
**Problem:** Adding a new {concept} requires modifying {ClassName} at line {n}.
**Why:** Every new {concept} increases the risk of breaking existing {concepts}.
**Fix:** Use {pattern: strategy/polymorphism/factory} to make {ClassName} extensible without modification.
```

## Liskov Substitution Principle (LSP)

Objects of a superclass should be replaceable with objects of a subclass without breaking the program.

### What to Look For

- **Derived class throwing NotImplementedException:** If a subclass overrides a method and throws "not supported", it cannot replace the base class.
- **Derived class ignoring base class behavior:** If a subclass overrides a method and does nothing (empty override), the contract is violated.
- **Precondition strengthening:** A subclass that adds extra validation the base class does not require is violating LSP.
- **Postcondition weakening:** A subclass that returns less or does less than the base class promises.

### Report Template

```
**Problem:** {DerivedClass} violates the contract of {BaseClass} by {violation}.
**Why:** Code that depends on {BaseClass} will break when given {DerivedClass}.
**Fix:** {Specific fix: remove inheritance, adjust contract, or fix the override}.
```

## Interface Segregation Principle (ISP)

Clients should not be forced to depend on interfaces they do not use.

### What to Look For

- **Fat interfaces:** An interface with 10+ methods where most implementations only use 3-4. Split into focused interfaces.
- **Implementations with throw NotImplementedException:** If an implementation must throw because it doesn't support half the interface, the interface is too broad.
- **Marker methods:** Methods that exist in an interface only because "some" implementations need them, while others return null or do nothing.

### Report Template

```
**Problem:** {InterfaceName} has {n} methods but {ImplementationName} only uses {m}.
**Why:** {ImplementationName} is forced to implement methods it doesn't need, creating dead code and confusion.
**Fix:** Split {InterfaceName} into {Interface-A} ({methods}) and {Interface-B} ({methods}).
```

## Dependency Inversion Principle (DIP)

High-level modules should not depend on low-level modules. Both should depend on abstractions.

### What to Look For

- **Handler using concrete class:** A handler that instantiates `new SmtpEmailSender()` instead of injecting `IEmailSender`.
- **Application layer importing Infrastructure:** If a class in the Application layer has a `using` statement pointing to Infrastructure, the dependency direction is wrong.
- **Direct database access in handlers:** A handler that creates `new SqlConnection()` instead of going through an abstraction.
- **Concrete configuration in business logic:** A handler reading `Environment.GetEnvironmentVariable()` directly instead of using an injected options class.

### Report Template

```
**Problem:** {ClassName} depends on concrete {ConcreteDependency} instead of {Abstraction}.
**Why:** {ClassName} cannot be tested in isolation and is coupled to the implementation detail.
**Fix:** Define {InterfaceName} in Application, implement in Infrastructure, inject via constructor.
```
