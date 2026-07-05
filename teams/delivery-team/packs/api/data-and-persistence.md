# Data & persistence — access layer, migrations, transactions

The database is the API's durable truth, and it is the part hardest to fix after the fact: a bad
query is a code change, but a bad migration or a lost-update race can corrupt data that no redeploy
brings back. This topic is the generic Express/TypeScript craft for touching a database **safely and
reversibly** — reach it through one access layer, evolve its schema through forward-only migrations,
and wrap multi-write invariants in transactions. The through-line is that persistence mistakes are
*durable*, so the conventions here trade a little ceremony for the guarantee that state stays
correct and every schema change is auditable and rollable.

The project's *specific* choices — which driver/ORM, the actual schema, the table conventions —
live in the project wiki (`Architecture/`, `Conventions/`), named by the tech-lead's canonical
brief. This pack fixes the *discipline*; the project fills the *specifics*.

## One data-access layer, never ad-hoc queries in a route

All database access goes through a dedicated layer — a **repository** module (or the ORM's model
layer) — not raw queries scattered through controllers and services. A service depends on a typed
repository interface; the controller never sees SQL at all.

```typescript
// repositories/accountRepo.ts — the ONLY place account SQL lives
export interface AccountRepo {
  get(id: string): Promise<Account>;
  updateBalance(id: string, delta: number, tx?: Tx): Promise<void>;
}
```

- A **service** takes a repository as a dependency (constructor arg or parameter), so in a unit test
  it receives a fake/in-memory repo and needs no database at all ([testing.md](testing.md)).
- The repository is where query construction lives — and where **parameterized queries** are
  non-negotiable: values are always bound parameters, never string-concatenated into SQL. String
  interpolation of request data into a query is a SQL-injection hole
  ([untrusted-input](../../../../core/rules/untrusted-input.md): validated-then-still-parameterized —
  the boundary schema shapes it, the bound parameter makes it inert).

**Why one layer.** Centralizing access means the injection guard, the connection/transaction
handling, and the mapping between rows and domain types are each written once and inherited by every
caller — a reviewer audits data-safety in one module, not across every route. It also decouples the
business logic from the storage engine: the service tests never touch a real database, and swapping
or tuning the query layer doesn't ripple into services. Ad-hoc queries in a route defeat all three —
each is its own injection-audit surface, each re-implements connection handling, and each welds the
logic to the transport.

## Migrations — forward-only, versioned, reviewed

The schema evolves **only** through committed, ordered migration files run by a migration tool (the
team pins one — e.g. `knex migrate`, `prisma migrate`, `drizzle-kit`, or `node-pg-migrate`). Never
mutate a schema by hand against a database, and never edit an already-applied migration.

- **Each migration is a numbered/timestamped file, applied in order, tracked in a migrations table**
  so a fresh environment reaches the exact same schema deterministically. The migration set *is* the
  schema's history.
- **Forward-only.** A schema change is a *new* migration, never an edit to one that already ran. An
  applied migration is immutable — editing it desynchronizes every environment that already advanced
  past it (the tracking table thinks it ran; the new content never will), which is exactly the kind
  of silent, un-diffable drift that corrupts data. To undo, write a **new** compensating migration.
- **Expand → migrate → contract for anything a running app reads.** A rename or type change is not
  one destructive step; it is: add the new column (expand), backfill + dual-write, then drop the old
  column in a *later* migration once no code reads it (contract). This keeps a migration compatible
  with the code that's already deployed, so a deploy and its migration don't have to be atomic —
  critical because the app and the schema roll forward independently.
- **Migrations are diffed and reviewed like code**, because a migration is the single most
  irreversible artifact the API ships. A destructive migration (a `DROP`, a `NOT NULL` on a
  populated table, a type narrowing) gets extra scrutiny in review — the tech-lead's gate
  ([review-craft](../../agents/tech-lead/children/review-craft.md)) treats it as high-risk.

**Why forward-only + expand/contract.** The database has state the code doesn't — you can redeploy
code freely, but you cannot un-run a migration that already dropped a column of live data. Making
every change additive-first and every undo a new migration means the schema's history is a clean,
replayable, auditable log, and no deploy is wedged into a big-bang "schema and code must switch at
the same instant" step that has no safe rollback.

## Transactions — bound the multi-write invariant

When a single logical operation performs **more than one write that must all succeed or all fail**,
it runs inside a database transaction. The transaction boundary is the operation's atomicity
guarantee: partway-through is never observable, and a failure rolls the whole thing back.

```typescript
// A transfer debits one account and credits another — both, or neither.
await db.transaction(async (tx) => {
  await accounts.updateBalance(sourceId, -amount, tx);
  await accounts.updateBalance(destId, +amount, tx);
  // throw anywhere here → the whole transaction rolls back; no half-transfer persists.
});
```

- **Draw the boundary around the invariant, not around every query.** A single read, or a single
  write, needs no explicit transaction. The transaction exists for the *multi-write invariant* — the
  set of changes that would corrupt state if only some landed (the debit-without-the-credit).
- **Pass the transaction handle through the repository** (the optional `tx` parameter above) so every
  write in the operation joins the *same* transaction; a repository method that opens its own
  connection instead would silently escape the boundary.
- **Concurrency: guard the read-modify-write.** When two requests can race on the same row (read a
  balance, compute, write it back), the naive version loses an update. Use the database's guarantee —
  a conditional/atomic update (`UPDATE ... WHERE balance >= :amount`), row-level locking
  (`SELECT ... FOR UPDATE`), or an appropriate isolation level — inside the transaction. The
  correctness of a concurrent invariant is enforced by the database, not by hoping the two requests
  don't overlap.
- **Keep transactions short.** A transaction holds locks; long-running work (an external HTTP call,
  heavy computation) inside one blocks other writers and invites deadlocks and timeouts. Do the slow
  work outside, transact only the writes.

**Why transactions are load-bearing.** Half-applied multi-write state is the worst failure a data
layer has: it's silent (no error surfaced), durable (it's persisted), and often undetectable until
much later (a balance that doesn't reconcile). The transaction is the mechanism that makes "all or
nothing" a guarantee rather than a hope, and the concurrency guard is what makes it hold *under load*
— which is exactly the condition (`atl work dispatch`'s parallel workers, and later real traffic)
where a lost update actually happens.

## Test seam — persistence is verified, not assumed

Because access goes through a repository interface, the layers test at the right cost:

- **Service logic** unit-tests against a **fake repository** — no database, fast, the wide base.
- **The repository + migrations + transaction behavior** are verified by **integration tests against
  a real database instance** (a disposable/ephemeral database, reset per run) — this is the only way
  to prove the SQL is valid, the migration applies cleanly, and the transaction actually rolls back.
  A mock cannot prove a migration runs or a rollback fires; those are real-database facts. See
  [testing.md](testing.md) for the integration-test setup and the CI command.

## Persistence checklist

- [ ] All DB access is behind a repository/model layer; no raw queries in controllers or services.
- [ ] Every query is parameterized (bound values); no request data string-concatenated into SQL.
- [ ] Schema changes ship as new, ordered, forward-only migration files — no edits to applied
      migrations; destructive changes use expand→migrate→contract.
- [ ] Migrations reviewed as code; destructive ones flagged for extra scrutiny.
- [ ] Multi-write invariants run in a transaction; the `tx` handle threads through every write in
      the operation; the boundary wraps the invariant, not every query.
- [ ] Concurrent read-modify-write is guarded by the database (atomic/conditional update or lock),
      not by luck; transactions kept short.
- [ ] Service logic unit-tested with a fake repo; the repository + migrations + rollback verified by
      integration tests against a real disposable database ([testing.md](testing.md)).
