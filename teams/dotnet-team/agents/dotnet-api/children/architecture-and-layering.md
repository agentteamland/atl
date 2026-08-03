---
knowledge-base-summary: "Where a change goes: inward-pointing Domain / Application / Infrastructure / Api layers, feature-slice organization, and the rule that an endpoint is a bridge while the handler is the one place logic lives. Answered before any of that: which host can even hold the change — only the composed host owns the DbContext and dispatch, so 'just move it to the worker' means moving the whole persistence stack. Plus composition-root craft, including options binding where the section name IS the environment-variable prefix, so a prefix or property drift binds nothing and still starts cleanly."
---

# Architecture and layering

Where a change goes, decided before a line of it is written. Three questions in order: **which host**
can hold it at all, **which layer** owns it, and **which slice** inside that layer. Getting the first
one wrong is the expensive mistake — it is not a refactor, it is a re-platforming.

## 1. Which host — the reference graph is the boundary

A .NET solution of any size is usually several hosts, not one: an HTTP API, plus some subset of a
scheduler, a queue consumer, a realtime hub, a log or mail relay. They are not peers. **One of them
composes the domain stack and the rest are deliberately thin.**

The composed host is the one whose project references reach Infrastructure — so it is the only one
that has a `DbContext`, a cache client, a storage client, and the dispatch/mediator pipeline
registered. A thin host references almost nothing and reaches the domain **over HTTP**, through the
composed host's internal endpoints.

**Read the `.csproj` reference graph before deciding where a change lands.** It is a two-minute read
and it answers the question mechanically:

```bash
grep -rn "ProjectReference" --include=*.csproj .
grep -rn "AddInfrastructure\|AddPersistence\|AddDbContext" --include=*.cs .
```

The registration call that wires persistence usually has **exactly one caller**, and that single line
is the whole boundary. Its consequences, in the order they bite:

- **"Just move it to the worker" is not a small change.** Moving persistence-dependent work into a
  thin host means moving the persistence and dispatch stack with it — and then that host is no longer
  thin, and there are two composition roots to keep in agreement.
- **A queue consumer that needs the DbContext runs inside the composed host**, registered with
  `AddHostedService` in the same DI extension that wires persistence. This looks wrong (a consumer in
  the API process) and is the right answer: the alternative is duplicating the stack.
- **A thin host holds no domain state.** It schedules, it relays, it pushes. If it starts wanting an
  entity type, that is the signal it is being asked to do the composed host's job.

**Practical rule:** if the change needs the DbContext, it lands in the composed host's layers. If it
needs to be *scheduled*, a thin host triggers it over the composed host's internal HTTP surface, and
the logic stays where the data is.

## 2. Which layer — dependencies point inward

Four layers, each depending only on the ones inside it.

| Layer | Depends on | Holds |
|---|---|---|
| **Domain** | nothing | entities, enums, value objects, domain exceptions, domain events |
| **Application** | Domain | commands, queries, handlers, validators, **interfaces** for everything outside |
| **Infrastructure** | Application, Domain | EF Core + the DbContext, auth, broker, cache, storage — the **implementations** |
| **Api** (host) | all three | the composition root: DI, the request pipeline, endpoint mapping |

The load-bearing half is the **interface in Application, implementation in Infrastructure** split. A
handler that wants to send an email, publish a message, read a cache or write a blob depends on an
interface it can see; the concrete client lives one layer out and is registered in the composition
root. That is what keeps a handler unit-testable without a broker, and what keeps a swap of the
concrete client from touching a single handler.

```csharp
// Application/Common/Interfaces/IEmailSender.cs — the handler's whole view of email
public interface IEmailSender
{
    Task SendAsync(string template, object data, string to, CancellationToken ct);
}

// Infrastructure/Messaging/RmqEmailSender.cs — the concrete client, never referenced from a handler
internal sealed class RmqEmailSender : IEmailSender { /* ... */ }
```

**Two pragmatic exceptions worth knowing before a reviewer flags them:**

- **Application usually references EF Core's abstractions.** Expressing the DB-facing interface as
  `IApplicationDbContext { DbSet<T> Xs { get; } Task<int> SaveChangesAsync(CancellationToken ct); }`
  means Application imports `Microsoft.EntityFrameworkCore` to get `DbSet<>`. That is a deliberate,
  common trade — the purist alternative hides everything behind `IQueryable<T>` and gives up
  `Entry()`, `SaveChangesAsync` on the interface, and most of EF's ergonomics. **Follow whichever
  shape the project already has**; do not convert one to the other as a side effect of a feature.
- **A narrow interface hides its own members.** `IApplicationDbContext` typically exposes the sets,
  `Database`, and `SaveChangesAsync` — and *nothing else*. So `ChangeTracker` is not reachable through
  it and a cast is required to touch it (`(_db as DbContext)?.ChangeTracker.Clear()`). That cast is
  structural, not stylistic; see the persistence topic for when it is mandatory.

## 3. Which slice — feature slices, not technical folders

Inside Application, organize by **feature**, not by artifact type. A `Handlers/` folder holding every
handler in the system is a directory of unrelated files; a feature folder is a unit you can read, move,
or delete whole.

```
Application/Features/<Feature>/
├── Commands/<Action>/
│   ├── <Action>Command.cs      // record : IRequest<<Action>Response>
│   ├── <Action>Handler.cs      // IRequestHandler<Command, Response>
│   └── <Action>Validator.cs    // AbstractValidator<Command>
├── Queries/<Query>/
│   ├── <Query>Query.cs
│   └── <Query>Handler.cs
└── Common/<Feature>Dto.cs      // shared shapes, when two slices genuinely share one
```

The command, its handler and its validator sit **together**, because they change together. A response
record either nests in the command file or lives under the slice's `Common/`. This is a convention, not
a framework rule — its value is that every slice looks the same, so a worker who has read one can
navigate any of them.

**Match the project's existing slice names and depth.** Slice vocabulary is project-shaped: the
durable-knowledge pages the brief names hold the real list. Adding a differently-shaped slice next to
nine consistent ones is a cost paid by every later reader.

## 4. An endpoint is a bridge; the handler is the one logic point

The single most useful placement rule in this stack, and the one most often broken under time pressure.

```csharp
// Api/Endpoints/<Feature>Endpoints.cs — translation only
group.MapPost("/", CreateAsync)
     .RequireAuthorization(<policy>)
     .WithName("CreateReport");

static async Task<IResult> CreateAsync(
    CreateReportRequest request,
    IMediator mediator,
    CancellationToken ct)
{
    var response = await mediator.Send(new CreateReportCommand(request.Title, request.Range), ct);
    return TypedResults.Ok(response);
}
```

The endpoint method does exactly four things: bind, map the wire shape to the command, dispatch, and
shape the result. **Everything else is the handler's.** A rule that lives in an endpoint is a rule that:

- cannot be reached by any other caller — a scheduled job, a queue consumer, a second endpoint that
  needs the same behaviour later;
- is not covered by validator or handler tests, because those never run the HTTP layer;
- has to be re-derived (and re-diverged) the first time a second write path appears.

Input validation is the validator's, not the endpoint's and not the handler's: a hand-written
`if (request.X is null) return BadRequest()` at the boundary duplicates a guard the pipeline already
owns and leaves the failure shape inconsistent with every other endpoint.

The mirror rule: **the handler does not know about HTTP.** No `IResult`, no status codes, no
`HttpContext`. It returns a response or throws a typed exception the error-mapping seam knows how to
turn into a status. That is what lets the same handler serve an endpoint today and a consumer tomorrow.

## 5. The composition root

`Program.cs` is the only place that knows every concrete type. Three things live here and nowhere else:
DI registration, the request pipeline order, and options binding.

### DI registration and lifetimes

Register per-layer through one extension method per layer (`AddApplication()`, `AddInfrastructure()`),
so the host reads as a list of layers rather than a hundred lines of `AddScoped`.

**Lifetime mismatches are the recurring DI defect, and the default builder catches most of them at
startup** — a singleton that captures a scoped dependency fails validation in Development rather than
silently holding one request's context forever. Two consequences:

- **A `BackgroundService` is a singleton.** It cannot constructor-inject anything scoped — the
  DbContext included. It injects `IServiceScopeFactory` and creates a scope per unit of work. This is
  not optional and not a style choice; the alternative is a context shared across the process lifetime.
- **Cross-cutting behaviour that needs the current request** (a current-user accessor, a tenant
  resolver) is **scoped**, and anything consuming it must be scoped too. Where such behaviour is
  implemented inside the DbContext itself, the lifetimes align for free — both scoped — which is a real
  argument for putting it there rather than in a separately-registered singleton.

**Keyed services** (`AddKeyedScoped<T>`/`[FromKeyedServices("name")]`, available from .NET 8) are the
clean answer to "two implementations of one interface, chosen by name" — a storage client per bucket, a
formatter per output kind. Reach for them instead of a hand-rolled factory-with-a-switch, but only when
the choice is genuinely by key; a single implementation registered with a key is indirection for
nothing.

### Endpoint mapping

Group endpoints per feature and mount the groups in one place, so the composition root shows the whole
surface:

```csharp
// Api/Endpoints/<Feature>Endpoints.cs
public static class ReportEndpoints
{
    public static void MapReportEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/reports").WithTags("Reports");
        group.MapPost("/", CreateAsync).RequireAuthorization(<policy>).WithName("CreateReport");
        group.MapGet("/{id:guid}", GetAsync).RequireAuthorization(<policy>).WithName("GetReport");
    }
}

// Program.cs — the second wiring, without which the first one does nothing
app.MapReportEndpoints();
```

**That is two registrations, not one**, and the second is in a different file. A handler and a mapped
route in the feature's own file compile, unit-test green, and are unreachable if nobody called
`MapReportEndpoints()` in `Program.cs`. The endpoint blueprint topic owns the verification step; the
point here is architectural — *the composition root is where reachability is decided*, and a change
that adds a group is not finished inside its own file.

**Pipeline order is behaviour, not formatting.** Middleware runs in registration order, so where a
component sits decides what it can see: something registered before authentication never sees an
identity, something registered after the endpoint terminal never runs at all. When adding to the
pipeline, place it against the existing order deliberately and say why in the hand-off.

### Options binding — the section name *is* the environment prefix

This is the composition-root failure that produces no error at all. A consumer binds a section:

```csharp
builder.Services.Configure<SmtpOptions>(builder.Configuration.GetSection("Smtp"));
```

so every environment variable feeding it must be `Smtp__<ExactPropertyName>` — the double underscore
is the configuration hierarchy separator, and environment variables outrank `appsettings.json` in the
default builder's provider order, so under a container orchestrator the appsettings literal stops
deciding anything.

**A prefix drift and a property-name drift both bind nothing, and neither fails.** The process starts,
the options object is materialized with its defaults, the pipeline runs. Before adding or renaming any
options key:

1. **Read the `GetSection(...)` call in the consuming service.** That string is the source of truth, not
   the class name, not the appsettings block, not the variable naming in a neighbouring service.
2. **Read the options class for the exact property names.** `Username`/`Password` is not `User`/`Pass`;
   a class with `FromAddress` and `FromName` and no `From` property will ignore `From` forever.
3. **Check which host actually binds the section.** Keys set on a host that binds no such section are
   **inert** — and configuring them there, then testing there, reproduces exactly the
   "looks configured, does nothing" reading that started the investigation.

Three defences, in increasing order of strength:

- **Verify a binding with a distinctive override.** Where an options class's defaults were chosen to
  match the development environment — the natural, helpful thing to do — **a total binding failure and
  a working configuration are observationally identical**. So prove it with a value the default cannot
  produce and observe that value downstream. An override that happens to match the default proves
  nothing.
- **Fail fast at startup.** `AddOptions<T>().Bind(section).ValidateDataAnnotations().ValidateOnStart()`
  converts the silent case into a startup failure — the single highest-value line in this whole
  section, because it moves the defect from "months later, in production, as an absence" to "now, in
  the log, by name".
- **Derive both sides of a fact from one variable.** Where a value must be true on both sides of a
  boundary — a published port and the URL handed to a client, an issuer and its validator, a service
  name and the connection string that dials it — write it once and derive both. Two literals for one
  fact are not "in sync", they are *currently equal*, and correcting the second literal restores the
  equality while **preserving the defect**. Where derivation is genuinely impossible, make the variable
  required and fail loudly rather than defaulting quietly.

## 6. References that bite on a cold build

These are project-file facts, not architecture — but each one produces a compile or tooling error whose
message does not name the cause, and each costs a worker an hour the first time.

- **EF Core Design must be referenced by the *startup* project**, not only by the project holding the
  DbContext, or `dotnet ef migrations add` refuses with *"Your startup project doesn't reference
  Microsoft.EntityFrameworkCore.Design"*.
- **A non-web project using ASP.NET Core abstractions needs the framework reference.** If Infrastructure
  implements a current-user accessor over `IHttpContextAccessor`, it needs
  `<FrameworkReference Include="Microsoft.AspNetCore.App" />`; without it a cold build fails with
  `CS0234: namespace 'AspNetCore' does not exist`. (The clean-architecture alternative — move the
  accessor to the host and keep only the interface in Infrastructure — is equally valid; follow the
  project.)
- **Hosting is required wherever a `BackgroundService` lives**, including inside Infrastructure.
- **A Worker-SDK project may need `Microsoft.Extensions.Hosting` explicitly.** Whether `Host` is
  transitively available has changed across SDK versions; if a cold build fails with
  `CS0103: The name 'Host' does not exist in the current context`, that is the cause.

**Follow the project's pinned versions rather than a version list from anywhere else.** Major-version
API breaks are real in this ecosystem's common dependencies — the broker client's channel API differs
between its major lines, and the storage and validation packages have moved too. Read the `.csproj`
before writing the first call.
