# Endpoint conventions — routes, validation, errors, status codes

An HTTP endpoint is a contract with every caller: given a shaped request, it returns a shaped
response or a shaped failure. This topic is the generic Express/TypeScript craft for making that
contract **uniform, testable, and honest** — the same three-layer route→service split, the same
validation-at-the-boundary rule, the same one error-envelope, on every endpoint. Uniformity is
the point: a caller (and a reviewer) should be able to predict an endpoint's success and failure
shape without reading its body, and a developer should be able to add an endpoint without
re-inventing the response contract each time.

Project-specific rules (this project's actual route prefixes, its error-code catalog, its auth
middleware) layer ATOP this in the project wiki `Conventions/` page, named by the tech-lead's
canonical brief. This topic is the stack-generic base those overrides sit on.

## The three-layer split: route → service → data

Structure every feature as three thin layers, each independently testable:

```
routes/        HTTP wiring only — method, path, middleware chain, delegate to a controller
controllers/   translate HTTP ⇄ domain: validate the request, call a service, shape the response
services/      business logic — plain functions, typed in and out, NO knowledge of req/res
```

- A **route** does nothing but register the method + path and its middleware, then hand off. It
  contains no logic to test.
- A **controller** (the route handler) is the only place that touches `req`/`res`. It validates the
  input (below), calls a service with plain typed arguments, and maps the service's result to a
  response. It holds no business rules — just translation.
- A **service** is a plain async function: typed input → typed output (or a thrown typed error). It
  never imports `express`, never sees `req`/`res`. **This is where the logic — and therefore the
  bulk of the tests — lives.**

**Why this split.** The service is pure logic with no HTTP scaffolding, so it's covered by fast,
in-process unit tests — no server, no socket, no supertest. Push logic down into services and the
wide, cheap base of the test pyramid does the heavy lifting; the thin controller layer needs only
a handful of integration tests to prove the wiring ([testing.md](testing.md)). Invert this — logic
in the controller reaching into `req` — and every rule can only be exercised through an HTTP round
trip, which is slower and couples the rule to the transport. The split *is* the testability.

```typescript
// services/transfer.ts — pure logic, no HTTP
export async function transferFunds(
  input: TransferInput,
  repo: AccountRepo,
): Promise<TransferResult> {
  const source = await repo.get(input.sourceId);
  if (source.balance < input.amount) {
    throw new DomainError("INSUFFICIENT_FUNDS", "transfer exceeds source balance");
  }
  // ... the rule, unit-testable without a server
}

// controllers/transfer.ts — HTTP ⇄ domain translation only
export const postTransfer: RequestHandler = async (req, res) => {
  const input = TransferSchema.parse(req.body); // validate at the boundary
  const result = await transferFunds(input, req.repos.accounts);
  res.status(201).json(result);
};
```

## Validate at the boundary, trust inward

Every request's `body`, `params`, and `query` is validated by a schema **at the controller edge,
before any service runs**. A validated value carries its static type from that point on; a service
receives input that is already both shape-checked *and* typed.

- Use a schema library that yields the static type from the same schema (the baseline pins `zod`) —
  one declaration produces both the runtime check and the TypeScript type, so the two can never
  drift. `type TransferInput = z.infer<typeof TransferSchema>`.
- A validation failure is a client error, not a server crash: the schema parse throws, and the
  central error handler (below) turns it into a `400` with the field-level detail. The controller
  does not hand-write `if (!req.body.amount)` checks — the schema *is* the input contract.

**Why boundary validation.** [untrusted-input](../../../../core/rules/untrusted-input.md): request
data is data the server did not author — it must be validated before it's acted on. Concentrating
that at the boundary means a service can be written against a trusted, typed input and stays free of
defensive re-checking; the safety is proven once, at the edge, and the type system carries it
inward. Scattered inline checks, by contrast, leave gaps no reviewer can spot and duplicate the same
guard across handlers.

## One error-envelope, one central handler

Every failure the API returns has the **same JSON shape**, produced in **one place** — a single
Express error-handling middleware (the four-argument `(err, req, res, next)` handler, registered
last). Controllers and services `throw` typed errors; they never format an error response
themselves.

The envelope (a baseline shape a project may extend in its `Conventions/` wiki):

```json
{
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "transfer exceeds source balance",
    "details": [{ "field": "amount", "issue": "must be <= source balance" }]
  }
}
```

- `code` — a stable, machine-readable string a client can branch on. It is **not** the HTTP status
  and **not** a free-text message — it's the contract token. The project's code catalog lives in its
  `Conventions/` wiki page; this pack fixes the *shape*, the project fills the *values*.
- `message` — a human-readable, safe-to-expose summary. Never leak a stack trace, a SQL string, or
  an internal path into `message` — that is both an information-disclosure risk and a contract
  smell.
- `details` — optional, structured, per-field context (validation failures populate it).

The central handler maps a thrown error to `(status, envelope)`:

```typescript
// The last middleware registered on the app.
export const errorHandler: ErrorRequestHandler = (err, _req, res, _next) => {
  if (err instanceof ZodError) {
    return res.status(400).json(toEnvelope("VALIDATION_ERROR", err));
  }
  if (err instanceof DomainError) {
    return res.status(err.status).json(toEnvelope(err.code, err));
  }
  // Unknown = a bug: log server-side, return an opaque 500 (never echo internals).
  logger.error(err);
  return res.status(500).json(toEnvelope("INTERNAL_ERROR", "an unexpected error occurred"));
};
```

**Why one handler.** The failure contract is part of the API just as much as the success contract;
if it's assembled ad-hoc in each controller, it drifts — one endpoint returns `{message}`, another
`{err}`, a third a bare string — and every client must special-case each. Centralizing it means the
contract is defined in exactly one file, every path inherits it, and a reviewer verifies the failure
shape by reading one handler, not N controllers. It also closes the leak risk in one place: the
`500` branch is the single guard that stops an unexpected error's internals from reaching a client.

## Status-code discipline

Status codes are part of the contract; use them to their standard meaning so callers can rely on
the class of a code without parsing the body.

- **2xx** — `200` a successful read/update returning a body; `201` a create (return the created
  resource, or its location); `204` a success with no body (a delete).
- **4xx — the caller's fault, do not retry unchanged.** `400` malformed/failed-validation input
  (the schema-parse path); `401` unauthenticated; `403` authenticated but not permitted; `404` the
  resource does not exist; `409` a conflict with current state (a duplicate, a version clash);
  `422` a well-formed request that violates a business rule (distinct from `400`'s *malformed*).
- **5xx — the server's fault.** `500` an unhandled/unexpected error (the catch-all branch — opaque
  body, logged internally); `503` a dependency is unavailable. A predictable business rejection is a
  4xx, **never** a 5xx — reserving 5xx for genuine server faults is what lets alerting treat a 5xx
  spike as a real incident rather than noise from ordinary rejections.

**Why status discipline is load-bearing.** Callers, proxies, retry logic, and monitoring all key off
the status *class*. Returning `200` with an `{error}` body, or `500` for an ordinary validation
failure, breaks every one of them: a client that trusts the status ships a bug, a retry loop
hammers a request that will never succeed, and the 5xx alert cries wolf. Mapping each failure to its
correct code is how the envelope's `code` and the HTTP status stay a coherent pair — the status
tells the caller the *class*, the `code` tells it the *specific* reason.

## Endpoint checklist

- [ ] Route is thin — method + path + middleware, delegates to a controller; no logic in the route.
- [ ] Business logic is in a plain typed service (no `req`/`res`), unit-tested without a server.
- [ ] Request `body`/`params`/`query` validated by a schema at the controller boundary; the service
      receives typed, safe input.
- [ ] Failures `throw` typed errors; no controller hand-formats an error response.
- [ ] The response (success and failure) matches the project's envelope shape; the central handler
      owns failure formatting.
- [ ] Status code matches the outcome's class (2xx / 4xx-caller / 5xx-server); business rejections
      are 4xx, never 5xx; the `500` branch is opaque (no leaked internals).
- [ ] Covered per [testing.md](testing.md): service logic by unit tests, the wiring + envelope +
      status by a supertest integration test.
