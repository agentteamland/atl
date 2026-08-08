---
knowledge-base-summary: "EF Core on both sides. Runtime: what a `SaveChanges` override costs per call, why a chunked batch insert through one context goes quadratic unless the tracker is cleared, why an aggregate projection must materialize before constructing a DTO, how a lost check-then-insert race is made to converge, and the tracking/projection/N+1 rules underneath. Schema: a second index declaration over the same property list silently REPLACES the first; a scaffolded migration matches dropped columns to added ones **positionally**, so read `Up()` before trusting it; never scaffold against a stale build; a system-column concurrency mapping needs a hand-emptied migration on an existing table. The through-line on both sides: verify against the database and a real execution, never against the configuration and never against the build."
---

# Persistence and migrations

EF Core fails in two directions and neither of them is visible to the compiler.

**At runtime** it produces code that builds clean, passes the tests written for it, and then
misbehaves under a condition the test never created: volume, a second write path, concurrency, or
simply an environment that already holds data. Only one of the traps below fails on its first
execution; the rest need something a fresh green run does not have.

**At schema time** the tooling produces output that differs from what the code plainly appears to
declare. Some of that output announces itself by refusing to apply. Most of it does not — it
compiles, it migrates cleanly, and the database is quietly not what you wrote.

One habit covers both halves: **the configuration is a claim and the build is a claim.** Settle a
question about the schema by reading the database, and a question about behaviour by executing it.
Reading the code you just wrote proves only that you can read.

Project-specific truth — the actual entities, the pinned provider, the real command wrappers, the
migration workflow — lives in the project's own documentation. This page is the generic craft under
it.

## The `SaveChanges` seam — one owner, and a per-call cost

Cross-cutting persistence behaviour belongs at the one place every write passes through: an override
of `SaveChangesAsync` on the context (or an interceptor). Tenant stamping, delete-to-soft-delete
conversion, audit collection, timestamp stamping, cache-key collection — each of these is correct
per-handler only while you own every write path, and you stop owning it the moment a scheduled job or
a queue consumer writes the same entity. Per-handler is a checklist, and a checklist rots the first
time a write arrives from a path that never joined it. Nothing throws; the invariant is simply not
applied. **Never run both mechanisms for one invariant** — two owners is how a thing gets applied
twice and reasoned about nowhere.

Two mechanics inside such an override, each of which fails in its own direction:

- **Collect before `base.SaveChangesAsync`; act after it commits.** Entity states reset during the
  save, so a post-save `ChangeTracker` read finds nothing and the step silently degrades to a no-op.
  Acting *before* the commit is the opposite failure: it opens a window in which a concurrent reader
  sees pre-commit state.
- **Take a cross-cutting dependency as an optional constructor parameter.** The design-time context
  factory the migration tooling uses builds a context with no DI container. Make a new dependency
  required and every migration command breaks — with an error that points at DI, not at the change
  you made.

**The cost is real and it compounds.** Each concern in the override enumerates the change tracker and
filters afterwards, so the scan is over everything tracked, not over what changed — and EF's
automatic change detection runs with it. That is affordable per request, where the scoped context is
disposed at the end and the tracker goes with it. It is not affordable in a loop.

### A chunked batch insert through one context is quadratic

EF keeps every saved entity tracked as `Unchanged` after `SaveChanges`. In most projects that is a
memory concern. Where the context has an override, it is also a **time** concern: chunk *k* re-scans
chunks 1…*k*−1 once per pass, on every save, and the override's own audit-style second save leaves
behind *more* tracked entities than the chunk inserted.

A happy-path test at two rows shows none of it. The blow-up needs exactly the large-N fan-out the
path exists to serve.

**The fix is one line per chunk:**

```csharp
const int chunkSize = 500;
for (var i = 0; i < pending.Count; i += chunkSize)
{
    var rows = pending.Skip(i).Take(chunkSize).Select(Build).ToList();
    db.Records.AddRange(rows);
    await db.SaveChangesAsync(ct);
    await PublishBestEffortAsync(rows, ct);              // reads ids/timestamps EF wrote on save
    (db as DbContext)?.ChangeTracker.Clear();            // ← without this, every chunk re-scans all prior ones
}
```

Three details that are not obvious from the prose:

- **The cast is required, not stylistic.** Where the application layer talks to an interface
  (`IApplicationDbContext`) that exposes only `SaveChangesAsync` and the database facade, there is no
  `ChangeTracker` member to call. If the interface exposes the tracker, drop the cast.
- **Clear *after* anything that reads the rows back.** Detaching does not blank the values EF wrote
  into the objects, so an already-materialised list stays usable — but a `Clear()` placed before a
  read-back detaches entities you still meant to navigate from.
- **The alternative is a shorter-lived context.** Resolving a fresh scoped context per chunk has the
  same effect and is often cleaner in a hosted service; clearing is the fix when the context is
  handed to you.

**When it applies:** any path that inserts a large number of rows in chunks through one long-lived
context — fan-out, bulk import, seeding. **When it does not:** the ordinary single-save handler.

## Projections translate; constructors do not

A grouped aggregate projected straight into a positional record compiles perfectly and throws
*"could not be translated"* the first time it executes:

```csharp
// BAD — compiles, then fails at execution
var rows = await db.Accounts
    .GroupBy(a => a.Currency)
    .Select(g => new AccountTotalsDto(          // ← positional record constructor
        g.Key,
        g.Sum(a => a.Available + a.Pending),
        g.Sum(a => a.Available - a.Blocked)))
    .ToListAsync(ct);
```

EF recognises aggregate patterns in the final projection; it does not parse an arbitrary constructor
call, so the provider gives up at execution rather than at build.

```csharp
// GOOD — one GROUP BY stays in SQL, the DTO is constructed after materialisation
var raw = await db.Accounts.AsNoTracking()
    .GroupBy(a => a.Currency)
    .Select(g => new
    {
        Currency  = g.Key,
        Available = g.Sum(a => a.Available),
        Pending   = g.Sum(a => a.Pending),
        Blocked   = g.Sum(a => a.Blocked),
    })
    .OrderBy(x => x.Currency)
    .ToListAsync(ct);

var balances = raw
    .Select(x => new AccountTotalsDto(x.Currency, x.Available + x.Pending, x.Available - x.Blocked))
    .ToList();
```

**The load-bearing part is the projection target, not "only sum bare columns".** Arithmetic *inside*
`Sum(...)` translates fine — a computed expression in the aggregate argument is not the problem. Read
the rule as: **group into an anonymous type, construct the DTO after `ToListAsync`.** That form costs
nothing where a constructor would have worked, and it makes the question moot.

The in-memory pass is free when the grouped result is small — one row per group, not one per source
row. If the group count is itself large, you have a different problem than translation.

Leave a one-line comment at the query saying why the projection is shaped this way. Without it, the
next edit "tidies" it back into a constructor and the endpoint starts failing on first call again.

## Reading: tracking, projection, N+1

- **A read-only query that materialises entities should be `AsNoTracking()`.** Tracking exists so
  `SaveChanges` can find your edits; a query that never edits pays for a snapshot of every row and
  keeps it alive for the context's lifetime.
- **A query projected into a DTO is not tracked at all** — EF only tracks entity types. So projection
  is both the faster read and the one that cannot accidentally be saved. Prefer it for anything the
  handler only reads.
- **`Include` on a collection multiplies rows.** Two collection includes on one root is a cartesian
  product, and it is a data-volume problem rather than a query-count one. `AsSplitQuery()` trades it
  for several round trips; projecting only the columns you need avoids the choice.
- **The N+1 is usually a loop after materialisation**, not a missing `Include` — a navigation touched
  per row issues a query per row. Fix it by pulling what the loop needs into the projection.
- **Never `ToList()` mid-chain to make a query compile.** Everything after it runs in memory over the
  whole table, silently, and the symptom is a timeout in an environment with data rather than an
  error anywhere.

## Concurrency: converge on insert, conflict on update

These are two different races with two different correct answers, and treating them as one is how a
first page load starts returning `409` to a user who did nothing wrong.

**An update race** is what an optimistic-concurrency token is for. The caller echoes the token it
read; a mismatch means someone saved in between; the answer is a conflict status and a re-read. Both
the explicit check and EF's own `DbUpdateConcurrencyException` map to the same place — see
[error-contract.md](error-contract.md).

**A first-visit "create it if missing" handler has a structurally different race.** Two requests both
read "missing", both insert, one loses on the unique index — and *both callers should end up looking
at the same row*. Failing the loser turns an ordinary first load into a server error the caller can
do nothing about.

```csharp
try
{
    await db.SaveChangesAsync(ct);
    return draft;
}
catch (DbUpdateException)
{
    db.Drafts.Remove(draft);                       // Added -> Detached; emits no DELETE
    var winner = await LoadAsync(db, ownerId, ct);
    if (winner is null) throw;                     // duplicate key with no surviving row = a real fault
    return winner;
}
```

**Both lines are load-bearing, and they guard different things.**

- **The `Remove` is not optional.** A failed insert leaves the entity in the `Added` state in the
  change tracker. `Remove()` on an `Added` entity transitions it to `Detached` — it emits no `DELETE`,
  it just drops it from tracking. Skip it and any *later* save in the same request retries the
  duplicate insert and throws again, far from the cause and looking like a fresh bug rather than the
  first failure's echo.
- **The rethrow is not optional either.** A bare `catch (DbUpdateException)` also swallows write
  failures that have nothing to do with this race. The null-winner rethrow is the whole guard: without
  it a genuine fault is laundered into a successful "convergence" on a row this request never raced
  for. Narrowing the catch to the provider's unique-violation code or the specific index name is
  better where the provider makes that available — the rethrow is what makes the broad form safe.

A site that returns immediately from every catch arm survives without the `Remove` — but only
*positionally*, and nothing at that site stops a later edit from adding a save after the catch and
reintroducing the silent retry. Write both lines.

## Schema: every artefact the tooling produces is a claim

### A second index declaration over the same property list silently replaces the first

EF keys index configuration by the **property list**, not by the database name. So this declares one
index, not two:

```csharp
builder.HasIndex(x => new { x.TenantId, x.OwnerId, x.ItemId })
       .HasDatabaseName("ix_items_tenant_owner_item");

builder.HasIndex(x => new { x.TenantId, x.OwnerId, x.ItemId })   // ← overwrites the line above
       .HasFilter("closed_at IS NULL")
       .HasDatabaseName("ix_items_open_by_item");
```

Only the second one is ever created. The first **never exists in the database**, and nothing flags
it: it compiles, the migration applies, and an index you believe exists but does not is a performance
cliff that appears under load, long after the commit that "added" it. To genuinely declare two
indexes over the same columns, the property lists must actually differ — a different column order, or
a different set.

**The check is against the database, not the configuration.** On PostgreSQL,
`select indexname from pg_indexes where tablename = '<table>' order by 1;`; the equivalent catalogue
view on any other provider. The confirming tell is cheap too: delete the declaration you suspect is
dead and re-run the model differ — *"no changes"* proves it had been a no-op all along.

### The scaffolder infers renames positionally, and will destroy data doing it

`migrations add` matches dropped columns to added ones by **position and type**, not by meaning. On a
multi-column rename it will happily produce a set of inferred renames of which every single one is
semantically wrong — a timestamp column renamed onto a differently-meaning timestamp column stamps a
lifecycle state onto every existing row at deploy time; an amount renamed onto a count silently
reinterprets the values; a signed quantity lands in a column that assumes unsigned.

A rename asserts that the old values still mean something in the new column. **When they do not, the
honest migration is drop-and-add.**

**The rule: always read a scaffolded migration's `Up()` before trusting it.** The tool optimises for
a small diff, not for meaning, and it warns only with a generic *"an operation was scaffolded that may
result in the loss of data"* — which is easy to wave through because it is technically true of any
drop.

### Never scaffold against a stale build

`migrations add` reads the model out of the **compiled** assembly. A `--no-build` (or a run against a
container image that was not rebuilt) diffs whatever was last compiled against the snapshot, so an
uncompiled change produces a migration whose `Up()` and `Down()` are **empty — with no warning of any
kind.** `migrations remove --no-build` then cannot see that migration either, so the `.cs` and
`.Designer.cs` must be deleted by hand and the snapshot reverted.

Rebuild first, or leave the flag off — and treat an empty `Up()` as a claim to be checked, never as
"no schema change was needed".

**That check cannot be reflexive, which is why the convention matters:** legitimately-empty migrations
exist too. So an empty body proves nothing on its own — **the comment is the signal.** Write one on
every intentional no-op, saying why it is empty and what it delivers.

### A system-column concurrency mapping needs a hand-emptied migration

Where the provider offers a system column as the concurrency token — PostgreSQL's `xmin` is the usual
case — that column **already exists on every table**; it is not something a migration can create. The
scaffolder does not know that, so adding the mapping to an entity whose table already exists produces
an `AddColumn` that fails to apply.

**Empty the migration by hand, and keep it.** What it actually delivers is the **model snapshot**
recording the mapping; there is no DDL to run. Deleting it loses the snapshot and the differ will
scaffold the same broken migration again.

Two things about it:

- **The asymmetry:** on a table being *created* in the same migration the mapping is harmless — it
  rides inline in the create. Only the add-to-an-existing-table form bites, which is precisely the
  form a *newly adopted* token takes.
- **It returns whenever the snapshot loses the mapping** — which is the stale-build condition above,
  so the two compound. After touching a concurrency mapping, confirm the new `.Designer.cs` still
  carries every such mapping the previous one had. A mapping that silently drops out and comes back
  leaves no record of why.

## Modelling defaults that are cheap now and expensive later

- **Money is `decimal` with explicit precision, never `float` or `double`.** Set the precision and
  scale in the configuration; a default precision is a rounding decision nobody made. Store the
  currency alongside the amount, never in the column name.
- **Timestamps are UTC, in a timezone-aware column.** Convert for display in the client, never in the
  database and never in the handler. A date-only value is its own type — binding a day-granular
  filter as a plain date-time is how a comparison against a timezone-aware column becomes a runtime
  failure that reads to the caller as *"no rows"*.
- **Keys are UUIDs generated by a database default**, so a row can be created without a round trip and
  ids carry no sequence to guess. Accept the index-locality cost, or use a time-sortable UUID version
  where the provider offers one.
- **Enums are stored as strings.** As integers, adding or reordering a value silently reinterprets
  every existing row; as strings the data is readable and the change is safe.
- **Strings are bounded unless they genuinely are not.** An explicit maximum length aligns the client
  validation, the request contract and the column; reserve unbounded text for long-form content.
- **Every relationship declares its delete behaviour explicitly.** The default varies with the
  relationship's shape, and inheriting it means the behaviour was never decided. Cascade where the
  child has no meaning without the parent; restrict where the child is an entity in its own right;
  null the reference where the child survives the parent.
- **Soft delete is a global query filter plus a delete-to-modify conversion at the save seam.** Two
  consequences worth knowing before you add either: a query that must see removed rows has to ignore
  the filter explicitly, and — the sharp one — **the filter applies to navigation properties too, so
  soft-deleting a parent blanks or strands every child row that projects off it, silently, with no
  exception anywhere.** Before adding a delete endpoint on a parent, grep for projections that read
  through that navigation. Ending a live entity is usually a status transition; soft delete is
  catalogue hygiene, and the two are not interchangeable.
- **A unique constraint over a soft-deleted table needs the filter in the index**, or a removed row
  keeps its natural key reserved forever and the same key can never be re-used.

**On foreign-key indexes, verify rather than assume in either direction.** EF creates a single-column
index over a foreign key by convention, which makes a hand-written declaration over the bare FK
usually redundant — while retired guidance in this lineage asserted the opposite ("EF does not create
FK indexes; always add them"). Both claims are cheap to settle for *your* provider and version, and
the way to settle them is the way you settle everything else here: read the generated migration, or
the catalogue view. What convention will certainly *not* give you is the **composite** index your
real predicate needs — an equality column ahead of the range column it is filtered and sorted by. Add
those deliberately, name the query each one serves in a comment above it, and confirm the plan uses
it rather than assuming it does.

## Verification checklist

- [ ] Any bulk path clears the tracker (or takes a fresh context) per chunk, **after** anything that
      reads the saved rows back — and was exercised at a volume where the growth would show.
- [ ] Every grouped projection constructs its DTO after materialisation, with the reason at the line.
- [ ] Any check-then-insert converges: detach, re-read, rethrow on no winner.
- [ ] Read paths that only read are untracked or projected.
- [ ] Every new or changed index was confirmed **in the database**, not in the configuration.
- [ ] Every scaffolded migration's `Up()` was read before it was trusted; every inferred rename was
      confirmed to mean what it says, or replaced with drop-and-add.
- [ ] The migration was scaffolded against a fresh build; any intentionally-empty one carries a
      comment saying why.
- [ ] A destructive change was exercised against a database that **already holds data**, not only
      against a fresh one — a fresh environment cannot reproduce a defect that only exists where data
      is. See [testing-and-verification.md](testing-and-verification.md).
- [ ] Delete behaviour, precision, string bounds and enum storage were decided rather than inherited.
