---
name: dotnet-api
description: ".NET API craft — ASP.NET Core and EF Core plus the queue, cache, realtime and out-of-request surfaces around them: the stack knowledge a delivery worker loads when its unit lands on a .NET backend."
---

# .NET API

## Identity

I am **.NET API craft** — the knowledge of building a server-side HTTP API in .NET: ASP.NET Core on
the request path, EF Core behind it, and the four surfaces that hang off it (a cache, a message
broker, a realtime hub, and work that runs outside a request).

I am **knowledge, not a worker.** `atl work dispatch` spawns the delivery-team's `developer`; that
worker becomes competent on this stack by loading me, exactly where it would otherwise load a stack
pack. So I carry none of its role-craft — no worktree handling, no claim protocol, no PR contract,
no escalation rules. Those live in the `developer`'s own `children/` and travel unchanged across
every stack; duplicating them here would create a second copy that drifts.

**I declare what I am; I never declare which area I belong to.** An `area:` tag is a functional
slice of one particular system — this project may call my slice `api`, the next `backend`, the next
`core-service` — and that vocabulary belongs to the project's tech-lead, who binds stack to area on
the `Architecture/` page they already own. I am the .NET API stack. Where it lands is not my call.

What makes this stack worth a specialist is the *shape* of its failures. The characteristic .NET
API defect is not one the compiler, the test suite or a clean-environment run can see: a batch
insert that is quadratic only because of a `SaveChanges` override, a projection that throws on its
first execution, a query filter frozen into a cached plan, a configuration section that binds
nothing and starts cleanly, a broker argument that fails only where data already exists. My job is
to make a worker aware of those before it writes the line that produces one.

**Version baseline:** authored against .NET 8/9-era ASP.NET Core (Minimal APIs, `IExceptionHandler`,
`AddProblemDetails`) and EF Core 8/9. Treat that as the baseline a project may move, not as gospel;
where a rule is genuinely bound to a version, the child topic says so at the line.

## Area of Responsibility

I do:
- **Placement** — where a change goes: the layer, the feature slice, and which host in a multi-host
  solution can even hold it (the composed host owns persistence; thin hosts reach it over HTTP).
- **The production unit** — an endpoint, end to end: decide → scaffold the request/handler/validator
  → **register it in the endpoint mapping and the composition root** → verify it is actually
  reachable in the running app.
- **Persistence** — EF Core query and save craft, the `SaveChanges` seam, tracking cost, projection
  translation, plus the DDL side: migrations, indexes, concurrency tokens, soft delete.
- **The isolation boundary** — multi-tenancy and authorization: where the boundary genuinely is,
  what it costs to bypass it, and why it degrades *open* the moment there is no request.
- **The wire contract** — the error shape the API emits, the one place status codes are mapped, and
  the paths that bypass that place entirely.
- **The surrounding surfaces** — cache and Redis primitives, the message broker and its consumers,
  scheduled and hosted work, and the realtime push channel.
- **Verification** — what counts as evidence that a .NET API change works, and which cheap-looking
  checks produce a green that is not one.
- **Accumulate craft** — durable .NET lessons land in my `children/` via the capture → `/drain`
  loop, so what one project pays for, the next one inherits.

I do NOT:
- **Own an area, or name one.** Binding is the tech-lead's, on their `Architecture/` page. A shipped
  agent that hardcoded `area:api` would be wrong on the first project that calls it something else.
- **Do the delivery worker's job.** Claiming the work-item, running the worktree, the Level-1
  self-test protocol, opening the PR, escalating a blocker — all the `developer`'s, all
  stack-independent, all deliberately absent here.
- **Review or merge.** Review is the tech-lead's and merge follows it; I am read by the hand that
  writes the code, which is exactly the hand that must not sign it off.
- **Hold project truth.** The project's real module layout, its pinned libraries, its error-code
  catalog, its area names and its deployment topology live in the durable-knowledge store
  (`Architecture/`, `Conventions/`, `Domain/`), named page-by-page in the tech-lead's canonical
  brief. I am generic .NET craft layered *under* that; where the two disagree, the project wins.
- **Stack alongside a generic pack for the same unit.** Where I am bound I *replace* the area's
  pack. A worker reading both reads two documents written by different hands about the same
  decision, and has no rule for which to obey.
- **Cover the browser half.** A client's consumption of this API — its data-fetching, its cache
  invalidation, its reconnect behaviour — belongs to the web specialist. I own the server side of
  every contract I describe, and say where the other half lives.

## Core Principles

### 1. The compiler is not the gate
Every failure worth writing down in this stack builds clean and passes the tests written for it.
Some need volume, some need a second write path, some need concurrency, and some need only an
environment that already holds data. So the standing question for any change is not *does it
compile* but **what would a fresh environment fail to reproduce** — and then whether the change has
been exercised against that condition rather than around it.

### 2. One owner per invariant, and it sits at the seam
Cross-cutting behaviour — tenant stamping, soft delete, audit, cache invalidation, error mapping,
a user-facing string — belongs at the chokepoint every path passes through, not re-declared per
handler. Per-handler is correct only while you own every write path, and you stop owning it the
first time a scheduled job or a queue consumer writes the same entity; then it is a checklist, and
it rots silently. **Never run both mechanisms for one invariant:** two owners is how a thing gets
applied twice and reasoned about nowhere. The corollary is an estimating rule — for a cross-cutting
change, count the chokepoints, not the call sites.

### 3. A boundary is only where the server enforces it
The query filter and the endpoint policy are the boundary. A hidden menu, a role guard in the UI, a
pre-filtered list are presentation. And where a filter has been stripped, a hand-written scope
comparison *becomes* the boundary at that point — so a scope variable that resolves to null there
means **every** tenant, not *my* tenant. Never copy a scope-narrowing idiom out of a neighbouring
handler without checking whether that neighbour still had its filter on.

### 4. Outside a request there is nothing ambient
No user, no tenant, no correlation, no idempotency header, no `HttpContext` — and the dangerous part
is that the defaults do not throw, they **open**. Background work therefore carries its scope in the
message and applies it by hand, stamps its writes explicitly, and — because a broker redelivers and
a scheduler re-fires — is idempotent by construction, with a database-hard backstop rather than a
hopeful check.

### 5. Registered is not written, and green is not reached
A handler with no endpoint mapping, an endpoint absent from the composition root, a hosted service
never registered, an options class whose section name drifted, a migration never applied: each one
compiles, unit-tests green, and does nothing at runtime. Every unit ends by **observing the durable
effect** where the system itself would find it — the route in the running app, the row in the
migrations history, the bound value at startup — and by naming out loud which passing gate would
have missed the omission.

## Knowledge Base

Read the child file before acting on its topic; the summaries below are a routing index, not the
full instructions.

<!-- Auto-rebuilt from children/*.md frontmatter. Do not hand-edit — /drain rebuilds this from each child's `knowledge-base-summary`. -->

### Architecture And Layering
Where a change goes: inward-pointing Domain / Application / Infrastructure / Api layers, feature-slice
organization, and the rule that an endpoint is a bridge while the handler is the one place logic lives.
Answered before any of that: which host can even hold the change — only the composed host owns the
DbContext and dispatch, so "just move it to the worker" means moving the whole persistence stack. Plus
composition-root craft, including options binding where the section name IS the environment-variable
prefix, so a prefix or property drift binds nothing and still starts cleanly.
→ [Details](children/architecture-and-layering.md)

---

### Caching And Redis
Cache-aside as a pipeline concern rather than handler code, key naming, and TTL discipline — but the
load-bearing half is **ownership of invalidation**: it belongs at the persistence seam every write
passes through, never per command, and never both. Collect keys before the save, evict after it
commits. Plus Redis as a distributed primitive — one multiplexer, locks, sliding-window rate limits,
and the request-path idempotency store (which has no equivalent inside a consumer).
→ [Details](children/caching-and-redis.md)

---

### Endpoint Blueprint
My production unit, end to end: decide the contract (route, verb, success status, auth, does it change
the schema) → scaffold the request, validator, handler and endpoint → **register it on its endpoint
group AND in the composition root** → verify it is reachable in the composed app → pitfalls →
hand-off. The characteristic failure is a correct handler nothing routes to, and it passes every unit
test; step 4 names the false green out loud, including the variant a fully compliant worker still
hits — an in-process HTTP test that builds its own host instead of the app's.
→ [Details](children/endpoint-blueprint.md)

---

### Error Contract
The wire shape errors go out in, and the single place status is mapped. `ProblemDetails` everywhere,
with the validation status carrying a field-keyed dictionary whose keys are the property names a
client binds form fields on — a contract, not display text. Three traps: serialize by the RUNTIME
type or a derived problem held in a base variable silently loses its dictionary; framework-generated
responses (auth challenge, policy denial, unmatched route) never reach the exception handler at all;
and a model-binding failure escapes as a 500 unless one central arm maps it.
→ [Details](children/error-contract.md)

---

### Messaging And Background Work
Everything that runs outside a request and the invariants it shares: no ambient identity or tenant
(carry the scope in the message, stamp writes by hand), redelivery is guaranteed so idempotency is by
construction with a database-hard backstop, and cancellation is honoured on shutdown. Broker craft —
topology, publisher confirms, prefetch, DLX — turns on one rule: **a queue's arguments are fixed at
creation**, so changing them crash-loops the consumer on exactly the environments that already have
data and never on a fresh broker. Plus scheduled hosted work and the internal-endpoint call-back seam.
→ [Details](children/messaging-and-background-work.md)

---

### Multi Tenancy And Authorization
The isolation boundary: a global query filter on the context is the boundary, an inbound guard is
defence in depth, and scoping stays centralized rather than scattered into handlers. Reference the
tenant through the context instance, never a captured service — a captured constant is funcletized
once and frozen into the cached query plan, which is a real cross-tenant leak. Authorization is an
endpoint policy, never a UI affordance. Reason about bypassing a filter from the query's ROOT, not a
count of call sites — and remember the filter degrades OPEN where there is no request.
→ [Details](children/multi-tenancy-and-authorization.md)

---

### Persistence And Migrations
EF Core on both sides. Runtime: what a `SaveChanges` override costs per call, why a chunked batch
insert through one context goes quadratic unless the tracker is cleared, why an aggregate projection
must materialize before constructing a DTO, how a lost check-then-insert race is made to converge, and
the tracking/projection/N+1 rules underneath. Schema: a second index declaration over the same
property list silently REPLACES the first; a scaffolded migration matches dropped columns to added
ones **positionally**, so read `Up()` before trusting it; never scaffold against a stale build; a
system-column concurrency mapping needs a hand-emptied migration on an existing table. The
through-line on both sides: verify against the database and a real execution, never against the
configuration and never against the build.
→ [Details](children/persistence-and-migrations.md)

---

### Realtime Push
The server half of the push channel: hubs as bridges rather than logic, authenticating a connection
whose transport cannot carry headers, group and connection tracking, and the delivery contract that
keeps a push cheap — **persist the record first as the source of truth, then push best-effort**,
short-timeout and swallowed on failure because the client's next fetch already covers it. Token
lifetime is the sharp edge: an idle client generates no other traffic to refresh with, so expiry
handling that looks safe on the server kills push precisely while nobody is watching.
→ [Details](children/realtime-push.md)

---

### Testing And Verification
What counts as evidence, and which cheap-looking checks produce a green that is not one. The tiering:
unit tests on the rule, in-process HTTP tests against the **composed** app (the only gate that catches
an unregistered endpoint), and a real dependency — a throwaway container — for anything whose
behaviour lives in the broker, the cache or the database rather than in C#. The rule that generalizes:
a fresh environment cannot prove a fix for a defect that only exists where data already is, so
exercise BOTH states. A method with zero callers is unverified code, not working code.
→ [Details](children/testing-and-verification.md)
