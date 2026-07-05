---
area: api
stack: "Node.js + Express + TypeScript"
---

# API pack — Node.js + Express + TypeScript

This pack covers **server-side HTTP API** work: request-handling endpoints, their
validation and error contracts, and the data/persistence layer behind them. The tech-lead
tags a work-unit `area:api` when its acceptance criteria live in a backend service —
routes, request/response shapes, business logic that runs on the server, database reads and
writes — with no rendered UI surface of its own (a `web` or `mobile` unit consumes this API
over HTTP). A developer worker that loads this pack is building or extending an
Express/TypeScript service, and verifies it entirely on the **code surface** (CI); it has no
browser or emulator surface to drive.

This pack is the **generic stack craft** for this concern — how to build and test an
Express/TypeScript API *anywhere*. The **project's** specific conventions (its actual module
layout, its chosen ORM, its error-code catalog) live in the Azure project wiki
(`Conventions/`, `Architecture/`) layered ATOP this pack, and the tech-lead's canonical brief
names the exact pages. See [`knowledge/pack-format.md`](../../knowledge/pack-format.md) for
the three-layer read contract (pack = generic stack / wiki = project-specific / brief = the
bridge).

## Topics

- [endpoint-conventions.md](endpoint-conventions.md) — route/controller/service structure, request
  validation, the error-envelope response shape, and status-code discipline.
- [data-and-persistence.md](data-and-persistence.md) — database-access conventions, schema
  migrations, and transaction boundaries.
- [testing.md](testing.md) — unit tests + HTTP integration tests via supertest, the CI commands
  that constitute the code-surface gate, and how evidence attaches to the work-item.

## Test commands

- unit / integration: `npm test` (Jest or Vitest running the `*.test.ts` suites, including the
  supertest integration tests — see [testing.md](testing.md))
- typecheck (part of the gate — a green suite over code that doesn't compile is a false green):
  `npm run typecheck` (`tsc --noEmit`)
- lint: `npm run lint` (`eslint .`)

There is no web or mobile surface command — an `api` unit is verified on the **code surface**
(CI) only ([mobile-and-web-surfaces.md](../../agents/tester/children/mobile-and-web-surfaces.md)).

## Key conventions

- **Thin routes, testable services.** A route wires HTTP to a service function; business logic
  lives in a plain function that takes typed inputs and returns typed outputs. This is what lets
  the bulk of coverage be fast in-process unit tests rather than HTTP round-trips (the pyramid's
  wide base). See [endpoint-conventions.md](endpoint-conventions.md).
- **Validate at the boundary, trust inward.** Every request body / params / query is parsed and
  validated by a schema *before* it reaches a service; a service receives already-typed, already-safe
  input. Untrusted input never flows past the edge unchecked.
- **One error-envelope shape, one central error handler.** Handlers throw typed errors; a single
  Express error-handling middleware maps them to the one JSON envelope and the right status code —
  so the API's failure contract is uniform and defined in exactly one place.
- **The database is reached through one data-access layer**, never ad-hoc from a route. Schema
  changes ship as **forward-only migrations**; multi-write invariants run inside a **transaction**.
  See [data-and-persistence.md](data-and-persistence.md).
- **The code surface is the whole gate for `api` units.** `npm run typecheck && npm run lint &&
  npm test` all green is the developer's Level-1 self-test; the tester's Level-2 pass gates the
  green. Evidence (the test-run output) attaches to the work-item.

## Dependency baseline

Versions a team *pins* as its starting point — plausible and current-ish, not gospel; a real team
sets its own baseline in the project wiki and pins exact versions in `package.json` +
`package-lock.json`.

- **express** `^4.19` (or `^5.x` once the team adopts it — v5 is now stable) — the HTTP framework and
  the reference for router/middleware structure.
- **typescript** `^5.4` — static types across routes, services, and the persistence layer; the
  `tsc --noEmit` typecheck is a gate.
- **zod** `^3.23` — runtime request-schema validation that also yields the static input type
  (one schema → both the runtime check and the compile-time type), so the boundary contract can't
  drift from its type.
- **the DB driver / query builder / ORM** — team's choice (e.g. `pg` `^8`, `knex` `^3`,
  `prisma` `^5`, or `drizzle-orm` `^0.30`); pin one, own its migration tool. See
  [data-and-persistence.md](data-and-persistence.md).
- **jest** `^29` (or **vitest** `^1`) + **supertest** `^7` — unit runner + in-process HTTP
  integration against the Express app. See [testing.md](testing.md).
- **eslint** `^9` + **@typescript-eslint** `^7` — lint gate.
