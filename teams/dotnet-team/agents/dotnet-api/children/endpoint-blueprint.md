---
knowledge-base-summary: "My production unit, end to end: decide the contract (route, verb, success status, auth, does it change the schema) → scaffold the request, validator, handler and endpoint → **register it on its endpoint group AND in the composition root** → verify it is reachable in the composed app → pitfalls → hand-off. The characteristic failure is a correct handler nothing routes to, and it passes every unit test; step 4 names the false green out loud, including the variant a fully compliant worker still hits — an in-process HTTP test that builds its own host instead of the app's."
---

# Endpoint blueprint

The thing this stack builds over and over is an **endpoint**: a request type, the validator that
rejects bad input at the boundary, the handler that holds the rule, and the route line that exposes
it. This page is the lifecycle for one, start to finish.

Two of its six steps exist for one reason, and it is worth stating before the code. The
characteristic failure of a scaffolded endpoint is **not** a bad handler — it is a **correct handler
that nothing routes to**. In this stack that failure has more than one shape, because more than one
thing here is wired by convention rather than by a reference the compiler can check: the route
mapping, the composition-root call that runs it, the mediator's handler discovery, and the
validator's. Every one of them is an *assembly scan or a call site*, not a type dependency. So the
build is green, the handler's own unit test is green, a reviewer reads correct code, and the endpoint
either does not exist at runtime or exists without its validation.

Step 3 is where those wirings are named. Step 4 is where you go and look, because "I added the
registration line" is itself a claim.

Baseline: ASP.NET Core Minimal APIs with route groups, a mediator, and FluentValidation in a pipeline
behaviour. A controller-based project maps the same six steps onto attribute routing plus
`AddControllers()` / `MapControllers()` — the *decisions* and the *false greens* are identical, only
the registration surface moves. The project's real layout, host topology and route prefixes live in
its own documentation; see
[architecture-and-layering.md](architecture-and-layering.md).

## 1. Decide before you write

Settle these first. Renaming later touches the request type, the handler, the route, the tests, the
generated client, and every caller that already integrated against the path.

| Question | How to answer it |
|---|---|
| **A new endpoint, or a new shape on an existing one?** | A new path when it is a different resource or a genuinely different action on one; a new field or query parameter when it refines the same action. A path is a public contract — cheaper to add than to retire. |
| **Which resource owns it, and therefore which endpoint group?** | The group that already owns that resource's paths. A *new* group is also a new call in the composition root and a new prefix in every caller's mental model. Take one only when the resource is genuinely new. |
| **Verb, path, success status.** | Settle all three before writing: the verb, the path shape (plural resource, id in the path, filters in the query), and the success code — `201` with a location for a create, `204` for a delete with no body, `200` otherwise. These three *are* the contract. |
| **Command or query?** | A command mutates and gets a validator; a query reads and often does not need one. This decides which folder the slice lands in and whether the unit touches the `SaveChanges` seam at all. |
| **What is the rule, and what are its failures?** | Name the handler and every exception it throws, with the status each maps to. If you cannot name a failure, this is probably a thin read whose tests belong at the handler, not at the boundary. Status mapping lives in one place — [error-contract.md](error-contract.md). |
| **Authenticated? Authorized by what?** | Decide before you map it: an anonymous endpoint, the group's default policy, or a named policy. Getting this wrong produces a green happy-path test and an open endpoint — see step 4. |
| **Does it change the schema?** | If yes it ships a migration, which changes both the verification in step 4 and the review risk. Decide now, not after the handler is written — see [persistence-and-migrations.md](persistence-and-migrations.md). |
| **Does it need a dependency the app does not already provide?** | A new injected service is a **third** wiring (step 3). It resolves per request, so a missing registration fails at request time, not at build time. |

## 2. Scaffold

Four files, in this order — the contract first, the rule next, the HTTP last. Layering per
[architecture-and-layering.md](architecture-and-layering.md).

```csharp
// Application/Features/Orders/Commands/ArchiveOrder/ArchiveOrderCommand.cs
// The request IS the contract. A record, not a class: value equality and no mutable state.
public sealed record ArchiveOrderCommand(Guid OrderId) : IRequest<ArchiveOrderResponse>;

// The response is a DTO, never the entity — see the pitfalls.
public sealed record ArchiveOrderResponse(Guid Id, string Status);
```

```csharp
// Application/Features/Orders/Commands/ArchiveOrder/ArchiveOrderValidator.cs
// Input shape only. Business rules that need the database live in the handler.
public sealed class ArchiveOrderValidator : AbstractValidator<ArchiveOrderCommand>
{
    public ArchiveOrderValidator()
    {
        RuleFor(x => x.OrderId).NotEmpty();
    }
}
```

```csharp
// Application/Features/Orders/Commands/ArchiveOrder/ArchiveOrderHandler.cs
// The rule lives here. No HttpContext, no IResult, no status codes.
public sealed class ArchiveOrderHandler(IApplicationDbContext db)
    : IRequestHandler<ArchiveOrderCommand, ArchiveOrderResponse>
{
    public async Task<ArchiveOrderResponse> Handle(ArchiveOrderCommand request, CancellationToken ct)
    {
        var order = await db.Orders.FirstOrDefaultAsync(o => o.Id == request.OrderId, ct)
            ?? throw new NotFoundException(nameof(Order), request.OrderId);

        if (order.Status == OrderStatus.Shipped)
            throw new ConflictException("A shipped order cannot be archived.");

        order.Status = OrderStatus.Archived;
        await db.SaveChangesAsync(ct);

        return new ArchiveOrderResponse(order.Id, order.Status.ToString());
    }
}
```

```csharp
// Api/Endpoints/OrderEndpoints.cs — HTTP wiring only. No rule to test lives in this file.
public static class OrderEndpoints
{
    public static IEndpointRouteBuilder MapOrderEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/v1/orders")
            .WithTags("Orders")
            .RequireAuthorization();

        group.MapPost("/{id:guid}/archive", ArchiveOrderAsync)
             .WithName("ArchiveOrder")
             .Produces<ArchiveOrderResponse>(StatusCodes.Status200OK);

        return app;
    }

    private static async Task<IResult> ArchiveOrderAsync(Guid id, ISender mediator, CancellationToken ct)
        => Results.Ok(await mediator.Send(new ArchiveOrderCommand(id), ct));
}
```

The endpoint method never hand-formats an error and never hand-writes an `if (request.X is null)`
check: the validator *is* the input contract and one exception handler owns every failure response.
The handler never returns an `IResult` — keeping status out of it is what lets the same rule be
called from a queue consumer or a scheduled job without an `HttpContext`.

Thread the `CancellationToken` through every `await`. It is free here and it is the only thing that
stops work continuing after the caller has hung up.

## 3. Register it — three wirings, not one

The endpoint is reachable, validated and resolvable only when **all** of these are true, and only the
first is in the file you just wrote.

```csharp
// 1. Api/Endpoints/OrderEndpoints.cs — the verb + path + policy, on the resource's group.
group.MapPost("/{id:guid}/archive", ArchiveOrderAsync);

// 2. Api/Program.cs — the composition root: the group's map call, in the pipeline's order.
var app = builder.Build();

app.UseExceptionHandler();     // early — everything after it is covered
app.UseAuthentication();
app.UseAuthorization();

app.MapOrderEndpoints();       // ← a NEW group needs this line; an existing one already has it

app.Run();

public partial class Program;  // makes the composed app reachable from a test host — see step 4
```

```csharp
// 3. Application/DependencyInjection.cs — discovery is by ASSEMBLY SCAN, not by reference.
services.AddMediatR(c => c.RegisterServicesFromAssembly(typeof(DependencyInjection).Assembly));
services.AddValidatorsFromAssembly(typeof(DependencyInjection).Assembly);
services.AddTransient(typeof(IPipelineBehavior<,>), typeof(ValidationBehavior<,>));
```

**Read the sibling registrations before writing yours, and match them.** If the group already applies
the auth policy at `MapGroup`, do not re-declare it per route; if every sibling declares its policy
inline, declare yours inline too. A registration that looks unlike its neighbours is the one a
reviewer skims past — and the one whose *ordering* difference nobody notices.

**Order is part of the registration, not a detail.** The pipeline runs in the order you write it: an
exception handler registered after the endpoints never sees their throws, and authentication
registered after authorization has produced no principal for it to judge. Position is behaviour here.

**Three things about the scan, because it is where this stack differs from most.**

- The handler and the validator are found by scanning an *assembly*, so a slice placed in a project
  the registration does not name compiles perfectly and is simply never discovered. The handler's
  absence surfaces at request time (the mediator cannot resolve one). The validator's absence
  surfaces **not at all** — the request is accepted unvalidated.
- The validation behaviour is a **separate** registration from the validators. With the validators
  registered and the behaviour missing, every validator in the solution is inert.
- `AddValidatorsFromAssembly` is not in the base FluentValidation package — it ships in the DI
  extensions package. A project missing that reference has no scan at all.

**A new dependency is the third wiring.** If the handler took a constructor parameter the app does
not already register, it resolves *per request* and fails there, not at build time. The default host
turns DI validation on in the Development environment and off elsewhere, so a lifetime mistake — a
singleton consuming a scoped service — surfaces as a startup failure locally and as silent
misbehaviour where validation is off. Turn it on explicitly if you want that guarantee everywhere.

## 4. Verify the durable effect

**Take the good news first, because it is real: an in-process HTTP test against the composed app
catches every classic omission here.** A test host built from the real `Program` issues a real
request through the real pipeline, so a never-mapped route comes back `404` and the test fails; the
boundary-rejection test extends that to validation and to the exception handler's position. If you
wrote those, the ordinary mistakes are caught. Say that plainly instead of performing doubt.

**The false green this stack does not prevent is one line further up: a test that builds its own
host.**

```csharp
// ✅ the app the process actually serves
var factory = new WebApplicationFactory<Program>();
var client  = factory.CreateClient();

// ❌ *a* composed app, not *the* one — and it can mirror the prefix, so the request URL is identical
var builder = WebApplication.CreateBuilder();
builder.Services.AddApplication();
var app = builder.Build();
app.MapOrderEndpoints();          // ← mapped HERE, by the test
await app.StartAsync();
```

The second one is a fictional guarantee and **every gate goes green over it**: it compiles, it starts,
it issues a genuine HTTP request over a genuine pipeline, it exercises routing and the response
shape — against an application that exists only inside the test file, while the real composition root
never calls `MapOrderEndpoints()`.

This is the variant worth the paragraph because **a fully compliant worker reaches it.** Nothing in
the instruction "write an in-process HTTP test that issues a real request through the pipeline" rules
it out; it satisfies that sentence to the letter. And this stack actively pushes you toward it: with
top-level statements the generated `Program` class is internal, so `WebApplicationFactory<Program>`
does not compile until someone adds the `public partial class Program;` line or an
`InternalsVisibleTo`. Hitting that friction and reaching for `WebApplication.CreateBuilder()` instead
is a two-minute decision that silently voids the whole guarantee. The difference is one line at the
top of the test file — **check it by eye.**

Then observe the endpoint outside the suite. Two surfaces, both cheap:

```csharp
// The composed app's route table — this stack's equivalent of a CLI's --help.
// Every mapped route, printed next to its siblings, from the SAME service provider the app uses.
var source = factory.Services.GetRequiredService<EndpointDataSource>();
foreach (var e in source.Endpoints.OfType<RouteEndpoint>())
{
    var verbs = e.Metadata.GetMetadata<HttpMethodMetadata>()?.HttpMethods ?? new[] { "*" };
    Console.WriteLine($"{string.Join('|', verbs)} {e.RoutePattern.RawText}");
}
```

```bash
# And once through a real socket — success, rejection, and no credentials.
curl -s -i -X POST localhost:<port>/<prefix>/<path>          -H 'authorization: Bearer <token>'
curl -s -i -X POST localhost:<port>/<prefix>/<bad-input>     -H 'authorization: Bearer <token>'
curl -s -i -X POST localhost:<port>/<prefix>/<path>          # ← no token: prove the 401 you designed
```

The route table earns its place: a test asserts the path *the developer believed in*, while the table
prints every mounted route next to its neighbours — so a doubled prefix
(`POST /v1/orders/v1/orders/{id}`), or a route sitting under `/orders` while every sibling is under
`/v1/orders`, is visible at a glance and invisible to a test written from the same wrong assumption.
The port, prefix and run command are project truths; take them from the durable-knowledge pages the
brief names, not from this page.

Three more durable effects to observe when they apply:

- **Validation actually running.** The happy-path call passes whether or not the validator was
  discovered. Send input the validator is designed to reject and confirm you get the rejection status
  *with the field key* — that single call is the only thing that distinguishes "validated" from
  "the scan never found it".
- **Authorization, where the green can be *caused by* the omission.** A `200` in a happy-path test
  says nothing about whether the endpoint is protected, and a test host that swaps in a permissive
  authentication scheme makes every request authenticated by construction — so a policy you forgot to
  declare returns exactly what one you declared returns. Call it once with no credentials, against a
  host that has not been made permissive. (ASP.NET Core does refuse at request time when an endpoint
  carries authorization metadata and no authorization middleware ran — a genuine safety net, and one
  that only fires if something actually calls the endpoint.)
- **A migration.** An integration run applies migrations to a *fresh, empty* database, which proves
  the SQL is valid — not that it applies to a populated one. For anything destructive, confirm the
  sequencing in [persistence-and-migrations.md](persistence-and-migrations.md) and say so in the
  hand-off.

**The unit is not done until it ships a test covering the behaviour it added**, and at least one that
goes **red when the change is reverted**. That second half is not ceremony: coverage proves a line
*ran*, never that anything *checked* the result — a test that exercises the endpoint and asserts
nothing scores full marks. Confirming it costs a minute. The numeric gate is the project's; the shape
is this stack's, and the details are in
[testing-and-verification.md](testing-and-verification.md).

## 5. Common pitfalls

- **The group is new and the composition root never calls it.** The commonest form: the route line
  looks complete because the file it sits in is complete. Grep the composition root for the map call.
- **A doubled prefix.** `MapPost("/v1/orders/{id}")` inside a group already mapped at `/v1/orders`.
  The route table shows it immediately; a test written from the same assumption never will.
- **The slice is in an assembly nothing scans.** The handler fails at request time; the validator
  fails *silently* and the endpoint accepts input it was written to reject. Two different failures
  from one missing registration.
- **The validation behaviour is not registered.** Validators exist, are discovered, and never run.
  Nothing anywhere reports this.
- **A new injected service is not registered.** It resolves per request, so this is a runtime failure
  in a build that is green — and in a deployed environment it can be the first request of the day
  that finds it.
- **Middleware in the wrong order.** An exception handler after the endpoints, or authorization
  before authentication. The pipeline is ordered by the lines you wrote, not by intent.
- **A model-binding failure escaping as a 500.** A malformed enum, id or date in the route or query
  string throws before the handler is ever reached, and reads to the caller as a server fault unless
  one central arm maps it — [error-contract.md](error-contract.md).
- **Returning the entity instead of a DTO.** It publishes every field the schema happens to have,
  welds the wire shape to the table, and — once a navigation property is populated — can throw on
  serialization for a reference cycle. Return the record you designed.
- **Business validation pushed into the validator.** A rule that needs a database read is the
  handler's; a validator that queries turns the boundary check into a second data-access path and
  leaves the rule untestable where it actually lives.

## 6. Hand-off

The work-item is ready for the tester when: the solution builds and the full test command is green;
the integration test constructs the app from the **real composition root** and covers the success
shape plus the boundary rejection; that output is attached; the endpoint has been observed on the
composed app (route table and/or a live call); authorization has been *exercised* rather than
assumed; validation has been proven to run by a rejected request; and any migration's sequencing is
stated.

State plainly which of these you **observed** and which you **assumed** — the test's host
construction most of all, since it is the difference between a proven contract and a fictional one.
An unverified claim is worse than an admitted gap: the review gate is what authorizes the merge, and
it can only check what it is told about.
