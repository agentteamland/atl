---
knowledge-base-summary: "Cache-aside as a pipeline concern rather than handler code, key naming, and TTL discipline — but the load-bearing half is **ownership of invalidation**: it belongs at the persistence seam every write passes through, never per command, and never both. Collect keys before the save, evict after it commits. Plus Redis as a distributed primitive — one multiplexer, locks, sliding-window rate limits, and the request-path idempotency store (which has no equivalent inside a consumer)."
---

# Caching and Redis

One dependency, two jobs that behave nothing alike.

- **A cache** is data the application is allowed to lose. Losing it costs latency, never correctness.
- **Distributed primitives** — locks, rate-limit counters, an idempotency store — are data the
  application is *not* allowed to lose or to guess at. Losing one costs correctness directly.

They share a client and a connection and almost nothing else, and the single most common mistake in
this area is applying the cache's tolerant failure posture to a primitive. Keep the split in mind
while reading: every rule below belongs to one half or the other, and the last section says what to
do when Redis is down, which is where the difference bites.

## Cache-aside, declared rather than coded

Cache-aside only: the application owns both the read and the write path, and the cache is **never**
the source of truth. Read-through, write-through and write-behind are not used — they put the cache
on the correctness path, which is exactly the property the first paragraph says a cache does not have.

The handler should not know a cache exists. The query *declares* that it is cacheable; a pipeline
behaviour does the work.

```csharp
// The query declares intent. Nothing in the handler changes.
public sealed record GetWidgetQuery(Guid Id) : IQuery<WidgetResponse>, ICacheable
{
    public string CacheKey => CacheKeys.Widget(Id);
    public TimeSpan? CacheDuration => null;   // null = resolve from configuration
}
```

The behaviour, constrained to `where TRequest : ICacheable`, runs before the handler: read the key,
return on a hit, otherwise call the handler and write the result with a TTL. (The exact behaviour
signature depends on the dispatcher the project uses — MediatR and the source-generated `Mediator`
differ in parameter order and return type. Copy the project's existing behaviours, not this sketch.)

Two rules that are not stylistic:

- **Cache the response DTO, never a domain entity.** An entity carries navigation properties, lazy
  proxies and tracking state; serializing one produces something between a bloated blob and an
  exception, and deserializing one produces an object EF will not recognise.
- **A cache read or write that throws must not fail the request.** Wrap each call; a connection
  failure degrades to "miss", which is a correct outcome. This is the *cache* half of the
  degradation rule, and it is the opposite of what the primitives need.

**What to cache:** read-heavy and write-light data (catalogs, lookup tables, settings), expensive
aggregations, and anything shared across callers. **What not to cache:** per-caller data that
changes inside a single session, anything that must be transactionally consistent at the moment it
is read (a balance, a stock count during checkout), unbounded result sets, and authorization
decisions. **Short-TTL grey area:** profiles, search results, permission lookups.

## Key naming

`{scope}:{entity}:{identifier}`, lowercase, colon-separated, hyphens inside a segment.

| Scope | Holds | Typical TTL |
|---|---|---|
| `cache:` | cached query results and computed values | 15–60 min |
| `lock:` | distributed locks | 30 s – 5 min |
| `session:` | sessions, refresh/verification/OTP tokens | 3 min – 30 days |
| `rate:` | rate-limit counters | 1–5 min |
| `settings:` | dynamic settings, feature flags | 1–7 days |
| `idempotency:` | request-path dedup records | ~24 h |

**Construct every key in one place** — a static class of key-building methods, never an interpolated
string at the call site. The reason is the failure mode: a mistyped key does not error. The write
lands under one name and the read looks under another, so the cache reports a permanent miss, the
feature works, and the only symptom is that the cache never helps. There is no exception, no log
line and no test that fails. Centralizing the construction is what makes the typo impossible rather
than merely unlikely.

The same class owns the wildcard **patterns** used for bulk eviction. Enumerate matching keys with
the client's `KeysAsync`, which issues `SCAN` against any modern server; `KEYS` walks the entire
keyspace and Redis executes commands on a single thread, so `KEYS` in production stalls every other
caller for the duration.

## TTL

**Every key has a TTL. There are no immortal keys** — a key with no expiry is a memory leak that
survives every deploy. Four categories:

| Category | Range | For |
|---|---|---|
| Short | 1–5 min | rate-limit windows, OTPs, locks, volatile search results |
| Medium | 15–60 min | query cache, verification tokens |
| Long | hours–days | sessions and refresh tokens, settings snapshots, idempotency records |
| Very long | days–weeks | feature flags, genuinely static reference data |

Resolve the value in one place, in this order: an explicit per-query override → configuration or a
dynamic settings store → a hard-coded fallback. Scattering `TimeSpan.FromMinutes(30)` through the
codebase means changing a TTL is a deployment.

**Absolute expiry is the default.** Use it for anything whose staleness matters (cache entries),
anything whose window must reset on a fixed schedule (rate limits), and anything security-bearing
(verification tokens, idempotency records). **Sliding expiry** — reset the TTL on each access — is
for state that should live exactly as long as someone is using it: sessions, connection tracking.

TTL is the **backstop**, not the mechanism. A correct TTL bounds how long a missed invalidation can
hurt; it does not excuse the missed invalidation. Which leads to the part that matters most.

## Who owns invalidation

This is the load-bearing decision on this page, and getting it wrong produces stale data that no
test asserts and no exception reports.

**Declaring invalidation per command is correct only while you own every write path.** The pattern
looks clean — the command lists the keys it dirties, a behaviour evicts them after the handler
succeeds — and it holds right up until the same entity is also written by a scheduled job, a queue
consumer, a bulk sweep or a sibling command that a later change added. Then the declaration is a
checklist, and a checklist rots silently: the new write path simply never joined it, nothing fails,
and the cache serves the old value until its TTL runs out.

**So cross-cutting invalidation belongs at the persistence seam — the `SaveChanges` override every
write passes through — not on the command.** That seam cannot be bypassed by a write path that
forgot to opt in, because there is no way to write without going through it.

**Never run both mechanisms for one key.** Two owners for one invariant is how a key gets evicted
twice and reasoned about nowhere; when the eviction is later found to be wrong, neither owner is
obviously the one to fix. The choice is *per key* and it is exclusive: the per-command form remains
correct for a key that exactly one command can ever dirty, and the seam owns everything else.

### The ordering: collect before the save, evict after it commits

Both halves are load-bearing, for two independent reasons.

1. Walk the change tracker and **collect** the keys this save invalidates.
2. Call `base.SaveChangesAsync`.
3. **Evict** the collected keys.

- **Collect before**, because entity states reset during the save. A post-save read of the change
  tracker finds an empty or already-`Unchanged` set, so the eviction list comes out empty — and it
  comes out empty *silently*.
- **Evict after it commits**, because evicting before the commit opens a window in which a
  concurrent read repopulates the key from pre-commit data. The cache is then stale with no stale
  key left to evict, which is worse than never having evicted at all.

### The optional-dependency rule this creates

Putting invalidation on the seam means the `DbContext` now takes a cache dependency — and that
breaks the design-time tooling unless it is declared optional:

```csharp
public ApplicationDbContext(
    DbContextOptions<ApplicationDbContext> options,
    ICurrentUser currentUser,
    ICacheStore? cacheStore = null)   // optional on purpose — see below
```

Where a project supplies an `IDesignTimeDbContextFactory<T>`, `dotnet ef migrations` constructs the
context **by hand, with no DI container**. A required constructor parameter there is not a runtime
error a test would catch — it breaks every migration command, at design time, for everyone.

The general rule: **any dependency added to the `DbContext` constructor must either be optional or
be supplied by the design-time factory.** This is one of the few places where adding a dependency
breaks a tool rather than the application, so the build and the tests stay green while
`migrations add` stops working.

## Redis as a distributed primitive

The half that is not a cache.

**One multiplexer, registered as a singleton, for the whole process.** `IConnectionMultiplexer` is
thread-safe, holds an internal connection pool and reconnects on its own; creating one per request
is the classic way to exhaust sockets under load. Inject the multiplexer and call `GetDatabase()`
where you need it. Set `AbortOnConnectFail = false` so a cold or briefly-absent Redis does not stop
the application from starting, and subscribe to `ConnectionFailed` / `ConnectionRestored` so the
outage is visible in logs rather than inferred from latency. (If the project separates concerns by
logical database number, note that Redis Cluster exposes only db0 — that separation does not
survive a move to Cluster.)

### Distributed lock

One instance at a time across replicas: set the key **only if absent** (`When.NotExists`, i.e.
`SETNX`) with a **TTL** and a **unique token** as its value, and release with a Lua script that
deletes only if the stored value still equals your token.

```csharp
// release — atomic compare-and-delete
if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end
```

Each of the three is guarding a distinct failure:

- **The TTL** guards a crashed holder. Without it a process that dies while holding the lock blocks
  every replica permanently, and the only recovery is a manual key delete.
- **The unique token** guards the slow holder. If your work outran the TTL, the lock has already been
  granted to someone else — a plain `DEL` at the end of your work then releases *their* lock.
- **The Lua script** guards the gap between checking the token and deleting the key. Read-then-delete
  from the client is two round trips with a window in between; the script is one atomic operation.

Keep the expiry just long enough for the operation. A lock is a scheduling tool, not a transaction.

### Rate limiting

`INCR` the counter, and set the expiry **only when the increment returns 1** — that first increment
is what opens the window. Re-setting the expiry on every hit is the common bug and it inverts the
mechanism: the window then never closes while traffic continues, so the counter never resets and the
caller stays blocked for as long as anyone keeps hitting the key. Sustained traffic against a
victim's key becomes an indefinite lockout.

**Name this shape honestly: `INCR` + expire-on-first is a *fixed* window, not a sliding one.** Its
known weakness is the boundary — a caller can spend the full allowance at the end of one window and
the full allowance at the start of the next, so the true worst case is roughly twice the nominal
limit across the seam. That is acceptable for login throttling and abuse control, and not acceptable
where the limit is a contractual quota. A genuine sliding window costs more: a sorted set of request
timestamps trimmed on each check (exact, memory grows with the rate), or two adjacent fixed-window
counters weighted by how far into the current window you are (approximate, constant memory). Reach
for one of those only when the boundary burst actually matters — the fixed window is the right
default.

Which counter is consumed when is what separates a defensible rate limit from a denial-of-service
tool you built for someone else:

- **A per-source (IP) gate is consumed on every attempt.** An attacker burning it throttles only
  themselves.
- **A per-identity counter is a pure *failure* counter** — incremented only *after* the credential
  check fails, cleared on success, and it must **never gate the check itself**. That ordering is the
  whole trick: the legitimate owner's correct value always reaches the verifier no matter how many
  failures preceded it. A per-identity *lockout* and "the real owner always gets in" cannot both
  hold, because the only signal separating attacker from victim is the source, so anyone who knows
  an account identifier can lock it out on demand.
- Moving the counter to the failure path shortens the early-return path, so an **unknown** identity
  must still be verified against a fixed dummy hash — otherwise response time becomes an
  account-enumeration oracle.

Operational note: when several parallel processes exercise the same seeded accounts, a burst of
auth failures in a smoke test is far more likely to be this limit (surfacing as `429` with empty
tokens downstream) than a regression. Confirm against an untouched account before diagnosing the app.

### Request-path idempotency store

The client sends a unique `X-Idempotency-Key` header on mutating requests; a behaviour claims the
key with `SETNX` + a short-lived `"processing"` marker, runs the handler, then overwrites the marker
with the serialized response under a ~24 h TTL. A second request carrying the same key returns the
stored response without the handler running at all. An absent key skips the behaviour entirely, so
the mechanism is opt-in per call rather than mandatory per endpoint.

The claim marker is what makes it safe under concurrency: two replicas cannot both claim the key, so
the loser waits briefly and reads the result rather than executing a second time.

**This is the request-path flavour of exactly-once, and it has no equivalent inside a queue
consumer.** A consumer has no request, therefore no header to carry — and reaching for this
mechanism there is a dead end, not a shortcut. The queue-path flavour is a different construction
entirely: an identifier minted **once by the producer** and carried on every redelivery, plus a
database-hard unique constraint over it. That belongs to
[messaging-and-background-work](messaging-and-background-work.md); do not blur the two.

## When Redis is down, the two halves diverge

This is where treating the whole dependency uniformly causes real damage.

- **The cache fails open.** A read that throws is a miss; a write that throws is skipped. The request
  is slower and completely correct. Log it and continue.
- **The primitives fail closed.** A lock acquisition that threw was **not** acquired — treating an
  exception as "no one else holds it" is how two replicas run the job that must run once. A
  rate-limit check that threw did not establish that the caller is under the limit. An idempotency
  claim that threw did not establish that this is the first delivery.

Never wrap a primitive in the cache's tolerant `catch`. The correct answer for a primitive is to
fail the operation or refuse to proceed — loudly.

## Verifying a change here

Everything on this page fails silently by default, so "it worked when I clicked it" is not evidence.

- **A cache miss proves nothing.** The first request after a deploy always misses. Assert the *second*
  identical request is served from the cache — the hit/miss log line, or the key's presence.
- **Assert the eviction from a write path other than the obvious handler.** If the entity has a
  second writer (a job, a consumer, a sweep), exercise *that* one. Invalidation declared on the
  command is exactly the shape that passes when driven from its own handler and fails from everywhere
  else — the test that only drives the handler cannot see the defect the seam exists to prevent.
- **Check for a missing TTL.** A key with no expiry is a bug regardless of what the feature does;
  reading back the remaining TTL after the write is a one-line assertion.
- **Exercise the Redis-down path at least once.** Stop the container and issue the request. The cache
  path must still answer; a lock or idempotency path must refuse rather than sail through.

## Related

- [persistence-and-migrations](persistence-and-migrations.md) — the `SaveChanges` seam this page hangs
  invalidation on, and the rest of what that override costs per call.
- [messaging-and-background-work](messaging-and-background-work.md) — the queue-path flavour of
  exactly-once, which this page's idempotency store deliberately does not cover.
- [architecture-and-layering](architecture-and-layering.md) — where a pipeline behaviour and the
  multiplexer registration belong, and the composition-root wiring both need.
