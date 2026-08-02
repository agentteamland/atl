# Production unit — adding an endpoint

The thing this pack builds over and over is an **endpoint**: a route line, the schema that validates
the request, the controller that translates HTTP to the domain, and the service that holds the rule.
This page is the lifecycle for one, start to finish.

Two of its steps exist for one reason, and it is worth stating up front. The characteristic failure
of a scaffolded endpoint is **not** a bad body — it is a **correct body that nothing routes to**. A
controller written but never mounted typechecks, lints and unit-tests completely green, and is
unreachable from the running service.

This pack is better armed against that than most, and step 4 says so plainly: the supertest
integration test prescribed in [testing.md](testing.md) drives the **real assembled app**, so a
never-mounted route fails it. What step 4 then does is mark exactly where that guarantee stops —
because it stops one line earlier than it looks.

## 1. Decide before you write

Settle these first — renaming later means touching the route, the controller, the schema, the tests,
and every caller that already integrated against the path.

| Question | How to answer it |
|---|---|
| **A new endpoint, or a new shape on an existing one?** | A new path when it is a different resource, or a genuinely different action on one. A new field / query param when it refines the same action. A path is a public contract: cheaper to add than to retire. |
| **Which resource owns it — and therefore which router?** | The router that already owns that resource's paths. A *new* router is also a new mount point in the composition root and a new prefix in every caller's mental model; take one only when the resource is genuinely new. |
| **Method, path, success status.** | Settle all three before writing: the verb, the path shape (plural resource, id in the path, filters in the query), and the success code — `201` for a create, `204` for a delete with no body. See the status-code discipline in [endpoint-conventions.md](endpoint-conventions.md). These three *are* the contract. |
| **What is the rule, and what are its failures?** | Name the service function, and each typed error it throws with its envelope `code` and its 4xx. If you cannot name a failure, this is probably a thin read and its tests belong one layer down at the service. |
| **Does it change the schema?** | If yes it ships a forward-only migration ([data-and-persistence.md](data-and-persistence.md)) — which changes both the verification in step 4 and the review risk. Decide now, not after the controller is written. |
| **Is it authenticated / authorized?** | Decide before mounting: this determines *where in the middleware chain* it goes. Getting it wrong produces a green test and an open endpoint — see step 4. |

## 2. Scaffold

Three files, in this order — the rule first, the HTTP last. Layering per
[endpoint-conventions.md](endpoint-conventions.md).

```typescript
// services/<resource>.ts — plain typed function, no express, no req/res. The rule lives here.
export async function archiveOrder(input: ArchiveInput, repo: OrderRepo): Promise<Order> {
  const order = await repo.get(input.id);
  if (order.status === "shipped") {
    throw new DomainError("ORDER_ALREADY_SHIPPED", "a shipped order cannot be archived");
  }
  return repo.archive(order.id);
}
```

```typescript
// controllers/<resource>.ts — validate at the boundary, call the service, shape the response.
export const ArchiveSchema = z.object({ id: z.string().uuid() });
export type ArchiveInput = z.infer<typeof ArchiveSchema>;   // one schema → runtime check + type

export const postArchiveOrder: RequestHandler = async (req, res) => {
  const input = ArchiveSchema.parse({ ...req.params, ...req.body });  // throws → central handler
  const order = await archiveOrder(input, req.repos.orders);
  res.status(200).json(order);
};
```

```typescript
// routes/<resource>.ts — HTTP wiring only. No logic to test lives here.
ordersRouter.post("/:id/archive", requireAuth, postArchiveOrder);
```

The controller never hand-formats an error and never hand-writes an `if (!req.body.x)` check: the
schema *is* the input contract, and the one central error handler owns every failure response.

One version caveat on that `throws → central handler` comment: a rejection from an `async` handler
reaches the error middleware natively on express `^5.x` **but not on `^4.x`**, which the dependency
baseline still pins by default. On a `^4.x` project, wrap the handler exactly the way its existing
handlers are wrapped — see the pitfall in step 5.

## 3. Register it — two wirings, not one

An endpoint is reachable only after **both** of these are true, and only one of them is in the file
you just wrote:

```typescript
// 1. routes/<resource>.ts — the method + path + middleware chain, on the resource's router.
ordersRouter.post("/:id/archive", requireAuth, postArchiveOrder);

// 2. src/app.ts — the composition root: the router mounted under its prefix, in the chain's order.
app.use(express.json());
app.use("/v1/orders", ordersRouter);   // ← a NEW router needs this line; an existing one already has it
app.use(errorHandler);                 // ← the four-arg handler, registered LAST, always
```

**Read both files before writing either, and match the siblings.** If the router already applies
`requireAuth` at the mount, do not re-declare it per route; if every sibling declares its middleware
inline, declare yours inline too. A registration that looks unlike its neighbours is the one a
reviewer skims past — and the one whose *ordering* differences nobody notices.

**Order is part of the registration, not a detail.** Express runs middleware in registration order, so
a router mounted after `errorHandler` never reaches it, and a router mounted outside the auth
middleware is simply unauthenticated. Position is behavior here.

**A new dependency is a third wiring.** If the controller reads something off the request that the app
attaches (`req.repos.orders` above, a request-scoped context, a tenant), a new one must be attached in
the composition root too. Missing it fails at *runtime*, not at compile time — `undefined` is a valid
type-check away from a 500.

## 4. Verify the durable effect

The good news first, because it is real: **this pack's prescribed test already drives the composed
root.** The supertest integration test in [testing.md](testing.md) imports the assembled `app` from
the composition root and issues a real HTTP request through the full middleware chain — so the plain
never-mounted route comes back `404` and that test fails. The success-shape test ("creates … and
returns 201") is what provides the guarantee; the boundary `400` and the `4xx`-with-envelope tests
extend it to the error path, which means a router mounted *after* `errorHandler` also fails. If you
wrote the prescribed integration tests, the classic omissions are caught. Say that plainly instead of
performing doubt.

**The false green this pack does not prevent is one line further up: an integration test that
assembles its own app.**

```typescript
// ✅ the app the process actually serves
import { app } from "../src/app";

// ❌ *a* composed root, not *the* one — and it can mirror the prefix, so the test URL is identical
const app = express().use(express.json()).use("/v1/orders", ordersRouter);
```

The second one is a completely fictional guarantee, and **every gate goes green over it**:
`npm run typecheck` compiles it, `npm run lint` likes it, `npm test` passes it — the suite exercises
routing, validation and the envelope, all against an app that exists only inside the test file, while
`src/app.ts` never mounts the router. No gate in this pack can tell the two apps apart.

This is measured, not assumed. Against a composition root that builds the router and never mounts it,
the *same* request URL gives:

```
prescribed test, imports the real app  →  404   test FAILS, omission caught
test assembles its own app             →  200   test PASSES, endpoint unreachable in production
```

The difference is one line at the top of the test file, so **check it by eye**, then observe the
endpoint outside the suite:

```bash
npm ci && npm run typecheck && npm run lint && npm test   # necessary — not sufficient here

grep -n "src/app" <the endpoint's test file>              # the test imports the REAL app

# The composed app's route table — this stack's equivalent of a CLI's `--help`.
# Run it against the build output with plain node; no new dependency is needed.
node -e '
  const { app } = require("<build-output>/app");
  const router = app._router ?? app.router;   // check _router FIRST: on express 4 `.router` THROWS
  const mount = (l) => l.regexp               // express 5 layers no longer carry `.regexp`
    ? l.regexp.source.replace(/\\\//g, "/").replace(/^\^/, "").replace(/\/\?\(\?=\/\|\$\)$/, "")
    : "?";
  for (const l of router.stack) {
    if (l.route) console.log(Object.keys(l.route.methods).join("|").toUpperCase(), l.route.path);
    else if (l.handle?.stack) for (const s of l.handle.stack)
      if (s.route) console.log(Object.keys(s.route.methods).join("|").toUpperCase(), mount(l) + s.route.path);
  }'

# And once through a real socket, success AND failure:
curl -s -i -X POST localhost:<port>/<prefix>/<path> -H 'content-type: application/json' -d '<valid>'
curl -s -i -X POST localhost:<port>/<prefix>/<path> -H 'content-type: application/json' -d '<invalid>'
```

The route table earns its place: a test asserts the path *the developer believed in*, while the table
prints **every mounted route next to its siblings** — so a doubled prefix
(`POST /v1/orders/v1/orders/:id`), or a route sitting under `/orders` while every neighbour is under
`/v1/orders`, is visible at a glance and invisible to a test written from the same wrong assumption.
Mind the version: on express `^4.x` the mount resolves and each line *is* the caller-visible path; on
`^5.x` the layer no longer exposes its mount, so the prefix prints as `?` and only the
router-relative half shows — a duplicated segment still stands out, but there the **live call is what
proves the caller-visible path**. The port, prefix and start command are project truths — take them
from the durable-knowledge pages the brief names, not from this pack.

Two more durable effects to observe when they apply:

- **Auth placement — the case where the green is *caused by* the omission.** A router mounted outside
  the auth chain returns `200` in the happy-path test **because** nothing guards it. No status
  assertion can distinguish "authorized correctly" from "not protected at all". Call the endpoint
  once with no credentials and confirm you get the `401` you designed.
- **A migration.** The integration suite applies migrations to a *fresh, empty* database, which proves
  the SQL is valid — not that it applies to a populated one. For anything destructive (a `NOT NULL`
  on a populated table, a type narrowing, a drop) confirm the expand→migrate→contract sequencing in
  [data-and-persistence.md](data-and-persistence.md) and say so in the hand-off.

Attach the integration-test output — the `2xx` shape plus the boundary `400` and the `4xx`-with-envelope
rejections — to the work-item as the Level-1 self-test evidence ([testing.md](testing.md)). A raw pass
count is not evidence for this unit type; the demonstrated contract is.

## 4b. Ship the test — the unit is not done without one

Step 4 proved the thing works **now**, by looking at it. This step is what keeps it working: a unit
is not complete until it ships a test covering the behaviour it added. The policy, identical across
packs, is [`testing-surfaces.md` §7](../../knowledge/testing-surfaces.md):

> **diff coverage >= 90%** of the lines this unit added or modified, **and** at least one test that
> goes RED when the change is reverted.

The second half is not ceremony. Coverage proves a line *ran*; it says nothing about whether anything
*checked* the result — a test that exercises the code and asserts nothing scores 100%. Confirming it
costs a minute: revert the change, run the test, see red, restore.

Coverage for this stack:

```bash
npm test -- --coverage
```

**The false green peculiar to this stack is already named in step 4:** a test that builds its own
little app instead of importing the real one. It exercises routing, validation and the envelope
against an application that exists only inside the test file — so it passes while the real endpoint
is unmounted, unauthenticated, or wired into the wrong middleware position. Coverage does not catch
it; the test genuinely runs those lines. Check the import by eye.

If 90% is genuinely unreachable because the new code is entangled with something untestable, record
the exception on the work-item — which lines, and why. Never a silent pass.

## 5. Common pitfalls

- **The router is new and nobody mounted it.** The commonest form of the failure: the route line looks
  complete because the file it is in is complete. A new `Router` needs its `app.use(prefix, router)`.
- **A doubled prefix.** `router.post("/v1/orders/:id")` inside a router already mounted at `/v1/orders`
  yields `/v1/orders/v1/orders/:id`. The route table in step 4 shows it immediately.
- **`errorHandler` registered before the new router.** Express matches in order, so that router's
  throws never reach it — failures fall through to Express's default handler and return an HTML stack
  trace instead of the envelope, leaking internals on the way. The error handler is registered last.
- **An async rejection that never reaches the handler.** On express `^4.x` a rejected promise from an
  async handler is **not** forwarded to the error middleware — the request hangs and the process logs
  an unhandled rejection. Wrap async handlers (or be on `^5.x`, which forwards rejections natively).
  Whichever the project pins, match its existing handlers.
- **`app.listen` called at import time.** It makes supertest bind a real port and flake under the
  parallel-worker load. `src/app.ts` exports the app; `src/server.ts` listens ([testing.md](testing.md)).
- **Validation moved into the service.** It duplicates the guard across handlers, leaves the `400` path
  untestable at the boundary, and means the service can no longer trust its own input type. Validate
  at the edge ([endpoint-conventions.md](endpoint-conventions.md)).
- **The published contract not updated.** A new endpoint is a caller-facing surface; if the project
  keeps an API spec or generated client types, they drift silently. The brief names that gate when it
  applies.

## 6. Hand-off

The work-item is ready for the tester when: the full code-surface gate is green
(`npm ci && npm run typecheck && npm run lint && npm test`), the integration tests **import the
composed app** and cover the success shape plus the boundary `400` and the `4xx`-with-envelope
rejections, that output is attached, the endpoint has been observed on the composed app (route table
and/or a live call), auth behavior has been exercised rather than assumed, and any migration's
sequencing is stated.

State plainly in the hand-off which of these you **observed** and which you **assumed** — the import
line in the test file most of all, since it is the difference between a proven contract and a fictional
one. An unverified claim is worse than an admitted gap: the tester's Level-2 pass is what gates the
green, and it can only check what it is told about.
