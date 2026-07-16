# Testing — unit, supertest integration, the CI gate, and evidence

An `api` work-unit is verified entirely on the **code surface** — there is no browser or emulator
to drive ([mobile-and-web-surfaces.md](../../agents/tester/children/mobile-and-web-surfaces.md)).
That makes the code-surface gate *the whole gate*, so it has to be trustworthy on its own: unit tests
for the logic, HTTP integration tests for the wiring and the contract, a typecheck and lint so a
green suite over broken or sloppy code can't pass as green. This topic is the generic
Express/TypeScript craft for that gate, and how its result becomes attached evidence on the
work-item.

The developer's testing here is **Level-1 self-test** — the developer's own micro-loop step (fast,
driven by this pack), which the developer runs before opening a PR. It is **not** Level-2: a
separate `tester` worker does the thorough strategy/edge/regression pass that gates the green
([test-strategy.md](../../agents/tester/children/test-strategy.md)). `green = test-gates ∧ review`
([review-craft.md](../../agents/tech-lead/children/review-craft.md)) — the developer's job is to make
the code-surface gate green and attach the proof; it does not merge and does not sign off on itself.

## The pyramid for an Express API

Match effort to the layering in [endpoint-conventions.md](endpoint-conventions.md): most tests are
cheap in-process unit tests; a smaller band of integration tests proves the HTTP wiring; there is no
higher surface for an `api` unit.

- **Unit (the wide base) — services + pure logic.** Because services are plain typed functions with
  no `req`/`res` and take their repository as a dependency, they test with a **fake repository** and
  no server and no database — the fastest, most numerous tests. Every business rule, boundary, and
  error path a service can produce is proven here, in milliseconds, in parallel.
- **Integration (the thin band) — the HTTP contract, via supertest.** A handful of tests drive the
  *real Express app in-process* to prove the wiring the unit tests can't see: routing, the validation
  middleware, the central error-envelope, the status codes, and (for the repository/migration/
  transaction layer) behavior against a real disposable database.
- **No web / no mobile surface.** An `api` unit ships no rendered UI, so it has neither the web
  (chrome-devtools/preview MCP) nor the mobile (emulator-lease) surface. Its consumers — a `web` or
  `mobile` unit — exercise the API over HTTP from *their* surface; this unit's contract is proven at
  the HTTP boundary here.

**Why push logic down.** An HTTP round-trip test is slower and couples a rule to the transport;
proving a rule through supertest when a unit test would do wastes the isolated worker's time and the
CI budget. Reserve integration tests for what only the assembled app can show — the wiring and the
contract — and let the wide, fast base carry the logic. This is the pyramid's "thin top" rule applied
to a code-only unit.

## Integration tests with supertest

`supertest` drives the Express `app` **in-process** — no listening port, no network — issuing real
HTTP requests through the full middleware chain and asserting on the response. It is the tool that
proves the endpoint's *contract*: not "does the service compute the right number" (a unit test) but
"does a `POST` with this body come back `201` with this shape, and a bad body come back `400` with
the error-envelope."

```typescript
import request from "supertest";
import { app } from "../src/app";

describe("POST /transfers", () => {
  it("creates a transfer and returns 201 with the result body", async () => {
    const res = await request(app)
      .post("/transfers")
      .send({ sourceId: "a", destId: "b", amount: 50 });
    expect(res.status).toBe(201);
    expect(res.body).toMatchObject({ status: "completed" });
  });

  it("rejects an over-balance transfer with 4xx and the error envelope", async () => {
    const res = await request(app)
      .post("/transfers")
      .send({ sourceId: "a", destId: "b", amount: 999999 });
    expect(res.status).toBe(422);            // a business-rule rejection is 4xx, never 5xx
    expect(res.body.error.code).toBe("INSUFFICIENT_FUNDS");  // the contract token
  });

  it("rejects a malformed body with 400 and field details", async () => {
    const res = await request(app).post("/transfers").send({ amount: "not-a-number" });
    expect(res.status).toBe(400);
    expect(res.body.error.code).toBe("VALIDATION_ERROR");
  });
});
```

- **Export the `app`, listen elsewhere.** `src/app.ts` builds and exports the configured Express
  `app`; a separate `src/server.ts` calls `app.listen(...)`. supertest imports the `app` and never
  starts a server — this split is what makes the integration tests fast and port-free, and it's a
  convention worth stating because getting it wrong (calling `listen` at import) makes the tests bind
  a real port and flake under the parallel-worker load.
- **Test the failure contract, not just the happy path.** The boundary criteria — the `400`
  validation reject and the `4xx` business reject *with their envelope `code`* — are the evidence a
  reviewer actually needs; a suite that only proves the happy path proves the least interesting half.
  This mirrors the tester's "evidence the guard holds, not the happy path" rule
  ([evidence-collection.md](../../agents/tester/children/evidence-collection.md)).
- **Integration tests that touch the DB run against a real disposable database** — the repository,
  the migrations, and the transaction rollback are real-database facts a mock can't prove
  ([data-and-persistence.md](data-and-persistence.md)). Reset the database per run (apply migrations
  to a fresh/ephemeral instance) so tests are deterministic and independent.

## The CI gate — what "green on the code surface" means

The developer's Level-1 self-test — and the CI gate the tester and the tech-lead's review build on —
is the full conjunction, all green:

```bash
npm ci               # deterministic install from package-lock.json (never `npm install` in CI)
npm run typecheck    # tsc --noEmit — a suite passing over code that doesn't compile is a false green
npm run lint         # eslint . — style/quality gate
npm test             # the unit + supertest integration suites (e.g. jest --ci / vitest run)
```

- **Typecheck is part of the gate, not optional.** TypeScript tests can pass while the shipped code
  has a type error the runner never exercised; `tsc --noEmit` closes that gap. A green *runtime* suite
  over code that doesn't type-check is a false green — one of the cheapest false greens to catch and
  the easiest to forget.
- **`npm ci`, not `npm install`, in CI.** `ci` installs the *exact* locked tree and fails on a
  lockfile mismatch, so the gate runs against a reproducible dependency set — the same
  pin-your-baseline discipline the pack's dependency baseline states.
- **The conjunction is ordered and complete.** All four green is the code-surface gate; any one red
  is not green. This is the developer's half of `green = test-gates ∧ review` — the developer makes
  this pass, the tester's Level-2 pass and the tech-lead's review complete the green, and only then
  does the deterministic engine merge (the developer never merges or self-sets Done —
  [azure-adapter.md](../../backends/azure/adapter.md) §6 keeps the Done transition runtime-resolved).

## Evidence — attach the proof to the work-item

A green claim is only as trustworthy as its proof. The developer captures the gate's output and
attaches it to the Azure work-item so the tech-lead's review and the PO's sprint-review can inspect
the green rather than take it on faith — the same evidence discipline the tester follows
([evidence-collection.md](../../agents/tester/children/evidence-collection.md)).

- **Capture the test-run output** — the full `typecheck + lint + test` result showing every gate
  green (or the failure, on a fail verdict) — to a file, e.g. `test-run.txt`.
- **Attach it via the ONE REST carve-out**, `scripts/az-attach.sh` (azure-adapter §9 — the single
  Azure operation with no MCP tool; every other Azure call goes through the `azureDevOps` MCP). From
  a worker's context the relative path is
  [`../../scripts/az-attach.sh`](../../scripts/az-attach.sh):

  ```bash
  ../../scripts/az-attach.sh <work-item-id> test-run.txt "code-surface gate: typecheck + lint + unit + integration all green"
  ```

  The helper reads the PAT from the worker's **environment** (the same `AZURE_DEVOPS_PAT` the MCP
  uses), never from the argv and never logged — the developer never writes, echoes, or passes a
  literal token.
- **Self-describing, tied to the criterion.** Name the file and pass a `comment` that says what it
  proves, so the attachment is legible on its own at review time. For an `api` unit the decisive
  evidence is the integration-test output demonstrating the *contract* — the `2xx` success shape and
  the boundary `4xx`-with-envelope rejections — not just a raw pass count.
- **Reading evidence back** uses the MCP (`wit_get_work_item_attachment`), so only the upload leg is
  REST; the developer owns the upload, the reviewer pulls it back through the MCP.

## Testing checklist

- [ ] Service/pure logic covered by unit tests against a fake repository (no server, no DB) — the
      wide base.
- [ ] The HTTP contract covered by supertest integration tests: success shape + status, boundary
      `400` validation reject, `4xx` business reject *with the error-envelope `code`*.
- [ ] `app` exported separately from `listen` so supertest runs in-process, port-free.
- [ ] DB-touching integration tests run against a real disposable database (migrations applied fresh,
      reset per run); rollback behavior actually exercised.
- [ ] The full gate green: `npm ci && npm run typecheck && npm run lint && npm test` (typecheck and
      lint are part of the gate — a runtime-only green is a false green).
- [ ] Test-run output captured and attached to the work-item via `../../scripts/az-attach.sh` (PAT
      from env, never argv), with a self-describing filename + comment tied to the unit.
- [ ] Understood as the developer's **Level-1** self-test — the `tester`'s Level-2 pass and the
      tech-lead's review complete `green = test-gates ∧ review`; the developer neither merges nor
      self-signs-off.
