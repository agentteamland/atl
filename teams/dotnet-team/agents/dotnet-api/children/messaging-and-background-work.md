---
knowledge-base-summary: "Everything that runs outside a request and the invariants it shares: no ambient identity or tenant (carry the scope in the message, stamp writes by hand), redelivery is guaranteed so idempotency is by construction with a database-hard backstop, and cancellation is honoured on shutdown. Broker craft — topology, publisher confirms, prefetch, DLX — turns on one rule: **a queue's arguments are fixed at creation**, so changing them crash-loops the consumer on exactly the environments that already have data and never on a fresh broker. Plus scheduled hosted work and the internal-endpoint call-back seam."
---

# Messaging and background work

Queue consumers and scheduled jobs are one topic because they share their invariants. Both run
**outside a request**, so neither has an ambient anything; both are **re-entered** — a broker
redelivers, a scheduler re-fires, a pod restarts mid-batch — so neither may assume it runs once; and
both are **stopped by a signal**, so both must honour cancellation or be killed mid-write.

## 1. Outside a request there is nothing ambient — and the defaults open

A `BackgroundService` or a consumer has no `HttpContext`. Whatever the request path reads off it —
the user, the tenant, the correlation id, an idempotency header, the culture — is **null here**.

The dangerous part is not that these are null. It is that **the null case is usually the permissive
branch.** A tenant filter written as *"no current tenant ⇒ see everything"* — the natural way to give
a system-level operator a cross-tenant sweep — degrades **open** in exactly the code that has no
request to blame, silently, with no exception anywhere.

Three rules follow, and they apply to every background writer:

- **Carry the scope in the message, not in the environment.** Publish the *filter*, not the resolved
  result set: a concrete scope id for a scoped caller, an explicit null for a system-level one. The
  payload stays small regardless of how large the result set is, and the set gets resolved fresh at
  delivery time rather than at publish time. Resolve and clamp the scope **at the producer** — a
  caller-supplied scope id is an input, not a fact.
- **On the read side, the hand-written comparison IS the boundary** wherever the root of the query is
  not covered by a filter. A scope variable that resolves to `null` there means *every* tenant, not
  *my* tenant. Never copy a scope-narrowing idiom out of a request-path handler without checking
  whether that handler still had its filter on.
- **On the write side, stamp the scope explicitly on every insert.** Auto-fill that derives the scope
  from the ambient request cannot work here. If the auto-fill *throws* on an unstamped insert, that is
  the good case — loud beats silent — but it means relying on it fails outright.

**The count and the work must come from the same query.** If a caller was told "queued for N" and the
consumer re-resolves the set with a different join, the reported number and the rows actually written
diverge, and nobody can tell which one is wrong. Mirror the producer's query in the consumer, joins
included, or share it.

## 2. Idempotency is by construction, not by hope

Redelivery is not a failure mode, it is a property. A consumer crashes after processing and before
acking; a network partition requeues in flight messages; a channel times out; an operator replays a
dead-letter queue during an incident; a scheduled job fires on three replicas at once. **Design for
"this will run again" and the whole class disappears.**

### The producer mints the key; the database enforces it

```csharp
// Producer — once, at publish. Carried unchanged on every redelivery.
var job = new FanOutJob(BatchId: Guid.NewGuid(), ScopeId: scopeId, Filter: filter);
```

The consumer **never regenerates** that id. Regenerating it makes each redelivery a different logical
operation and defeats the entire mechanism.

Then make the database enforce it. A **partial unique index** over `(batch_id, target_id)` — filtered
to rows where `batch_id IS NOT NULL`, so it covers only the rows this mechanism owns and leaves every
other row outside it — is the backstop that survives redelivery, a partial commit, and any future
multi-replica race. Two details that are easy to get wrong:

- **Give it a dedicated column.** Overloading an existing "related entity" pair is tempting and wrong:
  a dedicated column is self-documenting and, the load-bearing half, it can carry a **unique
  constraint of its own** without constraining every other use of the shared pair.
- **Do not filter the index on the soft-delete flag.** A soft-deleted row still occupies its slot, so
  a target is never re-processed for the same batch after someone deletes the evidence.

### Check-before-insert is the primary path; the index is the backstop

Read the ids already persisted for this batch, skip them, insert the rest. Do **not** rely on catching
the unique violation as the normal path:

- Ordinary inserts then still flow through `SaveChanges` and its interceptors — scope stamping, audit
  collection, soft-delete conversion — rather than around them.
- A failed insert leaves a poisoned tracker; recovering from it correctly is more code than avoiding it.

The dedup read usually has to **ignore query filters**, on purpose: it must see the *physical* rows,
because the consumer has no ambient scope and a soft-deleted row still occupies the index slot a
filtered read would miss. Scope it to the one batch id and it leaks nothing.

### Commit in chunks, and converge across a partial completion

Chunk large inserts and let each chunk commit independently. A crash after chunk *k* leaves *k*
committed, and the redelivery re-resolves the same targets and skips them — convergence holds across
partial completion, which is the only completion a long fan-out can promise.

**Clear the change tracker after each chunk.** A `SaveChanges` override that walks the tracked entries
runs once per call over *every entity saved so far*, so a chunked insert through one context is
quadratic — invisible at test volumes, fatal at real ones. This is the persistence topic's finding; it
is repeated here because the chunked batch insert is where it bites.

### The request-path flavour is a different mechanism

A request-scoped idempotency key (a client-supplied header, stored in a cache with a TTL) is the
**request** path's answer. **Do not reach for it inside a consumer** — there is no request to carry it,
and a key derived inside the consumer is minted fresh on every redelivery, which is the exact defect
the mechanism exists to prevent.

Where a cache-backed dedup *is* the right tool (a consumer with no database write to constrain), the
rules are: an atomic set-if-not-exists, a **mandatory TTL** (keys without one grow forever), a distinct
key prefix per consumer, and — the one people invert — **release the key on failure so the retry can
run, and never release it on success**; let it expire.

## 3. Broker craft

### Topology, declared by whoever needs it

Name things by purpose, consistently: an exchange as `<purpose>.<type>` (`emails.fanout`,
`logs.fanout`), a queue as `<purpose>.<consumer>` (`emails.smtp`, `logs.elasticsearch`), the dead-letter
exchange as `<purpose>.dlx` and its queue as `<purpose>.<consumer>.dead`. Keep the names in **one shared
static class or a single static declare method** that both the producer and the consumer call — two
copies of a topology dictionary is a divergence waiting for its first edit.

**Every consumer declares its own topology at startup.** It cannot assume a producer started first.
Re-declaring an entity that already exists *with the same arguments* is a no-op, so both sides declaring
is safe and removes the startup-order dependency entirely.

Declare in dependency order: the dead-letter exchange and its queue **first** (the main queue's argument
references the DLX by name), then the main exchange, then the main queue with its arguments, then the
binding.

Four flags that are decisions, not defaults: `durable: true` on exchanges and queues (transient entities
vanish on a broker restart), and `exclusive: false` / `autoDelete: false` in production (both delete the
queue — and everything in it — the moment the consumer disconnects, which is precisely when you want it
retained).

### A queue's arguments are **fixed at creation**

This is the rule the whole topic turns on, and the one whose failure mode is disproportionate to its
cause.

Re-declaring an existing queue with a **different** argument set — adding a dead-letter exchange to a
queue created without one, changing a TTL, changing a max length — is a `PRECONDITION_FAILED`
**channel-level** exception. And the part that converts a bug into an outage: **a failed declare kills
the channel it ran on.** Declare topology on the consumer's own channel and there is nothing left to
recover onto — the service exits, the container restarts, the same declare fails again. A crash loop.

The asymmetry is what makes it expensive:

> **It cannot happen on a fresh broker.** Every developer machine, every CI run, every clean install
> creates the queue from the new code and gets the new arguments. Only a broker that predates the
> change fails — which is to say, **only the environments that hold data.**

**The upgrade path — probe on a throwaway channel, then declare for real** (shown against the
synchronous client API; the async-first major line spells the same calls `CreateChannelAsync` /
`QueueDeclareAsync` — read the pinned version before copying the shape):

```csharp
var argsApplied = true;
try
{
    using var probe = connection.CreateModel();     // throwaway: safe to lose
    probe.QueueDeclare(Queue, durable: true, exclusive: false, autoDelete: false, arguments: newArgs);
}
catch (OperationInterruptedException ex)            // RabbitMQ.Client.Exceptions
{
    argsApplied = false;
    logger.LogWarning(ex, "Queue {Queue} already exists with different arguments — " +
                          "running with the legacy topology. One-time fix: <the runbook step>", Queue);
}

channel.QueueDeclare(Queue, durable: true, exclusive: false, autoDelete: false,
                     arguments: argsApplied ? newArgs : oldArgsOrNull);
```

Three properties carry it, and prose alone re-derives the wrong one — probing on the consumer's own
channel, which is the original outage:

- **The probe is also the declare.** On a fresh broker the throwaway declare *creates* the queue with
  the new arguments and the second declare is a matching no-op. There is no separate "check" API to get
  wrong.
- **The fallback is the legacy declare, not a failure.** Reproducing the original topology exactly means
  the service keeps consuming with the old (worse) behaviour instead of refusing to start. Degrading
  beats stopping.
- **The warning names the remedy, not the symptom.** Point it at the operator step that fixes it —
  typically stop the consumer, delete the queue, restart so it is recreated with the new arguments.
  Anything still queued is lost, so it is a quiet-window operation and the message should say so.

**The rejected alternative is renaming the queue**, and it is worse than the problem it avoids. The old
queue's binding is durable broker state that no code path removes, so it stays bound, a fanout keeps
delivering to it, and it now has **no consumer**. Unless something bounds it (a message TTL, a max
length), it fills forever, and a broker disk or memory alarm takes down *every* queue on the host. A
rename converts a loud, immediate, single-service crash loop into a silent accumulation with a much
larger blast radius.

**The rule that falls out: declare a new queue with the arguments you will eventually want, on the first
commit.** Getting it right before any broker holds the queue costs one line. Getting it right afterwards
costs a probe, a warning path, a runbook section, and a manual step on every existing environment.

Two second-order checks before changing an argument:

- **Grep for every declare site of that queue.** If two files duplicate the argument dictionary, both
  need the probe, and fixing one re-introduces the mismatch from the other side.
- **Check whether the producer also declares the queue.** A single declaring party is what makes the
  probe sufficient; a producer that declares it with the old arguments defeats it.

### Publish reliably, consume conservatively

- **Publisher confirms.** Without them a broker-side rejection loses the message silently. Turn them on
  and **log an unconfirmed publish as undelivered** — a confirm you never check is worse than none,
  because it reads as coverage.
- **Persistent delivery mode + durable queue.** Both, or the message does not survive a broker restart.
  One without the other is a half-guarantee.
- **`prefetch` is sized to the unit of work, not to throughput.** Where one message *is* a whole batch,
  `prefetchCount: 1` is correct — an in-flight limit of one keeps a redelivery from overlapping the
  original. Where messages are small and independent, a higher prefetch is the throughput knob. Decide
  it from the message's meaning.
- **Never `autoAck`.** Ack **after** the work is committed, not on receipt.
- **Nack with `requeue: false` for a message that cannot succeed.** With `requeue: true` a poisoned
  message spins forever at full speed; with `false` it dead-letters and becomes inspectable. Reserve
  `requeue: true` for a *transient* failure you are deliberately choosing to retry immediately.

### Retry with delay, and where the poison goes

Do not `Task.Delay` inside a consumer to retry — it holds the message, the channel and the slot. Let the
broker hold the delay: nack to a **retry queue carrying a message TTL and a dead-letter exchange
pointing back to the main exchange**. The message ages out and is re-delivered to the main queue,
consumer fully responsive throughout.

Bound it. The broker stamps a dead-lettering history (an `x-death` header) that carries the count —
**read that rather than inventing a private retry-count header**, and read it defensively, since its
exact shape varies by client version. Past a small ceiling (three to five), the failure is not transient:
publish it to a parking-lot queue with diagnostic headers (original exchange, original routing key,
failure timestamp, retry count) and ack the original, so it stops circulating and becomes a thing a human
can look at.

**Distinguish transient from permanent at the throw site.** A timeout or a temporarily unavailable
dependency is worth a retry; a malformed payload or a validation failure will fail identically forever
and belongs in the parking lot on the first attempt.

**Dead-letter queues need a watcher.** A filling DLQ with no alert is invisible until someone lists the
queues by hand — a loud error nobody reads is a silent failure with extra steps. If the project has no
alerting on it, say so in the hand-off rather than assuming someone will notice.

### Message shape

Wrap the payload in an envelope — `{ type, version, data, timestamp, correlationId }` — rather than
publishing a bare DTO. It costs nothing and it is what makes an evolving contract legible: a consumer
can route on `type`, reject an unknown `version` deliberately instead of deserializing into a wrong
shape, and a correlation id is the only way to follow one logical operation across the producer, the
queue and the consumer's logs.

Use the **same serializer options on both sides** (a shared static instance, not two configurations that
happen to match), and set the AMQP properties that mean something: content type, persistent delivery
mode, a message id, and the type. **Evolve additively** — a new optional field is safe; a removed or
retyped field is a new message type, not a version bump on the old one.

## 4. Scheduled and hosted work

A scheduled job is a **thin shell**: wait for the tick, take the lock, do the one call, record the
outcome. Business logic belongs behind the same handler an endpoint would call, not in the job.

```csharp
protected override async Task ExecuteAsync(CancellationToken stoppingToken)
{
    while (!stoppingToken.IsCancellationRequested)
    {
        var next = /* next occurrence from the schedule, in UTC */;
        try { await Task.Delay(next - DateTime.UtcNow, stoppingToken); }
        catch (OperationCanceledException) { break; }

        if (!await _lock.TryAcquireAsync(LockKey, LockTtl, stoppingToken))
        {
            _logger.LogInformation("{Job} skipped — another instance holds the lock", JobName);
            continue;
        }
        try { /* one call; the work lives elsewhere */ }
        finally { await _lock.ReleaseAsync(LockKey, CancellationToken.None); }
    }
}
```

- **Schedule in UTC, always.** Parse the expression with a real cron parser (`Cronos` is the usual
  choice — it parses and computes next-occurrence; it schedules nothing, the loop above is yours) and
  pass UTC explicitly. If the requirement is "9am local", resolve the offset when the expression is
  *written*, not at runtime — and record that daylight-saving changes it.
- **Re-read the schedule each cycle** if it is configurable. Reading a string per tick is free next to
  the delay, and it means changing a schedule does not need a redeploy.
- **Single-flight across replicas.** Every replica runs every hosted service, so without coordination a
  midnight job runs on all of them at midnight. A set-if-not-exists lock with a TTL is enough: whoever
  acquires it runs, the rest skip this cycle. The **TTL is the dead-lock protection** — size it above the
  job's worst realistic duration, because a crash mid-run releases nothing else. Release only if you
  still hold it (compare the stored holder id), or a slow run releases the *next* holder's lock.
- **A skipped cycle is normal, not an error.** Log it at information; alerting on it trains everyone to
  ignore the alert.
- **Record health where an operator can read it.** Last run, last success, last failure with its reason,
  consecutive failures, duration. The question this answers is *"is this job stuck?"* — and stuck jobs
  are otherwise indistinguishable from healthy idle ones. **Derive the timestamps from the runtime**, not
  from anything the job itself reports as "now".

**Cancellation is not optional.** Pass `stoppingToken` to *every* async call in the loop — the delay, the
HTTP call, the lock, the save. A call without it hangs through shutdown until the host's bounded shutdown
timeout expires and the process is force-killed mid-operation. Keep that timeout under the orchestrator's
own grace period, or the orchestrator kills the container while the host is still politely waiting. For a
long batch, check the token **between items** and exit on a clean boundary rather than trying to finish.

The one deliberate exception: a **compensating or releasing** call in a `finally` should not use the
token that just fired — pass `CancellationToken.None` there, or shutdown cancels the cleanup.

## 5. The internal-endpoint seam

A thin host that needs the domain calls **back into the composed host over HTTP**, on endpoints that are
not part of the public API:

```csharp
group.MapPost("/internal/<action>", <Action>Async)
     .AddEndpointFilter<InternalSecretFilter>()   // shared-secret check; no user identity involved
     .WithName("<Action>");
```

- **A shared secret, not a user token.** There is no user in this call — inventing one (a service
  account with a real identity) creates an account with broad rights and no owner. An endpoint filter
  comparing a header against a configured secret is the whole check.
- **One value, injected into both hosts from one variable.** Two independently-set secrets are the
  configuration failure from the layering topic: they are not in sync, they are currently equal.
- **These endpoints are a real attack surface.** They must not be reachable from outside the private
  network, and the secret must not be logged — not in the request log, not in an error message, not in a
  retry's diagnostic payload.
- **The caller is a typed client with the header attached once**, in a delegating handler on the
  `HttpClient`, so no job can forget it and no job can hand-roll it differently.
- **Make the endpoint idempotent, exactly like a consumer.** A scheduler retries, and the call may have
  succeeded before the response was lost.

The same seam carries a push to a realtime host: persist the record first as the source of truth, then
POST best-effort with a short timeout, log-and-swallow the failure. The realtime topic owns that half.

## Checklist for anything running outside a request

- [ ] Scope/identity travels **in the message or the schedule payload** and is re-applied by hand; no
      code path reads it off an ambient accessor.
- [ ] Writes stamp their scope explicitly; reads that rely on a hand-written comparison have been checked
      for the null case.
- [ ] A stable id is minted **once by the producer** and carried unchanged on every redelivery.
- [ ] A database-hard constraint backs the dedup; check-before-insert is the primary path.
- [ ] Long work is chunked, each chunk commits, and the change tracker is cleared per chunk.
- [ ] The queue's arguments are the ones it should have **forever**; if they changed, the probe pattern is
      in place and both broker states were exercised.
- [ ] Manual ack after commit; nack `requeue: false` for anything that cannot succeed; a DLX exists and
      someone watches it.
- [ ] Publisher confirms on, and an unconfirmed publish is logged as undelivered.
- [ ] `prefetch` justified by the unit of work.
- [ ] The cancellation token reaches every async call; cleanup in `finally` uses `CancellationToken.None`.
- [ ] A scheduled job takes a TTL'd lock before doing anything, and releases only its own.
